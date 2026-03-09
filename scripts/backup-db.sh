#!/usr/bin/env bash
set -euo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"
: "${BACKUP_DIR:=./backups}"

mkdir -p "$BACKUP_DIR"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
output_path="${BACKUP_DIR%/}/ctf-${timestamp}.sql.gz"

pg_dump "$DATABASE_URL" | gzip -c > "$output_path"
echo "backup written to $output_path"
