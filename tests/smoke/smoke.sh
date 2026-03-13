#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-ctf}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@ctf.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-Admin123!}"
PLAYER_PASSWORD="${PLAYER_PASSWORD:-PlayerPass123!}"
PLAYER_DISPLAY_NAME="${PLAYER_DISPLAY_NAME:-Smoke Player}"
PLAYER_USERNAME="${PLAYER_USERNAME:-smoke_$(date +%s)}"
PLAYER_EMAIL="${PLAYER_EMAIL:-${PLAYER_USERNAME}@example.com}"
REQUIRE_DYNAMIC_IMAGE="${REQUIRE_DYNAMIC_IMAGE:-1}"
CHECK_DB="${CHECK_DB:-1}"
HAS_JQ=0

TMPDIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMPDIR"
}
trap cleanup EXIT

log() {
  printf '[smoke] %s\n' "$*"
}

die() {
  printf '[smoke][fail] %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

json_compact() {
  local payload="$1"
  if [[ "$HAS_JQ" == "1" ]]; then
    jq -cn "$payload"
    return
  fi
  python3 - <<'PY' "$payload"
import json
import sys
print(json.dumps(eval(sys.argv[1]), ensure_ascii=False, separators=(",", ":")))
PY
}

json_get() {
  local file="$1"
  shift
  if [[ "$HAS_JQ" == "1" ]]; then
    jq -er "$@" "$file"
    return
  fi
  local expr="$1"
  shift || true
  python3 - "$file" "$expr" "$@" <<'PY'
import json
import sys

path = sys.argv[1]
expr = sys.argv[2]
extra = sys.argv[3:]
with open(path, 'r', encoding='utf-8') as fh:
    data = json.load(fh)

args = {}
i = 0
while i < len(extra):
    key = extra[i]
    if key.startswith('--arg'):
        name = extra[i + 1]
        value = extra[i + 2]
        args[name] = value
        i += 3
    else:
        i += 1


def output(value):
    if isinstance(value, bool):
        print('true' if value else 'false')
    elif value is None:
        print('null')
    else:
        print(value)

if expr == '.database_connected == true':
    if data.get('database_connected') is True:
        print('true')
        sys.exit(0)
    sys.exit(1)
if expr == '.user.role':
    output(data['user']['role'])
    sys.exit(0)
if expr == '.token':
    output(data['token'])
    sys.exit(0)
if expr == '.items[] | select(.slug == "web-welcome") | .id':
    for item in data['items']:
        if item.get('slug') == 'web-welcome':
            output(item['id'])
            sys.exit(0)
    sys.exit(1)
if expr == '.correct == true':
    sys.exit(0 if data.get('correct') is True else 1)
if expr == '.solved == true':
    sys.exit(0 if data.get('solved') is True else 1)
if expr == '.status == "running"':
    sys.exit(0 if data.get('status') == 'running' else 1)
if expr == '.instance.status == "terminated"':
    sys.exit(0 if data.get('instance', {}).get('status') == 'terminated' else 1)
if expr == '.items[] | select(.username == $username) | .id':
    username = args['username']
    for item in data['items']:
        if item.get('username') == username:
            output(item['id'])
            sys.exit(0)
    sys.exit(1)
if expr == '.items[] | select(.challenge_slug == "web-welcome" and .username == $username) | .id':
    username = args['username']
    for item in data['items']:
        if item.get('challenge_slug') == 'web-welcome' and item.get('username') == username:
            output(item['id'])
            sys.exit(0)
    sys.exit(1)
if expr == '.items[] | select(.username == $username) | .score >= 100':
    username = args['username']
    for item in data['items']:
        if item.get('username') == username and int(item.get('score', 0)) >= 100:
            print('true')
            sys.exit(0)
    sys.exit(1)

raise SystemExit(f'unsupported json_get expression without jq: {expr}')
PY
}

build_register_payload() {
  if [[ "$HAS_JQ" == "1" ]]; then
    jq -cn \
      --arg username "$PLAYER_USERNAME" \
      --arg email "$PLAYER_EMAIL" \
      --arg password "$PLAYER_PASSWORD" \
      --arg displayName "$PLAYER_DISPLAY_NAME" \
      '{username:$username,email:$email,password:$password,display_name:$displayName}'
    return
  fi
  python3 - <<'PY' "$PLAYER_USERNAME" "$PLAYER_EMAIL" "$PLAYER_PASSWORD" "$PLAYER_DISPLAY_NAME"
import json
import sys
print(json.dumps({
    'username': sys.argv[1],
    'email': sys.argv[2],
    'password': sys.argv[3],
    'display_name': sys.argv[4],
}, ensure_ascii=False, separators=(",", ":")))
PY
}

build_login_payload() {
  local identifier="$1"
  local password="$2"
  if [[ "$HAS_JQ" == "1" ]]; then
    jq -cn --arg identifier "$identifier" --arg password "$password" '{identifier:$identifier,password:$password}'
    return
  fi
  python3 - <<'PY' "$identifier" "$password"
import json
import sys
print(json.dumps({'identifier': sys.argv[1], 'password': sys.argv[2]}, separators=(",", ":")))
PY
}

api_call() {
  local name="$1"
  local method="$2"
  local path="$3"
  local body="${4:-}"
  local token="${5:-}"
  local body_file="$TMPDIR/${name}.body"
  local status_file="$TMPDIR/${name}.status"
  local curl_args=( -sS -o "$body_file" -w '%{http_code}' -X "$method" )
  if [[ -n "$token" ]]; then
    curl_args+=( -H "Authorization: Bearer ${token}" )
  fi
  if [[ -n "$body" ]]; then
    curl_args+=( -H 'Content-Type: application/json' --data "$body" )
  fi
  local status
  status="$(curl "${curl_args[@]}" "${BASE_URL}${path}")"
  printf '%s' "$status" > "$status_file"
  printf '%s\n' "$body_file"
}

expect_status() {
  local file="$1"
  local expected="$2"
  local name="$3"
  local status_file="${file%.body}.status"
  local status
  status="$(cat "$status_file")"
  if [[ "$status" != "$expected" ]]; then
    log "$name response:"
    cat "$file" >&2
    die "$name expected HTTP $expected, got $status"
  fi
}

psql_query() {
  PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -Atqc "$1"
}

require_cmd curl
require_cmd python3
if command -v jq >/dev/null 2>&1; then
  HAS_JQ=1
fi
if [[ "$CHECK_DB" == "1" ]]; then
  require_cmd psql
fi
if [[ "$REQUIRE_DYNAMIC_IMAGE" == "1" ]]; then
  require_cmd docker
  docker image inspect ctf/web-welcome:dev >/dev/null 2>&1 || die "missing docker image ctf/web-welcome:dev, run scripts/build-web-welcome-image.sh first"
fi

fetch_instance_url() {
  local file="$1"
  if [[ "$HAS_JQ" == "1" ]]; then
    jq -er '.access_url // empty' "$file"
    return
  fi
  python3 - "$file" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, 'r', encoding='utf-8') as fh:
    data = json.load(fh)
value = data.get('access_url')
if value:
    print(value)
    sys.exit(0)
sys.exit(1)
PY
}

log "checking health and readiness"
health_body="$(api_call health GET /api/v1/health)"
expect_status "$health_body" 200 health
ready_body="$(api_call ready GET /api/v1/ready)"
expect_status "$ready_body" 200 ready
json_get "$ready_body" '.database_connected == true' >/dev/null || die 'readiness check reports database_connected=false'

log "registering smoke player ${PLAYER_USERNAME}"
register_payload="$(build_register_payload)"
register_body="$(api_call register POST /api/v1/auth/register "$register_payload")"
expect_status "$register_body" 201 register
player_role="$(json_get "$register_body" '.user.role')"
[[ "$player_role" == "player" ]] || die "unexpected player role: $player_role"

log "logging in as smoke player"
login_payload="$(build_login_payload "$PLAYER_EMAIL" "$PLAYER_PASSWORD")"
login_body="$(api_call player_login POST /api/v1/auth/login "$login_payload")"
expect_status "$login_body" 200 player_login
player_token="$(json_get "$login_body" '.token')"

log "checking public endpoints"
announcements_body="$(api_call announcements GET /api/v1/announcements)"
expect_status "$announcements_body" 200 announcements
challenges_body="$(api_call challenges GET /api/v1/challenges)"
expect_status "$challenges_body" 200 challenges
challenge_id="$(json_get "$challenges_body" '.items[] | select(.slug == "web-welcome") | .id')"
challenge_detail_body="$(api_call challenge_detail GET "/api/v1/challenges/${challenge_id}")"
expect_status "$challenge_detail_body" 200 challenge_detail
scoreboard_body="$(api_call scoreboard GET /api/v1/scoreboard)"
expect_status "$scoreboard_body" 200 scoreboard

log "checking authenticated player endpoints"
me_body="$(api_call me GET /api/v1/me '' "$player_token")"
expect_status "$me_body" 200 me
my_submissions_body="$(api_call my_submissions GET /api/v1/me/submissions '' "$player_token")"
expect_status "$my_submissions_body" 200 my_submissions
my_solves_body="$(api_call my_solves GET /api/v1/me/solves '' "$player_token")"
expect_status "$my_solves_body" 200 my_solves

log "submitting correct flag"
submit_payload='{"flag":"flag{welcome}"}'
submit_body="$(api_call submit POST "/api/v1/challenges/${challenge_id}/submissions" "$submit_payload" "$player_token")"
expect_status "$submit_body" 200 submit
json_get "$submit_body" '.correct == true' >/dev/null || die 'flag submission did not return correct=true'
json_get "$submit_body" '.solved == true' >/dev/null || die 'flag submission did not create solve state'

scoreboard_after_body="$(api_call scoreboard_after GET /api/v1/scoreboard)"
expect_status "$scoreboard_after_body" 200 scoreboard_after
json_get "$scoreboard_after_body" '.items[] | select(.username == $username) | .score >= 100' --arg username "$PLAYER_USERNAME" >/dev/null || die 'smoke player was not ranked on scoreboard after solve'

log "starting challenge instance"
instance_create_body="$(api_call instance_create POST "/api/v1/challenges/${challenge_id}/instances/me" '' "$player_token")"
expect_status "$instance_create_body" 201 instance_create
json_get "$instance_create_body" '.status == "running"' >/dev/null || die 'instance was not reported as running'

if [[ "${CHECK_INSTANCE_URL:-1}" == "1" ]]; then
  instance_url="$(fetch_instance_url "$instance_create_body")" || die 'instance response missing access_url'
  log "checking instance url ${instance_url}"
  curl -fsS --max-time 5 "${instance_url}" >/dev/null 2>&1 || die "failed to reach instance url: ${instance_url}"
fi

log "loading active instance and renewing"
instance_get_body="$(api_call instance_get GET "/api/v1/challenges/${challenge_id}/instances/me" '' "$player_token")"
expect_status "$instance_get_body" 200 instance_get
instance_renew_body="$(api_call instance_renew POST "/api/v1/challenges/${challenge_id}/instances/me/renew" '' "$player_token")"
expect_status "$instance_renew_body" 200 instance_renew

log "logging in as admin"
admin_login_payload="$(build_login_payload "$ADMIN_EMAIL" "$ADMIN_PASSWORD")"
admin_login_body="$(api_call admin_login POST /api/v1/auth/login "$admin_login_payload")"
expect_status "$admin_login_body" 200 admin_login
admin_token="$(json_get "$admin_login_body" '.token')"

log "checking admin endpoints"
admin_challenges_body="$(api_call admin_challenges GET /api/v1/admin/challenges '' "$admin_token")"
expect_status "$admin_challenges_body" 200 admin_challenges
admin_submissions_body="$(api_call admin_submissions GET /api/v1/admin/submissions '' "$admin_token")"
expect_status "$admin_submissions_body" 200 admin_submissions
admin_instances_body="$(api_call admin_instances GET /api/v1/admin/instances '' "$admin_token")"
expect_status "$admin_instances_body" 200 admin_instances
instance_id="$(json_get "$admin_instances_body" '.items[] | select(.challenge_slug == "web-welcome" and .username == $username) | .id' --arg username "$PLAYER_USERNAME")"
admin_users_body="$(api_call admin_users GET /api/v1/admin/users '' "$admin_token")"
expect_status "$admin_users_body" 200 admin_users
json_get "$admin_users_body" '.items[] | select(.username == $username) | .id' --arg username "$PLAYER_USERNAME" >/dev/null || die 'smoke player not visible in admin users list'
audit_logs_body="$(api_call audit_logs GET /api/v1/admin/audit-logs '' "$admin_token")"
expect_status "$audit_logs_body" 200 audit_logs

log "terminating instance from admin path"
admin_terminate_body="$(api_call admin_terminate POST "/api/v1/admin/instances/${instance_id}/terminate" '' "$admin_token")"
expect_status "$admin_terminate_body" 200 admin_terminate
json_get "$admin_terminate_body" '.instance.status == "terminated"' >/dev/null || die 'admin termination did not return terminated status'

instance_after_delete_body="$(api_call instance_after_delete GET "/api/v1/challenges/${challenge_id}/instances/me" '' "$player_token")"
expect_status "$instance_after_delete_body" 404 instance_after_delete

if [[ "$CHECK_DB" == "1" ]]; then
  log "checking database side effects"
  user_count="$(psql_query "select count(*) from users where username = '${PLAYER_USERNAME}'")"
  [[ "$user_count" == "1" ]] || die "expected smoke player to exist in database"
  solve_count="$(psql_query "select count(*) from solves s join users u on u.id = s.user_id where u.username = '${PLAYER_USERNAME}'")"
  [[ "$solve_count" == "1" ]] || die "expected exactly one solve record for smoke player"
  audit_count="$(psql_query "select count(*) from audit_logs where action = 'instance.terminate'")"
  [[ "$audit_count" =~ ^[0-9]+$ ]] || die 'failed to read audit log count'
  (( audit_count >= 1 )) || die 'expected at least one instance.terminate audit log'
fi

log "smoke test completed successfully"
