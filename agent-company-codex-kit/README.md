# AgentCompany Codex Kit

AgentCompany Codex Kit is a reusable company-level Codex plugin package for engineering standards, risk controls, project workflows, MCP configuration, and AgentCompanyOS development.

It is not application business code. It packages the operating rules that Codex should follow when working in company repositories.

## Why This Plugin Exists

Company agents need consistent rules for code quality, architecture, security, approvals, and high-risk actions. This kit makes those rules installable and repeatable across projects instead of copying prompts by hand.

The kit is safe by default:

- It prefers draft, read-only, and approval flows.
- It does not configure production deploy, wallet, funds, trading, or live risk-rule MCP servers.
- It does not contain real secrets.
- High-risk actions create approval requests instead of being executed automatically.

## Included Skills

- `company-go-backend`: Go, Gin, PostgreSQL, Redis, Docker Compose, REST API, and backend service standards.
- `company-architecture-review`: architecture review, module boundaries, data flow, risks, and phased plans.
- `company-risk-review`: release, deployment, merge, production, funds, wallet, trading, KOL, public announcement, and sensitive-data review.
- `company-prd-to-tasks`: converts ideas, PRDs, and requirements into implementation tasks.
- `company-code-review`: code, diff, PR, bug fix, and regression review.
- `company-test-and-lint`: test, lint, CI, and validation workflow.
- `agent-company-os-dev`: AgentCompanyOS Phase 0 MVP development rules.

## Included References

- `references/company-policy.md`
- `references/engineering-standard.md`
- `references/golang-standard.md`
- `references/security-standard.md`
- `references/risk-control-standard.md`
- `references/approval-policy.md`
- `references/project-workflow.md`
- `references/mcp-policy.md`
- `references/agent-company-os-architecture.md`

## Local Installation

From this repository root, validate the package:

```bash
python3 agent-company-codex-kit/scripts/validate_policy.py
```

Copy the plugin into your local Codex plugins directory:

```bash
mkdir -p ~/.codex/plugins
cp -R agent-company-codex-kit ~/.codex/plugins/
```

If you prefer a symlink while developing the kit:

```bash
mkdir -p ~/.codex/plugins
ln -s "$(pwd)/agent-company-codex-kit" ~/.codex/plugins/agent-company-codex-kit
```

## Marketplace Configuration

This kit includes a local marketplace example at `marketplace.json`:

```json
{
  "name": "agent-company-local-marketplace",
  "interface": {
    "displayName": "AgentCompany Local Plugins"
  },
  "plugins": [
    {
      "name": "agent-company-codex-kit",
      "source": {
        "source": "local",
        "path": "./agent-company-codex-kit"
      },
      "policy": {
        "installation": "AVAILABLE",
        "authentication": "ON_INSTALL"
      },
      "category": "Productivity"
    }
  ]
}
```

Use this file as a template for a team marketplace. Keep `policy.installation`, `policy.authentication`, and `category` explicit.

## Generate AGENTS.md In A Project

Generate the default company `AGENTS.md`:

```bash
python3 ~/.codex/plugins/agent-company-codex-kit/scripts/init_project.py /path/to/project
```

Do not overwrite an existing `AGENTS.md` unless intentional:

```bash
python3 ~/.codex/plugins/agent-company-codex-kit/scripts/init_project.py /path/to/project --force
```

Generate to stdout instead:

```bash
python3 ~/.codex/plugins/agent-company-codex-kit/scripts/generate_agents_md.py > AGENTS.md
```

## Enable MCP

The included `.mcp.json` configures only safe default servers:

- `filesystem-readonly`
- `docs`

It intentionally does not configure real funds, wallets, trading, production deploy, or live risk-rule tools.

Review `references/mcp-policy.md` before adding new MCP servers. Write-capable or production-capable MCP tools require approval.

## Hooks

`hooks/session_start.py` prints a short policy summary at session start.

`hooks/pre_tool_use.py` reads hook payloads from stdin and warns on high-risk keywords such as `rm -rf`, `drop database`, `terraform destroy`, `git push --force`, `npm publish`, `wallet`, `transfer`, and `withdraw`.

The hook warns without blocking normal development. High-risk matches should be converted into an approval request or explicit human confirmation.

## Risk Rules

Risk levels:

- `low`: local-only or read-only changes with limited blast radius.
- `medium`: internal behavior, non-production integrations, or reversible schema changes.
- `high`: deploys, merges, public content, sensitive data access, external integrations, roadmap changes, or risk-policy changes.
- `critical`: production destructive actions, funds, wallets, private keys, trading, live risk rules, irreversible data loss, or broad public announcements.

High-risk actions include deploy, publish, merge, production access, trading, wallet activity, private keys, funds, KOL contact, public announcements, risk rules, and sensitive user data.

## Approval Examples

Approval-required action types include:

- `publish_content`
- `merge_code`
- `deploy_staging`
- `deploy_production`
- `change_risk_rule`
- `contact_kol`
- `send_telegram_announcement`
- `enable_live_trading`
- `access_sensitive_data`
- `connect_external_tool`
- `modify_project_roadmap`

Agents should create approval requests with action type, summary, scope, risk level, approver, reason, and rollback or mitigation plan.

## AgentCompanyOS Development Example

Use the kit when building Phase 0 MVP:

```text
Use AgentCompany Codex Kit to develop AgentCompanyOS Phase 0 MVP.
Implement Telegram command ingestion, task creation, approval records, agent registry, audit logs, daily reports, and weekly reports.
Do not connect real GitHub, Codex, Claude Code, deployment, trading, wallet, or production systems.
```

Expected Phase 0 stack:

- Golang
- Gin
- PostgreSQL
- Redis
- Docker Compose
- zap
- migrations
- Makefile
- baseline tests

Core records:

- `telegram_messages`
- `agent_runs`
- `audit_logs`
- tasks
- approvals
- projects
- agents

## Roadmap

- Add stricter plugin schema validation.
- Add more generated project templates.
- Add approval request templates.
- Add CI examples for Go services.
- Add policy tests for risky prompt and command detection.
- Add optional team marketplace publishing workflow.

