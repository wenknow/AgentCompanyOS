# Database Schema

Migration `000001_init` creates `agents`, `projects`, `tasks`, `task_events`, `approvals`, `agent_runs`, `audit_logs`, `telegram_messages`, `company_memory`, `tool_connections`, and `artifacts`.

All primary keys are UUIDs. JSONB defaults are valid empty objects or arrays. Time fields use `timestamptz`.
