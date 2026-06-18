package model

import "time"

type Agent struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
}

type Project struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	CurrentPhase string `json:"current_phase"`
	Owner        string `json:"owner"`
}

type Task struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	OwnerAgent  string     `json:"owner_agent"`
	Priority    string     `json:"priority"`
	Status      string     `json:"status"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Approval struct {
	ID             string                 `json:"id"`
	ProjectID      string                 `json:"project_id"`
	ApprovalType   string                 `json:"approval_type"`
	ItemType       string                 `json:"item_type"`
	ItemID         string                 `json:"item_id"`
	RequestedBy    string                 `json:"requested_by"`
	ApprovalStatus string                 `json:"approval_status"`
	ApprovedBy     string                 `json:"approved_by"`
	Reason         string                 `json:"reason"`
	RiskLevel      string                 `json:"risk_level"`
	Payload        map[string]interface{} `json:"payload"`
	CreatedAt      time.Time              `json:"created_at"`
}

type Status struct {
	ProjectsCount         int `json:"projects_count"`
	TasksCount            int `json:"tasks_count"`
	PendingApprovalsCount int `json:"pending_approvals_count"`
	ActiveAgentsCount     int `json:"active_agents_count"`
	BlockedTasksCount     int `json:"blocked_tasks_count"`
}

type AgentRun struct {
	ID           string                 `json:"id"`
	AgentID      string                 `json:"agent_id"`
	ProjectID    string                 `json:"project_id"`
	TaskID       string                 `json:"task_id"`
	Input        map[string]interface{} `json:"input"`
	Output       map[string]interface{} `json:"output"`
	ToolsUsed    []string               `json:"tools_used"`
	Status       string                 `json:"status"`
	ErrorMessage string                 `json:"error_message"`
	CreatedAt    time.Time              `json:"created_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
}

type Artifact struct {
	ID           string                 `json:"id"`
	ProjectID    string                 `json:"project_id"`
	TaskID       string                 `json:"task_id"`
	AgentID      string                 `json:"agent_id"`
	ArtifactType string                 `json:"artifact_type"`
	Title        string                 `json:"title"`
	Content      string                 `json:"content"`
	Status       string                 `json:"status"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}
