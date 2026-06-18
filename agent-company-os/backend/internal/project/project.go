package project

import (
	"context"
	"fmt"

	"github.com/agentcompany/agent-company-os/backend/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetOrCreateDefault(ctx context.Context, name string) (*model.Project, error)
	Create(ctx context.Context, p model.Project) (*model.Project, error)
	List(ctx context.Context) ([]model.Project, error)
	Count(ctx context.Context) (int, error)
}

type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository { return &PostgresRepository{db: db} }

func (r *PostgresRepository) GetOrCreateDefault(ctx context.Context, name string) (*model.Project, error) {
	rows, err := r.db.Query(ctx, `insert into projects (id,name,description,owner) values ($1,$2,$3,$4)
		on conflict (name) do update set updated_at=now()
		returning id::text,name,coalesce(description,''),status,current_phase,coalesce(owner,'')`,
		uuid.NewString(), name, "Default Phase 0 project", "founder")
	if err != nil {
		return nil, fmt.Errorf("upsert default project: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		p, err := scan(rows)
		return &p, err
	}
	return nil, rows.Err()
}

func (r *PostgresRepository) Create(ctx context.Context, p model.Project) (*model.Project, error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	err := r.db.QueryRow(ctx, `insert into projects (id,name,description,status,current_phase,owner) values ($1,$2,$3,coalesce(nullif($4,''),'active'),coalesce(nullif($5,''),'phase_0'),$6)
		returning id::text,name,coalesce(description,''),status,current_phase,coalesce(owner,'')`,
		p.ID, p.Name, p.Description, p.Status, p.CurrentPhase, p.Owner).Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.CurrentPhase, &p.Owner)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]model.Project, error) {
	rows, err := r.db.Query(ctx, `select id::text,name,coalesce(description,''),status,current_phase,coalesce(owner,'') from projects order by created_at desc`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()
	var out []model.Project
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `select count(*) from projects`).Scan(&count)
	return count, err
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scan(row scanner) (model.Project, error) {
	var p model.Project
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.CurrentPhase, &p.Owner)
	return p, err
}
