package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Log(ctx context.Context, entry Entry) error
}

type Entry struct {
	ProjectID string
	Actor     string
	Action    string
	Target    string
	RiskLevel string
	Metadata  map[string]interface{}
}

type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository { return &PostgresRepository{db: db} }

func (r *PostgresRepository) Log(ctx context.Context, e Entry) error {
	meta, _ := json.Marshal(e.Metadata)
	var projectID interface{}
	if e.ProjectID != "" {
		projectID = e.ProjectID
	}
	if e.RiskLevel == "" {
		e.RiskLevel = "low"
	}
	_, err := r.db.Exec(ctx, `insert into audit_logs (id, project_id, actor, action, target, risk_level, metadata) values ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.NewString(), projectID, e.Actor, e.Action, e.Target, e.RiskLevel, meta)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}
