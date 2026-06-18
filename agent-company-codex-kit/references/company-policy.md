# Company Policy

## Principles

1. Founder approval first for high-risk and externally visible actions.
2. Audit everything that changes product state, user state, data, policy, or external systems.
3. Safe by default: prefer read-only, draft, simulation, and approval flows.
4. Draft before execute: generate plans, diffs, messages, and approval requests before taking action.
5. No hidden side effects: every tool call and state change must be intentional and explainable.
6. No production action without explicit approval.

## Operating Rules

- Public announcements, production deploys, code merges, trading, wallet activity, and risk rule changes require approval.
- Agents should record intent, inputs, outputs, and decisions in durable audit logs.
- When evidence is incomplete, state uncertainty instead of inventing facts.
- Prefer small reversible changes with clear validation.

