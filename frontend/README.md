# Frontend

当前前端使用 `Vite + React + TypeScript` 作为基础骨架。

## Demo UI

当前提供一个“聚焦做题”的前端 demo：

- `Briefing`：比赛状态 + 阶段 gating + 公告 + 登录/注册
- `Challenges`：题目列表/筛选 + 题目详情/附件 + Flag 提交 + 动态实例（dynamic 题目）
- `Scoreboard`：排行榜（可展开 solves）
- `My`：个人解题/提交记录

说明：

- UI 行为会遵循 `GET /api/v1/contest` 返回的 `phase.*` 进行 gating
- 后端 base URL 由 `vite.config.ts` 代理 `/api` 到 `http://localhost:8080`

后续建议拆分的页面模块：

- 登录 / 注册
- 比赛首页
- 题目列表与详情
- 动态实例控制面板
- 排行榜
- 管理后台
