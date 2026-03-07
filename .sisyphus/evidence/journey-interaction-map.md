# 前后端交互链路映射（按用户旅程）+ 风险/薄弱点清单

> 范围：仅覆盖本仓库已验证链路（注册/登录/题目浏览/提交判题/榜单/后台题目管理），并基于 `.sisyphus/evidence/` 与 notepads 的既有证据。

## 0) 全局链路基线（入口与鉴权）

- **API 前缀**：`/api/v1`（`backend/internal/router/router.go`）
- **核心鉴权中间件**：
  - `middleware.Auth`：读取 `Authorization: Bearer <token>`，缺失返回 `AUTH_MISSING_TOKEN`（401）
    - 文件：`backend/internal/middleware/auth.go`
  - `middleware.RequireRoles(auth.RoleAdmin)`：角色不匹配返回 `AUTH_FORBIDDEN`（403）
    - 文件：`backend/internal/middleware/rbac.go`
- **前端会话守卫**：`useRequireAuth()`（无会话跳转 `/login`；可做 adminOnly）
  - 文件：`frontend/src/lib/use-auth.ts`

---

## 1) 用户旅程：注册（Register）

### 1.1 交互步骤
1. 用户访问 `GET /register`
2. 前端提交注册表单（email/password/displayName）
3. 后端创建账号后返回 201
4. 前端跳转到 `/login`

### 1.2 FE ↔ BE 映射
- **FE 页面**：`frontend/src/app/register/page.tsx`
- **FE API 调用**：`frontend/src/lib/api/auth.ts#register()` → `POST /api/v1/auth/register`
- **BE 路由**：`backend/internal/router/router.go`（`authGroup.Post("/register", ...)`）
- **BE Handler/Service**：
  - `backend/internal/modules/auth/handler.go#Register`
  - `backend/internal/modules/auth/service.go#Register`

### 1.3 证据
- 成功截图目标任务存在，但当前环境 Playwright 安装/拉取受阻：
  - `.sisyphus/evidence/task-4-register-failure.txt`

---

## 2) 用户旅程：登录与会话（Login / Me）

### 2.1 交互步骤
1. 用户访问 `GET /login`
2. 前端提交邮箱密码
3. 后端返回 `accessToken + user`
4. 前端写入本地会话并跳转 `/challenges`
5. 业务页通过 token 调 `GET /api/v1/auth/me` 验证会话

### 2.2 FE ↔ BE 映射
- **FE 页面**：`frontend/src/app/login/page.tsx`
- **会话存储**：`frontend/src/lib/auth-storage.ts`
- **会话守卫**：`frontend/src/lib/use-auth.ts`
- **FE API 调用**：
  - `frontend/src/lib/api/auth.ts#login()` → `POST /api/v1/auth/login`
  - `frontend/src/lib/api/auth.ts#me()` → `GET /api/v1/auth/me`
- **BE 路由与实现**：
  - `backend/internal/router/router.go`
  - `backend/internal/modules/auth/handler.go`（`Login` / `Me`）
  - `backend/internal/modules/auth/service.go`（签发与解析 JWT）
  - `backend/internal/middleware/auth.go`（token 解析）

### 2.3 证据
- 登录成功（含 accessToken）：`.sisyphus/evidence/task-5-login-success.txt`
- 错误凭据 401：`.sisyphus/evidence/task-5-login-invalid-credentials-401.txt`
- 无 token 访问 me 401：`.sisyphus/evidence/task-5-me-without-token-401.txt`
- 过期 token 401：`.sisyphus/evidence/task-10-expired-token-401.json`

---

## 3) 用户旅程：题目浏览（列表/详情）

### 3.1 交互步骤
1. 用户在 `/challenges` 拉取题目列表
2. 用户进入 `/challenges/[id]` 拉取题目详情
3. 详情页并行拉取榜单 Top 5 作为侧边信息

### 3.2 FE ↔ BE 映射
- **FE 页面**：
  - `frontend/src/app/challenges/page.tsx`
  - `frontend/src/app/challenges/[id]/page.tsx`
- **FE API 调用**：
  - `frontend/src/lib/api/challenges.ts#listChallenges()` → `GET /api/v1/challenges?limit&offset`
  - `frontend/src/lib/api/challenges.ts#getChallenge()` → `GET /api/v1/challenges/:id`
  - `frontend/src/lib/api/scoreboard.ts#listScoreboard()` → `GET /api/v1/scoreboard`
- **BE 路由与实现**：
  - `backend/internal/modules/challenge/handler.go`
    - `List/Get` 对 admin 与 player 采用不同 `publishedOnly` 逻辑
  - `backend/internal/modules/challenge/service.go`

### 3.3 证据
- 玩家题目列表：`.sisyphus/evidence/task-6-player-challenges-list.txt`
- 管理员题目列表：`.sisyphus/evidence/task-6-admin-challenges-list.txt`
- admin/player 可见性差异（含 unpublished）：
  - `.sisyphus/evidence/task-9-admin-challenge-management-admin-list-before-toggle.json`
  - `.sisyphus/evidence/task-9-admin-challenge-management-player-list-before-toggle.json`

---

## 4) 用户旅程：提交与判题（Submission → Judge Worker）

### 4.1 交互步骤
1. 用户在题目详情提交 flag
2. 后端创建 submission
   - static：直接判定 correct/wrong
   - dynamic：先返回 `pending`，并创建 `judgeJob`
3. worker 拉取 queued job 执行，原子写回 submission/job 最终状态

### 4.2 FE ↔ BE 映射
- **FE 页面/组件入口**：`frontend/src/app/challenges/[id]/page.tsx`（提交后刷新）
- **FE API 调用**：`frontend/src/lib/api/submissions.ts#createSubmission()` → `POST /api/v1/submissions`
- **BE 路由与实现**：
  - `backend/internal/modules/submission/handler.go`
  - `backend/internal/modules/submission/service.go`
    - dynamic 分支：`StatusPending` + `queue.Enqueue`
  - `backend/internal/modules/judge/queue.go`（`Worker.ProcessOnce`）
  - `backend/internal/modules/judge/repository.go`（`FinalizeExecution`）

### 4.3 证据
- dynamic 提交初始 pending：`.sisyphus/evidence/task-7-dynamic-submit.pending.json`
- submission 最终写回（correct + 分数）：`.sisyphus/evidence/task-7-submission.after.txt`
- judge job 前后状态：`.sisyphus/evidence/task-7-judge-job.before.txt`、`.sisyphus/evidence/task-7-judge-job.after.txt`

---

## 5) 用户旅程：榜单（Scoreboard）

### 5.1 交互步骤
1. 用户请求榜单（首页/详情页都可能触发）
2. 后端聚合 solved 数据并排序返回
3. 前端展示 rank / points / solved count

### 5.2 FE ↔ BE 映射
- **FE 页面**：`frontend/src/app/scoreboard/page.tsx`、`frontend/src/app/challenges/[id]/page.tsx`
- **FE API 调用**：`frontend/src/lib/api/scoreboard.ts#listScoreboard()` → `GET /api/v1/scoreboard?limit&offset`
- **BE 路由与实现**：
  - `backend/internal/modules/scoreboard/handler.go`
  - `backend/internal/modules/scoreboard/service.go`
    - 排序规则：`totalPoints desc` → `lastAcceptedAt asc` → `userId asc`

### 5.3 证据
- API 响应：`.sisyphus/evidence/task-8-scoreboard-response-20260215T235229Z.json`
- 排序核验：`.sisyphus/evidence/task-8-scoreboard-ordering-check.txt`
- 场景元数据：`.sisyphus/evidence/task-8-scoreboard-meta-20260215T235229Z.txt`

---

## 6) 用户旅程：后台题目管理（Admin Challenge Flow）

### 6.1 交互步骤
1. 管理员进入 `/admin/challenges`
2. 拉取题目列表（含 unpublished）
3. 执行创建/编辑/发布切换/删除
4. 普通玩家访问同类变更接口应被拒绝（403）

### 6.2 FE ↔ BE 映射
- **FE 页面**：`frontend/src/app/admin/challenges/page.tsx`
- **FE 组件**：
  - `frontend/src/components/admin/ChallengeEditor.tsx`
  - `frontend/src/components/admin/ChallengeTable.tsx`
- **FE API 调用**（均在 `frontend/src/lib/api/challenges.ts`）：
  - `createChallenge()` → `POST /api/v1/challenges`
  - `updateChallenge()` → `PUT /api/v1/challenges/:id`
  - `deleteChallenge()` → `DELETE /api/v1/challenges/:id`
  - `listChallenges()` → `GET /api/v1/challenges`
- **BE 鉴权与路由**：
  - `backend/internal/modules/challenge/handler.go#RegisterRoutes`
  - `middleware.RequireRoles(auth.RoleAdmin)` 限制变更接口

### 6.3 证据
- 汇总：`.sisyphus/evidence/task-9-admin-challenge-management-summary.json`
- 玩家写操作被拒绝：`.sisyphus/evidence/task-9-admin-challenge-management-player-post-forbidden.json`
- 发布前后可见性变化：
  - `.sisyphus/evidence/task-9-admin-challenge-management-player-list-before-toggle.json`
  - `.sisyphus/evidence/task-9-admin-challenge-management-player-list-after-toggle.json`

---

## 7) 风险 / 薄弱点清单（基于现有证据）

### R1. E2E 浏览器验证链路不稳定（环境依赖）
- **现象**：Playwright Chrome 缺失、browser_install 超时、容器镜像拉取长期超时，导致注册 UI happy-path 截图证据缺口。
- **证据**：`.sisyphus/evidence/task-4-register-failure.txt`
- **影响**：前端关键跳转与交互只能由 API 证据替代，UI 回归风险上升。
- **建议缓解**：
  1. 在 CI 预置 Playwright 浏览器缓存（或固定可用镜像）。
  2. 增加“API + 组件测试 + 最小浏览器冒烟”分层，避免单点依赖真实浏览器安装。

### R2. 依赖服务异常时，错误外显统一但可恢复策略不足
- **现象**：postgres 不可用时，题目列表返回 500（`CHALLENGE_LIST_FAILED`），日志可定位但用户侧仅泛化错误。
- **证据**：
  - `.sisyphus/evidence/task-10-dependency-unavailable-500.json`
  - `.sisyphus/evidence/task-10-dependency-unavailable-backend.log`
- **影响**：可观测性有了，但前端缺“可重试/降级提示”标准交互，故障体验较硬。
- **建议缓解**：
  1. 前端统一 5xx 提示模板 + retry CTA。  
  2. 后端按 error code 输出可机器识别的 retryable 标记（不暴露内部细节）。

### R3. 动态判题链路对 queue 可用性敏感
- **现象**：dynamic 提交依赖 `queue.Enqueue`；若 queue 不可用会返回 `JUDGE_QUEUE_UNAVAILABLE` / `JUDGE_JOB_ENQUEUE_FAILED`。
- **依据文件**：`backend/internal/modules/submission/service.go`
- **影响**：用户提交成功率受队列/worker 状态直接影响；高峰下可能出现 pending 堆积。
- **建议缓解**：
  1. 增加队列健康探针与 backlog 指标告警。
  2. 为 pending 超时增加后台补偿扫描/重试机制。

### R4. RBAC 在 API 侧可靠，前端侧主要为体验兜底
- **现象**：403 拦截已由后端保证，但前端 admin 页面仍基于会话 role 做展示层判断。
- **证据**：`.sisyphus/evidence/task-10-player-admin-endpoint-403.json`
- **影响**：安全上可接受（以后端为准），但前端角色态与真实权限若短时不一致会产生体验抖动。
- **建议缓解**：
  1. 管理页首次加载可附带一次 `/auth/me` 强校验。  
  2. 403 统一落地到可解释错误页，减少“空白/闪跳”体验。

### R5. 本地验证路径对执行环境（工具链/权限/网络）敏感
- **现象**：历史记录中出现 Go 缺失、Vitest 缓存权限错误、compose build 网络超时等。
- **证据来源**：`.sisyphus/notepads/project-status-next-steps/issues.md`
- **影响**：不同执行机上“同一任务证据可重复性”下降，影响交付稳定性。
- **建议缓解**：
  1. 固化 devcontainer/CI 基线镜像（go/node/playwright）。
  2. 在 `README` 增加“一键环境自检”脚本与降级执行路径说明。

---

## 8) 结论（当前可用性）

- 主链路（register/login/challenge/submission/scoreboard/admin challenge）在 **API 与代码映射层面完整闭环**。  
- 已有证据可支撑权限边界、发布可见性、动态判题 pending→finalized、榜单排序一致性。  
- 当前主要薄弱点在 **执行环境稳定性与故障体验细节**，非主业务流程缺失。
