# Logging Guidelines

> How logging is done in this project.

---

## Overview

- Use **structured JSON logging**.
- Include request correlation fields for API logs.
- Keep logs actionable, not verbose-by-default.

Recommended baseline fields:
- `timestamp`
- `level`
- `message`
- `service` (`ctf-api`, `judge-worker`)
- `requestId` (for HTTP logs)
- `userId` (when authenticated)
- `module`

---

## Log Levels

- `DEBUG`: local/dev diagnostics only.
- `INFO`: important business events (login success, submission accepted).
- `WARN`: recoverable anomalies (rate-limit hit, retry event).
- `ERROR`: failures that affect behavior (DB unavailable, judge execution failure).

Rules:
- Do not log expected validation failures as `ERROR`.
- Do not use `DEBUG` in hot paths for production unless temporary.

---

## Structured Logging

- Always log as key-value pairs.
- One event per log line.
- Attach `requestId` from middleware context in every handler log.
- For background jobs, include `jobId` / `submissionId`.

---

## What to Log

- Auth events (login success/failure, token refresh/revoke)
- Submission lifecycle (received, judged, scored)
- Privileged admin operations (create/update challenge)
- External dependency failures (DB/Redis/container runtime)

---

## What NOT to Log

- Passwords, JWT secrets, raw tokens
- Full flag values
- PII beyond what is required for troubleshooting
- Large request/response bodies by default
