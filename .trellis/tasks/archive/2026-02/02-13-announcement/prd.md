# Announcement

## Goal
Implement announcement module for CTF platform with admin management and authenticated user read access.

## Requirements
- Add `announcement` module under `internal/modules/announcement/` with handler/service/repository layering.
- Add migration for announcements table.
- Admin-only endpoints:
  - `POST /api/v1/announcements`
  - `PUT /api/v1/announcements/:id`
  - `DELETE /api/v1/announcements/:id`
- Authenticated endpoints:
  - `GET /api/v1/announcements` (list published announcements)
  - `GET /api/v1/announcements/:id` (detail; players only published, admins any)
- Fields for MVP:
  - `id`, `title`, `content`, `is_published`, `published_at`, timestamps
- List endpoint supports pagination (`limit`, `offset`) and returns newest first.

## Acceptance Criteria
- [ ] Migration exists and is valid.
- [ ] Admin can create/update/delete announcements.
- [ ] Player can list and view only published announcements.
- [ ] Admin can view unpublished announcements.
- [ ] Unified error schema used for validation/not-found errors.
- [ ] Tests cover publish visibility and ordering behavior.

## Technical Notes
- For published announcement creation/update, set `published_at` if absent.
- Soft-delete or hard-delete is acceptable for MVP; keep behavior consistent.
- No HTML sanitization in this task; store plain text/markdown content.
