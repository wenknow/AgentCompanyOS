package main

import (
	"context"
	"log"

	"github.com/agentcompany/agent-company-os/backend/internal/agents/registry"
	"github.com/agentcompany/agent-company-os/backend/internal/api"
	"github.com/agentcompany/agent-company-os/backend/internal/app"
	"github.com/agentcompany/agent-company-os/backend/internal/approval"
	"github.com/agentcompany/agent-company-os/backend/internal/artifact"
	"github.com/agentcompany/agent-company-os/backend/internal/audit"
	"github.com/agentcompany/agent-company-os/backend/internal/config"
	"github.com/agentcompany/agent-company-os/backend/internal/database"
	"github.com/agentcompany/agent-company-os/backend/internal/logger"
	"github.com/agentcompany/agent-company-os/backend/internal/project"
	redispkg "github.com/agentcompany/agent-company-os/backend/internal/redis"
	"github.com/agentcompany/agent-company-os/backend/internal/task"
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
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logg.Fatal("database unavailable", zap.Error(err))
	}
	defer db.Close()
	redisClient, err := redispkg.Connect(ctx, cfg)
	if err != nil {
		logg.Warn("redis unavailable", zap.Error(err))
	}
	agents := registry.NewPostgresRepository(db)
	if err := agents.SeedBuiltins(ctx); err != nil {
		logg.Fatal("seed agents failed", zap.Error(err))
	}
	services := app.NewServices(cfg, agents, project.NewPostgresRepository(db), task.NewPostgresRepository(db), approval.NewPostgresRepository(db), audit.NewPostgresRepository(db), artifact.NewPostgresRepository(db))
	router := api.NewRouter(services, db, redisClient)
	logg.Info("api listening", zap.String("port", cfg.HTTPPort))
	if err := router.Run(":" + cfg.HTTPPort); err != nil {
		logg.Fatal("api stopped", zap.Error(err))
	}
}
