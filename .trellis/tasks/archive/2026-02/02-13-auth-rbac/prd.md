# Auth & RBAC

## Goal
Implement backend authentication with JWT and role-based access control for the CTF platform.

## Requirements
- Create `auth` module with handler/service/repository layering.
- User registration endpoint (email, password, display name).
- User login endpoint returning JWT access token.
- Passwords hashed with bcrypt (never stored in plain text).
- Define roles: `admin`, `player` (default on signup).
- Auth middleware validates JWT and injects user context.
- RBAC middleware/guard enforces role requirements for admin routes.
- Add database schema + migrations for users.
- Standardized error responses using project error format.

## Acceptance Criteria
- [ ] `/api/v1/auth/register` creates a user and returns safe user payload.
- [ ] `/api/v1/auth/login` validates credentials and returns JWT.
- [ ] JWT-protected route example exists (e.g. `/api/v1/auth/me`).
- [ ] RBAC-protected route example exists (admin-only sample).
- [ ] Passwords are hashed and verified with bcrypt.
- [ ] Migrations provided for user table with required fields.
- [ ] Tests cover auth service happy path + invalid credentials.

## Technical Notes
- Use GORM for database access.
- JWT secret and token TTL from env.
- Error codes use `AUTH_*` namespace (e.g., `AUTH_INVALID_CREDENTIALS`).
- Keep auth module self-contained under `internal/modules/auth/`.
