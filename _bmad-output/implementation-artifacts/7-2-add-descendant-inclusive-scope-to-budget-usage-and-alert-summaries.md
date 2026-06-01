---
baseline_commit: 81a6607704e894f3bc679071dba96493262686a6
---

# Story 7.2: 为预算/用量/告警 summary 增加包含子部门作用域

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员或经营负责人,
I want 在汇总视图中切换“仅当前部门 / 包含子部门”作用域,
so that 我可以按治理树层级查看预算、用量和风险概览。

## Acceptance Criteria

1. **Given** 管理员位于预算 summary、用量 summary 或告警 summary 页面  
   **When** 切换 `include_descendants=true`  
   **Then** 系统返回当前部门及其全部后代部门的聚合结果  
   **And** 页面明确展示当前作用域。
2. **Given** 页面未显式指定子部门作用域  
   **When** 首次加载  
   **Then** 默认保持“仅当前部门”  
   **And** 与既有口径保持兼容。
3. **Given** 管理员查看 detail list 或逐笔明细  
   **When** 页面处于 V1.3 首轮交付  
   **Then** 不要求自动扩展整棵子树  
   **And** 该限制在文案或交互上可被理解。

## Tasks / Subtasks

- [ ] 建立共享部门 scope resolver，复用邻接表树关系 (AC: 1, 2)
  - [ ] 在 `service/enterprise/department_scope_resolver.go` 或同包等价文件新增稳定入口，例如 `ResolveDepartmentScope(tenantId int, departmentId *int, includeDescendants bool)`，返回当前部门、全部后代部门 ID、scope label/metadata；不引入 closure table，不新增跨库不兼容递归 SQL。[Source: _bmad-output/planning-artifacts/architecture.md#Descendant Scope (V1.3)]
  - [ ] `department_id` 为空时保持既有“全量部门 summary”行为；`department_id` 非空且 `include_descendants=false` 时只返回当前部门；`include_descendants=true` 时返回当前部门及所有后代。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-22: 支持包含子部门的查询作用域]
  - [ ] resolver 必须按 `tenant_id` 过滤 `enterprise_departments`，兼容三层以上树、孤儿节点和环路防护；环路不能导致死循环，非法部门返回 `ErrDepartmentNotFound` 或现有等价错误。[Source: model/enterprise/department.go; service/enterprise/department.go]
  - [ ] 不复用 Story 7.1 的 owner resolution 做树解析；7.1 的 `PermissionService.departmentLineage` 是权限继承语义，7.2 需要 summary 查询作用域，二者可以共享低层 helper，但不能把权限优先级和数据查询 scope 混在一起。[Source: service/enterprise/permission.go; _bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md]

- [ ] 扩展预算 summary，只聚合预算池列表/概览，不扩展 wallet 明细 (AC: 1, 2, 3)
  - [ ] 在 `dto/enterprise/department_budget.go` 的 `DepartmentBudgetListQuery` 增加 `IncludeDescendants *bool form:"include_descendants,omitempty"`；可选布尔必须用指针，避免未传参数与显式 false 混淆。[Source: AGENTS.md#Rule 6: Upstream Relay Request DTOs — Preserve Explicit Zero Values]
  - [ ] `service/enterprise/department_budget.go` 的 `ListByDepartment` 接收 `IncludeDescendants` 语义，使用 resolver 得到部门 ID 集合后查询 `enterprise_department_budgets.department_id IN ?`；默认 false 时维持当前只查 `department_id = ?`。[Source: service/enterprise/department_budget.go]
  - [ ] 为预算列表返回项补充 `department_name` 或等价字段，确保 include descendants 时前端能区分预算池来自哪个后代部门；现有 `department_id` 必须继续保留，不能破坏旧 UI。[Source: dto/enterprise/department_budget.go; web/default/src/features/enterprise-organization/index.tsx]
  - [ ] `GetDepartmentBudget`、`GetDepartmentBudgetDetail`、quota allocation 创建/撤销和 wallet 明细仍只服务当前部门/当前预算池，不因 `include_descendants` 自动展开；这是 AC3 的明确边界。[Source: controller/enterprise/department_budget.go; service/enterprise/quota_allocation.go]

- [ ] 扩展用量 summary 与 CSV 当前筛选条件，保持 detail list 不变 (AC: 1, 2, 3)
  - [ ] 在 `dto/enterprise/usage.go` 的 `DepartmentUsageSummaryQuery` 和 `DepartmentUsageExportQuery` 增加部门 scope 参数与 `IncludeDescendants *bool form:"include_descendants,omitempty"`；后端 API 推荐使用 `department_id`，前端路由 search 可继续保留现有 `dept_id` 并在 API 层映射，避免破坏当前 URL 兼容；不要改 `DepartmentUsageDetailQuery` 的语义。[Source: dto/enterprise/usage.go; web/default/src/features/enterprise-usage/types.ts; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.2 包含子部门的查询作用域]
  - [ ] `service/enterprise/usage_aggregation.go` 的 `UsageSummaryQuery` 增加 `DeptId *int`、`IncludeDescendants bool`，在读取 `enterprise_usage_snapshots` 时对 resolved dept IDs 做过滤；无 `department_id` 时必须保留当前所有部门 + 未归属桶行为。[Source: service/enterprise/usage_aggregation.go]
  - [ ] include descendants 的用量汇总响应必须包含清晰的 scope metadata（例如 `scope_department_id`、`scope_department_name`、`include_descendants`、`scope_department_ids`）和 scope total；若继续返回子部门分行，子部门行只能作为明细/排行输入，不能替代 scope total。实现选择需在 Dev Agent Record 说明。[Source: _bmad-output/planning-artifacts/architecture.md#Descendant Scope (V1.3)]
  - [ ] 用户数去重必须跨当前部门及后代部门做集合去重；同一用户在多个后代部门出现时 scope total 的 `user_count` 不能简单相加。消费请求和 quota 仍遵循现有“多部门重复计入、部门间不可加和”口径。[Source: _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增)]
  - [ ] `usage_export.go` 应接收相同 scope 参数，使 CSV 与当前 summary 筛选一致；CSV 继续输出既有不可加和说明，不扩展最近消费日志入口。[Source: service/enterprise/usage_export.go]

- [ ] 扩展告警 summary，风险分子和分母使用同一部门 scope (AC: 1, 2, 3)
  - [ ] 在 `dto/enterprise/alert.go` 的 `DepartmentRiskSummaryQuery` 增加 `DepartmentId *int form:"department_id,omitempty"` 与 `IncludeDescendants *bool form:"include_descendants,omitempty"`。[Source: dto/enterprise/alert.go]
  - [ ] `service/enterprise/alert.go` 的 `DepartmentRiskSummaryQuery` 增加相同字段；`GetDepartmentRiskSummary` 调用 usage summary 时必须传递相同 scope，保证风险率分母与风险事件分子使用同一部门 ID 集合。[Source: service/enterprise/alert.go#GetDepartmentRiskSummary]
  - [ ] 风险事件来源是 `AlertEvent.DepartmentSnapshot` JSON；过滤 descendant scope 时应解析 snapshot 后匹配 resolved dept IDs，不直接使用数据库 JSON 专属操作符，避免 PostgreSQL/MySQL/SQLite 兼容风险。[Source: model/enterprise/alert_event.go; AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]
  - [ ] `event_entry` 跳转仍进入 events detail list，默认只带当前 scope 能表达的筛选；若 events detail list 暂不支持 descendant 展开，UI 文案必须说明“明细列表当前按单部门/事件筛选，不自动展开整棵子树”。[Source: _bmad-output/planning-artifacts/epics.md#Story 7.2: 为预算/用量/告警 summary 增加包含子部门作用域]

- [ ] 明确 summary API 鉴权，不无意扩大风险数据可见性 (AC: 1, 2, 3)
  - [ ] 预算 summary 继续通过 `/departments/:id/budgets` 的 `EnterpriseDepartmentAdmin("id")` 鉴权，部门负责人只能查看其可治理部门及后代 scope；本地 deny 仍必须阻断目标部门。[Source: router/enterprise-router.go; middleware/enterprise_dept_admin.go; _bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md]
  - [ ] 用量/告警 summary 若支持非 Admin 的经营负责人访问，必须要求显式 `department_id`，并在 controller/service 层调用 Story 7.1 的 `CanGovernDepartment` 或等价入口校验当前用户对 scope root 有治理权限；未传 `department_id` 的全租户 summary 继续 Admin-only。[Source: service/enterprise/permission.go; router/enterprise-router.go]
  - [ ] 告警 summary 涉及风险事件，不能因为 `include_descendants` 新参数让普通部门负责人读取全租户或非后代部门风险数据；权限拒绝应使用现有 i18n 错误，不泄露 scope 外部门是否存在。[Source: controller/enterprise/alert.go; i18n/keys.go]

- [ ] 更新 Default 前端三个 summary 面的 scope 切换和查询参数 (AC: 1, 2, 3)
  - [ ] `enterprise-organization` 的预算面板在当前部门上下文内增加“仅当前部门 / 包含子部门”切换；queryKey 必须包含 `includeDescendants`，mutation 后失效当前部门和 descendant summary 的相关 query，避免显示旧预算池列表。[Source: web/default/src/features/enterprise-organization/api.ts; web/default/src/features/enterprise-organization/index.tsx]
  - [ ] `enterprise-usage` 的 search schema 增加 `include_descendants`，summary query/export queryKey 和 API params 都带该字段；URL 可继续使用现有 `dept_id` 选中部门，但 API 请求需映射为后端约定的 department scope 参数；首次加载默认 false，detail panel 和 recent logs 不跟随展开。[Source: web/default/src/features/enterprise-usage/api.ts; web/default/src/features/enterprise-usage/index.tsx]
  - [ ] `enterprise-alerts` overview search/API 增加 `department_id` 与 `include_descendants`；overview 上展示当前部门与当前 scope，events/deliveries/rules tab 不因本故事自动改成整树明细。[Source: web/default/src/features/enterprise-alerts/api.ts; web/default/src/features/enterprise-alerts/index.tsx]
  - [ ] 使用现有 shadcn/Base UI 控件风格：二元切换优先用 Switch/segmented select，按钮带 lucide 图标；不要新增说明型落地页或营销式页面。[Source: AGENTS.md#Frontend guidance]
  - [ ] 新增前端文案进入 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，英文源字符串作为 key，并从 `web/default/` 运行 `bun run i18n:sync` 或记录本地阻塞原因。[Source: AGENTS.md#Internationalization (i18n)]

- [ ] 补齐后端、前端和边界测试 (AC: 1, 2, 3)
  - [ ] Service tests：新增 resolver 测试覆盖 root -> child -> grandchild、仅当前部门、包含后代、孤儿节点、环路防护、跨 tenant 不串数据。[Source: service/enterprise/department_test.go]
  - [ ] Budget tests：`ListByDepartment` / controller 覆盖 include false 默认兼容、include true 返回当前 + 后代预算池、返回 department name、detail API 不展开 wallet 明细。[Source: service/enterprise/department_budget_test.go; controller/enterprise/department_budget_test.go]
  - [ ] Usage tests：覆盖 `department_id + include_descendants` 的 snapshot 过滤、scope total 用户去重、未归属桶在 scoped 查询中不被错误混入、CSV 参数与 summary 一致。[Source: service/enterprise/usage_aggregation_test.go; controller/enterprise/usage_test.go]
  - [ ] Alert tests：覆盖风险事件 snapshot 在后代部门时计入祖先 scope、非后代部门不计入、风险率分母复用同 scope usage summary、trend 不把 scope 外事件混入。[Source: service/enterprise/alert_test.go; controller/enterprise/alert_test.go]
  - [ ] Frontend tests：`enterprise-organization`、`enterprise-usage`、`enterprise-alerts` 分别覆盖默认 false、切换 true 后 queryKey/API params、页面展示当前作用域、detail list 文案边界。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx; web/default/src/features/enterprise-usage/enterprise-usage.test.tsx; web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx]
  - [ ] Boundary tests：继续确认本故事不修改 `relay/**`、`pkg/billingexpr/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`。[Source: service/enterprise/boundary_test.go; _bmad-output/planning-artifacts/architecture.md#Anti-Patterns]

## Dev Notes

- Story 7.2 是 FR22 的 scope 能力故事，不是预算委派、allocation 调整或额度申请故事。不要提前实现 Story 7.3-7.5 的 mutation 流程，也不要新增部门到部门 delegation ledger。[Source: _bmad-output/planning-artifacts/epics.md#Epic 7: 分层预算治理与额度审批]
- “包含子部门”第一版只覆盖预算 summary、用量 summary、告警 summary。detail list、成员明细、逐笔日志、wallet 明细不要求扩展整棵子树；实现时必须在 API 和 UI 上保持这个边界。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.2 包含子部门的查询作用域]
- 当前后端已有三个主要落点：预算 `ListDepartmentBudgets` 只查当前 `department_id`，用量 `GetDepartmentSummary` 当前返回全量部门汇总，告警 `GetDepartmentRiskSummary` 当前调用用量 summary 后解析所有 alert events。7.2 应在这些现有入口上演进，不新增平行 summary API。[Source: controller/enterprise/department_budget.go; service/enterprise/usage_aggregation.go; service/enterprise/alert.go]
- 预算 summary 的“聚合结果”更接近预算池列表/概览：可以返回当前部门及后代部门的预算池行，并加 `department_name` 辅助展示；不要把不同预算池的 balance/subscription 数值强行合成一个父池，因为两类预算池语义不同且 Story 7.3 才处理跨层预算委派。[Source: model/enterprise/department_budget.go; _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- 用量与告警 summary 需要小心“部门间不可加和”口径。现有聚合按成员消费视角重复计入，多部门用户消费在所有归属部门各计一次；include descendants 的 scope total 如果需要展示用户数，应跨 scope 去重，但请求数/quota 仍按既有快照口径汇总。[Source: _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增)]
- 告警风险率必须保证分子/分母同 scope。分母来自 usage snapshots，分子来自 alert event department snapshot；只改其中一边会制造错误风险率。[Source: service/enterprise/alert.go#GetDepartmentRiskSummary]
- 三库兼容是硬约束。部门树第一版继续用邻接表在 Go 层解析或 GORM 普通查询，不使用 recursive CTE、JSONB 操作符、MySQL JSON 函数或闭包表迁移。[Source: AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]
- 所有 JSON marshal/unmarshal 继续走 `common.Marshal`、`common.UnmarshalJsonStr` 等 wrapper。Alert snapshot、Usage model distribution、UserIds 都已有 model helper，不要直接调用 `encoding/json` marshal/unmarshal。[Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go]
- Story 7.1 已完成并提交为 `81a660770 feat(story-7.1): 建立部门负责人权限模型与生效优先级`。本故事可以依赖其部门负责人/中间件结果，但 summary 读取仍由后端路由权限控制；前端按钮可见性不是安全边界。[Source: _bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md; router/enterprise-router.go]
- 当前 `router/enterprise-router.go` 的 usage/alerts summary 仍使用 `EnterpriseAdmin()`，预算接口使用 `EnterpriseDepartmentAdmin("id")`。本故事实现时必须在“全租户 summary”和“指定 department scope summary”之间做显式权限分支：全租户继续 Admin-only；指定部门 scope 可以开放给对该 scope root 有治理权限的部门负责人，但必须补 controller / middleware 测试，尤其是 alert 风险数据不可越权。[Source: router/enterprise-router.go; middleware/enterprise_dept_admin.go; _bmad-output/planning-artifacts/architecture.md#Permission UI]
- 不触碰 relay、billing expression、`logs` 表结构或 AI API 协议面。本故事只读既有 `logs` 派生的 `enterprise_usage_snapshots` 和企业 alert/budget 表。[Source: _bmad-output/planning-artifacts/epics.md#Additional Requirements]

### Project Structure Notes

- 重点预计修改：
  - `dto/enterprise/department_budget.go`
  - `dto/enterprise/usage.go`
  - `dto/enterprise/alert.go`
  - `service/enterprise/department_scope_resolver.go`（新增或等价同包文件）
  - `service/enterprise/department_budget.go`
  - `service/enterprise/usage_aggregation.go`
  - `service/enterprise/usage_export.go`
  - `service/enterprise/alert.go`
  - `controller/enterprise/department_budget.go`
  - `controller/enterprise/usage.go`
  - `controller/enterprise/alert.go`
  - `web/default/src/features/enterprise-organization/api.ts`
  - `web/default/src/features/enterprise-organization/types.ts`
  - `web/default/src/features/enterprise-organization/index.tsx`
  - `web/default/src/features/enterprise-usage/api.ts`
  - `web/default/src/features/enterprise-usage/types.ts`
  - `web/default/src/features/enterprise-usage/hooks/use-department-usage-summary.ts`
  - `web/default/src/features/enterprise-usage/index.tsx`
  - `web/default/src/features/enterprise-alerts/api.ts`
  - `web/default/src/features/enterprise-alerts/types.ts`
  - `web/default/src/features/enterprise-alerts/index.tsx`
  - `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`
  - 对应 `*_test.go` 与 frontend test 文件
- 保持不改：
  - `relay/**`
  - `pkg/billingexpr/**`
  - `model/log.go`
  - `controller/log.go`
  - `router/relay-router.go`
  - `docs/openapi/relay.json`
  - `relay/channel/**`
- 当前未发现仓库内 `project-context.md`；已按 skill persistent facts 检索，结果为空。

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 7.2: 为预算/用量/告警 summary 增加包含子部门作用域]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic 7: 分层预算治理与额度审批]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-22: 支持包含子部门的查询作用域]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.2 包含子部门的查询作用域]
- [Source: _bmad-output/planning-artifacts/architecture.md#Descendant Scope (V1.3)]
- [Source: _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增)]
- [Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- [Source: _bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md]
- [Source: controller/enterprise/department_budget.go]
- [Source: controller/enterprise/usage.go]
- [Source: controller/enterprise/alert.go]
- [Source: service/enterprise/department.go]
- [Source: service/enterprise/department_budget.go]
- [Source: service/enterprise/usage_aggregation.go]
- [Source: service/enterprise/usage_export.go]
- [Source: service/enterprise/alert.go]
- [Source: service/enterprise/permission.go]
- [Source: dto/enterprise/department_budget.go]
- [Source: dto/enterprise/usage.go]
- [Source: dto/enterprise/alert.go]
- [Source: router/enterprise-router.go]
- [Source: web/default/src/features/enterprise-organization/api.ts]
- [Source: web/default/src/features/enterprise-organization/index.tsx]
- [Source: web/default/src/features/enterprise-usage/api.ts]
- [Source: web/default/src/features/enterprise-usage/index.tsx]
- [Source: web/default/src/features/enterprise-alerts/api.ts]
- [Source: web/default/src/features/enterprise-alerts/index.tsx]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-01 17:17:51 +0800: create-story 启动；已按用户要求读取 `.agents/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`。
- 2026-06-01 17:17:51 +0800: `_bmad/scripts/resolve_customization.py` 因当前 `python3` 缺少 Python 3.11+ 的 `tomllib` 失败；已按 skill fallback 手工读取 `.agents/skills/bmad-create-story/customize.toml`，确认无项目/用户 override 文件，persistent facts 仅包含 `file:{project-root}/**/project-context.md`。
- 2026-06-01 17:17:51 +0800: 已读取 `_bmad/bmm/config.yaml`，确认 `project_name=new-api`、`user_name=hth`、`communication_language=Chinese`、`document_output_language=Chinese`。
- 2026-06-01 17:17:51 +0800: 已检索 `project-context.md`，仓库内未发现匹配文件。
- 2026-06-01 17:17:51 +0800: 已完整读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认目标 story key 为 `7-2-add-descendant-inclusive-scope-to-budget-usage-and-alert-summaries`，原状态为 `backlog`，Epic 7 已是 `in-progress`。
- 2026-06-01 17:17:51 +0800: 已执行 discover-inputs：加载 `epics.md`、`architecture.md`、`prds/prd-new-api-2026-05-27/prd.md`、`prds/prd-new-api-2026-05-27/addendum.md` 的 FR22 / Epic 7 / descendant scope 相关段落；未发现独立 UX 文档，UX 要求来自 PRD/Architecture。
- 2026-06-01 17:17:51 +0800: 已读取上一故事 `_bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md`，确认 7.1 已完成并提交 `81a660770`，本故事需要继承其权限模型但不能混淆权限 lineage 与查询 scope。
- 2026-06-01 17:17:51 +0800: 已读取相关现有代码：`controller/enterprise/{department_budget,usage,alert}.go`、`service/enterprise/{department,department_budget,usage_aggregation,usage_export,alert,permission}.go`、`dto/enterprise/{department_budget,usage,alert}.go`、`router/enterprise-router.go`、Default 三个 feature 的 `api.ts` / `index.tsx` / `types.ts`。
- 2026-06-01 17:17:51 +0800: 已读取测试落点：`service/enterprise/department_budget_test.go`、`service/enterprise/usage_aggregation_test.go`、`service/enterprise/alert_test.go`、`controller/enterprise/*_test.go`、Default enterprise feature tests。
- 2026-06-01 17:17:51 +0800: 已读取 `go.mod` 与 `web/default/package.json`，确认当前 Go module、Gin/GORM、React 19、TanStack Query、Rsbuild、Bun 脚本等本地版本；本故事不需要外部最新 API 研究。

### Completion Notes List

- 已创建 Story 7.2 故事文档，状态为 `ready-for-dev`。
- 已明确 FR22 的默认行为：未传 `include_descendants` 保持“仅当前部门”；仅 summary/overview 支持 descendant scope，不扩展 detail list / wallet 明细 / 逐笔日志。
- 已写清共享 scope resolver、预算 summary、用量 summary/CSV、告警 summary、Default 前端和测试要求。
- 已在 checklist 复核后补强 scope total / metadata 契约和 usage/alert scoped summary 鉴权边界，防止开发时误做子部门行相加或扩大风险数据可见性。
- 已按 checklist 进行自检：故事包含用户故事、AC、任务拆解、架构约束、当前代码状态、测试要求、文件位置、前后故事依赖、边界约束与来源引用。

### File List

- `_bmad-output/implementation-artifacts/7-2-add-descendant-inclusive-scope-to-budget-usage-and-alert-summaries.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

- 2026-06-01 17:17:51 +0800: 创建 Story 7.2 上下文文档，准备进入开发。
- 2026-06-01 17:20:00 +0800: checklist 复核后补强 scoped summary 响应契约与鉴权边界。
