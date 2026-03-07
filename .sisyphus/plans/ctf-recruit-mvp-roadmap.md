# CTF Recruit MVP Roadmap

## TL;DR

> **Quick Summary**: Deliver a stable CTF 招新平台 MVP (管理员 + 选手) using existing backend/ frontend foundations, closing missing flows (公告、报名表、用户管理、提交状态) with TDD and Docker-first deployment.
>
> **Deliverables**:
> - 完整的管理员与选手流程（公告、题目、提交、排行榜、报名、管理后台）
> - 新增报名表模块（后端 + 前端）
> - 提交状态/历史可视化 + 基础反作弊/限流
> - CI + 覆盖率脚本 + 稳定可复现的 Docker 运行链路
>
> **Estimated Effort**: Medium
> **Parallel Execution**: YES — 3 waves
> **Critical Path**: 测试/CI基线 → 报名模块 → 提交状态与记分验证 → 管理后台完善

---

## Context

### Original Request
- 你是单人团队，需要完成 CTF 招新平台开发，先做稳健 MVP，再继续推进。

### Interview Summary
**Key Discussions**:
- 角色：管理员 + 选手
- 功能：注册/登录、招新公告与报名、题目列表与动态 flag 校验、排行榜、管理后台（全都要）
- 赛制：Jeopardy only
- 动态 flag：MVP 先统一 flag
- 部署：本地 + Docker
- 测试策略：TDD
- 容器实例策略：每用户独立实例（per-user）
- 容器生命周期：用户可手动启动/关闭，启动后严格 1 小时自动关闭
- 冷却策略：关闭/到期后冷却 1 分钟
- 资源限制：每实例固定 0.5 CPU + 512MB

**Research Findings**:
- Backend 已具备 auth/challenge/submission/scoreboard/announcement/judge 基础模块。
- Frontend 已具备 login/register/challenges/scoreboard/admin-challenges 页面与组件。
- 缺口：公告 UI、报名表模块、管理员用户管理、提交状态/历史 UX。
- 测试基础存在但 CI/覆盖率缺失；根脚本存在前端 `type-check` 名称不一致问题。

### Metis Review
**Identified Gaps (addressed)**:
- 明确 MVP 不做团队、动态 flag、WebSocket、文件上传等高风险扩展。
- 增加基础限流、审计日志、评分一致性与幂等性约束。

---

## Work Objectives

### Core Objective
在现有代码基础上，以 TDD 方式补齐 CTF 招新平台 MVP 的缺失能力，并确保评分/权限/提交流程稳定可靠，可用 Docker 一键启动。

### Concrete Deliverables
- 招新公告与报名流程（选手提交 + 管理员审核/查看）
- 提交状态/历史列表 + 评分一致性验证
- 管理后台：公告、题目、用户管理
- CI + 覆盖率脚本 + root 脚本修复

### Definition of Done
- [x] `pnpm test`、`pnpm lint`、`pnpm type-check` 全部通过
- [x] 选手可完成“注册→登录→查看公告→报名→做题→提交→排行榜更新”的闭环
- [x] 管理员可完成“公告/题目管理→用户管理→查看报名记录”的闭环
- [x] Docker compose 可本地启动前后端与数据库

### Must Have
- RBAC 严格生效（管理员功能必须后端校验）
- 提交评分幂等与一致性（防止重复得分）
- 明确的提交状态（正确/错误/待判）与用户反馈
- 报名表包含最小字段集（默认：姓名、学校、年级、方向、联系方式、个人简介）
- 容器实例生命周期（start/stop/auto-expire）可用，且严格服务端校验
- 单用户并发实例上限 = 1（强约束）

### Must NOT Have (Guardrails)
- 不实现团队赛制、动态 flag、WebSocket 实时推送
- 不实现文件上传（附件/提交包）
- 不实现高级反作弊与多维风控（只做基础限流）
- 不在本期扩展到 Kubernetes/多节点编排（MVP 先 Docker Engine）
- 不实现团队实例共享与多实例并发（先单用户单实例）

---

## Container Lifecycle Extension (NEW)

### Core Behavior Contract
- `POST /instances/start`：
  - 若用户已有运行中实例 → 返回冲突（并发上限）
  - 若处于冷却期（1 分钟）→ 返回 `retryAt`
  - 否则创建实例并返回 `instanceId/status/expiresAt/accessInfo`
- `POST /instances/stop`：
  - 仅允许停止“自己的实例”
  - 停止后写入 `cooldownUntil = now + 1m`
- 自动回收：
  - 严格 `startedAt + 1h` 到期即停机并清理资源
  - 到期语义是“绝对 TTL”，不是空闲 TTL

### Safety Guardrails
- 服务器端原子状态机（避免并发 start/stop 竞争）
- Docker 运行参数固定：`--cpus=0.5 --memory=512m`（并设置最小安全参数）
- 允许镜像白名单（按 challenge 配置，不允许任意镜像名）
- 清理保障：stop/expiry/failure 都必须 remove 容器与临时资源

### Defaults Applied
- 生命周期状态集合：`starting / running / stopping / stopped / expired / failed / cooldown`
- TTL 基准时间：以“实例创建成功并进入 running”的时间戳计时
- 用户体验：挑战详情页默认不展示全局榜单（榜单保留在独立 `/scoreboard` 页面）

---

## Verification Strategy (MANDATORY)

> **UNIVERSAL RULE: ZERO HUMAN INTERVENTION**
>
> ALL tasks in this plan MUST be verifiable WITHOUT any human action.
>
> **FORBIDDEN** — acceptance criteria that require human action.

### Test Decision
- **Infrastructure exists**: YES (Vitest + Go tests)
- **Automated tests**: **TDD**
- **Framework**: Vitest (frontend), Go test (backend)

### TDD Structure
Each feature task follows RED → GREEN → REFACTOR. Backend tests run via `go test ./...`, frontend via `vitest run`.

### Agent-Executed QA Scenarios
Every task includes QA scenarios with tools:
- UI: Playwright
- API: Bash (curl)
- CLI/Infra: Bash

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately):
├── Task 1: 测试/CI基线 + 脚本一致性

Wave 2 (After Wave 1):
├── Task 2: 公告（前端 + 管理端）
├── Task 3: 招新报名模块（后端 + 前端）

Wave 3 (After Wave 2):
├── Task 4: 提交状态/历史 + 评分一致性
├── Task 5: 管理员用户管理 + 审计/限流

Wave 4 (After Wave 3):
└── Task 6: Docker 启动链路与文档完善
```

Critical Path: Task 1 → Task 3 → Task 4 → Task 5

---

## TODOs

> Implementation + Test = ONE Task. Never separate.

- [x] 1. 测试/CI 基线 + 脚本一致性修复

  **What to do**:
  - 修复 root `type-check` 与前端脚本名不一致
  - 增加覆盖率脚本（前端/后端）
  - 增加 CI workflow：运行 lint/type-check/test
  - 引入前端 UI 测试依赖（例如 Testing Library）以支持 TDD

  **Must NOT do**:
  - 不引入复杂 CI（只需基础 pipeline）

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 基础工程配置与脚本修复
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `playwright`: 本任务仅基础脚本与 CI

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 1 (独立任务)
  - Blocks: Task 2, 3, 4, 5

  **References**:
  - `package.json` (root scripts)
  - `frontend/package.json` (test/lint/typecheck scripts)
  - `backend/Makefile` (go test/lint)
  - `frontend/vitest.config.ts` (tests config)

  **Acceptance Criteria**:
  - [x] `pnpm lint` → PASS
  - [x] `pnpm type-check` → PASS
  - [x] `pnpm test` → PASS
  - [x] `pnpm -C frontend test:coverage` → PASS (新增)
  - [x] `go test -cover ./...` → PASS (新增)

  **Agent-Executed QA Scenarios**:
  - Scenario: CI-equivalent local run
    - Tool: Bash
    - Steps:
      1. `pnpm lint`
      2. `pnpm type-check`
      3. `pnpm test`
    - Expected Result: All commands exit 0
    - Evidence: terminal output logs saved

- [x] 2. 公告功能补齐（选手阅读 + 管理员管理）

  **What to do**:
  - 前端新增公告列表/详情页
  - 管理员公告 CRUD 页面与组件
  - API 客户端封装与类型定义
  - 导航中加入公告入口

  **Must NOT do**:
  - 不支持富文本/HTML（避免 XSS）；默认纯文本或安全 Markdown

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: UI 页面与交互
  - **Skills**: [`frontend-ui-ux`]

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 2 (with Task 3)
  - Blocked By: Task 1

  **References**:
  - Backend announcement module pattern: `backend/internal/modules/announcement/*`
  - Router wiring: `backend/internal/router/router.go`
  - Frontend nav: `frontend/src/components/layout/AppNav.tsx`
  - Frontend admin challenge UI pattern: `frontend/src/components/admin/*`

  **Acceptance Criteria (TDD)**:
  - [x] Frontend test for announcements list renders items
  - [x] Frontend test for admin create/update flows
  - [x] API calls use auth token and respect publish status

  **Agent-Executed QA Scenarios**:
  - Scenario: Player views announcements
    - Tool: Playwright
    - Steps:
      1. Login as player
      2. Navigate to `/announcements`
      3. Assert list contains at least 1 published item
      4. Click first item → detail page
      5. Screenshot `.sisyphus/evidence/task-2-announcements-player.png`
    - Expected Result: Published announcements visible
  - Scenario: Admin creates announcement
    - Tool: Playwright
    - Steps:
      1. Login as admin
      2. Navigate to `/admin/announcements`
      3. Fill title/body → Publish
      4. Assert list shows new item
      5. Screenshot `.sisyphus/evidence/task-2-announcements-admin.png`

- [x] 3. 招新报名模块（后端 + 前端）

  **What to do**:
  - 新建 recruitment 模块：model/dto/repo/service/handler
  - 增加迁移（recruitment_submissions 表）
  - API：
    - `POST /recruitments` (player)
    - `GET /recruitments` (admin list)
    - `GET /recruitments/:id` (admin detail)
  - 前端报名页面与管理员查看页面
  - 默认字段集：姓名、学校、年级、方向、联系方式、个人简介（可后续扩展）

  **Must NOT do**:
  - 不实现复杂审批流或多阶段筛选

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: 后端模块 + 前端页面 + 迁移
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 2 (with Task 2)
  - Blocked By: Task 1

  **References**:
  - Module pattern: `backend/internal/modules/announcement/*`, `backend/internal/modules/challenge/*`
  - Migration pattern: `backend/migrations/*.sql`
  - Router wiring: `backend/internal/router/router.go`
  - Frontend page pattern: `frontend/src/app/challenges/page.tsx`

  **Acceptance Criteria (TDD)**:
  - [x] Backend tests for POST/GET recruitment endpoints
  - [x] Frontend test validates required fields & submission success
  - [x] Admin-only list/detail endpoints return 403 for non-admin

  **Agent-Executed QA Scenarios**:
  - Scenario: Player submits recruitment form
    - Tool: Playwright
    - Steps:
      1. Login as player
      2. Navigate to `/recruitment`
      3. Fill required fields
      4. Submit → assert success message
      5. Screenshot `.sisyphus/evidence/task-3-recruitment-submit.png`
  - Scenario: Admin views recruitment submissions
    - Tool: Playwright
    - Steps:
      1. Login as admin
      2. Navigate to `/admin/recruitments`
      3. Assert submitted entry appears
      4. Screenshot `.sisyphus/evidence/task-3-recruitment-admin.png`

- [x] 4. 提交状态/历史 + 评分一致性

  **What to do**:
  - 新增 submissions 查询 API：`/submissions/me`、`/submissions/challenge/:id`
  - 前端挑战详情页展示当前提交状态与历史
  - 支持 pending 状态轮询刷新
  - 评分逻辑幂等，确保重复提交不重复得分
  - flag 规则默认：trim 空白，大小写敏感

  **Must NOT do**:
  - 不实现实时推送（WebSocket）

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: 后端逻辑 + 前端交互 + 评分一致性
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 3 (with Task 5)
  - Blocked By: Task 2, Task 3

  **References**:
  - Submission module: `backend/internal/modules/submission/*`
  - Scoreboard module: `backend/internal/modules/scoreboard/*`
  - Judge queue: `backend/internal/modules/judge/*`
  - Frontend submission UI: `frontend/src/components/submission/SubmissionForm.tsx`

  **Acceptance Criteria (TDD)**:
  - [x] Backend test: correct submission updates scoreboard once
  - [x] Backend test: duplicate correct submission does not re-award points
  - [x] Frontend test: pending status renders and later resolves

  **Agent-Executed QA Scenarios**:
  - Scenario: Player submits correct flag
    - Tool: Playwright
    - Steps:
      1. Login as player
      2. Navigate to `/challenges/[id]`
      3. Submit valid flag
      4. Assert status becomes `correct` (or pending → correct)
      5. Screenshot `.sisyphus/evidence/task-4-submit-correct.png`
  - Scenario: Duplicate correct submission does not change score
    - Tool: Bash (curl)
    - Steps:
      1. POST correct flag twice
      2. GET `/scoreboard`
      3. Assert score increases only once
    - Evidence: response body logs

- [x] 5. 管理员用户管理 + 基础审计/限流

  **What to do**:
  - 管理员用户列表、角色变更、禁用用户
  - 添加基础审计日志（管理员操作、提交评分）
  - 添加登录/提交基础限流

  **Must NOT do**:
  - 不实现复杂风控（设备指纹、多维分析）

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: 权限 + 安全 + 管理功能
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 3 (with Task 4)
  - Blocked By: Task 1

  **References**:
  - Auth middleware: `backend/internal/middleware/auth.go`, `backend/internal/middleware/rbac.go`
  - Auth module: `backend/internal/modules/auth/*`
  - Router wiring: `backend/internal/router/router.go`
  - Frontend admin UI pattern: `frontend/src/components/admin/*`

  **Acceptance Criteria (TDD)**:
  - [x] Admin-only endpoints return 403 to player
  - [x] Role change reflected in `/auth/me` response
  - [x] Rate limit triggers on rapid submissions

  **Agent-Executed QA Scenarios**:
  - Scenario: Player cannot access admin users
    - Tool: Bash (curl)
    - Steps:
      1. Login as player, get token
      2. GET `/api/v1/admin/users`
      3. Assert HTTP 403
  - Scenario: Admin promotes user
    - Tool: Playwright
    - Steps:
      1. Login as admin
      2. Navigate `/admin/users`
      3. Promote a user
      4. Logout and login as that user → role shows admin
      5. Screenshot `.sisyphus/evidence/task-5-admin-promote.png`

- [x] 6. Docker 启动链路与文档完善

  **What to do**:
  - 确保 docker-compose 一键启动前后端 + DB
  - 补充 README 启动步骤与测试命令
  - 明确种子管理员初始化方式

  **Must NOT do**:
  - 不引入复杂部署/云平台配置

  **Recommended Agent Profile**:
  - **Category**: `writing`
    - Reason: 文档与启动说明
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: NO
  - Parallel Group: Wave 4
  - Blocked By: Task 3-5

  **References**:
  - Docker compose: `docker-compose.yml`
  - Backend env example: `backend/.env.example`
  - Root scripts: `package.json`

  **Acceptance Criteria**:
  - [x] `docker compose up` → frontend/backed reachable
  - [x] README 包含完整运行/测试步骤

  **Agent-Executed QA Scenarios**:
  - Scenario: Local docker boot
    - Tool: Bash
    - Steps:
      1. `docker compose up -d`
      2. `curl http://localhost:8080/api/v1/health`
      3. Assert HTTP 200
    - Evidence: response body log

- [x] 7. 实例生命周期数据模型与状态机（后端）

  **What to do**:
  - 新增 `challenge_instances`（或等价）数据表：user_id/challenge_id/status/container_id/started_at/expires_at/cooldown_until
  - 建立状态机与原子迁移（start/stop/expire/fail）
  - 加入并发保护（同用户同一时刻最多 1 个 running/starting）

  **Acceptance Criteria (TDD)**:
  - [x] 并发 start 请求下仅 1 条实例进入 running/starting
  - [x] 冷却期内 start 返回冲突并含 retryAt
  - [x] 状态迁移非法路径被拒绝

  **Agent-Executed QA Scenarios**:
  - Tool: Bash (API + DB)
  - Steps: 并发触发 start 两次 → 校验仅一条活动实例

- [x] 8. Docker 运行控制器（start/stop）

  **What to do**:
  - 新增容器控制器模块：按 challenge 配置启动容器、停止容器、查询状态
  - 固定资源限制：0.5CPU + 512MB
  - 运行参数安全基线（非特权、最小权限）

  **Acceptance Criteria (TDD)**:
  - [x] start 成功返回 accessInfo + expiresAt
  - [x] stop 成功后容器被移除
  - [x] 容器 inspect 可见 CPU/内存限制符合配置

  **Agent-Executed QA Scenarios**:
  - Tool: Bash (API + docker inspect)
  - Steps: start → inspect 资源限制 → stop → verify removed

- [x] 9. 自动回收 Worker（严格 1h TTL）

  **What to do**:
  - 增加周期任务扫描到期实例并强制 stop
  - 写回 `expired` 状态并设置 cooldown
  - 失败重试与告警日志

  **Acceptance Criteria (TDD)**:
  - [x] 到期实例在调度周期内被回收
  - [x] 回收后状态为 expired 且不可访问
  - [x] 失败路径有可追踪日志

  **Agent-Executed QA Scenarios**:
  - Tool: Bash
  - Steps: 构造短 TTL 测试实例 → 等待回收周期 → 验证实例下线

- [x] 10. 实例 API 与权限边界

  **What to do**:
  - 新增 API：`POST /instances/start`, `POST /instances/stop`, `GET /instances/me`
  - RBAC：仅本人可控；管理员仅可查看/强停（若启用）

  **Acceptance Criteria (TDD)**:
  - [x] 非实例拥有者 stop 返回 403
  - [x] 用户在 cooldown 中 start 返回 409 + retryAt
  - [x] 用户已有 running 实例时 start 返回 409

  **Agent-Executed QA Scenarios**:
  - Tool: Bash (curl)
  - Steps: A 用户启动，B 用户尝试停止 A → 403

- [x] 11. 前端实例控制 UI（挑战详情页）

  **What to do**:
  - 挑战详情页加入 Start/Stop 按钮与状态显示（含剩余 TTL 与 cooldown 倒计时）
  - 去掉详情页 Top5 榜单块（减少认知负担）

  **Acceptance Criteria (TDD)**:
  - [x] 状态为 running 时仅显示 Stop
  - [x] 状态为 cooldown 时显示 retry 时间并禁用 Start
  - [x] 不再请求详情页榜单接口

  **Agent-Executed QA Scenarios**:
  - Tool: Playwright
  - Steps: 登录选手 → 进入题目详情 → start/stop/cooldown 交互验证并截图

- [x] 12. 端到端验收与运维文档补充

  **What to do**:
  - 补充本地 Docker 运行说明（含实例管理模块）
  - 增加故障排查：容器启动失败、TTL 未回收、冷却异常

  **Acceptance Criteria**:
  - [x] 文档覆盖启动、停止、自动回收、冷却策略
  - [x] 全链路验收证据齐全（start/stop/expire/cooldown）

---

## Commit Strategy

| After Task | Message | Files | Verification |
|---|---|---|---|
| 1 | `chore(ci): add test/coverage baselines` | root + frontend + backend configs | `pnpm test` |
| 2 | `feat(announcements): add player/admin UI` | frontend | `pnpm -C frontend test` |
| 3 | `feat(recruitment): add submission flow` | backend + frontend | `pnpm test` |
| 4 | `feat(submissions): add status history` | backend + frontend | `pnpm test` |
| 5 | `feat(admin): user management + audit` | backend + frontend | `pnpm test` |
| 6 | `docs(dev): docker + runbook` | README | N/A |

---

## Success Criteria

### Verification Commands
```bash
pnpm lint
pnpm type-check
pnpm test
docker compose up -d
curl http://localhost:8080/api/v1/health
```

### Final Checklist
- [x] MVP 功能闭环可用（选手+管理员）
- [x] RBAC + 限流 + 评分幂等生效
- [x] TDD 用例覆盖关键流程
- [x] Docker 启动链路稳定
