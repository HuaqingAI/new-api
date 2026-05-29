# 部门用量聚合

## 默认规则

- 默认聚合窗口为 1 小时，快照表为 `enterprise_usage_snapshots`。
- 聚合 watermark 存在 `options` 表，键名格式为 `EnterpriseUsageAggregationWatermark:{tenant_id}`。
- 后台任务只处理“已经结束的完整小时窗口”；如果 watermark 不存在，会从最早一条未聚合消费所在的完整小时窗口开始回填，再按 watermark 顺序追赶历史缺口。

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
