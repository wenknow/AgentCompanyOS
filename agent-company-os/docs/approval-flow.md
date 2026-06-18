# Approval Flow

High-risk assignment flow:

1. Create task and task event.
2. Run deterministic RuleBasedRuntime.
3. Store agent run.
4. Detect risk.
5. Create approval when risk is high or critical.
6. Write audit log.

`/approve` and `/reject` only update approval state. They do not execute external actions.
