# State Management

> How state is managed in this project.

---

## Overview

State strategy:
- **Server state**: TanStack Query
- **UI local state**: React `useState` / `useReducer`
- **Cross-page client state**: lightweight store only when necessary (e.g. Zustand)
- **URL state**: route params/search params for filter/sort/pagination

---

## State Categories

- Local state: input values, modal open/close, transient UI selections.
- Global client state: auth session snapshot, global UI preferences.
- Server state: challenge list, submissions, scoreboard, announcements.
- URL state: challenge filters, ranking page index.

---

## When to Use Global State

Use global state only if state is:
1. needed by multiple distant branches,
2. not naturally represented as server state, and
3. not better kept in URL.

Avoid promoting local component state prematurely.

---

## Server State

- Query keys must encode all variables affecting data.
- Prefer stale-while-revalidate defaults over manual refetch loops.
- Mutations should update cache intentionally (invalidate or optimistic update where safe).

---

## Common Mistakes

- Copying query data into local state without reason.
- Storing URL-derivable filters in global store.
- Using global store for one-page-only state.
