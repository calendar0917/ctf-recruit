# Capacity Record Template

每次执行基础压测后，至少补一条记录。建议把已填写版本放到团队内部运行手册或单独记录仓库中。

## 基本信息

- 日期：
- 执行人：
- commit id：
- 环境：开发 / 预发布 / 正式宿主机
- 宿主机规格：CPU / 内存 / 磁盘 / Docker 版本
- 数据库与 Redis 部署方式：同机 / 分离
- API 配置要点：限流窗口、实例清理周期、是否开启生产模式

## 执行命令

```bash
python3 tests/load/basic.py ...
```

## 场景摘要

- 场景：`public | player | submit | login`
- 并发：
- 时长：
- 目标题目：
- 是否复用单账号：是 / 否
- 是否保留真实限流：是 / 否

## 结果摘要

- 总请求数：
- RPS：
- 错误率：
- 平均延迟：
- p95：
- 最大延迟：
- 主要错误类型：

## 指标观察

- `ctf_http_requests_total` 增量：
- `ctf_http_request_duration_ms_total` 增量：
- `ctf_rate_limit_hits_total` 增量：
- `ctf_rate_limit_errors_total` 增量：
- 是否出现异常 `5xx`：

## 结论

- 是否通过本轮基线：
- 当前建议承载范围：
- 是否需要调大限流：
- 是否需要调优数据库 / Redis / Docker runtime：
- 是否需要在正式环境复测：

## 后续动作

1. 
2. 
3. 

