# API 文档

统一前缀：`/api/v1`

## 约定

- 所有成功响应为 JSON（附件下载除外）。
- 所有错误响应为 JSON：`{"error":"<code>","message":"<human-readable>"}`。
- 需要认证的接口必须携带：`Authorization: Bearer <token>`。
- 路由中出现的 `{challengeID}` 在玩家侧接口中既可为数字 ID，也可为 slug。
  - 注意：当参数是纯数字时，服务端会优先按 ID 匹配。
- 时间字段统一使用 RFC3339（UTC）字符串。
- `GET /api/v1/challenges` 返回的 `items[].id` 为字符串；而 `GET /api/v1/challenges/{challengeID}` 返回的 `challenge.id` 为数字。
  - 前端建议把“挑战引用”统一当作字符串处理。

## 快速对接（前端从零开发建议）

1. 先调用 `GET /api/v1/contest`，根据 `phase.*` 控制页面能力（未开放时直接禁用/隐藏入口）。
2. 展示题库：`GET /api/v1/challenges`（注意 `items[].id` 是字符串）。
3. 进入题面：`GET /api/v1/challenges/{challengeID}`（`{challengeID}` 可用上一步的 `id` 字段或 slug）。
4. 登录/注册：
   - 注册：`POST /api/v1/auth/register`（可能返回 `registration_closed` 或 `register_rate_limited`）
   - 登录：`POST /api/v1/auth/login`（可能返回 `login_rate_limited` 或 `invalid_credentials`）
5. 提交 Flag：`POST /api/v1/challenges/{challengeID}/submissions`（可能返回 `submission_closed` 或 `submission_rate_limited`）。
6. 动态实例（如果题目 `dynamic=true`）：
   - 创建：`POST /api/v1/challenges/{challengeID}/instances/me`
   - 查询：`GET /api/v1/challenges/{challengeID}/instances/me`
   - 续期：`POST /api/v1/challenges/{challengeID}/instances/me/renew`
   - 回收：`DELETE /api/v1/challenges/{challengeID}/instances/me`

建议：前端统一把服务端错误 `error` 当作稳定错误码处理（用 code 映射友好提示），不要依赖 `message` 文案做逻辑分支。

本文档描述当前已实现的主要 API 面。

## 公共接口

- `GET /api/v1/health`
- `GET /api/v1/ready`
- `GET /api/v1/metrics`
- `GET /api/v1/contest`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/announcements`
- `GET /api/v1/challenges`
- `GET /api/v1/challenges/{challengeID}`
- `GET /api/v1/challenges/{challengeID}/attachments/{attachmentID}`
- `GET /api/v1/scoreboard`

说明：

- 附件下载当前仍是公开路由，但仅允许下载当前可见题目的附件

## 公共接口返回结构

### `GET /api/v1/contest`

返回当前单场比赛信息和阶段判定。

响应：

```json
{
  "contest": {
    "id": 1,
    "slug": "recruit-2025",
    "title": "CTF Recruit 2025",
    "description": "Initial contest seed",
    "status": "running",
    "starts_at": null,
    "ends_at": null
  },
  "phase": {
    "status": "running",
    "announcement_visible": true,
    "challenge_list_visible": true,
    "challenge_detail_visible": true,
    "attachment_visible": true,
    "scoreboard_visible": true,
    "submission_allowed": true,
    "runtime_allowed": true,
    "registration_allowed": true,
    "starts_at": null,
    "ends_at": null,
    "message": "比赛进行中，题目、提交和排行榜已开放。"
  }
}
```

说明：

- `contest.status` 可能取值：`draft | upcoming | running | frozen | ended`。
- `phase.*` 用于前端做能力开关（例如未开放时隐藏或禁用提交/实例等）。

### `GET /api/v1/announcements`

响应：

```json
{
  "items": [
    {
      "id": 1,
      "title": "Welcome",
      "content": "...",
      "pinned": true,
      "published_at": "2026-03-14T00:00:00Z"
    }
  ]
}
```

### `GET /api/v1/challenges`

说明：该接口返回的是 runtime 的挑战摘要（用于题库列表）。

响应：

```json
{
  "items": [
    {
      "id": "1",
      "slug": "web-welcome",
      "title": "Web Welcome",
      "category": "web",
      "points": 100,
      "difficulty": "easy",
      "dynamic": true
    }
  ]
}
```

字段说明：

- `id` 为字符串（兼容 slug/ID 统一引用）。
- `dynamic` 表示是否允许创建动态实例。

### `GET /api/v1/challenges/{challengeID}`

响应：

```json
{
  "challenge": {
    "id": 1,
    "slug": "web-welcome",
    "title": "Web Welcome",
    "category": "web",
    "points": 100,
    "difficulty": "easy",
    "description": "...",
    "dynamic": true,
    "attachments": [
      {
        "id": 1,
        "filename": "challenge.zip",
        "content_type": "application/zip",
        "size_bytes": 12345
      }
    ]
  }
}
```

说明：

- `flag_value` 不会在公共接口返回（只在管理端返回）。
- `flag_type` 当前会在题目详情接口返回（用于前端提示/校验），未来如需隐藏可再调整。

### `GET /api/v1/scoreboard`

响应：

```json
{
  "items": [
    {
      "rank": 1,
      "user_id": 1,
      "username": "player",
      "display_name": "Player",
      "score": 100,
      "last_solve_at": "2026-03-14T00:00:00Z",
      "solves": [
        {
          "challenge_id": 1,
          "challenge_slug": "web-welcome",
          "challenge_title": "Web Welcome",
          "category": "web",
          "difficulty": "easy",
          "awarded_points": 100,
          "solved_at": "2026-03-14T00:00:00Z"
        }
      ]
    }
  ]
}
```

## 已认证用户接口

这些接口当前要求 `Authorization: Bearer <token>`：

- `GET /api/v1/me`
- `GET /api/v1/me/submissions`
- `GET /api/v1/me/solves`
- `POST /api/v1/challenges/{challengeID}/instances/me`
- `GET /api/v1/challenges/{challengeID}/instances/me`
- `DELETE /api/v1/challenges/{challengeID}/instances/me`
- `POST /api/v1/challenges/{challengeID}/instances/me/renew`
- `POST /api/v1/challenges/{challengeID}/submissions`

## 认证接口返回结构

### `POST /api/v1/auth/register`

请求：

```json
{"username":"player","email":"player@example.com","password":"...","display_name":"Player"}
```

响应：

```json
{
  "token": "<jwt>",
  "expires_at": "2026-03-15T00:00:00Z",
  "user": {
    "id": 2,
    "role": "player",
    "username": "player",
    "email": "player@example.com",
    "display_name": "Player",
    "status": "active",
    "last_login_at": "2026-03-14T00:00:00Z"
  }
}
```

### `POST /api/v1/auth/login`

请求：

```json
{"identifier":"player@example.com","password":"..."}
```

响应同注册接口。

### `GET /api/v1/me`

响应：

```json
{"user":{"id":2,"role":"player","username":"player","email":"player@example.com","display_name":"Player","status":"active","last_login_at":"2026-03-14T00:00:00Z"}}
```


### 认证与提交限流错误语义

以下接口在限流命中时会返回 `429 Too Many Requests`：

- `POST /api/v1/auth/register` -> `register_rate_limited`
- `POST /api/v1/auth/login` -> `login_rate_limited`
- `POST /api/v1/challenges/{challengeID}/submissions` -> `submission_rate_limited`
- 后台关键写接口 -> `admin_rate_limited`

说明：

- 当前默认使用 Redis 维护共享限流状态
- 当 Redis 不可用时，服务会回退到进程内内存限流

### 动态实例错误语义

动态实例相关接口在冲突场景下会返回 `409 Conflict`，并使用稳定错误码区分原因：

- `challenge_not_dynamic`：题目不是动态题
- `runtime_config_missing`：题目已标记为动态题，但运行配置不完整
- `instance_not_found`：当前用户在该题下没有活动实例
- `instance_renew_limit_reached`：实例已达到最大续期次数
- `instance_capacity_reached`：题目已达到配置的总并发实例上限
- `instance_cooldown_active`：用户仍处于该题实例创建冷却期内
- `instance_port_exhausted`：实例端口池已耗尽（需要运维扩容端口段或回收实例）

### 动态实例接口返回结构

#### `POST /api/v1/challenges/{challengeID}/instances/me`

响应（201 创建或 200 复用）：

```json
{
  "challenge_id": "1",
  "status": "running",
  "access_url": "http://inst.yulinsec.cn:20000",
  "host_port": 20000,
  "renew_count": 0,
  "started_at": "2026-03-14T00:00:00Z",
  "expires_at": "2026-03-14T01:00:00Z",
  "terminated_at": null
}
```

#### `GET /api/v1/challenges/{challengeID}/instances/me`

响应结构同上。

#### `POST /api/v1/challenges/{challengeID}/instances/me/renew`

响应结构同上。

#### `DELETE /api/v1/challenges/{challengeID}/instances/me`

响应结构同上。

说明：

- `POST /api/v1/challenges/{challengeID}/instances/me` 会先检查用户现有活动实例，再检查题目并发上限与用户冷却时间
- 管理端题目运行配置中的 `max_active_instances` 和 `user_cooldown_seconds` 会直接影响上述接口行为

### 比赛生命周期接口

- `GET /api/v1/contest` 返回当前单场比赛信息和阶段判定
- 当比赛处于 `draft` 或 `upcoming` 时，公开题目、题目详情、附件、排行榜会按阶段规则关闭
- 当比赛处于 `frozen` 或 `ended` 时，Flag 提交与动态实例创建/续期/查询会按阶段规则关闭

阶段限制错误码：

- `contest_not_public`
- `scoreboard_not_public`
- `submission_closed`
- `runtime_closed`
- `registration_closed`

## 管理接口

这些接口当前要求带有后台权限的 Bearer Token：

- `GET /api/v1/admin/contest`
- `PATCH /api/v1/admin/contest`
- `GET /api/v1/admin/challenges`
- `POST /api/v1/admin/challenges`
- `GET /api/v1/admin/challenges/{challengeID}`
- `PATCH /api/v1/admin/challenges/{challengeID}`
- `GET /api/v1/admin/challenges/{challengeID}/authors`
- `PUT /api/v1/admin/challenges/{challengeID}/authors`
- `POST /api/v1/admin/challenges/{challengeID}/attachments`
- `GET /api/v1/admin/announcements`
- `POST /api/v1/admin/announcements`
- `DELETE /api/v1/admin/announcements/{announcementID}`
- `GET /api/v1/admin/submissions`
- `GET /api/v1/admin/instances`
- `POST /api/v1/admin/instances/{instanceID}/terminate`
- `GET /api/v1/admin/users`
- `PATCH /api/v1/admin/users/{userID}`
- `GET /api/v1/admin/audit-logs`

## 管理接口返回结构（节选）

### `GET /api/v1/admin/contest`

响应结构同 `GET /api/v1/contest`。

### `PATCH /api/v1/admin/contest`

请求：

```json
{"status":"running","starts_at":"2026-03-01T00:00:00Z","ends_at":"2026-03-31T00:00:00Z"}
```

说明：

- `starts_at` 与 `ends_at` 当前为字符串字段（由后端解析）。
- 建议使用 RFC3339 格式（UTC），例如：`2026-03-01T00:00:00Z`。

响应结构同 `GET /api/v1/contest`。

### `GET /api/v1/admin/challenges`

响应：

```json
{"items":[{"id":1,"slug":"web-welcome","title":"Web Welcome","category":"web","points":100,"status":"published","visible":true,"dynamic_enabled":true}]}
```

### `POST /api/v1/admin/challenges`

请求：

```json
{
  "slug": "web-welcome",
  "title": "Web Welcome",
  "category_slug": "web",
  "description": "...",
  "points": 100,
  "difficulty": "easy",
  "flag_type": "static",
  "flag_value": "flag{...}",
  "dynamic_enabled": true,
  "status": "published",
  "visible": true,
  "sort_order": 10,
  "runtime_config": {
    "enabled": true,
    "image_name": "ctf/web-welcome:dev",
    "exposed_protocol": "http",
    "container_port": 80,
    "default_ttl_seconds": 3600,
    "max_renew_count": 3,
    "memory_limit_mb": 256,
    "cpu_limit_millicores": 500,
    "max_active_instances": 100,
    "user_cooldown_seconds": 30,
    "env": {"KEY":"VALUE"},
    "command": ["/bin/sh","-lc","..."]
  }
}
```

响应：

```json
{"challenge":{"id":1,"slug":"web-welcome","title":"Web Welcome","category":"web","points":100,"status":"published","visible":true,"dynamic_enabled":true}}
```

### `POST /api/v1/admin/challenges/{challengeID}/attachments`

请求：`multipart/form-data`，字段名必须为 `file`。

响应：

```json
{"attachment":{"id":1,"filename":"challenge.zip","content_type":"application/zip","size_bytes":12345}}
```

## 当前约定

- 成功响应统一返回 JSON
- 错误响应统一包含 `error` 与 `message`
- 注册和登录接口返回 `token`、`expires_at` 和 `user`
- `GET /api/v1/me` 返回当前登录用户信息
- `GET /api/v1/challenges/{challengeID}` 返回题目详情与附件元数据
- `POST /api/v1/challenges/{challengeID}/submissions` 返回提交结果、是否首次解题和得分
- 当前 `flag_type` 已支持 `static`、`case_insensitive`、`regex` 三种判题策略
- 动态实例接口返回实例状态、访问地址和过期时间
- 管理接口当前已覆盖题目、附件、公告、提交记录、实例、用户和审计日志的基础能力
- `author` 角色在 `GET/POST/PATCH /api/v1/admin/challenges`、`GET /api/v1/admin/challenges/{challengeID}/authors` 与 `POST /api/v1/admin/challenges/{challengeID}/attachments` 上会被限制为仅操作自己负责的题目，未归属题目统一返回 `404 challenge_not_found`
- `PUT /api/v1/admin/challenges/{challengeID}/authors` 当前仅允许 `admin` 调用，用于维护题目负责人集合

## 后续计划中但尚未完成的能力

- 更细粒度的权限模型
- 超出 `static`、`case_insensitive`、`regex` 的更复杂判题语义
