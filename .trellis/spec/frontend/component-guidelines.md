# Component Guidelines

> How components are built in this project.

---

## Overview

Use composable React function components with explicit props types.
Prefer server components by default in Next.js App Router, and switch to client components only when interactivity is required.

---

## Component Structure

Recommended structure:
1. Imports
2. Type definitions (`type Props`)
3. Component implementation
4. Small local helpers (if needed)

Rules:
- One exported primary component per file.
- Keep presentation-only components free of data fetching side effects.
- Feature-level container components can wire data hooks + UI components.

---

## Props Conventions

- Define `type Props = { ... }` near component.
- Avoid passing large opaque objects when only few fields are used.
- Use callback names by intent (`onSubmitFlag`, `onRetry`) instead of generic `onClick` pass-through.
- Prefer explicit optional props over broad union that hides states.

---

## Styling Patterns

- Tailwind utility classes as default.
- Shared UI patterns should be extracted into `components/ui`.
- Use `cn()` helper for conditional classes.
- Avoid inline style unless truly dynamic and not representable in class utilities.

---

## Accessibility

- Interactive elements must use semantic tags (`button`, `a`, `input`).
- Forms must have labels and validation messages linked accessibly.
- Dialogs/drawers should support keyboard close and focus management.
- Avoid click-only interaction without keyboard equivalent.

---

## Common Mistakes

- Adding `"use client"` to large page trees unnecessarily.
- Mixing data fetching, mutation, and rendering logic in one giant component.
- Using non-semantic clickable `div`.
