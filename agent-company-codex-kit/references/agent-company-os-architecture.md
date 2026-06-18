# AgentCompanyOS Architecture

## Phase 0 Summary

AgentCompanyOS Phase 0 is a safe, auditable company Agent control plane.

```text
Founder
  -> Telegram Bot
  -> Command Parser
  -> Agent Router
  -> Task Manager / Approval System / Agent Registry / Audit Log
  -> PostgreSQL / Redis
```

## Boundaries

- Telegram Bot receives founder commands and writes `telegram_messages`.
- Command Parser normalizes commands and extracts intent.
- Agent Router selects built-in agents and creates `agent_runs`.
- Task Manager tracks work items and status transitions.
- Approval System records high-risk requests and decisions.
- Agent Registry tracks available agents and capabilities.
- Audit Log records critical actions and policy decisions.
- PostgreSQL is the durable source of truth.
- Redis supports cache, queues, locks, and short-lived state.

Phase 0 simulates external integrations and does not connect to real production, trading, wallet, GitHub, Codex, Claude Code, or deployment systems.
