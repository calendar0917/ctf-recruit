# Hook Guidelines

> How hooks are used in this project.

---

## Overview

Use hooks to encapsulate reusable stateful logic and server interaction patterns.
All custom hooks must start with `use` and be located either in feature folders or `src/hooks` when broadly shared.

---

## Custom Hook Patterns

- Feature-scoped hook example path: `src/features/challenge/hooks/useChallengeList.ts`
- Shared hook example path: `src/hooks/useDebounce.ts`

Rules:
- Hooks should expose stable, minimal API (`data`, `isLoading`, `error`, `actions`).
- Avoid returning massive objects with unrelated state.
- Keep hooks pure from UI concerns (no toast rendering inside hook).

---

## Data Fetching

- Use TanStack Query for server state.
- Query keys should be centralized per feature (`challengeKeys.list(filters)`).
- Mutations should invalidate/update related queries explicitly.
- Authentication token handling should be centralized in API client middleware/interceptor.

---

## Naming Conventions

- `useXxxQuery` for read hooks that wrap queries.
- `useXxxMutation` for write operations.
- `useXxxState` for local complex state abstractions.

---

## Common Mistakes

- Calling hooks conditionally.
- Duplicating identical query logic across pages instead of shared hook.
- Fetching server state with `useEffect + fetch` when query abstraction exists.
