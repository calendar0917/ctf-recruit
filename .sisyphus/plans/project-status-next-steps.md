# 项目现状盘点与下一步（体验优先 + TDD）工作计划

## TL;DR

> **快速结论**：先把仓库从“危险脏状态（大量未暂存删除）”安全隔离出来，再用“一键可跑 + 种子数据/默认账号”的方式让你立刻体验完整主流程（注册→登录→做题/提交→榜单）与后台题目管理；并用 **TDD** 为关键链路补齐自动化验证。
>
> **核心交付物**：
> - 可运行体验环境（docker-compose/本地启动均可）+ seed 数据 + 默认账号（玩家/管理员）
> - 体验路径的端到端验证（Agent 执行的 QA 场景 + 必要的自动化测试）
> - 前后端交互链路映射（按用户旅程）与风险清单
>
> **Estimated Effort**：Medium
> **Parallel Execution**：YES（2-3 waves）
> **Critical Path**：环境可跑 → 体验主流程可走通 → 后台管理可用 → 自动化覆盖与稳态

---

## Context

### Original Request
你希望“先看一下现在项目的情况，然后继续安排下一步”。你强调的“可视化”不是画图，而是**能真实体验产品效果**。

### Interview Summary
- 盘点维度：Git与变更面、构建与依赖、测试与质量、架构与代码健康、产品与任务优先级。
- 体验主流程：**注册 → 登录 → 写题/做题 → 提交**，以及**后台管理**。
- 输出深度：标准评估。
- 下一步执行策略：**TDD**。

### Evidence / Repo Facts (已核实)
- Repo 顶层：`frontend/`（Next.js）+ `backend/`（Go Fiber）+ `docker-compose.yml`。
- 前端：Next.js 14.2.5；路由包含：
  - `/register`、`/login`、`/challenges`、`/challenges/[id]`、`/admin/challenges`
- 后端：Go + Fiber + GORM + Postgres；API 前缀：`/api/v1`。
- 认证：JWT Bearer；中间件：`middleware.Auth` + `middleware.RequireRoles(admin)`。
- docker-compose：postgres/redis/backend/frontend；frontend 通过 `NEXT_PUBLIC_API_BASE_URL` 指向 `http://backend:8080`。

### Current Risk Snapshot (高优先)
- 当前在 `main` 分支存在 **大量未暂存删除（~123项）**，集中于 `.trellis/.claude/.cursor`。若误提交会破坏团队流程资产。

---

## Work Objectives

### Core Objective
在不引入无边界扩张的前提下，**让项目可一键启动并可端到端体验主流程**，同时对关键链路建立可重复、可自动验证的质量门槛（TDD + Agent QA）。

### Concrete Deliverables
- [x] **可运行体验环境**：提供“从零到可体验”的最短路径（命令、环境变量、数据库迁移/seed、默认账号）。
- [x] **体验主流程可走通**：注册、登录、题目列表/详情、提交、榜单刷新。
- [x] **后台管理可走通**：管理员登录后进入 `/admin/challenges` 完成题目 CRUD + 发布/下线。
- [x] **前后端交互链路映射**（按用户旅程）+ 风险/薄弱点清单。
- [x] **自动化验证**：
  - 后端：关键 API 行为的 go test（TDD）
  - 前端：关键页面/表单最小 vitest 覆盖 + 端到端 Agent QA 场景

### Must Have
- 一键体验（docker-compose 或本地双端启动）
- 主流程与后台管理均可体验
- TDD（至少覆盖主流程关键后端行为）

### Must NOT Have (Guardrails)
- 不将“后台管理”扩展成完整运营系统（用户审计、封禁、复杂权限体系）——本阶段锁定**题目管理**为主。
- 不做大规模 UI 视觉重构：只修复阻塞体验/信息架构问题。
- 不把“可运行环境准备”升级为生产化部署（k8s/监控/CI/CD 全套）——本阶段只保证本地/compose 可重复跑。

---

## Verification Strategy (MANDATORY)

> **UNIVERSAL RULE: ZERO HUMAN INTERVENTION**
>
> 所有验收必须可由执行 agent 通过命令/工具直接验证，不允许“请用户手动打开浏览器确认”。

### Test Decision
- **Infrastructure exists**: YES
  - Frontend: `vitest`（`frontend/package.json`）
  - Backend: `go test`（`backend/Makefile` & `go.mod`）
- **Automated tests**: **TDD**
- **Agent-Executed QA**: ALWAYS（用于端到端体验验证，补足单测覆盖不足）

### Agent-Executed QA Scenarios
执行 agent 需使用：
- 前端体验：Playwright（推荐）或同等浏览器自动化
- 后端 API：curl/httpie（Bash）
- 服务启动：docker-compose / 本地命令（Bash）

每个关键功能至少提供：
- 1 个 happy-path 场景
- 1 个 failure-path 场景（401/403/422/500 等）

证据存放：`.sisyphus/evidence/`（截图、响应体、命令输出）。

---

## Execution Strategy

### Parallel Execution Waves

Wave 1（安全与可跑基线）
1) 收敛 Git 变更面（避免误提交大量删除）
2) 环境可启动（compose/本地）+ 最小 smoke 验证
3) 生成可体验所需的 seed 数据与默认账号策略（玩家/管理员）

Wave 2（体验主流程）
4) 注册/登录端到端可用（含错误态）
5) 题目浏览（列表/详情）端到端可用
6) 提交与判题队列联动可观测（含 pending→done/failed）
7) 榜单刷新与排序逻辑体验可用

Wave 3（后台管理 + 质量门槛）
8) 管理端题目 CRUD/发布流程可用（RBAC）
9) TDD 覆盖关键后端行为 + 前端关键表单最小测试
10) 稳态与失败场景加固（token 过期、权限不足、依赖服务不可用等）

---

## TODOs

> 说明：每条任务都要求「TDD（若适用）+ Agent 可执行 QA 场景 + 证据路径」。

### 1. Git 变更面安全隔离（防误删）

**What to do**:
- 识别当前 `main` 工作区中约 123 个“删除”变更（集中 `.trellis/.claude/.cursor`）的来源：
  - 是工具生成目录？还是误操作？是否应恢复？
- 明确策略：
  - 若这些目录应保留：恢复为干净状态
  - 若确实要清理：单独分支/单独提交，并明确理由与影响

**Must NOT do**:
- 不允许在未确认意图前，把 123 个删除直接提交到主线。

**Recommended Agent Profile**:
- Category: `quick`
- Skills: `git-master`

**Parallelization**: Wave 1，可与任务 2 并行（但建议先处理，以免后续 diff 噪声）

**References**:
- 现状风险快照：工作区大量未暂存删除（来自先前 git 盘点结果）

**Acceptance Criteria**:
- [x] `git status` 显示变更面已被明确处理：要么恢复干净，要么变更被隔离到明确的分支/提交集合
- [x] 不存在“未解释的大规模删除”悬挂在工作区

**Agent-Executed QA Scenarios**:
- Scenario: Working tree becomes clean
  - Tool: Bash
  - Steps: 运行 `git status` 并保存输出到 `.sisyphus/evidence/task-1-git-status.txt`
  - Expected: 无意外大规模删除残留

---

### 2. 可运行体验环境基线（compose/local）

**What to do**:
- 建立“从零到可跑”的最短路径：
  - docker-compose 启动 postgres/redis/backend/frontend
  - 或本地分别启动（pnpm + go）
- 明确必须的环境变量：后端 `DATABASE_URL`、`JWT_SECRET` 等（见 config.Validate）。
- 补齐迁移/seed 的执行方式（若当前未自动执行）。

**Recommended Agent Profile**:
- Category: `unspecified-high`
- Skills: `git-master`

**Parallelization**: Wave 1

**References**:
- `docker-compose.yml`：服务定义与端口映射
- `backend/internal/config/config.go`：必需 env（DATABASE_URL/JWT_SECRET）
- `backend/cmd/api/main.go`：启动入口与校验
- `frontend/package.json`：`dev/start` 脚本

**Acceptance Criteria**:
- [x] Agent 能用命令启动全套服务（compose 或本地），并在日志中看到 backend listen 与 frontend ready
- [x] `GET /api/v1/health` 返回 200（或等价健康检查）并保存响应

**Agent-Executed QA Scenarios**:
```
Scenario: Compose brings up full stack
  Tool: Bash
  Steps:
    1. docker-compose up -d
    2. 等待 backend 8080 可用
    3. curl -s -D - http://localhost:8080/api/v1/health
    4. 保存响应头/体到 .sisyphus/evidence/task-2-health.txt
  Expected Result: health 200
  Evidence: .sisyphus/evidence/task-2-health.txt
```

---

### 3. Seed 数据与默认账号（体验必需）

**What to do**:
- 提供：
  - 至少 1 个管理员账号（能进入 `/admin/challenges`）
  - 至少 1 个普通玩家账号
  - 至少若干题目（含静态 flag 与动态判题 mode 的样例）
- 明确密码策略（避免弱口令/固定明文写入仓库）。

**Guardrails**:
- 不引入邮件验证/邀请码系统（除非后续明确需求）。

**Recommended Agent Profile**:
- Category: `unspecified-high`
- Skills: `git-master`

**Parallelization**: Wave 1（依赖任务 2 的数据库可用）

**References**:
- `backend/internal/modules/auth/service.go`：注册时默认 role=player
- `backend/internal/middleware/rbac.go`：admin role 校验
- `backend/migrations/`：数据库结构（需执行 agent 具体查看并按实际写 seed）

**Acceptance Criteria**:
- [x] Agent 可用 seed 后的管理员账号登录成功并访问管理页面
- [x] Agent 可用玩家账号登录成功并浏览题目列表

---

### 4. 注册（Register）端到端体验 + TDD

**What to do (TDD)**:
- RED：为后端注册服务/handler 添加失败测试用例（缺 email/password/displayName、重复邮箱）
- GREEN：实现/修复以通过
- REFACTOR：整理错误码一致性

**Recommended Agent Profile**:
- Category: `unspecified-high`
- Skills: `git-master`

**References**:
- FE: `frontend/src/app/register/page.tsx`、`frontend/src/lib/api/auth.ts`
- BE: `backend/internal/modules/auth/handler.go`、`backend/internal/modules/auth/service.go`

**Acceptance Criteria**:
- [x] go test 覆盖：注册成功、重复邮箱 409、缺字段 400（或约定码）

**Agent-Executed QA Scenarios** (Playwright):
```
Scenario: Register success then redirect to login
  Tool: Playwright
  Steps:
    1. Open http://localhost:3000/register
    2. Fill input[type="text"] (Display Name) -> "Test User"
    3. Fill input[type="email"] -> "test+1@example.com"
    4. Fill input[type="password"] -> "Passw0rd!"
    5. Click button[type="submit"]
    6. Wait for URL /login
    7. Screenshot .sisyphus/evidence/task-4-register-success.png
```

---

### 5. 登录（Login）与会话（Me）端到端体验 + TDD

**What to do (TDD)**:
- RED：登录成功返回 accessToken；错误密码/不存在用户返回 401；`/me` 无 token 401
- GREEN：对齐错误码/错误信息

**References**:
- FE: `frontend/src/app/login/page.tsx`、`frontend/src/lib/auth-storage.ts`、`frontend/src/lib/use-auth.ts`
- BE: `backend/internal/middleware/auth.go`、`backend/internal/modules/auth/service.go`

**Agent-Executed QA Scenarios**:
- 成功登录后跳转 `/challenges`
- 无 token 访问 `/challenges` 自动跳 `/login`

---

### 6. 题目浏览（列表/详情）端到端体验 + TDD

**What to do**:
- 确认列表分页与 publishedOnly 行为：
  - 玩家：只能看到已发布
  - 管理员：能看到全部

**References**:
- FE: `frontend/src/app/challenges/page.tsx`、`frontend/src/app/challenges/[id]/page.tsx`
- BE: `backend/internal/modules/challenge/handler.go`（publishedOnly 逻辑）

**Agent-Executed QA Scenarios**:
- 玩家登录后看到题目列表
- 打开某题详情页，能看到题目内容与榜单 Top 5

---

### 7. 提交（Submission）→ 判题队列（Judge）联动体验 + TDD

**What to do (重点边界)**:
- 静态题：flag hash 匹配即 correct 并加分
- 动态题：创建 submission 为 pending，并 enqueue judge job；worker 处理后 finalize 写回
- failure-path：queue 不可用、job 执行失败、重复提交/唯一约束冲突路径

**References**:
- BE: `backend/internal/modules/submission/service.go`（静态/动态分支 + enqueue）
- BE: `backend/internal/modules/judge/queue.go`（MockExecutor + worker 流）
- BE: `backend/internal/modules/judge/repository.go`（FinalizeExecution 原子写回）
- DTO: `backend/internal/modules/submission/dto.go`

**Agent-Executed QA Scenarios**:
```
Scenario: Dynamic challenge submission becomes pending then finalized
  Tool: Bash + Playwright
  Steps:
    1. Playwright 登录玩家账号
    2. 进入某个动态题目详情，提交 flag
    3. 断言提交响应 status=pending（或 UI 展示）并截图
    4. 启动/确认 worker 在跑（或触发一次 ProcessOnce）
    5. 再次刷新详情页/榜单，断言状态变为 done(correct/wrong/failed)
  Evidence: .sisyphus/evidence/task-7-dynamic-flow.png
```

---

### 8. 榜单（Scoreboard）体验与一致性校验 + TDD

**What to do**:
- 校验排序：总分降序；同分按 lastAcceptedAt 早者在前（见 service.go）

**References**:
- BE: `backend/internal/modules/scoreboard/service.go`
- FE: `frontend/src/lib/api/scoreboard.ts`、`frontend/src/app/challenges/[id]/page.tsx`

---

### 9. 后台管理（Admin Challenge Management）端到端体验 + TDD

**What to do**:
- 管理员可：创建/编辑/删除题目；切换发布状态
- 非管理员：访问 `/admin/challenges` 显示 403/引导

**Scope Guardrail**:
- 不做用户管理/审计面板等扩张

**References**:
- FE: `frontend/src/app/admin/challenges/page.tsx`、`frontend/src/components/admin/*`
- BE: `backend/internal/modules/challenge/handler.go`（admin-only CRUD）
- BE: `backend/internal/middleware/rbac.go`

---

### 10. 稳态与失败场景加固（体验质量门槛）

**What to do**:
- token 过期/无 token/权限不足 → 前端提示一致、不会泄露缓存数据
- 依赖服务不可用（postgres/redis）→ 后端错误码/日志可定位

**Acceptance Criteria**:
- [x] failure-path 场景均有自动化验证或 QA 场景覆盖（至少 1 条/功能）

---

## Commit Strategy

- 建议以体验路径为单位做原子提交：
  - `chore(repo): stabilize working tree and env bootstrap`
  - `feat(auth): enable register/login e2e flow with tests`
  - `feat(submission): dynamic judge writeback verified by tests`
  - `feat(admin): challenge management flow with RBAC checks`

---

## Success Criteria

### Verification Commands (examples)
- `pnpm test`（前端 vitest）
- `cd backend && go test ./...`
- `pnpm lint` / `pnpm type-check`
- `curl http://localhost:8080/api/v1/health`

### Final Checklist
- [x] 你指定的体验链路（注册→登录→做题/提交→榜单）由 agent 端到端跑通并产出证据
- [x] 后台管理（题目 CRUD + 发布）由 agent 端到端跑通并产出证据
- [x] TDD 产出：关键后端行为有测试覆盖；前端关键表单有最小测试覆盖
- [x] Git 变更面安全：不存在未解释的大规模删除风险
