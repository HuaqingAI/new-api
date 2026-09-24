# 企业账号标识与展示契约

本文冻结企业治理、预算、用量、风险和告警链路中的账号标识语义，避免后续迭代把展示值、当前身份和历史快照混用。

## 字段语义

| 字段 | 所属链路 | 语义 | 可否作为查询筛选值 | rename 后行为 |
| --- | --- | --- | --- | --- |
| `user_id` | 当前身份、日志、风险、预算、治理 | 用户表主键，是稳定身份锚点 | 可以。用于精确定位当前用户 | 不变 |
| `username` | 当前身份、日志筛选、风险筛选 | 当前用户表用户名，或日志模块中的用户名筛选语义 | 可以。最近日志入口、风险 username 筛选和现有日志查询继续使用该语义 | 当前身份读侧随用户表变化；历史日志中的 `logs.username` 不回写 |
| `display_name` | 当前身份展示 | 当前用户表展示名，只用于 UI 主展示 | 不可以 | 当前身份读侧随用户表变化 |
| `username_snapshot` | 风险事件 | 风险事件写入时的用户名快照 | 不可以作为当前身份筛选值；风险 username 筛选仍使用 `username` 参数语义 | 不回写、不重算 |
| `department_snapshot` | 风险事件、告警投递 trace | 风险事件写入时的部门归属快照 | 不可以 | 不回写、不重算 |
| `target_username` | 预算 wallet/allocation、治理目标 | 目标用户当前 username | 可以在需要定位当前目标用户时配合 `target_user_id` 使用 | 当前身份读侧随用户表变化 |
| `target_display_name` | 预算 wallet/allocation、治理目标 | 目标用户当前展示名 | 不可以 | 当前身份读侧随用户表变化 |
| `requester_display_name` | 额度申请 | 申请人当前展示名 | 不可以 | 当前身份读侧随用户表变化 |
| `trace.display_name` | 告警投递 trace | trace 中事件用户的当前展示名 | 不可以 | 当前身份读侧随用户表变化；trace 的事件、部门快照不因此回写 |
| recent logs `user_options` | 用量详情最近日志入口 | UI 下拉/快捷入口展示当前 `user_id`、`username`、`display_name` | 传参只能使用其中的 `username` 或 `user_id` 语义；不得传 `display_name` | 选项展示随当前身份变化，历史统计快照不回写 |

## 当前身份、历史快照和展示值

- 当前身份字段来自 `service/enterprise/user_lookup.go` 的 `loadCurrentUserIdentities`，包含 `user_id`、`username` 和 `display_name`。
- UI 主展示必须按 `display_name -> username -> user_id` 兜底。辅助信息只能展示真实存在的 `username` 或 `user_id`，不得生成 `#-`、空 ID 或把展示名伪装成用户名。
- `display_name` 只用于展示。任何日志、风险、预算、用量和治理筛选入口都不得把 `display_name` 作为查询参数传给后端。
- `logs.username` 是日志写入时的用户名语义。用户 rename 后，当前治理视图可以通过 `user_id` 展示新的 `username` / `display_name`，但不得批量回写历史日志。
- 风险事件写入时的 `event.Username` / `username_snapshot` 和 `department_snapshot` 是历史快照。用户 rename 或部门调整后，风险列表可以展示当前身份字段，但不得回写或重算历史风险事件、投递 trace 或用量快照。

## 查询边界

- 最近日志入口继续传递 `recent_logs_entry.filters.username`、`username_options` 或 `user_options[].username` 表示的 username 语义。
- 风险事件筛选继续使用 `username` / `user_id` 参数；`username_snapshot` 只用于展示历史事件写入时的用户名。
- 现有日志模块查询仍使用 `username` / `user_id` 语义，不接受 `display_name` 作为等价筛选值。
- 后续新增企业 DTO 或 UI 查询入口时，如果需要当前用户展示，必须同时保留底层筛选字段和展示字段的区别。

## 回归护栏

后续修改以下区域时必须保留上述契约：

- 企业 DTO 与 OpenAPI 响应字段。
- `service/enterprise/user_lookup.go`、用量详情、风险事件、预算 wallet/allocation、治理时间线、额度申请列表。
- Default 与 Classic 的用户展示 helper。
- 最近日志、风险 username 筛选和任何现有日志模块查询入口。
