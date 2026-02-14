# Type Safety

> Type safety patterns in this project.

---

## Overview

Use strict TypeScript and runtime validation for external boundaries.

- Compile with strict mode enabled.
- API payloads should have both TS types and runtime validation schemas.
- Share API contract types through dedicated modules.

---

## Type Organization

- Feature-local types: `src/features/<feature>/types.ts`
- Shared domain/API types: `src/types/`
- Avoid a single giant `types.ts` file for entire app.

Rules:
- Prefer `type` for object aliases and unions.
- Keep DTO types close to API client functions.

---

## Validation

- Use Zod for request/response runtime validation at boundaries.
- Parse/validate untrusted data (API responses, form data) before use.
- Derive TS types from schemas when practical (`z.infer<typeof schema>`).

---

## Common Patterns

- Discriminated unions for async/result states.
- `satisfies` operator for constant config objects.
- Narrow types with type guards instead of broad assertions.

---

## Forbidden Patterns

- `any` in feature code.
- Double assertion (`as unknown as`) unless absolutely unavoidable and documented.
- Ignoring nullable/undefined from API fields without checks.
