package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/agentcompany/agent-company-os/backend/internal/agents/registry"
	"github.com/agentcompany/agent-company-os/backend/internal/agents/runtime"
	"github.com/agentcompany/agent-company-os/backend/internal/approval"
	"github.com/agentcompany/agent-company-os/backend/internal/artifact"
	"github.com/agentcompany/agent-company-os/backend/internal/audit"
	"github.com/agentcompany/agent-company-os/backend/internal/llm"
	"github.com/agentcompany/agent-company-os/backend/internal/model"
	"github.com/agentcompany/agent-company-os/backend/internal/project"
	"github.com/agentcompany/agent-company-os/backend/internal/risk"
	"github.com/agentcompany/agent-company-os/backend/internal/task"
)

type progressKey struct{}
type compactProgressKey struct{}

type ProgressFunc func(message string)

func WithProgress(ctx context.Context, fn ProgressFunc) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, progressKey{}, fn)
}

func WithCompactProgress(ctx context.Context) context.Context {
	return context.WithValue(ctx, compactProgressKey{}, true)
}

func compactProgress(ctx context.Context) bool {
	v, _ := ctx.Value(compactProgressKey{}).(bool)
	return v
}

func Report(ctx context.Context, message string) {
	fn, ok := ctx.Value(progressKey{}).(ProgressFunc)
	if ok && fn != nil {
		fn(message)
	}
}

type Service struct {
	projects    project.Repository
	agents      registry.Repository
	tasks       task.Repository
	approvals   *approval.Service
	audit       audit.Repository
	artifacts   artifact.Repository
	runtime     runtime.AgentRuntime
	defaultName string
}

type Result struct {
	Workflow  string           `json:"workflow"`
	Task      model.Task       `json:"task"`
	Risk      risk.Result      `json:"risk"`
	Runs      []model.AgentRun `json:"runs,omitempty"`
	Artifacts []model.Artifact `json:"artifacts"`
	Approval  *model.Approval  `json:"approval,omitempty"`
}

func NewService(projects project.Repository, agents registry.Repository, tasks task.Repository, approvals *approval.Service, auditRepo audit.Repository, artifacts artifact.Repository, rt runtime.AgentRuntime, defaultProject string) *Service {
	return &Service{projects: projects, agents: agents, tasks: tasks, approvals: approvals, audit: auditRepo, artifacts: artifacts, runtime: rt, defaultName: defaultProject}
}

func (s *Service) Plan(ctx context.Context, idea, actor string) (*Result, error) {
	return s.run(ctx, "plan", idea, actor, []string{"chief_of_staff", "product", "cto", "designer", "frontend", "backend", "qa"}, "product_plan")
}

func (s *Service) Build(ctx context.Context, title, actor string) (*Result, error) {
	return s.run(ctx, "build", title, actor, []string{"coding"}, "coding_prompt")
}

func (s *Service) Launch(ctx context.Context, topic, actor string) (*Result, error) {
	return s.run(ctx, "launch", topic, actor, []string{"content", "growth", "compliance"}, "launch_draft")
}

func (s *Service) Review(ctx context.Context, item, actor string) (*Result, error) {
	return s.run(ctx, "review", item, actor, []string{"qa", "compliance"}, "review_draft")
}

func (s *Service) run(ctx context.Context, workflow, text, actor string, agentNames []string, artifactType string) (*Result, error) {
	text = strings.TrimSpace(llm.SanitizeText(text))
	if text == "" {
		return nil, fmt.Errorf("workflow text is required")
	}
	if !compactProgress(ctx) {
		Report(ctx, "阶段："+workflow+"。")
	}
	project, err := s.projects.GetOrCreateDefault(ctx, s.defaultName)
	if err != nil {
		return nil, err
	}
	r := risk.Detect(text)
	status := "assigned"
	if risk.NeedsApproval(r.Level) {
		status = "needs_founder_approval"
	}
	owner := agentNames[0]
	t, err := s.tasks.Create(ctx, model.Task{ProjectID: project.ID, Title: workflow + ": " + text, Description: text, OwnerAgent: owner, Status: status, CreatedBy: actor})
	if err != nil {
		return nil, err
	}
	_ = s.tasks.AddEvent(ctx, t.ID, "workflow_started", actor, workflow+" workflow started", map[string]interface{}{"workflow": workflow, "risk_level": r.Level})
	if risk.NeedsApproval(r.Level) && !compactProgress(ctx) {
		Report(ctx, fmt.Sprintf("风险：%s，需要审批。", r.Level))
	}
	_ = s.audit.Log(ctx, audit.Entry{ProjectID: project.ID, Actor: actor, Action: "workflow." + workflow + ".started", Target: t.ID, RiskLevel: r.Level, Metadata: map[string]interface{}{"workflow": workflow, "external_execution": false}})

	var artifacts []model.Artifact
	var createdApproval *model.Approval
	if risk.NeedsApproval(r.Level) && workflow == "build" {
		Report(ctx, "触发风险门禁：本次执行需要审批，正在创建审批请求。")
		createdApproval, err = s.createApproval(ctx, project, t, actor, r, workflow, text)
		if err != nil {
			return nil, err
		}
		Report(ctx, "审批已创建："+createdApproval.ID+"。")
	}

	for i, name := range agentNames {
		_ = i
		agent, err := s.agents.GetByName(ctx, name)
		if err != nil {
			return nil, err
		}
		if agent == nil {
			return nil, fmt.Errorf("agent %q not found", name)
		}
		out, err := s.runtime.Run(ctx, runtime.AgentRunInput{Agent: *agent, Task: *t})
		if err != nil {
			Report(ctx, fmt.Sprintf("Agent %s 执行失败：%v", name, err))
			return nil, err
		}
		if !compactProgress(ctx) {
			Report(ctx, fmt.Sprintf("%s：%s", agentDisplayName(name), summarizeForTelegram(out.Summary, 50)))
		}
		output := map[string]interface{}{"summary": out.Summary, "details": out.Details}
		_ = s.tasks.RecordAgentRun(ctx, agent.ID, project.ID, t.ID, map[string]interface{}{"workflow": workflow, "task": text}, output)
		_ = s.tasks.AddEvent(ctx, t.ID, "agent_run_completed", name, out.Summary, map[string]interface{}{"workflow": workflow, "provider": out.Details["provider"], "fallback_used": out.Details["fallback_used"]})
		artifactContent := artifactContent(workflow, name, text, out)
		created, err := s.artifacts.Create(ctx, model.Artifact{ProjectID: project.ID, TaskID: t.ID, AgentID: agent.ID, ArtifactType: artifactType, Title: workflow + " / " + name, Content: artifactContent, Status: "draft", Metadata: map[string]interface{}{"workflow": workflow, "agent": name, "risk_level": r.Level, "external_execution": false}})
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, *created)
	}

	if risk.NeedsApproval(r.Level) && createdApproval == nil {
		createdApproval, err = s.createApproval(ctx, project, t, actor, r, workflow, text)
		if err != nil {
			return nil, err
		}
		Report(ctx, "审批已创建："+createdApproval.ID+"。")
	}

	_ = s.tasks.AddEvent(ctx, t.ID, "workflow_completed", actor, workflow+" workflow completed", map[string]interface{}{"workflow": workflow, "artifacts": len(artifacts), "approval_id": approvalID(createdApproval)})
	_ = s.audit.Log(ctx, audit.Entry{ProjectID: project.ID, Actor: actor, Action: "workflow." + workflow + ".completed", Target: t.ID, RiskLevel: r.Level, Metadata: map[string]interface{}{"workflow": workflow, "artifacts": len(artifacts), "approval_id": approvalID(createdApproval), "external_execution": false}})
	runs, _ := s.tasks.ListAgentRuns(ctx, t.ID, 20)
	return &Result{Workflow: workflow, Task: *t, Risk: r, Runs: runs, Artifacts: artifacts, Approval: createdApproval}, nil
}

func (s *Service) createApproval(ctx context.Context, project *model.Project, t *model.Task, actor string, r risk.Result, workflow, text string) (*model.Approval, error) {
	payload := map[string]interface{}{
		"action_type":                 r.ApprovalType,
		"requester":                   actor,
		"project":                     project.Name,
		"workflow":                    workflow,
		"environment":                 "phase_1_backend_closed_loop",
		"risk_level":                  r.Level,
		"summary":                     llm.SanitizeText(text),
		"evidence":                    r.Reason,
		"expected_impact":             "需要负责人确认后，才允许执行外部可见、生产、敏感或编码运行时动作。",
		"rollback_or_mitigation_plan": "当前尚未执行外部动作。可以保留草稿或拒绝审批。",
		"required_approver":           "founder",
		"review_deadline":             time.Now().Add(72 * time.Hour).Format(time.RFC3339),
	}
	return s.approvals.Create(ctx, model.Approval{ProjectID: project.ID, ApprovalType: r.ApprovalType, ItemType: "task", ItemID: t.ID, RequestedBy: actor, RiskLevel: r.Level, Payload: payload})
}

func agentDisplayName(name string) string {
	switch name {
	case "chief_of_staff":
		return "总协调 Agent"
	case "product":
		return "产品 Agent"
	case "cto":
		return "CTO Agent"
	case "backend":
		return "后端 Agent"
	case "designer":
		return "设计师 Agent"
	case "frontend":
		return "前端 Agent"
	case "qa":
		return "QA Agent"
	case "coding":
		return "编码 Agent"
	case "content":
		return "内容 Agent"
	case "growth":
		return "增长 Agent"
	case "compliance":
		return "合规 Agent"
	default:
		return name
	}
}

func summarizeForTelegram(text string, max int) string {
	text = strings.TrimSpace(llm.SanitizeText(text))
	if text == "" {
		return "无文本输出。"
	}
	text = strings.Join(strings.Fields(text), " ")
	if max > 0 && len(text) > max {
		return text[:max-3] + "..."
	}
	return text
}

func summarizeArtifactsForPrompt(items []model.Artifact, max int) string {
	var parts []string
	for _, item := range items {
		content := summarizeForTelegram(item.Content, 1200)
		parts = append(parts, item.Title+": "+content)
	}
	out := strings.Join(parts, "\n\n")
	if max > 0 && len(out) > max {
		return out[:max-3] + "..."
	}
	return out
}

func artifactContent(workflow, agent, text string, out *runtime.AgentRunOutput) string {
	lines := []string{
		"Workflow: " + workflow,
		"Agent: " + agent,
		"Task: " + text,
		"",
		llm.SanitizeText(out.Summary),
	}
	if prompt, ok := out.Details["manual_prompt"].(string); ok && prompt != "" {
		lines = append(lines, "", "Manual Claude Code prompt:", llm.SanitizeText(prompt))
	}
	return strings.Join(lines, "\n")
}

func approvalID(a *model.Approval) string {
	if a == nil {
		return ""
	}
	return a.ID
}
