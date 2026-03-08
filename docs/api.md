# API 文档

统一前缀：`/api/v1`

本文档描述当前已实现的主要 API 面。

## 公共接口

- `GET /api/v1/health`
- `GET /api/v1/ready`
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

## 管理接口

这些接口当前要求带有管理员权限的 Bearer Token：

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

- 比赛生命周期控制接口
- 更细粒度的权限模型
- 与 `flag_type` 对应的更丰富判题接口语义
