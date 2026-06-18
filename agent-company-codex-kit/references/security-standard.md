# Security Standard

## Rules

1. Do not hardcode secrets.
2. Do not print tokens, API keys, session cookies, private keys, or credentials.
3. Do not save wallet private keys.
4. Use least privilege for users, services, database roles, and tool access.
5. Validate inputs at API and command boundaries.
6. Prevent SQL injection with parameterized queries or safe ORM/query-builder APIs.
7. Require approval for sensitive operations.

## Handling Sensitive Data

- Store secrets only in approved secret managers or runtime environment configuration.
- Mask sensitive fields in logs and audit output.
- Treat user data exports, admin access, production access, and external integrations as approval-requiring operations.

