# Risk Control Standard

## Risk Levels

- `low`: local-only or read-only changes with limited blast radius.
- `medium`: internal behavior changes, non-production integrations, schema changes with safe rollback, or user-visible changes behind review.
- `high`: deploys, merges, public content, sensitive data access, external integrations, roadmap changes, or risk-policy changes.
- `critical`: production destructive actions, funds, wallets, private keys, trading, live risk rules, irreversible data loss, or broad public announcements.

## High-Risk Keywords

Treat these as risk signals that require extra review:

- 发布
- 公告
- 上线
- 部署
- 生产
- merge
- 合并
- 交易
- 钱包
- 私钥
- 资金
- KOL
- 风控
- risk rule
- live trading
- production
- deploy
- publish
- announcement

## Required Response

For high or critical risk, create an approval request with action, reason, risk level, affected scope, rollback or mitigation plan, and required approver.

