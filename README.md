# CTF Recruit MVP（Docker 运行手册）

这是一个 CTF 招募系统 MVP，默认通过 Docker Compose 一键启动整套服务。

## 服务组成

- `frontend`：Next.js 前端
- `backend`：Go(Fiber) API 服务
- `worker`：异步任务/实例清理工作进程
- `postgres`：主数据库
- `redis`：缓存/队列依赖
- `migrate`：一次性迁移任务（自动执行 `backend/migrations/*.up.sql`）

## 默认端口

- 前端：`http://localhost:13001`
- 后端：`http://localhost:18080`
- 健康检查：`http://localhost:18080/api/v1/health`

说明：项目不会占用宿主机 `3000` 端口（容器内前端是 `3000`，映射到宿主机 `13001`）。

## 快速启动

```bash
# 1) 检查 compose 配置
docker compose config

# 2) 构建并启动
docker compose up -d --build

# 3) 查看状态
docker compose ps

# 4) 健康检查
curl -i http://localhost:18080/api/v1/health
```

期望：健康检查返回 `HTTP 200` 且响应包含 `"status":"ok"`。

### 自定义端口

```bash
BACKEND_PORT=28080 FRONTEND_PORT=23001 docker compose up -d --build
```

## 停止与清理

```bash
# 停止服务（保留数据卷）
docker compose down

# 停止并清空数据库卷（完全重置）
docker compose down -v --remove-orphans
```

## 初始化种子数据（管理员/选手/示例题）

> 推荐在全新环境中执行，可复现实例生命周期流程。

```bash
docker compose run --rm \
  -e DATABASE_URL="postgres://postgres:postgres@postgres:5432/ctf_recruit?sslmode=disable" \
  -e SEED_ADMIN_EMAIL="admin@ctf.local" \
  -e SEED_ADMIN_PASSWORD="AdminPass123!" \
  -e SEED_PLAYER_EMAIL="player@ctf.local" \
  -e SEED_PLAYER_PASSWORD="PlayerPass123!" \
  backend /bin/seed
```

默认会创建：
- 管理员账号 `admin@ctf.local`
- 选手账号 `player@ctf.local`
- 两道示例题，其中 `Log Trail` 为 `dynamic` 模式（含 runtime 配置）

更多 seed 细节见 `backend/README.seed.md`。

## 实例生命周期最小验证（可复制）

### 1) 玩家登录并导出 Token

```bash
PLAYER_LOGIN_JSON="$(curl -fsS -X POST http://localhost:18080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"player@ctf.local","password":"PlayerPass123!"}')"
printf '%s\n' "$PLAYER_LOGIN_JSON"

export PLAYER_TOKEN="$(printf '%s' "$PLAYER_LOGIN_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["accessToken"])')"
```

### 2) 读取 `Log Trail` 挑战 ID

```bash
PLAYER_CHALLENGES_JSON="$(curl -fsS http://localhost:18080/api/v1/challenges \
  -H "Authorization: Bearer $PLAYER_TOKEN")"

export LOG_TRAIL_ID="$(printf '%s' "$PLAYER_CHALLENGES_JSON" | python3 -c 'import json,sys; items=json.load(sys.stdin)["items"]; print(next(i["id"] for i in items if i["title"]=="Log Trail"))')"
```

### 3) 启动实例（期望 `201`，`status=running`）

```bash
START_JSON="$(curl -fsS -X POST http://localhost:18080/api/v1/instances/start \
  -H "Authorization: Bearer $PLAYER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"challengeId\":\"$LOG_TRAIL_ID\"}")"
printf '%s\n' "$START_JSON"

export INSTANCE_ID="$(printf '%s' "$START_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"
```

### 4) 查询当前实例

```bash
curl -fsS http://localhost:18080/api/v1/instances/me \
  -H "Authorization: Bearer $PLAYER_TOKEN"
```

### 5) 停止实例（期望 `200`，返回 `cooldownUntil`）

```bash
curl -fsS -X POST http://localhost:18080/api/v1/instances/stop \
  -H "Authorization: Bearer $PLAYER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"instanceId\":\"$INSTANCE_ID\"}"
```

### 6) 立即重启验证冷却（期望 `409`）

```bash
curl -sS -X POST http://localhost:18080/api/v1/instances/start \
  -H "Authorization: Bearer $PLAYER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"challengeId\":\"$LOG_TRAIL_ID\"}"
```

期望错误码：`INSTANCE_COOLDOWN_ACTIVE`，并包含 `error.details.retryAt`。

## 常用开发命令

在仓库根目录执行：

```bash
pnpm lint
pnpm type-check
pnpm test
pnpm test:coverage
```

说明：这些脚本会同时检查前端和后端。

## 常见问题

### 1) 端口冲突

优先检查 `18080` 和 `13001` 是否已被占用；必要时使用 `BACKEND_PORT`/`FRONTEND_PORT` 覆盖。

### 2) `/instances/start` 返回 `INSTANCE_RUNTIME_START_FAILED`

通常是容器内无法访问 Docker daemon：

```bash
docker compose ps
docker compose logs backend --tail 200
docker compose logs worker --tail 200
docker compose exec backend docker version
docker compose exec worker docker version
```

若异常，重建运行时相关服务：

```bash
docker compose up -d --build backend worker
```

### 3) 冷却期看起来“消失”

`/api/v1/instances/me` 在无活动实例时可能返回：
- `{ "instance": null }`
- 或 `{ "instance": null, "cooldown": { "retryAt": "..." } }`

只要 `retryAt` 未到，就会继续阻止 `start`。

## 补充说明

- 根目录已忽略构建产物与依赖目录（如 `node_modules`、`frontend/.next` 等）。
- 若你刚修改过提交历史（例如 `git commit --amend`），推送到远端前请确认分支状态：

```bash
git status
git log --oneline -n 3
```
