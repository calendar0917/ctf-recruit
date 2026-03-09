#!/usr/bin/env bash
set -euo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"

if [[ -d backend && -f backend/go.mod ]]; then
  REPO_ROOT="$(pwd)"
  cd backend

  args=()
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --path)
        shift
        [[ $# -gt 0 ]] || { echo "--path requires a value" >&2; exit 1; }
        args+=(--path "$REPO_ROOT/$1")
        ;;
      --root)
        shift
        [[ $# -gt 0 ]] || { echo "--root requires a value" >&2; exit 1; }
        args+=(--root "$REPO_ROOT/$1")
        ;;
      *)
        args+=("$1")
        ;;
    esac
    shift
  done

  exec go run ./cmd/import-challenges "${args[@]}"
fi

exec /usr/local/bin/import-challenges "$@"
