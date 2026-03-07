# Instance lifecycle contract baseline

This document is the implementation-accurate reference for instance lifecycle behavior across backend API and current frontend challenge detail UI.

## Scope and source of truth

- Backend routes: `backend/internal/modules/instance/handler.go`
- Backend service rules: `backend/internal/modules/instance/service.go`
- State transitions: `backend/internal/modules/instance/state_machine.go`
- Persistence behavior: `backend/internal/modules/instance/repository.go`
- Error envelope: `backend/internal/middleware/error_handler.go`
- Frontend orchestration: `frontend/src/app/challenges/[id]/page.tsx`
- Frontend rendering rules: `frontend/src/components/challenge/ChallengeDetail.tsx`
- Existing runbook baseline: `README.md` section `Instance lifecycle API quick runbook (reproducible)`

## Global API error envelope

All instance endpoints use Fiber global `ErrorHandler` and return this error shape for app errors:

```json
{
  "error": {
    "code": "STRING_CODE",
    "message": "Human readable message",
    "details": {}
  },
  "requestId": "req_xxx"
}
```

Notes:

- `requestId` is included when request-id middleware populated it.
- `details` is optional and only present when backend sets it.
- Cooldown rejection uses `details.retryAt` (RFC3339).
- Active-instance conflict (`INSTANCE_ACTIVE_EXISTS`) may include additive `details` with active instance metadata when available.

## Endpoint contracts

## `POST /api/v1/instances/start`

Authentication: required bearer token.

Request body:

```json
{
  "challengeId": "uuid"
}
```

Success response: `201 Created`, body is `InstanceResponse`.

Current app wiring behavior:

- Router uses `NewServiceWithRuntime(...)`, so successful calls transition to `running` before response.
- `startedAt` and `expiresAt` are set when status becomes `running`.
- `accessInfo` is present only when runtime config exposes a port.

## `POST /api/v1/instances/stop`

Authentication: required bearer token.

Request body (optional):

```json
{
  "instanceId": "uuid"
}
```

Resolution behavior:

- If `instanceId` is present, backend fetches that specific instance.
- If omitted, backend fetches current user active instance (`starting` or `running` or `stopping`) and then enforces `status == running` before stop.

Success response: `200 OK`, body is `InstanceResponse` with:

- `status = "stopped"`
- `cooldownUntil` set to `now + cooldown` (default one minute)

## `GET /api/v1/instances/me`

Authentication: required bearer token.

Success response: `200 OK`

```json
{
  "instance": null
}
```

or

```json
{
  "instance": {
    "id": "uuid",
    "userId": "uuid",
    "challengeId": "uuid",
    "status": "starting|running|stopping",
    "containerId": "string?",
    "accessInfo": {
      "host": "string",
      "port": 0,
      "connectionString": "string?"
    },
    "startedAt": "RFC3339?",
    "expiresAt": "RFC3339?",
    "cooldownUntil": "RFC3339?"
  }
}
```

Important current semantics:

- `/instances/me` only returns active statuses from repository filter: `starting`, `running`, `stopping`.
- Cooldown-only records are not returned by this endpoint.

## Lifecycle status and transition matrix

Statuses come from `instance.Status` enum:

- `starting`, `running`, `stopping`, `stopped`, `expired`, `failed`, `cooldown`

Allowed transition matrix from `CanTransition(from, to)`:

| From | Allowed to |
|---|---|
| starting | starting, running, failed |
| running | running, stopping, expired, failed |
| stopping | stopping, stopped, expired, failed |
| stopped | stopped, cooldown |
| expired | expired, cooldown |
| failed | failed, cooldown |
| cooldown | cooldown |

Implementation path matrix:

| Trigger | From | To | Side effects |
|---|---|---|---|
| `POST /instances/start` create row | none | starting | Creates new instance row, one active instance lock per user |
| Runtime start success in service | starting | running | Sets `startedAt=now`, `expiresAt=now+ttl`, clears cooldown |
| Runtime start failure in service | starting | failed | Returns 500 `INSTANCE_RUNTIME_START_FAILED` |
| `POST /instances/stop` step 1 | running | stopping | Keeps container id for stop attempt |
| `POST /instances/stop` step 2 | stopping | stopped | Sets `cooldownUntil=now+cooldown` |
| `POST /instances/stop` runtime failure | stopping | failed | Returns 500 `INSTANCE_RUNTIME_STOP_FAILED`, cooldown set on failed transition |
| Sweeper expiry process | running | expired | Then cooldown timestamp is set by transition rule |

## Backend error mapping matrix

Endpoint-specific app errors and current HTTP statuses:

| Endpoint | Condition | HTTP | `error.code` | `error.details` |
|---|---|---:|---|---|
| `POST /instances/start` | malformed JSON body | 400 | `INSTANCE_INVALID_PAYLOAD` | none |
| `POST /instances/start` | `challengeId` missing/invalid UUID | 400 | `INSTANCE_VALIDATION_ERROR` | none |
| `POST /instances/start` | user already has active `starting/running` | 409 | `INSTANCE_ACTIVE_EXISTS` | optional active-instance metadata (`activeInstanceId`, `activeUserId`, `activeChallengeId`, `activeStatus`, `activeStartedAt?`, `activeExpiresAt?`) |
| `POST /instances/start` | cooldown still active | 409 | `INSTANCE_COOLDOWN_ACTIVE` | `{ "retryAt": "RFC3339" }` |
| `POST /instances/start` | challenge missing/unpublished for runtime | 404 | `INSTANCE_CHALLENGE_NOT_FOUND` | none |
| `POST /instances/start` | challenge runtime image missing | 400 | `INSTANCE_CHALLENGE_RUNTIME_MISSING` | none |
| `POST /instances/start` | generic create/start failure | 500 | `INSTANCE_START_FAILED` | none |
| `POST /instances/start` | runtime container start failure | 500 | `INSTANCE_RUNTIME_START_FAILED` | none |
| `POST /instances/start` | transition to running fails | 500 | `INSTANCE_TRANSITION_FAILED` | none |
| `POST /instances/stop` | malformed JSON body | 400 | `INSTANCE_INVALID_PAYLOAD` | none |
| `POST /instances/stop` | `instanceId` invalid UUID | 400 | `INSTANCE_VALIDATION_ERROR` | none |
| `POST /instances/stop` | resolved target not found | 404 | `INSTANCE_NOT_FOUND` | none |
| `POST /instances/stop` | target owned by another user | 403 | `INSTANCE_FORBIDDEN` | none |
| `POST /instances/stop` | target status is not `running` | 409 | `INSTANCE_NOT_RUNNING` | none |
| `POST /instances/stop` | invalid transition in stop flow | 409 | `INSTANCE_INVALID_TRANSITION` | none |
| `POST /instances/stop` | runtime container stop failure | 500 | `INSTANCE_RUNTIME_STOP_FAILED` | none |
| `POST /instances/stop` | other transition failures | 500 | `INSTANCE_TRANSITION_FAILED` | none |
| `GET /instances/me` | fetch active instance fails | 500 | `INSTANCE_FETCH_FAILED` | none |

`INSTANCE_ACTIVE_EXISTS` details semantics:

- Additive and backward-compatible: `error.code` and `error.message` are unchanged.
- `error.details` is optional and can be ignored by legacy clients.
- When present, timestamp fields (`activeStartedAt`, `activeExpiresAt`) are RFC3339.

Shared auth/middleware errors that can apply to all three endpoints:

| Condition | HTTP | `error.code` |
|---|---:|---|
| Missing bearer token | 401 | `AUTH_MISSING_TOKEN` |
| Invalid/expired bearer token | 401 | `AUTH_INVALID_TOKEN` |

## Frontend UI reaction mapping matrix

Current page scope: `frontend/src/app/challenges/[id]/page.tsx` and `ChallengeDetail.tsx`.

| Backend/API signal | Frontend state update | User-visible UI reaction |
|---|---|---|
| Initial load `GET /instances/me` returns active instance | `setInstance(instanceResp.instance)` and `setCooldownUntil(instanceResp.instance?.cooldownUntil)` | Shows instance status from instance object. Stop button shown for `starting/running/stopping`. |
| Start success `POST /instances/start` 201 | `setInstance(started)`, `setCooldownUntil(started.cooldownUntil)` | Button toggles to stop path once status is running. Expires/access info shown when present. |
| Start conflict `INSTANCE_COOLDOWN_ACTIVE` with `details.retryAt` | `setCooldownUntil(details.retryAt)`, then fallback refresh from `/instances/me` on catch | Status text becomes `cooldown` when no instance and remaining seconds > 0, Start disabled, retry timestamp and countdown shown. |
| Start conflict other (for example `INSTANCE_ACTIVE_EXISTS`) | `setInstanceError(err.message)` and refresh `/instances/me` | Error text rendered. UI recovers to active instance view if backend reports active instance. Optional conflict `details` can be consumed for richer context but is not required for legacy behavior. |
| Stop success `POST /instances/stop` 200 | `setInstance(stopped)`, `setCooldownUntil(stopped.cooldownUntil)` | Since stopped is not treated as running, Start button appears. If cooldown still active, Start stays disabled with retry info. |
| Stop failure (for example forbidden/not running) | `setInstanceError(err.message)` and refresh `/instances/me` | Error text shown. UI attempts to reconcile by re-fetching current active instance. |
| Polling while instance transitional (`starting` or `stopping`) | Interval refresh every 3s via `handleManualRefresh` | UI updates status after backend transition completes. |
| Cooldown countdown tick | local `nowMs` interval every 1s | Remaining seconds and Start disabled/enabled update in place, no API call needed. |

Current frontend control logic summary:

- `instanceRunning` is true for `starting`, `running`, or `stopping`, this drives Stop button visibility.
- `startDisabled` is true when action in progress or cooldown active.
- Cooldown UI can be driven by locally stored `cooldownUntil` even when `/instances/me` returns `instance: null`.

## Known ambiguity and risk points

## 1) Reload cooldown visibility gap

- Backend stores cooldown on non-active statuses and `/instances/me` only returns active statuses.
- Frontend currently reads cooldown from `/instances/me` and from immediate error details during same page session.
- After full page reload during cooldown, frontend may lose cooldown context because `/instances/me` returns `instance: null` and no separate cooldown endpoint exists.

## 2) Cross-challenge context mismatch risk

- Challenge detail page calls `/instances/me` global active instance without challenge scoping.
- If user has active instance for challenge A and opens challenge B, UI can show Stop state or instance info that belongs to challenge A.
- This is implementation-accurate today and should be treated as a known baseline risk for follow-up hardening tasks.

## 3) Runtime-dependent start response details

- Router wiring currently enables runtime controller, so success path is effectively `running` state with TTL fields.
- `accessInfo` depends on challenge runtime configuration and exposed port mapping, not guaranteed on every successful start.
