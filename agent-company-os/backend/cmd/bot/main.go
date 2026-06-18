package main

import (
	"context"
	"log"

	"github.com/agentcompany/agent-company-os/backend/internal/agents/registry"
	"github.com/agentcompany/agent-company-os/backend/internal/app"
	"github.com/agentcompany/agent-company-os/backend/internal/approval"
	"github.com/agentcompany/agent-company-os/backend/internal/artifact"
	"github.com/agentcompany/agent-company-os/backend/internal/audit"
	"github.com/agentcompany/agent-company-os/backend/internal/config"
	"github.com/agentcompany/agent-company-os/backend/internal/database"
	"github.com/agentcompany/agent-company-os/backend/internal/logger"
	"github.com/agentcompany/agent-company-os/backend/internal/project"
	"github.com/agentcompany/agent-company-os/backend/internal/task"
	"github.com/agentcompany/agent-company-os/backend/internal/telegram"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()
	logg, err := logger.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer logg.Sync()
	if cfg.TelegramBotToken == "" {
		logg.Fatal("TELEGRAM_BOT_TOKEN is required for bot process")
	}
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logg.Fatal("database unavailable", zap.Error(err))
	}
	defer db.Close()
	agents := registry.NewPostgresRepository(db)
	if err := agents.SeedBuiltins(ctx); err != nil {
		logg.Fatal("seed agents failed", zap.Error(err))
	}
	services := app.NewServices(cfg, agents, project.NewPostgresRepository(db), task.NewPostgresRepository(db), approval.NewPostgresRepository(db), audit.NewPostgresRepository(db), artifact.NewPostgresRepository(db))
	bot, err := telegram.NewBot(cfg.TelegramBotToken, services, telegram.NewPostgresMessageStore(db), logg)
	if err != nil {
		logg.Fatal("telegram init failed", zap.Error(err))
	}
	logg.Info("telegram bot polling")
	if err := bot.Run(ctx); err != nil {
		logg.Fatal("telegram bot stopped", zap.Error(err))
	}
}
