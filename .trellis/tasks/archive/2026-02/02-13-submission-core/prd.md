# Submission Core

## Goal
Implement core flag submission and scoring flow for the CTF platform.

## Requirements
- Add `submission` module under `internal/modules/submission/` with handler/service/repository layering.
- Add submission schema and migration for submission records.
- Endpoint:
  - `POST /api/v1/submissions` (authenticated player/admin)
- Submission payload includes `challenge_id` and `flag`.
- Validate challenge exists and is published for player submissions.
- Compare submitted flag with stored challenge flag hash.
- Persist submission record with result status (`correct` / `wrong`) and timestamp.
- Implement first-blood/duplicate prevention logic for score:
  - First correct submission per user/challenge awards points.
  - Repeated correct submissions for same user/challenge do not award additional points.
- Expose score-related data needed for future leaderboard (minimal user score aggregate update or query-ready records).
- Never log or return raw flag.

## Acceptance Criteria
- [ ] Migration exists for submissions table and runs cleanly.
- [ ] `POST /api/v1/submissions` accepts submission and returns result.
- [ ] Correct flag marks result as correct.
- [ ] Wrong flag marks result as wrong.
- [ ] Duplicate correct submission does not add score twice.
- [ ] API responses use unified error schema.
- [ ] Tests cover correct/wrong/duplicate scoring behavior.

## Technical Notes
- Use SHA-256 hash compare with existing `challenge.flag_hash`.
- Keep scoring state simple for now; can derive leaderboard from accepted submissions.
- Restrict player submission to published challenges; admin can submit regardless for testing.
