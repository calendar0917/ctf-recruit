# CTF Recruit Platform

面向单场 CTF 招新的比赛平台，当前仓库已经进入初始化开发阶段。

项目当前采用以下已确认方向：

- 平台定位：单场 CTF 招新比赛平台
- 比赛模式：个人赛
- 后端：Go
- 前端：Vite + React + TypeScript
- 数据库：PostgreSQL
- 部署模型：单机 Docker Compose
- 动态题目：首期纳入，按用户独立实例作为默认模型

## 当前状态

仓库已经具备首批工程基线：

- 产品范围、需求、架构和路线文档
- 动态实例设计草案
- 数据模型与 API 草案
- 后端最小可运行骨架
- 前端基础骨架
- Docker Compose 开发环境草案
- 动态题目模板示例

## 目录规划

```text
.
|-- backend/               # Go API、迁移、后端实现
|-- frontend/              # React 前端
|-- deploy/                # Compose、Nginx、部署说明
|-- docs/                  # 产品、架构、接口、数据模型文档
|-- challenges/            # 动态题目模板与示例资源
|-- scripts/               # 开发辅助脚本
`-- tests/                 # 集成测试或 E2E 预留目录
```

## 文档入口

- [项目范围](docs/scope.md)
- [需求梳理](docs/requirements.md)
- [技术方案](docs/architecture.md)
- [开发路线](docs/roadmap.md)
- [动态实例设计](docs/dynamic-instances.md)
- [数据模型草案](docs/data-model.md)
- [API 草案](docs/api.md)

## 快速开始

后端本地运行：

```bash
make backend-run
```

后端测试：

```bash
make backend-test
```

启动开发依赖：

```bash
docker compose -f deploy/docker-compose.yml up postgres redis api
```

说明：

- `frontend/` 已经建立基础骨架，但当前还没有执行依赖安装
- `api` 服务已经预留动态实例管理接口和后台清理任务入口
- 动态实例当前先按 `每个用户在每道题最多一个运行中实例` 设计

## 近期开发顺序

1. 完成数据库迁移与数据访问层
2. 完成认证、用户、题目和公告基础 API
3. 接入动态实例管理的真实 Docker 实现
4. 完成前端页面和管理后台基础视图
5. 打通 Flag 提交、排行榜和实例监控
