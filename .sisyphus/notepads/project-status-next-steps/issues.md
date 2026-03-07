# Issues / Risks (append-only)

## 2026-02-15
- 
- 风险：frontend 镜像构建在 `npm install` 阶段曾超时（此前 120s 超时窗口），导致 full compose 路径不稳定。
- 处置：本次任务采用 backend-only compose 路径（postgres/redis/backend）完成可运行基线与健康检查；frontend 未在本次尝试中启动。
- 阻塞：宿主机缺少 `go`/`gofmt`（`go: command not found`），无法本地直接 `go run`/`go test`。
- 最短绕过：使用 Docker 构建并运行 seed（`ctf-recruit-backend-local` 镜像内执行 `/bin/seed`），并用容器化 backend + curl 验证登录与题目列表行为。

### task-1-git-change-surface-isolation
- 风险：`main` 工作区出现大规模未暂存删除（`.trellis`/`.claude`/`.cursor` 共 122 条），存在误提交并污染后续开发链路的高风险。
- 处置：采用最保守“恢复”策略，执行 `git restore .claude .cursor .trellis`，未进行提交、未修改 git config、未使用破坏性命令。
- 结果：删除项由 122 降至 0，`git status --short` 不再有未解释的大规模删除；当前仅剩 `?? .sisyphus/`（任务证据与记录文件）。
- 证据：`.sisyphus/evidence/task-1-git-status.txt`（同一文件追加记录了 BEFORE/AFTER 状态、目录分布与恢复结果）。

### task-4-register-e2e-playwright
- 阻塞：Playwright MCP 环境缺失 Chrome (`browserType.launchPersistentContext: Chromium distribution 'chrome' is not found at /opt/google/chrome/chrome`)，且 `browser_install` 超时。
- 处置：回退容器方案，尝试拉取 `mcr.microsoft.com/playwright:v1.50.1-jammy` 并在容器内执行注册 happy-path；镜像拉取在当前网络下多次超时（300s/900s/1800s）。
- 结果：后端 TDD 与测试已完成，但 Task 4 所需 screenshot 证据未生成；已记录失败证据文件。
- 证据：`.sisyphus/evidence/task-4-register-failure.txt`。

## 2026-02-16

### task-5-login-me-e2e-playwright
- 说明：按计划 Task 5 原本建议 Playwright 覆盖“登录后跳转/无 token 跳转登录”。受同一浏览器环境阻塞影响，本次以 backend TDD + curl QA 证据替代（计划中允许 API QA 场景）。
- 证据：`.sisyphus/evidence/task-5-login-success.txt`、`.sisyphus/evidence/task-5-login-invalid-credentials-401.txt`、`.sisyphus/evidence/task-5-me-without-token-401.txt`。

### task-6-tests-frontend-permission-cache
- 现象：执行 `npm test` 后，Vitest 在写入 `frontend/node_modules/.vite` 缓存时出现 `EACCES: permission denied`。
- 处置：使用 `npm test -- --cache false` 运行前端测试，测试通过；未修改依赖与权限配置，避免超出本任务范围。

### task-7-submission-judge-worker-evidence
- 风险：`docker-compose build backend worker` 在当前网络环境下失败（Alpine `apk add` 拉取超时），且 compose 提示 buildx 插件未安装。
- 处置：测试与 worker 验证改用 `golang:1.22-alpine` 容器直接运行（挂载源码 + `GOPROXY=https://goproxy.cn,direct`），避免依赖本地构建 worker 镜像。
- 结果：Task 7 所需 API/DB 证据已生成并满足 pending→finalized 可观测性；compose 中 worker 服务定义保留以支持后续网络恢复后直接启用。

### task-8-scoreboard-ordering-consistency
- 未发现新的阻塞项；Scoreboard 排序规则在单测与 API 证据中均与预期一致。

### task-9-admin-challenge-management-tdd
- 环境约束：前端 TypeScript LSP (`typescript-language-server`) 在当前执行环境未安装，导致 `lsp_diagnostics` 无法用于 TS 文件；改用 `npm run typecheck` + `npm test` 作为等价质量门槛。
- 说明：后端 seed 账户在本环境未配置固定密码；本次 API QA 通过使用后端 `.env.example` 的 `JWT_SECRET=change-me` 生成 admin/player JWT（role claim）进行 RBAC 端到端验证，避免依赖未知 seed 密码。

### task-10-dependency-unavailable-simulation
- 说明：依赖不可用路径使用 `docker compose stop postgres` 来模拟数据库不可用，触发 handler 500 structured error。
- 风险：backend 连接 docker network 内的 `postgres` hostname；当 postgres 停止时会出现 DNS lookup 失败（`no such host`）而非连接拒绝，这属于 compose 网络行为；错误仍可通过 requestId + repo file/line 输出定位。
