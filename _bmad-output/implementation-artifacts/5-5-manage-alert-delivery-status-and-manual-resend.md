---
baseline_commit: b95fa2c7f8b2d7a5eacf1df370a2e8f7e291e9b5
---

# Story 5.5: 管理告警发送状态和人工重发

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员,
I want 查看告警发送状态并重新发送失败告警,
so that 通知失败可以被追踪和人工补救。

## Acceptance Criteria

1. **Given** 告警发送记录存在  
   **When** 管理员查看告警发送列表  
   **Then** 系统区分待发送、发送成功、发送失败、最终失败、已重发状态  
   **And** 失败状态包含可读错误原因。
2. **Given** 告警最终失败  
   **When** 管理员点击重新发送  
   **Then** 系统创建新一轮投递并标记为人工触发  
   **And** 不重置原风险事件追溯信息。
3. **Given** 通知投递失败  
   **When** 风险事件已产生  
   **Then** 通知失败不会阻止风险事件落库  
   **And** 后台仍可查看事件详情。

## Tasks / Subtasks

- [x] 扩展 delivery 服务层，提供最终失败查询、人工重发建单与状态衍生逻辑 (AC: 1, 2, 3)
  - [x] 在 `service/enterprise/alert.go` 或新建邻近服务文件中新增人工重发入口，例如 `ResendAlertDelivery(...)`，仅接受 `final_failed` 或等价允许人工补救的 delivery；不得对 `sent`、`pending`、`failed` 直接原地改状态。[Source: _bmad-output/planning-artifacts/epics.md#Story 5.5: 管理告警发送状态和人工重发; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#348]
  - [x] 人工重发必须创建一条新的 `enterprise_alert_deliveries` 记录，而不是覆盖原记录；新记录需设置 `trigger_source = manual_resend`，并通过 `manual_parent_id` 指向原 `final_failed` delivery，保留完整历史链路。[Source: model/enterprise/alert_delivery.go; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#348]
  - [x] 新一轮 delivery 必须继承原事件与追溯信息（`event_id`、`rule_id`、`channel_type`、`trace_payload`、错误前上下文），但不得复制旧的失败状态与 `final_failed_at`；初始状态应回到 `pending`，供后台调度再次投递。[Source: service/enterprise/alert.go#mapAlertDeliveryItem; service/enterprise/alert_dispatch.go; model/enterprise/alert_delivery.go]
  - [x] 手工重发不能影响原风险事件事实源 `enterprise_alert_events`，也不能回写旧 delivery 使其看起来像成功发送；旧记录仍保持 `final_failed`，新记录单独承担新的发送生命周期。[Source: _bmad-output/implementation-artifacts/5-4-send-traceable-alert-notifications.md#Dev Notes; _bmad-output/planning-artifacts/architecture.md#1154]

- [x] 补齐 delivery 列表查询能力，使后台可以围绕状态和失败原因做运维排查 (AC: 1, 3)
  - [x] 在现有 `ListAlertDeliveries` 查询基础上补充 5.5 所需筛选维度，至少包括 `status`、`channel_type`、`rule_id`、`event_id`、分页，以及可选的“仅最终失败/仅人工触发”视角；保持参数校验和 tenant 作用域约束。[Source: service/enterprise/alert.go#ListAlertDeliveries; controller/enterprise/alert.go#ListAlertDeliveries]
  - [x] 列表返回要稳定暴露运维字段：当前状态、错误原因、尝试次数、最近尝试时间、最终失败时间、触发来源、父重发链路 ID、追溯摘要；不得返回 secret、完整 webhook query、邮箱凭据或原始敏感内容。[Source: dto/enterprise/alert.go#AlertDeliveryItem; _bmad-output/test-artifacts/test-design-progress.md#ENT-SEC-P0-002]
  - [x] 如果 5.5 需要单条详情接口，优先复用 `GET /api/enterprise/alerts/deliveries` 的结果模型或追加管理后台只读详情端点，避免新造与列表不一致的 delivery DTO。[Source: _bmad-output/planning-artifacts/architecture.md#840; docs/openapi/api.json]

- [x] 新增管理员人工重发 API、审计与 OpenAPI 合同，保持 controller 只做边界处理 (AC: 2, 3)
  - [x] 在 `router/enterprise-router.go` 下新增管理员专用的人工重发 mutation，例如 `POST /api/enterprise/alerts/deliveries/:id/resend`；继续使用 `middleware.EnterpriseAdmin()`，不要把重发能力暴露给普通用户或部门管理员。[Source: router/enterprise-router.go; middleware/enterprise_admin.go]
  - [x] 在 `controller/enterprise/alert.go` 增加对应 handler，只负责路径参数解析、tenant scope、错误映射和调用 service；不要在 controller 中直接拼装新 delivery 或写数据库事务。[Source: controller/enterprise/alert.go; _bmad-output/planning-artifacts/architecture.md#840]
  - [x] 人工重发成功后应记录低频管理动作审计，摘要中包含原 delivery ID、新 delivery ID、event ID、rule ID、channel_type 和 actor；审计 payload 继续遵守不泄露 secret/原始敏感文本的规则。[Source: service/enterprise/admin_action.go; _bmad-output/planning-artifacts/architecture.md#393]
  - [x] `docs/openapi/api.json` 必须补充重发 mutation 和 5.5 使用到的 delivery 列表筛选/字段说明，仍然只写入 `Enterprise / Alerts` 管理后台契约。[Source: _bmad-output/planning-artifacts/architecture.md#611]

- [x] 让后台调度与人工重发路径共享既有 dispatch 能力，而不是平行实现第二套发送逻辑 (AC: 2, 3)
  - [x] 人工重发创建的新 delivery 仍由现有 `alert_dispatch_task` / `AlertDispatchService` 消费，复用 5.4 已交付的邮件、Webhook、钉钉机器人发送链路与 30s/2m/10m 退避策略；不要在重发 API 中同步发送通知。[Source: service/enterprise/alert_dispatch.go; service/enterprise/alert_dispatch_task.go; service/enterprise/scheduler.go]
  - [x] 对人工重发后的 delivery，`AlertDispatchService` 发送成功时应写成 `resent`，失败时沿用已有 `failed/final_failed` 语义，并保留新的失败原因。[Source: service/enterprise/alert_dispatch.go#dispatchDelivery; model/enterprise/alert_delivery.go]
  - [x] 人工重发路径必须继续具备幂等保护：短时间内重复点重发不能制造无穷多条等价 pending 记录。优先在 service 层对“同一父 delivery 的未完成人工重发”做约束，而不是依赖前端禁用按钮。[Source: _bmad-output/test-artifacts/test-design-progress.md#ENT-ALERT-P1-002; _bmad-output/planning-artifacts/architecture.md#1089]

- [x] Default 与 Classic 前端在现有 alerts 页面内补齐投递管理与人工重发最小闭环 (AC: 1, 2)
  - [x] Default 继续扩展 `web/default/src/features/enterprise-alerts/`，把 5.4 的只读 `Deliveries` 视图升级为可按状态查看最终失败、展示错误原因、尝试次数、触发来源，并对 `final_failed` 记录提供重发入口；不要新开独立 feature 或平行页面。[Source: web/default/src/features/enterprise-alerts/index.tsx; _bmad-output/implementation-artifacts/5-4-send-traceable-alert-notifications.md#Tasks--Subtasks]
  - [x] Default 的人工重发交互应明确显示这是“创建新一轮投递”而非覆盖旧记录；重发后刷新列表，让管理员能同时看到旧 `final_failed` 和新 `pending/resent` 记录。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#348]
  - [x] Classic 继续在 `web/classic/src/pages/Enterprise/Alerts.js` 内补最小投递运维区，至少支持查看失败原因和触发重发；不要为 Classic 先做复杂筛选面板或新页面。[Source: web/classic/src/pages/Enterprise/Alerts.js; _bmad-output/planning-artifacts/architecture.md#840]
  - [x] 新增文案继续走 i18n：Default 六语种、Classic 中英文，尤其是 `Resend`、`Manual resend`、`Final failed`、`Retry source`、`Parent delivery`、`Resend created` 等状态/提示不可硬编码。[Source: AGENTS.md#internationalization-i18n]

- [x] 补齐 5.5 的后端、API、前端与调度测试，锁定人工重发不破坏 5.4 基础模型 (AC: 1, 2, 3)
  - [x] 在 `service/enterprise/alert_dispatch_test.go`、新增测试文件或等效位置覆盖：`final_failed` 人工重发创建新 delivery、`manual_parent_id` 正确关联、`trigger_source=manual_resend`、发送成功写 `resent`、发送失败保留原 delivery 不变。[Source: service/enterprise/alert_dispatch_test.go; model/enterprise/alert_delivery.go]
  - [x] 在 `controller/enterprise/alert_test.go` 与 `tests/api/enterprise_alerts_test.go` 增加重发 API 权限与结果测试，覆盖管理员可重发、普通用户/部门管理员拒绝、非最终失败不可重发、列表返回新旧两条记录、返回字段不泄露 secret。[Source: controller/enterprise/alert_test.go; tests/api/enterprise_alerts_test.go]
  - [x] 为 Default `enterprise-alerts` 增加重发按钮/状态列表测试，至少覆盖：仅 `final_failed` 显示重发入口、重发后 query invalidation 精确刷新 deliveries、旧记录仍显示 `final_failed`、新记录显示 `pending/resent`；Classic 至少补 smoke 断言确认 `getAlertDeliveries` 与重发入口仍在页面结构中。[Source: web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx; web/classic/src/pages/Enterprise/Alerts.smoke.test.js]
  - [x] 如新增任何调度或状态约束，补充 `service/enterprise/scheduler_test.go` 或相关 task test，确保 5.4 刚修复的 `30s` 调度语义不会被 5.5 回归破坏。[Source: service/enterprise/scheduler_test.go; _bmad-output/implementation-artifacts/5-4-send-traceable-alert-notifications.md#Senior Developer Review (AI)]

## Dev Notes

- 5.5 是建立在 5.4 已交付的 `enterprise_alert_deliveries` 状态表之上的“运维面”和“人工补救面”，不是重新设计告警投递链路。优先复用 `AlertDispatchService`、`RunAlertDispatchTaskOnce`、现有 DTO 和现有 `Deliveries` tab，再补管理员重发与列表筛选。[Source: _bmad-output/implementation-artifacts/5-4-send-traceable-alert-notifications.md#Completion Notes List]
- `model/enterprise/alert_delivery.go` 已经预留了 `trigger_source = manual_resend` 与 `manual_parent_id`，这是 5.5 的关键切入点。不要另加一张“resend jobs”表，也不要把人工重发混进旧 delivery 的同一行里更新。[Source: model/enterprise/alert_delivery.go]
- 5.4 已经把后台调度拆成了独立的 `enterpriseAlertDispatchTaskTickInterval = 30s`。5.5 的任何人工重发实现都必须继续利用这条调度链路，而不是在 API 请求里同步发送、等待 SMTP/Webhook 返回后再响应。[Source: service/enterprise/scheduler.go; service/enterprise/alert_dispatch_task.go]
- 现有 `ListAlertDeliveries` 已支持 `status`、`channel_type`、`rule_id`、`event_id` 和分页，但还没有人工重发 mutation，也还没有 delivery-level 详情/更细筛选。5.5 的开发重点是补服务行为和最小管理交互，而不是推翻 5.4 的返回模型。[Source: service/enterprise/alert.go#ListAlertDeliveries; controller/enterprise/alert.go#ListAlertDeliveries]
- `AlertDispatchService` 当前已经区分：首次成功写 `sent`，重试成功或人工触发成功写 `resent`。这意味着 5.5 不需要重写状态机，只需要正确地产生新的人工触发 delivery 并让调度消费它。[Source: service/enterprise/alert_dispatch.go#dispatchDelivery]
- PRD 明确要求“重新发送计为新一轮投递，不重置原事件追溯”。因此 5.5 的前后端都要把“旧记录仍保留 + 新记录单独产生”作为核心语义，不要做覆盖式刷新，不要让前端把一行状态从 `final_failed` 直接改成 `resent`。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#348]
- 继续遵守 5.4 的隐私边界：错误原因可读，但 secret、query token、完整凭据、原始敏感文本不能出现在 API 响应、日志、审计 payload 或前端状态中。人工重发相关审计同样如此。[Source: _bmad-output/test-artifacts/test-design-progress.md#ENT-SEC-P0-002; _bmad-output/implementation-artifacts/5-4-send-traceable-alert-notifications.md#Tasks--Subtasks]

### Project Structure Notes

- 预计新增或重点修改文件：
  - `service/enterprise/alert.go`
  - `service/enterprise/alert_dispatch.go`
  - `service/enterprise/errors.go`
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
- 可能新增测试文件：
  - `service/enterprise/alert_resend_test.go` 或并入 `alert_dispatch_test.go`
  - `service/enterprise/scheduler_test.go`（如需补 5.5 相关回归）
- 保持不改或只读依赖：
  - `relay/**`
  - `pkg/billingexpr/**`
  - `service/enterprise/alert_dispatch_task.go` 的既有调度职责（除确有必要的最小配套修改外）
  - `model/enterprise/alert_delivery.go` 的既有状态模型（优先复用，不另起平行模型）

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 5.5: 管理告警发送状态和人工重发]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#347]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#348]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#353]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#355]
- [Source: _bmad-output/planning-artifacts/architecture.md#393]
- [Source: _bmad-output/planning-artifacts/architecture.md#611]
- [Source: _bmad-output/planning-artifacts/architecture.md#840]
- [Source: _bmad-output/planning-artifacts/architecture.md#849]
- [Source: _bmad-output/planning-artifacts/architecture.md#1089]
- [Source: _bmad-output/planning-artifacts/architecture.md#1142]
- [Source: _bmad-output/planning-artifacts/architecture.md#1143]
- [Source: _bmad-output/planning-artifacts/architecture.md#1154]
- [Source: _bmad-output/test-artifacts/test-design-progress.md#ENT-SEC-P0-002]
- [Source: _bmad-output/test-artifacts/test-design-progress.md#ENT-ALERT-P1-002]
- [Source: _bmad-output/implementation-artifacts/5-4-send-traceable-alert-notifications.md]
- [Source: service/enterprise/alert.go]
- [Source: service/enterprise/alert_dispatch.go]
- [Source: service/enterprise/alert_dispatch_task.go]
- [Source: service/enterprise/errors.go]
- [Source: service/enterprise/scheduler.go]
- [Source: service/enterprise/scheduler_test.go]
- [Source: model/enterprise/alert_delivery.go]
- [Source: dto/enterprise/alert.go]
- [Source: controller/enterprise/alert.go]
- [Source: controller/enterprise/alert_test.go]
- [Source: tests/api/enterprise_alerts_test.go]
- [Source: router/enterprise-router.go]
- [Source: web/default/src/features/enterprise-alerts/api.ts]
- [Source: web/default/src/features/enterprise-alerts/types.ts]
- [Source: web/default/src/features/enterprise-alerts/index.tsx]
- [Source: web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx]
- [Source: web/classic/src/pages/Enterprise/Alerts.js]
- [Source: web/classic/src/pages/Enterprise/Alerts.smoke.test.js]
- [Source: web/classic/src/services/enterprise.js]
- [Source: AGENTS.md#rule-1-json-package--use-commonjsongo]
- [Source: AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
- [Source: AGENTS.md#rule-6-upstream-relay-request-dtos--preserve-explicit-zero-values]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 先后 3 次由 `bmad-story-automator` 触发 `5.5 create-story` 会话，但都停在通用 create-story 质量复核阶段，未生成 `5-5-*.md` artifact，也未把 `sprint-status.yaml` 从 `backlog` 推到 `ready-for-dev`。
- 已读取 `.agents/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`，并按同一 workflow 手动接管 `5.5` 的 story 生成。
- 已重新读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认 `5-5-manage-alert-delivery-status-and-manual-resend` 仍为 `backlog`，Epic 5 处于 `in-progress`。
- 已系统复盘 5.4 交付物、PRD FR19、Epic 5 的 5.5 BDD、Architecture 中 `enterprise_alert_deliveries` / `alert_dispatch_task` / `Enterprise / Alerts` 路由与前端 feature 约束，以及测试设计文档中的人工重发与状态可观测性要求。
- 已确认当前仓库现状：存在 `GET /api/enterprise/alerts/deliveries`、`manual_parent_id`、`trigger_source=manual_resend` 和 `resent` 状态，但尚未实现管理员人工重发 mutation、delivery 管理交互与对应测试。
- 本 story 通过手动接管落盘，避免 story automator 在 `5.5 create-story` 上无限重试；后续 dev-story 应直接基于本文件继续推进。
- 已按 dev-story workflow 完成 5.5 实现：后端新增 `ResendAlertDelivery`、delivery 查询筛选增强、管理员重发审计、OpenAPI 合同、Default/Classic 最小重发闭环与 i18n 文案，并通过定向 Go/Bun 测试。
- 运行 `go test ./controller/enterprise -run 'TestAlert'`、`go test ./tests/api -run 'TestEnterpriseAlert'`、`go test ./service/enterprise -run 'TestAlert(ServiceListAlertDeliveriesFiltersTenantStatusAndPagination|ServiceResendAlertDeliveryCreatesManualPendingDelivery|ServiceResendAlertDeliveryReusesInFlightManualResend|ServiceResendAlertDeliveryRejectsNonFinalFailedStatus|DispatchServiceSuccessMarksResentOnRetry|DispatchServiceRetriesAndMarksFinalFailure)'`、`bun test src/features/enterprise-alerts/enterprise-alerts.test.tsx`、`bun test src/pages/Enterprise/Alerts.smoke.test.js` 与 `bun run i18n:sync`；Default 六语种同步报告 `missingCount=0`。
- 额外尝试过 `go test ./service/enterprise/...` 全包回归，但当前仓库已有与本故事无关的失败：`service/enterprise/usage_report_task_test.go` 断言不稳定，以及 `service/enterprise/dingtalk_client_test.go` 在当前沙箱无法绑定本地端口。

### Completion Notes List

- 手动接管 create-story，成功为 5.5 建立可开发的 story artifact。
- 5.5 的核心范围已锁定为：delivery 状态管理、最终失败人工重发、新一轮 delivery 建单、原追溯链路保留、管理员专用 API 与最小前端交互。
- 明确要求继续复用 5.4 的 `enterprise_alert_deliveries`、`AlertDispatchService`、`alert_dispatch_task` 和 `Deliveries` tab，不新造平行状态模型或同步发送逻辑。
- 明确 5.5 的重发语义是“旧 `final_failed` 保留 + 新 `pending/resent` delivery 产生”，而不是覆盖旧记录。
- 后端已支持 `manual_parent_id` / `trigger_source` 过滤、`trace_summary` 返回、`POST /api/enterprise/alerts/deliveries/:id/resend` 管理员 mutation、人工重发审计和错误映射。
- Default `Deliveries` tab 已支持最终失败筛选、触发来源/父投递/最近尝试/最终失败时间展示，以及“创建新一轮投递”的人工重发交互；Classic 在原页内补齐了最小重发入口。
- `docs/openapi/api.json` 已补充 delivery 重发 mutation、筛选参数、`trace_summary` 与 `AlertDeliveryResendResponse` 契约；Default 六语种与 Classic 中英文文案已同步。

### File List

- `_bmad-output/implementation-artifacts/5-5-manage-alert-delivery-status-and-manual-resend.md`
- `controller/enterprise/alert.go`
- `controller/enterprise/alert_test.go`
- `docs/openapi/api.json`
- `dto/enterprise/alert.go`
- `i18n/keys.go`
- `i18n/locales/en.yaml`
- `i18n/locales/zh-CN.yaml`
- `router/enterprise-router.go`
- `service/enterprise/admin_action.go`
- `service/enterprise/alert.go`
- `service/enterprise/alert_dispatch_test.go`
- `service/enterprise/errors.go`
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

- 2026-05-31 06:07:52 +0800: 手动接管 `5.5 create-story`，在 story automator 三次 create 会话均未落 artifact 的情况下，直接生成 `5.5` 故事文件并将其恢复到可继续 dev 的 `ready-for-dev` 状态。
- 2026-05-31 06:34:42 +0800: 完成 5.5 开发，实现管理员人工重发与 delivery 状态运维闭环，补齐后端服务、审计、OpenAPI、Default/Classic 前端、i18n 与定向回归测试，并将故事状态推进到 `review`。
- 2026-05-31 06:47:44 +0800: Senior Developer Review (AI) 自动修复 Default 投递视图在人工重发成功后仍停留 `final_failed` 过滤、导致新建 pending/resent delivery 不可见的问题；同步清理 story File List 偏差，并将故事/冲刺状态更新为 `done`。

## Senior Developer Review (AI)

Reviewer: GPT-5 Codex  
Date: 2026-05-31  
Outcome: Approved after automatic fixes.

### Findings and Fixes

- [HIGH] `web/default/src/features/enterprise-alerts/index.tsx` 在人工重发成功后仅执行 deliveries query invalidation，但页面默认仍保留 `status=final_failed` 过滤，管理员刷新后只能继续看到旧的 `final_failed` 父记录，看不到新建的 `pending/resent` 子投递，未满足 AC2 中“重发后同时看到旧 `final_failed` 和新 `pending/resent` 记录”的闭环。已在重发成功回调中自动清除阻塞该闭环的 `status` / `trigger_source` 过滤，并新增针对该状态切换的前端单测。
- [MEDIUM] Story `File List` 错误记录了与 5.5 无关的 `controller/enterprise/department_membership_test.go`，会让后续 review 误判该文件属于本故事实现面。已从 File List 移除，保留仅与 5.5 实际实现/验证相关的文件。
- [LOW] 工作区仍存在与 5.5 无关的既有变更，例如 `.agents/skills/bmad-story-automator/**/__pycache__`、`.tmp/go-build-cache/**`、`service/enterprise/usage_report_task_test.go`、`web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`。这些文件不属于本故事 review 修复范围，本次未改动。

### Validation

- Passed: `GOCACHE=$(pwd)/.tmp/go-build-cache go test ./controller/enterprise -run 'TestAlert'`
- Passed: `GOCACHE=$(pwd)/.tmp/go-build-cache go test ./tests/api -run 'TestEnterpriseAlert'`
- Passed: `GOCACHE=$(pwd)/.tmp/go-build-cache go test ./service/enterprise -run 'TestAlert(ServiceListAlertDeliveriesFiltersTenantStatusAndPagination|ServiceResendAlertDeliveryCreatesManualPendingDelivery|ServiceResendAlertDeliveryReusesInFlightManualResend|ServiceResendAlertDeliveryRejectsNonFinalFailedStatus|DispatchServiceSuccessMarksResentOnRetry|DispatchServiceRetriesAndMarksFinalFailure)'`
- Passed: `bun test src/features/enterprise-alerts/enterprise-alerts.test.tsx`
- Passed: `bun test src/pages/Enterprise/Alerts.smoke.test.js`

### Review Notes

- 已按 story-automator review workflow 交叉核对 story claims、git 变更、后端 delivery/resend 实现、Default/Classic 前端交互与定向测试结果。
- 5.5 的管理员人工重发、新 delivery 建单、`manual_parent_id` / `trigger_source=manual_resend`、旧 `final_failed` 保留、审计记录、OpenAPI 合同与 i18n 文案均已在源码与测试中找到对应证据。
- 当前无残留 CRITICAL 问题；故事状态与 `sprint-status.yaml` 已同步为 `done`，可继续推进 Story 5.6。
