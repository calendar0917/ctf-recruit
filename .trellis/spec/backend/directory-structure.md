# Directory Structure

> How backend code is organized in this project.

---

## Overview

This project uses a **separated fullstack layout**:
- `backend/` for Go + Fiber API
- `frontend/` for Next.js app

Backend follows a **modular monolith** style for MVP. Keep domain logic in `internal/`, and keep `cmd/` thin.

---

## Directory Layout

```text
backend/
├── cmd/
│   └── api/
│       └── main.go                 # app bootstrap only
├── internal/
│   ├── config/                     # env/config loading
│   ├── platform/                   # db, redis, logger, middleware setup
│   ├── modules/
│   │   ├── auth/
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── model.go
│   │   │   └── dto.go
│   │   ├── challenge/
│   │   └── submission/
│   ├── middleware/                 # auth/rate-limit/recovery
│   └── router/                     # route registration
├── migrations/                     # SQL migrations
├── tests/                          # integration/e2e tests
├── go.mod
└── go.sum
```

---

## Module Organization

For each domain module:
- `handler.go`: HTTP boundary (Fiber ctx, request/response mapping)
- `service.go`: business rules
- `repository.go`: DB access only
- `model.go`: DB models/entities
- `dto.go`: request/response structs

Rules:
- Handler must not contain business logic.
- Repository must not know HTTP/Fiber types.
- Cross-module calls go through `service` layer, not direct repository coupling.

---

## Naming Conventions

- Folders/files: `snake_case` only when required by tooling; otherwise lowercase short names.
- Go packages: lowercase, no underscores if avoidable.
- Interfaces: define near usage site (usually service layer), avoid global `interfaces` package.
- HTTP routes: plural resources (`/api/v1/challenges`, `/api/v1/submissions`).

---

## Examples

Current repository is in bootstrap stage and does not yet contain production backend modules.
Use this file as the source-of-truth pattern for upcoming implementation.
