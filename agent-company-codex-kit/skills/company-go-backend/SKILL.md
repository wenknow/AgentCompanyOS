---
name: company-go-backend
description: Use when working on Golang, Gin, PostgreSQL, Redis, Docker Compose, REST APIs, or backend services under company standards.
---

# Company Go Backend

Follow these rules for backend implementation and review:

- Use clear layers: `cmd`, `internal/config`, `internal/logger`, `internal/database`, `internal/api`, `internal/service`, and `internal/repository`.
- Do not place business logic in `main.go`; keep startup wiring there only.
- Pass `context.Context` through request, service, repository, cache, and external calls.
- Use zap for structured logging and avoid logging secrets or tokens.
- Read all configuration from environment variables or documented config loaders.
- Never hardcode secrets, credentials, private keys, tokens, production URLs, or wallet data.
- Use PostgreSQL migrations for every schema change.
- Treat PostgreSQL as the source of truth.
- Use Redis only for cache, rate limiting, short-lived coordination, or queues; never as the core fact store.
- Implement consistent API error handling with stable status codes and safe response bodies.
- After changes, run `gofmt` and `go test ./...`; use project `Makefile` targets when present.
- Update `README.md` or docs for important behavior, config, API, migration, or workflow changes.

