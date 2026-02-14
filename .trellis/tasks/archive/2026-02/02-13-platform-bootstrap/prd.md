# Platform Bootstrap

## Goal
Initialize a runnable fullstack skeleton for the CTF recruitment platform with:
- Go + Fiber backend
- Next.js + TypeScript frontend
- Docker Compose local environment (frontend, backend, postgres, redis)

This task establishes project structure and baseline health checks only. Feature modules (auth/challenges/submission) will be implemented in later tasks.

## Requirements
- Create backend project skeleton following `.trellis/spec/backend/*` conventions.
- Create frontend project skeleton following `.trellis/spec/frontend/*` conventions.
- Provide docker-compose for local multi-service startup.
- Add environment variable templates for backend and frontend.
- Add minimal API health endpoint and frontend landing page to verify integration.
- Ensure backend and frontend can run independently and via Docker Compose.

## Acceptance Criteria
- [ ] Repository contains `backend/` and `frontend/` directories with initial app scaffolds.
- [ ] Backend starts successfully and exposes health endpoint (e.g., `/api/v1/health`).
- [ ] Frontend starts successfully and renders baseline page.
- [ ] `docker-compose up` can start required services (frontend, backend, postgres, redis).
- [ ] Env template files exist with required keys documented.
- [ ] Lint/type/build/test baseline commands run successfully for new scaffolds.

## Technical Notes
- Backend stack: Go + Fiber, PostgreSQL, Redis placeholders for upcoming modules.
- Frontend stack: Next.js App Router + TypeScript.
- Keep bootstrap implementation minimal and non-opinionated beyond established project specs.
- Do not implement auth/challenge/submission business logic in this task.
