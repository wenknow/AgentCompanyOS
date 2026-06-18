---
name: agent-company-os-dev
description: Use when developing AgentCompanyOS, its Phase 0 MVP, Telegram command flow, agent control plane, approval system, or company agent runtime foundations.
---

# AgentCompanyOS Development

Project rules:

- AgentCompanyOS is a company-level Agent control plane, not a single-project script.
- Founder commands the system through Telegram.
- Version 1 implements Phase 0 MVP only.
- Phase 0 does not connect to real GitHub, Codex, Claude Code, deployment systems, trading systems, wallets, or production tools.
- High-risk actions create approval records only; they are not executed.
- All Telegram commands are written to `telegram_messages`.
- All Agent actions are written to `agent_runs`.
- All critical actions are written to `audit_logs`.
- Use Golang, Gin, PostgreSQL, Redis, Docker Compose, and zap.
- Reserve interfaces for Agent Runtime, Tool Gateway, and LLM Provider.
- Built-in agents are `chief_of_staff`, `product`, `cto`, `backend`, `content`, and `compliance`.
- Implement task system, approval system, Agent Registry, project system, daily reports, and weekly reports.
- Include migrations, `README.md`, docs, `Makefile`, and baseline tests.

Keep Phase 0 safe and simulated. Prefer auditable drafts, approvals, and internal state transitions over external side effects.
