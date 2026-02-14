# Dynamic Judge Skeleton

## Goal
Set up non-executing infrastructure skeleton for future dynamic container judging.

## Requirements
- Add `judge` module and worker entrypoint skeleton (no real container execution yet).
- Introduce queue abstraction interface for judging jobs.
- Add persistent judge job table migration.
- Extend submission flow to optionally enqueue dynamic judge job when challenge mode is dynamic.
- Add placeholder worker loop that fetches pending jobs and marks lifecycle states (`queued` -> `running` -> `done/failed` mock).
- Add configs/env placeholders for worker polling interval and max concurrency.
- Keep static judging behavior unchanged.

## Acceptance Criteria
- [ ] Worker skeleton exists and compiles.
- [ ] Queue/job interface and implementation scaffolds exist.
- [ ] Migration exists for judge jobs.
- [ ] Submission path can create queue job for dynamic challenge mode (without real execution).
- [ ] Job lifecycle state transitions are represented in code and storage.
- [ ] Tests cover enqueue and mock lifecycle transition behavior.

## Technical Notes
- Do not invoke Docker/container runtime in this task.
- Treat dynamic judge result as deterministic mock placeholder.
- Ensure code paths are clearly marked for future real judge integration.
