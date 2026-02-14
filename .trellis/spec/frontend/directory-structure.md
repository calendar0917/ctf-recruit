# Directory Structure

> How frontend code is organized in this project.

---

## Overview

Frontend uses **Next.js + TypeScript** with App Router.
Organize code by feature/domain while keeping reusable UI primitives in shared directories.

---

## Directory Layout

```text
frontend/
├── src/
│   ├── app/
│   │   ├── (public)/
│   │   │   ├── page.tsx
│   │   │   └── login/page.tsx
│   │   ├── (dashboard)/
│   │   │   ├── challenges/page.tsx
│   │   │   ├── scoreboard/page.tsx
│   │   │   └── admin/challenges/page.tsx
│   │   ├── api/                    # optional BFF route handlers
│   │   ├── layout.tsx
│   │   └── globals.css
│   ├── components/
│   │   ├── ui/                     # shadcn/ui primitives
│   │   └── shared/                 # shared business components
│   ├── features/
│   │   ├── auth/
│   │   ├── challenge/
│   │   ├── submission/
│   │   └── scoreboard/
│   ├── hooks/
│   ├── lib/                        # api client, utils, constants
│   ├── stores/                     # global client state (if needed)
│   └── types/
├── public/
├── package.json
└── tsconfig.json
```

---

## Module Organization

- Route files in `app/` should stay thin and compose feature components.
- Business logic and data hooks belong in `features/*`.
- Reusable visual primitives belong in `components/ui`.
- Avoid large cross-feature utility buckets; keep helper near feature unless truly shared.

---

## Naming Conventions

- Components: `PascalCase.tsx`
- Hooks: `useXxx.ts`
- Utilities/types/constants: `kebab-case.ts` or feature-local naming consistency
- Route segments: lowercase and semantic (`scoreboard`, `admin/challenges`)

---

## Examples

Project is currently in bootstrap stage with no production frontend code yet.
Use this document as baseline structure for first implementation.
