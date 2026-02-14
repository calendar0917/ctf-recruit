# Quality Guidelines

> Code quality standards for frontend development.

---

## Overview

Frontend quality baseline:
- Lint passes
- Typecheck passes
- Build succeeds
- Core user flows are manually verified (login, challenge list, submit flag, scoreboard)

---

## Forbidden Patterns

- Large page components with mixed fetch/mutation/render logic.
- Unhandled loading/error states for network requests.
- Using `any` to bypass type checks.
- Directly embedding secrets or environment-dependent constants in UI code.

---

## Required Patterns

- Explicit loading/error/empty states for data-driven UI.
- Feature-based structure for business code.
- Accessible semantics for interactive controls and forms.
- Reusable API client layer instead of scattered `fetch` calls.

---

## Testing Requirements

- Unit tests for non-trivial pure logic.
- Component tests for critical interactive components.
- At least one e2e smoke path for core MVP flow.

MVP minimum manual checks before merge:
- Login/logout works
- Challenge list loads
- Submission feedback displayed correctly
- Scoreboard updates visible

---

## Code Review Checklist

- Route/component boundaries are clean.
- Query key and cache invalidation logic are correct.
- UI states (loading/error/empty) are covered.
- No inaccessible custom controls.
- No unnecessary client component expansion.
