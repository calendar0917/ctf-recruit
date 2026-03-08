# 技术方案

## 已确认技术栈

### 后端

- Go 1.26+
- 首轮先用标准库 `net/http`
- 数据访问层后续接入 `sqlc + PostgreSQL`
- JWT 作为鉴权方案

选择理由：

- 先用标准库把骨架、路由和生命周期跑通，初始化成本最低
- 后续接入 `sqlc` 时不会推翻当前目录结构
- Go 适合同时处理 HTTP、后台清理任务和 Docker 交互

### 前端

- Vite
- React
- TypeScript

选择理由：

- 初始化成本低，适合快速搭建选手端和管理端
- 后续接入路由、表单、请求库都很直接

### 数据层

- PostgreSQL 16
- Redis 7 作为可选缓存、限流和短期状态存储
- 文件存储初期使用本地磁盘

### 动态题目运行

- Docker Engine
- 后端通过 Docker Socket 与 Engine 交互
- 单机部署阶段使用端口映射暴露题目实例

## 推荐最小架构

```text
Browser
  -> Frontend
  -> API
    -> PostgreSQL
    -> Redis(optional)
    -> Docker Engine
      -> Challenge Containers
```

## 动态实例基线设计

当前默认设计如下：

- 实例粒度：按 `用户 + 题目` 独立分配
- 实例数量：每个用户在同一题目上只允许一个运行中实例
- 生命周期：默认 `30m`
- 访问方式：单机阶段先使用宿主机端口映射
- 清理方式：API 内置后台 sweeper 定时回收过期实例
- 记录方式：数据库维护 `challenge_runtime_configs` 与 `challenge_instances`

## 系统拆分

### `frontend`

- 前台比赛页面
- 管理后台页面
- 动态实例状态展示和控制入口

### `api`

- 认证
- 比赛与题目管理
- Flag 提交
- 排行榜
- 动态实例生命周期管理
- 后台清理任务

### `postgres`

- 用户、题目、提交、公告、实例元数据

### `redis`

- 二阶段再引入，用于限流、缓存和临时态

## 安全设计基线

- 密码哈希存储
- JWT 设置过期时间
- 管理接口做 RBAC
- 提交接口加限流
- 动态容器禁止 privileged
- 动态容器限制 CPU、内存和 TTL
- 动态容器使用标签标识归属用户和题目
- API 对容器状态做服务端校验，不直接信任前端

## 当前不做的事

- 一开始拆微服务
- 一开始上 Kubernetes
- 一开始引入消息队列
- 过早抽象复杂题目调度系统
