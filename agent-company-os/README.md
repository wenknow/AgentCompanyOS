# AgentCompanyOS Phase 1 MVP

AgentCompanyOS is a safe company Agent control plane for a one-person company or small team. Phase 1 exposes Telegram commands and REST APIs for agents, projects, tasks, approvals, audits, and reports. Normal agent drafts can use DeepSeek through its OpenAI-compatible chat completions API. It is still safe by design: no GitHub, Codex, Claude Code, deployment, publishing, trading, wallet, production, or external execution tool is called.

## Quick Start

```sh
cp .env.example .env
docker compose -f infra/docker-compose.yml up --build
curl http://localhost:8080/health
```

PostgreSQL and Redis are expected to run locally on the host. For local `go run`, use `localhost` in `.env`. For the Docker Compose backend container, `infra/docker-compose.yml` connects to the host through `host.docker.internal`.

DeepSeek is enabled by setting `LLM_PROVIDER=deepseek` and `DEEPSEEK_API_KEY` in `.env`. If the API key is missing, unsupported, timed out, rate limited, or returns an invalid response, task assignment automatically falls back to the rule-based runtime. Critical wallet, private-key, and funds-related tasks are never sent to the LLM.

Run locally after PostgreSQL and Redis are available:

```sh
cd backend
go run ./cmd/api
```

Run the bot separately when `TELEGRAM_BOT_TOKEN` is configured:

```sh
make run-bot
```

## Commands

Telegram supports `/start`, `/help`, `/status`, `/agents`, `/projects`, `/assign [agent] [task]`, `/tasks`, `/approvals`, `/approve [approval_id]`, `/reject [approval_id] [reason]`, `/daily`, and `/weekly`.

All inbound Telegram commands are stored in `telegram_messages`. Key state changes are stored in `audit_logs`.

## REST API

The API listens on `HTTP_PORT` and exposes:

- `GET /health`
- `GET /api/v1/runtime/status`
- `GET /api/v1/agents`
- `GET /api/v1/projects`
- `POST /api/v1/projects`
- `GET /api/v1/tasks`
- `POST /api/v1/tasks`
- `GET /api/v1/tasks/:id`
- `PATCH /api/v1/tasks/:id/status`
- `GET /api/v1/approvals`
- `POST /api/v1/approvals/:id/approve`
- `POST /api/v1/approvals/:id/reject`
- `GET /api/v1/reports/daily`
- `GET /api/v1/reports/weekly`

## Safety Model

High-risk and critical tasks create approval records only. Approving or rejecting an approval updates state and audit logs, but never executes external work in Phase 1. LLM output is stored as draft text only and includes provider, model, fallback, error class, and token usage metadata when available.

## Development

```sh
make test
make lint
docker compose -f infra/docker-compose.yml config
```
