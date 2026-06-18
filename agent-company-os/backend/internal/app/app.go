package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/agentcompany/agent-company-os/backend/internal/agents/registry"
	"github.com/agentcompany/agent-company-os/backend/internal/agents/runtime"
	"github.com/agentcompany/agent-company-os/backend/internal/approval"
	"github.com/agentcompany/agent-company-os/backend/internal/artifact"
	"github.com/agentcompany/agent-company-os/backend/internal/audit"
	"github.com/agentcompany/agent-company-os/backend/internal/config"
	"github.com/agentcompany/agent-company-os/backend/internal/deployment"
	"github.com/agentcompany/agent-company-os/backend/internal/llm"
	"github.com/agentcompany/agent-company-os/backend/internal/model"
	"github.com/agentcompany/agent-company-os/backend/internal/project"
	"github.com/agentcompany/agent-company-os/backend/internal/report"
	"github.com/agentcompany/agent-company-os/backend/internal/task"
	"github.com/agentcompany/agent-company-os/backend/internal/workflow"
)

type Services struct {
	Config       config.Config
	Agents       registry.Repository
	Projects     project.Repository
	Tasks        *task.Service
	TaskRepo     task.Repository
	Approvals    *approval.Service
	ApprovalRepo approval.Repository
	Audit        audit.Repository
	Reports      *report.Service
	Runtime      runtime.AgentRuntime
	Workflows    *workflow.Service
	Artifacts    artifact.Repository
	Deployment   *deployment.Service
	autopilotMu  sync.Mutex
	autopilots   map[string]*autopilotRun
}

func NewServices(cfg config.Config, agents registry.Repository, projects project.Repository, taskRepo task.Repository, approvalRepo approval.Repository, auditRepo audit.Repository, artifactRepo artifact.Repository) *Services {
	approvalSvc := approval.NewService(approvalRepo, auditRepo)
	rt := buildRuntime(cfg)
	taskSvc := task.NewService(taskRepo, projects, agents, approvalSvc, auditRepo, rt, cfg.DefaultProjectName)
	workflowSvc := workflow.NewService(projects, agents, taskRepo, approvalSvc, auditRepo, artifactRepo, rt, cfg.DefaultProjectName)
	deploySvc := deployment.NewService(cfg.DeployAllowedRoot, time.Duration(cfg.DeployTimeoutSeconds)*time.Second, cfg.DeployMaxOutputBytes)
	return &Services{Config: cfg, Agents: agents, Projects: projects, Tasks: taskSvc, TaskRepo: taskRepo, Approvals: approvalSvc, ApprovalRepo: approvalRepo, Audit: auditRepo, Reports: report.NewService(taskRepo, approvalRepo), Runtime: rt, Workflows: workflowSvc, Artifacts: artifactRepo, Deployment: deploySvc, autopilots: map[string]*autopilotRun{}}
}

func buildRuntime(cfg config.Config) runtime.AgentRuntime {
	var deep runtime.AgentRuntime
	deepStatus := runtime.ToolStatus{Name: "deepseek", Runtime: "deepseek", Configured: false, Available: false, FallbackMode: true}
	switch {
	case cfg.LLMProvider == "deepseek" && cfg.DeepSeekAPIKey != "":
		provider := llm.NewDeepSeekProvider(llm.DeepSeekConfig{
			APIKey:          cfg.DeepSeekAPIKey,
			BaseURL:         cfg.DeepSeekBaseURL,
			Model:           cfg.DeepSeekModel,
			ReasoningEffort: cfg.DeepSeekReasoningEffort,
			Thinking:        cfg.DeepSeekThinking,
			Timeout:         time.Duration(cfg.LLMTimeoutSeconds) * time.Second,
		})
		deep = runtime.NewLLMRuntime(provider, "deepseek", cfg.DeepSeekModel)
		deepStatus = runtime.ToolStatus{Name: "deepseek", Runtime: "deepseek", Configured: true, Enabled: true, Available: true, FallbackMode: false, Command: cfg.DeepSeekModel, TimeoutSeconds: cfg.LLMTimeoutSeconds}
	case cfg.LLMProvider == "deepseek":
		deep = runtime.NewRuleBasedFallbackRuntime("deepseek", cfg.DeepSeekModel, false)
		deepStatus = runtime.ToolStatus{Name: "deepseek", Runtime: "deepseek", Configured: false, Enabled: false, Available: false, FallbackMode: true, Command: cfg.DeepSeekModel, TimeoutSeconds: cfg.LLMTimeoutSeconds, ErrorClass: "missing_api_key"}
	case cfg.LLMProvider != "":
		deep = runtime.NewRuleBasedFallbackRuntime(cfg.LLMProvider, "", false)
		deepStatus = runtime.ToolStatus{Name: "deepseek", Runtime: cfg.LLMProvider, Configured: false, Enabled: false, Available: false, FallbackMode: true, ErrorClass: "unsupported_provider"}
	default:
		deep = runtime.NewRuleBasedRuntime()
		deepStatus = runtime.ToolStatus{Name: "deepseek", Runtime: "rule_based", Configured: false, Enabled: false, Available: false, FallbackMode: true}
	}
	claude := runtime.NewClaudeCodeAdapter(runtime.ClaudeCodeConfig{Command: cfg.ClaudeCodeCommand, Timeout: time.Duration(cfg.ClaudeCodeTimeoutSeconds) * time.Second, Workdir: cfg.ClaudeCodeWorkdir, AllowedRoot: cfg.ClaudeCodeAllowedRoot, MaxOutputBytes: cfg.ClaudeCodeMaxOutputBytes, Enabled: cfg.ClaudeCodeEnabled})
	return runtime.NewRouterRuntime(runtime.RouterConfig{DeepSeekRuntime: deep, DeepSeekStatus: deepStatus, CodingRuntime: cfg.CodingRuntime, ClaudeCode: claude})
}

func (s *Services) Status(ctx context.Context) (model.Status, error) {
	projects, _ := s.Projects.Count(ctx)
	tasks, _ := s.TaskRepo.Count(ctx)
	pending, _ := s.ApprovalRepo.CountPending(ctx)
	active, _ := s.Agents.ActiveCount(ctx)
	blocked, _ := s.TaskRepo.BlockedCount(ctx)
	return model.Status{ProjectsCount: projects, TasksCount: tasks, PendingApprovalsCount: pending, ActiveAgentsCount: active, BlockedTasksCount: blocked}, nil
}

func (s *Services) CreateProject(ctx context.Context, p model.Project, actor string) (*model.Project, error) {
	created, err := s.Projects.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, audit.Entry{ProjectID: created.ID, Actor: actor, Action: "project.created", Target: created.ID, RiskLevel: "low", Metadata: map[string]interface{}{"name": created.Name}})
	return created, nil
}

func (s *Services) HandleCommand(ctx context.Context, name string, args []string, actor string) (string, error) {
	switch name {
	case "start":
		return "AgentCompanyOS Phase 1 is online. Use /help for commands.", nil
	case "help":
		return "/start /help /status /agents /project list|create|show|config [name] /projects /assign [agent] [task] /task list|get|status /tasks /feedback [project] [text] /autopilot start|status|stop|run [project] /deploy [project] /approvals /approve [n] /reject [n|all] [reason] /daily /weekly /plan [idea] /build [task] /launch [topic] /review [item] /runs [n] /artifacts [n] /runtime. Lists show numeric IDs; use those numbers in commands.", nil
	case "status":
		st, err := s.Status(ctx)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Status: projects=%d tasks=%d pending_approvals=%d active_agents=%d blocked=%d", st.ProjectsCount, st.TasksCount, st.PendingApprovalsCount, st.ActiveAgentsCount, st.BlockedTasksCount), nil
	case "agents":
		agents, err := s.Agents.List(ctx)
		if err != nil {
			return "", err
		}
		var lines []string
		for _, a := range agents {
			lines = append(lines, a.Name+" - "+a.Role)
		}
		return strings.Join(lines, "\n"), nil
	case "project":
		return s.projectCommand(ctx, args, actor)
	case "projects":
		_, _ = s.Projects.GetOrCreateDefault(ctx, s.Config.DefaultProjectName)
		projects, err := s.Projects.List(ctx)
		if err != nil {
			return "", err
		}
		return renderProjects(projects), nil
	case "assign":
		if len(args) < 2 {
			return "Usage: /assign [agent] [task]", nil
		}
		res, err := s.Tasks.Assign(ctx, args[0], strings.Join(args[1:], " "), actor)
		if err != nil {
			return "", err
		}
		msg := fmt.Sprintf("Task created and assigned to %s. Use /tasks for its number.\nRisk: %s.\n%s", res.Task.OwnerAgent, res.Risk.Level, res.AgentOutput)
		if res.Approval != nil {
			msg += "\nApproval created. Use /approvals for its number. This action will not execute externally in Phase 1."
		}
		return msg, nil
	case "task":
		return s.taskCommand(ctx, args, actor)
	case "tasks":
		tasks, err := s.Tasks.List(ctx, 20)
		if err != nil {
			return "", err
		}
		return renderTasks(tasks), nil
	case "feedback":
		return s.feedbackCommand(ctx, args, actor)
	case "autopilot":
		return s.autopilotCommand(ctx, args, actor)
	case "deploy":
		return s.deployCommand(ctx, args, actor)
	case "approvals":
		items, err := s.Approvals.List(ctx, "pending")
		if err != nil {
			return "", err
		}
		return renderApprovals(items), nil
	case "approve":
		if len(args) < 1 {
			return "Usage: /approve [approval_id]", nil
		}
		approvalID, err := s.resolveApprovalID(ctx, args[0])
		if err != nil {
			return err.Error(), nil
		}
		a, deployResult, err := s.ApproveApproval(ctx, approvalID, actor)
		if err != nil {
			return "", err
		}
		_ = a
		if deployResult != nil {
			if deployResult.Status == "completed" {
				return "Approved. Deployment executed successfully.", nil
			}
			return "Approved, but deployment failed: " + deployResult.ErrorClass, nil
		}
		return "Approved. No external action was executed.", nil
	case "reject":
		if len(args) < 1 {
			return "Usage: /reject [approval_id|all] [reason]", nil
		}
		reason := strings.TrimSpace(strings.Join(args[1:], " "))
		if strings.EqualFold(args[0], "all") {
			if reason == "" {
				reason = "bulk reject from Telegram"
			}
			items, err := s.Approvals.List(ctx, "pending")
			if err != nil {
				return "", err
			}
			count := 0
			for _, item := range items {
				if _, err := s.Approvals.Reject(ctx, item.ID, actor, reason); err == nil {
					count++
				}
			}
			return fmt.Sprintf("Rejected %d pending approvals. No external action was executed.", count), nil
		}
		approvalID, err := s.resolveApprovalID(ctx, args[0])
		if err != nil {
			return err.Error(), nil
		}
		a, err := s.Approvals.Reject(ctx, approvalID, actor, reason)
		if err != nil {
			return "", err
		}
		_ = a
		return "Rejected. No external action was executed.", nil

	case "plan":
		return s.workflowCommand(ctx, "plan", args, actor)
	case "build":
		return s.workflowCommand(ctx, "build", args, actor)
	case "launch":
		return s.workflowCommand(ctx, "launch", args, actor)
	case "review":
		return s.workflowCommand(ctx, "review", args, actor)
	case "runs":
		taskID, err := s.resolveOptionalTaskID(ctx, firstArg(args))
		if err != nil {
			return err.Error(), nil
		}
		items, err := s.TaskRepo.ListAgentRuns(ctx, taskID, 20)
		if err != nil {
			return "", err
		}
		if len(items) == 0 {
			return "No agent runs yet.", nil
		}
		return renderAgentRuns(items), nil
	case "artifacts":
		taskID, err := s.resolveOptionalTaskID(ctx, firstArg(args))
		if err != nil {
			return err.Error(), nil
		}
		items, err := s.Artifacts.List(ctx, taskID, 20)
		if err != nil {
			return "", err
		}
		if len(items) == 0 {
			return "No artifacts yet.", nil
		}
		return renderArtifacts(items), nil
	case "runtime":
		st := s.Runtime.Status()
		return fmt.Sprintf("Runtime: deepseek configured=%t fallback=%t; claude enabled=%t available=%t fallback=%t", st.DeepSeek.Configured, st.DeepSeek.FallbackMode, st.ClaudeCodeLocal.Enabled, st.ClaudeCodeLocal.Available, st.ClaudeCodeLocal.FallbackMode), nil
	case "daily":
		return s.Reports.Daily(ctx)
	case "weekly":
		return s.Reports.Weekly(ctx)
	default:
		return "Unknown command. Use /help.", nil
	}
}

func (s *Services) projectCommand(ctx context.Context, args []string, actor string) (string, error) {
	if len(args) == 0 || args[0] == "list" {
		_, _ = s.Projects.GetOrCreateDefault(ctx, s.Config.DefaultProjectName)
		projects, err := s.Projects.List(ctx)
		if err != nil {
			return "", err
		}
		return renderProjects(projects), nil
	}

	if args[0] == "show" {
		if len(args) < 2 {
			return "Usage: /project show [project_number_or_name]", nil
		}
		project, err := s.resolveProject(ctx, args[1])
		if err != nil {
			return err.Error(), nil
		}
		cfg, err := s.getProjectConfig(ctx, project.Name)
		if err != nil {
			return "", err
		}
		if cfg == nil {
			return "No project config found.", nil
		}
		return renderProjectConfig(*cfg), nil
	}
	if args[0] == "config" {
		if len(args) < 3 {
			return "Usage: /project config [project_number_or_name] workdir=/path doc=project.md auto_deploy=true service=api:pm2 restart api", nil
		}
		project, err := s.resolveOrCreateProject(ctx, args[1], actor)
		if err != nil {
			return "", err
		}
		cfg, err := parseProjectConfigArgs(project.Name, args[2:], s.Config.DeployAllowedRoot)
		if err != nil {
			return err.Error(), nil
		}
		if err := cfg.Validate(); err != nil {
			return err.Error(), nil
		}
		content, _ := json.Marshal(cfg)
		_, err = s.Artifacts.Create(ctx, model.Artifact{ProjectID: project.ID, ArtifactType: "project_config", Title: project.Name, Content: string(content), Status: "active", Metadata: map[string]interface{}{"project": project.Name}})
		if err != nil {
			return "", err
		}
		_ = s.Audit.Log(ctx, audit.Entry{ProjectID: project.ID, Actor: actor, Action: "project.configured", Target: project.ID, RiskLevel: "medium", Metadata: map[string]interface{}{"project": project.Name, "workdir": cfg.Workdir, "doc_path": cfg.DocPath}})
		return "Project config saved for " + project.Name + ". Use /project show " + project.Name + " to review.", nil
	}
	if args[0] != "create" || len(args) < 2 {
		return "Usage: /project list|show|config|create", nil
	}
	name := strings.TrimSpace(args[1])
	description := strings.TrimSpace(strings.Join(args[2:], " "))
	created, err := s.CreateProject(ctx, model.Project{Name: name, Description: description, Owner: actor}, actor)
	if err != nil {
		return "", err
	}
	return "Project created: " + created.Name + ". Use /project list for its number.", nil
}

func (s *Services) taskCommand(ctx context.Context, args []string, actor string) (string, error) {
	if len(args) == 0 || args[0] == "list" {
		items, err := s.Tasks.List(ctx, 20)
		if err != nil {
			return "", err
		}
		return renderTasks(items), nil
	}
	switch args[0] {
	case "get":
		if len(args) < 2 {
			return "Usage: /task get [task_id]", nil
		}
		taskID, err := s.resolveTaskID(ctx, args[1])
		if err != nil {
			return err.Error(), nil
		}
		item, err := s.Tasks.Get(ctx, taskID)
		if err != nil {
			return "", err
		}
		if item == nil {
			return "Task not found.", nil
		}
		return fmt.Sprintf("Task\nstatus=%s owner=%s priority=%s\ntitle=%s\ndescription=%s", item.Status, item.OwnerAgent, item.Priority, item.Title, item.Description), nil
	case "status":
		if len(args) < 3 {
			return "Usage: /task status [task_id] [status]", nil
		}
		taskID, err := s.resolveTaskID(ctx, args[1])
		if err != nil {
			return err.Error(), nil
		}
		item, err := s.Tasks.UpdateStatus(ctx, taskID, args[2], actor)
		if err != nil {
			return "", err
		}
		return "Task status updated to " + item.Status, nil
	default:
		return "Usage: /task list|get|status", nil
	}
}

func (s *Services) resolveOptionalTaskID(ctx context.Context, ref string) (string, error) {
	if strings.TrimSpace(ref) == "" {
		return "", nil
	}
	return s.resolveTaskID(ctx, ref)
}

func (s *Services) resolveTaskID(ctx context.Context, ref string) (string, error) {
	idx, ok := parseIndex(ref)
	if !ok {
		return ref, nil
	}
	items, err := s.Tasks.List(ctx, 100)
	if err != nil {
		return "", err
	}
	if idx < 1 || idx > len(items) {
		return "", fmt.Errorf("task number %d is out of range", idx)
	}
	return items[idx-1].ID, nil
}

func (s *Services) resolveApprovalID(ctx context.Context, ref string) (string, error) {
	idx, ok := parseIndex(ref)
	if !ok {
		return ref, nil
	}
	items, err := s.Approvals.List(ctx, "pending")
	if err != nil {
		return "", err
	}
	if idx < 1 || idx > len(items) {
		return "", fmt.Errorf("approval number %d is out of range", idx)
	}
	return items[idx-1].ID, nil
}

func parseIndex(ref string) (int, bool) {
	idx, err := strconv.Atoi(strings.TrimSpace(ref))
	return idx, err == nil && idx > 0
}

func renderProjects(projects []model.Project) string {
	if len(projects) == 0 {
		return "No projects yet."
	}
	var lines []string
	for i, p := range projects {
		lines = append(lines, fmt.Sprintf("%d. %s %s phase=%s", i+1, p.Status, p.Name, p.CurrentPhase))
	}
	return strings.Join(lines, "\n")
}

func renderTasks(tasks []model.Task) string {
	if len(tasks) == 0 {
		return "No tasks yet."
	}
	var lines []string
	for i, t := range tasks {
		lines = append(lines, fmt.Sprintf("%d. %s %s - %s", i+1, t.Status, t.OwnerAgent, t.Title))
	}
	return strings.Join(lines, "\n")
}

func renderApprovals(items []model.Approval) string {
	if len(items) == 0 {
		return "No pending approvals."
	}
	var lines []string
	for i, a := range items {
		project := payloadValue(a.Payload, "project")
		workflowName := payloadValue(a.Payload, "workflow")
		summary := truncateText(payloadValue(a.Payload, "summary"), 110)
		if summary == "" {
			summary = truncateText(a.ItemID, 110)
		}
		parts := []string{fmt.Sprintf("%d. %s %s", i+1, a.RiskLevel, a.ApprovalType)}
		if project != "" {
			parts = append(parts, "project="+project)
		}
		if workflowName != "" {
			parts = append(parts, "workflow="+workflowName)
		}
		parts = append(parts, "summary="+summary)
		lines = append(lines, strings.Join(parts, " "))
	}
	return strings.Join(lines, "\n")
}

type projectGitSnapshot struct {
	Valid          bool
	MainStatus     string
	ClaudeWorktree string
	Error          string
}

func safeAutopilotWorkflowText(text string) string {
	replacements := []struct {
		old string
		new string
	}{
		{"private key", "sensitive credential reference"},
		{"seed phrase", "sensitive recovery phrase reference"},
		{"live trading", "market automation reference"},
		{"production", "runtime environment"},
		{"deploy", "rollout"},
		{"Deploy", "Rollout"},
		{"wallet", "account-service"},
		{"Wallet", "Account-service"},
		{"funds", "balances"},
		{"publish", "prepare"},
		{"merge", "combine"},
		{"部署", "运行更新"},
		{"上线", "运行更新"},
		{"钱包", "账户服务"},
		{"资金", "余额"},
	}
	for _, r := range replacements {
		text = strings.ReplaceAll(text, r.old, r.new)
	}
	return text
}

func collectProjectGitSnapshot(ctx context.Context, workdir string) projectGitSnapshot {
	cmd := exec.CommandContext(ctx, "git", "status", "--short")
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return projectGitSnapshot{Error: llm.SanitizeText(strings.TrimSpace(string(out) + " " + err.Error()))}
	}
	var mainLines []string
	var claudeLines []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, ".claude/worktrees/") {
			claudeLines = append(claudeLines, line)
			continue
		}
		mainLines = append(mainLines, line)
	}
	return projectGitSnapshot{Valid: true, MainStatus: strings.Join(mainLines, "\n"), ClaudeWorktree: strings.Join(claudeLines, "\n")}
}

func (s projectGitSnapshot) MainChangedFrom(before projectGitSnapshot) bool {
	return strings.TrimSpace(s.MainStatus) != strings.TrimSpace(before.MainStatus)
}

func (s projectGitSnapshot) ClaudeWorktreeChangedFrom(before projectGitSnapshot) bool {
	return strings.TrimSpace(s.ClaudeWorktree) != strings.TrimSpace(before.ClaudeWorktree)
}

func gitStatusSummary(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "主工作区无未提交变化。"
	}
	lines := strings.Split(status, "\n")
	if len(lines) > 20 {
		lines = append(lines[:20], fmt.Sprintf("... 还有 %d 行变化", len(lines)-20))
	}
	return strings.Join(lines, "\n")
}

func shortChineseSummary(text string, max int) string {
	text = strings.TrimSpace(llm.SanitizeText(text))
	if text == "" {
		return "暂无有效摘要。"
	}
	text = strings.Join(strings.Fields(text), " ")
	prefixes := []string{"Workflow:", "Agent:", "Task:", "Manual Claude Code prompt:"}
	for _, prefix := range prefixes {
		text = strings.ReplaceAll(text, prefix, "")
	}
	if max <= 0 || len(text) <= max {
		return text
	}
	return text[:max-3] + "..."
}

func summarizeWorkflowArtifacts(res *workflow.Result, max int) string {
	if res == nil || len(res.Artifacts) == 0 {
		return ""
	}
	var parts []string
	for _, item := range res.Artifacts {
		content := strings.TrimSpace(item.Content)
		content = strings.Join(strings.Fields(content), " ")
		if content == "" {
			continue
		}
		parts = append(parts, item.Title+": "+content)
	}
	out := strings.Join(parts, "\n\n")
	if max > 0 && len(out) > max {
		return out[:max-3] + "..."
	}
	return out
}

func claudeLimitWait(res *workflow.Result) (time.Duration, bool) {
	if res == nil {
		return 0, false
	}
	for _, run := range res.Runs {
		details, _ := run.Output["details"].(map[string]interface{})
		if details == nil {
			continue
		}
		if fmt.Sprint(details["error_class"]) != "claude_rate_limited" {
			continue
		}
		seconds := int64(0)
		switch v := details["retry_after_seconds"].(type) {
		case float64:
			seconds = int64(v)
		case int64:
			seconds = v
		case int:
			seconds = int64(v)
		}
		if seconds < 60 {
			seconds = 3600
		}
		return time.Duration(seconds+60) * time.Second, true
	}
	for _, item := range res.Artifacts {
		lower := strings.ToLower(item.Content)
		if strings.Contains(lower, "claude_rate_limited") || (strings.Contains(lower, "limit") && strings.Contains(lower, "reset")) {
			return time.Hour, true
		}
	}
	return 0, false
}

func formatDurationCN(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d 秒", int(d.Seconds()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%d 小时 %d 分钟", h, m)
	}
	return fmt.Sprintf("%d 分钟", m)
}

func deploymentTargetNames(cfg deployment.ProjectConfig) string {
	var names []string
	for _, target := range cfg.DeploymentTargets() {
		if target.Name != "" {
			names = append(names, target.Name)
			continue
		}
		app := pm2AppName(target.DeployCommand)
		if app != "" {
			names = append(names, app)
		} else if len(target.DeployCommand) > 0 {
			names = append(names, strings.Join(target.DeployCommand, " "))
		}
	}
	if len(names) == 0 {
		return "未配置部署服务"
	}
	return strings.Join(names, ", ")
}

func commitProjectChanges(ctx context.Context, workdir, projectName string) (string, error) {
	add := exec.CommandContext(ctx, "git", "add", "-A")
	add.Dir = workdir
	if out, err := add.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git add failed: %s", llm.SanitizeText(strings.TrimSpace(string(out))))
	}
	message := fmt.Sprintf("autopilot: update %s", projectName)
	commit := exec.CommandContext(ctx, "git", "commit", "-m", message)
	commit.Dir = workdir
	if out, err := commit.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git commit failed: %s", llm.SanitizeText(strings.TrimSpace(string(out))))
	}
	rev := exec.CommandContext(ctx, "git", "rev-parse", "--short", "HEAD")
	rev.Dir = workdir
	out, err := rev.CombinedOutput()
	if err != nil {
		return "已提交，但读取 commit id 失败", nil
	}
	return strings.TrimSpace(string(out)), nil
}

func payloadValue(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(payload[key]))
}

func truncateText(text string, max int) string {
	text = strings.Join(strings.Fields(text), " ")
	if max <= 0 || len(text) <= max {
		return text
	}
	if max <= 3 {
		return text[:max]
	}
	return text[:max-3] + "..."
}

func renderAgentRuns(items []model.AgentRun) string {
	if len(items) == 0 {
		return "No agent runs yet."
	}
	var lines []string
	for i, r := range items {
		lines = append(lines, fmt.Sprintf("%d. %s task=%s", i+1, r.Status, r.TaskID))
	}
	return strings.Join(lines, "\n")
}

func renderArtifacts(items []model.Artifact) string {
	if len(items) == 0 {
		return "No artifacts yet."
	}
	var lines []string
	for i, a := range items {
		lines = append(lines, fmt.Sprintf("%d. %s - %s", i+1, a.ArtifactType, a.Title))
	}
	return strings.Join(lines, "\n")
}

type autopilotRun struct {
	projectName string
	startedAt   time.Time
	lastRunAt   time.Time
	lastStatus  string
	nextRunAt   time.Time
	cancel      context.CancelFunc
}

func (s *Services) feedbackCommand(ctx context.Context, args []string, actor string) (string, error) {
	if len(args) < 2 {
		return "Usage: /feedback [project_number_or_name] [feedback text]", nil
	}
	project, err := s.resolveProject(ctx, args[0])
	if err != nil {
		return err.Error(), nil
	}
	content := strings.TrimSpace(strings.Join(args[1:], " "))
	if content == "" {
		return "Feedback text is required.", nil
	}
	_, err = s.Artifacts.Create(ctx, model.Artifact{ProjectID: project.ID, ArtifactType: "user_feedback", Title: project.Name, Content: content, Status: "new", Metadata: map[string]interface{}{"project": project.Name, "actor": actor, "source": "telegram"}})
	if err != nil {
		return "", err
	}
	_ = s.Audit.Log(ctx, audit.Entry{ProjectID: project.ID, Actor: actor, Action: "feedback.recorded", Target: project.ID, RiskLevel: "low", Metadata: map[string]interface{}{"project": project.Name, "source": "telegram"}})
	return "已记录反馈：" + project.Name + ".", nil
}

func (s *Services) autopilotCommand(ctx context.Context, args []string, actor string) (string, error) {
	if len(args) < 1 {
		return "Usage: /autopilot start|status|stop|run [project]", nil
	}
	action := args[0]
	if action == "status" && len(args) == 1 {
		return s.autopilotStatusAll(), nil
	}
	if len(args) < 2 {
		return "Usage: /autopilot start|status|stop|run [project]", nil
	}
	project, err := s.resolveProject(ctx, args[1])
	if err != nil {
		return err.Error(), nil
	}
	switch action {
	case "start":
		return s.startAutopilot(ctx, project, actor)
	case "stop":
		return s.stopAutopilot(project.Name), nil
	case "status":
		return s.autopilotStatus(project.Name), nil
	case "run":
		go s.runAutopilotOnce(ctx, *project, actor)
		return "已启动一次自动调度：" + project.Name + ".", nil
	default:
		return "Usage: /autopilot start|status|stop|run [project]", nil
	}
}

func (s *Services) startAutopilot(ctx context.Context, project *model.Project, actor string) (string, error) {
	s.autopilotMu.Lock()
	if _, exists := s.autopilots[project.Name]; exists {
		s.autopilotMu.Unlock()
		return "自动调度已经在运行：" + project.Name + ".", nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	r := &autopilotRun{projectName: project.Name, startedAt: time.Now(), lastStatus: "starting", cancel: cancel}
	s.autopilots[project.Name] = r
	s.autopilotMu.Unlock()
	go s.autopilotLoop(runCtx, *project, actor)
	return "自动调度已启动：" + project.Name + "。连续推进 project.md 中的下一项任务；日志只用于错误诊断。遇到 Claude limit 会等 reset 后继续，只有主代码目录发生变化时才会提交/部署。", nil
}

func (s *Services) stopAutopilot(projectName string) string {
	s.autopilotMu.Lock()
	defer s.autopilotMu.Unlock()
	r, ok := s.autopilots[projectName]
	if !ok {
		return "自动调度未运行：" + projectName + "."
	}
	r.cancel()
	delete(s.autopilots, projectName)
	return "自动调度已停止：" + projectName + "."
}

func (s *Services) autopilotStatus(projectName string) string {
	s.autopilotMu.Lock()
	defer s.autopilotMu.Unlock()
	r, ok := s.autopilots[projectName]
	if !ok {
		return "自动调度未运行：" + projectName + "."
	}
	return fmt.Sprintf("自动调度 %s 运行中。启动=%s，上次运行=%s，下次运行=%s，状态=%s", projectName, r.startedAt.Format(time.RFC3339), formatTime(r.lastRunAt), formatTime(r.nextRunAt), r.lastStatus)
}

func (s *Services) autopilotStatusAll() string {
	s.autopilotMu.Lock()
	defer s.autopilotMu.Unlock()
	if len(s.autopilots) == 0 {
		return "当前没有运行中的自动调度。"
	}
	var lines []string
	for name, r := range s.autopilots {
		lines = append(lines, fmt.Sprintf("%s 状态=%s 上次运行=%s 下次运行=%s", name, r.lastStatus, formatTime(r.lastRunAt), formatTime(r.nextRunAt)))
	}
	return strings.Join(lines, "\n")
}

func (s *Services) autopilotLoop(ctx context.Context, project model.Project, actor string) {
	for {
		wait := s.runAutopilotOnce(ctx, project, actor)
		if wait <= 0 {
			wait = 2 * time.Minute
		}
		s.setAutopilotNextRun(project.Name, time.Now().Add(wait))
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

func (s *Services) runAutopilotOnce(ctx context.Context, project model.Project, actor string) time.Duration {
	s.setAutopilotStatus(project.Name, "running")
	workflow.Report(ctx, "开始："+project.Name+"，按 project.md 推进下一项。")
	cfg, err := s.getProjectConfig(ctx, project.Name)
	if err != nil || cfg == nil {
		s.setAutopilotStatus(project.Name, "missing project config")
		workflow.Report(ctx, "自动调度停止：没有找到 "+project.Name+" 的项目配置。请先发送 /project config。")
		return 10 * time.Minute
	}
	workflow.Report(ctx, fmt.Sprintf("配置：提交=%t 部署=%t 服务=%s", cfg.AutoCommit, cfg.AutoDeploy, deploymentTargetNames(*cfg)))
	feedback := s.collectFeedback(ctx, project.Name)
	logs := s.collectLocalLogs(ctx, *cfg)
	if strings.TrimSpace(logs) != "" && !strings.Contains(logs, "未发现关键") {
		workflow.Report(ctx, "日志："+shortChineseSummary(logs, 50))
	}
	before := collectProjectGitSnapshot(ctx, cfg.Workdir)
	if before.Valid && strings.TrimSpace(before.MainStatus) != "" {
		workflow.Report(ctx, "注意：本轮开始前主工作区已有未提交改动。bot 不会自动 commit，避免混入人工改动。\n"+truncateText(before.MainStatus, 600))
	}
	workflow.Report(ctx, "规划：读取文档、反馈和关键错误。")
	contextText := fmt.Sprintf("Project: %s\nWorkdir: %s\nDoc: %s\nRecent Telegram feedback:\n%s\nRecent local error logs for diagnostics only:\n%s", project.Name, cfg.Workdir, cfg.DocPath, feedback, logs)
	compactCtx := workflow.WithCompactProgress(ctx)
	planResult, _ := s.Workflows.Plan(compactCtx, safeAutopilotWorkflowText(contextText)+"\nCTO must produce the near-term roadmap, assumptions, and task breakdown. Treat logs as diagnostics only; do not change product direction based solely on logs.", actor)
	planSummary := summarizeWorkflowArtifacts(planResult, 3000)
	if planSummary != "" {
		workflow.Report(ctx, "规划："+shortChineseSummary(planSummary, 50))
	}
	workflow.Report(ctx, "编码：按规划修改主项目代码。")
	buildPrompt := strings.Join([]string{
		"Project " + project.Name + ": work directly in " + cfg.Workdir + ".",
		"Read " + cfg.DocPath + " plus existing code.",
		"Implement the highest-priority concrete code changes from this CTO/product/backend/frontend/QA plan:",
		planSummary,
		"You may modify any file under " + cfg.Workdir + ".",
		"Do not create .claude/worktrees, background agents, branches, commits, pull requests, or touch files outside that directory.",
		"If you decide no code change is needed, explain the exact blocker and the exact next implementation task.",
	}, "\n")
	buildResult, _ := s.Workflows.Build(compactCtx, buildPrompt, actor)
	if buildSummary := summarizeWorkflowArtifacts(buildResult, 500); buildSummary != "" {
		workflow.Report(ctx, "编码："+shortChineseSummary(buildSummary, 50))
	}
	after := collectProjectGitSnapshot(ctx, cfg.Workdir)
	if !before.Valid {
		workflow.Report(ctx, "无法读取本轮开始前 git 状态，出于安全考虑跳过提交和部署："+before.Error)
		s.setAutopilotStatus(project.Name, "idle")
		return 10 * time.Minute
	}
	if !after.Valid {
		workflow.Report(ctx, "无法读取本轮结束后 git 状态，出于安全考虑跳过提交和部署："+after.Error)
		s.setAutopilotStatus(project.Name, "idle")
		return 10 * time.Minute
	}
	if !after.MainChangedFrom(before) {
		if after.ClaudeWorktreeChangedFrom(before) {
			workflow.Report(ctx, "Claude 写入临时目录，主项目未变。")
		} else {
			workflow.Report(ctx, "无代码变化，跳过提交和部署。")
		}
		_, _ = s.Workflows.Review(compactCtx, "Project "+project.Name+": no main checkout code change was detected after the latest build attempt. Review why implementation did not modify "+cfg.Workdir+" and identify the next concrete coding task.", actor)
		s.setAutopilotStatus(project.Name, "idle")
		if wait, ok := claudeLimitWait(buildResult); ok {
			workflow.Report(ctx, "Claude 额度已达上限，等待到 reset 后继续。预计等待："+formatDurationCN(wait))
			return wait
		}
		return 8 * time.Minute
	}
	workflow.Report(ctx, "代码变化：\n"+gitStatusSummary(after.MainStatus))
	_, _ = s.Workflows.Review(compactCtx, "Project "+project.Name+": review the latest local implementation in "+cfg.Workdir+" and identify readiness blockers before release approval.", actor)
	changedThisRun := true
	if cfg.AutoCommit {
		if strings.TrimSpace(before.MainStatus) != "" {
			workflow.Report(ctx, "auto_commit=true，但本轮开始前已有未提交改动；为避免混入人工改动，本轮不自动 commit。")
		} else {
			commitID, err := commitProjectChanges(ctx, cfg.Workdir, project.Name)
			if err != nil {
				workflow.Report(ctx, "自动 commit 失败，已停止部署："+err.Error())
				s.setAutopilotStatus(project.Name, "idle")
				return 10 * time.Minute
			}
			workflow.Report(ctx, "代码已提交："+commitID)
			after = collectProjectGitSnapshot(ctx, cfg.Workdir)
		}
	} else {
		workflow.Report(ctx, "代码尚未 commit。需要自动提交请配置 auto_commit=true；当前保留为工作区改动供你检查。")
	}
	if changedThisRun && cfg.AutoDeploy {
		workflow.Report(ctx, "部署："+deploymentTargetNames(*cfg))
		result, err := s.Deployment.Execute(ctx, *cfg)
		if err != nil {
			workflow.Report(ctx, "部署失败："+err.Error())
		} else if result.Status != "completed" {
			workflow.Report(ctx, "部署失败："+result.ErrorClass)
		} else {
			workflow.Report(ctx, "部署完成："+project.Name+"。")
		}
		_ = s.Audit.Log(ctx, audit.Entry{ProjectID: project.ID, Actor: actor, Action: "autopilot.deployment", Target: project.ID, RiskLevel: "high", Metadata: map[string]interface{}{"external_execution": true, "project": project.Name}})
	} else if changedThisRun && !s.pendingDeployApprovalExists(ctx, project.Name) {
		_, _ = s.deployCommand(ctx, []string{project.Name, "Autopilot release candidate after code changes"}, actor)
		workflow.Report(ctx, "已改代码，已创建部署审批。")
	} else if changedThisRun {
		workflow.Report(ctx, "已有部署审批，不重复创建。")
	}
	s.setAutopilotStatus(project.Name, "idle")
	return 90 * time.Second
}

func (s *Services) setAutopilotStatus(projectName, status string) {
	s.autopilotMu.Lock()
	defer s.autopilotMu.Unlock()
	if r, ok := s.autopilots[projectName]; ok {
		r.lastStatus = status
		r.lastRunAt = time.Now()
	}
}

func (s *Services) setAutopilotNextRun(projectName string, next time.Time) {
	s.autopilotMu.Lock()
	defer s.autopilotMu.Unlock()
	if r, ok := s.autopilots[projectName]; ok {
		r.nextRunAt = next
		if time.Until(next) > 5*time.Minute {
			r.lastStatus = "waiting"
		}
	}
}

func (s *Services) collectFeedback(ctx context.Context, projectName string) string {
	items, err := s.Artifacts.ListByType(ctx, "user_feedback", projectName, 10)
	if err != nil || len(items) == 0 {
		return "No recent Telegram feedback."
	}
	var lines []string
	for _, item := range items {
		lines = append(lines, "- "+item.Content)
	}
	return strings.Join(lines, "\n")
}

func (s *Services) collectLocalLogs(ctx context.Context, cfg deployment.ProjectConfig) string {
	var chunks []string
	seen := map[string]bool{}
	for _, target := range cfg.DeploymentTargets() {
		appName := pm2AppName(target.DeployCommand)
		if appName == "" || seen[appName] {
			continue
		}
		seen[appName] = true
		cmd := exec.CommandContext(ctx, "pm2", "logs", appName, "--lines", "120", "--nostream")
		cmd.Dir = cfg.Workdir
		out, err := cmd.CombinedOutput()
		if err != nil {
			chunks = append(chunks, appName+": 日志采集失败："+llm.SanitizeText(err.Error()))
			continue
		}
		text := extractDiagnosticLogLines(llm.SanitizeText(string(out)))
		if text != "" {
			chunks = append(chunks, appName+":\n"+text)
		}
	}
	if len(chunks) == 0 {
		return "未发现关键接口或运行错误。"
	}
	return strings.Join(chunks, "\n")
}

func extractDiagnosticLogLines(text string) string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || ignoreNoisyLogLine(trimmed) {
			continue
		}
		if isImportantLogLine(trimmed) {
			lines = append(lines, trimmed)
		}
	}
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	return strings.Join(lines, "\n")
}

func ignoreNoisyLogLine(line string) bool {
	lower := strings.ToLower(line)
	noisy := []string{
		"[gin-debug] [warning] running in \"debug\" mode",
		"[gin-debug]",
		"debug mode",
		"trusted proxies",
		"listening and serving",
		"telegram command received",
	}
	for _, item := range noisy {
		if strings.Contains(lower, item) {
			return true
		}
	}
	return false
}

func isImportantLogLine(line string) bool {
	lower := strings.ToLower(line)
	if containsHTTPProblemStatus(lower) {
		return true
	}
	keywords := []string{"error", "failed", "failure", "panic", "exception", "fatal", "timeout", "refused", "not found", "错误", "失败", "异常"}
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func containsHTTPProblemStatus(line string) bool {
	statuses := []string{" 400 ", " 401 ", " 403 ", " 404 ", " 405 ", " 408 ", " 409 ", " 422 ", " 429 ", " 500 ", " 502 ", " 503 ", " 504 ", "| 400 |", "| 401 |", "| 403 |", "| 404 |", "| 405 |", "| 408 |", "| 409 |", "| 422 |", "| 429 |", "| 500 |", "| 502 |", "| 503 |", "| 504 |"}
	for _, status := range statuses {
		if strings.Contains(line, status) {
			return true
		}
	}
	return false
}

func pm2AppName(command []string) string {
	if len(command) >= 3 && command[0] == "pm2" && (command[1] == "restart" || command[1] == "reload" || command[1] == "start") {
		return command[2]
	}
	return ""
}

func (s *Services) pendingDeployApprovalExists(ctx context.Context, projectName string) bool {
	items, err := s.Approvals.List(ctx, "pending")
	if err != nil {
		return false
	}
	for _, a := range items {
		if a.ApprovalType == "deploy_production" && strings.EqualFold(fmt.Sprint(a.Payload["project"]), projectName) {
			return true
		}
	}
	return false
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format(time.RFC3339)
}

func (s *Services) deployCommand(ctx context.Context, args []string, actor string) (string, error) {
	if len(args) < 1 {
		return "Usage: /deploy [project_number_or_name] [reason]", nil
	}
	project, err := s.resolveProject(ctx, args[0])
	if err != nil {
		return err.Error(), nil
	}
	cfg, err := s.getProjectConfig(ctx, project.Name)
	if err != nil {
		return "", err
	}
	if cfg == nil {
		return "No project config found. Use /project config first.", nil
	}
	reason := strings.TrimSpace(strings.Join(args[1:], " "))
	if reason == "" {
		reason = "Deploy project " + project.Name
	}
	payload := map[string]interface{}{
		"action_type":                 "deploy_production",
		"requester":                   actor,
		"project":                     project.Name,
		"risk_level":                  "high",
		"summary":                     reason,
		"evidence":                    "manual /deploy request",
		"expected_impact":             "Runs the configured deployment service commands in the configured project workdir.",
		"rollback_or_mitigation_plan": "Stop or revert the deployed PM2/service process using the project's operational runbook.",
		"required_approver":           "founder",
		"execution":                   map[string]interface{}{"kind": "deploy_command"},
		"project_config":              cfg,
	}
	_, err = s.Approvals.Create(ctx, model.Approval{ProjectID: project.ID, ApprovalType: "deploy_production", ItemType: "project", ItemID: project.ID, RequestedBy: actor, RiskLevel: "high", Payload: payload})
	if err != nil {
		return "", err
	}
	return "已创建部署审批。发送 /approvals 查看编号，再发送 /approve [编号] 执行所有配置服务。", nil
}

func (s *Services) ApproveApproval(ctx context.Context, id, actor string) (*model.Approval, *deployment.Result, error) {
	a, err := s.Approvals.Approve(ctx, id, actor)
	if err != nil {
		return nil, nil, err
	}
	if a.ApprovalType != "deploy_production" || payloadString(a.Payload, "execution", "kind") != "deploy_command" {
		return a, nil, nil
	}
	cfg, err := projectConfigFromPayload(a.Payload["project_config"])
	if err != nil {
		return a, &deployment.Result{Status: "failed", ErrorClass: "invalid_project_config"}, nil
	}
	result, err := s.Deployment.Execute(ctx, cfg)
	if err != nil {
		return a, nil, err
	}
	_ = s.Audit.Log(ctx, audit.Entry{ProjectID: a.ProjectID, Actor: actor, Action: "deployment." + result.Status, Target: a.ID, RiskLevel: a.RiskLevel, Metadata: map[string]interface{}{"external_execution": true, "error_class": result.ErrorClass, "project": cfg.ProjectName}})
	return a, result, nil
}

func (s *Services) resolveProject(ctx context.Context, ref string) (*model.Project, error) {
	items, err := s.Projects.List(ctx)
	if err != nil {
		return nil, err
	}
	if idx, ok := parseIndex(ref); ok {
		if idx < 1 || idx > len(items) {
			return nil, fmt.Errorf("project number %d is out of range", idx)
		}
		return &items[idx-1], nil
	}
	for _, p := range items {
		if strings.EqualFold(p.Name, ref) || p.ID == ref {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("project not found")
}

func (s *Services) resolveOrCreateProject(ctx context.Context, ref, actor string) (*model.Project, error) {
	project, err := s.resolveProject(ctx, ref)
	if err == nil {
		return project, nil
	}
	if _, ok := parseIndex(ref); ok {
		return nil, err
	}
	return s.CreateProject(ctx, model.Project{Name: ref, Owner: actor}, actor)
}

func (s *Services) getProjectConfig(ctx context.Context, projectName string) (*deployment.ProjectConfig, error) {
	items, err := s.Artifacts.ListByType(ctx, "project_config", projectName, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	var cfg deployment.ProjectConfig
	if err := json.Unmarshal([]byte(items[0].Content), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func parseProjectConfigArgs(projectName string, args []string, defaultRoot string) (deployment.ProjectConfig, error) {
	cfg := deployment.ProjectConfig{ProjectName: projectName, AllowedRoot: defaultRoot, DocPath: "project.md"}
	for i := 0; i < len(args); i++ {
		part := args[i]
		switch {
		case strings.HasPrefix(part, "workdir="):
			cfg.Workdir = strings.TrimPrefix(part, "workdir=")
		case strings.HasPrefix(part, "doc="):
			cfg.DocPath = strings.TrimPrefix(part, "doc=")
		case strings.HasPrefix(part, "root="):
			cfg.AllowedRoot = strings.TrimPrefix(part, "root=")
		case strings.HasPrefix(part, "auto_deploy="):
			value := strings.TrimPrefix(part, "auto_deploy=")
			cfg.AutoDeploy = parseBoolConfig(value)
		case strings.HasPrefix(part, "auto_commit="):
			value := strings.TrimPrefix(part, "auto_commit=")
			cfg.AutoCommit = parseBoolConfig(value)
		case strings.HasPrefix(part, "service="):
			target, next, err := parseServiceTarget(args, i)
			if err != nil {
				return cfg, err
			}
			cfg.Services = append(cfg.Services, target)
			i = next - 1
		case strings.HasPrefix(part, "deploy="):
			cmd, next := parseCommandValue(strings.TrimPrefix(part, "deploy="), args, i+1)
			cfg.DeployCommand = cmd
			i = next - 1
		default:
			return cfg, fmt.Errorf("unknown config field %q", part)
		}
	}
	return cfg, nil
}

func parseServiceTarget(args []string, start int) (deployment.Target, int, error) {
	value := strings.TrimPrefix(args[start], "service=")
	name, first, ok := strings.Cut(value, ":")
	name = strings.TrimSpace(name)
	if name == "" || !ok {
		return deployment.Target{}, start + 1, fmt.Errorf("service must use service=name:command")
	}
	cmd, next := parseCommandValue(first, args, start+1)
	if len(cmd) == 0 {
		return deployment.Target{}, next, fmt.Errorf("service %s deploy command is required", name)
	}
	return deployment.Target{Name: name, DeployCommand: cmd}, next, nil
}

func parseCommandValue(first string, args []string, start int) ([]string, int) {
	cmd := []string{}
	if first != "" {
		cmd = append(cmd, first)
	}
	i := start
	for i < len(args) && !isProjectConfigField(args[i]) {
		cmd = append(cmd, args[i])
		i++
	}
	return cmd, i
}

func isProjectConfigField(value string) bool {
	return strings.HasPrefix(value, "workdir=") || strings.HasPrefix(value, "doc=") || strings.HasPrefix(value, "root=") || strings.HasPrefix(value, "auto_deploy=") || strings.HasPrefix(value, "auto_commit=") || strings.HasPrefix(value, "deploy=") || strings.HasPrefix(value, "service=")
}

func parseBoolConfig(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "true" || value == "1" || value == "yes" || value == "on"
}

func renderProjectConfig(cfg deployment.ProjectConfig) string {
	lines := []string{
		fmt.Sprintf("Project config %s", cfg.ProjectName),
		"workdir=" + cfg.Workdir,
		"doc=" + cfg.DocPath,
		"allowed_root=" + cfg.AllowedRoot,
		fmt.Sprintf("auto_deploy=%t", cfg.AutoDeploy),
		fmt.Sprintf("auto_commit=%t", cfg.AutoCommit),
	}
	if len(cfg.Services) > 0 {
		for _, target := range cfg.Services {
			lines = append(lines, fmt.Sprintf("service=%s deploy=%s", target.Name, strings.Join(target.DeployCommand, " ")))
		}
	} else {
		lines = append(lines, "deploy="+strings.Join(cfg.DeployCommand, " "))
	}
	return strings.Join(lines, "\n")
}

func projectConfigFromPayload(raw interface{}) (deployment.ProjectConfig, error) {
	var cfg deployment.ProjectConfig
	b, err := json.Marshal(raw)
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(b, &cfg)
	return cfg, err
}

func payloadString(payload map[string]interface{}, keys ...string) string {
	var current interface{} = payload
	for _, key := range keys {
		m, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current = m[key]
	}
	value, _ := current.(string)
	return value
}

func (s *Services) workflowCommand(ctx context.Context, kind string, args []string, actor string) (string, error) {
	text := strings.TrimSpace(strings.Join(args, " "))
	if text == "" {
		return "Usage: /" + kind + " [text]", nil
	}
	var res *workflow.Result
	var err error
	switch kind {
	case "plan":
		res, err = s.Workflows.Plan(ctx, text, actor)
	case "build":
		res, err = s.Workflows.Build(ctx, text, actor)
	case "launch":
		res, err = s.Workflows.Launch(ctx, text, actor)
	case "review":
		res, err = s.Workflows.Review(ctx, text, actor)
	}
	if err != nil {
		return "", err
	}
	msg := fmt.Sprintf("%s workflow 已创建任务。发送 /tasks 查看编号。风险：%s。产物：%d。", res.Workflow, res.Risk.Level, len(res.Artifacts))
	if res.Approval != nil {
		msg += " 已创建审批。发送 /approvals 查看编号；未执行任何外部动作。"
	}
	return msg, nil
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
