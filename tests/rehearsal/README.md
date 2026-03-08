# Rehearsal

本目录记录赛前彩排的最小执行标准。

## 目标

在接近真实部署的环境中，验证以下关键链路：

- API 与数据库联通
- 默认公开内容可访问
- 选手注册与登录可用
- Flag 提交、解题和排行榜更新正常
- 动态实例启动、续期、终止正常
- 管理员后台查询与实例干预正常

## 最小彩排步骤

1. 准备环境变量、数据库、Redis、Docker Engine 和动态题镜像
2. 按部署文档启动服务
3. 执行 [tests/smoke/smoke.sh](/home/calendar/code/ctf/tests/smoke/smoke.sh)
4. 记录执行时间、失败点和关键日志
5. 清理测试账号、异常实例和临时数据

## 彩排通过标准

- smoke test 全部通过
- `GET /api/v1/ready` 返回数据库连通
- 后台能看到 smoke 账号提交与实例记录
- 管理员终止实例后，选手侧实例查询返回 `404 instance_not_found`
- 宿主机上没有遗留的 smoke 动态容器

## 彩排后必须记录

- 部署版本或 commit id
- 执行时间与环境说明
- 是否使用开发 seed 或生产 bootstrap 管理员
- 失败问题、修复方式与复验结果
