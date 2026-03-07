# Seed data (admin/player + sample challenges)

## Purpose
Creates a minimal baseline for demos:
- 1 admin account (can access admin challenge endpoints)
- 1 player account (can view published challenges)
- 2 sample challenges (static + dynamic), published by default
- Dynamic challenge includes runtime config for deterministic instance lifecycle validation

## Prerequisites
- Database migrated (see existing migration flow for `backend/migrations/*.sql`).
- Required env vars: `DATABASE_URL` (JWT is not needed for the seed binary).

Recommended deterministic baseline:

```bash
docker compose down -v --remove-orphans
docker compose up -d --build
curl -i http://localhost:18080/api/v1/health
```

## Run the seed

### Option A: local Go toolchain

```bash
go run ./cmd/seed
```

### Option B: Docker image (no local Go needed)

```bash
docker compose build backend
docker compose run --rm \
  -e DATABASE_URL="postgres://postgres:postgres@postgres:5432/ctf_recruit?sslmode=disable" \
  -e SEED_ADMIN_EMAIL="admin@ctf.local" \
  -e SEED_ADMIN_PASSWORD="AdminPass123!" \
  -e SEED_PLAYER_EMAIL="player@ctf.local" \
  -e SEED_PLAYER_PASSWORD="PlayerPass123!" \
  backend /bin/seed
```

### Optional seed env vars

```bash
SEED_ADMIN_EMAIL=admin@ctf.local
SEED_ADMIN_PASSWORD=use-a-strong-password
SEED_ADMIN_DISPLAY_NAME="CTF Admin"

SEED_PLAYER_EMAIL=player@ctf.local
SEED_PLAYER_PASSWORD=use-a-strong-password
SEED_PLAYER_DISPLAY_NAME="CTF Player"
```

If `SEED_ADMIN_PASSWORD` or `SEED_PLAYER_PASSWORD` is omitted, a strong random password is generated and printed to stdout once. Store it securely and rotate after first login.

## What gets created
- Users: admin + player (idempotent by email; existing users are not modified)
- Challenges: two published challenges (idempotent by title+category)

Seeded challenge runtime characteristics:

1. `Welcome Static`
   - `mode`: `static`
   - no runtime image/port required

2. `Log Trail`
   - `mode`: `dynamic`
   - `runtimeImage`: `busybox:1.36`
   - `runtimeCommand`: `httpd -f -p 8080`
   - `runtimeExposedPort`: `8080`
   - expected to return `accessInfo` when `/api/v1/instances/start` succeeds

## Verification (manual, API + UI-aligned)
1. Start backend with `JWT_SECRET` and `DATABASE_URL`.
2. Login as admin via `/api/v1/auth/login` using seeded credentials.
3. Access `/api/v1/challenges` with admin token (returns all challenges, including unpublished if any).
4. Login as player via `/api/v1/auth/login` using seeded credentials.
5. Access `/api/v1/challenges` with player token (returns published challenges including `Log Trail`).
6. Verify `Log Trail` includes runtime fields (`runtimeImage`, `runtimeCommand`, `runtimeExposedPort`).
7. Start an instance for `Log Trail`.
8. Call `/api/v1/instances/me` while running.
9. Try a second start while active and confirm `INSTANCE_ACTIVE_EXISTS`.
10. Stop the instance.
11. Retry start immediately and confirm cooldown conflict (`INSTANCE_COOLDOWN_ACTIVE` + `error.details.retryAt`).
12. Call `/api/v1/instances/me` during cooldown and confirm additive cooldown metadata when no active instance exists.

Deterministic verification commands:

```bash
PLAYER_LOGIN_JSON="$(curl -fsS -X POST http://localhost:18080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"player@ctf.local","password":"PlayerPass123!"}')"
printf '%s\n' "$PLAYER_LOGIN_JSON"
PLAYER_TOKEN="$(printf '%s' "$PLAYER_LOGIN_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["accessToken"])')"

CHALLENGES_JSON="$(curl -fsS http://localhost:18080/api/v1/challenges \
  -H "Authorization: Bearer $PLAYER_TOKEN")"
printf '%s\n' "$CHALLENGES_JSON"

LOG_TRAIL_ID="$(printf '%s' "$CHALLENGES_JSON" | python3 -c 'import json,sys; items=json.load(sys.stdin)["items"]; print(next(i["id"] for i in items if i["title"] == "Log Trail"))')"

START_JSON="$(curl -fsS -X POST http://localhost:18080/api/v1/instances/start \
  -H "Authorization: Bearer $PLAYER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"challengeId\":\"$LOG_TRAIL_ID\"}")"
printf '%s\n' "$START_JSON"

ME_RUNNING_JSON="$(curl -fsS http://localhost:18080/api/v1/instances/me \
  -H "Authorization: Bearer $PLAYER_TOKEN")"
printf '%s\n' "$ME_RUNNING_JSON"

ACTIVE_CONFLICT_JSON="$(curl -sS -X POST http://localhost:18080/api/v1/instances/start \
  -H "Authorization: Bearer $PLAYER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"challengeId\":\"$LOG_TRAIL_ID\"}")"
printf '%s\n' "$ACTIVE_CONFLICT_JSON"

INSTANCE_ID="$(printf '%s' "$START_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"

STOP_JSON="$(curl -fsS -X POST http://localhost:18080/api/v1/instances/stop \
  -H "Authorization: Bearer $PLAYER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"instanceId\":\"$INSTANCE_ID\"}")"
printf '%s\n' "$STOP_JSON"

COOLDOWN_JSON="$(curl -sS -X POST http://localhost:18080/api/v1/instances/start \
  -H "Authorization: Bearer $PLAYER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"challengeId\":\"$LOG_TRAIL_ID\"}")"
printf '%s\n' "$COOLDOWN_JSON"

ME_COOLDOWN_JSON="$(curl -fsS http://localhost:18080/api/v1/instances/me \
  -H "Authorization: Bearer $PLAYER_TOKEN")"
printf '%s\n' "$ME_COOLDOWN_JSON"
```

Expected outputs:
- Start response: HTTP `201`, `status=running`, `accessInfo` present.
- `/instances/me` while active: `instance` object present.
- Active conflict response: HTTP `409`, `error.code=INSTANCE_ACTIVE_EXISTS`, optional additive `error.details` may include `activeInstanceId`, `activeChallengeId`, `activeStatus`, `activeStartedAt`, `activeExpiresAt`.
- Cooldown conflict response: HTTP `409`, `error.code=INSTANCE_COOLDOWN_ACTIVE`, `error.details.retryAt` present.
- `/instances/me` during cooldown: `instance` can be `null` and additive `cooldown.retryAt` can be present.

## UI route alignment (seed contracts used by frontend)

- `/login` establishes session used by player routes.
- `/challenges` lists published challenges for player role.
- `/challenges/[id]` consumes start/stop/cooldown contracts from `/api/v1/instances/*`.
- `/admin/challenges` and `/admin/users` require admin role.
  - Missing session redirects to `/login`.
  - Authenticated non-admin redirects to `/challenges`.

Troubleshooting (`INSTANCE_RUNTIME_START_FAILED`):

```bash
docker compose logs backend --tail 200
docker compose exec backend ls -l /var/run/docker.sock
docker compose exec backend docker version
docker compose exec worker docker version
docker compose up -d --build backend worker
```

If docker socket/controller access is unavailable inside backend/worker, `/instances/start` fails with 500 until runtime access is restored.

Troubleshooting (auth/session redirect confusion):

```bash
curl -fsS -X POST http://localhost:18080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"player@ctf.local","password":"PlayerPass123!"}'
```

If API login succeeds but UI still redirects unexpectedly, sign out and sign in again at `http://localhost:13001/login`, then retry the target route.

Troubleshooting (admin role guard redirect behavior):

```bash
pnpm -C frontend test "src/__tests__/admin-challenges-access.test.ts"
pnpm -C frontend test "src/__tests__/admin-users-access.test.ts"
```

Expected: unauthenticated -> `/login`, non-admin -> `/challenges`, admin -> admin route renders.

Troubleshooting (cooldown-after-reload expectations):

```bash
curl -fsS http://localhost:18080/api/v1/instances/me \
  -H "Authorization: Bearer $PLAYER_TOKEN"
```

During active cooldown, verify response can include `{"instance":null,"cooldown":{"retryAt":"..."}}`.

Note: this API list maps directly to frontend routes `/challenges`, `/challenges/[id]`, `/admin/challenges`, and `/admin/users`.
