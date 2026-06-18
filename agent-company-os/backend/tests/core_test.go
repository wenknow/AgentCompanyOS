package tests

import (
	"context"
	"testing"
	"time"

	"github.com/agentcompany/agent-company-os/backend/internal/agents/builtin"
	"github.com/agentcompany/agent-company-os/backend/internal/agents/registry"
	"github.com/agentcompany/agent-company-os/backend/internal/agents/runtime"
	"github.com/agentcompany/agent-company-os/backend/internal/approval"
	"github.com/agentcompany/agent-company-os/backend/internal/audit"
	"github.com/agentcompany/agent-company-os/backend/internal/command"
	"github.com/agentcompany/agent-company-os/backend/internal/model"
	"github.com/agentcompany/agent-company-os/backend/internal/project"
	"github.com/agentcompany/agent-company-os/backend/internal/risk"
	"github.com/agentcompany/agent-company-os/backend/internal/task"
	"github.com/agentcompany/agent-company-os/backend/internal/workflow"
)

func TestCommandParser(t *testing.T) {
	cmd := command.Parse("/assign backend 设计 schema", 10, 20)
	if cmd.Name != "assign" || cmd.ChatID != 10 || cmd.UserID != 20 || cmd.RawText == "" {
		t.Fatalf("unexpected command: %#v", cmd)
	}
	if len(cmd.Args) != 3 || cmd.Args[0] != "backend" {
		t.Fatalf("unexpected args: %#v", cmd.Args)
	}
}

func TestRiskDetector(t *testing.T) {
	cases := []struct {
		text         string
		level        string
		approvalType string
	}{
		{"发布公告", "high", "publish_content"},
		{"deploy production", "high", "deploy_production"},
		{"enable live trading", "critical", "enable_live_trading"},
		{"访问钱包私钥", "critical", "access_sensitive_data"},
		{"普通数据库设计", "low", ""},
	}
	for _, tc := range cases {
		got := risk.Detect(tc.text)
		if got.Level != tc.level || got.ApprovalType != tc.approvalType {
			t.Fatalf("Detect(%q)=%#v", tc.text, got)
		}
	}
}

func TestBuiltinAgents(t *testing.T) {
	agents := builtin.Agents()
	if len(agents) != 14 {
		t.Fatalf("expected 14 agents, got %d", len(agents))
	}
	seen := map[string]bool{}
	for _, a := range agents {
		seen[a.Name] = true
		if len(a.Permissions) == 0 {
			t.Fatalf("agent %s has no permissions", a.Name)
		}
	}
	for _, name := range []string{"chief_of_staff", "product", "cto", "backend", "designer", "frontend", "qa", "devops", "content", "growth", "sales", "finance", "compliance", "coding"} {
		if !seen[name] {
			t.Fatalf("missing built-in agent %s", name)
		}
	}
}

func TestApprovalServiceApproveRejectNoExternalExecution(t *testing.T) {
	ctx := context.Background()
	repo := newFakeApprovals()
	auditRepo := &fakeAudit{}
	svc := approval.NewService(repo, auditRepo)
	created, err := svc.Create(ctx, model.Approval{ProjectID: "project-1", ApprovalType: "deploy_production", RequestedBy: "founder", RiskLevel: "high", Payload: map[string]interface{}{"action_type": "deploy_production"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Approve(ctx, created.ID, "founder"); err != nil {
		t.Fatal(err)
	}
	if repo.items[created.ID].ApprovalStatus != "approved" {
		t.Fatalf("approval not approved")
	}
	if _, err := svc.Reject(ctx, created.ID, "founder", "not ready"); err != nil {
		t.Fatal(err)
	}
	if repo.items[created.ID].Reason != "not ready" {
		t.Fatalf("reject reason not saved")
	}
	if len(auditRepo.entries) < 3 {
		t.Fatalf("expected audit entries, got %d", len(auditRepo.entries))
	}
}

func TestWorkflowServiceBuildCreatesPromptArtifactAndApprovalForHighRisk(t *testing.T) {
	ctx := context.Background()
	agents := &fakeAgents{items: map[string]model.Agent{"coding": {ID: "agent-coding", Name: "coding", Role: "Coding Agent", Status: "active"}}}
	projects := &fakeProjects{project: model.Project{ID: "project-1", Name: "AgentCompanyOS"}}
	tasks := &fakeTasks{}
	approvals := newFakeApprovals()
	auditRepo := &fakeAudit{}
	artifacts := &fakeArtifacts{}
	approvalSvc := approval.NewService(approvals, auditRepo)
	rt := runtime.NewRouterRuntime(runtime.RouterConfig{DeepSeekRuntime: runtime.NewRuleBasedRuntime(), CodingRuntime: "claude_code_local", ClaudeCode: runtime.NewClaudeCodeAdapter(runtime.ClaudeCodeConfig{Command: "missing-claude", Enabled: false, Workdir: "."})})
	svc := workflow.NewService(projects, agents, tasks, approvalSvc, auditRepo, artifacts, rt, "AgentCompanyOS")
	res, err := svc.Build(ctx, "deploy production API", "founder")
	if err != nil {
		t.Fatal(err)
	}
	if res.Approval == nil || res.Task.Status != "needs_founder_approval" {
		t.Fatalf("expected high-risk build approval, got %#v", res)
	}
	if len(res.Artifacts) != 1 || len(tasks.runs) != 1 {
		t.Fatalf("expected artifact and run, got artifacts=%d runs=%d", len(res.Artifacts), len(tasks.runs))
	}
	if len(auditRepo.entries) == 0 || len(tasks.events) == 0 {
		t.Fatalf("expected audit and task events")
	}
}

func TestTaskServiceAssignmentCreatesApprovalForHighRisk(t *testing.T) {
	ctx := context.Background()
	agents := &fakeAgents{items: map[string]model.Agent{"backend": {ID: "agent-1", Name: "backend", Role: "Backend Agent", Status: "active"}}}
	projects := &fakeProjects{project: model.Project{ID: "project-1", Name: "AgentCompanyOS"}}
	tasks := &fakeTasks{}
	approvals := newFakeApprovals()
	auditRepo := &fakeAudit{}
	approvalSvc := approval.NewService(approvals, auditRepo)
	svc := task.NewService(tasks, projects, agents, approvalSvc, auditRepo, runtime.NewRuleBasedRuntime(), "AgentCompanyOS")
	res, err := svc.Assign(ctx, "backend", "deploy production API", "founder")
	if err != nil {
		t.Fatal(err)
	}
	if res.Approval == nil {
		t.Fatalf("expected approval for high-risk task")
	}
	if res.Task.Status != "assigned" {
		t.Fatalf("unexpected task status %s", res.Task.Status)
	}
	if len(tasks.runs) != 1 {
		t.Fatalf("expected one agent run")
	}
}

type fakeAudit struct{ entries []audit.Entry }

func (f *fakeAudit) Log(ctx context.Context, entry audit.Entry) error {
	f.entries = append(f.entries, entry)
	return nil
}

type fakeApprovals struct{ items map[string]model.Approval }

func newFakeApprovals() *fakeApprovals { return &fakeApprovals{items: map[string]model.Approval{}} }
func (f *fakeApprovals) Create(ctx context.Context, a model.Approval) (*model.Approval, error) {
	if a.ID == "" {
		a.ID = "approval-1"
	}
	a.ApprovalStatus = "pending"
	a.CreatedAt = time.Now()
	f.items[a.ID] = a
	return &a, nil
}
func (f *fakeApprovals) List(ctx context.Context, status string) ([]model.Approval, error) {
	var out []model.Approval
	for _, a := range f.items {
		if status == "" || a.ApprovalStatus == status {
			out = append(out, a)
		}
	}
	return out, nil
}
func (f *fakeApprovals) CountPending(ctx context.Context) (int, error) {
	n := 0
	for _, a := range f.items {
		if a.ApprovalStatus == "pending" {
			n++
		}
	}
	return n, nil
}
func (f *fakeApprovals) UpdateStatus(ctx context.Context, id, status, actor, reason string) (*model.Approval, error) {
	a := f.items[id]
	a.ApprovalStatus = status
	a.ApprovedBy = actor
	a.Reason = reason
	f.items[id] = a
	return &a, nil
}

type fakeAgents struct{ items map[string]model.Agent }

func (f *fakeAgents) SeedBuiltins(ctx context.Context) error { return nil }
func (f *fakeAgents) List(ctx context.Context) ([]model.Agent, error) {
	var out []model.Agent
	for _, a := range f.items {
		out = append(out, a)
	}
	return out, nil
}
func (f *fakeAgents) GetByName(ctx context.Context, name string) (*model.Agent, error) {
	a, ok := f.items[name]
	if !ok {
		return nil, nil
	}
	return &a, nil
}
func (f *fakeAgents) Exists(ctx context.Context, name string) (bool, error) {
	_, ok := f.items[name]
	return ok, nil
}
func (f *fakeAgents) ActiveCount(ctx context.Context) (int, error) { return len(f.items), nil }

var _ registry.Repository = (*fakeAgents)(nil)

type fakeProjects struct{ project model.Project }

func (f *fakeProjects) GetOrCreateDefault(ctx context.Context, name string) (*model.Project, error) {
	return &f.project, nil
}
func (f *fakeProjects) Create(ctx context.Context, p model.Project) (*model.Project, error) {
	f.project = p
	return &f.project, nil
}
func (f *fakeProjects) List(ctx context.Context) ([]model.Project, error) {
	return []model.Project{f.project}, nil
}
func (f *fakeProjects) Count(ctx context.Context) (int, error) { return 1, nil }

var _ project.Repository = (*fakeProjects)(nil)

type fakeTasks struct {
	items  []model.Task
	events []string
	runs   []string
}

func (f *fakeTasks) Create(ctx context.Context, t model.Task) (*model.Task, error) {
	t.ID = "task-1"
	t.CreatedAt = time.Now()
	f.items = append(f.items, t)
	return &t, nil
}
func (f *fakeTasks) List(ctx context.Context, limit int) ([]model.Task, error) { return f.items, nil }
func (f *fakeTasks) Get(ctx context.Context, id string) (*model.Task, error)   { return &f.items[0], nil }
func (f *fakeTasks) UpdateStatus(ctx context.Context, id, status string) (*model.Task, error) {
	f.items[0].Status = status
	return &f.items[0], nil
}
func (f *fakeTasks) AddEvent(ctx context.Context, taskID, eventType, actor, message string, metadata map[string]interface{}) error {
	f.events = append(f.events, eventType)
	return nil
}
func (f *fakeTasks) RecordAgentRun(ctx context.Context, agentID, projectID, taskID string, input, output map[string]interface{}) error {
	f.runs = append(f.runs, taskID)
	return nil
}
func (f *fakeTasks) ListAgentRuns(ctx context.Context, taskID string, limit int) ([]model.AgentRun, error) {
	var out []model.AgentRun
	for _, id := range f.runs {
		if taskID == "" || taskID == id {
			out = append(out, model.AgentRun{ID: "run-" + id, TaskID: id, Status: "completed"})
		}
	}
	return out, nil
}
func (f *fakeTasks) Count(ctx context.Context) (int, error)        { return len(f.items), nil }
func (f *fakeTasks) BlockedCount(ctx context.Context) (int, error) { return 0, nil }

var _ task.Repository = (*fakeTasks)(nil)

type fakeArtifacts struct{ items []model.Artifact }

func (f *fakeArtifacts) Create(ctx context.Context, a model.Artifact) (*model.Artifact, error) {
	if a.ID == "" {
		a.ID = "artifact-1"
	}
	a.CreatedAt = time.Now()
	a.UpdatedAt = a.CreatedAt
	f.items = append(f.items, a)
	return &a, nil
}
func (f *fakeArtifacts) List(ctx context.Context, taskID string, limit int) ([]model.Artifact, error) {
	var out []model.Artifact
	for _, a := range f.items {
		if taskID == "" || a.TaskID == taskID {
			out = append(out, a)
		}
	}
	return out, nil
}
func (f *fakeArtifacts) ListByType(ctx context.Context, artifactType, title string, limit int) ([]model.Artifact, error) {
	var out []model.Artifact
	for _, a := range f.items {
		if a.ArtifactType == artifactType && (title == "" || a.Title == title) {
			out = append(out, a)
		}
	}
	return out, nil
}
func (f *fakeArtifacts) Get(ctx context.Context, id string) (*model.Artifact, error) {
	for _, a := range f.items {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, nil
}
