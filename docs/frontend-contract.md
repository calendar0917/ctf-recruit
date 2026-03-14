# Frontend Contract (Pragmatic)

This document is a frontend-oriented contract for the current backend.

- Base path: `/api/v1`
- Auth: `Authorization: Bearer <token>`
- Error shape: `{ "error": "<code>", "message": "<message>" }`
- Time strings: RFC3339 (UTC)

## Contest Phase Gating

Always call `GET /contest` first and gate UI based on `phase.*`.

Phase-related stable error codes:

- `contest_not_public` (announcements / challenge list / challenge detail / attachments)
- `scoreboard_not_public`
- `submission_closed`
- `runtime_closed`
- `registration_closed`

Recommended frontend behavior:

- Do not rely on `message` for logic.
- Map `error` codes to friendly copy; keep a fallback for unknown codes.

## Challenge Reference Rules

Many player routes accept `{challengeID}` which can be either:

- numeric ID (example: `1`)
- slug (example: `web-welcome`)

Server disambiguation:

- if the path value is all digits, backend matches by ID first.

Important type mismatch to handle in frontend:

- `GET /challenges` returns `items[].id` as **string**
- `GET /challenges/{challengeID}` returns `challenge.id` as **number**

Treat challenge references as strings on the frontend.

## Minimal Flow (Player)

1. `GET /contest`
2. `GET /challenges` -> list
3. `GET /challenges/{challengeID}` -> detail + attachments
4. `POST /auth/register` or `POST /auth/login`
5. `POST /challenges/{challengeID}/submissions`
6. If `dynamic=true` and `phase.runtime_allowed=true`:
   - `POST /challenges/{challengeID}/instances/me` -> instance
   - `GET /challenges/{challengeID}/instances/me` -> poll
   - `POST /challenges/{challengeID}/instances/me/renew`
   - `DELETE /challenges/{challengeID}/instances/me`

## Minimal Flow (Admin)

1. Login -> `user.role` must satisfy permission mapping
2. `GET /admin/contest` and `PATCH /admin/contest`
3. `GET/POST/PATCH /admin/challenges`
4. `POST /admin/challenges/{challengeID}/attachments` (multipart form field name: `file`)
5. `GET /admin/submissions`, `GET /admin/instances`, `POST /admin/instances/{instanceID}/terminate`
6. `GET/PATCH /admin/users/{userID}`
7. `GET /admin/audit-logs`
