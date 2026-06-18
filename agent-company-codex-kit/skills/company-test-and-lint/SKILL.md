---
name: company-test-and-lint
description: Use when a task involves tests, linting, CI, validation, acceptance checks, or release readiness.
---

# Company Test And Lint

Validation rules:

- Run `gofmt` for Go changes.
- Run `go test ./...` unless the project has a better documented target.
- If a `Makefile` exists, prefer `make test` for test execution.
- If tests fail, locate the failure cause before proposing fixes.
- Do not hide or downplay failures.
- Output a concise test result summary.
- State uncovered risks and any tests that could not be run.

Prefer project-native commands and keep validation reproducible.

