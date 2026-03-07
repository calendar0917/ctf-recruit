# CTF Recruit MVP - Docker Runbook

## Quick start (first-time setup)

> Runtime defaults intentionally avoid port **3000** for this project.

```bash
# 1) Validate compose wiring
docker compose config

# 2) Build + start stack
docker compose up -d

# 3) Check service status
docker compose ps

# 4) Health check (project backend default port)
curl -i http://localhost:18080/api/v1/health
```

Default published ports:
- Frontend: `http://localhost:13001`
- Backend API: `http://localhost:18080`

Override ports if needed:

```bash
BACKEND_PORT=28080 FRONTEND_PORT=23001 docker compose up -d
```

## Stack lifecycle

```bash
# Start / recreate
docker compose up -d

# Stop
docker compose down

# Stop + remove DB volume
docker compose down -v
```

## Automated migrations

`docker-compose.yml` includes a one-shot `migrate` service that applies all `backend/migrations/*.up.sql` before backend/worker start.

## Seed demo admin/player accounts (reproducible)

Use explicit credentials to make seed deterministic:

```bash
docker compose run --rm \
  -e DATABASE_URL="postgres://postgres:postgres@postgres:5432/ctf_recruit?sslmode=disable" \
  -e SEED_ADMIN_EMAIL="admin@ctf.local" \
  -e SEED_ADMIN_PASSWORD="AdminPass123!" \
  -e SEED_PLAYER_EMAIL="player@ctf.local" \
  -e SEED_PLAYER_PASSWORD="PlayerPass123!" \
  backend /bin/seed
```

Seed behavior details are in `backend/README.seed.md`.

## Test commands

```bash
pnpm lint
pnpm type-check
pnpm test
pnpm test:coverage
```

## Deterministic instance lifecycle API runbook (compose + seed + login + start/stop/cooldown)

### 0) Reset to a clean runtime baseline

Run this block exactly to guarantee deterministic seed + runtime prerequisites:

```bash
docker compose down -v --remove-orphans
docker compose up -d --build
docker compose ps
curl -i http://localhost:18080/api/v1/health
```

Expected health check response:
- HTTP status `200 OK`
- body includes `"status":"ok"`

### 1) Seed deterministic users + baseline challenges

```bash
docker compose run --rm \
  -e DATABASE_URL="postgres://postgres:postgres@postgres:5432/ctf_recruit?sslmode=disable" \
  -e SEED_ADMIN_EMAIL="admin@ctf.local" \
  -e SEED_ADMIN_PASSWORD="AdminPass123!" \
  -e SEED_PLAYER_EMAIL="player@ctf.local" \
  -e SEED_PLAYER_PASSWORD="PlayerPass123!" \
  backend /bin/seed
```

Expected seed output includes `seed completed`.

### 2) Login as seeded player and export token

```bash
PLAYER_LOGIN_JSON="$(curl -fsS -X POST http://localhost:18080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"player@ctf.local","password":"PlayerPass123!"}')"
printf '%s\n' "$PLAYER_LOGIN_JSON"
export PLAYER_TOKEN="$(printf '%s' "$PLAYER_LOGIN_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["accessToken"])')"
```

Expected login response fields:
- `accessToken` (JWT)
- `tokenType` = `Bearer`
- `user.email` = `player@ctf.local`

### 3) Resolve seeded dynamic challenge ID (`Log Trail`)

```bash
PLAYER_CHALLENGES_JSON="$(curl -fsS http://localhost:18080/api/v1/challenges \
  -H "Authorization: Bearer $PLAYER_TOKEN")"
printf '%s\n' "$PLAYER_CHALLENGES_JSON"
export LOG_TRAIL_ID="$(printf '%s' "$PLAYER_CHALLENGES_JSON" | python3 -c 'import json,sys; items=json.load(sys.stdin)["items"]; print(next(i["id"] for i in items if i["title"] == "Log Trail"))')"
```

### 4) Start instance for `Log Trail`

```bash
START_JSON="$(curl -fsS -X POST http://localhost:18080/api/v1/instances/start \
  -H "Authorization: Bearer $PLAYER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"challengeId\":\"$LOG_TRAIL_ID\"}")"
printf '%s\n' "$START_JSON"
export INSTANCE_ID="$(printf '%s' "$START_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"
```

Expected start response:
- HTTP status `201`
- `status` = `running`
- `startedAt` and `expiresAt` populated
- `accessInfo.host` + `accessInfo.port` populated (seeded dynamic runtime config)

### 5) Query active instance (`/instances/me`)

```bash
curl -fsS http://localhost:18080/api/v1/instances/me \
  -H "Authorization: Bearer $PLAYER_TOKEN"
```

Expected:
- `{ "instance": { ... } }` while status is `starting`, `running`, or `stopping`
- `{ "instance": null }` when no active instance
- Additive cooldown metadata may be present with no active instance:
  `{ "instance": null, "cooldown": { "retryAt": "2026-02-17T00:02:00Z" } }`

### 6) Stop running instance

```bash
STOP_JSON="$(curl -fsS -X POST http://localhost:18080/api/v1/instances/stop \
  -H "Authorization: Bearer $PLAYER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"instanceId\":\"$INSTANCE_ID\"}")"
printf '%s\n' "$STOP_JSON"
```

Expected stop response:
- HTTP status `200`
- `status` = `stopped`
- `cooldownUntil` populated

### 7) Verify cooldown rejection (immediate restart)

```bash
COOLDOWN_JSON="$(curl -sS -X POST http://localhost:18080/api/v1/instances/start \
  -H "Authorization: Bearer $PLAYER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"challengeId\":\"$LOG_TRAIL_ID\"}")"
printf '%s\n' "$COOLDOWN_JSON"
```

Expected cooldown response:
- HTTP status `409`
- `error.code` = `INSTANCE_COOLDOWN_ACTIVE`
- `error.details.retryAt` is present (RFC3339)

## Dynamic runtime-config validation path (admin create -> player run -> verify)

Use this when you need a dedicated runtime-config challenge beyond seeded fixtures.

### A) Login as seeded admin

```bash
ADMIN_LOGIN_JSON="$(curl -fsS -X POST http://localhost:18080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@ctf.local","password":"AdminPass123!"}')"
printf '%s\n' "$ADMIN_LOGIN_JSON"
export ADMIN_TOKEN="$(printf '%s' "$ADMIN_LOGIN_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["accessToken"])')"
```

### B) Create published dynamic challenge with runtime fields

```bash
RUNTIME_CHALLENGE_JSON="$(curl -fsS -X POST http://localhost:18080/api/v1/challenges \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Runtime Probe","description":"Deterministic runtime validation challenge","category":"ops","difficulty":"easy","mode":"dynamic","runtimeImage":"busybox:1.36","runtimeCommand":"httpd -f -p 8080","runtimeExposedPort":8080,"points":150,"flag":"CTF{runtime-probe-flag}","isPublished":true}')"
printf '%s\n' "$RUNTIME_CHALLENGE_JSON"
export RUNTIME_CHALLENGE_ID="$(printf '%s' "$RUNTIME_CHALLENGE_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"
```

Expected create response:
- HTTP status `201`
- `mode` = `dynamic`
- `runtimeImage` = `busybox:1.36`
- `runtimeCommand` = `httpd -f -p 8080`
- `runtimeExposedPort` = `8080`

### C) Start that challenge as player

```bash
RUNTIME_START_JSON="$(curl -fsS -X POST http://localhost:18080/api/v1/instances/start \
  -H "Authorization: Bearer $PLAYER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"challengeId\":\"$RUNTIME_CHALLENGE_ID\"}")"
printf '%s\n' "$RUNTIME_START_JSON"
export RUNTIME_INSTANCE_ID="$(printf '%s' "$RUNTIME_START_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"
export RUNTIME_CONTAINER_ID="$(printf '%s' "$RUNTIME_START_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["containerId"])')"
```

Expected start response:
- HTTP status `201`
- `status` = `running`
- `containerId` is present
- `accessInfo.connectionString` may be present depending on runtime behavior

### D) Verify runtime container + port mapping deterministically

```bash
docker inspect "$RUNTIME_CONTAINER_ID" --format 'running={{.State.Running}} image={{.Config.Image}}'
docker port "$RUNTIME_CONTAINER_ID" 8080/tcp
```

Expected:
- `running=true image=busybox:1.36`
- host port mapping output similar to `8080/tcp -> 127.0.0.1:32xxx`

If container exits quickly, treat it as runtime failure and use troubleshooting section (`INSTANCE_RUNTIME_START_FAILED`) to inspect backend/worker docker access and logs.

### E) Cleanup (stop runtime + delete challenge)

```bash
curl -fsS -X POST http://localhost:18080/api/v1/instances/stop \
  -H "Authorization: Bearer $PLAYER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"instanceId\":\"$RUNTIME_INSTANCE_ID\"}"

curl -i -X DELETE "http://localhost:18080/api/v1/challenges/$RUNTIME_CHALLENGE_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Expected delete response: HTTP status `204`.

### Start/Stop/Cooldown expected behavior summary

- One active instance per user (`starting/running` conflict on second start).
- `stop` is owner-only and requires current status `running`.
- `stop`/`expire` enters cooldown window (default 1 minute).
- During cooldown, `start` must return `409` with `retryAt`.

### Additive lifecycle contract details

- `GET /api/v1/instances/me` keeps existing shape and can now include optional cooldown metadata when no active instance exists:
  - baseline: `{ "instance": null }`
  - additive: `{ "instance": null, "cooldown": { "retryAt": "<RFC3339>" } }`
- `POST /api/v1/instances/start` during cooldown returns:
  - HTTP `409`
  - `error.code = INSTANCE_COOLDOWN_ACTIVE`
  - `error.details.retryAt` in RFC3339 format
- `POST /api/v1/instances/start` active-instance conflict keeps status/code/message unchanged for backward compatibility:
  - HTTP `409`
  - `error.code = INSTANCE_ACTIVE_EXISTS`
  - optional additive details may include `activeInstanceId`, `activeChallengeId`, `activeStatus`, `activeStartedAt`, `activeExpiresAt`

## UI verification flow (reproducible, no browser automation required)

Use this flow after compose + seed from the sections above.

1. Open `http://localhost:13001/login` and sign in as player (`player@ctf.local` / `PlayerPass123!`).
   - Expected: login succeeds and app routes to `/challenges`.
2. Open `http://localhost:13001/challenges`.
   - Expected: published challenge cards are visible, including `Log Trail`.
3. Open `http://localhost:13001/challenges/[id]` using the `Log Trail` id.
   - Expected before start: start action enabled.
4. Start instance from challenge detail.
   - Expected: action state transitions to running and instance panel shows runtime access info.
5. Open a different challenge detail route while the instance is still active.
   - Expected: context guard shows mismatch guidance, start/stop actions are disabled, and guidance points to the active challenge route.
6. Trigger conflict and cooldown checks:
   - If start is attempted while an active instance already exists, expect diagnostics that include `INSTANCE_ACTIVE_EXISTS` and may include additive details.
   - After stop, immediate restart should show `INSTANCE_COOLDOWN_ACTIVE` diagnostics with `retryAt`.
7. Reload `/challenges/[id]` during cooldown.
   - Expected: no active instance card, cooldown countdown still visible from `/instances/me` additive `cooldown.retryAt`.
8. Log out, then open `http://localhost:13001/admin/challenges` and `http://localhost:13001/admin/users` while unauthenticated.
   - Expected: redirect to `/login`.
9. Log in as non-admin player and open both admin routes again.
   - Expected: redirect to `/challenges`.
10. Log in as admin (`admin@ctf.local` / `AdminPass123!`) and open both admin routes.
   - Expected: admin pages render normally.
11. In `/admin/challenges`, create or edit a dynamic challenge and set runtime fields (`runtimeImage`, `runtimeCommand`, `runtimeExposedPort`).
    - Expected: player can open that challenge and start instance successfully using saved runtime config.

Deterministic references for this behavior:
- `.sisyphus/evidence/task-16-orchestration-tests.txt`
- `.sisyphus/evidence/task-16-error-coverage.txt`
- `.sisyphus/evidence/task-17-admin-tests.txt`
- `.sisyphus/evidence/task-17-nonadmin.txt`

### Strict 1h TTL semantics

- TTL is **absolute**: `expiresAt = startedAt + 1h`.
- Baseline is the time instance enters `running` (not idle time).
- Worker sweeper checks for `expiresAt <= now` and performs stop+state transition.
- Expiry also applies cooldown (same one-minute policy).

TTL verification options:
- Production semantics: wait until absolute expiry.
- Controlled short-path verification: force `expires_at` in DB for test/evidence and verify sweeper path/logs.

## Troubleshooting / environment caveats

1. **Port conflicts (common in this environment)**
   - This project does **not** bind host `3000`.
   - Backend default host port is `18080`; frontend default host port is `13001`.
   - If conflicts remain, set `BACKEND_PORT` and `FRONTEND_PORT` when running compose.

2. **Health check command mismatch**
   - If you run `curl http://localhost:8080/api/v1/health` and still get `200`, that may be another pre-existing local service.
   - For this compose stack, use `curl http://localhost:18080/api/v1/health` by default.

3. **Root-owned `frontend/.next`**
   - Host-side `pnpm -C frontend build` may fail with EACCES due to root-owned `.next`.
   - Use Docker build/run path (`docker compose up -d`) as the reliable workaround in this environment.

4. **Missing TypeScript LSP**
   - `typescript-language-server` may be unavailable; use `pnpm -C frontend type-check` for static validation.

5. **Missing bun runtime**
   - This repo workflow uses `pnpm` + Docker; do not require bun for startup/test flow.

6. **Container start failure (`INSTANCE_RUNTIME_START_FAILED`)**
   - Symptoms: `POST /instances/start` returns `500`, often with docker daemon/runtime errors in backend logs.
   - Checks:
     1. `docker compose ps` (backend + worker should be running)
     2. `docker compose logs backend --tail 200`
     3. `docker compose exec backend ls -l /var/run/docker.sock`
     4. `docker compose exec backend docker version`
     5. `docker compose exec worker docker version`
   - Recovery (socket/controller unavailable):
     - Ensure host Docker daemon is running (`docker info` on host).
     - Recreate runtime services with fresh mounts:
       `docker compose up -d --build backend worker`
     - Re-run seed command, then retry `POST /instances/start`.
    - Recovery (invalid challenge runtime config):
      - Validate `runtimeImage`, `runtimeCommand`, and `runtimeExposedPort` on the challenge.
      - Fix via admin `PUT /api/v1/challenges/:id` and retry start.

7. **Auth/session redirect confusion (`/login` vs protected routes)**
   - Symptoms:
     - Navigating directly to `/challenges/[id]` or admin pages bounces to `/login`.
     - Session appears lost after reload.
   - Checks:
     1. Re-login via `http://localhost:13001/login`.
     2. Verify API login still works:
        `curl -fsS -X POST http://localhost:18080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"player@ctf.local","password":"PlayerPass123!"}'`
     3. Confirm frontend/backend ports match this runbook (`13001` and `18080` unless overridden).
   - Recovery:
     - Sign out, sign in again, retry target route.
     - If API login fails, rerun deterministic seed command and retry.

8. **Admin role guard redirect behavior**
   - Symptoms:
     - Non-admin user opening `/admin/challenges` or `/admin/users` is redirected to `/challenges`.
   - Expected behavior:
     - Unauthenticated user -> `/login`
     - Authenticated non-admin user -> `/challenges`
     - Authenticated admin user -> admin page loads
   - Verification commands:
     - `pnpm -C frontend test "src/__tests__/admin-challenges-access.test.ts"`
     - `pnpm -C frontend test "src/__tests__/admin-users-access.test.ts"`

9. **Cooldown missing after page reload**
   - Symptoms:
     - User stops an instance, reloads `/challenges/[id]`, then assumes cooldown disappeared because no active instance is shown.
   - Expected behavior:
     - `/instances/me` can return `instance: null` and additive `cooldown.retryAt`.
     - UI should still render cooldown state until `retryAt`.
   - Checks:
     1. `curl -fsS http://localhost:18080/api/v1/instances/me -H "Authorization: Bearer $PLAYER_TOKEN"`
     2. Confirm response includes `"instance":null` and `"cooldown":{"retryAt":"..."}` during active cooldown.
   - Recovery:
     - Wait until `retryAt` before retrying start.
     - If cooldown metadata is absent during known active cooldown, collect API response plus backend logs and run `pnpm -C frontend test "src/__tests__/challenge-detail-page-orchestration.test.tsx"`.

10. **TTL not expiring (instance remains running after forced/expected expiry)**
   - Symptoms: `expiresAt` is in the past but `/instances/me` still returns running.
   - Checks:
     1. `docker compose logs worker --since 10m`
     2. Confirm sweeper errors (`instance sweeper stop failed ... docker daemon ...`)
     3. Ensure worker has Docker daemon access in runtime environment.
   - Recovery:
     - Fix worker runtime Docker access (socket mount/permission).
     - Restart worker: `docker compose up -d worker`
     - Re-validate with expiry scenario.

11. **Cooldown anomalies**
   - Symptoms:
     - Immediate restart succeeds unexpectedly after stop/expire, or
     - Restart blocked too long.
   - Checks:
     1. Inspect stop/expire response `cooldownUntil` and compare to current UTC time.
     2. Confirm `409 INSTANCE_COOLDOWN_ACTIVE` includes `details.retryAt`.
     3. Inspect backend logs around transition and cooldown fields.
   - Recovery:
     - Wait until `retryAt` before retrying start.
     - If mismatch persists, collect API body + logs and run focused tests:
       `go test -v ./internal/modules/instance -run TestInstancesStartBlockedByCooldownReturnsRetryAt`
