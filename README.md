# CTF Recruit Platform

面向单场校内 CTF 招新、允许校外访问的比赛平台。

当前项目已经完成基础业务闭环，接下来进入“生产化加固与上线准备”阶段，而不是继续停留在初始化脚手架阶段。

## 当前定位

- 平台定位：单场 CTF 招新比赛平台
- 比赛模式：个人赛
- 访问范围：校内为主，可开放校外访问
- 目标规模：`500 ~ 1000` 注册量
- 后端：Go
- 前端：Vite + React + TypeScript
- 数据库：PostgreSQL
- 部署模型：单机优先，动态题基于 Docker Engine

## 当前已具备的能力

- 注册、登录、Bearer Token 鉴权、`GET /me`
- 公告列表、题目列表、题目详情、附件下载、Flag 提交、排行榜
- 用户提交记录、解题记录查询
- 动态实例启动、查询、删除、续期、后台过期清理
- 管理端题目、附件、公告、提交记录、实例、用户、审计日志基础能力
- Docker Compose 开发环境和生产部署骨架
- 后端单元测试基础覆盖

## 文档入口

优先阅读：

- [开发基线与升级路线](docs/development-baseline.md)

补充文档：

- [项目范围](docs/scope.md)
- [需求梳理](docs/requirements.md)
- [技术方案](docs/architecture.md)
- [开发路线](docs/roadmap.md)
- [动态实例设计](docs/dynamic-instances.md)
- [数据模型](docs/data-model.md)
- [API 文档](docs/api.md)
- [部署说明](deploy/README.md)

## 目录规划

```text
.
|-- backend/               # Go API、迁移、后端实现
|-- frontend/              # React 前端
|-- deploy/                # Compose、Nginx、部署说明
|-- docs/                  # 产品、架构、接口、数据模型文档
|-- challenges/            # 动态题模板与示例资源
|-- scripts/               # 开发辅助脚本
`-- tests/                 # 集成测试与 E2E 预留目录
```

## 快速开始

后端本地运行：

```bash
make backend-run
```

后端测试：

```bash
cd backend && GOCACHE=/tmp/ctf-go-build GOMODCACHE=/tmp/ctf-go-mod go test ./...
```

前端开发：

```bash
cd frontend && pnpm install && pnpm dev
```

启动开发依赖：

```bash
docker compose -f deploy/docker-compose.yml up postgres redis api
```

开发环境如需临时关闭 Redis 共享限流，可将 API 的 `REDIS_ADDR` 设为空，此时会自动回退到进程内内存限流。

动态实例公网访问说明：

- 动态实例的对外访问地址由 `RUNTIME_PUBLIC_BASE_URL` 决定，默认回退到 `PUBLIC_BASE_URL`
- 如需通过固定端口池暴露实例（便于防火墙和 frp 端口段转发），可设置：`RUNTIME_PORT_MIN`、`RUNTIME_PORT_MAX`
- 端口绑定地址可通过 `RUNTIME_BIND_ADDR` 控制（开发/内网场景通常使用 `127.0.0.1`）

初始化数据库结构与公开示例数据：

```bash
export DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/ctf?sslmode=disable'
scripts/apply-migrations.sh
```

如需本地开发默认管理员账号，再显式执行开发 seed：

```bash
export DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/ctf?sslmode=disable'
scripts/dev-seed.sh
```

## 生产部署骨架

最小生产部署入口：

```bash
export POSTGRES_PASSWORD='replace-with-strong-db-password'
export JWT_SECRET='replace-with-long-random-secret'
export PUBLIC_BASE_URL='https://ctf.example.edu'
export RUNTIME_PUBLIC_BASE_URL='http://inst.example.edu'
export RUNTIME_PORT_MIN='20000'
export RUNTIME_PORT_MAX='20499'
export RUNTIME_BIND_ADDR='127.0.0.1'
export REDIS_PASSWORD=''
make prod-compose-up
```

迁移数据库并创建首个管理员：

```bash
docker compose -f deploy/docker-compose.prod.yml exec -T \
  -e DATABASE_URL="postgres://postgres:${POSTGRES_PASSWORD}@postgres:5432/ctf?sslmode=disable" \
  api /usr/local/bin/apply-migrations.sh

docker compose -f deploy/docker-compose.prod.yml exec -T \
  -e DATABASE_URL="postgres://postgres:${POSTGRES_PASSWORD}@postgres:5432/ctf?sslmode=disable" \
  -e BOOTSTRAP_ADMIN_USERNAME='admin' \
  -e BOOTSTRAP_ADMIN_EMAIL='admin@example.edu' \
  -e BOOTSTRAP_ADMIN_PASSWORD='replace-with-strong-admin-password' \
  api /usr/local/bin/bootstrap-admin
```

说明：

- `scripts/apply-migrations.sh` 不再创建默认管理员账号，避免生产式初始化路径自动带出已知口令
- `scripts/dev-seed.sh` 仅用于本地开发，会创建 `admin@ctf.local / Admin123!`
- API 在非 `development` 环境下会拒绝空值、`change-me` 和开发态默认 `JWT_SECRET`
- `deploy/docker-compose.yml` 是开发环境；`deploy/docker-compose.prod.yml` 是当前最小生产骨架
- 生产环境首个管理员必须通过显式 bootstrap 命令创建
- 登录、注册、Flag 提交和后台关键写接口当前已接入 Redis 共享限流
- API 当前已提供 `GET /api/v1/metrics`、数据库备份恢复脚本和 `tests/load/basic.py` 基线压测入口
- 接下来的开发优先级以 [开发基线与升级路线](docs/development-baseline.md) 为准
