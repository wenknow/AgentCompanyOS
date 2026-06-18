# MCP Policy

## Rules

1. MCP access is read-only by default.
2. Write tools require approval.
3. Production tools are disabled by default.
4. Wallet, funds, and trading tools are not connected by default.
5. Every MCP server must declare its permissions and intended scope.
6. MCP output is not automatically treated as fact; critical actions require second verification.

## Safe Defaults

- Prefer local docs, read-only filesystem, and internal reference servers.
- Do not configure production deploy, wallet, trading, or fund-transfer MCP servers in the base kit.
- Treat unknown MCP servers as untrusted until reviewed.

