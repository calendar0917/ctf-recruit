# Deploy

当前开发环境使用 `deploy/docker-compose.yml`。

默认包含：

- PostgreSQL
- Redis
- API
- Frontend（放在 `ui` profile 中）

说明：

- API 通过挂载 Docker Socket 预留动态实例管理能力
- 这适合单机开发和小规模部署，不适合作为最终高隔离生产方案
