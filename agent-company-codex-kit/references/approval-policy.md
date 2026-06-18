# Approval Policy

The following action types require explicit approval:

1. `publish_content`
2. `merge_code`
3. `deploy_staging`
4. `deploy_production`
5. `change_risk_rule`
6. `contact_kol`
7. `send_telegram_announcement`
8. `enable_live_trading`
9. `access_sensitive_data`
10. `connect_external_tool`
11. `modify_project_roadmap`

## Approval Request Fields

- Action type.
- Requester or agent name.
- Project and environment.
- Risk level.
- Summary of intended action.
- Evidence and expected impact.
- Rollback or mitigation plan.
- Required approver.
- Expiration or review deadline.

No approval request should contain secrets, private keys, or full sensitive payloads.

