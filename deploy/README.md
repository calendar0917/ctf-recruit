# Deploy

当前 `deploy/docker-compose.yml` 主要用于开发环境。

默认包含：

- PostgreSQL
- Redis
- API
- Frontend（放在 `ui` profile 中）

## 当前状态

- API 通过挂载 Docker Socket 预留动态实例管理能力
- 当前 Compose 仍然是开发态：源码挂载、`go run`、Vite dev server
- 这适合本地开发和联调，不适合作为正式比赛的最终部署形态

## 开发环境约束

- `APP_ENV` 固定为 `development`
- `JWT_SECRET` 使用开发占位值 `dev-only-insecure-jwt-secret`
- 这个占位值只允许在开发环境使用；一旦切到非 `development` 环境，API 启动会直接失败
- 默认迁移不再自动创建管理员；如需本地默认账号，需要额外执行 `scripts/dev-seed.sh`

## 目标生产形态

比赛前的目标部署应至少包含：

- 反向代理
- 构建后的前端静态资源
- 构建后的 Go API 服务
- PostgreSQL
- Redis
- Docker Engine 与动态题容器

## 比赛前必须补齐的部署项

- 生产模式构建与运行脚本
- TLS 和反向代理配置
- 数据库初始化与种子流程统一
- 备份与恢复说明
- 日志、指标和基础告警
- 赛前彩排与压测流程

具体优先级以 [开发基线与升级路线](/home/calendar/code/ctf/docs/development-baseline.md) 为准。
