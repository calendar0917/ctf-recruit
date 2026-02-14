# Error Handling

> How errors are handled in this project.

---

## Overview

Use **structured, typed errors** and a unified HTTP error response format.

Principles:
- Internal error details stay in logs.
- Client receives stable `code` + human-readable `message`.
- All handlers return errors through centralized middleware.

---

## Error Types

Define app-level errors with fields:
- `code` (stable machine code, e.g. `AUTH_INVALID_CREDENTIALS`)
- `message` (safe for clients)
- `status` (HTTP status)
- `cause` (internal wrapped error, optional)

Categories:
- Validation errors (400)
- Unauthorized/forbidden (401/403)
- Not found (404)
- Conflict (409)
- Internal (500)

---

## Error Handling Patterns

- Handler validates input -> calls service -> returns mapped response.
- Service returns typed domain/app errors.
- Middleware maps unknown errors to 500.
- Always wrap external failures (`fmt.Errorf("...: %w", err)`).

Do not:
- Leak raw DB or stack errors to API responses.
- Build ad-hoc JSON error formats per endpoint.

---

## API Error Responses

Standard shape:

```json
{
  "error": {
    "code": "AUTH_INVALID_CREDENTIALS",
    "message": "Invalid email or password"
  },
  "requestId": "req_..."
}
```

For validation errors, optionally include field-level details:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request payload",
    "details": [
      {"field": "email", "reason": "invalid_format"}
    ]
  }
}
```

---

## Common Mistakes

- Returning different error schema across modules.
- Converting all errors to 500 without distinguishing user errors.
- Logging same error repeatedly at multiple layers.
