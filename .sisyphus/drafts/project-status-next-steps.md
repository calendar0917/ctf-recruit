# Draft: 项目现状盘点与下一步安排

## Requirements (confirmed)
- 用户需求：先看一下当前项目情况，再安排下一步。
- 盘点维度（用户已选）：
  - Git与变更面
  - 构建与依赖
  - 测试与质量
  - 架构与代码健康
  - 产品与任务优先级
- 额外诉求：希望有“可视化”，先看效果。
- 可视化重点：前后端交互情况（而非仅项目管理图表）。
- 用户更强调“体验产品本身”，不只看交互图。
- 核心体验路径（用户指定）：注册 → 登录 → 写题/做题 → 提交；以及后台管理。
- 用户最新反馈：`.trellis/.claude/.cursor` 这类流程资产“可能不再需要”。
- 用户决策：选择 B（先不动流程资产，先把产品体验跑通，再回头处理）。
- 用户确认：同意进入 Wave 1 Task 2，并需要可直接执行的命令清单。
- 输出深度：标准评估。
- 优先级依据：由我选择最优方法（可综合现有线索 + 风险/价值判断）。

## Technical Decisions
- 当前阶段：Interview Mode（先澄清盘点范围，再进入计划生成）。
- 评估输出将包含“现状结论 + 风险清单 + 下一步波次建议”。
- 优先级策略（暂定）：采用“价值/风险/依赖”综合排序，必要时参考现有 issues 或 backlog。
- 评估输出将加入“产品体验路径”（从用户视角走主流程）并映射到前后端交互链路。
- 执行优先级调整：先推进 Wave 1 的环境可运行与体验验证；流程资产治理后置。

## Research Findings
- 现有仓库结构（顶层）：frontend/ + backend/ + docker-compose.yml + pnpm-workspace.yaml。
- 前端：Next.js 14.2.5，包含路由 /register /login /challenges /challenges/[id] /admin/challenges。
- 前端测试：vitest（已有 smoke.test.ts）。
- 后端：Go + Fiber + GORM + Postgres；API 前缀 /api/v1。
- 后端路由已包含：auth(register/login/me)、challenges、submissions、scoreboard、announcement。
- 认证方式：JWT Bearer，角色包含 admin/player（RBAC 中间件）。
- 可运行环境：docker-compose 包含 postgres/redis/backend/frontend；frontend 通过 NEXT_PUBLIC_API_BASE_URL 指向 backend 容器。
- 风险：当前 main 工作区存在大量未暂存删除（主要在 .trellis/.claude/.cursor）+ 1 个未跟踪 draft 文件。
- 前后端交互主链路已识别：
  - FE /register → POST /api/v1/auth/register → auth handler/service/repo → users表
  - FE /login → POST /api/v1/auth/login → JWT签发 → 前端本地会话存储
  - FE /challenges → GET /api/v1/challenges → challenge service/repo
  - FE /challenges/[id] → GET /api/v1/challenges/:id + GET /api/v1/scoreboard
  - FE challenge submit → POST /api/v1/submissions → submission service + judge queue
  - FE /admin/challenges → challenge admin CRUD（RBAC admin）

## Metis Gap Review (incorporated)
- 需显式护栏：后台范围防膨胀（本阶段锁定题目管理，不扩用户审计等）。
- 需补充验收：体验链路需含 happy-path + failure-path 的 agent 可执行验证。
- 需校验假设：docker-compose 一键可跑、API契约与前端调用一致、RBAC覆盖完整。
- 需纳入边界：token过期/无权限/重复提交/服务不可用等失败场景。

## Open Questions
- 产品体验优先流程待确认（登录/注册、核心业务主流程、管理后台等）。
- 是否需要在计划中包含“可运行体验环境准备”（本地启动、测试账号、种子数据）。
- 下一步执行偏好的测试策略待确认（TDD / 先实现后补测 / 无自动化测试）。
- 需确认：流程资产删除是“正式决策”还是“暂缓观察”（影响 Task 1 的收尾方式与风险控制）。

## Test Strategy Decision
- **Infrastructure exists**: YES（frontend: vitest；backend: go test，且已有 *_test.go）
- **Automated tests**: YES（TDD）
- **Agent-Executed QA**: ALWAYS（用于端到端体验路径验证，补足单测覆盖不足）

## Scope Boundaries
- INCLUDE：项目现状评估、下一步工作规划。
- EXCLUDE：直接实现代码变更（仅规划，不执行）。
