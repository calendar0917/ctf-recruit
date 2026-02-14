# Scoreboard

## Goal
Implement leaderboard API based on awarded points from submissions.

## Requirements
- Add `scoreboard` module under `internal/modules/scoreboard/` with handler/service/repository layering.
- Endpoint:
  - `GET /api/v1/scoreboard`
- Scoreboard should aggregate points per user from submissions (`awarded_points > 0`).
- Include basic ranking fields:
  - `rank`, `user_id`, `display_name`, `total_points`, `solved_count`
- Ranking order:
  1) higher `total_points` first,
  2) for tie: earlier last accepted submission time first,
  3) stable tie-breaker by `user_id`.
- Support pagination params: `limit`, `offset`.
- Endpoint available to authenticated users.

## Acceptance Criteria
- [ ] `GET /api/v1/scoreboard` returns paginated ranking list.
- [ ] Points are computed from awarded submissions only (no double counting).
- [ ] Solved count reflects number of distinct challenges solved.
- [ ] Ranking tie-break rules are applied consistently.
- [ ] API response uses unified error handling when invalid query params are supplied.
- [ ] Tests cover ranking order and tie-break behavior.

## Technical Notes
- Use SQL/GORM aggregation with joins to users/submissions.
- Never expose sensitive auth data.
- Keep implementation query-efficient (single aggregate query where possible).
