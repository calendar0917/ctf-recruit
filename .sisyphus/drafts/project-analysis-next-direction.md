# Draft: Project Analysis & Next Direction

## Requirements (confirmed)
- 用户希望“完整分析当前项目”。
- 用户希望给出“现有问题”。
- 用户希望给出“下一步开发方向”。
- 用户要求先做上下文收集（并行探索 + 定向检索）再综合结论。

## Technical Decisions
- 分析基于当前仓库静态结构与关键代码路径，不做实现变更。
- 采用“架构现状 → 风险清单 → 发展方向”的组织方式，便于后续转工作计划。
- 风险分级使用：Critical / High / Medium。

## Research Findings
- 单仓结构：根脚本统一 lint/type-check/test，前端 pnpm、后端 Go 模块（`package.json`, `frontend/package.json`, `backend/go.mod`）。
- 运行拓扑：Docker Compose 管理 postgres/redis/migrate/backend/worker/frontend（`docker-compose.yml`）。
- 后端路由与模块装配集中在 `backend/internal/router/router.go`。
- 实例生命周期（start/stop/me/transition）、Docker runtime 控制、TTL sweeper 是核心复杂域（`backend/internal/modules/instance/*`）。
- 动态判题通过 DB 队列 + worker 轮询执行（`backend/internal/modules/judge/*`, `backend/cmd/worker/main.go`）。
- 前端 API 访问统一经过 `frontend/src/lib/http.ts`，会话保存在 localStorage（`frontend/src/lib/auth-storage.ts`）。
- CI 当前执行 lint/type-check/test，缺少覆盖率门禁、e2e、安全扫描（`.github/workflows/ci.yml`）。

## Scope Boundaries
- INCLUDE: 架构边界、关键流程、问题诊断、开发方向建议。
- EXCLUDE: 直接代码修改、重构实施、部署变更执行。

## Open Questions
- 下一步方向优先级是偏“安全稳定”还是“功能迭代速度”？
- 是否希望我把本分析直接转成可执行工作计划（.sisyphus/plans/*.md）？
