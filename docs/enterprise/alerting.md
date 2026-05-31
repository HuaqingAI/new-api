# 企业内容风险告警

## 范围

本文档描述企业内容风险子系统的当前实现，包括：

- 风险事件事实源
- 告警规则与通道配置
- 告警投递状态与手动重发
- 部门风险概览
- enterprise scheduler 中的告警任务

实现对应代码主要位于：

- `model/enterprise/alert_event.go`
- `model/enterprise/alert_rule.go`
- `model/enterprise/alert_delivery.go`
- `service/enterprise/alert.go`
- `service/enterprise/alert_dispatch.go`
- `service/enterprise/alert_dispatch_task.go`
- `controller/enterprise/alert.go`
- `router/enterprise-router.go`

## 核心事实源

### 风险事件

- 表：`enterprise_alert_events`
- 写入入口：`RecordRiskEvent(...)`
- 当前已接通主路径：`controller/relay.go` 的敏感词命中分支
- 关键约束：
  - 旁路写入，不阻塞 relay 正常错误返回
  - 仅保存追溯所需摘要，不保存原始 prompt 或完整敏感内容
  - 保存请求发生时的部门快照，而不是查询时再 join 当前成员关系

部门快照字段：

- `department_snapshot`：JSON 文本
- `department_tokens`：部门 ID token 文本，用于“包含某部门”语义筛选

## 规则与通道

### 告警规则

- 表：`enterprise_alert_rules`
- API：
  - `GET /api/enterprise/alerts/rules`
  - `GET /api/enterprise/alerts/rules/:id`
  - `PUT /api/enterprise/alerts/rules`
  - `DELETE /api/enterprise/alerts/rules/:id`

规则字段当前包括：

- `name`
- `enabled`
- `risk_types`
- `department_ids`
- `channel_configs`
- `dedupe_window_seconds`

当前支持的通道类型：

- `email`
- `webhook`
- `dingtalk_robot`

## 投递状态

### 告警投递

- 表：`enterprise_alert_deliveries`
- API：
  - `GET /api/enterprise/alerts/deliveries`
  - `POST /api/enterprise/alerts/deliveries/:id/resend`

关键字段：

- `status`：`pending` / `sent` / `failed` / `final_failed` / `resent`
- `attempt_count`
- `max_attempts`
- `next_retry_at`
- `error_reason`
- `dedupe_key`
- `trace_payload`
- `trigger_source`
- `manual_parent_id`

设计语义：

- `trace_payload` 保存事件、规则和 drill-down 上下文，便于管理员追溯
- `trigger_source` 区分自动规则匹配和手动重发
- 手动重发不会覆盖原记录，而是追加新的投递记录并保留链路

## 部门风险概览

- API：`GET /api/enterprise/alerts/department-summary`
- 必填参数：`from`、`to`
- 可选参数：`tenant_id`、`summary_sort`、`summary_order`
- 时间语义：`[from, to)`

统计口径：

- 风险分子：`enterprise_alert_events`
- 风险分母：`enterprise_usage_snapshots`
- 多部门成员请求：在每个相关部门重复计入
- 未归属请求：单独显示为“未归属”，不伪造部门 ID
- disclaimer：继续复用 `enterprise.usage.multi_dept_disclaimer`

风险率公式：

- `风险率 = 该部门成员触发风险事件的请求数 ÷ 该部门成员请求总数`

## 后台任务

enterprise 告警相关任务由 `service/enterprise/scheduler.go` 统一启动：

- `StartEnterpriseTasks()` 在 master 节点启动
- maintenance ticker：余额回收、钱包同步、用量聚合、用量报告
- alert-dispatch ticker：告警规则匹配、发送、重试与最终失败收口

这意味着：

- 告警投递不会在每个请求路径中直接发送
- 多节点部署时只有 master 节点执行告警 dispatch，避免重复发送

## API 清单

- `GET /api/enterprise/alerts/events`
- `GET /api/enterprise/alerts/department-summary`
- `GET /api/enterprise/alerts/deliveries`
- `POST /api/enterprise/alerts/deliveries/:id/resend`
- `GET /api/enterprise/alerts/rules`
- `GET /api/enterprise/alerts/rules/:id`
- `PUT /api/enterprise/alerts/rules`
- `DELETE /api/enterprise/alerts/rules/:id`

## 当前已知限制

- 目前风险事件主写入路径仍以敏感词拦截为主，其他过滤规则接入点有待继续扩展。
- 定向自动化较完整，但仓库级全量 Go 回归仍受环境问题影响。
- 人类可读文档刚补齐，后续若调整状态机或通道类型，需要同步更新本文档与 OpenAPI。
