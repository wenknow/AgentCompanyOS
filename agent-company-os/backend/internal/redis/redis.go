package redis

import (
	"context"

	"github.com/agentcompany/agent-company-os/backend/internal/config"
	goredis "github.com/redis/go-redis/v9"
)

func Connect(ctx context.Context, cfg config.Config) (*goredis.Client, error) {
	client := goredis.NewClient(&goredis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return client, nil
}

func Health(ctx context.Context, client *goredis.Client) string {
	if client == nil {
		return "unavailable"
	}
	if err := client.Ping(ctx).Err(); err != nil {
		return "down"
	}
	return "ok"
}
