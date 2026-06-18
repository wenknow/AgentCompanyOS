package artifact

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agentcompany/agent-company-os/backend/internal/llm"
	"github.com/agentcompany/agent-company-os/backend/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, a model.Artifact) (*model.Artifact, error)
	List(ctx context.Context, taskID string, limit int) ([]model.Artifact, error)
	ListByType(ctx context.Context, artifactType, title string, limit int) ([]model.Artifact, error)
	Get(ctx context.Context, id string) (*model.Artifact, error)
}

type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository { return &PostgresRepository{db: db} }

func (r *PostgresRepository) Create(ctx context.Context, a model.Artifact) (*model.Artifact, error) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.Status == "" {
		a.Status = "draft"
	}
	if a.Metadata == nil {
		a.Metadata = map[string]interface{}{}
	}
	a.Content = llm.SanitizeText(a.Content)
	metadata, _ := json.Marshal(a.Metadata)
	err := r.db.QueryRow(ctx, `insert into artifacts (id,project_id,task_id,agent_id,artifact_type,title,content,status,metadata)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		returning id::text,coalesce(project_id::text,''),coalesce(task_id::text,''),coalesce(agent_id::text,''),artifact_type,title,content,status,metadata,created_at,updated_at`,
		a.ID, emptyUUID(a.ProjectID), emptyUUID(a.TaskID), emptyUUID(a.AgentID), a.ArtifactType, a.Title, a.Content, a.Status, metadata).
		Scan(&a.ID, &a.ProjectID, &a.TaskID, &a.AgentID, &a.ArtifactType, &a.Title, &a.Content, &a.Status, &metadata, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create artifact: %w", err)
	}
	_ = json.Unmarshal(metadata, &a.Metadata)
	return &a, nil
}

func (r *PostgresRepository) List(ctx context.Context, taskID string, limit int) ([]model.Artifact, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `select id::text,coalesce(project_id::text,''),coalesce(task_id::text,''),coalesce(agent_id::text,''),artifact_type,title,content,status,metadata,created_at,updated_at from artifacts`
	args := []interface{}{}
	if taskID != "" {
		query += ` where task_id=$1`
		args = append(args, taskID)
	}
	query += fmt.Sprintf(` order by created_at desc limit %d`, limit)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list artifacts: %w", err)
	}
	defer rows.Close()
	var out []model.Artifact
	for rows.Next() {
		a, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListByType(ctx context.Context, artifactType, title string, limit int) ([]model.Artifact, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `select id::text,coalesce(project_id::text,''),coalesce(task_id::text,''),coalesce(agent_id::text,''),artifact_type,title,content,status,metadata,created_at,updated_at from artifacts where artifact_type=$1`
	args := []interface{}{artifactType}
	if title != "" {
		query += ` and title=$2`
		args = append(args, title)
	}
	query += fmt.Sprintf(` order by created_at desc limit %d`, limit)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list artifacts by type: %w", err)
	}
	defer rows.Close()
	var out []model.Artifact
	for rows.Next() {
		a, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (*model.Artifact, error) {
	a, err := scanArtifact(r.db.QueryRow(ctx, `select id::text,coalesce(project_id::text,''),coalesce(task_id::text,''),coalesce(agent_id::text,''),artifact_type,title,content,status,metadata,created_at,updated_at from artifacts where id=$1`, id))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get artifact: %w", err)
	}
	return &a, nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanArtifact(row scanner) (model.Artifact, error) {
	var a model.Artifact
	var metadata []byte
	err := row.Scan(&a.ID, &a.ProjectID, &a.TaskID, &a.AgentID, &a.ArtifactType, &a.Title, &a.Content, &a.Status, &metadata, &a.CreatedAt, &a.UpdatedAt)
	_ = json.Unmarshal(metadata, &a.Metadata)
	return a, err
}

func emptyUUID(id string) interface{} {
	if id == "" {
		return nil
	}
	return id
}
