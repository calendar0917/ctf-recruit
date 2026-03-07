# Frontend Polish + Instance Lifecycle Coordination Plan

## TL;DR

> **Quick Summary**: Perform a medium-strength frontend refactor (security-competition visual style) while hardening instance lifecycle contract consistency between frontend/backend, adding reproducible runtime validation, and synchronizing runbook/docs with actual behavior.
>
> **Deliverables**:
> - Unified dark/high-contrast UI shell and reusable primitives across player/admin surfaces
> - Reliable, observable start/stop/cooldown UX with challenge-context correctness
> - Backend/Frontend contract alignment for instance lifecycle edge cases
> - Tests-after implementation (unit/integration/e2e) + agent-executed QA evidence
> - Updated docs with full reproducible UI+API validation path
>
> **Estimated Effort**: Large
> **Parallel Execution**: YES — 4 implementation waves + final verification wave
> **Critical Path**: T1 → T4 → T9 → T15 → T17 → T20

---

## Context

### Original Request
User requested improving the frontend because it is too basic, fixing perceived missing functionality (especially container start/stop), and ensuring frontend-backend coordination and documentation accuracy.

### Interview Summary
**Key Decisions**:
- Scope includes: instance flow page, challenge pages, global shell (home/nav/login/register), and admin pages.
- Visual direction: security competition style (dark, high contrast, clear hierarchy).
- Refactor intensity: medium (not full redesign).
- Backend changes allowed where needed for contract consistency.
- Must provide reproducible runtime validation path (user currently cannot test container lifecycle).
- Test strategy: tests-after implementation.

**Research Findings**:
- Frontend already has `instances` API client and challenge detail start/stop UI.
- Backend lifecycle routes/services/tests are present and robust.
- Main gap is reliability/clarity/consistency, not complete absence of feature.
- Cooldown/context/polling/error semantics need explicit cross-layer handling.

### Metis Review (addressed)
- Added strict guardrails against scope creep into full redesign/platform rewrite.
- Added explicit lifecycle contract matrix and status/code/UI mapping requirements.
- Added acceptance criteria for cooldown reload behavior, cross-challenge context, concurrency, and `Retry-After` handling.
- Added reproducible runtime path tasks (seeded dynamic challenge + evidence workflow).

### Defaults Applied (can be overridden)
- Default to **targeted backend contract enhancement** (not frontend-only workaround) for cooldown reload consistency.
- Default to **non-breaking contract evolution** (additive fields/semantics preferred over breaking response shape changes).
- Default to **same-language consistency per page flow** while preserving existing domain language where already user-facing and validated.

---

## Work Objectives

### Core Objective
Deliver a visually improved, production-coherent frontend and reliable instance lifecycle UX that is contract-consistent with backend behavior and fully reproducible/verified via automated and agent-executed scenarios.

### Concrete Deliverables
- Updated shared UI shell and style primitives in frontend.
- Refactored challenge detail lifecycle orchestration.
- Contract alignment updates in backend/frontend types+API handling.
- Expanded tests for lifecycle orchestration/admin flows.
- Reproducible runbook + docs aligned to real shipped behavior.

### Definition of Done
- [ ] `pnpm lint && pnpm type-check && pnpm test` pass at repo root.
- [ ] Lifecycle API runbook and UI runbook both executable with deterministic outcomes.
- [ ] Evidence artifacts exist under `.sisyphus/evidence/` for all mandatory scenarios.

### Must Have
- Start/Stop/Cooldown behavior must be clear and recoverable after reload.
- UI must not misrepresent cross-challenge active instance context.
- Frontend and backend lifecycle status/error contracts must be documented and tested.
- Docs must accurately match implemented behavior.

### Must NOT Have (Guardrails)
- No full product rebrand or deep IA rewrite.
- No unrelated backend architecture rewrite.
- No hidden contract changes without corresponding frontend/types/docs updates.
- No acceptance criteria requiring manual human-only verification.

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** — all verification is agent-executed.

### Test Decision
- **Infrastructure exists**: YES
- **Automated tests**: Tests-after
- **Framework**: Frontend `vitest`, Backend `go test`, E2E via `playwright`

### QA Policy
Every task contains executable QA scenarios (happy + failure/edge). Evidence paths are mandatory.

| Deliverable Type | Verification Tool | Method |
|---|---|---|
| Frontend/UI | Playwright | Navigate, interact, assert DOM/text/button states, screenshot |
| API/Backend | Bash (curl) | Request lifecycle endpoints, assert status/error/body |
| Module/Logic | Bash (test runners) | Run focused test files and assert pass/fail counts |
| CLI/Troubleshooting | interactive_bash (tmux) | Run commands, validate output/exit behavior |

---

## Execution Strategy

### Parallel Execution Waves

```text
Wave 1 (Foundation, 7 tasks):
1 Contract matrix + status/error mapping baseline
2 Global dark shell + tokens in globals/layout/nav
3 Shared state/feedback primitives
4 HTTP error/retry-after normalization path
5 Reproducible runtime seed/runtime prerequisites
6 Challenge detail orchestration scaffold refactor
7 Shared table/action/form shell primitives for admin

Wave 2 (Core feature hardening, 6 tasks):
8 Challenge instance context guard (cross-challenge correctness)
9 Cooldown persistence and reload-consistent visibility
10 Start/stop UX hardening (action states/recovery/diagnostics)
11 Auth/nav/home flow polish (no invisible header flicker)
12 Challenge/announcement visual harmonization
13 Admin challenge runtime-config UX + contract wiring

Wave 3 (Coverage + docs alignment, 5 tasks):
14 Backend lifecycle contract enhancement where needed
15 Frontend API/types alignment to backend contract updates
16 Page-level orchestration tests (challenge detail lifecycle)
17 Admin flow + auth-guard tests
18 Runbook/docs synchronization (API + UI reproducible flows)

Wave 4 (Stabilization, 3 tasks):
19 E2E Playwright lifecycle journeys + evidence capture
20 Regression fixes from automated/e2e feedback
21 Accessibility + responsive polish + final UX consistency sweep

Wave FINAL (Independent review, 4 parallel):
F1 Plan compliance audit (oracle)
F2 Code quality review
F3 Real QA replay of all scenarios
F4 Scope fidelity check
```

### Dependency Matrix (FULL)

| Task | Depends On | Blocks | Wave |
|---|---|---|---|
| 1 | — | 8,9,10,14,15 | 1 |
| 2 | — | 11,12,21 | 1 |
| 3 | — | 10,11,12,13,21 | 1 |
| 4 | 1 | 9,10,15,16 | 1 |
| 5 | — | 18,19 | 1 |
| 6 | 1,3 | 8,9,10,16 | 1 |
| 7 | 2,3 | 12,13,17,21 | 1 |
| 8 | 1,6 | 10,16,19 | 2 |
| 9 | 1,4,6 | 10,15,16,19 | 2 |
| 10 | 1,3,4,6,8,9 | 16,19 | 2 |
| 11 | 2,3 | 19,21 | 2 |
| 12 | 2,3,7 | 21 | 2 |
| 13 | 1,7 | 14,15,17,19 | 2 |
| 14 | 1,13 | 15,16,18,19 | 3 |
| 15 | 4,9,14 | 16,19 | 3 |
| 16 | 6,8,9,10,15 | 19,20 | 3 |
| 17 | 7,13 | 20 | 3 |
| 18 | 5,14,15,16,17 | 19,21 | 3 |
| 19 | 5,8,9,10,11,13,15,16,18 | 20,F1-F4 | 4 |
| 20 | 16,17,19 | 21,F1-F4 | 4 |
| 21 | 2,3,7,11,12,18,20 | F1-F4 | 4 |
| F1 | 1-21 | — | FINAL |
| F2 | 1-21 | — | FINAL |
| F3 | 1-21 | — | FINAL |
| F4 | 1-21 | — | FINAL |

### Agent Dispatch Summary

| Wave | Parallel | Suggested Categories |
|---|---:|---|
| 1 | 7 | quick, visual-engineering, unspecified-high |
| 2 | 6 | visual-engineering, unspecified-high, deep |
| 3 | 5 | unspecified-high, deep, writing |
| 4 | 3 | unspecified-high, visual-engineering |
| FINAL | 4 | oracle, unspecified-high, deep |

---

## TODOs

- [x] 1. Lifecycle contract matrix + UI mapping baseline
  - **What to do**: Produce explicit mapping table (status transitions, HTTP status, error codes, frontend behavior/actions) and pin it as implementation reference.
  - **Must NOT do**: Change business rules yet.
  - **Recommended Agent Profile**: Category `unspecified-high`; Skills `frontend-ui-ux` (UI mapping clarity). Omit `playwright` (no browser execution needed).
  - **Parallelization**: YES, Wave 1. Blocks 8/9/10/14/15.
  - **References**:
    - `backend/internal/modules/instance/service.go:99-171` (start semantics + errors)
    - `backend/internal/modules/instance/service.go:173-229` (stop semantics + errors)
    - `backend/internal/modules/instance/repository.go:105-118` (`me` active-only behavior)
    - `frontend/src/components/challenge/ChallengeDetail.tsx:50-116` (current status/button behavior)
  - **Acceptance Criteria**:
    - [ ] Contract matrix doc committed in project docs/runbook section.
    - [ ] Every lifecycle code maps to explicit frontend UI reaction.
  - **QA Scenarios**:
    - Happy: run `go test -v ./internal/modules/instance -run TestInstancesConcurrentStartOnlyOneCreated` and verify referenced code path in matrix. Evidence: `.sisyphus/evidence/task-1-contract-baseline.txt`
    - Edge: verify matrix includes `INSTANCE_COOLDOWN_ACTIVE` with `retryAt` handling. Evidence: `.sisyphus/evidence/task-1-cooldown-contract.txt`

- [x] 2. Global dark shell + design tokens (security competition style)
  - **What to do**: Refactor global styling variables/classes for dark high-contrast style across layout/nav/page wrappers.
  - **Must NOT do**: Introduce full redesign/new route structure.
  - **Recommended Agent Profile**: `visual-engineering`; Skills `frontend-ui-ux`, `playwright`.
  - **Parallelization**: YES, Wave 1.
  - **References**:
    - `frontend/src/app/globals.css:1-285`
    - `frontend/src/app/layout.tsx:15-23`
    - `frontend/src/components/layout/AppNav.tsx:27-86`
  - **Acceptance Criteria**:
    - [ ] Global shell uses dark palette with minimum readable contrast.
    - [ ] Nav and page wrappers remain functional across authenticated/unauthenticated states.
  - **QA Scenarios**:
    - Happy (Playwright): login page + challenges page screenshot diff baseline exists. Evidence: `.sisyphus/evidence/task-2-dark-shell.png`
    - Edge: small viewport nav overflow not breaking primary links. Evidence: `.sisyphus/evidence/task-2-nav-mobile.png`

- [x] 3. Shared state/feedback primitives
  - **What to do**: Create reusable loading/error/empty/info presentation primitives and migrate repeated inline cards.
  - **Must NOT do**: Couple primitives to API fetching logic.
  - **Recommended Agent Profile**: `visual-engineering`; Skills `frontend-ui-ux`.
  - **Parallelization**: YES, Wave 1.
  - **References**:
    - `frontend/src/app/challenges/page.tsx:62-93`
    - `frontend/src/app/announcements/page.tsx:62-93`
    - `frontend/src/app/scoreboard/page.tsx:63-117`
    - `frontend/src/app/admin/users/page.tsx:98-194`
  - **Acceptance Criteria**:
    - [ ] At least 4 pages replace duplicated state cards with shared primitive.
  - **QA Scenarios**:
    - Happy: forced success path renders shared state wrappers. Evidence: `.sisyphus/evidence/task-3-shared-state-happy.png`
    - Edge: forced error path renders unified message style. Evidence: `.sisyphus/evidence/task-3-shared-state-error.png`

- [x] 4. HTTP error normalization + Retry-After parsing
  - **What to do**: Extend HTTP error typing to normalize headers/details for lifecycle handling (e.g., retry semantics).
  - **Must NOT do**: Break existing API client signatures.
  - **Recommended Agent Profile**: `unspecified-high`; Skills `frontend-ui-ux`.
  - **Parallelization**: YES, Wave 1. Depends on 1.
  - **References**:
    - `frontend/src/lib/http.ts:12-92`
    - `frontend/src/lib/types.ts:164-171`
    - `backend/internal/middleware/error_handler.go:11-54`
  - **Acceptance Criteria**:
    - [ ] HttpError includes structured retry metadata when provided.
    - [ ] Existing API client tests still pass.
  - **QA Scenarios**:
    - Happy: mocked 409 cooldown response parsed with retry detail. Evidence: `.sisyphus/evidence/task-4-retry-parse.txt`
    - Edge: non-JSON error body still yields safe fallback error object. Evidence: `.sisyphus/evidence/task-4-nonjson.txt`

- [x] 5. Reproducible runtime validation baseline
  - **What to do**: Ensure seed + compose path reliably provides dynamic challenge/runtime prerequisites for lifecycle testing.
  - **Must NOT do**: Add unrelated infrastructure services.
  - **Recommended Agent Profile**: `unspecified-high`; Skills `frontend-ui-ux` (workflow docs clarity).
  - **Parallelization**: YES, Wave 1.
  - **References**:
    - `docker-compose.yml:41-105`
    - `backend/cmd/seed/main.go:152-217`
    - `README.md:73-168`
    - `backend/README.seed.md:44-56`
  - **Acceptance Criteria**:
    - [ ] Runbook has deterministic start/seed/login/start/stop/cooldown steps.
    - [ ] At least one dynamic challenge has runtime config path documented.
  - **QA Scenarios**:
    - Happy (Bash): run compose + seed commands; health endpoint 200. Evidence: `.sisyphus/evidence/task-5-compose-health.txt`
    - Edge: missing docker socket scenario documented with recovery commands. Evidence: `.sisyphus/evidence/task-5-runtime-failure-doc.txt`

- [x] 6. Challenge detail orchestration scaffold refactor
  - **What to do**: Refactor challenge detail page orchestration for explicit load/refresh/action states and side-effect isolation.
  - **Must NOT do**: Change core backend API surface yet.
  - **Recommended Agent Profile**: `unspecified-high`; Skills `frontend-ui-ux`.
  - **Parallelization**: YES, Wave 1. Depends 1,3.
  - **References**:
    - `frontend/src/app/challenges/[id]/page.tsx:38-233`
    - `frontend/src/components/challenge/ChallengeDetail.tsx:33-167`
  - **Acceptance Criteria**:
    - [ ] No duplicated refresh logic blocks.
    - [ ] Poll timers are cleanly managed with clear conditions.
  - **QA Scenarios**:
    - Happy: pending submission triggers refresh polling and exits when resolved. Evidence: `.sisyphus/evidence/task-6-submission-poll.txt`
    - Edge: unmount/navigation clears timers (no duplicate calls). Evidence: `.sisyphus/evidence/task-6-timer-cleanup.txt`

- [x] 7. Shared admin primitives (table/actions/form shells)
  - **What to do**: Extract reusable visual wrappers for admin tables/action groups/forms without domain lock-in.
  - **Must NOT do**: Convert all admin logic into one mega component.
  - **Recommended Agent Profile**: `visual-engineering`; Skills `frontend-ui-ux`.
  - **Parallelization**: YES, Wave 1.
  - **References**:
    - `frontend/src/components/admin/ChallengeTable.tsx:24-87`
    - `frontend/src/components/admin/AnnouncementTable.tsx:38-98`
    - `frontend/src/components/admin/ChallengeEditor.tsx:79-211`
    - `frontend/src/components/admin/AnnouncementEditor.tsx:63-134`
  - **Acceptance Criteria**:
    - [ ] Admin challenge/announcement/users use shared visual wrappers.
  - **QA Scenarios**:
    - Happy: admin pages visually consistent across tables/forms. Evidence: `.sisyphus/evidence/task-7-admin-consistency.png`
    - Edge: empty table state still readable/consistent. Evidence: `.sisyphus/evidence/task-7-admin-empty.png`

- [x] 8. Challenge instance context guard
  - **What to do**: Prevent misleading actions when active instance belongs to another challenge; add explicit context mismatch messaging and action guidance.
  - **Must NOT do**: Silently stop/override another challenge instance.
  - **Recommended Agent Profile**: `unspecified-high`; Skills `frontend-ui-ux`.
  - **Parallelization**: Wave 2. Depends 1,6.
  - **References**:
    - `frontend/src/app/challenges/[id]/page.tsx:68-79`
    - `frontend/src/lib/api/instances.ts:11-38`
    - `backend/internal/modules/instance/repository.go:105-118`
  - **Acceptance Criteria**:
    - [ ] Cross-challenge active instance shows deterministic warning and safe actions.
  - **QA Scenarios**:
    - Happy: current challenge instance shows normal start/stop flow. Evidence: `.sisyphus/evidence/task-8-context-happy.png`
    - Edge: mismatch instance shows guard state and disables unsafe action. Evidence: `.sisyphus/evidence/task-8-context-mismatch.png`

- [x] 9. Cooldown reload persistence
  - **What to do**: Ensure cooldown visibility survives reload with backend-consistent source of truth (endpoint contract update if necessary).
  - **Must NOT do**: Fake cooldown solely from stale client memory.
  - **Recommended Agent Profile**: `deep`; Skills `frontend-ui-ux`.
  - **Parallelization**: Wave 2. Depends 1,4,6.
  - **References**:
    - `frontend/src/app/challenges/[id]/page.tsx:76-79`
    - `backend/internal/modules/instance/service.go:231-248`
    - `backend/internal/modules/instance/repository.go:46-54`
  - **Acceptance Criteria**:
    - [ ] Reload during cooldown still shows retry/counter state without first failing `start` call.
  - **QA Scenarios**:
    - Happy (curl+UI): stop instance, reload page, cooldown still visible. Evidence: `.sisyphus/evidence/task-9-cooldown-reload.png`
    - Edge: cooldown elapsed then reload enables start automatically. Evidence: `.sisyphus/evidence/task-9-cooldown-expired.png`

- [x] 10. Start/Stop UX hardening (recovery + diagnostics)
  - **What to do**: Improve action-state labels, error diagnostics, fallback refresh behavior, and disabled-state transparency.
  - **Must NOT do**: Hide backend errors from users.
  - **Recommended Agent Profile**: `visual-engineering`; Skills `frontend-ui-ux`, `playwright`.
  - **Parallelization**: Wave 2. Depends 1,3,4,6,8,9.
  - **References**:
    - `frontend/src/components/challenge/ChallengeDetail.tsx:90-116`
    - `frontend/src/app/challenges/[id]/page.tsx:132-192`
    - `backend/internal/modules/instance/service.go:119-129`
  - **Acceptance Criteria**:
    - [ ] Action buttons always reflect deterministic state.
    - [ ] Errors include actionable guidance (`retryAt`, auth, runtime failure).
  - **QA Scenarios**:
    - Happy: start→running→stop→cooldown flow with proper button transitions. Evidence: `.sisyphus/evidence/task-10-lifecycle-flow.mp4`
    - Edge: runtime start failure shows explicit error and recovery path. Evidence: `.sisyphus/evidence/task-10-runtime-failure.png`

- [x] 11. Auth/nav/home flow polish
  - **What to do**: Remove nav flicker/null render, improve home redirect UX, keep auth transitions explicit.
  - **Must NOT do**: change auth model.
  - **Recommended Agent Profile**: `visual-engineering`; Skills `frontend-ui-ux`.
  - **Parallelization**: Wave 2. Depends 2,3.
  - **References**:
    - `frontend/src/components/layout/AppNav.tsx:20-22`
    - `frontend/src/app/page.tsx:11-30`
    - `frontend/src/lib/use-auth.ts:45-58`
  - **Acceptance Criteria**:
    - [ ] No invisible header flash while auth state initializes.
  - **QA Scenarios**:
    - Happy: authenticated user lands smoothly on challenges with stable header. Evidence: `.sisyphus/evidence/task-11-authenticated-nav.png`
    - Edge: expired session redirects with clear message and no broken nav state. Evidence: `.sisyphus/evidence/task-11-expired-session.png`

- [x] 12. Challenge/Announcement visual harmonization
  - **What to do**: Replace leaky class semantics (`challenge-*` reused for announcements) with shared neutral card/meta patterns.
  - **Must NOT do**: duplicate per-page style forks.
  - **Recommended Agent Profile**: `visual-engineering`; Skills `frontend-ui-ux`.
  - **Parallelization**: Wave 2. Depends 2,3,7.
  - **References**:
    - `frontend/src/components/challenge/ChallengeList.tsx:16-31`
    - `frontend/src/components/announcement/AnnouncementList.tsx:30-43`
    - `frontend/src/components/announcement/AnnouncementDetail.tsx:25-32`
  - **Acceptance Criteria**:
    - [ ] Shared card/meta primitive used by both features.
  - **QA Scenarios**:
    - Happy: list pages show consistent metadata chips and card spacing. Evidence: `.sisyphus/evidence/task-12-lists.png`
    - Edge: empty state layout remains stable after class migration. Evidence: `.sisyphus/evidence/task-12-empty-state.png`

- [x] 13. Admin challenge runtime-config UX + contract wiring
  - **What to do**: Expose runtime fields in admin challenge editor and wire frontend types/API payloads to backend runtime contract.
  - **Must NOT do**: break existing static challenges.
  - **Recommended Agent Profile**: `unspecified-high`; Skills `frontend-ui-ux`.
  - **Parallelization**: Wave 2. Depends 1,7.
  - **References**:
    - `backend/internal/modules/challenge/dto.go:3-29`
    - `backend/internal/modules/challenge/service.go:64-66,139-150`
    - `frontend/src/components/admin/ChallengeEditor.tsx:25-49`
    - `frontend/src/lib/types.ts:31-42`
  - **Acceptance Criteria**:
    - [ ] Admin can view/edit runtimeImage/runtimeCommand/runtimeExposedPort.
    - [ ] Validation errors mapped clearly from backend to UI.
  - **QA Scenarios**:
    - Happy: create dynamic challenge with runtime fields and publish. Evidence: `.sisyphus/evidence/task-13-runtime-create.png`
    - Edge: invalid runtimeExposedPort rejected with surfaced message. Evidence: `.sisyphus/evidence/task-13-runtime-invalid.png`

- [x] 14. Backend lifecycle contract enhancements (targeted)
  - **What to do**: Add only required backend adjustments for frontend consistency (e.g., cooldown visibility endpoint semantics or richer conflict details).
  - **Must NOT do**: redesign lifecycle architecture.
  - **Recommended Agent Profile**: `deep`; Skills `frontend-ui-ux` (contract readability).
  - **Parallelization**: Wave 3. Depends 1,13.
  - **References**:
    - `backend/internal/modules/instance/service.go:115-127`
    - `backend/internal/modules/instance/repository.go:105-118`
    - `backend/internal/modules/instance/handler_test.go:281-323`
  - **Acceptance Criteria**:
    - [ ] Contract change documented with tests proving new semantics.
  - **QA Scenarios**:
    - Happy: contract returns expected lifecycle metadata for UI reload consistency. Evidence: `.sisyphus/evidence/task-14-contract-happy.json`
    - Edge: legacy clients still receive non-breaking core fields. Evidence: `.sisyphus/evidence/task-14-contract-compat.txt`

- [x] 15. Frontend API/types alignment after backend updates
  - **What to do**: Update types/API client usage to consume finalized lifecycle contract precisely.
  - **Must NOT do**: ad-hoc `any` casting.
  - **Recommended Agent Profile**: `unspecified-high`; Skills `frontend-ui-ux`.
  - **Parallelization**: Wave 3. Depends 4,9,14.
  - **References**:
    - `frontend/src/lib/types.ts:173-210`
    - `frontend/src/lib/api/instances.ts:11-38`
    - `frontend/src/lib/http.ts:12-25`
  - **Acceptance Criteria**:
    - [ ] Lifecycle-related types fully represent backend response/error contract.
  - **QA Scenarios**:
    - Happy: typed client consumes new fields without type assertion escapes. Evidence: `.sisyphus/evidence/task-15-typecheck.txt`
    - Edge: missing optional fields handled gracefully. Evidence: `.sisyphus/evidence/task-15-optional-fields.png`

- [x] 16. Page-level orchestration tests (challenge detail)
  - **What to do**: Add tests for challenge detail page orchestration: parallel load, start/stop fallback refresh, cooldown handling, polling conditions.
  - **Must NOT do**: only snapshot static component.
  - **Recommended Agent Profile**: `unspecified-high`; Skills `frontend-ui-ux`.
  - **Parallelization**: Wave 3. Depends 6,8,9,10,15.
  - **References**:
    - `frontend/src/app/challenges/[id]/page.tsx:38-233`
    - `frontend/src/__tests__/challenge-detail-submission-status.test.tsx:58-118`
    - `frontend/src/__tests__/instances-api.test.ts:18-95`
  - **Acceptance Criteria**:
    - [ ] New tests cover happy + conflict + reload + timer cleanup paths.
    - [ ] `pnpm -C frontend test` passes.
  - **QA Scenarios**:
    - Happy: orchestration test suite passes with expected assertions. Evidence: `.sisyphus/evidence/task-16-orchestration-tests.txt`
    - Edge: simulated 409 cooldown and runtime failure paths are covered. Evidence: `.sisyphus/evidence/task-16-error-coverage.txt`

- [x] 17. Admin flow + auth-guard tests
  - **What to do**: Add/extend tests for admin challenge runtime form, admin user actions, auth guard redirects.
  - **Must NOT do**: leave admin critical path untested.
  - **Recommended Agent Profile**: `unspecified-high`; Skills `frontend-ui-ux`.
  - **Parallelization**: Wave 3. Depends 7,13.
  - **References**:
    - `frontend/src/__tests__/admin-challenges-access.test.ts:23-48`
    - `frontend/src/__tests__/admin-users-api.test.ts:23-72`
    - `frontend/src/lib/use-auth.ts:39-77`
  - **Acceptance Criteria**:
    - [ ] Admin flow tests include runtime field and role guard behaviors.
  - **QA Scenarios**:
    - Happy: admin authorized flows pass tests. Evidence: `.sisyphus/evidence/task-17-admin-tests.txt`
    - Edge: non-admin role blocked with expected behavior. Evidence: `.sisyphus/evidence/task-17-nonadmin.txt`

- [x] 18. Docs/runbook synchronization (API + UI)
  - **What to do**: Update README/runbook/seed docs to include UI verification flow and align wording with final contract behavior.
  - **Must NOT do**: stale API-only instructions.
  - **Recommended Agent Profile**: `writing`; Skills `frontend-ui-ux`.
  - **Parallelization**: Wave 3. Depends 5,14,15,16,17.
  - **References**:
    - `README.md:73-168`
    - `README.md:170-224`
    - `backend/README.seed.md:48-56`
  - **Acceptance Criteria**:
    - [ ] Documentation includes reproducible UI lifecycle verification steps and expected outputs.
  - **QA Scenarios**:
    - Happy: follow docs from clean state to successful UI lifecycle demonstration. Evidence: `.sisyphus/evidence/task-18-docs-happy.txt`
    - Edge: troubleshooting section includes known failure signatures and recovery commands. Evidence: `.sisyphus/evidence/task-18-docs-troubleshoot.txt`

- [x] 19. E2E Playwright lifecycle journeys
  - **What to do**: Implement e2e flows for login→challenge→start/stop/cooldown and admin runtime-config journey.
  - **Must NOT do**: rely only on unit tests.
  - **Recommended Agent Profile**: `unspecified-high`; Skills `playwright`, `frontend-ui-ux`.
  - **Parallelization**: Wave 4. Depends 5,8,9,10,11,13,15,16,18.
  - **References**:
    - `frontend/src/app/challenges/[id]/page.tsx:132-223`
    - `frontend/src/components/challenge/ChallengeDetail.tsx:72-116`
    - `README.md:98-150`
  - **Acceptance Criteria**:
    - [ ] E2E suite validates lifecycle happy path + cooldown conflict + admin runtime form.
  - **QA Scenarios**:
    - Happy: UI start/stop flow completes with expected status updates. Evidence: `.sisyphus/evidence/task-19-e2e-happy.mp4`
    - Edge: immediate restart in cooldown produces expected rejection UI. Evidence: `.sisyphus/evidence/task-19-e2e-cooldown.mp4`

- [x] 20. Regression fixes from automated/e2e feedback
  - **What to do**: Address failures surfaced by tasks 16/17/19.
  - **Must NOT do**: introduce unrelated refactors.
  - **Recommended Agent Profile**: `quick`; Skills `frontend-ui-ux`.
  - **Parallelization**: Wave 4. Depends 16,17,19.
  - **References**:
    - Test outputs generated in `.sisyphus/evidence/`
  - **Acceptance Criteria**:
    - [ ] All failing tests/scenarios become green without scope creep.
  - **QA Scenarios**:
    - Happy: rerun previously failing suite now passes. Evidence: `.sisyphus/evidence/task-20-regression-pass.txt`
    - Edge: verify no new failures introduced in untouched modules. Evidence: `.sisyphus/evidence/task-20-no-new-regression.txt`

- [x] 21. Accessibility + responsive consistency sweep
  - **What to do**: Validate keyboard navigation, contrast/readability, and narrow-screen behavior for updated pages.
  - **Must NOT do**: add heavy a11y framework migration.
  - **Recommended Agent Profile**: `visual-engineering`; Skills `playwright`, `frontend-ui-ux`.
  - **Parallelization**: Wave 4. Depends 2,3,7,11,12,18,20.
  - **References**:
    - `frontend/src/app/globals.css:1-285`
    - `frontend/src/components/layout/AppNav.tsx:35-83`
    - `frontend/src/app/admin/users/page.tsx:142-191`
  - **Acceptance Criteria**:
    - [ ] Keyboard navigation reaches lifecycle actions and admin actions.
    - [ ] Dark theme contrast remains readable on key text/status components.
  - **QA Scenarios**:
    - Happy: keyboard-only flow for login→challenge start action succeeds. Evidence: `.sisyphus/evidence/task-21-keyboard-flow.mp4`
    - Edge: 360px viewport screenshots show no blocked critical action controls. Evidence: `.sisyphus/evidence/task-21-mobile-viewport.png`

---

## Final Verification Wave (MANDATORY)

- [x] F1. **Plan Compliance Audit** — `oracle`
  - Verify all Must Have present, Must NOT Have absent, evidence files exist.
  - Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  - Run lint/type/test and check anti-patterns (`as any`, empty catch, dead code, random naming).
  - Output: `Build/Lint/Tests + file issue counts + VERDICT`

- [x] F3. **Real QA Replay** — `unspecified-high` + `playwright`
  - Replay all QA scenarios, capture final evidence under `.sisyphus/evidence/final-qa/`.
  - Output: `Scenarios [N/N] | Integration [N/N] | Edge Cases [N] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  - Compare delivered diff against plan boundaries and contamination risk.
  - Output: `Tasks compliant [N/N] | Contamination [CLEAN/N] | VERDICT`

---

## Commit Strategy

| Batch | Message Pattern | Verification Before Commit |
|---|---|---|
| Wave 1 | `refactor(ui): establish dark shell and lifecycle contract baseline` | `pnpm lint && pnpm type-check` |
| Wave 2 | `feat(instance-ui): harden lifecycle UX and context correctness` | `pnpm -C frontend test` |
| Wave 3 | `test/docs: add orchestration coverage and sync runbooks` | `pnpm test && go test ./...` |
| Wave 4 | `fix(stability): resolve regressions and polish accessibility` | full test + e2e replay |

---

## Success Criteria

### Verification Commands
```bash
pnpm lint
pnpm type-check
pnpm test
go test ./...    # from backend/
```

### Final Checklist
- [ ] All lifecycle UX states are explicit and reproducible.
- [ ] Cooldown/start/stop semantics are contract-consistent across FE/BE/docs.
- [ ] Challenge-context mismatch case is safely handled.
- [ ] Admin runtime config path supports reproducible lifecycle testing.
- [ ] Automated tests and agent QA evidence complete.
- [ ] Documentation accurately reflects shipped behavior and recovery paths.
