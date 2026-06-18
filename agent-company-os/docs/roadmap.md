# Roadmap

Phase 0 delivered the simulated control plane, Telegram, REST, PostgreSQL, Redis health, approvals, audits, and rule-based agents.

Phase 1 adds a real DeepSeek-backed LLM runtime for draft agent output only. It does not add GitHub, Codex, Claude Code, deployment, publishing, trading, wallet, production, or external execution tool integrations. High-risk actions still create approvals only, and LLM failures fall back to the rule-based runtime.

Later phases may add LangGraph orchestration, external coding adapters, admin web UI, richer reporting, and production-grade queues. Each external action must keep Founder approval and audit guarantees.
