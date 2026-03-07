
## 2026-02-17 Wave 1 Task 1 lifecycle contract decisions

- Established `docs/instance-lifecycle-contract.md` as the canonical baseline for instance lifecycle implementation references in this plan wave.
- Kept contract strictly implementation-accurate to current code, including runtime-enabled router behavior and active-only `/instances/me` semantics.
- Captured endpoint contracts for `POST /instances/start`, `POST /instances/stop`, and `GET /instances/me` with current request and response shapes.
- Used matrix-first format for lifecycle transitions, backend error mapping, and frontend UI reaction mapping to unblock downstream contract-alignment tasks.
- Documented ambiguity section explicitly instead of proposing behavior changes in this task, because scope forbids business-logic modifications.

## 2026-02-17 Wave 1 Task 2 dark shell decisions

- Centralized dark-theme primitives as CSS custom properties in `:root` and refit existing global classes to consume those tokens.
- Kept component structure unchanged (no route/layout rewrite); applied medium-scope refactor exclusively in `globals.css` to preserve compatibility.
- Retained class API (`page`, `page-content`, `card`, `button`, `error-text`, `info-text`, `empty-text`) and upgraded visual semantics only.
- Captured evidence with Playwright-generated desktop/mobile screenshots at required paths, with desktop proof combining login and challenges views for single-file requirement compliance.

## 2026-02-17 Wave 1 Task 2 dark shell decisions

- Applied dark visual system by introducing root-level color tokens and reusing all existing utility classes.
- Kept `AppNav` behavior unchanged (no auth/route logic edits); styling handled through existing nav class selectors.
- Added focus ring and hover/active styles for high-contrast interaction cues without introducing new dependencies.
- Captured mandatory evidence as CLI Playwright screenshots under `.sisyphus/evidence/`.

## 2026-02-17 Wave 1 Task 3 shared state feedback decisions

- Added a shared presentational primitive module at `frontend/src/components/ui/StateFeedback.tsx` with variant-based text/card rendering and no data-fetch coupling.
- Migrated duplicated inline state cards in four target pages (`challenges`, `announcements`, `scoreboard`, `admin/users`) while preserving existing message copy and state gating behavior.
- Kept existing `card` and status text class semantics intact so the new primitives inherit current dark-shell styling without CSS/API changes.

## 2026-02-17 Wave 1 Task 4 retry metadata normalization decisions

- Extended `HttpError` additively with `retry?: HttpRetryMetadata` to preserve API/client compatibility.
- Introduced `HttpRetryMetadata`/`HttpRetrySource` types in `types.ts` for explicit, typed retry semantics.
- Added dedicated frontend tests (`http-error-normalization.test.ts`) covering details-based retry, header seconds/date parsing, and non-JSON fallback.

## 2026-02-17 Wave 1 Task 5 runtime validation baseline decisions

- Expanded root README runbook into a command-by-command deterministic flow covering reset, seed, login, start, stop, cooldown, and dynamic runtime validation.
- Added backend seed README guidance to cross-reference deterministic compose+seed commands and validation expectations.
- Produced required evidence artifacts for compose/health/seed and runtime-failure troubleshooting under `.sisyphus/evidence/task-5-*.txt`.

## 2026-02-17 Wave 1 Task 6 challenge detail orchestration scaffold refactor decisions

- Introduced page-local orchestration helpers (`loadChallengeContext`, `refreshChallengeContext`, `reconcileInstanceState`) in `challenges/[id]/page.tsx` to isolate fetch/apply/reconcile side effects while keeping backend contracts unchanged.
- Replaced dual refresh booleans with explicit refresh state (`idle|manual|background`) so manual refresh UX remains deterministic while background polling stays transparent.
- Replaced ad-hoc interval effects with a shared `usePollingInterval` helper and explicit enable predicates for submission pending polling, transitional instance polling, and cooldown ticker cleanup.
- Kept `ChallengeDetail` prop contract unchanged by deriving `refreshing` from orchestrator state in the page container.

## 2026-02-17 Wave 1 Task 7 shared admin primitives decisions

- Introduced `frontend/src/components/admin/AdminPrimitives.tsx` as a visual-only wrapper module containing `AdminSection`, `AdminDataTable`, `AdminEditorShell`, and `AdminActionGroup`.
- Kept primitives children-driven and domain-agnostic (no API calls, no entity-specific props), with optional state text props only for table loading/empty rendering.
- Preserved existing styling contracts by reusing current utility classes (`card`, `table-wrap`, `inline-actions`, `row-actions`, `stack-sm`, `empty-text`, `error-text`).
- Migrated challenge/announcement editors + tables and admin list sections (including users table/actions) to enforce consistent shell structure without changing business behavior.

## 2026-02-17 Wave 1 Task 7 shared admin primitives decisions

- Introduced `frontend/src/components/admin/AdminPrimitives.tsx` as a visual-only primitive layer for admin shells, action groups, and table/empty/loading wrappers.
- Migrated admin challenge table/editor and admin challenge page sections to shared primitives first, while keeping behavior and endpoint wiring unchanged.
- Confirmed admin announcement/users pages already consume the same primitive contracts, achieving cross-admin visual consistency without mega-component refactor.

## 2026-02-17 Wave 1 Task 8 challenge instance context guard decisions

- Added an explicit derived flag (`instanceChallengeMismatch`) in `challenges/[id]/page.tsx` based on `instance.challengeId !== route challengeId` and passed it into `ChallengeDetail` as additive frontend-only state.
- Chose deterministic, user-actionable mismatch copy that names the conflicting challenge id and points to `/challenges/{id}` for safe management.
- Enforced mismatch safety in both action handlers (`handleStartInstance`, `handleStopInstance`) via early-return guards that set an error message instead of calling instance APIs.
- Kept backend contracts unchanged and preserved normal behavior for matched challenge ids by only gating mismatch branch paths.

## 2026-02-17 Wave 1 Task 7 shared admin primitives decisions

- Introduced  as a visual-only wrapper module containing , , , and .
- Kept primitives children-driven and domain-agnostic (no API calls, no entity-specific props), with optional state text props only for table loading/empty rendering.
- Preserved existing styling contracts by reusing current utility classes (, , , , , , ).
- Migrated challenge/announcement editors + tables and admin list sections (including users table/actions) to enforce consistent shell structure without changing business behavior.

## 2026-02-17 Wave 1 Task 7 shared admin primitives decisions (final)

- Standardized admin visual wrappers through `AdminPrimitives` and reused existing utility class semantics to preserve dark-shell compatibility.
- Kept wrappers purely presentational and composable (children-based), avoiding domain/API centralization.
- Unified challenge/announcement/users management shells by adopting shared section, table, editor, and action-group wrappers with no behavior changes.

## 2026-02-17 Wave 1 Task 7 shared admin primitives decisions (execution)

- Added new `frontend/src/components/admin/AdminPrimitives.tsx` and wired it into challenge/announcement table+editor components and admin page section shells.
- Kept behavior copy/actions unchanged by moving only presentational wrappers, not state/effect/API logic.
- Preserved SSR test compatibility in JSX-only pages/components by retaining explicit React runtime symbol where required by current Vitest setup.

## 2026-02-17 Wave 1 Task 5 reproducible runtime baseline decisions

- Updated seed baseline (`backend/cmd/seed/main.go`) so `Log Trail` includes runtime config (`runtimeImage`, `runtimeCommand`, `runtimeExposedPort`) out-of-the-box.
- Standardized runbook commands around deterministic credentials (`admin@ctf.local` / `player@ctf.local`) and explicit expected response contracts for login/start/stop/cooldown.
- Added troubleshooting and recovery steps focused on docker socket/runtime-controller availability (`docker compose exec backend|worker ... /var/run/docker.sock`, `docker version`, rebuild/restart runtime services).
