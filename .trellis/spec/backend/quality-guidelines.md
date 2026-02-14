# Quality Guidelines

> Code quality standards for backend development.

---

## Overview

Backend quality baseline:
- Compile cleanly (`go build ./...`)
- Tests pass (`go test ./...`)
- Lint passes (`golangci-lint run`)
- No API contract regressions for existing endpoints

---

## Forbidden Patterns

- Business logic inside Fiber handlers.
- Direct DB access from handlers.
- Panics for normal error handling.
- Silent error swallowing (`_ = err` without reason).
- Hardcoded secrets/credentials in code.

---

## Required Patterns

- Context propagation across handler/service/repository.
- Explicit DTOs for request/response boundaries.
- Centralized error mapping and unified error response schema.
- Pagination for list endpoints.
- RBAC checks for admin-only routes.

---

## Testing Requirements

Minimum per new feature:
- Service-level unit tests for core business rules.
- Repository tests for non-trivial queries.
- Handler/integration tests for happy-path + major error path.

For scoring and submission changes:
- Add test covering scoring consistency.
- Add test covering duplicate/invalid submissions.

---

## Code Review Checklist

- Layering respected (handler/service/repository).
- Error codes and statuses are correct and consistent.
- Query performance reasonable (no obvious N+1).
- Sensitive data is not exposed in logs/responses.
- Route naming and DTO naming follow conventions.
