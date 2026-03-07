# Learnings (append-only)

## 2026-02-15
- Backend-only compose: bring up postgres/redis, then backend once DB is ready; /api/v1/health returns 200 on localhost:8080.
- Backend required env vars validated at startup: DATABASE_URL, JWT_SECRET (PORT/JWT_TTL/WORKER_* have defaults).
- Added seed entrypoint `backend/cmd/seed/main.go`: idempotently creates admin/player users and two published challenges (modes: static + dynamic).
- Seed password policy: no weak plaintext defaults in repo; `SEED_ADMIN_PASSWORD` / `SEED_PLAYER_PASSWORD` can be injected, otherwise secure random passwords are generated at runtime and printed once.
- Verification path executed via containerized backend: admin login + `/api/v1/auth/admin-sample` returned 200, admin/player both listed seeded published challenges including static and dynamic modes.

## 2026-02-16
- Task 4 (Register + TDD): added backend RED tests for missing register fields (email/password/displayName) and duplicate-email conflict semantics.
- GREEN/REFACTOR: register now normalizes duplicate-user creation DB errors to `409 AUTH_EMAIL_ALREADY_EXISTS`, aligning with existing error contract.
- Verification: `go test ./internal/modules/auth` and `go test ./...` both pass via `golang:1.22-alpine` container with `GOPROXY=https://goproxy.cn,direct`.

- Task 5 (Login + Me + TDD): added handler-level tests (Fiber + ErrorHandler) to verify: login returns `accessToken`; invalid credentials returns 401 with `AUTH_INVALID_CREDENTIALS`; `/api/v1/auth/me` without token returns 401 with `AUTH_MISSING_TOKEN`.
- QA evidence captured via curl under `.sisyphus/evidence/`: login success includes `accessToken`, invalid credentials shows 401, and `/auth/me` without Authorization shows 401.

- Task 6 (Challenge browse list/detail + TDD): added handler-level tests to verify role-based `publishedOnly` behavior (player sees published only; admin sees all) for both list and detail endpoints.
- QA evidence captured via curl under `.sisyphus/evidence/`: player list excludes unpublished, admin list includes unpublished, player 404 on unpublished detail while admin can fetch it; player can fetch published detail.

- Task 7 (Submission → Judge + TDD): expanded submission service tests for dynamic failure paths (`JUDGE_QUEUE_UNAVAILABLE`, `JUDGE_JOB_ENQUEUE_FAILED`) while preserving pending submission persistence semantics.
- Dynamic flow evidence captured via API + DB snapshots: POST `/api/v1/submissions` on dynamic challenge returns `status=pending` + `judgeJobId`; worker processing moves judge job `queued -> done` and updates submission from `pending -> correct` with awarded points.

- Task 8 (Scoreboard ordering + consistency): existing unit test `TestServiceListRanksByPointsThenTieBreakers` already verifies ordering contract in `scoreboard/service.go` (totalPoints desc, then earlier `lastAcceptedAt`, then deterministic `userId` asc).
- API QA evidence confirms runtime behavior with fresh users who solved same 50-point static challenge in sequence; scoreboard ranks earlier solver above later solver under equal points.

- Task 9 (Admin Challenge Management + TDD): added handler-level backend tests covering full admin CRUD + publish toggle happy-path and player-forbidden mutation paths (POST/PUT/DELETE -> 403 `AUTH_FORBIDDEN`).
- Added frontend minimal guard test for `/admin/challenges` rendering 403 hint for authenticated non-admin sessions; updated Vitest alias config for `@/` path resolution.
- API QA evidence for task 9 confirms admin create/update/delete + publish toggle behavior and role-based visibility effect (player list excludes unpublished before toggle and includes after publish).

- Task 10 (Steady-state + failure-path hardening): added handler-level tests for malformed and expired JWT -> 401 `AUTH_INVALID_TOKEN`; confirmed missing-token -> 401 `AUTH_MISSING_TOKEN` remains consistent. Added error middleware logging for AppError status>=500 with requestId/code/status for diagnosability. Captured curl evidence for 401/403 and dependency-unavailable (DB down) -> 500 structured error response + backend log excerpt.
