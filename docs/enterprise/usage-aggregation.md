# 部门用量聚合

## 默认规则

- 默认聚合窗口为 1 小时，快照表为 `enterprise_usage_snapshots`。
- 聚合 watermark 存在 `options` 表，键名格式为 `EnterpriseUsageAggregationWatermark:{tenant_id}`。
- 后台任务只处理“已经结束的完整小时窗口”；如果 watermark 不存在，会从最早一条未聚合消费所在的完整小时窗口开始回填，再按 watermark 顺序追赶历史缺口。
- enterprise 后台任务统一通过 `service/enterprise/scheduler.go` 在 `common.IsMasterNode` 下运行；用量聚合与定期报告都由同一 enterprise scheduler 驱动。

## 归因口径

- 聚合仅从 `logs` 表读取 `LogTypeConsume` 明细，不修改 relay / consume 链路，也不向 `logs` 回写任何部门字段。
- 用户一条消费命中多个有效部门成员关系时，会在每个部门各计一次。
- 成员关系按消费发生时的 `joined_at` / `left_at` 生效区间归因；`pending` 状态不计入，有历史区间的 `active` / `inactive` / `left` 记录会按区间参与归因。
- 没有任何有效成员关系的消费进入 `dept_id = NULL` 桶，并统一显示为 `dept_name = "未归属"`。

## 幂等与回填

- 同一窗口重算前会先删除该窗口已有快照，再按最新归因结果整窗重写；不会保留陈旧部门桶，也不会把同一批消费重复累计。
- summary API 只读取 `enterprise_usage_snapshots`，不会回退扫描 `logs`。
- 快照内部额外保存窗口级去重用户集合，用于 summary 跨多窗口合并时正确计算 `user_count`，避免把同一用户在多个小时窗口重复累加。
- 历史回填使用 service 层的 `AggregateWindow` 顺序补窗；建议按窗口连续补齐，再推进 watermark，避免在线查询回退到明细表。

## API 合同

### `GET /api/enterprise/usage/department-summary`

- 必填参数：`from`、`to`
- 可选参数：`tenant_id`、`summary_sort`、`summary_order`
- 时间过滤语义为 `[from, to)`；summary 只汇总 `window_end <= to` 的完整快照窗口。
- `summary_sort` 当前支持 `requests`、`quota`、`users`、`dept_name`；`summary_order` 支持 `asc`、`desc`。
- 返回结构中的 `model_distribution` 始终为数组；未归属部门返回 `dept_id = null`、`dept_name = "未归属"`。

### `GET /api/enterprise/usage/department-detail`

- 必填参数：`dept_id`、`from`、`to`
- 可选参数：`tenant_id`
- 返回部门汇总、用户排行、模型分布、按小时趋势，以及 `recent_logs_entry` drill-down 上下文。
- `recent_logs_entry` 当前输出 `/usage-logs/common` 入口、时间范围、来源部门信息和 `username_options`，用于最近消费日志联动，不改变日志后端语义。

### `GET /api/enterprise/usage/export`

- 必填参数：`from`、`to`
- 可选参数：`tenant_id`、`summary_sort`、`summary_order`
- 当前实现为同步文件流响应，不返回任务 ID。
- CSV 首行固定写入“部门间不可加和”说明，列顺序固定为：
  `部门 ID, 部门名称, 父部门, 周期开始, 周期结束, 请求数, prompt_tokens, completion_tokens, quota, 用户数`
- 导出排序直接复用 summary 服务排序结果，避免页面与 CSV 各自维护一套排序逻辑。

## 定期报告

- 配置接口为 `GET /api/enterprise/usage/reports` 与 `PUT /api/enterprise/usage/reports`。
- 配置字段当前包括：`receivers`、`frequency`（`daily` / `weekly` / `monthly`）、`range_type`（`today` / `last7d` / `last30d`）、`enabled`、可选 `tenant_id`。
- 状态源为 `enterprise_usage_report_jobs`，同一租户只维护一条配置 + 多次执行状态，不为每次发送新建平行任务表。
- 报告任务复用 summary 聚合结果，邮件正文固定包含与页面/CSV 一致的“部门间不可加和”说明。
- 发送失败只更新 job 状态、`error_reason`、`failure_count`、`next_run_at` 等字段，不阻塞聚合任务、看板查询或 CSV 导出。
