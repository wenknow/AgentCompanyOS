package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/agentcompany/agent-company-os/backend/internal/agents/registry"
	"github.com/agentcompany/agent-company-os/backend/internal/agents/runtime"
	"github.com/agentcompany/agent-company-os/backend/internal/approval"
	"github.com/agentcompany/agent-company-os/backend/internal/audit"
	"github.com/agentcompany/agent-company-os/backend/internal/llm"
	"github.com/agentcompany/agent-company-os/backend/internal/model"
	"github.com/agentcompany/agent-company-os/backend/internal/project"
	"github.com/agentcompany/agent-company-os/backend/internal/risk"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, t model.Task) (*model.Task, error)
	List(ctx context.Context, limit int) ([]model.Task, error)
	Get(ctx context.Context, id string) (*model.Task, error)
	UpdateStatus(ctx context.Context, id, status string) (*model.Task, error)
	AddEvent(ctx context.Context, taskID, eventType, actor, message string, metadata map[string]interface{}) error
	RecordAgentRun(ctx context.Context, agentID, projectID, taskID string, input, output map[string]interface{}) error
	ListAgentRuns(ctx context.Context, taskID string, limit int) ([]model.AgentRun, error)
	Count(ctx context.Context) (int, error)
	BlockedCount(ctx context.Context) (int, error)
}

type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository { return &PostgresRepository{db: db} }

func (r *PostgresRepository) Create(ctx context.Context, t model.Task) (*model.Task, error) {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	err := r.db.QueryRow(ctx, `insert into tasks (id,project_id,title,description,owner_agent,priority,status,due_date,created_by)
		values ($1,$2,$3,$4,$5,coalesce(nullif($6,''),'P2'),coalesce(nullif($7,''),'assigned'),$8,$9)
		returning id::text,project_id::text,title,coalesce(description,''),coalesce(owner_agent,''),priority,status,due_date,coalesce(created_by,''),created_at`,
		t.ID, t.ProjectID, t.Title, t.Description, t.OwnerAgent, t.Priority, t.Status, t.DueDate, t.CreatedBy).
		Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.OwnerAgent, &t.Priority, &t.Status, &t.DueDate, &t.CreatedBy, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) List(ctx context.Context, limit int) ([]model.Task, error) {
	rows, err := r.db.Query(ctx, `select id::text,project_id::text,title,coalesce(description,''),coalesce(owner_agent,''),priority,status,due_date,coalesce(created_by,''),created_at from tasks order by created_at desc limit $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()
	var out []model.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (*model.Task, error) {
	t, err := scanTask(r.db.QueryRow(ctx, `select id::text,project_id::text,title,coalesce(description,''),coalesce(owner_agent,''),priority,status,due_date,coalesce(created_by,''),created_at from tasks where id=$1`, id))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id, status string) (*model.Task, error) {
	t, err := scanTask(r.db.QueryRow(ctx, `update tasks set status=$2, updated_at=now() where id=$1 returning id::text,project_id::text,title,coalesce(description,''),coalesce(owner_agent,''),priority,status,due_date,coalesce(created_by,''),created_at`, id, status))
	if err != nil {
		return nil, fmt.Errorf("update task status: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) AddEvent(ctx context.Context, taskID, eventType, actor, message string, metadata map[string]interface{}) error {
	meta, _ := json.Marshal(metadata)
	_, err := r.db.Exec(ctx, `insert into task_events (id,task_id,event_type,actor,message,metadata) values ($1,$2,$3,$4,$5,$6)`, uuid.NewString(), taskID, eventType, actor, message, meta)
	return err
}

func (r *PostgresRepository) RecordAgentRun(ctx context.Context, agentID, projectID, taskID string, input, output map[string]interface{}) error {
	in, _ := json.Marshal(sanitizeMap(input))
	out, _ := json.Marshal(sanitizeMap(output))
	tools, _ := json.Marshal(toolsFromOutput(output))
	_, err := r.db.Exec(ctx, `insert into agent_runs (id,agent_id,project_id,task_id,input,output,tools_used,status,completed_at) values ($1,$2,$3,$4,$5,$6,$7,'completed',now())`,
		uuid.NewString(), agentID, projectID, taskID, in, out, tools)
	return err
}

func (r *PostgresRepository) ListAgentRuns(ctx context.Context, taskID string, limit int) ([]model.AgentRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `select id::text,agent_id::text,project_id::text,task_id::text,input,output,tools_used,status,coalesce(error_message,''),created_at,completed_at from agent_runs`
	args := []interface{}{}
	if taskID != "" {
		query += ` where task_id=$1`
		args = append(args, taskID)
	}
	query += fmt.Sprintf(` order by created_at desc limit %d`, limit)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list agent runs: %w", err)
	}
	defer rows.Close()
	var out []model.AgentRun
	for rows.Next() {
		var run model.AgentRun
		var inputRaw, outputRaw, toolsRaw []byte
		if err := rows.Scan(&run.ID, &run.AgentID, &run.ProjectID, &run.TaskID, &inputRaw, &outputRaw, &toolsRaw, &run.Status, &run.ErrorMessage, &run.CreatedAt, &run.CompletedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(inputRaw, &run.Input)
		_ = json.Unmarshal(outputRaw, &run.Output)
		_ = json.Unmarshal(toolsRaw, &run.ToolsUsed)
		out = append(out, run)
	}
	return out, rows.Err()
}

func sanitizeMap(in map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range in {
		switch value := v.(type) {
		case string:
			out[k] = llm.SanitizeText(value)
		case map[string]interface{}:
			out[k] = sanitizeMap(value)
		default:
			out[k] = value
		}
	}
	return out
}

func toolsFromOutput(output map[string]interface{}) []string {
	details, ok := output["details"].(map[string]interface{})
	if !ok {
		return []string{}
	}
	raw, ok := details["tools_used"].([]string)
	if ok {
		return raw
	}
	anyList, ok := details["tools_used"].([]interface{})
	if !ok {
		return []string{}
	}
	out := []string{}
	for _, item := range anyList {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func (r *PostgresRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `select count(*) from tasks`).Scan(&count)
	return count, err
}

func (r *PostgresRepository) BlockedCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `select count(*) from tasks where status in ('blocked','needs_founder_approval','needs_compliance_review','needs_security_review')`).Scan(&count)
	return count, err
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanTask(row scanner) (model.Task, error) {
	var t model.Task
	err := row.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.OwnerAgent, &t.Priority, &t.Status, &t.DueDate, &t.CreatedBy, &t.CreatedAt)
	return t, err
}

type AssignmentResult struct {
	Task        model.Task      `json:"task"`
	AgentOutput string          `json:"agent_output"`
	Risk        risk.Result     `json:"risk"`
	Approval    *model.Approval `json:"approval,omitempty"`
}

type Service struct {
	tasks       Repository
	projects    project.Repository
	agents      registry.Repository
	approvals   *approval.Service
	audit       audit.Repository
	runtime     runtime.AgentRuntime
	defaultName string
}

func NewService(tasks Repository, projects project.Repository, agents registry.Repository, approvals *approval.Service, auditRepo audit.Repository, rt runtime.AgentRuntime, defaultProject string) *Service {
	return &Service{tasks: tasks, projects: projects, agents: agents, approvals: approvals, audit: auditRepo, runtime: rt, defaultName: defaultProject}
}

func (s *Service) Assign(ctx context.Context, agentName, title, actor string) (*AssignmentResult, error) {
	a, err := s.agents.GetByName(ctx, agentName)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("agent %q not found", agentName)
	}
	p, err := s.projects.GetOrCreateDefault(ctx, s.defaultName)
	if err != nil {
		return nil, err
	}
	r := risk.Detect(title)
	status := "assigned"
	if r.Level == "critical" {
		status = "needs_founder_approval"
	}
	t, err := s.tasks.Create(ctx, model.Task{ProjectID: p.ID, Title: title, OwnerAgent: a.Name, Status: status, CreatedBy: actor})
	if err != nil {
		return nil, err
	}
	_ = s.tasks.AddEvent(ctx, t.ID, "assigned", actor, "Task assigned to "+a.Name, map[string]interface{}{"risk_level": r.Level})
	out, err := s.runtime.Run(ctx, runtime.AgentRunInput{Agent: *a, Task: *t})
	if err != nil {
		return nil, err
	}
	_ = s.tasks.RecordAgentRun(ctx, a.ID, p.ID, t.ID, map[string]interface{}{"task": title}, map[string]interface{}{"summary": out.Summary, "details": out.Details})
	_ = s.audit.Log(ctx, audit.Entry{ProjectID: p.ID, Actor: actor, Action: "task.assigned", Target: t.ID, RiskLevel: r.Level, Metadata: map[string]interface{}{"agent": a.Name, "runtime_provider": out.Details["provider"], "runtime_model": out.Details["model"], "fallback_used": out.Details["fallback_used"], "external_execution": false}})
	var createdApproval *model.Approval
	if risk.NeedsApproval(r.Level) {
		payload := map[string]interface{}{
			"action_type":                 r.ApprovalType,
			"requester":                   actor,
			"project":                     p.Name,
			"environment":                 "phase_0_simulation",
			"risk_level":                  r.Level,
			"summary":                     title,
			"evidence":                    r.Reason,
			"expected_impact":             "Requires Founder review before any external or production-visible action.",
			"rollback_or_mitigation_plan": "No external action has been taken. Keep as draft or reject the approval.",
			"required_approver":           "founder",
			"review_deadline":             time.Now().Add(72 * time.Hour).Format(time.RFC3339),
		}
		createdApproval, err = s.approvals.Create(ctx, model.Approval{ProjectID: p.ID, ApprovalType: r.ApprovalType, ItemType: "task", ItemID: t.ID, RequestedBy: actor, RiskLevel: r.Level, Payload: payload})
		if err != nil {
			return nil, err
		}
	}
	return &AssignmentResult{Task: *t, AgentOutput: out.Summary, Risk: r, Approval: createdApproval}, nil
}

func (s *Service) List(ctx context.Context, limit int) ([]model.Task, error) {
	return s.tasks.List(ctx, limit)
}
func (s *Service) Get(ctx context.Context, id string) (*model.Task, error) {
	return s.tasks.Get(ctx, id)
}
func (s *Service) UpdateStatus(ctx context.Context, id, status, actor string) (*model.Task, error) {
	t, err := s.tasks.UpdateStatus(ctx, id, status)
	if err != nil {
		return nil, err
	}
	_ = s.tasks.AddEvent(ctx, id, "status_changed", actor, "Task status changed to "+status, map[string]interface{}{})
	_ = s.audit.Log(ctx, audit.Entry{ProjectID: t.ProjectID, Actor: actor, Action: "task.status_updated", Target: id, RiskLevel: "medium", Metadata: map[string]interface{}{"status": status}})
	return t, nil
}
