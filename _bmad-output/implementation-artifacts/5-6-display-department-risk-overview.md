---
baseline_commit: 0f29984b272688fb69529787454f42fa39ea37b4
---

# Story 5.6: 展示部门风险概览

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 安全/合规负责人,
I want 查看部门风险次数、风险率和趋势,
so that 我可以识别高风险部门和近期变化。

## Acceptance Criteria

1. **Given** 存在风险事件和部门用量数据  
   **When** 管理员打开部门风险概览  
   **Then** 页面展示部门风险次数、风险率、近期趋势、Top 风险部门和风险事件入口  
   **And** 部门风险统计与风险事件明细数量一致。
2. **Given** 用户属于多个部门  
   **When** 计算部门风险率  
   **Then** 同一笔请求按成员所属部门重复计入多个部门  
   **And** 页面展示风险率公式和 `enterprise.usage.multi_dept_disclaimer`。
3. **Given** 用户未归属任何部门  
   **When** 计算风险率  
   **Then** 该用户请求不纳入任何部门风险率  
   **And** 单独显示为“未归属”。

## Tasks / Subtasks

- [x] 基于现有风险事件与用量聚合事实源实现部门风险概览 service，不重造第二套统计链路 (AC: 1, 2, 3)
  - [x] 在 `service/enterprise/alert.go` 或相邻 service 中新增部门风险概览查询入口，例如 `GetDepartmentRiskSummary(...)`，统一处理时间范围、tenant 作用域、排序和概览聚合；不要把统计逻辑散落到 controller 或前端。[Source: _bmad-output/planning-artifacts/architecture.md#840; _bmad-output/planning-artifacts/architecture.md#1261]
  - [x] 风险分子必须基于 `enterprise_alert_events` 的事件事实源按部门快照归属统计，多部门事件沿用“在每个相关部门重复计入”的既定语义；不得回查当前部门关系替代历史快照。[Source: _bmad-output/implementation-artifacts/5-1-record-content-risk-events.md#Dev Notes; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#362]
  - [x] 风险分母必须复用 Epic 4 已交付的部门用量聚合口径，而不是单独再扫 `logs` 生成第三套统计。优先复用 `enterprise_usage_snapshots`、`GetDepartmentSummary` / `GetDepartmentDetail` 已有聚合边界，保证 FR13/14/20 口径一致。[Source: _bmad-output/implementation-artifacts/4-1-aggregate-department-usage-snapshots.md; _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md; service/enterprise/usage_aggregation.go]
  - [x] 风险率公式严格遵守 PRD：`风险率 = 该部门成员触发风险事件的请求数 ÷ 该部门成员请求总数`；被拦截请求同时计入分子与分母；未归属用户请求不纳入任何部门风险率，但需要单独显示为“未归属”。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#362]

- [x] 设计并暴露稳定的风险概览 API/DTO，支撑 Top 部门、趋势和事件入口 (AC: 1, 2, 3)
  - [x] 在 `dto/enterprise/alert.go` 新增 5.6 所需 DTO，至少覆盖：部门风险摘要项、趋势点、Top 风险部门、风险率分子/分母、未归属桶、以及 drill-down 到风险事件的入口上下文；列表/数组字段稳定输出 `[]`，可选标量继续使用指针类型和 `omitempty`。[Source: dto/enterprise/usage.go; AGENTS.md#rule-6-upstream-relay-request-dtos--preserve-explicit-zero-values]
  - [x] 在 `controller/enterprise/alert.go` 与 `router/enterprise-router.go` 新增 `GET /api/enterprise/alerts/department-summary`，继续使用 `middleware.EnterpriseAdmin()`；controller 只做参数校验、tenant scope、service 调用和错误映射，不直接聚合数据。[Source: _bmad-output/planning-artifacts/architecture.md#840; router/enterprise-router.go]
  - [x] 接口参数至少支持 `from`、`to`、可选 `tenant_id`，并在需要时支持排序字段；返回结果必须可驱动“Top 风险部门”“趋势折线/柱图”“跳到风险事件列表”三类 UI，不把实现局限成单一表格接口。[Source: _bmad-output/planning-artifacts/epics.md#Story 5.6: 展示部门风险概览]
  - [x] `docs/openapi/api.json` 必须同步加入 `Enterprise / Alerts` 下的 department risk summary 合同；不要写入 `docs/openapi/relay.json`，也不要把风险概览混进 `Enterprise / Usage` tag。[Source: _bmad-output/planning-artifacts/architecture.md#611]

- [x] 保持 “多部门重复计入 + 不可加和免责声明” 的跨页面一致性，避免 5.6 重新解释口径 (AC: 1, 2, 3)
  - [x] 5.6 页面顶部必须继续固定展示 `enterprise.usage.multi_dept_disclaimer`，因为 PRD 和架构明确要求看板、详情、风险概览和 CSV 都维持相同免责声明；不能把它缩到 tooltip、空状态或可关闭 notice。[Source: _bmad-output/planning-artifacts/architecture.md#421; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#277]
  - [x] 概览内风险率公式需要可见，可用固定文案、tooltip 或与免责声明相邻的解释模块，但必须让管理员明确理解“部门间数值不可加和”。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#362; _bmad-output/test-artifacts/test-design-progress.md#R-008]
  - [x] 未归属桶要显式显示为“未归属”，而不是在趋势/Top 列表里默默丢弃；如果某个时间范围内没有未归属数据，可返回空值或 0，但不能伪造部门 ID。[Source: _bmad-output/planning-artifacts/epics.md#Story 5.6: 展示部门风险概览; _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]

- [x] 在 Default `enterprise-alerts` / `enterprise-usage` 现有边界内交付风险概览 UI，不新造平行模块 (AC: 1, 2, 3)
  - [x] 优先在 `web/default/src/features/enterprise-alerts/` 内新增风险概览视图或 tab，并与现有 `Risk Events / Deliveries / Alert Rules` 保持连贯；不要再新开一个平行的 `enterprise-risk-overview` feature，除非现有结构无法承载。[Source: _bmad-output/planning-artifacts/architecture.md#840; web/default/src/features/enterprise-alerts/index.tsx]
  - [x] 页面至少展示：Top 风险部门、风险率、风险次数、近期趋势，以及到风险事件列表的入口；风险率和趋势展示可复用 Epic 4 已引入的图表/统计组件，不新增新的图表库。[Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md; web/default/package.json]
  - [x] 跳转到风险事件列表时应带上时间范围和部门筛选上下文，优先复用 `enterprise-alerts` 路由和 search schema，而不是新造第二个 event-list 页面。[Source: _bmad-output/implementation-artifacts/5-2-query-and-filter-risk-events.md; web/default/src/features/enterprise-alerts/index.tsx]
  - [x] Default 六语种必须补齐新增文案，尤其是风险率公式、Top 风险部门、近期趋势、未归属、不可加和免责声明相关提示；文案进入现有 i18n 体系并执行 `bun run i18n:sync`。[Source: AGENTS.md#internationalization-i18n]

- [x] 为 Classic 提供最小风险概览入口，但不做与 Default 等深度重构 (AC: 1, 2, 3)
  - [x] 在 `web/classic/src/pages/Enterprise/Alerts.js` 或与现有企业用量页面协同的位置补最小风险概览区，至少能查看 Top 风险部门/风险率摘要和进入风险事件列表；不要为 Classic 新建大型独立页面体系。[Source: web/classic/src/pages/Enterprise/Alerts.js; _bmad-output/planning-artifacts/architecture.md#840]
  - [x] Classic 继续只补中英文必需文案，不要求六语种，但必须保证现有页面结构不被破坏，并补 smoke test 验证风险概览入口/关键词存在。[Source: AGENTS.md#internationalization-i18n; web/classic/src/pages/Enterprise/Alerts.smoke.test.js]

- [x] 补齐后端、API、前端测试，锁定风险率口径、免责声明和 drill-down 边界 (AC: 1, 2, 3)
  - [x] 在 `service/enterprise/usage_aggregation_test.go`、`service/enterprise/alert_test.go` 或新增测试文件中覆盖：风险次数与事件数量一致、多部门重复计入语义、未归属桶、风险率分子/分母计算、Top 风险部门排序、趋势窗口稳定。[Source: _bmad-output/test-artifacts/test-design-progress.md#ENT-USAGE-P1-002]
  - [x] 在 `controller/enterprise/alert_test.go` 与 `tests/api/enterprise_alerts_test.go` 增加 `department-summary` API 的参数校验、管理员权限、空结果、未归属行、以及 drill-down 入口上下文测试；继续确保 payload 不泄露敏感原文。[Source: _bmad-output/test-artifacts/test-design-progress.md#ENT-SEC-P0-002]
  - [x] 扩展 `web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx` 或新增 feature test，覆盖风险率公式可见、免责声明固定显示、Top 风险部门/趋势渲染、跳转到事件列表的筛选参数，以及多部门不可加和文案存在。[Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md#Tasks--Subtasks]
  - [x] Classic 至少补一条 smoke test，确认风险概览入口和关键文案不会让已有 Alerts 页面失效；如修改 usage/alert 共享服务，补相应最小 service 测试。[Source: web/classic/src/pages/Enterprise/Alerts.smoke.test.js]

## Dev Notes

- 5.6 是 Epic 5 的“风险概览读侧”，不是再做一遍事件列表，也不是重写用量聚合。最稳妥的实现是：风险分子来自 `enterprise_alert_events`，风险分母复用 Epic 4 的 `enterprise_usage_snapshots` 口径，然后把结果挂进现有 `enterprise-alerts` 页面。[Source: _bmad-output/planning-artifacts/architecture.md#1261; _bmad-output/implementation-artifacts/4-1-aggregate-department-usage-snapshots.md]
- PRD 明确规定多部门用户在每个相关部门都重复计入，而且部门间数值不可加和。这一口径在 4.2、4.3、4.4 和 5.6 是同一条规则，不能在风险概览里私自改成“按请求唯一归属”或“企业总量归一化”。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#277; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#362]
- 5.2 已经明确“风险率公式、Top 风险部门或趋势统计”不属于事件查询页，这正好说明 5.6 应单独承担这些统计展示职责；不要回头把 5.2 的 event list 扩到承担整块概览逻辑。[Source: _bmad-output/implementation-artifacts/5-2-query-and-filter-risk-events.md#Dev Notes]
- 4.2 / 4.3 已经建立了 usage summary/detail 的前端模式、disclaimer 固定展示位置、时间范围筛选和图表/表格展示习惯。5.6 应尽量复用这些交互和文案模式，减少新的心智负担。[Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md; _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md]
- 现有 `enterprise-alerts` feature 已在 5.5 扩成 events / deliveries / rules 的组合页，因此 5.6 最自然的增量是在这个 feature 里再加入风险概览，而不是另外开新 feature 或把风险概览塞进 `enterprise-usage` 总览页。[Source: web/default/src/features/enterprise-alerts/index.tsx; _bmad-output/planning-artifacts/architecture.md#840]
- 权限仍以 `EnterpriseAdmin()` 为真源。即便文案里是“安全/合规负责人”，当前代码里没有独立的更细粒度角色模型，5.6 不应擅自扩展 middleware 或角色表。[Source: router/enterprise-router.go; middleware/enterprise_admin.go]
- 5.6 的 drill-down 需要让管理员从概览跳到风险事件列表，而不是日志页；这意味着应优先复用 `enterprise-alerts` 搜索参数而不是再去拼一套新的 URL 语义。[Source: _bmad-output/implementation-artifacts/5-2-query-and-filter-risk-events.md; web/default/src/features/enterprise-alerts/index.tsx]

### Project Structure Notes

- 预计新增或重点修改文件：
  - `service/enterprise/alert.go`
  - `controller/enterprise/alert.go`
  - `controller/enterprise/alert_test.go`
  - `dto/enterprise/alert.go`
  - `router/enterprise-router.go`
  - `tests/api/enterprise_alerts_test.go`
  - `docs/openapi/api.json`
  - `web/default/src/features/enterprise-alerts/index.tsx`
  - `web/default/src/features/enterprise-alerts/api.ts`
  - `web/default/src/features/enterprise-alerts/types.ts`
  - `web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx`
  - `web/default/src/i18n/locales/en.json`
  - `web/default/src/i18n/locales/zh.json`
  - `web/default/src/i18n/locales/fr.json`
  - `web/default/src/i18n/locales/ru.json`
  - `web/default/src/i18n/locales/ja.json`
  - `web/default/src/i18n/locales/vi.json`
  - `web/classic/src/pages/Enterprise/Alerts.js`
  - `web/classic/src/pages/Enterprise/Alerts.smoke.test.js`
  - `web/classic/src/i18n/locales/en.json`
  - `web/classic/src/i18n/locales/zh.json`
- 复用但尽量不重构：
  - `service/enterprise/usage_aggregation.go`
  - `controller/enterprise/usage.go`
  - `dto/enterprise/usage.go`
  - `web/default/src/features/enterprise-usage/*`
- 保持不改或只读依赖：
  - `model/log.go`
  - `controller/log.go`
  - `relay/**`
  - `pkg/billingexpr/**`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 5.6: 展示部门风险概览]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#277]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#286]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#357]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#362]
- [Source: _bmad-output/planning-artifacts/architecture.md#421]
- [Source: _bmad-output/planning-artifacts/architecture.md#840]
- [Source: _bmad-output/planning-artifacts/architecture.md#1153]
- [Source: _bmad-output/planning-artifacts/architecture.md#1261]
- [Source: _bmad-output/planning-artifacts/architecture.md#1444]
- [Source: _bmad-output/test-artifacts/test-design-progress.md#R-008]
- [Source: _bmad-output/test-artifacts/test-design-progress.md#ENT-USAGE-P1-002]
- [Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]
- [Source: _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md]
- [Source: _bmad-output/implementation-artifacts/5-1-record-content-risk-events.md]
- [Source: _bmad-output/implementation-artifacts/5-2-query-and-filter-risk-events.md]
- [Source: controller/enterprise/usage.go]
- [Source: dto/enterprise/usage.go]
- [Source: service/enterprise/usage_aggregation.go]
- [Source: web/default/src/features/enterprise-usage/api.ts]
- [Source: web/default/src/features/enterprise-usage/index.tsx]
- [Source: web/default/src/features/enterprise-alerts/index.tsx]
- [Source: web/classic/src/pages/Enterprise/Alerts.js]
- [Source: AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
- [Source: AGENTS.md#internationalization-i18n]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- `bmad-story-automator` 已顺利完成 5.4 和 5.5，但 5.6 的自动 `create-story` 会话再次表现出和 5.5 类似的 create 阶段空转，因此转为手动接管。
- 已读取 `.agents/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md` 与 `.agents/skills/bmad-create-story/customize.toml`；`resolve_customization.py` 仍因当前 `python3` 缺少 `tomllib` 无法执行，按 skill fallback 手工处理。
- 已重新读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认 `5-6-display-department-risk-overview` 仍为 `backlog`，Epic 5 处于 `in-progress`。
- 已分析 Epic 5 的 Story 5.6、PRD FR20、Architecture 中 `FR-20` / `department-summary` / `enterprise.usage.multi_dept_disclaimer` / `enterprise-alerts` feature 约束，以及 4.2、4.3、5.1、5.2、5.5 的现有交付物和代码模式。
- 已确认当前仓库现状：存在风险事件事实源、告警规则/投递管理和用量 summary/detail 能力，但尚未实现 `/api/enterprise/alerts/department-summary`、风险率概览 DTO/service/controller，也尚未在 `enterprise-alerts` 中提供部门风险概览展示。
- 本 story 通过手动接管落盘，避免在 `5.6 create-story` 上重复进入和 5.5 同类的 create-session stall。

### Completion Notes List

- 手动接管 create-story，成功为 5.6 建立可开发的 story artifact。
- 5.6 的核心范围已锁定为：基于事件事实源 + 用量聚合事实源计算部门风险次数、风险率、趋势和 Top 风险部门，并提供跳回风险事件列表的入口。
- 明确 5.6 继续沿用多部门重复计入与不可加和免责声明，不得重新定义 FR20 口径。
- 明确 5.6 优先扩展现有 `enterprise-alerts` feature，而不是新造平行 risk overview feature。
- 已实现 `GetDepartmentRiskSummary` 及 `GET /api/enterprise/alerts/department-summary`，复用 `enterprise_usage_snapshots` 作为分母、`enterprise_alert_events` 快照作为分子，并保留多部门重复计入口径。
- Default 已在 `enterprise-alerts` 组合页中新增风险概览 tab，展示免责声明、风险率公式、Top 风险部门、趋势和 drill-down 到风险事件列表的入口。
- Classic 已补最小风险概览卡片与风险事件入口，中英文文案和 smoke test 均已更新。
- 已补齐 Go API/service 测试、Default feature test、Classic smoke test，并执行 `bun run i18n:sync` 与 Default `typecheck`。

### File List

- `_bmad-output/implementation-artifacts/5-6-display-department-risk-overview.md`
- `controller/enterprise/alert.go`
- `controller/enterprise/alert_test.go`
- `controller/enterprise/department_membership_test.go`
- `docs/openapi/api.json`
- `dto/enterprise/alert.go`
- `router/enterprise-router.go`
- `service/enterprise/alert.go`
- `service/enterprise/alert_test.go`
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

- 2026-05-31 06:54:32 +0800: 手动接管 `5.6 create-story`，在自动 create 会话未产出 artifact 的情况下，直接生成 5.6 故事文件并将其推进到可继续开发的 `ready-for-dev` 状态。
- 2026-05-31 08:58:00 +0800: 完成 5.6 部门风险概览实现，交付后端汇总 service / API、Default 与 Classic UI、OpenAPI 合同、i18n 与测试，并将故事推进到 `review`。
- 2026-05-31 07:39:12 +0800: 执行 story-automator review，修复风险事件查询与部门风险概览对 `to` 上边界处理不一致的问题（统一为半开区间 `[from, to)`），新增 service 回归测试，并补强 Classic 风险概览 smoke 断言以锁定 drill-down 上下文透传；复核后维持故事为 `done`。

## Senior Developer Review (AI)

### Findings

- High: `service/enterprise/alert.go` 中部门风险概览和事件列表对 `to` 使用了包含上边界的 `<=` 过滤，但 Epic 4 的用量快照窗口与趋势窗口使用的是半开区间 `[from, to)`。当风险事件时间戳恰好等于 `to` 时，会被错误计入概览/明细，导致 AC1 要求的“概览统计与事件明细一致”被破坏。已统一改为 `< to`。
- Medium: 缺少针对 `created_at == to` 边界值的风险概览回归测试，导致上述计数口径偏差可以悄悄回归。已在 `service/enterprise/alert_test.go` 增加专门覆盖。
- Medium: Classic smoke test 之前只验证“有风险概览入口”，没有锁定 drill-down 是否继续透传 `from` / `to` / `unassigned_only`，无法有效守护 AC2/AC3 所依赖的上下文复用。已补充断言。

### Validation

- 已复核 AC1-AC3 与实现一致。
- 已确认不存在未修复的 Critical 问题。
- 已运行定向测试：
  - `GOCACHE=/tmp/new-api-go-cache GOMODCACHE=/tmp/new-api-go-mod GOTMPDIR=/tmp/new-api-go-tmp go test ./service/enterprise -run 'TestAlertService'`
  - `GOCACHE=/tmp/new-api-go-cache GOMODCACHE=/tmp/new-api-go-mod GOTMPDIR=/tmp/new-api-go-tmp go test ./controller/enterprise -run 'TestDepartmentRiskSummaryAPI|TestAlertEventsAPI|TestAlertDeliveriesAPI|TestAlertDeliveryResendAPI'`
  - `GOCACHE=/tmp/new-api-go-cache GOMODCACHE=/tmp/new-api-go-mod GOTMPDIR=/tmp/new-api-go-tmp go test ./tests/api -run 'TestEnterpriseAlerts'`
  - `TMPDIR=/tmp/new-api-bun-tmp bun test web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx web/default/src/features/enterprise-usage/enterprise-usage.test.tsx web/classic/src/pages/Enterprise/Alerts.smoke.test.js`
- 已获得 Go service/controller/API 与 Default/Classic 前端定向测试通过证据。
