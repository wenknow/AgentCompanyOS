# Golang Standard

## Packages

- Use short lowercase package names that describe responsibility.
- Keep `cmd` focused on process startup and dependency wiring.
- Keep domain logic in `internal/service` and persistence in `internal/repository`.
- Keep transport logic in `internal/api` or handler packages.

## Context

- Accept `context.Context` as the first argument for request-scoped work.
- Pass context through database, cache, queue, and external calls.
- Respect cancellation and deadlines.

## Errors

- Wrap errors with useful context using `%w`.
- Return safe API errors to clients and detailed structured errors to logs.
- Do not compare wrapped errors by string.

## Transactions

- Keep transaction boundaries explicit in service layer.
- Avoid long-running transactions and external network calls inside transactions.
- Make retries idempotent where possible.

## Logging

- Use zap structured logging.
- Include request, task, project, and approval identifiers when relevant.
- Do not log secrets, tokens, private keys, or sensitive payloads.

## Testing And Migrations

- Add unit tests for service logic and integration tests for repository behavior when practical.
- Use migrations for every PostgreSQL schema change.
- Validate migrations against realistic existing data when the change is risky.

