---
baseline_commit: 553d9ac0f4c41fbf4fb0c7747e31db9a2319fc812
---

# Story 7B.2: 重构额度申请预算池选择与已选摘要

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 普通员工或负责人,
I want 在 quota request 表单里清楚区分各个可选预算池，并在提交前再次确认已选目标,
so that 我不会因为预算池名称过弱或信息缺失而误选。

## Acceptance Criteria

1. **Given** 用户打开钱包 quota request 表单或企业组织工作台 quota request 表单  
   **When** 系统展示预算池选项  
   **Then** 每个选项至少显示部门名、预算池标识或名称、预算类型、剩余额度与状态  
   **And** 不再只有近似的单行短标签。
2. **Given** 用户已选择预算池  
   **When** 表单进入提交前状态  
   **Then** 页面显示“已选请求范围/已选预算池摘要”  
   **And** 摘要中包含部门和具体预算池身份，而不是只显示部门。

## Tasks / Subtasks

- [x] 统一 quota request 预算池选项展示模型，复用现有 capability 响应字段 (AC: 1)
  - [x] 在 Default 前端抽取或复用一个轻量 helper，基于 `QuotaRequestCapabilityBudgetItem` / `DepartmentBudgetItem` 生成可辨识预算池展示信息：部门名、`Budget #id`、预算类型、剩余额度、状态。
  - [x] 预算类型必须走既有 `Balance Budget` / `Subscription Budget` 文案或同等稳定映射，不直接展示裸 `balance` / `subscription`。
  - [x] 状态必须走既有 `enterpriseBudgetStatusLabel` 或同等稳定映射，不直接展示裸 `active` / `paused` 等内部状态。
  - [x] 剩余额度使用现有 `formatNumber` 或项目 quota 格式化 helper；不要引入金额显示或 quota/$ 切换，本能力属于 Story 7B.5。

- [x] 重构钱包侧 quota request 表单的预算池下拉与已选摘要 (AC: 1, 2)
  - [x] 更新 `web/default/src/features/wallet/components/employee-quota-request-card.tsx` 中 `formatBudgetOption` 及下拉内容，使每个选项不再只有 `{{department}} · Budget #{{budgetId}}`，而是至少包含部门、预算池 ID、类型、remaining、status。
  - [x] 钱包侧 `Selected request scope` 卡片必须在已选预算池后显示具体预算池身份与关键信息；未选预算池时保留清晰空态，不能只显示当前部门。
  - [x] 切换部门时继续清空 `department_budget_id`，避免上一部门预算池残留；当前 `handleDepartmentChange` 的 reset 行为必须保留。
  - [x] 保持钱包入口继续调用现有 `/api/enterprise/users/:id/departments`、`/api/enterprise/quota-requests/capability/:departmentId` 和 `/api/enterprise/quota-requests`，不新增后端接口。

- [x] 重构企业组织工作台 quota request 表单的预算池下拉与已选摘要 (AC: 1, 2)
  - [x] 更新 `web/default/src/features/enterprise-organization/index.tsx` 中 `Employee Quota Requests` 表单的 `Target Budget Pool` 下拉，使用同一可辨识展示规则。
  - [x] 在企业组织工作台表单内补充“已选请求范围/已选预算池摘要”，摘要必须包含当前部门与具体预算池 ID/类型/remaining/status。
  - [x] 保留当前工作台的 `quotaRequestCapabilityQuery`、`budgetListQuery` fallback 和 `quotaRequestForm` 同步逻辑；不要改成手输 `department_id` / `budget_id`。
  - [x] 不要借本故事重构 `QuotaRequestTable`、审批状态机、通知投递语义、wallet 过期时间或金额视角；这些由 7B.1、7B.3、7B.4、7B.5 分别承接。

- [x] 补齐 Default 前端 i18n 与组件测试 (AC: 1, 2)
  - [x] 如新增英文源 key，必须同步 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，保持 flat JSON 结构。
  - [x] 扩展 `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx`，覆盖预算池选项/摘要包含部门名、Budget ID、类型、remaining、status，且部门切换后无效预算池会被清空。
  - [x] 扩展 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`，覆盖组织工作台 quota request 表单的预算池选项与已选摘要，不只验证 schema。
  - [x] 保留既有 query key 精确失效，不使用 `invalidateQueries({ queryKey: ['enterprise'] })` 全刷。

- [x] 运行并记录验证 (AC: 1, 2)
  - [x] `cd web/default && bun test src/features/wallet/components/employee-quota-request-card.test.tsx`
  - [x] `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx`
  - [x] `cd web/default && bun run typecheck`
  - [x] `cd web/default && bun run i18n:sync`
  - [x] `cd web/default && bun run test:e2e`（如耗时不可接受，至少记录未运行原因，并保留上面的定向测试）

## Dev Notes

### 需求与业务上下文

- Epic 7B 是 Epic 7 / 7A 后的企业治理 follow-up，目标包含治理通知语义解耦、预算池可辨识性、i18n 残留、一次性 wallet 展示和金额视角修正；本故事只处理 quota request 的预算池选择和已选摘要。[Source: `_bmad-output/planning-artifacts/epics.md`#Epic 7B]
- Story 7B.2 的验收来自 FR25 follow-up：钱包 quota request 表单和企业组织工作台 quota request 表单都必须展示可辨识预算池选项，并在提交前显示已选范围/已选预算池摘要。[Source: `_bmad-output/planning-artifacts/epics.md`#Story 7B.2]
- 2026-06-03 correct-course 明确当前问题不是“没有预算池选择”，而是“预算池虽然可选但不可辨识”；提交前摘要必须帮助用户确认目标部门和具体预算池。[Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Change H]
- 架构 V1.3B 的额度申请审批流要求表单加载“可辨识预算池选项”，字段至少包括部门名、预算池标识或名称、预算类型、剩余额度、状态；审批和自动分配仍复用既有 quota request / quota allocation 后端链路。[Source: `_bmad-output/planning-artifacts/architecture.md`#Data Flow and Integration Patterns]
- 后端 capability API 已具备本故事需要的字段：`QuotaRequestCapabilityBudgetItem` 返回 `id`、`department_name`、`type`、`status`、`remaining`、`cycle_quota`、`total_quota`、`usage_ratio` 等。除非发现字段缺失，不应新增 API 或改 schema。[Source: `dto/enterprise/quota_request.go`; `service/enterprise/quota_request.go`#GetCapability]

### 当前实现状态（已读 UPDATE 文件）

- `web/default/src/features/wallet/components/employee-quota-request-card.tsx`
  - 钱包侧入口已经存在，使用 `getUserDepartments` 获取当前用户 active membership，再按选中部门调用 `getQuotaRequestCapability`。
  - 当前 `formatBudgetOption` 只返回 `{{department}} · Budget #{{budgetId}}`；没有显示预算类型、剩余额度、状态。
  - 当前 `Selected request scope` 卡片只显示 `currentDepartment.department_name` 或 `No department selected`；即使用户已选预算池，也没有显示具体预算池身份。
  - `handleDepartmentChange` 会设置 department/tenant 并清空 `department_budget_id`，这正是防止跨部门误选的关键行为，必须保留。

- `web/default/src/features/enterprise-organization/index.tsx`
  - 企业组织工作台已有 `Employee Quota Requests` 表单，使用 `quotaRequestCapabilityQuery` 获取可申请预算池，fallback 到 `budgetListQuery.data?.items`。
  - 当前 `Target Budget Pool` 下拉同样只渲染 `{{department}} · Budget #{{budgetId}}`。
  - 当前表单顶部只显示 requester 和 `Target Department {{departmentName}}`，没有已选预算池摘要。
  - 文件内已有 `enterpriseBudgetStatusLabel`、`formatNumber`、`DepartmentBudgetListCard`、`DepartmentBudgetOverviewCard` 等预算展示工具和模式；本故事应复用这些现有映射，不重复发明状态/格式化体系。

- `web/default/src/features/enterprise-organization/types.ts`
  - `QuotaRequestCapabilityBudgetItem` 当前等同 `DepartmentBudgetItem`，字段足够支撑本故事展示。
  - 不需要在前端类型里新增仅 UI 用字段；优先通过 helper 从现有 item 派生 label 和摘要。

- `web/default/src/features/enterprise-organization/api.ts`
  - quota request query key 已按 `['enterprise', 'organization', 'quota-request', departmentId, tenantId]` 组织。
  - 钱包侧通过 `quotaRequestQueryScopeKey(...), 'capability', 'wallet-entry'` 细分缓存；组织页通过同一 scope 加 `capability`。
  - mutation 成功后已有 quota request scope 和治理 timeline/notification 精确失效；本故事不应扩大缓存失效范围。

- `dto/enterprise/quota_request.go` / `service/enterprise/quota_request.go`
  - capability 响应只返回 active 且属于当前部门的预算池；这符合“可申请预算池”窄读语义。
  - `SubmitQuotaRequestRequest` 已使用 `*int64` / `omitempty` 保存显式零值语义；本故事不应改动 request DTO 或审批状态机。

### Previous Story Intelligence

- Story 7B.1 已完成治理动作结果与通知投递结果解耦，并改动 `web/default/src/features/enterprise-organization/index.tsx` 的治理活动展示。7B.2 修改同一大文件时必须保留 7B.1 的 `unconfigured` delivery 状态 label、neutral 样式和“不显示 resend”的行为。[Source: `_bmad-output/implementation-artifacts/7B-1-decouple-governance-action-result-from-notification-delivery.md`#Completion Notes List]
- Story 7A.2 已把员工 quota request 入口移到钱包用户流，并要求提交成功后精确失效 quota request、governance timeline、notification query。7B.2 只能增强同一钱包入口的选择体验，不要把用户重新导回企业组织工作台。[Source: `_bmad-output/implementation-artifacts/7a-2-move-employee-quota-request-entry-to-balance-wallet-user-flow.md`#Senior Developer Review]
- Story 7A.3 已确认同一部门允许多个不同类型预算池并存，类型兼容性只作用于当前选中预算池。因此可辨识选项必须显示 type，不能继续暗示“一个部门只有一个预算池”。[Source: `_bmad-output/implementation-artifacts/7a-3-relax-budget-pool-type-locking-to-per-budget-invariance.md`#Completion Notes List]
- Story 6.3 已把成员、预算池和 quota allocation 主流程收拢到当前部门工作区，并强调部门/预算/成员切换时要清理旧上下文。7B.2 的摘要增强必须延续这个上下文驱动模式。[Source: `_bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md`#Dev Notes]
- Story 3.5 已将预算视图升级为预算池列表 + 详情 + wallet 明细，并要求多预算池排序、状态和阈值可见。7B.2 不要回退成单预算池或裸 ID 表单。[Source: `_bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md`#Dev Notes]

### Git Intelligence Summary

- 最近提交 `553d9ac0f feat(story-7B.1): 解耦治理动作结果与通知投递结果` 已完成 7B.1，并触碰 `web/default/src/features/enterprise-organization/index.tsx`、企业组织测试和全部 Default locale。开发 7B.2 时先查看当前文件，避免覆盖 7B.1 的状态映射与测试。
- 当前工作树存在与本故事无关的未提交改动（BMAD story automator、hooks、测试摘要等）。Dev agent 不得回退、清理或重写这些无关变更。

### Architecture Compliance

- Default 前端继续使用 React 19、TypeScript、TanStack Query、i18next、Base UI/shadcn-style primitives、Tailwind 和 lucide icons；不新增依赖。[Source: `web/default/package.json`; `_bmad-output/planning-artifacts/architecture.md`#Frontend Architecture]
- 前端 feature 边界保持不变：钱包入口在 `web/default/src/features/wallet/components/employee-quota-request-card.tsx`；企业组织工作台在 `web/default/src/features/enterprise-organization/index.tsx`；共享 API/类型继续从 `enterprise-organization/api.ts` / `types.ts` 引入。[Source: `_bmad-output/planning-artifacts/architecture.md`#Source Organization]
- Query key 必须保持 feature-scoped 精确模式：`['enterprise', 'organization', ...]`。不要用 `invalidateQueries({ queryKey: ['enterprise'] })` 全刷。[Source: `_bmad-output/planning-artifacts/architecture.md`#State & Data Fetching]
- 新文案使用 `t('English source string')`，translation JSON 为 flat key；新增 key 必须覆盖 en、zh、fr、ru、ja、vi。[Source: `AGENTS.md`#Internationalization]
- 本故事不改数据库、不改 Go migration、不改 relay、不改 billing expression、不改 upstream DTO。若开发时意外触碰 `pkg/billingexpr/**`，必须先读 `pkg/billingexpr/expr.md`，但本故事正常不需要。[Source: `AGENTS.md`#Rule 7; `_bmad-output/planning-artifacts/architecture.md`#No Relay-Path Department Deduction]
- `new-api` 与 `QuantumNous` 是受保护标识；不要删除、替换或改名相关版权、品牌和元数据。[Source: `AGENTS.md`#Rule 5]

### Library / Framework Requirements

- 使用现有 `Select` / `SelectItem` / `Card` / `Badge` / `Input` / `Textarea` / `Empty` 等 UI primitives；预算池选项可以在 `SelectItem` 内使用多行紧凑布局，但必须保证移动端不溢出、不遮挡。
- 使用 `formatNumber` 或既有 quota 格式化 helper；不要自行引入 `Intl` 包装库或新数字格式化依赖。
- 使用现有 `enterpriseBudgetStatusLabel` 处理状态；如果 helper 当前不导出，优先在同文件内复用或抽出到小 helper，避免跨 feature 循环 import。
- 不需要 web research 或依赖升级；package 已固定 React `^19.2.6`、TanStack Query `^5.100.14`、i18next `^26.2.0`、Base UI `^1.5.0`。

### File Structure Requirements

Expected UPDATE files:

- `web/default/src/features/wallet/components/employee-quota-request-card.tsx` — 钱包侧预算池选项与已选摘要。
- `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx` — 钱包侧选项/摘要/无效选择测试。
- `web/default/src/features/enterprise-organization/index.tsx` — 组织工作台 quota request 表单选项与摘要。
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx` — 组织工作台选项/摘要测试与 i18n required key 补充。
- `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` — 仅当新增英文源 key 时追加翻译。

Likely no changes needed:

- `controller/enterprise/quota_request.go`、`service/enterprise/quota_request.go`、`dto/enterprise/quota_request.go` — capability 已返回所需字段。
- `router/enterprise-router.go` — 现有 quota request capability、submit、list、decision 路由足够。
- `model/enterprise/*`、`model/main.go` — 无 schema 变更。
- `pkg/billingexpr/**`、`relay/**`、`model/subscription.go` — 本故事只改变展示选择体验。

### Project Structure Notes

- 本故事是 Default 前端体验增强，不是后端能力新增。实现应收敛在现有钱包卡片和企业组织工作台，不新增页面、不移动路由、不创建第二套 quota request feature。
- 预算池“名称”当前没有独立 name 字段；可使用稳定身份 `Budget #{{id}}` 作为预算池标识，并搭配部门名、类型、remaining、status 达成 AC。不要为了“名称”临时改表。
- 两个入口应尽量共享同一预算池展示 helper 或同等逻辑，避免钱包侧和组织页显示口径再次分叉。
- 摘要是提交前确认区，不是说明文档；文案应短、可扫描，并显示实际选择，不要添加长段使用说明。

### Testing Requirements

Minimum targeted validation:

```bash
cd web/default && bun test src/features/wallet/components/employee-quota-request-card.test.tsx
cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx
cd web/default && bun run typecheck
cd web/default && bun run i18n:sync
```

Recommended assertions:

- 钱包侧预算池选项包含 `Engineering`、`Budget #31`、`Balance Budget`、`Remaining` / remaining 数值、`Active` 或本地化状态。
- 钱包侧已选摘要在选中预算池后同时包含部门名和具体预算池身份；未选预算池时不伪造摘要。
- 钱包侧切换部门或 capability budgets 变化后，旧 `department_budget_id` 被清空，不会提交上一部门预算池。
- 组织工作台 quota request 表单的预算池选项同样包含部门名、预算池 ID、类型、remaining、status。
- 组织工作台已选摘要显示当前部门和具体预算池，不只显示 `Target Department {{departmentName}}`。
- 新增文案在 en、zh、fr、ru、ja、vi 均存在；`bun run i18n:sync` 后无未同步 key。
- 7B.1 的 governance delivery `unconfigured` 状态和 resend 行为测试继续通过。

### Anti-Patterns to Avoid

- 不要继续使用只有 `{{department}} · Budget #{{budgetId}}` 的短标签作为唯一预算池信息。
- 不要只在钱包侧修复而遗漏企业组织工作台，或反过来只修组织页。
- 不要让已选摘要只显示部门名；AC 明确要求包含具体预算池身份。
- 不要把预算池选择退化成手输 ID。
- 不要新增后端 API、数据库字段或预算池 name 字段来完成本故事；现有 capability 数据足够。
- 不要混入 7B.3 的全量 i18n 收尾、7B.4 的 wallet 过期时间语义、7B.5 的金额显示/单位切换。
- 不要改变 quota request submit/approve/reject 状态机，或改变 allocation 自动分配事务。
- 不要移除或覆盖受保护的 `new-api`、`QuantumNous` 标识和版权头。

### References

- [Source: `_bmad-output/planning-artifacts/epics.md`#Story 7B.2]
- [Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Change H]
- [Source: `_bmad-output/planning-artifacts/architecture.md`#Data Flow and Integration Patterns]
- [Source: `web/default/src/features/wallet/components/employee-quota-request-card.tsx`#formatBudgetOption / Selected request scope]
- [Source: `web/default/src/features/enterprise-organization/index.tsx`#Employee Quota Requests]
- [Source: `web/default/src/features/enterprise-organization/types.ts`#QuotaRequestCapabilityBudgetItem]
- [Source: `web/default/src/features/enterprise-organization/api.ts`#quotaRequestQueryScopeKey / getQuotaRequestCapability]
- [Source: `dto/enterprise/quota_request.go`#QuotaRequestCapabilityBudgetItem]
- [Source: `service/enterprise/quota_request.go`#GetCapability]
- [Source: `_bmad-output/implementation-artifacts/7B-1-decouple-governance-action-result-from-notification-delivery.md`#Completion Notes List]
- [Source: `_bmad-output/implementation-artifacts/7a-2-move-employee-quota-request-entry-to-balance-wallet-user-flow.md`#Senior Developer Review]
- [Source: `_bmad-output/implementation-artifacts/7a-3-relax-budget-pool-type-locking-to-per-budget-invariance.md`#Completion Notes List]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-03 14:26:31 +0800: BMAD create-story workflow executed for Story 7B.2 in YOLO mode.
- 2026-06-03: Read `.agents/skills/bmad-create-story/SKILL.md`, `discover-inputs.md`, `template.md`, and `checklist.md`.
- 2026-06-03: Workflow resolver failed because local `python3` lacks Python 3.11 `tomllib`; customization was resolved manually per skill fallback. Default workflow had no activation prepend/append and one persistent fact glob.
- 2026-06-03: Loaded `_bmad/bmm/config.yaml`: `user_name=hth`, communication/document language Chinese, planning artifacts `_bmad-output/planning-artifacts`, implementation artifacts `_bmad-output/implementation-artifacts`.
- 2026-06-03: Persistent facts glob found no `project-context.md` files.
- 2026-06-03: Input discovery loaded relevant sections from `epics.md`, `sprint-change-proposal-2026-06-03.md`, `architecture.md`, previous story `7B.1`, git history, sprint status, and current wallet/enterprise organization quota request implementation files.
- 2026-06-03: Confirmed story key from sprint status: `7b-2-redesign-budget-pool-selection-and-selected-summary`, previous status `backlog`.
- 2026-06-03: Checklist validation applied: story includes exact ACs, current-state notes for UPDATE files, previous story intelligence, file boundaries, anti-patterns, i18n/queryKey guardrails, and targeted tests.

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- Story scope is intentionally frontend-focused because quota request capability already returns the required budget fields.
- Primary current defect is identified in both Default quota request form surfaces: budget options only show a short department + budget ID label, and the wallet selected-scope summary only shows department.
- Dev agent must enhance both wallet and enterprise organization workspace entries while preserving existing query scopes, submit API, department-change reset behavior, and 7B.1 notification semantics.
- Implemented a shared quota request budget display helper and shared option/summary components so wallet and organization quota request forms use the same department + Budget ID + type + remaining + status presentation.
- Wallet quota request selected scope now shows the concrete selected budget pool; stale selected budget IDs are cleared when the selected department capability no longer includes them.
- Enterprise organization quota request form now uses capability/fallback budget options with the same identifiable display and adds a selected request scope summary.
- `bun run i18n:sync` was executed; locale files were rewritten by the sync tool while preserving the required flat JSON translation coverage.

### Senior Developer Review (AI)

Reviewer: GPT-5 Codex on 2026-06-03 14:44:14 +0800

#### Outcome

Approved after automatic fixes. No CRITICAL issues remain. Story status set to `done`.

#### Findings Fixed

- [MEDIUM] Story File List did not include the actual application files changed for the implementation, only the story and sprint tracking files. Fixed by updating this Dev Agent Record file list to include the wallet, enterprise organization, shared display helper/component, tests, and locale files.
- [MEDIUM] The initial implementation shared only display text generation; both quota request surfaces duplicated option/summary JSX. Fixed by adding `web/default/src/features/enterprise-organization/quota-request-budget-display-components.tsx` and using it from both wallet and organization forms.
- [MEDIUM] Added tests verified helper output but did not render the actual option and summary components required by AC1/AC2. Fixed by extending wallet and enterprise organization tests to render `QuotaRequestBudgetOption` and `QuotaRequestBudgetSummary` and assert department, `Budget #id`, budget type, remaining quota, and status.

#### Acceptance Criteria Verification

- AC1: Implemented. Wallet and enterprise organization quota request budget options now render department name, `Budget #{{budgetId}}`, budget type label, formatted remaining quota, and status label.
- AC2: Implemented. Wallet and enterprise organization selected request scope summaries now include the specific selected budget pool identity and key details, with clear empty states when no budget pool is selected.

#### Validation

- PASS: `cd web/default && bun test src/features/wallet/components/employee-quota-request-card.test.tsx` — 8 passed.
- PASS: `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx` — 47 passed.
- PASS: `cd web/default && bun run typecheck`.
- PASS: `cd web/default && bun run i18n:sync`.
- PASS: `cd web/default && bun run test:e2e` — full bundled frontend Node test suite passed. Rsbuild emitted a non-blocking optional dependency warning for `supports-color` from `debug`.

### File List

- `_bmad-output/implementation-artifacts/7b-2-redesign-budget-pool-selection-and-selected-summary.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `web/default/src/features/enterprise-organization/quota-request-budget-display.ts`
- `web/default/src/features/enterprise-organization/quota-request-budget-display-components.tsx`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/features/wallet/components/employee-quota-request-card.tsx`
- `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/vi.json`
