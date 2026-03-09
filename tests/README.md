# Tests

该目录现在提供最小 smoke test 与赛前彩排入口。

## Smoke Test

文件：

- [tests/smoke/smoke.sh](/home/calendar/code/ctf/tests/smoke/smoke.sh)
- [tests/smoke/run-local.sh](/home/calendar/code/ctf/tests/smoke/run-local.sh)

覆盖链路：

- `GET /health`
- `GET /ready`
- `GET /metrics`
- 选手注册、登录、`GET /me`
- 公告、题目列表、题目详情、排行榜
- Flag 提交与解题入榜
- 动态实例启动、查询、续期
- 管理员登录、题目/提交/实例/用户/审计查询
- 管理员终止实例
- 数据库侧用户、解题、审计日志落库校验

## 本地一键执行

前置条件：

- 本机已安装 `docker`、`curl`、`psql`、`python3`
- 如已安装 `jq`，脚本会优先使用；否则自动回退到 `python3` 解析 JSON
- 本机 `5432` 和 `8080` 端口可用

执行：

```bash
tests/smoke/run-local.sh
```

该脚本会自动：

1. 构建 `ctf/web-welcome:dev` 镜像
2. 启动开发 PostgreSQL / Redis
3. 执行迁移和开发 seed
4. 本地启动 API
5. 执行完整 smoke test

## 对已运行环境执行

如果数据库、API 和动态题镜像已经准备好，可直接执行：

```bash
BASE_URL='http://127.0.0.1:8080' tests/smoke/smoke.sh
```

常用环境变量：

- `BASE_URL`
- `DB_HOST`
- `DB_PORT`
- `DB_NAME`
- `DB_USER`
- `DB_PASSWORD`
- `ADMIN_EMAIL`
- `ADMIN_PASSWORD`
- `PLAYER_USERNAME`
- `PLAYER_EMAIL`
- `PLAYER_PASSWORD`
- `REQUIRE_DYNAMIC_IMAGE=0|1`
- `CHECK_DB=0|1`

## 失败排查

- 若实例启动失败，先确认 `ctf/web-welcome:dev` 是否已构建
- 若登录管理员失败，先确认已执行 `scripts/dev-seed.sh` 或等价管理员 bootstrap
- 若 readiness 失败，先确认数据库已完成迁移且 API 能连接 PostgreSQL
- 若数据库校验失败，先确认脚本连接的是与 API 相同的数据库实例
