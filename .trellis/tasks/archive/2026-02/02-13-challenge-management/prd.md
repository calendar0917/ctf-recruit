# Challenge Management

## Goal
Implement challenge management APIs for CTF platform, including admin CRUD and player-visible listing/detail.

## Requirements
- Add `challenge` module under `internal/modules/challenge/` with handler/service/repository layering.
- Define challenge schema and migration.
- Admin-only endpoints:
  - `POST /api/v1/challenges` (create)
  - `PUT /api/v1/challenges/:id` (update)
  - `DELETE /api/v1/challenges/:id` (delete/soft-delete)
- Player/admin accessible endpoints:
  - `GET /api/v1/challenges` (list, only published challenges for player)
  - `GET /api/v1/challenges/:id` (detail, only published for player)
- Challenge fields for MVP:
  - `id`, `title`, `description`, `category`, `difficulty`, `points`, `flag_hash`, `is_published`, timestamps
- Never expose raw flag or hash to clients.
- Use auth + RBAC middleware:
  - admin endpoints require `admin`
  - player endpoints require authenticated user

## Acceptance Criteria
- [ ] Migration exists for challenge table and runs cleanly.
- [ ] Admin can create/update/delete challenge.
- [ ] Player can list/detail published challenges only.
- [ ] Responses exclude `flag_hash`.
- [ ] Unified error schema used on all challenge endpoints.
- [ ] Tests cover service-level create/list visibility behavior.

## Technical Notes
- Keep repository queries paginated for list endpoint.
- `difficulty` can be string enum (`easy|medium|hard`) for MVP.
- Use soft-delete boolean or `deleted_at` strategy, but keep API behavior consistent.
