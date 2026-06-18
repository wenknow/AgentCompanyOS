package main

import (
	"context"
	"log"

	"github.com/agentcompany/agent-company-os/backend/internal/config"
	"github.com/agentcompany/agent-company-os/backend/internal/logger"
	"go.uber.org/zap"
)

func main() {
	_ = context.Background()
	cfg := config.Load()
	logg, err := logger.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer logg.Sync()
	logg.Info("worker started", zap.String("mode", "phase_0_no_background_jobs"))
}
