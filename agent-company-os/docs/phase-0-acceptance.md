# Phase 0 Acceptance

- The API starts and returns `/health`.
- Built-in agents seed idempotently.
- `/assign` creates task, task event, agent run, and audit log.
- High-risk and critical tasks create approvals and do not execute externally.
- Telegram commands are stored in `telegram_messages`.
- `go test ./...`, `make test`, and `docker compose -f infra/docker-compose.yml config` are reproducible checks.
