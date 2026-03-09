# Load Testing

本目录提供一套不依赖 `k6`、`wrk`、`vegeta` 等外部工具的基础压测入口。

目标不是替代专业性能平台，而是为当前仓库提供一个可直接落地、可复现、可记录的最小容量验证流程。

## 文件

- [tests/load/basic.py](/home/calendar/code/ctf/tests/load/basic.py)
- [tests/load/README.md](/home/calendar/code/ctf/tests/load/README.md)
- [tests/load/capacity-record-template.md](/home/calendar/code/ctf/tests/load/capacity-record-template.md)

## 覆盖场景

脚本内置 5 类场景：

- `public`：公告、题目列表、题目详情、排行榜
- `player`：公开读接口 + `GET /me`、`GET /me/submissions`、`GET /me/solves`
- `submit`：`player` 场景 + Flag 提交写路径
- `login`：重复登录请求，用于单独验证认证路径
- `instance`：动态实例创建、查询、续期、删除

说明：

- `submit` 和 `login` 场景默认把 `429` 视为可接受响应，因为它们会直接命中真实限流策略
- `instance` 场景默认会复用同一个选手账号，并在执行前预热一个活动实例；因此它更适合做运行时回归和低并发压力验证，而不是模拟大规模独立用户
- `instance` 场景结束后会主动尝试删除活动实例，避免在压测后遗留运行中的容器
- 如果目标是测纯吞吐能力，建议在专用环境中临时调高限流窗口和阈值，再执行压测
- 如果目标是验证比赛日安全姿势，应保留真实限流配置并把 `429` 命中率纳入结果解释

## 前置条件

- API 已启动，且 `BASE_URL` 可访问
- 数据库已完成迁移
- 至少存在公开题目 `web-welcome` 或通过 `--challenge-slug` 指定其他公开题目
- 对 `player`、`submit`、`login`、`instance` 场景，需要：
  - 提供现有选手账号 `--player-email`
  - 或使用 `--register-player` 在执行前创建一个一次性账号
- 对 `instance` 场景，还需要：
  - Docker Engine 可用
  - 动态题镜像已经构建，例如 `scripts/build-web-welcome-image.sh`

注意：

- 当前注册接口默认限流为 `5` 次 / `300s`，因此压测脚本不会在运行过程中批量注册账号
- 脚本默认复用单个选手账号或 Bearer Token，这适合当前项目的基线容量验证，不适合模拟大规模独立用户行为

## 快速执行

公共读流量：

```bash
python3 tests/load/basic.py \
  --base-url 'http://127.0.0.1:8080' \
  --scenario public \
  --concurrency 16 \
  --duration-seconds 60 \
  --output-dir tests/load/output/public-16x60
```

已认证读流量：

```bash
python3 tests/load/basic.py \
  --base-url 'http://127.0.0.1:8080' \
  --scenario player \
  --player-email 'player@example.com' \
  --player-password 'PlayerPass123!' \
  --concurrency 16 \
  --duration-seconds 60 \
  --output-dir tests/load/output/player-16x60
```

使用一次性账号做本地干跑：

```bash
python3 tests/load/basic.py \
  --base-url 'http://127.0.0.1:8080' \
  --scenario submit \
  --register-player \
  --concurrency 4 \
  --duration-seconds 20 \
  --output-dir tests/load/output/submit-dry-run
```

认证接口单压：

```bash
python3 tests/load/basic.py \
  --base-url 'http://127.0.0.1:8080' \
  --scenario login \
  --player-email 'player@example.com' \
  --player-password 'PlayerPass123!' \
  --concurrency 8 \
  --duration-seconds 30 \
  --output-dir tests/load/output/login-8x30
```

动态实例低并发压测：

```bash
python3 tests/load/basic.py \
  --base-url 'http://127.0.0.1:8080' \
  --scenario instance \
  --register-player \
  --concurrency 2 \
  --duration-seconds 20 \
  --output-dir tests/load/output/instance-2x20
```

## 常用参数

- `--base-url`：API 根地址，默认读取 `BASE_URL`，否则为 `http://127.0.0.1:8080`
- `--scenario`：`public | player | submit | login | instance`
- `--concurrency`：并发 worker 数
- `--duration-seconds`：执行时长
- `--timeout-seconds`：单请求超时
- `--challenge-slug`：目标题目，默认 `web-welcome`
- `--flag`：`submit` 场景提交的 flag，默认 `flag{welcome}`
- `--player-email` / `--player-password`：复用现有选手账号
- `--player-token`：直接复用 Bearer Token，跳过登录准备步骤
- `--register-player`：执行前注册一个唯一账号并自动登录
- `--output-dir`：输出目录，包含报告 JSON 和前后指标快照
- `--report-json`：额外把报告写到指定 JSON 文件
- `--max-error-rate`：若整体错误率超过阈值则返回非零退出码
- `--max-p95-ms`：若整体 p95 延迟超过阈值则返回非零退出码
- `--no-metrics`：不抓取 `/api/v1/metrics` 快照

## 输出内容

脚本会在标准输出打印：

- 总请求数
- 每秒请求数近似值
- 成功数 / 失败数 / 错误率
- 平均延迟 / p95 / 最大延迟
- 各接口维度统计
- 错误分布
- `/api/v1/metrics` 前后快照差值摘要

如果设置了 `--output-dir`，还会写出：

- `report.json`
- `metrics.before.txt`
- `metrics.after.txt`

## 推荐执行顺序

对于当前仓库的基线验证，建议至少记录以下 4 组结果：

1. `public`：确认公开读路径的基础吞吐与延迟
2. `player`：确认已认证常用读取路径在登录态下的表现
3. `submit` 或 `login`：确认写路径或认证路径在真实限流配置下不会异常退化
4. `instance`：确认动态实例链路在低并发下不会出现显著异常或失控延迟

## 结果解释

建议重点看：

- 是否出现大量 `5xx`
- `429` 是否符合预期而不是异常放大
- `instance` 场景里是否出现异常 `5xx`、持续 `404` 或大量非预期 `409`
- `p95` 是否随并发上升快速失控
- `/api/v1/metrics` 中 `ctf_http_requests_total`、`ctf_http_request_duration_ms_total`、`ctf_rate_limit_hits_total`、`ctf_rate_limit_errors_total` 的增量是否符合场景预期

## 基线建议

针对当前 `500 ~ 1000` 注册量、`100 ~ 250` 峰值在线的目标，赛前至少应完成以下记录：

1. 单机环境 `public` 场景 `16` 并发、`60s`
2. 单机环境 `player` 场景 `16` 并发、`60s`
3. 单机环境 `submit` 或 `login` 场景 `8` 并发、`30s`
4. 单机环境 `instance` 场景 `2 ~ 4` 并发、`20 ~ 30s`
5. 贴近正式部署配置的环境中，再做一轮更接近目标宿主机规格的重复验证

这些数字只是最小建议，不是最终容量承诺。

## 记录要求

每次正式记录结果时，请同步填写：

- [tests/load/capacity-record-template.md](/home/calendar/code/ctf/tests/load/capacity-record-template.md)

