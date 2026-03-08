#!/usr/bin/env bash
set -euo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"

for f in backend/migrations/*.sql; do
  echo "applying $(basename "$f")"
  psql "$DATABASE_URL" -f "$f"
done
