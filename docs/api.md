# API 草案

统一前缀：`/api/v1`

## 公共接口

- `GET /health`
- `GET /ready`
- `POST /auth/register`
- `POST /auth/login`
- `GET /announcements`
- `GET /challenges`
- `GET /challenges/{challengeID}`
- `GET /scoreboard`

## 已认证用户接口

- `GET /me`
- `GET /me/submissions`
- `GET /me/solves`

## 动态实例接口

这些接口当前要求 `Authorization: Bearer <token>`：

- `POST /challenges/{challengeID}/instances/me`
- `GET /challenges/{challengeID}/instances/me`
- `DELETE /challenges/{challengeID}/instances/me`
- `POST /challenges/{challengeID}/instances/me/renew`

## 提交接口

这些接口当前要求 `Authorization: Bearer <token>`：

- `POST /challenges/{challengeID}/submissions`

## 管理接口

- `GET /admin/users`
- `PATCH /admin/users/{userID}`
- `GET /admin/challenges`
- `POST /admin/challenges`
- `PATCH /admin/challenges/{challengeID}`
- `POST /admin/challenges/{challengeID}/attachments`
- `PUT /admin/challenges/{challengeID}/runtime-config`
- `GET /admin/instances`
- `POST /admin/instances/{instanceID}/terminate`
- `GET /admin/submissions`
- `GET /admin/announcements`
- `POST /admin/announcements`

## 当前约定

- 成功响应统一返回 JSON
- 错误响应统一包含 `error` 与 `message`
- 注册和登录接口返回 `token`、`expires_at` 和 `user`
- `GET /me` 返回当前登录用户信息
- `GET /challenges/{challengeID}` 返回题目详情与附件元数据
- `POST /challenges/{challengeID}/submissions` 返回提交结果、是否首次解题和得分
- 动态实例接口返回实例状态、访问地址和过期时间
