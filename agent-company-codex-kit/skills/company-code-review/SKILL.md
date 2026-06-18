---
name: company-code-review
description: Use for code review, diff review, PR review, bug fix review, regression review, and change-risk assessment.
---

# Company Code Review

Review changes in a findings-first style. Check:

- Code correctness and edge cases.
- Architecture boundaries and layer ownership.
- Error handling and user-safe responses.
- Test coverage and missing scenarios.
- Security issues, including injection, unsafe auth, secret exposure, and sensitive logging.
- Database migration risks, rollback path, indexes, locks, and data compatibility.
- Configuration and secret management risks.
- Concurrency, idempotency, race, and transaction risks.
- Maintainability, readability, and operational clarity.
- Whether approval is required before merge, deploy, publish, or external action.

Order findings by severity and include file and line references when available.

