package logger

import (
	"github.com/agentcompany/agent-company-os/backend/internal/config"
	"go.uber.org/zap"
)

func New(cfg config.Config) (*zap.Logger, error) {
	if cfg.AppEnv == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
