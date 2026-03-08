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

## 用户接口

- `GET /me`
- `GET /me/submissions`
- `GET /me/solves`

## 动态实例接口

- `POST /challenges/{challengeID}/instances/me`
- `GET /challenges/{challengeID}/instances/me`
- `DELETE /challenges/{challengeID}/instances/me`
- `POST /challenges/{challengeID}/instances/me/renew`

## 提交接口

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
- 动态实例接口返回实例状态、访问地址和过期时间
- 是否已解题、是否有动态实例等派生状态由服务端计算返回
