package approval

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agentcompany/agent-company-os/backend/internal/audit"
	"github.com/agentcompany/agent-company-os/backend/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, a model.Approval) (*model.Approval, error)
	List(ctx context.Context, status string) ([]model.Approval, error)
	CountPending(ctx context.Context) (int, error)
	UpdateStatus(ctx context.Context, id, status, actor, reason string) (*model.Approval, error)
}

type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository { return &PostgresRepository{db: db} }

func (r *PostgresRepository) Create(ctx context.Context, a model.Approval) (*model.Approval, error) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	payload, _ := json.Marshal(a.Payload)
	err := r.db.QueryRow(ctx, `insert into approvals (id,project_id,approval_type,item_type,item_id,requested_by,risk_level,payload)
		values ($1,$2,$3,$4,$5,$6,$7,$8)
		returning id::text,project_id::text,approval_type,coalesce(item_type,''),coalesce(item_id::text,''),coalesce(requested_by,''),approval_status,coalesce(approved_by,''),coalesce(reason,''),risk_level,payload,created_at`,
		a.ID, a.ProjectID, a.ApprovalType, a.ItemType, emptyUUID(a.ItemID), a.RequestedBy, a.RiskLevel, payload).
		Scan(&a.ID, &a.ProjectID, &a.ApprovalType, &a.ItemType, &a.ItemID, &a.RequestedBy, &a.ApprovalStatus, &a.ApprovedBy, &a.Reason, &a.RiskLevel, &payload, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create approval: %w", err)
	}
	_ = json.Unmarshal(payload, &a.Payload)
	return &a, nil
}

func (r *PostgresRepository) List(ctx context.Context, status string) ([]model.Approval, error) {
	query := `select id::text,project_id::text,approval_type,coalesce(item_type,''),coalesce(item_id::text,''),coalesce(requested_by,''),approval_status,coalesce(approved_by,''),coalesce(reason,''),risk_level,payload,created_at from approvals`
	args := []interface{}{}
	if status != "" {
		query += ` where approval_status=$1`
		args = append(args, status)
	}
	query += ` order by created_at desc limit 50`
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list approvals: %w", err)
	}
	defer rows.Close()
	var out []model.Approval
	for rows.Next() {
		var a model.Approval
		var payload []byte
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.ApprovalType, &a.ItemType, &a.ItemID, &a.RequestedBy, &a.ApprovalStatus, &a.ApprovedBy, &a.Reason, &a.RiskLevel, &payload, &a.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(payload, &a.Payload)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) CountPending(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `select count(*) from approvals where approval_status='pending'`).Scan(&count)
	return count, err
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id, status, actor, reason string) (*model.Approval, error) {
	var a model.Approval
	var payload []byte
	err := r.db.QueryRow(ctx, `update approvals set approval_status=$2, approved_by=$3, reason=$4, updated_at=now() where id=$1
		returning id::text,project_id::text,approval_type,coalesce(item_type,''),coalesce(item_id::text,''),coalesce(requested_by,''),approval_status,coalesce(approved_by,''),coalesce(reason,''),risk_level,payload,created_at`,
		id, status, actor, reason).
		Scan(&a.ID, &a.ProjectID, &a.ApprovalType, &a.ItemType, &a.ItemID, &a.RequestedBy, &a.ApprovalStatus, &a.ApprovedBy, &a.Reason, &a.RiskLevel, &payload, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("update approval: %w", err)
	}
	_ = json.Unmarshal(payload, &a.Payload)
	return &a, nil
}

type Service struct {
	repo  Repository
	audit audit.Repository
}

func NewService(repo Repository, auditRepo audit.Repository) *Service {
	return &Service{repo: repo, audit: auditRepo}
}

func (s *Service) Create(ctx context.Context, a model.Approval) (*model.Approval, error) {
	created, err := s.repo.Create(ctx, a)
	if err != nil {
		return nil, err
	}
	_ = s.audit.Log(ctx, audit.Entry{ProjectID: created.ProjectID, Actor: a.RequestedBy, Action: "approval.created", Target: created.ID, RiskLevel: created.RiskLevel, Metadata: map[string]interface{}{"approval_type": created.ApprovalType}})
	return created, nil
}

func (s *Service) List(ctx context.Context, status string) ([]model.Approval, error) {
	return s.repo.List(ctx, status)
}

func (s *Service) Approve(ctx context.Context, id, actor string) (*model.Approval, error) {
	a, err := s.repo.UpdateStatus(ctx, id, "approved", actor, "")
	if err != nil {
		return nil, err
	}
	_ = s.audit.Log(ctx, audit.Entry{ProjectID: a.ProjectID, Actor: actor, Action: "approval.approved", Target: id, RiskLevel: a.RiskLevel, Metadata: map[string]interface{}{"external_execution": false}})
	return a, nil
}

func (s *Service) Reject(ctx context.Context, id, actor, reason string) (*model.Approval, error) {
	a, err := s.repo.UpdateStatus(ctx, id, "rejected", actor, reason)
	if err != nil {
		return nil, err
	}
	_ = s.audit.Log(ctx, audit.Entry{ProjectID: a.ProjectID, Actor: actor, Action: "approval.rejected", Target: id, RiskLevel: a.RiskLevel, Metadata: map[string]interface{}{"reason": reason, "external_execution": false}})
	return a, nil
}

func emptyUUID(id string) interface{} {
	if id == "" {
		return nil
	}
	return id
}
