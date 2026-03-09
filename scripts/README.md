# Scripts

该目录用于存放开发辅助脚本，例如：

- 数据库迁移执行脚本
- 数据初始化脚本
- 本地联调脚本
- 数据库备份与恢复脚本

当前关键脚本：

- [apply-migrations.sh](/home/calendar/code/ctf/scripts/apply-migrations.sh)
- [bootstrap-admin.sh](/home/calendar/code/ctf/scripts/bootstrap-admin.sh)
- [dev-seed.sh](/home/calendar/code/ctf/scripts/dev-seed.sh)
- [import-challenges.sh](/home/calendar/code/ctf/scripts/import-challenges.sh)
- [backup-db.sh](/home/calendar/code/ctf/scripts/backup-db.sh)
- [restore-db.sh](/home/calendar/code/ctf/scripts/restore-db.sh)

## 题目导入

`scripts/import-challenges.sh` 会调用后端导入器，把 `challenge.yaml` 中的题目元数据、附件清单和 runtime 配置幂等同步到数据库与附件目录。

导入全部模板：

```bash
export DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/ctf?sslmode=disable'
export ATTACHMENT_STORAGE_DIR='/tmp/ctf-attachments'
scripts/import-challenges.sh --contest recruit-2025 --root challenges
```

导入单个模板：

```bash
export DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/ctf?sslmode=disable'
export ATTACHMENT_STORAGE_DIR='/tmp/ctf-attachments'
scripts/import-challenges.sh --contest recruit-2025 --path challenges/templates/web-welcome/challenge.yaml
```

说明：

- 默认同步题目基础信息、`challenge_attachments` 和 `challenge_runtime_configs`
- 推荐在模板中显式设置 `meta.status`；旧模板中的 `meta.visible` 仍会兼容映射到 `published` 或 `draft`
- 不负责构建镜像，也不负责上传公告或富文本题面资源
- 若 `slug` 已存在，会按模板内容覆盖更新对应题目配置与附件记录
