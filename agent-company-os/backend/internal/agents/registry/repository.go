package registry

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agentcompany/agent-company-os/backend/internal/agents/builtin"
	"github.com/agentcompany/agent-company-os/backend/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	SeedBuiltins(ctx context.Context) error
	List(ctx context.Context) ([]model.Agent, error)
	GetByName(ctx context.Context, name string) (*model.Agent, error)
	Exists(ctx context.Context, name string) (bool, error)
	ActiveCount(ctx context.Context) (int, error)
}

type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository { return &PostgresRepository{db: db} }

func (r *PostgresRepository) SeedBuiltins(ctx context.Context) error {
	for _, a := range builtin.Agents() {
		perms, _ := json.Marshal(a.Permissions)
		_, err := r.db.Exec(ctx, `insert into agents (id,name,role,description,permissions,status) values ($1,$2,$3,$4,$5,$6)
			on conflict (name) do update set role=excluded.role, description=excluded.description, permissions=excluded.permissions, status=excluded.status, updated_at=now()`,
			uuid.NewString(), a.Name, a.Role, a.Description, perms, a.Status)
		if err != nil {
			return fmt.Errorf("seed agent %s: %w", a.Name, err)
		}
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]model.Agent, error) {
	rows, err := r.db.Query(ctx, `select id::text,name,role,coalesce(description,''),permissions,status from agents order by name`)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()
	var out []model.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) GetByName(ctx context.Context, name string) (*model.Agent, error) {
	row := r.db.QueryRow(ctx, `select id::text,name,role,coalesce(description,''),permissions,status from agents where name=$1`, name)
	a, err := scanAgent(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) Exists(ctx context.Context, name string) (bool, error) {
	a, err := r.GetByName(ctx, name)
	return a != nil, err
}

func (r *PostgresRepository) ActiveCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `select count(*) from agents where status='active'`).Scan(&count)
	return count, err
}

type agentScanner interface {
	Scan(dest ...interface{}) error
}

func scanAgent(row agentScanner) (model.Agent, error) {
	var a model.Agent
	var raw []byte
	if err := row.Scan(&a.ID, &a.Name, &a.Role, &a.Description, &raw, &a.Status); err != nil {
		return a, err
	}
	_ = json.Unmarshal(raw, &a.Permissions)
	return a, nil
}
