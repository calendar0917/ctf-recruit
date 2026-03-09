# API 文档

统一前缀：`/api/v1`

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

这些接口当前要求带有管理员权限的 Bearer Token：

- `GET /api/v1/admin/contest`
- `PATCH /api/v1/admin/contest`
- `GET /api/v1/admin/challenges`
- `POST /api/v1/admin/challenges`
- `GET /api/v1/admin/challenges/{challengeID}`
- `PATCH /api/v1/admin/challenges/{challengeID}`
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

## 当前约定

- 成功响应统一返回 JSON
- 错误响应统一包含 `error` 与 `message`
- 注册和登录接口返回 `token`、`expires_at` 和 `user`
- `GET /api/v1/me` 返回当前登录用户信息
- `GET /api/v1/challenges/{challengeID}` 返回题目详情与附件元数据
- `POST /api/v1/challenges/{challengeID}/submissions` 返回提交结果、是否首次解题和得分
- 动态实例接口返回实例状态、访问地址和过期时间
- 管理接口当前已覆盖题目、附件、公告、提交记录、实例、用户和审计日志的基础能力

## 后续计划中但尚未完成的能力

- 更细粒度的权限模型
- 与 `flag_type` 对应的更丰富判题接口语义
