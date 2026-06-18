# AGENTS.md

## Project Identity

This repository follows AgentCompanyOS company engineering standards.

## Non-negotiable Rules

- Do not hardcode secrets.
- Do not bypass approval.
- Do not deploy production.
- Do not merge code automatically.
- Do not publish public content automatically.
- Do not operate funds, wallets, trading, or risk rules.
- All high-risk actions must create an approval request.
- All significant code changes must include tests or explain why tests are not possible.

## Development Workflow

1. Understand the task.
2. Read relevant docs.
3. Propose a short implementation plan.
4. Make minimal focused changes.
5. Run formatters and tests.
6. Summarize changes, tests, and risks.

## Go Backend Standards

- Use context.Context.
- Keep handler/service/repository boundaries clear.
- Use migrations for schema changes.
- Use zap for structured logging.
- Read configuration from environment variables.
- Never commit secrets.

## Risk Policy

High-risk operations include deploy, publish, merge, trading, wallet, private key, production, KOL contact, and risk rule changes.

For high-risk operations, create an approval request instead of executing the action.

