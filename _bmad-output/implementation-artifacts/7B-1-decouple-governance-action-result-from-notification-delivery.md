---
baseline_commit: e735b306516f1e2cf923e263d541955c39d2b850
---

# Story 7B.1: 解耦治理动作结果与通知投递结果

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员或负责人,
I want 审批/分配结果与治理通知投递状态在后端语义和前端展示上独立存在,
so that 我不会把“未配置通知通道”或“投递失败”误判为“审批失败”。

## Acceptance Criteria

1. **Given** quota request 审批已成功并已创建 allocation
   **When** 系统没有可用治理通知通道或相关 alert rule
   **Then** 请求与 allocation 结果仍显示成功
   **And** 通知区域显示“未配置/未投递/待配置”一类独立状态，而不是 `final_failed` 误导业务结论。
2. **Given** 通知投递真实失败
   **When** 前端展示治理时间线
   **Then** 审批动作结果与投递结果分别可见
   **And** 错误原因仅归属通知投递记录，不覆盖治理动作结论。

## Tasks / Subtasks

- [x] 定义治理通知缺配置的非阻塞状态语义 (AC: 1)
  - [x] 在 `model/enterprise/governance_notification_delivery.go` 增加治理通知专用状态常量，例如 `unconfigured`、`skipped` 或 `configuration_missing`；命名应表达“未配置/未投递/待配置”，不得复用 `final_failed`。
  - [x] 更新 `service/enterprise/governance_notification.go` 的 `isKnownGovernanceNotificationStatus`，确保新状态可被列表接口按 status 查询并原样返回。
  - [x] 优先不改表结构；当前 `status`、`error_reason`、`trace_payload`、`next_retry_at` 已能表达该语义。若确需迁移，必须走 `model/enterprise/migration.go` 的三库兼容列追加模式。

- [x] 修正 governance notification dispatch 的缺配置处理 (AC: 1)
  - [x] 修改 `service/enterprise/governance_notification_dispatch.go`：`findDingTalkChannel` 找不到 enabled DingTalk robot channel、无可用 alert rule 或等价缺配置错误时，写入新的非阻塞状态，而不是由 `markDeliveryConfigurationFailure` 写入 `final_failed`。
  - [x] 新状态必须设置 `next_retry_at = 0`，避免后台任务无限重试；`attempt_count` 可以保持 0 或明确记录一次配置检查，但不得呈现为真实投递尝试失败。
  - [x] `error_reason` 只写非敏感配置原因，例如缺少治理通知通道；不得写 webhook token、secret、access token、SQL 细节或其他敏感配置。
  - [x] 保留真实 webhook 发送失败路径：网络错误、HTTP/webhook 错误仍走 `failed -> final_failed` 重试与手动 resend 语义。

- [x] 保持 quota request / allocation 核心事实独立成功 (AC: 1, 2)
  - [x] 确认 `service/enterprise/quota_request.go` 中 `Approve` 事务仍先创建 allocation 并把 request 写为 `fulfilled`；通知 enqueue/dispatch 失败不得回滚该事务。
  - [x] 确认 `service/enterprise/quota_allocation.go` 的创建、撤销、回收、revoke 等治理事实不依赖通知投递成功。
  - [x] 不要新增“通知失败则修正 request status / allocation status / timeline status”的补偿逻辑；业务事实状态源仍是 `enterprise_quota_requests`、`enterprise_quota_allocations` 和 timeline projection。

- [x] 调整 API/DTO 暴露，使前端能分层展示 (AC: 1, 2)
  - [x] `dto/enterprise/governance_notification.go` 与 `controller/enterprise/governance_notification.go` 应继续返回 delivery 自身 `status/error_reason/trace.status`；新状态必须原样返回。
  - [x] 默认优先复用 `status`，避免扩大 API 面；如确需增强，只能新增非破坏性字段，例如 `configuration_state` 或 `blocking=false`。
  - [x] `ResendGovernanceNotification` 只允许对真实 `final_failed` 或明确可重试终态失败触发；“未配置/跳过/待配置”状态不得显示或允许 resend。

- [x] 调整 Default 企业组织治理活动展示 (AC: 1, 2)
  - [x] 在 `web/default/src/features/enterprise-organization/index.tsx` 的 `GovernanceActivityCard` 中保持业务动作列与通知投递列分离：timeline `item.status` 展示审批/分配事实，delivery `item.status` 展示通知事实。
  - [x] `governanceDeliveryStatusLabel` 增加新状态文案，建议英文源 key 使用 `Not configured`、`Not delivered` 或 `Pending configuration` 之一，并同步全部 locale。
  - [x] `governanceDeliveryStatusVariant` 对新状态使用 grey/warning 等非业务失败视觉，不使用红色错误样式。
  - [x] `GovernanceNotificationDeliveryList` 对新状态不显示 `Resend`；错误原因只显示在 delivery 卡片内，不得覆盖同一 trace 的 `fulfilled` / allocation 成功状态。
  - [x] 新增文案必须同步 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` 的 flat translation keys，并运行 i18n/前端校验。

- [x] 补充后端和前端回归测试 (AC: 1, 2)
  - [x] 在 `service/enterprise/governance_notification_dispatch_test.go` 增加“无 alert rule / 无 enabled DingTalk robot channel”用例：dispatch 后 delivery 为新非阻塞状态，不是 `final_failed`，且不会进入继续重试队列。
  - [x] 保留并回归 `TestGovernanceNotificationDispatchFailureBackoffAndFinalFailed`：真实 webhook 发送失败仍先 `failed` 后 `final_failed`。
  - [x] 在 `controller/enterprise/governance_controller_test.go` 增加通知列表可按新状态返回，且 resend 对新状态不可用的 API 行为测试。
  - [x] 在 `service/enterprise/quota_request_test.go` 增加或确认审批成功后无通知通道时 request 为 `fulfilled`、allocation 已创建、delivery 为新非阻塞状态。
  - [x] 在 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx` 增加 UI 用例：同一 trace 中 timeline 显示 `fulfilled`，通知列显示“未配置/未投递/待配置”，页面不出现 `Final failed`，也不显示 `Resend`。
  - [x] 补充真实投递失败 UI 用例：timeline 仍显示动作成功状态，delivery 卡片显示 `Final failed`、错误原因和 `Resend`。

- [x] 运行并记录验证 (AC: 1, 2)
  - [x] `go test ./service/enterprise -run 'GovernanceNotification|QuotaRequest|QuotaAllocation'`
  - [x] `go test ./controller/enterprise -run 'GovernanceNotification|GovernanceTimeline|QuotaRequest'`
  - [x] `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx`
  - [x] `cd web/default && bun run typecheck`
  - [x] `cd web/default && bun run i18n:sync`
  - [x] `cd web/default && bun run test:e2e`（如耗时不可接受，至少记录未运行原因，并保留上面的定向测试）

## Dev Notes

### 需求与业务上下文

- Epic 7B 是 Epic 7 / 7A 后的 follow-up，目标是修正企业治理额度申请闭环中的通知语义耦合、预算池可辨识性、i18n 残留、wallet 展示语义和金额视角；Story 7B.1 只处理“治理动作成功事实”和“通知投递事实”被误读的问题。[Source: `_bmad-output/planning-artifacts/epics.md`#Epic 7B]
- Story 7B.1 的核心验收是：quota request 审批已成功且 allocation 已创建时，即使没有治理通知通道或相关 alert rule，请求和 allocation 仍显示成功；通知区域显示“未配置/未投递/待配置”一类独立状态，不再把缺配置显示为 `final_failed`。[Source: `_bmad-output/planning-artifacts/epics.md`#Story 7B.1]
- 2026-06-03 change proposal 已补充 FR-26 的 V1.3B 要求：治理动作结果与通知投递结果必须独立记录和独立展示；审批通过、自动分配成功等核心治理动作，不得因为通知未配置或通知投递失败而在产品上被呈现为治理动作失败。[Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Change B]
- 架构 V1.3B 明确：治理通知投递是核心治理事务之后的旁路动作；无可用通知通道或缺配置应记录为非阻塞状态，真实投递失败才需要可重试、可追踪并与治理动作结果分层展示。[Source: `_bmad-output/planning-artifacts/architecture.md`#治理通知投递语义（V1.3B）]
- 2026-06-03 correct-course 的排障结论是：当前最大误导不是审批主链路失败，而是“通知配置缺失/投递失败”被用户误读为“审批失败”。本故事不要重写审批主链路。[Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Problem Summary]

### 当前实现状态（已读 UPDATE 文件）

- `service/enterprise/quota_request.go`
  - `Approve` 调 `decide`，在 DB transaction 内调用 `QuotaAllocationService.createTx` 创建 allocation，然后把 `enterprise_quota_requests.status` 写为 `fulfilled`，并记录 `approved_quota/allocation_id/approved_at/fulfilled_at/processed_at`。
  - transaction 成功后才调用 `NewGovernanceNotificationService(s.db).EnqueueQuotaRequestDecision(...)`。这已经是非阻塞结构，应保留。
  - 不要为了修 UI 语义而改变 request 状态机，或把通知 delivery 状态写回 quota request。

- `model/enterprise/governance_notification_delivery.go`
  - 治理通知当前复用 alert delivery 状态：`pending/sent/failed/final_failed/resent`。
  - 新增治理通知专用状态可以继续存入同一 `status` varchar 字段，不需要新增数据库列；但必须更新 service known-status 校验和前端映射。
  - `trace_payload` 已使用 `common.Marshal` / `common.UnmarshalJsonStr`，继续遵守项目 JSON wrapper 规则。

- `service/enterprise/governance_notification_dispatch.go`
  - `DispatchDueDeliveries` 只处理 `pending` 和 `failed` 且 `next_retry_at <= now` 的记录。
  - `dispatchDelivery` 调 `findDingTalkChannel`；找不到 enabled DingTalk robot channel 时当前进入 `markDeliveryConfigurationFailure`，该函数直接写 `status=final_failed`、`next_retry_at=0`、`final_failed_at=now`。这是 AC1 指向的主要缺陷路径。
  - `markDeliverySendFailure` 是真实 webhook 发送失败路径，带 backoff 和最终 `final_failed`；这条路径应保留，以满足 AC2。

- `service/enterprise/governance_notification.go`
  - `createDeliveryIfMissing` 当前创建 delivery 时默认 `status=pending`、`channel_type=dingtalk_robot`、`trigger_source=governance_action`，trace payload 中保留业务事实 `Status`，例如 quota request trace status 为 `fulfilled`。
  - `ListDeliveries` 返回 delivery status 与 trace payload；`deliveryVisibleToViewer` 按 Admin、recipient、可治理部门过滤。新状态必须继续走这套权限边界。
  - `ResendDelivery` 当前只允许 parent `Status == final_failed`。Story 7B.1 应保持该限制，并确保新“未配置”状态不能 resend。

- `controller/enterprise/governance_notification.go` 与 `dto/enterprise/governance_notification.go`
  - API 当前把 delivery status、attempt、error_reason、trace.status 都返回前端。前端已有足够数据分层展示：`item.status` 是投递状态，`item.trace.status` 或 timeline row `status` 是业务动作状态。
  - 若增加状态常量，不需要破坏响应结构；保持 `{ success, message, data }` envelope 与 snake_case 字段。

- `web/default/src/features/enterprise-organization/index.tsx`
  - `GovernanceActivityCard` 当前同一行显示 timeline `Status` 与 `Notification Status` 两列，结构已经适合分层展示。
  - `GovernanceNotificationDeliveryList` 当前只对 `final_failed` 显示 `Resend`，这对新状态应保持不显示。
  - `governanceDeliveryStatusLabel` 目前只覆盖 `pending/sent/resent/failed/final_failed`，需要新增状态映射；未知状态保留 `Unknown delivery status` 兜底。

### Previous Story Intelligence

- Story 7.6 已建立统一治理 timeline projection 与独立 `enterprise_governance_notification_deliveries`，并明确防误点：不要新建万能审计表复制所有事实；不要把治理通知塞进风险告警专用 delivery；不要让通知失败回滚核心治理事务。[Source: `_bmad-output/implementation-artifacts/7-6-audit-governance-actions-and-send-dingtalk-notifications.md`#Dev Notes]
- Story 7.6 completion 记录显示当前实现已支持 request/approval/allocation 路径 enqueue、dedupe key、后台 dispatch、manual resend 和 Default 治理活动视图；7B.1 应在这些既有机制上修正状态语义，而不是另起通知系统。[Source: `_bmad-output/implementation-artifacts/7-6-audit-governance-actions-and-send-dingtalk-notifications.md`#Completion Notes List]
- Story 7A.1 修过治理 action label / i18n 映射，要求未知 action/status 使用安全兜底，不让内部 key 或裸状态直接露出；新增文案必须进入全部 Default locale。[Source: `_bmad-output/implementation-artifacts/7a-1-complete-enterprise-governance-i18n-and-action-label-mapping.md`#Completion Notes List]
- Story 7A.2 修过钱包侧 quota request 入口，并补充了提交成功后失效 quota request、governance timeline 和 notification query 的缓存。7B.1 不要拆分新的 query namespace。[Source: `_bmad-output/implementation-artifacts/7a-2-move-employee-quota-request-entry-to-balance-wallet-user-flow.md`#Senior Developer Review]
- Story 7A.3 已确认同部门多类型预算池并存与显式 budget id 选择，不属于本故事范围；不要把通知状态修复扩展成预算池选择或金额显示改造。[Source: `_bmad-output/implementation-artifacts/7a-3-relax-budget-pool-type-locking-to-per-budget-invariance.md`#Completion Notes List]

### Git Intelligence Summary

- 最近提交 `e735b3065 docs: finalize epic 7b governance follow-up plan` 已将 Epic 7B 规划落地到 planning artifacts 与 sprint status；本故事应按该 follow-up 执行，不重开 Epic 7A。
- 当前工作树已有用户/自动化产生的未提交文件和 sprint status 改动；dev agent 不得清理、回退或重写无关变更。

### Architecture Compliance

- 继续遵守 Router -> Controller -> Service -> Model 分层；状态判定与 dispatch 规则放 service/model，controller 只绑定参数、调用 service、映射 DTO。[Source: `AGENTS.md`; `_bmad-output/planning-artifacts/architecture.md`#Source Organization]
- 所有 JSON marshal/unmarshal 必须使用 `common.Marshal` / `common.Unmarshal` / `common.UnmarshalJsonStr`；不要直接调用 `encoding/json.Marshal` 或 `json.Unmarshal`。[Source: `AGENTS.md`#Rule 1]
- 数据库必须兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+；本故事优先不改 schema。若改 migration，只使用 GORM/AddColumn 兼容路径，不使用 JSONB、PG-only UPSERT 或 ALTER COLUMN。[Source: `AGENTS.md`#Rule 2]
- Default 前端 i18n 使用 `t('English source string')`，translation JSON 是 flat key；新增 key 必须覆盖 en、zh、fr、ru、ja、vi。[Source: `AGENTS.md`#Internationalization]
- `new-api` 与 `QuantumNous` 是受保护标识，不得删除、替换或改名。[Source: `AGENTS.md`#Rule 5]
- 本故事不得触碰 relay、billing expression、upstream DTO 或 `pkg/billingexpr/**`；它只处理企业治理通知展示语义。[Source: `_bmad-output/planning-artifacts/architecture.md`#Anti-Patterns]

### Library / Framework Requirements

- 后端继续使用 Go、Gin、GORM v2；不需要新增 Go 依赖。
- 前端继续使用 React 19、TypeScript、TanStack Query、i18next 和现有 Base UI 组件；使用 Bun 运行前端脚本。
- 不需要 web research 或依赖升级；不要借本故事升级 React、GORM、Gin、Rsbuild 或新增通知库。

### File Structure Requirements

Expected UPDATE files:

- `model/enterprise/governance_notification_delivery.go` — 新增治理通知非阻塞配置状态常量。
- `service/enterprise/governance_notification_dispatch.go` — 区分缺配置与真实发送失败。
- `service/enterprise/governance_notification.go` — 更新 known status / list / resend 相关规则。
- `service/enterprise/governance_notification_dispatch_test.go` — 覆盖缺配置状态与真实失败回归。
- `service/enterprise/quota_request_test.go` — 覆盖审批 fulfilled + allocation created + 通知缺配置不覆盖业务事实。
- `controller/enterprise/governance_controller_test.go` — 覆盖新状态列表与 resend 边界。
- `web/default/src/features/enterprise-organization/index.tsx` — 新增 delivery 状态 label/variant 与 resend 展示规则。
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx` — 覆盖分层展示。
- `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` — 如新增文案必须同步。

Likely no changes needed:

- `service/enterprise/quota_request.go`、`service/enterprise/quota_allocation.go` 的核心事务逻辑已非阻塞；除非测试暴露问题，不要重写。
- `service/enterprise/governance_timeline.go` 已投影业务事实状态；除非需要新增明确字段，不要改变 timeline source 聚合。
- `model/enterprise/migration.go` 预计不需要改，因为状态是 varchar 内容变化。
- `router/enterprise-router.go` 预计不需要改，现有 governance notification/timeline API 已存在。

### Project Structure Notes

- 本故事完全落在企业治理模块内，后端按 `model/enterprise`、`service/enterprise`、`controller/enterprise`、`dto/enterprise` 分层更新；前端按 Default theme 的 `web/default/src/features/enterprise-organization` feature 更新。
- 与统一项目结构对齐：不新增平行企业治理页面，不新增跨层共享 notification package，不把治理通知逻辑移动到风险告警 feature。
- 检测到的实现偏差是状态语义偏差，不是文件归属偏差：当前治理通知已在独立 delivery 模型和企业组织工作台中落位，dev agent 应修正现有文件而不是迁移模块边界。

### Testing Requirements

Minimum targeted validation:

```bash
go test ./service/enterprise -run 'GovernanceNotification|QuotaRequest|QuotaAllocation'
go test ./controller/enterprise -run 'GovernanceNotification|GovernanceTimeline|QuotaRequest'
cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx
cd web/default && bun run typecheck
cd web/default && bun run i18n:sync
```

Recommended assertions:

- 无 enabled DingTalk robot alert rule 时，dispatch 后 delivery 不是 `final_failed`，而是新“未配置/跳过/待配置”状态。
- 新状态 `next_retry_at=0`，不会被后台任务无限重试。
- 新状态不会显示 manual resend；真实 `final_failed` 仍可 resend。
- quota request approve 返回/查询时 status 为 `fulfilled`，allocation id 非零，delivery 状态不会覆盖 request/allocation 状态。
- 真实 webhook 发送失败仍先 `failed`、到达 max attempts 后 `final_failed`，错误原因只在 delivery 上。
- UI 同一 trace 同时展示业务状态 `fulfilled` 与通知状态“未配置/未投递/待配置”，不把业务状态渲染为失败。
- 新增 i18n key 在 en、zh、fr、ru、ja、vi 全部存在。

### Anti-Patterns to Avoid

- 不要把缺少 alert rule / DingTalk robot 配置写成 `final_failed`；`final_failed` 只代表真实投递尝试达到终态失败。
- 不要把 delivery status 回写到 `enterprise_quota_requests.status`、`enterprise_quota_allocations.status` 或 timeline 业务状态。
- 不要创建第二套治理通知表或万能审计表复制已有事实。
- 不要污染 `enterprise_alert_deliveries`；治理通知继续使用 `enterprise_governance_notification_deliveries`。
- 不要直接向前端暴露 webhook URL、secret、access token 或未脱敏错误。
- 不要全量 invalidate `['enterprise']`；继续使用 `governanceTimelineQueryKey`、`governanceNotificationQueryKey` 和 quota request scoped keys。
- 不要把 Story 7B.2、7B.3、7B.4、7B.5 的预算池摘要、i18n 全量收尾、wallet 时间语义、金额切换混入本故事。

### References

- [Source: `_bmad-output/planning-artifacts/epics.md`#Story 7B.1]
- [Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Change B]
- [Source: `_bmad-output/planning-artifacts/architecture.md`#治理通知投递语义（V1.3B）]
- [Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Change G / Suggested PR Split]
- [Source: `service/enterprise/governance_notification_dispatch.go`#dispatchDelivery / markDeliveryConfigurationFailure]
- [Source: `service/enterprise/governance_notification.go`#ResendDelivery / ListDeliveries / isKnownGovernanceNotificationStatus]
- [Source: `model/enterprise/governance_notification_delivery.go`#GovernanceNotificationStatus*]
- [Source: `service/enterprise/quota_request.go`#Approve / decide]
- [Source: `web/default/src/features/enterprise-organization/index.tsx`#GovernanceActivityCard / GovernanceNotificationDeliveryList]
- [Source: `_bmad-output/implementation-artifacts/7-6-audit-governance-actions-and-send-dingtalk-notifications.md`#Dev Notes]
- [Source: `_bmad-output/implementation-artifacts/7a-1-complete-enterprise-governance-i18n-and-action-label-mapping.md`#Completion Notes List]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-03: BMAD create-story workflow executed for Story 7B.1 with YOLO mode. The local workflow resolver required Python 3.11+ and failed on this machine, so customization was resolved manually per fallback instructions.
- 2026-06-03: Loaded BMAD config: `user_name=hth`, communication/document language Chinese, planning artifacts `_bmad-output/planning-artifacts`, implementation artifacts `_bmad-output/implementation-artifacts`.
- 2026-06-03: Persistent facts discovery found no `project-context.md` files under the repository.
- 2026-06-03: Input discovery loaded `epics.md`, `architecture.md`, relevant PRD/change proposal sections, previous stories 7.6/7A.1/7A.2/7A.3, current sprint status, git history, and UPDATE files around governance notification, quota request, controller DTO/API, and Default UI.
- 2026-06-03: Checklist validation applied: story includes exact ACs, current-state notes for update files, previous story intelligence, anti-patterns, cross-DB/i18n/project rules, and targeted tests.
- 2026-06-03: Confirmed `markDeliveryConfigurationFailure` is the current defect path, `ResendDelivery` is restricted to `final_failed`, Default UI currently maps only `pending/sent/resent/failed/final_failed`, and existing tests cover true webhook failure but not the no-channel configuration state.
- 2026-06-03: `sprint-status.yaml` already had `epic-7b: in-progress` and `7b-1-decouple-governance-action-result-from-notification-delivery: ready-for-dev`; no additional sprint-status transition was required in this execution.
- 2026-06-03: Implemented `unconfigured` delivery state for missing governance notification configuration and kept real webhook failures on `failed -> final_failed`.
- 2026-06-03: Senior Developer Review found and auto-fixed missing disabled/unavailable DingTalk robot coverage and non-blocking delivery error visual treatment.
- 2026-06-03: Validation passed: targeted Go service/controller tests, Default feature test, typecheck, i18n sync, and `test:e2e`.

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- Story scope is constrained to separating governance action success from notification delivery state.
- Primary current defect is identified in `GovernanceNotificationDispatchService.dispatchDelivery`: missing DingTalk notification channel currently flows into `markDeliveryConfigurationFailure` and writes `final_failed`.
- The dev agent must preserve fulfilled quota request / allocation facts and only adjust notification delivery semantics and presentation.
- Added `GovernanceNotificationStatusUnconfigured` without schema changes and allowed it in notification status filters.
- Missing or unavailable DingTalk robot governance notification configuration now stores `unconfigured`, `next_retry_at=0`, no final failure timestamp, and a sanitized non-sensitive reason.
- Quota request approval remains fulfilled with allocation created even when governance notification dispatch cannot find a channel.
- Default governance activity renders action status and notification delivery status separately; `unconfigured` uses `Not configured`, neutral styling, no resend action, and localized text in all supported languages.
- Review auto-fix added coverage for enabled alert rules that still lack an enabled DingTalk robot, and changed unconfigured delivery reasons from destructive red text to muted explanatory text.

### File List

- `_bmad-output/implementation-artifacts/7B-1-decouple-governance-action-result-from-notification-delivery.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `controller/enterprise/governance_controller_test.go`
- `model/enterprise/governance_notification_delivery.go`
- `service/enterprise/governance_notification.go`
- `service/enterprise/governance_notification_dispatch.go`
- `service/enterprise/governance_notification_dispatch_test.go`
- `service/enterprise/quota_request_test.go`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/i18n/locales/zh.json`

## Change Log

- 2026-06-03 00:00:00 +0800: 创建 Story 7B.1 上下文文档，纳入 ready-for-dev。
- 2026-06-03 00:00:00 +0800: 实现治理通知 `unconfigured` 非阻塞状态、缺配置 dispatch 语义、API/UI/i18n/test 覆盖，并进入 review。
- 2026-06-03 00:00:00 +0800: Senior Developer Review 自动修复 disabled DingTalk robot 覆盖缺口和未配置状态视觉语义，验证通过并将故事状态更新为 done。

## Senior Developer Review (AI)

**Reviewer:** hth
**Date:** 2026-06-03 00:00:00 +0800
**Outcome:** Approve / 自动修复后通过

### 发现并自动修复的问题

1. **MEDIUM — 缺配置测试只覆盖“无 alert rule”，未覆盖“有规则但无 enabled DingTalk robot channel”。**
   证据：`service/enterprise/governance_notification_dispatch_test.go` 原新增用例只在没有任何 enabled alert rule 时验证 `unconfigured`。AC1 明确要求无可用 alert rule 或无 enabled DingTalk robot channel 都必须进入非阻塞状态。
   修复：新增 `TestGovernanceNotificationDispatchUnavailableDingTalkRobotIsNonBlocking`，构造 enabled alert rule、enabled email channel、disabled DingTalk robot，并断言 sender 不执行、delivery 为 `unconfigured`、`next_retry_at=0`、不泄露 token/secret。

2. **MEDIUM — 未配置状态的 `error_reason` 在前端仍使用 destructive 红色文本，视觉上容易继续被误读为业务失败。**
   证据：`web/default/src/features/enterprise-organization/index.tsx` 的 `GovernanceNotificationDeliveryList` 对所有 `error_reason` 都使用 `text-destructive`。AC1 要求缺配置显示为独立“未配置/未投递/待配置”状态，不用业务失败视觉。
   修复：仅 `failed` / `final_failed` 使用 destructive 文本，`unconfigured` 等非失败状态使用 muted explanatory text；保留真实投递失败的错误强调。

3. **MEDIUM — Story File List 和状态未反映实际实现文件。**
   证据：story 原状态仍是 `ready-for-dev`，File List 只列 story 文件，但 git diff 已包含后端、前端、i18n 与测试变更。
   修复：更新 story status、Change Log、Completion Notes、File List、Senior Developer Review 和 sprint tracking。

### 验收标准复核

- **AC1:** 通过。审批成功和 allocation 创建不依赖通知投递；缺 alert rule / 缺 enabled DingTalk robot 均写入 `unconfigured`，`next_retry_at=0`，不显示 `final_failed` 或 `Resend`。
- **AC2:** 通过。真实 webhook 失败仍按 `failed -> final_failed` 重试路径处理；治理活动 UI 保持 timeline 业务状态与 delivery 投递状态分层展示，错误原因只在 delivery 卡片内。

### 验证

- `GOCACHE=/private/tmp/sa-codex-home-05934b33/go-build-cache go test ./service/enterprise -run 'GovernanceNotification|QuotaRequest|QuotaAllocation'` ✅
- `GOCACHE=/private/tmp/sa-codex-home-05934b33/go-build-cache go test ./controller/enterprise -run 'GovernanceNotification|GovernanceTimeline|QuotaRequest'` ✅
- `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx` ✅
- `cd web/default && bun run typecheck` ✅
- `cd web/default && bun run i18n:sync` ✅
- `cd web/default && bun run test:e2e` ✅（Rsbuild 仍输出既有 optional dependency warning：`supports-color` 未解析；所有 bundled tests 通过。）

### Checklist

- [x] Story file loaded from `_bmad-output/implementation-artifacts/7B-1-decouple-governance-action-result-from-notification-delivery.md`
- [x] Story Status verified as reviewable（原文件仍为 `ready-for-dev`，但工作区已有完整实现；已记录流程偏差并修正为 `done`）
- [x] Epic and Story IDs resolved (7B.1)
- [x] Story Context located or warning recorded（未找到独立 story context；已使用 story、epics、architecture、change proposal 和源码）
- [x] Epic Tech Spec located or warning recorded（已读取/检索 `_bmad-output/planning-artifacts/epics.md`、`architecture.md`、`sprint-change-proposal-2026-06-03.md`）
- [x] Architecture/standards docs loaded (as available)
- [x] Tech stack detected and documented（Go/Gin/GORM；React 19/TypeScript/Rsbuild/Bun/i18next）
- [x] MCP doc search performed (or web fallback) and references captured（本地源码/规划文档足够；未使用外部 web）
- [x] Acceptance Criteria cross-checked against implementation
- [x] File List reviewed and validated for completeness
- [x] Tests identified and mapped to ACs; gaps noted
- [x] Code quality review performed on changed files
- [x] Security review performed on changed files and dependencies
- [x] Outcome decided (Approve/Changes Requested/Blocked)
- [x] Review notes appended under "Senior Developer Review (AI)"
- [x] Change Log updated with review entry
- [x] Status updated according to settings (if enabled)
- [x] Sprint status synced (if sprint tracking enabled)
- [x] Story saved successfully
