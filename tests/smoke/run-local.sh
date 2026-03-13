#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
API_PID=""

cleanup() {
  if [[ -n "$API_PID" ]] && kill -0 "$API_PID" 2>/dev/null; then
    kill "$API_PID" >/dev/null 2>&1 || true
    wait "$API_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

cd "$ROOT_DIR"

: "${DATABASE_URL:=postgres://postgres:postgres@127.0.0.1:5432/ctf?sslmode=disable}"
: "${JWT_SECRET:=dev-only-insecure-jwt-secret}"
: "${HTTP_ADDR:=:8080}"
: "${PUBLIC_BASE_URL:=http://127.0.0.1:8080}"
: "${RUNTIME_PUBLIC_BASE_URL:=$PUBLIC_BASE_URL}"
: "${RUNTIME_PORT_MIN:=20000}"
: "${RUNTIME_PORT_MAX:=20499}"
: "${APP_ENV:=development}"
: "${ADMIN_EMAIL:=admin@ctf.local}"
: "${ADMIN_PASSWORD:=Admin123!}"

export DATABASE_URL JWT_SECRET HTTP_ADDR PUBLIC_BASE_URL RUNTIME_PUBLIC_BASE_URL RUNTIME_PORT_MIN RUNTIME_PORT_MAX APP_ENV ADMIN_EMAIL ADMIN_PASSWORD

command -v docker >/dev/null 2>&1 || { echo 'missing docker' >&2; exit 1; }
command -v psql >/dev/null 2>&1 || { echo 'missing psql' >&2; exit 1; }
command -v curl >/dev/null 2>&1 || { echo 'missing curl' >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo 'missing python3' >&2; exit 1; }

printf '[smoke-local] building dynamic challenge image\n'
scripts/build-web-welcome-image.sh

printf '[smoke-local] starting postgres and redis\n'
docker compose -f deploy/docker-compose.yml up -d postgres redis

printf '[smoke-local] waiting for postgres\n'
for _ in $(seq 1 60); do
  if psql "$DATABASE_URL" -Atqc 'select 1' >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
psql "$DATABASE_URL" -Atqc 'select 1' >/dev/null 2>&1 || {
  docker compose -f deploy/docker-compose.yml logs --tail=200 postgres >&2 || true
  echo '[smoke-local] postgres did not become ready' >&2
  exit 1
}

printf '[smoke-local] applying migrations and development seed\n'
scripts/apply-migrations.sh
scripts/dev-seed.sh

printf '[smoke-local] starting api\n'
(
  cd backend
  GOCACHE=/tmp/ctf-go-build GOMODCACHE=/tmp/ctf-go-mod \
  HTTP_ADDR="$HTTP_ADDR" \
  APP_ENV="$APP_ENV" \
  DATABASE_URL="$DATABASE_URL" \
  JWT_SECRET="$JWT_SECRET" \
  PUBLIC_BASE_URL="$PUBLIC_BASE_URL" \
  RUNTIME_PUBLIC_BASE_URL="$RUNTIME_PUBLIC_BASE_URL" \
  RUNTIME_PORT_MIN="$RUNTIME_PORT_MIN" \
  RUNTIME_PORT_MAX="$RUNTIME_PORT_MAX" \
  go run ./cmd/api
) >/tmp/ctf-smoke-api.log 2>&1 &
API_PID="$!"

printf '[smoke-local] waiting for readiness\n'
for _ in $(seq 1 60); do
  if curl -fsS "${PUBLIC_BASE_URL}/api/v1/ready" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
curl -fsS "${PUBLIC_BASE_URL}/api/v1/ready" >/dev/null 2>&1 || {
  cat /tmp/ctf-smoke-api.log >&2 || true
  echo '[smoke-local] api did not become ready' >&2
  exit 1
}

printf '[smoke-local] running smoke checks\n'
BASE_URL="$PUBLIC_BASE_URL" tests/smoke/smoke.sh

printf '[smoke-local] smoke flow completed\n'
