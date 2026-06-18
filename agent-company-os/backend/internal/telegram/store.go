package telegram

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agentcompany/agent-company-os/backend/internal/command"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageStore interface {
	Record(ctx context.Context, cmd command.Command) error
}

type PostgresMessageStore struct{ db *pgxpool.Pool }

func NewPostgresMessageStore(db *pgxpool.Pool) *PostgresMessageStore {
	return &PostgresMessageStore{db: db}
}

func (s *PostgresMessageStore) Record(ctx context.Context, cmd command.Command) error {
	intent, _ := json.Marshal(map[string]interface{}{"command": cmd.Name, "args": cmd.Args})
	_, err := s.db.Exec(ctx, `insert into telegram_messages (id,chat_id,user_id,command,raw_text,parsed_intent) values ($1,$2,$3,$4,$5,$6)`,
		uuid.NewString(), cmd.ChatID, cmd.UserID, cmd.Name, cmd.RawText, intent)
	if err != nil {
		return fmt.Errorf("record telegram message: %w", err)
	}
	return nil
}
