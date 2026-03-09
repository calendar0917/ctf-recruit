# 数据模型

当前核心数据结构定义以实际迁移文件为准：

- [0001_initial_schema.sql](/home/calendar/code/ctf/backend/migrations/0001_initial_schema.sql)
- 后续结构与公开示例迁移位于 [backend/migrations](/home/calendar/code/ctf/backend/migrations)
- 开发态默认管理员种子已迁移到 [scripts/dev-seed.sh](/home/calendar/code/ctf/scripts/dev-seed.sh)，不再属于默认初始化必经路径

## 核心实体

### `contests`

平台当前按单场比赛优先设计，但仍保留比赛实体，避免后续硬编码比赛状态。

### `users`

保存选手和后台账号信息，角色通过 `roles` 关联。当前基础角色包括 `player`、`author`、`ops`、`admin`。

### `categories`

保存题目分类，例如 `web`、`pwn`、`misc`、`crypto`。

### `challenges`

保存题目基本信息、分值、开放状态、Flag 校验方式与是否需要动态实例。

### `challenge_attachments`

保存附件文件元数据。

### `challenge_runtime_configs`

保存动态实例题目的运行配置，是当前模型里的关键表。

### `challenge_instances`

保存按 `用户 + 题目` 分配的实例记录。

### `submissions`

保存 Flag 提交记录。

### `solves`

保存正确解题记录和得分结果。

### `announcements`

保存比赛公告。

### `audit_logs`

保存关键后台操作和运维动作。

## 当前关键约束

- `users.username` 唯一
- `users.email` 唯一
- `roles.name` 唯一
- `categories.slug` 唯一
- `challenges.slug` 唯一
- `solves` 对 `user_id + challenge_id` 唯一
- `challenge_instances` 对 `user_id + challenge_id` 的运行中实例做唯一限制

## 后续演进方向

- 团队模式可在后续增加 `teams`、`team_members`、`team_solves`
- 长期训练模式可在后续支持多个长期 contest
- 比赛生命周期控制应与 `contests` 的状态字段一起落地
