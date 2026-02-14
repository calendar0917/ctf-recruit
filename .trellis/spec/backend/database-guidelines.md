# Database Guidelines

> Database patterns and conventions for this project.

---

## Overview

- DB: **PostgreSQL**
- Access layer: **GORM**
- Migrations: SQL migrations under `backend/migrations/`
- Cache/rate-limit helper: **Redis**

Principles:
- Keep SQL schema explicit in migrations.
- Keep business rules out of repository methods.
- Use transactions for multi-step writes that must be atomic.

---

## Query Patterns

- One repository per module (`auth`, `challenge`, `submission`).
- Use context-aware DB operations.
- Avoid N+1; preload or join intentionally.
- Pagination required for list endpoints (`limit`, `offset` or cursor).
- For leaderboard reads, allow Redis cache with TTL and explicit invalidation on score-changing writes.

---

## Migrations

- Migration files are append-only; never rewrite applied migrations.
- One migration should represent one coherent schema change.
- Include both up/down SQL where supported by the migration tool.
- Naming format:
  - `YYYYMMDDHHMMSS_create_users_table.up.sql`
  - `YYYYMMDDHHMMSS_create_users_table.down.sql`

---

## Naming Conventions

- Tables: plural snake_case (`users`, `challenge_submissions`).
- Columns: snake_case (`created_at`, `updated_at`, `is_active`).
- PK: `id` (UUID preferred for external-facing entities).
- FK naming: `<ref_table_singular>_id` (`user_id`, `challenge_id`).
- Index names: `idx_<table>_<column_list>`.
- Unique names: `uq_<table>_<column_list>`.

---

## Common Mistakes

- Performing business validation only in handlers and skipping service-level checks.
- Missing transaction boundaries for score update + submission write.
- Returning huge unpaginated lists.
- Putting migration logic in app startup code instead of migration pipeline.
