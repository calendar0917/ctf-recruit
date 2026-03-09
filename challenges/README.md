# Challenges

该目录用于存放动态题目模板、示例镜像和资源文件。

当前示例：

- `templates/web-welcome/`：一个最小的 Web 动态题模板

建议每个动态题模板至少包含：

- `challenge.yaml`：题目元数据、Flag 配置与运行配置
- `Dockerfile`：题目镜像构建定义
- 题目运行所需静态资源

## challenge.yaml 约定

当前导入器支持的结构：

```yaml
meta:
  slug: web-welcome
  title: Welcome Panel
  category: web
  points: 100
  difficulty: easy
  dynamic: true
  visible: true
  sort_order: 10

flag:
  type: static
  value: flag{welcome}

content:
  description: A minimal seeded web challenge for local runtime integration.
  author: platform

runtime:
  image: ctf/web-welcome:dev
  mode: per-user
  expose: http
  container_port: 80
  ttl: 30m
  memory_limit_mb: 256
  cpu_limit_millicores: 500
  max_renew_count: 1
  max_active_instances: 0
  user_cooldown: 0s
  env:
    MODE: dev
  command:
```

当前限制：

- `runtime.mode` 仅支持 `per-user`
- `flag.type` 当前仅支持 `static`、`case_insensitive`、`regex`
- 导入器只同步题目主信息和 runtime 配置，不处理附件上传
- 镜像构建仍需单独执行，例如 `scripts/build-web-welcome-image.sh`

## 导入方式

将模板同步到数据库：

```bash
export DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/ctf?sslmode=disable'
scripts/import-challenges.sh --contest recruit-2025 --root challenges
```

只导入单个模板：

```bash
export DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/ctf?sslmode=disable'
scripts/import-challenges.sh --contest recruit-2025 --path challenges/templates/web-welcome/challenge.yaml
```

说明：

- 导入是幂等的；若 `slug` 已存在，会更新题目主信息和 runtime 配置
- 目标比赛和分类必须已存在，否则导入会失败
- 该流程用于减少手工录入题目、镜像和 runtime 参数的重复工作
