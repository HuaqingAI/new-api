---
baseline_commit: 86e7bd31f2a61d7c8b0899b89c880731367d5b48
---

# Story 5.4: 发送可追溯告警通知

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 安全/合规负责人,
I want 收到包含追溯信息的风险告警,
so that 我可以直接定位后台事件详情。

## Acceptance Criteria

1. **Given** 风险事件命中启用的告警规则  
   **When** 告警投递任务执行  
   **Then** 通知内容至少包含部门、用户、模型、时间、风险类型、请求追踪 ID  
   **And** 提供能定位后台事件详情的追溯信息。
2. **Given** 告警通道返回失败  
   **When** 系统执行重试  
   **Then** 按默认策略重试 3 次：30 秒、2 分钟、10 分钟  
   **And** 3 次后标记为最终失败并保留错误原因。
3. **Given** 同一风险事件在 5 分钟内重复触发同一规则  
   **When** 系统准备发送通知  
   **Then** 聚合为一次告警发送  
   **And** 避免重复通知噪声。

## Tasks / Subtasks

- [x] 新增告警投递模型与迁移，建立 `enterprise_alert_deliveries` 作为 FR19 的单一状态表 (AC: 1, 2, 3)
  - [x] 新增 `model/enterprise/alert_delivery.go`，表名固定为 `enterprise_alert_deliveries`，至少覆盖：`id`、`tenant_id`、`event_id`、`rule_id`、`channel_type`、`status`、`attempt_count`、`max_attempts`、`next_retry_at`、`last_attempt_at`、`sent_at`、`final_failed_at`、`error_reason`、`dedupe_key`、`trace_payload`、`trigger_source`、`manual_parent_id`、`created_at`、`updated_at`。其中 `trace_payload` 使用 `TEXT` + `common.Marshal` / `common.UnmarshalJsonStr`，不要使用 `JSONB`、数组列或 DB 专属 JSON 操作。[Source: _bmad-output/planning-artifacts/architecture.md#840; _bmad-output/planning-artifacts/architecture.md#481; AGENTS.md#rule-1-json-package--use-commonjsongo; AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
  - [x] `status` 至少显式区分 `pending`、`sent`、`failed`、`final_failed`、`resent`，与 PRD 的“待发送/发送成功/发送失败/最终失败/已重发”一一对应；不要把 5.5 的列表页面延后到这里才补状态枚举，否则后续无法可靠查询。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-19-发送可追溯告警通知]
  - [x] `dedupe_key` 必须可由 `tenant_id + event_id + rule_id + 聚合窗口桶` 稳定推导，并建立唯一索引或等价幂等约束，保证同一事件在 5 分钟窗口内不会生成多条并发待发记录。不要依赖 Redis 作为唯一去重正确性来源。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#347; _bmad-output/planning-artifacts/architecture.md#1089]
  - [x] 在 `model/enterprise/migration.go` 注册 `AlertDelivery` 迁移，并对已存在库采用与 `department_tokens` 相同的跨库安全增量模式；不要引入 `ALTER COLUMN` 或单库专属 DDL。[Source: model/enterprise/migration.go; AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]

- [x] 在 `service/enterprise/alert.go` 与新增的 `alert_dispatch.go` 中实现“规则匹配 + 幂等建单 + 追溯载荷”边界，不重复造第二套风险事实源 (AC: 1, 3)
  - [x] 继续以 `enterprise_alert_events` 和 `enterprise_alert_rules` 为输入事实源：由 `service/enterprise/alert.go` 提供可复用入口，例如 `EnqueueAlertDeliveriesForPendingEvents(...)`、`MatchAlertRules(...)` 或等价方法，把规则匹配、部门范围过滤、5 分钟聚合判定与投递记录创建封装在 service 层；不要把匹配逻辑散落到 scheduler、controller 或前端。[Source: service/enterprise/alert.go; _bmad-output/implementation-artifacts/5-1-record-content-risk-events.md#Dev Notes; _bmad-output/implementation-artifacts/5-3-configure-content-risk-alert-rules-and-channels.md#Dev Notes]
  - [x] 规则命中语义必须基于 Story 5.1 保存的部门快照和 Story 5.3 保存的规则白名单条件：`risk_type` 精确匹配、`department_ids` 为空表示全局、非空则只要事件快照中包含任一部门即命中。不要回查当前 `enterprise_user_departments` 替代事件快照。[Source: _bmad-output/implementation-artifacts/5-1-record-content-risk-events.md#Dev Notes; _bmad-output/implementation-artifacts/5-2-query-and-filter-risk-events.md#Dev Notes; _bmad-output/implementation-artifacts/5-3-configure-content-risk-alert-rules-and-channels.md#Tasks--Subtasks]
  - [x] `trace_payload` 至少固化通知渲染所需字段：事件 ID、request ID、tenant_id、用户名、模型名、风险类型、动作结果、事件时间、部门快照摘要、事件摘要、规则名、规则 ID、推荐后台查看路径或可反查的 API 线索。不得把原始 prompt、敏感词原串、Webhook secret、Robot secret 或完整凭据写入 payload。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#352; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#421; _bmad-output/test-artifacts/test-design-progress.md#R-002]
  - [x] 对于同一事件命中多条规则或多种通道，投递粒度应为“每条规则、每个通道一条 delivery 记录”；邮件、Webhook、钉钉机器人之间不能共享一条状态记录，否则后续 5.5 无法区分通道级失败原因。[Source: _bmad-output/planning-artifacts/architecture.md#1154; _bmad-output/planning-artifacts/architecture.md#397]

- [x] 新增 `service/enterprise/alert_dispatch.go`，复用现有通知能力发送邮件/Webhook/钉钉机器人，并输出可追溯通知内容 (AC: 1, 2)
  - [x] 邮件通道继续复用 `common.SendEmail`；Webhook 复用 `service.SendWebhookNotify`；钉钉机器人优先复用仓库现有 webhook/notify 能力而不是新写 HTTP 客户端。企业告警模块只负责构造 `dto.Notify` 或等价通知载荷、接收通道返回结果、并写 `enterprise_alert_deliveries` 状态。[Source: _bmad-output/planning-artifacts/architecture.md#397; _bmad-output/planning-artifacts/architecture.md#1142; _bmad-output/planning-artifacts/architecture.md#1143; common/email.go; service/webhook.go; service/user_notify.go]
  - [x] 通知标题与内容至少包含：部门、用户、模型、发生时间、风险类型、请求追踪 ID、事件摘要，以及“到后台查看事件详情”的明确追溯信息。追溯信息可以是后台路由说明、管理 API 路径或稳定事件 ID，但必须让安全负责人不看数据库也能定位。[Source: _bmad-output/planning-artifacts/epics.md#Story 5.4: 发送可追溯告警通知; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#352]
  - [x] 多部门事件的通知内容必须遵守 Epic 5 的 N:N 语义：展示事件快照中的全部相关部门，且不要把其压扁成“主部门”。未归属事件可显示为空列表或“未归属”，但不能伪造部门 ID。[Source: _bmad-output/planning-artifacts/architecture.md#66; _bmad-output/implementation-artifacts/5-1-record-content-risk-events.md#Dev Notes]
  - [x] 日志只允许输出 `delivery_id`、`event_id`、`rule_id`、`channel_type`、`attempt_count`、摘要化错误原因等最小字段；不得打印邮件地址全集、secret、完整 webhook query string、DingTalk 签名串或完整通知正文。[Source: _bmad-output/planning-artifacts/architecture.md#393; service/enterprise/admin_action.go; _bmad-output/test-artifacts/test-design-progress.md#ENT-SEC-P0-002]

- [x] 新增 `service/enterprise/alert_dispatch_task.go` 并把任务接入 `scheduler.go`，严格按后台 job + 状态表模式执行重试 (AC: 2, 3)
  - [x] `alert_dispatch_task` 由 `service/enterprise/scheduler.go` 在 `common.IsMasterNode` 守卫下统一调度；不要单独起新的 `main.go` ticker，也不要在请求链路同步发送通知。[Source: _bmad-output/planning-artifacts/architecture.md#385; _bmad-output/planning-artifacts/architecture.md#849; service/enterprise/scheduler.go]
  - [x] 每轮任务至少完成两件事：先扫描近期风险事件并按规则创建缺失的 `pending` 投递，再拉取到期的 `pending/failed` 投递执行发送。这样可以保证事件落库仍是旁路，而通知由后台异步补齐。[Source: _bmad-output/planning-artifacts/architecture.md#1154; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#349]
  - [x] 重试策略必须严格固定为 3 次退避：第 1 次失败后 `next_retry_at = +30s`，第 2 次失败后 `+2m`，第 3 次失败后 `+10m`；超过后写成 `final_failed` 并持久化 `error_reason`。不要把“3 次”理解成“总共 3 次含首次发送”或随意调整间隔。[Source: _bmad-output/planning-artifacts/epics.md#Story 5.4: 发送可追溯告警通知; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#347]
  - [x] 调度与投递必须具备幂等性：重复执行同一 tick、进程重启或 Redis 不可用时，最多产生重复检查，不得把同一 dedupe window 内的相同 delivery 重复写成多条成功发送。[Source: _bmad-output/planning-artifacts/architecture.md#1089; _bmad-output/test-artifacts/test-design-progress.md#R-006]

- [x] 为 5.5 预留查询/人工重发所需 API 与 DTO 基础，但不要提前交付完整投递管理页面 (AC: 1, 2)
  - [x] 在 `dto/enterprise/alert.go` 增补 delivery 相关响应 DTO，例如 `AlertDeliveryItem`、`AlertDeliveriesResponse`、最小状态枚举和只读追溯字段；可选标量仍使用指针 + `omitempty` 规则，避免未来手动重发请求的 `false/0` 被吞掉。[Source: AGENTS.md#rule-6-upstream-relay-request-dtos--preserve-explicit-zero-values; dto/enterprise/alert.go]
  - [x] 在 `controller/enterprise/alert.go` 与 `router/enterprise-router.go` 至少预留 `GET /api/enterprise/alerts/deliveries` 的只读查询入口，便于 5.5 直接复用；controller 仍只做 tenant scope、参数绑定与错误映射，不在这里执行投递本体。[Source: _bmad-output/planning-artifacts/architecture.md#840; router/enterprise-router.go; controller/enterprise/alert.go]
  - [x] `docs/openapi/api.json` 必须同步加入 `Enterprise / Alerts` 下的 delivery 合同；不要写入 `docs/openapi/relay.json`。5.4 只提供最小查询与追溯字段，不提前加入“人工重发” mutation，避免侵入 5.5 范围。[Source: _bmad-output/planning-artifacts/architecture.md#611; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#349]

- [x] Default 与 Classic 前端只补“可追溯通知结果入口”的最小增量，不抢 5.5 的交互深度 (AC: 1)
  - [x] Default 继续在 `web/default/src/features/enterprise-alerts/` 内扩展，而不是新开平行模块。可接受的 5.4 最小 UI 增量包括：在现有 `Risk Events / Alert Rules` tabs 旁新增只读 `Deliveries` tab，或在事件/规则卡片中展示最近投递状态与追溯入口，但不要在 5.4 直接做完整失败列表筛选和手动重发工作流。[Source: web/default/src/features/enterprise-alerts/index.tsx; _bmad-output/planning-artifacts/architecture.md#347; _bmad-output/planning-artifacts/architecture.md#508]
  - [x] Classic 在 `web/classic/src/pages/Enterprise/Alerts.js` 中补最小只读状态区域或 tab 即可，保持与当前 `events + rules` 结构一致；不要在 Classic 先发明独立投递页面。[Source: web/classic/src/pages/Enterprise/Alerts.js; _bmad-output/planning-artifacts/architecture.md#351]
  - [x] 新增前端文案继续同步到 Default 六语种 locale；Classic 维持其现有中英文 locale 风格。任何 “Pending / Sent / Final failed / Trace details” 等状态文案都必须走 i18n，不要硬编码。[Source: AGENTS.md#internationalization-i18n]

- [x] 补齐模型、任务、API、前端测试，锁定去重、重试和隐私边界 (AC: 1, 2, 3)
  - [x] 新增 `model/enterprise/alert_delivery_test.go`，覆盖迁移成功、TEXT JSON 归一化、唯一 `dedupe_key` 约束语义、状态默认值、以及三库兼容 schema 行为。[Source: model/enterprise/alert_rule_test.go; model/enterprise/alert_event_test.go; AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
  - [x] 扩展 `service/enterprise/alert_test.go` 或新增 `alert_dispatch_test.go` / `alert_dispatch_task_test.go`，覆盖：规则命中创建 delivery、5 分钟内重复事件不重复建单、多通道各自产生记录、30s/2m/10m 退避、最终失败保留错误原因、以及通知失败不反向影响风险事件落库。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#347; _bmad-output/implementation-artifacts/5-1-record-content-risk-events.md#Completion Notes List]
  - [x] 为 `controller/enterprise/alert_test.go` 与 `tests/api/enterprise_alerts_test.go` 增加 delivery 查询接口测试，覆盖管理员可访问、普通用户/部门管理员拒绝、tenant 作用域正确、返回字段不含 secret、以及状态枚举/错误原因稳定可读。[Source: tests/api/enterprise_alerts_test.go; controller/enterprise/alert_test.go; middleware/enterprise_admin.go]
  - [x] 为 Default `enterprise-alerts` 增加投递状态 query key / 页面 helper 测试，至少覆盖状态映射、空态、追溯字段展示与“不显示 secret/原始敏感内容”的断言；Classic 至少补一条 smoke test，确认新增投递只读区不会破坏现有事件/规则页面。[Source: web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx; web/classic/src/pages/Enterprise/Alerts.smoke.test.js]

## Dev Notes

- Story 5.4 的真实交付物是“告警投递后端链路”，不是再做一遍规则配置，也不是把 5.5 的投递管理/人工重发 UI 提前塞进来。5.3 已经交付 `enterprise_alert_rules` 和现有 events/rules 页面；5.4 必须围绕 `alert_delivery.go`、`alert_dispatch.go`、`alert_dispatch_task.go`、scheduler 接线和最小 delivery 查询面推进。[Source: _bmad-output/implementation-artifacts/5-3-configure-content-risk-alert-rules-and-channels.md#Dev Notes; _bmad-output/planning-artifacts/architecture.md#538; _bmad-output/planning-artifacts/architecture.md#557]
- 当前仓库可以确认的现状是：`service/enterprise/alert.go` 已具备事件记录与规则 CRUD，但还没有 `alert_delivery.go`、`alert_dispatch.go`、`alert_dispatch_task.go`，`service/enterprise/scheduler.go` 也尚未挂接任何告警任务。这意味着 5.4 的开发重点是新增文件与扩展 scheduler，而不是微调 controller。[Source: service/enterprise/alert.go; service/enterprise/scheduler.go; model/enterprise/migration.go]
- `enterprise_alert_events` 是旁路写入、不阻塞 relay 的既定事实源；这一点在 5.4 不能被破坏。通知发送、失败重试和最终失败都必须建立在事件已成功落库之后的异步链路上，绝不能因为邮件/Webhook/机器人失败而影响原请求响应。[Source: _bmad-output/planning-artifacts/architecture.md#1154; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#352; _bmad-output/implementation-artifacts/5-1-record-content-risk-events.md#Dev Notes]
- 通知能力要复用仓库已有实现。邮件走 `common.SendEmail`，Webhook 走 `service.SendWebhookNotify`，DingTalk 机器人优先作为 webhook 风格通道复用，不要在 enterprise 包里重新实现 HTTP client、签名算法、notify limit 或 SSRF 防护逻辑。[Source: common/email.go; service/webhook.go; service/user_notify.go; _bmad-output/planning-artifacts/architecture.md#397]
- 5 分钟去重的正确性来源必须落在 DB 状态表和幂等键，而不是仅靠内存或 Redis。架构已经明确 Redis 只是增强，故障时最多允许额外扫描/重复尝试，不允许写出多条本应聚合的待发记录。[Source: _bmad-output/planning-artifacts/architecture.md#1089]
- PRD 在 FR19 中同时提到了“最终失败可查询”和“人工重发”。本故事只负责把 delivery 状态表、状态枚举和最小查询入口打通，为 5.5 的列表/手动重发做基础；不要在 5.4 直接实现完整 resend mutation 或复杂管理页，以免故事边界混乱。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#347; _bmad-output/planning-artifacts/epics.md#Story 5.5: 管理告警发送状态和人工重发]
- 现有 Default `enterprise-alerts` feature 和 Classic `Enterprise/Alerts.js` 都只覆盖 events/rules，没有 deliveries。5.4 的前端最稳妥增量是补一个只读 delivery 视图或追溯摘要入口，保持用户可以看到“通知是否已发、失败原因是什么”，但不在本故事内做完整投递运维台。[Source: web/default/src/features/enterprise-alerts/index.tsx; web/classic/src/pages/Enterprise/Alerts.js]
- 测试设计文档已经把告警投递列为 P1：要求重试节奏、最终失败可查询、手动重发新建一轮 delivery。5.4 至少要覆盖其中属于本故事范围的前两项，并把第三项需要的字段/契约设计好，否则 5.5 会被迫返工状态模型。[Source: _bmad-output/test-artifacts/test-design-progress.md#ENT-ALERT-P1-002; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#349]
- 本次 story 创建继续采用 skill fallback：`resolve_customization.py` 因 Python 版本不足未能执行，已手工读取 `.agents/skills/bmad-create-story/customize.toml` 与 `_bmad/bmm/config.yaml`，且仓库中未发现 `project-context.md` 额外持久事实文件。

### Project Structure Notes

- 预计新增文件：
  - `model/enterprise/alert_delivery.go`
  - `model/enterprise/alert_delivery_test.go`
  - `service/enterprise/alert_dispatch.go`
  - `service/enterprise/alert_dispatch_task.go`
  - `service/enterprise/alert_dispatch_test.go`
  - `service/enterprise/alert_dispatch_task_test.go`
- 预计修改文件：
  - `model/enterprise/migration.go`
  - `service/enterprise/alert.go`
  - `service/enterprise/errors.go`
  - `service/enterprise/scheduler.go`
  - `dto/enterprise/alert.go`
  - `controller/enterprise/alert.go`
  - `controller/enterprise/alert_test.go`
  - `router/enterprise-router.go`
  - `tests/api/enterprise_alerts_test.go`
  - `docs/openapi/api.json`
  - `web/default/src/features/enterprise-alerts/api.ts`
  - `web/default/src/features/enterprise-alerts/types.ts`
  - `web/default/src/features/enterprise-alerts/index.tsx`
  - `web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx`
  - `web/default/src/i18n/locales/en.json`
  - `web/default/src/i18n/locales/zh.json`
  - `web/default/src/i18n/locales/fr.json`
  - `web/default/src/i18n/locales/ru.json`
  - `web/default/src/i18n/locales/ja.json`
  - `web/default/src/i18n/locales/vi.json`
  - `web/classic/src/pages/Enterprise/Alerts.js`
  - `web/classic/src/pages/Enterprise/Alerts.smoke.test.js`
  - `web/classic/src/services/enterprise.js`
  - `web/classic/src/i18n/locales/en.json`
  - `web/classic/src/i18n/locales/zh.json`
- 保持不改或只读依赖：
  - `controller/relay.go`
  - `model/log.go`
  - `controller/log.go`
  - `relay/**`
  - `pkg/billingexpr/**`
  - `service/enterprise/usage_report_task.go` 的既有报表调度语义

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 5.4: 发送可追溯告警通知]
- [Source: _bmad-output/planning-artifacts/epics.md#Story 5.5: 管理告警发送状态和人工重发]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-19-发送可追溯告警通知]
- [Source: _bmad-output/planning-artifacts/architecture.md#385]
- [Source: _bmad-output/planning-artifacts/architecture.md#393]
- [Source: _bmad-output/planning-artifacts/architecture.md#397]
- [Source: _bmad-output/planning-artifacts/architecture.md#481]
- [Source: _bmad-output/planning-artifacts/architecture.md#538]
- [Source: _bmad-output/planning-artifacts/architecture.md#557]
- [Source: _bmad-output/planning-artifacts/architecture.md#840]
- [Source: _bmad-output/planning-artifacts/architecture.md#849]
- [Source: _bmad-output/planning-artifacts/architecture.md#1089]
- [Source: _bmad-output/planning-artifacts/architecture.md#1142]
- [Source: _bmad-output/planning-artifacts/architecture.md#1143]
- [Source: _bmad-output/planning-artifacts/architecture.md#1154]
- [Source: _bmad-output/test-artifacts/test-design-progress.md#ENT-ALERT-P1-002]
- [Source: _bmad-output/implementation-artifacts/5-1-record-content-risk-events.md]
- [Source: _bmad-output/implementation-artifacts/5-2-query-and-filter-risk-events.md]
- [Source: _bmad-output/implementation-artifacts/5-3-configure-content-risk-alert-rules-and-channels.md]
- [Source: AGENTS.md#rule-1-json-package--use-commonjsongo]
- [Source: AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
- [Source: AGENTS.md#rule-6-upstream-relay-request-dtos--preserve-explicit-zero-values]
- [Source: common/email.go]
- [Source: service/webhook.go]
- [Source: service/user_notify.go]
- [Source: service/enterprise/alert.go]
- [Source: service/enterprise/errors.go]
- [Source: service/enterprise/scheduler.go]
- [Source: service/enterprise/admin_action.go]
- [Source: model/enterprise/alert_event.go]
- [Source: model/enterprise/alert_rule.go]
- [Source: model/enterprise/migration.go]
- [Source: dto/enterprise/alert.go]
- [Source: controller/enterprise/alert.go]
- [Source: controller/enterprise/alert_test.go]
- [Source: tests/api/enterprise_alerts_test.go]
- [Source: router/enterprise-router.go]
- [Source: web/default/src/features/enterprise-alerts/api.ts]
- [Source: web/default/src/features/enterprise-alerts/types.ts]
- [Source: web/default/src/features/enterprise-alerts/index.tsx]
- [Source: web/classic/src/pages/Enterprise/Alerts.js]
- [Source: web/classic/src/services/enterprise.js]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 按用户要求先读取 `.agents/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`，随后按 `bmad-create-story` workflow 执行 story `5.4`。
- `python3 _bmad/scripts/resolve_customization.py --skill .agents/skills/bmad-create-story --key workflow` 因当前环境 Python 缺少 `tomllib` 失败，已按 skill fallback 手工读取 `.agents/skills/bmad-create-story/customize.toml` 与 `_bmad/bmm/config.yaml`；确认 prepend/append 为空、persistent facts 仅包含可选 `project-context.md`，且仓库中未找到该文件。
- 已完整读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认目标 story key 为 `5-4-send-traceable-alert-notifications`，当前状态为 `backlog`，Epic 5 已是 `in-progress`。
- 已加载并分析 Epic 5、PRD FR19、Architecture 中 `enterprise_alert_deliveries` / `alert_dispatch.go` / `alert_dispatch_task.go` / scheduler / `Enterprise / Alerts` 合同、Story 5.1/5.2/5.3 已完成成果、现有 `service/enterprise/alert.go`、`service/enterprise/scheduler.go`、`service/webhook.go`、`service/user_notify.go`、`common/email.go`、Default/Classic 现有 `enterprise-alerts` 页面与测试。
- 已确认当前仓库尚未实现 `model/enterprise/alert_delivery.go`、`service/enterprise/alert_dispatch.go`、`service/enterprise/alert_dispatch_task.go`、`/api/enterprise/alerts/deliveries`、以及 scheduler 中的告警投递任务接线；5.4 的主要工作是补齐这一整条异步投递链路。
- 已按 checklist 自检：故事包含明确 AC 映射、现有代码落点、DB 幂等去重要求、通知内容与隐私边界、scheduler 接线要求、5.5 边界说明，以及模型/service/API/前端测试护栏。
- 已实现 `enterprise_alert_deliveries` 模型、跨库迁移补列、规则匹配建单、可追溯 payload、邮件/Webhook/钉钉机器人发送器、3 次退避重试、最终失败状态保留与 scheduler 挂接。
- 已实现 `GET /api/enterprise/alerts/deliveries`、相关 DTO 与 `docs/openapi/api.json` 合同，并为 Default/Classic 补充只读 deliveries 入口与 i18n 文案。
- 已运行定向验证：`GOCACHE=$(pwd)/.tmp/go-build-cache go test ./model/enterprise ./controller/enterprise ./tests/api`、`GOCACHE=$(pwd)/.tmp/go-build-cache go test ./service/enterprise -run 'TestAlertServiceEnqueueAlertDeliveriesMatchesRulesAndDedupesWithinWindow|TestAlertDispatchServiceRetriesAndMarksFinalFailure|TestAlertDispatchServiceSuccessMarksResentOnRetry|TestRunAlertDispatchTaskOnceCreatesAndProcessesPendingDeliveries'`、`bun test web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx web/classic/src/pages/Enterprise/Alerts.smoke.test.js`、`cd web/default && bun run typecheck`。
- 已在 review 中补充 `service/enterprise/scheduler.go` 调度修复：把告警 dispatch 从 1 分钟维护任务中拆分为独立 30 秒 ticker，并新增 `service/enterprise/scheduler_test.go` 锁定 Story 5.4 的首次失败后 30 秒重试节奏。
- 额外检查发现 `go test ./service/enterprise` 全量包测试在当前环境仍存在与本 story 无关的既有失败：`usage_report_task_test.go` 的现有断言失败，以及 `dingtalk_client_test.go` 依赖 `httptest.NewServer` 监听端口但当前沙箱禁止绑定端口。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- 已明确 Story 5.4 的主目标是“后端告警投递链路 + 最小查询入口”，而不是重做 5.3 规则配置或提前交付 5.5 的完整人工重发管理台。
- 已明确 `enterprise_alert_deliveries` 必须成为 FR19 的单一状态来源，并以 DB 级 `dedupe_key` / 幂等约束保障 5 分钟聚合语义。
- 已明确通知投递必须完全复用现有邮件/Webhook/notify 能力，企业模块只负责渲染追溯内容、执行调度和记录状态。
- 已明确重试节奏必须严格固定为 `30s -> 2m -> 10m`，最终失败仍需保留错误原因，且通知失败绝不影响风险事件落库。
- 已明确 Default/Classic 前端在本故事仅补最小 delivery 可见性，不提前交付 5.5 的复杂交互。
- 已新增 `enterprise_alert_deliveries` 单表状态源、delivery 去重键、追溯 JSON payload、delivery 查询 API、后台 dispatch task 以及 Default/Classic 只读投递视图。
- 已补齐模型、service、controller、API、Default、Classic 的定向测试，覆盖去重、重试、最终失败、权限、追溯展示和 secret 不泄露边界。
- 已在 Senior Developer Review (AI) 中自动修复 scheduler 30 秒重试缺口，并补充调度节奏回归测试。
- ✅ Definition of Done（故事范围内）通过：所有任务完成，story 状态已切换为 `done`，sprint 状态已同步为 `done`。

### File List

- `_bmad-output/implementation-artifacts/5-4-send-traceable-alert-notifications.md`
- `controller/enterprise/alert.go`
- `controller/enterprise/alert_test.go`
- `controller/enterprise/department_membership_test.go`
- `docs/openapi/api.json`
- `dto/enterprise/alert.go`
- `model/enterprise/alert_delivery.go`
- `model/enterprise/alert_delivery_test.go`
- `model/enterprise/migration.go`
- `router/enterprise-router.go`
- `service/enterprise/alert.go`
- `service/enterprise/alert_dispatch.go`
- `service/enterprise/alert_dispatch_task.go`
- `service/enterprise/alert_dispatch_task_test.go`
- `service/enterprise/alert_dispatch_test.go`
- `service/enterprise/errors.go`
- `service/enterprise/scheduler.go`
- `service/enterprise/scheduler_test.go`
- `tests/api/enterprise_alerts_test.go`
- `web/classic/src/i18n/locales/en.json`
- `web/classic/src/i18n/locales/zh.json`
- `web/classic/src/pages/Enterprise/Alerts.js`
- `web/classic/src/pages/Enterprise/Alerts.smoke.test.js`
- `web/classic/src/services/enterprise.js`
- `web/default/src/features/enterprise-alerts/api.ts`
- `web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx`
- `web/default/src/features/enterprise-alerts/index.tsx`
- `web/default/src/features/enterprise-alerts/types.ts`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/i18n/locales/zh.json`

### Change Log

- 2026-05-31 05:27:39 +0800: 完成 Story 5.4，新增告警投递状态表、规则匹配建单、异步发送与重试、delivery 查询 API、OpenAPI 合同，以及 Default/Classic 最小只读投递结果入口。
- 2026-05-31 05:48:42 +0800: Senior Developer Review (AI) 自动修复 `service/enterprise/scheduler.go` 的 1 分钟 tick 无法满足首次失败后 30 秒重试的问题，新增 `service/enterprise/scheduler_test.go` 回归测试，并将故事/冲刺状态同步为 done。

## Senior Developer Review (AI)

Reviewer: GPT-5 Codex  
Date: 2026-05-31  
Outcome: Approved after automatic fixes.

### Findings and Fixes

- [HIGH] `service/enterprise/scheduler.go` 仅以 1 分钟 ticker 驱动全部 enterprise 后台任务，`RunAlertDispatchTaskOnce` 最快也只能在下一分钟边界再次执行，导致首次失败后写入的 `next_retry_at = +30s` 无法按 AC2 准时被消费。已将告警 dispatch 拆分为独立的 30 秒 ticker，同时保留原 1 分钟维护任务节奏，并新增 `service/enterprise/scheduler_test.go` 锁定该回归。
- [MEDIUM] Story 的 `File List` 缺少 review 实际新增的 `service/enterprise/scheduler_test.go`，会让后续审查者误以为 retry 节奏没有专门测试覆盖。已更新文件清单。
- [MEDIUM] 故事正文与 `sprint-status.yaml` 在进入 review 后仍停留在 `review`，与“自动修复后无 CRITICAL 残留”的最终状态不一致，容易阻塞 story-automator 后续编排。已同步更新为 `done`。
- [LOW] 工作区仍存在与本故事无关的既有变更，例如 `.agents/skills/bmad-story-automator/**/__pycache__`、`web/default/package.json`、`web/default/rsbuild.test.config.ts`、`web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`、`service/enterprise/usage_report_task_test.go`。这些文件不属于本故事源码实现或本次修复范围，review 已按 workflow 记录但未改动。

### Validation

- Passed: `GOCACHE=$(pwd)/.tmp/go-build-cache go test ./service/enterprise -run 'TestEnterpriseAlertDispatchTickerSupportsThirtySecondRetryWindow|TestAlertDispatchServiceRetriesAndMarksFinalFailure|TestRunAlertDispatchTaskOnceCreatesAndProcessesPendingDeliveries' -count=1`
- Passed: `git diff -- service/enterprise/scheduler.go service/enterprise/scheduler_test.go`

### Review Notes

- 已按 workflow 交叉核对 story claims、git 变更与应用源码实现；本次自动修复集中在 `service/enterprise/scheduler.go` 与新增回归测试 `service/enterprise/scheduler_test.go`。
- `service/enterprise/alert_dispatch.go` 中的 `30s -> 2m -> 10m` 退避计算和最终失败持久化逻辑已由现有测试覆盖，本次 review 补齐的是“调度频率必须真正允许 30 秒重试生效”的执行层缺口。
- 当前未发现残留 CRITICAL 问题；故事状态已更新为 `done`，可继续推进 Story 5.5。
