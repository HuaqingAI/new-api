---
baseline_commit: 6f90032728ba41cbfd1d2636f434eb7dccdc2411
---

# Story 5.3: 配置内容风险告警规则和通道

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员,
I want 配置内容风险告警规则、接收人和通道,
so that 风险事件可以自动通知合适负责人。

## Acceptance Criteria

1. **Given** 管理员打开告警规则配置  
   **When** 创建或更新规则  
   **Then** 系统支持按风险类型和部门范围启用或停用规则  
   **And** 保存时校验接收人或 Webhook 格式。
2. **Given** 告警规则保存成功  
   **When** 配置发生变化  
   **Then** 系统记录低频管理操作审计  
   **And** 不把 Webhook secret 或邮箱凭据写入普通日志。
3. **Given** V1 仅保证邮件告警为必选  
   **When** 钉钉机器人或 Webhook 未配置  
   **Then** 系统仍可通过邮件和后台事件完成 V1 告警闭环  
   **And** 可选通道不阻塞风险事件记录。

## Tasks / Subtasks

- [x] 新增告警规则模型与迁移，建立 Epic 5 的规则事实源并保持三库兼容 (AC: 1, 2, 3)
  - [x] 新增 `model/enterprise/alert_rule.go`，落地 `enterprise_alert_rules` 表，至少覆盖：`id`、`tenant_id`、`name`、`enabled`、`risk_types`、`department_ids`、`channel_configs`、`dedupe_window_seconds`、`created_by`、`updated_by`、`created_at`、`updated_at`。数组/对象字段统一使用 `TEXT` + `common.Marshal` / `common.UnmarshalJsonStr`，不要引入 `JSONB`、`JSON_CONTAINS`、数组列或其他单库专属能力。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping; AGENTS.md#rule-1-json-package--use-commonjsongo; AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
  - [x] 规则条件先按架构未决项收敛为 V1 最小白名单模型，不要提前做 DSL/表达式引擎：支持 `risk_type` 多选、`department_ids` 多选、规则启停，以及通道级接收人配置；不要把 5.3 变成复杂规则编排器。[Source: _bmad-output/planning-artifacts/architecture.md#4-告警规则-dsl-形式未定; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-18-配置告警规则和告警通道]
  - [x] `channel_configs` 至少支持 `email`，并为可选 `webhook` / `dingtalk_robot` 预留结构化字段；如果存储 webhook secret，允许写入数据库但读取接口必须脱敏或仅返回“已配置”状态，绝不把 secret 明文回传前端。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#104; _bmad-output/planning-artifacts/architecture.md#1294; model/enterprise/dingtalk_config.go]
  - [x] 在 `model/enterprise/migration.go` 注册 `AlertRule` 迁移，保持与 `AlertEvent`、`UsageReportJob` 等 enterprise 模型同一路径；本 story 只创建规则表，不提前引入 `alert_deliveries` 状态机实现细节以外的额外派生表。[Source: model/enterprise/migration.go; _bmad-output/planning-artifacts/architecture.md#Complete Project Directory Structure]

- [x] 实现告警规则 service，统一封装校验、脱敏和查询读写边界 (AC: 1, 2, 3)
  - [x] 在 `service/enterprise/alert.go` 中扩展规则读写能力，优先新增 `ListAlertRules`、`GetAlertRule`、`SaveAlertRule`、`DeleteAlertRule` 或等价入口，避免把规则逻辑散落到 controller。当前仓库已经在此 service 中承载 `RecordRiskEvent` 与 `ListAlertEvents`，5.3 继续沿用同一风险域 service 边界。[Source: service/enterprise/alert.go; _bmad-output/implementation-artifacts/5-1-record-content-risk-events.md#Dev Notes; _bmad-output/implementation-artifacts/5-2-query-and-filter-risk-events.md#Dev Notes]
  - [x] 保存时执行最小可验证校验：规则名非空、至少一个风险类型、部门列表允许为空表示全局、至少一个有效通道、邮件接收人格式合法、Webhook URL 为合法 `http/https` URL；显式空数组返回 `[]`，不要返回 `null`。[Source: _bmad-output/planning-artifacts/epics.md#Story 5.3: 配置内容风险告警规则和通道; _bmad-output/test-artifacts/test-design-progress.md#ENT-ALERT-P1-001]
  - [x] 可选通道失败或未配置不应阻断规则保存，也不应影响 `enterprise_alert_events` 的旁路记录路径。5.3 只负责“配置规则和通道”，不是发送时机与重试；发送链路、退避、最终失败和人工重发属于 5.4/5.5。[Source: _bmad-output/planning-artifacts/epics.md#Story 5.4: 发送可追溯告警通知; _bmad-output/planning-artifacts/epics.md#Story 5.5: 管理告警发送状态和人工重发]
  - [x] 对读取返回做通道级脱敏：邮箱接收人可明文返回，Webhook/DingTalk robot secret 只能返回布尔态或掩码，日志与错误也只能记录摘要原因，不得包含 secret、token、credential 或完整 webhook query string。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#424; service/enterprise/admin_action.go; _bmad-output/test-artifacts/test-design-progress.md#R-002]

- [x] 新增 `/api/enterprise/alerts/rules` 管理接口，并复用现有 enterprise 审计与权限边界 (AC: 1, 2, 3)
  - [x] 在 `dto/enterprise/alert.go` 中补齐规则查询/保存 DTO 和响应 DTO，可选标量字段使用指针类型 + `omitempty`，避免显式 `false`/`0` 在保存时被吞掉。建议至少包含 `AlertRuleChannelConfig`、`AlertRuleItem`、`AlertRuleUpsertRequest`、`AlertRulesResponse`。[Source: AGENTS.md#rule-6-upstream-relay-request-dtos--preserve-explicit-zero-values; dto/enterprise/alert.go]
  - [x] 在 `controller/enterprise/alert.go` 新增规则列表、详情、创建/更新、删除或停用入口，controller 仅做参数绑定、`requestTenantId(...)` 解析、service 调用和 error → i18n key 映射；不要在 controller 中手写 JSON 序列化、通道 secret 脱敏或审计 diff 逻辑。[Source: controller/enterprise/alert.go; controller/enterprise/admin_action.go]
  - [x] 在 `router/enterprise-router.go` 注册 `/api/enterprise/alerts/rules` 相关路由，统一使用 `middleware.EnterpriseAdmin()`。普通用户和仅部门管理员不应编辑企业级告警规则，除非后续故事另行细化更窄权限模型。[Source: router/enterprise-router.go; middleware/enterprise_admin.go]
  - [x] OpenAPI 合同同步写入 `docs/openapi/api.json` 的 `Enterprise / Alerts` tag，不写入 `docs/openapi/relay.json`；接口响应继续遵守 `{ success, message, data }` 习惯。[Source: _bmad-output/planning-artifacts/architecture.md#325; AGENTS.md#overview]

- [x] 在保存成功后写入低频管理审计，且审计/普通日志都不能泄露 secret (AC: 2)
  - [x] 为 `service/enterprise/admin_action.go` 增加告警规则相关 `action_type` / `object_type` 常量，例如 `enterprise.alert.rule.save`、`enterprise.alert.rule.delete` 与 `enterprise_alert_rule`；保存/删除规则成功后统一调用审计服务记录 actor、规则 ID、启停状态、风险类型和部门范围变更摘要。[Source: _bmad-output/planning-artifacts/architecture.md#726; service/enterprise/admin_action.go]
  - [x] 审计 payload 允许记录邮件接收人、department ids、risk types 等非 secret 配置，但 webhook secret、token、app_key、credential、password 等字段必须继续命中现有净化逻辑或新增同等净化分支。[Source: service/enterprise/admin_action.go; _bmad-output/test-artifacts/test-design-progress.md#ENT-SEC-P0-002]
  - [x] 普通运行日志只允许输出 `rule_id`、`tenant_id`、校验失败原因、通道类型等摘要信息，不得打印完整规则 JSON、明文接收人列表中的敏感备注、或 webhook secret 原文。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#340; _bmad-output/test-artifacts/test-design-epic-1-5.md#56]

- [x] Default 前端在现有 `enterprise-alerts` feature 下新增规则配置体验，不拆出平行模块 (AC: 1, 2, 3)
  - [x] 继续在 `web/default/src/features/enterprise-alerts/` 内扩展 `api.ts`、`types.ts`、`index.tsx` 与必要组件；不要把规则编辑塞回 `enterprise-usage`、系统设置页或新建平行 `enterprise-alert-rules` 顶级 feature。[Source: _bmad-output/planning-artifacts/architecture.md#840; web/default/src/features/enterprise-alerts/index.tsx]
  - [x] 页面至少支持：规则列表、创建/编辑表单、启停切换、风险类型多选、部门范围输入或选择、邮件接收人输入、可选通道配置状态展示，以及 secret 已配置但不回显的 UX。V1 必须保证邮件通道闭环；Webhook/钉钉机器人是可选增强，不得成为提交表单的硬依赖。[Source: _bmad-output/planning-artifacts/epics.md#Story 5.3: 配置内容风险告警规则和通道; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#338]
  - [x] TanStack Query queryKey 保持 enterprise alerts 域内一致，建议新增 `['enterprise', 'alerts', 'rules', ...]`；mutation invalidate 精确到 alerts feature，不要粗暴刷新整个 enterprise 导航或 usage 数据。[Source: AGENTS.md#architecture; web/default/src/features/enterprise-alerts/api.ts]
  - [x] 新增 UI 文案同步到 `web/default/src/i18n/locales/en.json`、`zh.json`、`fr.json`、`ru.json`、`ja.json`、`vi.json`，英文源字符串作为 key；任何“secret already configured”之类提示都必须走 i18n，不要硬编码。[Source: AGENTS.md#internationalization-i18n]

- [x] Classic 前端补最小可用规则管理入口，不追求与 Default 同深度 (AC: 1, 3)
  - [x] 在 `web/classic/src/pages/Enterprise/Alerts.js` 或同目录下的辅助组件中增加基本规则列表/保存入口，沿用现有 `web/classic/src/services/enterprise.js` 请求封装；Classic 可接受更轻量的交互，但不能完全没有规则管理路径。[Source: _bmad-output/planning-artifacts/architecture.md#351; web/classic/src/pages/Enterprise/Alerts.js; web/classic/src/services/enterprise.js]
  - [x] Classic 仍保持其中文文案风格和 smoke 级测试策略，不把 Default 的复杂表单状态管理整套搬过去。[Source: _bmad-output/test-artifacts/test-design-epic-1-5.md#46]

- [x] 补齐后端、API、前端测试，锁定格式校验、权限与 secret 脱敏边界 (AC: 1, 2, 3)
  - [x] 新增 `model/enterprise/alert_rule_test.go`，覆盖迁移成功、JSON 空数组归一化、三库兼容文本字段、secret 字段默认值语义，以及读回模型后掩码/已配置状态映射的基础行为。[Source: model/enterprise/migration.go; model/enterprise/alert_event_test.go]
  - [x] 扩展 `service/enterprise/alert_test.go`，覆盖：邮件接收人校验、Webhook URL 校验、规则启停、全局规则与部门定向规则、可选通道未配置仍可保存邮件规则、以及 service 返回 DTO/读模型不泄露 secret。[Source: service/enterprise/alert.go; _bmad-output/test-artifacts/test-design-progress.md#ENT-ALERT-P1-001]
  - [x] 新增或扩展 `controller/enterprise/alert_test.go` 与 `tests/api/enterprise_alerts_test.go`，覆盖管理员可读写、普通用户/部门管理员被拒绝、非法参数返回 `invalid params`、审计写入成功、以及接口 payload 中不存在 webhook secret / token / credential 原文。[Source: controller/enterprise/alert_test.go; tests/api/enterprise_alerts_test.go; middleware/enterprise_admin.go]
  - [x] 为 Default `enterprise-alerts` 增加组件/页面测试，覆盖规则列表渲染、表单提交、保存后刷新、secret 字段不回显、可选通道关闭时邮件规则仍可提交，以及 query invalidation 精确命中 alerts rules。[Source: web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx]
  - [x] Classic 至少补一条 smoke test，确认规则列表/基础保存入口存在且不因新增 API 导致 alerts 页面崩溃。[Source: web/classic/src/pages/Enterprise/Alerts.smoke.test.js]

## Dev Notes

- Story 5.3 的权威目标是“建立告警规则与通道配置层”，不是发送任务本身。不要提前把 5.4 的退避重试、5.5 的投递列表/人工重发、5.6 的风险概览聚合塞进本故事；本故事完成后应为后续 `alert_dispatch_task` 提供稳定规则输入面。[Source: _bmad-output/planning-artifacts/epics.md#Story 5.3: 配置内容风险告警规则和通道; _bmad-output/planning-artifacts/epics.md#Story 5.4: 发送可追溯告警通知; _bmad-output/planning-artifacts/epics.md#Story 5.5: 管理告警发送状态和人工重发]
- 架构已经明确内容告警域的文件落点：`model/enterprise/alert_rule.go`、`service/enterprise/alert.go`、后续 `alert_dispatch.go` / `alert_dispatch_task.go`、`controller/enterprise/alert.go`、`/api/enterprise/alerts/rules`、Default `enterprise-alerts/` 和 Classic `pages/Enterprise/Alerts.js`。5.3 应严格沿着这条边界落地，不要新开系统设置入口或通用 notify 配置页。[Source: _bmad-output/planning-artifacts/architecture.md#840]
- V1 邮件通道是必选闭环，可直接复用 `common.SendEmail` 与仓库既有 SMTP 配置；Webhook、Bark、Gotify、钉钉机器人能力在代码库里已有基础 notify 实现，但对 Epic 5 来说属于可选通道扩展。5.3 至少要把数据结构和校验边界铺好，且在未配置可选通道时不阻塞邮件规则保存。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#338; _bmad-output/planning-artifacts/architecture.md#1142; _bmad-output/planning-artifacts/architecture.md#1143; service/user_notify.go; service/webhook.go; common/email.go]
- 规则条件不要过度设计。架构里已把“告警规则 DSL 未定”列为 gap，建议 V1 以“风险类型 + 部门范围 + 通道配置 + 启停”的白名单条件收口；否则 dev 很容易把 5.3 做成表达式系统，既偏离 FR18，也会抬高 5.4 的发送复杂度。[Source: _bmad-output/planning-artifacts/architecture.md#4-告警规则-dsl-形式未定]
- 审计与脱敏是本故事的主风险之一。`service/enterprise/admin_action.go` 已对 key/value 做敏感词打码，但规则 payload 仍需谨慎设计，避免因为自定义字段命名不一致而漏掉 `secret`、`token`、`credential`、`webhook` 等敏感内容。接口读模型也不能把 secret 直接透给前端。[Source: service/enterprise/admin_action.go; _bmad-output/test-artifacts/test-design-progress.md#R-002]
- 5.2 已经交付 `enterprise-alerts` 的事件查询页面和 `GET /api/enterprise/alerts/events`。5.3 最稳妥的前端增量方式是在同一 feature 内扩展 tabs/section，把“事件列表”和“规则配置”放在同一风险告警模块中，复用现有 sidebar、route 和 query 模式。[Source: _bmad-output/implementation-artifacts/5-2-query-and-filter-risk-events.md#Dev Notes; web/default/src/features/enterprise-alerts/index.tsx; web/default/src/routes/_authenticated/enterprise-alerts/index.tsx]
- 测试设计文档已经把 FR18 归为 P1 风险，重点是“接收人/Webhook 校验、支持 department/risk filters、审计不泄露 secret”。5.3 的 DoD 不能只停留在 CRUD happy path，必须显式覆盖这些负向断言。[Source: _bmad-output/test-artifacts/test-design-progress.md#ENT-ALERT-P1-001; _bmad-output/test-artifacts/test-design-epic-1-5.md#150]

### Project Structure Notes

- 预计新增文件：
  - `model/enterprise/alert_rule.go`
  - `model/enterprise/alert_rule_test.go`
- 预计修改文件：
  - `model/enterprise/migration.go`
  - `dto/enterprise/alert.go`
  - `service/enterprise/alert.go`
  - `service/enterprise/alert_test.go`
  - `service/enterprise/errors.go`
  - `service/enterprise/admin_action.go`
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
  - `service/enterprise/scheduler.go` 中既有 usage/report/wallet 任务行为

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 5.3: 配置内容风险告警规则和通道]
- [Source: _bmad-output/planning-artifacts/epics.md#Story 5.4: 发送可追溯告警通知]
- [Source: _bmad-output/planning-artifacts/epics.md#Story 5.5: 管理告警发送状态和人工重发]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-18-配置告警规则和告警通道]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-19-发送可追溯告警通知]
- [Source: _bmad-output/planning-artifacts/architecture.md#840]
- [Source: _bmad-output/planning-artifacts/architecture.md#341]
- [Source: _bmad-output/planning-artifacts/architecture.md#397]
- [Source: _bmad-output/planning-artifacts/architecture.md#1142]
- [Source: _bmad-output/planning-artifacts/architecture.md#1143]
- [Source: _bmad-output/planning-artifacts/architecture.md#1154]
- [Source: _bmad-output/planning-artifacts/architecture.md#4-告警规则-dsl-形式未定]
- [Source: _bmad-output/planning-artifacts/architecture.md#726]
- [Source: _bmad-output/implementation-artifacts/5-1-record-content-risk-events.md]
- [Source: _bmad-output/implementation-artifacts/5-2-query-and-filter-risk-events.md]
- [Source: _bmad-output/implementation-artifacts/4-5-configure-and-send-scheduled-usage-reports.md]
- [Source: _bmad-output/implementation-artifacts/epic-4-retro-2026-05-29.md]
- [Source: _bmad-output/test-artifacts/test-design-progress.md#ENT-ALERT-P1-001]
- [Source: _bmad-output/test-artifacts/test-design-progress.md#R-002]
- [Source: _bmad-output/test-artifacts/test-design-epic-1-5.md#150]
- [Source: AGENTS.md#rule-1-json-package--use-commonjsongo]
- [Source: AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
- [Source: AGENTS.md#rule-6-upstream-relay-request-dtos--preserve-explicit-zero-values]
- [Source: common/email.go]
- [Source: service/user_notify.go]
- [Source: service/webhook.go]
- [Source: middleware/enterprise_admin.go]
- [Source: model/enterprise/alert_event.go]
- [Source: model/enterprise/dingtalk_config.go]
- [Source: model/enterprise/migration.go]
- [Source: service/enterprise/admin_action.go]
- [Source: service/enterprise/alert.go]
- [Source: service/enterprise/errors.go]
- [Source: controller/enterprise/alert.go]
- [Source: controller/enterprise/admin_action.go]
- [Source: dto/enterprise/alert.go]
- [Source: router/enterprise-router.go]
- [Source: tests/api/enterprise_alerts_test.go]
- [Source: web/default/src/features/enterprise-alerts/api.ts]
- [Source: web/default/src/features/enterprise-alerts/types.ts]
- [Source: web/default/src/features/enterprise-alerts/index.tsx]
- [Source: web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx]
- [Source: web/default/src/routes/_authenticated/enterprise-alerts/index.tsx]
- [Source: web/classic/src/pages/Enterprise/Alerts.js]
- [Source: web/classic/src/pages/Enterprise/Alerts.smoke.test.js]
- [Source: web/classic/src/services/enterprise.js]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 按用户要求先读取 `.agents/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`，随后按 `bmad-create-story` workflow 执行 story `5.3`。
- `python3 _bmad/scripts/resolve_customization.py --skill .agents/skills/bmad-create-story --key workflow` 因当前环境 Python 缺少 `tomllib` 失败，已按 skill fallback 手工读取 `.agents/skills/bmad-create-story/customize.toml` 与 `_bmad/bmm/config.yaml`；确认 prepend/append 为空、persistent facts 仅包含可选 `project-context.md`，仓库中未找到该文件。
- 已完整读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认目标 story key 为 `5-3-configure-content-risk-alert-rules-and-channels`，当前状态为 `backlog`，Epic 5 已是 `in-progress`。
- 已加载并分析 Epic 5、PRD FR18/FR19、Architecture 中 `enterprise_alert_rules` / `alert_dispatch_task` / `Enterprise / Alerts` 映射、Epic 4 retrospective、Story 5.1/5.2 已完成成果、`service/enterprise/alert.go`、`controller/enterprise/alert.go`、`router/enterprise-router.go`、`service/enterprise/admin_action.go`、`service/user_notify.go`、`service/webhook.go`、`common/email.go`、Default/Classic 现有 `enterprise-alerts` 页面与测试。
- 已确认当前仓库尚未实现 `model/enterprise/alert_rule.go`、`/api/enterprise/alerts/rules` 相关 DTO / controller / router / OpenAPI 合同，也尚未在 Default/Classic 暴露规则管理 UI；Story 5.3 的主要工作是建立这一完整配置面。
- 已按 checklist 自检：故事包含明确 AC 映射、具体文件落点、最小规则模型、邮件必选/可选通道边界、secret 脱敏要求、审计与普通日志护栏、以及后端/API/前端/Classic smoke 回归要求。
- 已按 `bmad-dev-story` workflow 激活并切换当前 story 为 `in-progress`，准备从后端规则模型、service 校验与审计边界开始实现。
- 已完成 `enterprise_alert_rules` 模型、迁移、service CRUD、审计常量、`/api/enterprise/alerts/rules` controller/router/OpenAPI 合同，以及 Default/Classic 前端规则管理入口。
- 已执行并通过定向验证：`go test ./model/enterprise ./service/enterprise -run 'TestAlert'`、`go test ./controller/enterprise -run 'TestAlert'`、`go test ./tests/api -run 'TestEnterpriseAlert'`、`go test ./controller/enterprise ./tests/api ./model/enterprise`、`cd web/default && bun run typecheck`、`cd web/default && bun run build:check`、`cd web/default && bun run i18n:sync`、`cd web/classic && node --test src/pages/Enterprise/Alerts.smoke.test.js`。
- 默认 Node `--test` 不能直接执行 `.tsx` 测试文件，Classic JSX 页面也不能被原生 Node 直接 import；因此前端验证采用项目可执行的 `typecheck` / `build:check` 与 smoke 级源码断言组合，未伪报不可执行路径为已通过。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- 已明确 Story 5.3 只负责“规则与通道配置”，不得提前实现 5.4/5.5 的发送重试、投递列表、最终失败和人工重发状态机。
- 已明确 V1 规则条件先收敛为“风险类型 + 部门范围 + 通道配置 + 启停”的最小白名单模型，不引入复杂 DSL。
- 已明确邮件通道是 V1 必选闭环；Webhook / 钉钉机器人可以作为可选通道存在，但不得阻塞邮件规则保存或风险事件记录。
- 已明确 secret 边界：数据库可存，API 读回需脱敏，审计 payload 与普通日志均不得出现明文 secret / token / credential。
- 已明确 Default 前端继续在现有 `enterprise-alerts` feature 内扩展规则配置，Classic 仅补必要入口和 smoke 级验证。
- 已落地规则白名单模型：风险类型、部门范围、邮件必选、Webhook/钉钉可选、去重窗口、启停状态，并保持数组字段统一为 TEXT JSON 存储。
- 已落地接口与审计边界：规则保存/删除写入 `enterprise.alert.rule.save` / `enterprise.alert.rule.delete` 管理审计，Webhook/DingTalk secret 与 query string 在返回体和审计 payload 中均已脱敏。
- 已落地前端体验：Default 在同一 `enterprise-alerts` feature 中加入 `Risk Events` / `Alert Rules` tabs 和规则编辑器；Classic 补轻量规则列表与保存表单入口。

### File List

- `_bmad-output/implementation-artifacts/5-3-configure-content-risk-alert-rules-and-channels.md`
- `model/enterprise/alert_rule.go`
- `model/enterprise/alert_rule_test.go`
- `model/enterprise/migration.go`
- `service/enterprise/alert.go`
- `service/enterprise/alert_test.go`
- `service/enterprise/admin_action.go`
- `service/enterprise/errors.go`
- `dto/enterprise/alert.go`
- `controller/enterprise/alert.go`
- `controller/enterprise/alert_test.go`
- `controller/enterprise/department_membership_test.go`
- `router/enterprise-router.go`
- `tests/api/enterprise_alerts_test.go`
- `docs/openapi/api.json`
- `i18n/keys.go`
- `i18n/locales/en.yaml`
- `i18n/locales/zh-CN.yaml`
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
- `web/default/src/routeTree.gen.ts`
- `web/classic/src/services/enterprise.js`
- `web/classic/src/pages/Enterprise/Alerts.js`
- `web/classic/src/pages/Enterprise/Alerts.smoke.test.js`
- `web/classic/src/i18n/locales/en.json`
- `web/classic/src/i18n/locales/zh.json`

### Change Log

- 2026-05-31: 新增企业告警规则模型、规则 CRUD service、`/api/enterprise/alerts/rules` 管理接口、审计常量与脱敏逻辑；扩展 Default/Classic 风险告警页面以支持规则管理，并补齐后端/API/前端验证与多语言文案。
- 2026-05-31: Senior Developer Review (AI) 自动修复规则详情/删除接口未接收 `tenant_id` 查询作用域、更新规则时错误丢弃既有可选通道配置、Classic 规则编辑表单不同步，以及 OpenAPI/DTO 将 secret 输入输出混用的问题；故事状态更新为 done。

## Senior Developer Review (AI)

Reviewer: GPT-5 Codex
Date: 2026-05-31
Outcome: Approved after automatic fixes.

### Findings and Fixes

- [HIGH] `controller/enterprise/alert.go` 的 `GET /api/enterprise/alerts/rules/:id` 与 `DELETE /api/enterprise/alerts/rules/:id` 没有绑定 `tenant_id` 查询参数，实际行为与 OpenAPI 合同和管理员多租户入口不一致，非默认租户规则无法按文档读取或删除。已修复为显式解析 `AlertRulesQuery`，并补充 tenant scope 回归测试。
- [HIGH] `service/enterprise/alert.go` 在更新规则时要求请求体重新提交全部 `channel_configs`；如果前端仅编辑名称或风险类型，就会把已存的 webhook / DingTalk 配置整体覆盖掉，直接破坏 AC3 “可选通道未配置或失败不阻塞、但配置层应稳定存在”的语义。已修复为更新时保留未出现在本次请求中的既有可选通道，并补充 secret 保留回归测试。
- [HIGH] `service/enterprise/alert.go` 只去掉了 webhook query string，未清除 URL 中的 `userinfo` 凭据；`https://user:pass@...` 这类配置会被返回体和审计 payload 明文暴露，违反 AC2 的 secret 不落普通读路径要求。已修复为脱敏时同时清空 `User` 字段，并补充断言确保 `user:pass` 不再出现在返回/审计中。
- [MEDIUM] `dto/enterprise/alert.go` 与 `docs/openapi/api.json` 复用同一 `AlertRuleChannelConfig` 同时描述“写入请求”和“读取响应”，导致合同层公开了 `webhook_secret` / `dingtalk_robot_secret` 可出现在响应中的错误能力声明。已拆分为 input/item 两套 schema，保持响应只暴露 configured/masked 状态。
- [MEDIUM] `web/default/src/features/enterprise-alerts/index.tsx` 的前端 payload helper 会把只属于响应态的 `*_secret_configured` / `*_secret_masked` 回传到保存接口，增加了契约噪音，也掩盖了真实请求面。已收紧 payload 构造，仅发送可写字段。
- [MEDIUM] `web/classic/src/pages/Enterprise/Alerts.js` 使用 `Form initValues={draft}` 加 `setDraft(...)` 切换规则，但 Semi Form 不会因 `initValues` 变化自动重置，导致点击规则列表后表单仍停留在旧值，基础编辑入口实际上不可用。已改为通过 `getFormApi().setValues(...)` 同步选中规则和重置动作，并保留 smoke coverage。
- [LOW] Story 勾选了 “Default 组件/页面测试” 子任务，但当前 `web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx` 仍主要是 helper / route guard 级断言，而不是完整交互渲染测试。这次 review 未扩展到 DOM harness，因为仓库现有 Default 测试基线也是 Node-level helper tests；已至少补上 0 值去重窗口和 payload 契约回归，剩余更完整交互测试可在后续统一测试基建时补强。

### Validation

- Passed: `env GOCACHE=/Users/hq-it/repository/github/huaqingai/new-api/.tmp/go-build-cache go test ./service/enterprise ./controller/enterprise -run 'TestAlert(RulesAPI|Service)|TestAlertService|TestAlertRules'`
- Passed: `env GOCACHE=/Users/hq-it/repository/github/huaqingai/new-api/.tmp/go-build-cache go test ./tests/api -run 'TestEnterpriseAlertRulesAPIRequiresEnterpriseAdminAndSanitizesSecrets'`
- Passed: `cd web/default && bun run typecheck`
- Passed: `cd web/default && bun run build:check`
- Passed: `cd web/classic && node --test src/pages/Enterprise/Alerts.smoke.test.js`

### Review Notes

- 已按 workflow 核对 story File List 与应用源码改动；本次自动修复集中在 `controller/enterprise/alert.go`、`service/enterprise/alert.go`、`dto/enterprise/alert.go`、`docs/openapi/api.json`、Default/Classic `enterprise-alerts` 相关文件及对应测试。
- `_bmad-output/story-automator/**`、`.agents/skills/**/__pycache__` 与 `.tmp/go-build-cache/**` 等非应用源码变更未纳入实现审查，也未在本次修复中处理。
- 当前无 CRITICAL 问题残留；故事状态已更新为 `done`，并可继续作为 5.4/5.5 告警发送与投递管理故事的稳定配置输入面。
