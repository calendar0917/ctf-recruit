# Deploy

当前仓库现在同时提供两套部署入口：

- [deploy/docker-compose.yml](/home/calendar/code/ctf/deploy/docker-compose.yml)：开发环境
- [deploy/docker-compose.prod.yml](/home/calendar/code/ctf/deploy/docker-compose.prod.yml)：最小生产部署骨架

## 开发环境

开发 Compose 默认包含：

- PostgreSQL
- Redis
- API
- Frontend（放在 `ui` profile 中）

开发环境约束：

- API 通过挂载 Docker Socket 预留动态实例管理能力
- 当前 Compose 是开发态：源码挂载、`go run`、Vite dev server
- `APP_ENV` 固定为 `development`
- `JWT_SECRET` 使用开发占位值 `dev-only-insecure-jwt-secret`
- 默认迁移不再自动创建管理员；如需本地默认账号，需要额外执行 `scripts/dev-seed.sh`
- 限流默认走 Redis；若 `REDIS_ADDR` 为空会自动回退到进程内内存限流

## 生产骨架

`deploy/docker-compose.prod.yml` 现在提供最小生产拓扑：

- `gateway`：对外 Nginx 反向代理
- `frontend`：构建后的静态资源镜像
- `api`：构建后的 Go 二进制镜像
- `postgres`
- `redis`

生产 Compose 特点：

- 不再依赖 `go run`
- 不再依赖 Vite dev server
- 前端通过 Nginx 提供静态资源并支持 SPA fallback
- API 运行在 `APP_ENV=production`
- `JWT_SECRET`、`POSTGRES_PASSWORD`、`PUBLIC_BASE_URL` 必须显式提供
- 附件目录持久化到 volume
- 登录、注册、Flag 提交和后台关键写接口默认通过 Redis 做共享限流
- 仍保留 Docker Socket 挂载给动态题运行时使用

## 生产初始化流程

1. 准备环境变量：

```bash
export POSTGRES_PASSWORD='replace-with-strong-db-password'
export JWT_SECRET='replace-with-long-random-secret'
export PUBLIC_BASE_URL='https://ctf.example.edu'
export REDIS_PASSWORD=''
```

2. 启动生产依赖和应用：

```bash
make prod-compose-up
```

3. 应用数据库迁移：

```bash
docker compose -f deploy/docker-compose.prod.yml exec -T \
  -e DATABASE_URL="postgres://postgres:${POSTGRES_PASSWORD}@postgres:5432/ctf?sslmode=disable" \
  api /usr/local/bin/apply-migrations.sh
```

4. 显式创建首个管理员：

```bash
docker compose -f deploy/docker-compose.prod.yml exec -T \
  -e DATABASE_URL="postgres://postgres:${POSTGRES_PASSWORD}@postgres:5432/ctf?sslmode=disable" \
  -e BOOTSTRAP_ADMIN_USERNAME='admin' \
  -e BOOTSTRAP_ADMIN_EMAIL='admin@example.edu' \
  -e BOOTSTRAP_ADMIN_PASSWORD='replace-with-strong-admin-password' \
  -e BOOTSTRAP_ADMIN_DISPLAY_NAME='CTF Admin' \
  api /usr/local/bin/bootstrap-admin
```

说明：

- `bootstrap-admin` 是一次性初始化入口，不应在日常运维流程中反复执行
- 默认迁移不会再生成任何已知管理员口令
- `scripts/dev-seed.sh` 只用于本地开发，不能进入生产流程

## 限流配置

当前已接入 Redis 共享限流的入口：

- 登录
- 注册
- Flag 提交
- 后台关键写接口

当前可用环境变量：

- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `REDIS_DB`
- `REDIS_KEY_PREFIX`
- `LOGIN_RATE_LIMIT_WINDOW_SECONDS`
- `LOGIN_RATE_LIMIT_MAX`
- `REGISTER_RATE_LIMIT_WINDOW_SECONDS`
- `REGISTER_RATE_LIMIT_MAX`
- `SUBMISSION_RATE_LIMIT_WINDOW_SECONDS`
- `SUBMISSION_RATE_LIMIT_MAX`
- `ADMIN_WRITE_RATE_LIMIT_WINDOW_SECONDS`
- `ADMIN_WRITE_RATE_LIMIT_MAX`

说明：

- 生产环境建议保持 `REDIS_ADDR` 指向 Compose 内的 `redis:6379` 或专用 Redis 实例
- 若 Redis 不可用，API 会回退到进程内内存限流并记录日志，但这只适合作为临时降级手段

## 赛前彩排

彩排前至少确认：

- 已构建动态题镜像：`scripts/build-web-welcome-image.sh`
- API、数据库、Redis 和 Docker Engine 已启动
- 数据库已完成迁移
- 已存在可用管理员账号

最小彩排执行：

```bash
BASE_URL='http://127.0.0.1:8080' tests/smoke/smoke.sh
```

本地全自动彩排：

```bash
tests/smoke/run-local.sh
```

更详细的通过标准见：

- [tests/README.md](/home/calendar/code/ctf/tests/README.md)
- [tests/rehearsal/README.md](/home/calendar/code/ctf/tests/rehearsal/README.md)

## 反向代理与 TLS

当前仓库已提供：

- [deploy/nginx/default.conf](/home/calendar/code/ctf/deploy/nginx/default.conf)：网关层，将 `/api/` 转发到 API，其余流量转发到前端静态服务
- [deploy/nginx/frontend.conf](/home/calendar/code/ctf/deploy/nginx/frontend.conf)：前端静态资源服务配置

当前仍未完成：

- TLS 证书接入
- HSTS / 安全响应头
- 访问日志与错误日志落盘策略

## 当前限制

这套生产 Compose 是赛前最小骨架，不是最终完备方案。当前仍待补齐：

- 数据库备份与恢复手册
- 结构化日志和指标采集
- 动态题宿主机和主站流量的更严格隔离

具体优先级以 [开发基线与升级路线](/home/calendar/code/ctf/docs/development-baseline.md) 为准。
