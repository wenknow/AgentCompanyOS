# Architecture

Founder commands enter through Telegram or REST. Transports call service-layer use cases. Services coordinate PostgreSQL repositories, risk detection, approvals, audit logs, and the configured agent runtime.

Phase 1 can use DeepSeek for synchronous draft generation when `LLM_PROVIDER=deepseek` and `DEEPSEEK_API_KEY` are configured. Missing or failing LLM configuration uses the rule-based runtime. Critical wallet, private-key, and funds-related tasks bypass the LLM entirely and use rule-based fallback while approval records are created.

`cmd/*` only wires dependencies. PostgreSQL is the source of truth. Redis is initialized and exposed in health checks but is not used for core facts in Phase 1. No external execution tools are connected in this phase.
