---
name: company-architecture-review
description: Use for system design, module decomposition, architecture review, refactoring, service boundaries, and cross-module design decisions.
---

# Company Architecture Review

When reviewing or proposing architecture, produce:

1. Current architecture understanding.
2. Impact scope across modules, data, APIs, jobs, infrastructure, and users.
3. Recommended design and rationale.
4. Module boundaries and ownership.
5. Data flow, request flow, and failure paths.
6. Risk points, observability needs, migration concerns, and rollback strategy.
7. What should not be done and why.
8. Phased implementation plan with validation steps.

Prefer simple, testable, observable designs with explicit interfaces and minimal hidden coupling.

