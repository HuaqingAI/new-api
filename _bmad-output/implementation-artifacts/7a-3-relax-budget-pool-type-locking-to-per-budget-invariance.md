---
baseline_commit: de419ee9feab6dbb35f0f9a45930c1484dadada8
---

# Story 7A.3: 放宽预算池类型约束至单池不可变

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员,
I want 在同一部门下创建多个不同类型的预算池,
so that 我可以同时维护一次性专项池和周期性运营池，而不互相锁死类型。

## Acceptance Criteria

1. **Given** 某部门已存在 `balance` 型预算池  
   **When** 管理员创建新的 `subscription` 型预算池  
   **Then** 系统允许创建  
   **And** 不因为部门已有池类型不同而拒绝。
2. **Given** 某个预算池已创建完成  
   **When** 管理员试图把该预算池从 `balance` 改成 `subscription` 或反向修改  
   **Then** 系统拒绝
   **And** 保持“单预算池实例类型不可变”。
3. **Given** 同一部门存在多个不同类型预算池  
   **When** 进行预算委派、员工申请或成员分配  
   **Then** 系统按当前显式选中的预算池进行类型校验
   **And** 不再把部门整体视为单一预算类型。
4. **Given** 钉钉通知发送失败
   **When** 核心治理事务已经提交
   **Then** 事务结果保持成功
   **And** 系统记录投递失败状态与错误原因以供重试。

## Tasks / Subtasks

- [x] 修正预算池创建约束粒度 (AC: 1, 2)
  - [x] 修改 `service/enterprise/department_budget.go` 中 `DepartmentBudgetService.Create` 的部门级类型锁定逻辑：移除 `latestBudget` 对新建预算池类型的拦截，让同一 `department_id` 下可以继续创建 `balance` 与 `subscription` 两类预算池。
  - [x] 保留创建参数校验：`balance` 仍要求 `total_quota > 0`；`subscription` 仍要求 `cycle_quota > 0`、合法 `cycle_type`、必要 `cycle_started_at`、`custom` 周期必须有 `custom_seconds > 0`。
  - [x] 不新增“编辑预算池类型”的 API 或 UI；如未来已有 update 路径，必须在该路径拒绝改变单个预算池实例的 `type`。本故事只修正 create 的错误拦截，不扩大预算池 CRUD 范围。
  - [x] 将 `ErrDepartmentBudgetTypeImmutable` 的语义收窄为“单个预算池实例类型不可变”，不得再用于“同部门新建不同类型预算池”场景。

- [x] 审查多类型并存下的预算委派、员工申请与成员分配链路 (AC: 3, 4)
  - [x] 核查 `service/enterprise/budget_delegation.go`：`validateBudgetDelegationType(sourceBudget, targetBudget)` 应继续只比较本次显式选择的 source/target budget 类型；不要改成按部门查 `latestBudget` 或部门级类型。
  - [x] 核查 `service/enterprise/quota_request.go`：`GetCapability` 应返回当前部门所有 active budget pools，员工申请表必须显示并提交显式 `department_budget_id`；`ensureRequesterContext` 应按提交的 budget id 校验 department/tenant/status，不按部门默认预算池推断。
  - [x] 核查 `service/enterprise/quota_allocation.go`：成员分配应按 `DepartmentBudgetId + DepartmentId` 锁定当前选中池，并按该池自身 `Type` 创建 wallet / 快照；不得依赖“部门只有一种预算类型”。
  - [x] 保持治理通知异步语义：`QuotaRequestService.Submit/decide`、`QuotaAllocationService.*` 成功后通知失败不得回滚核心事务；通知投递失败由 governance notification delivery 状态记录与重试处理。

- [x] 更新后端测试覆盖，防止回归 (AC: 1, 2, 3, 4)
  - [x] 修改 `service/enterprise/department_budget_test.go`：把当前 `TestCreateDepartmentBudgetKeepsExistingTypeImmutable` 改为验证同部门先创建 `balance` 再创建 `subscription` 成功，并断言两条预算池共存。
  - [x] 增加反向用例：同部门先创建 `subscription` 再创建 `balance` 成功。
  - [x] 增加 `ListByDepartment` / `GetByDepartment` 预期说明：list 必须返回多个混合类型池；`GetByDepartment` 如保留 latest 兼容接口，仅作为旧 summary/默认选中兜底，不代表部门类型锁定依据。
  - [x] 修改 `controller/enterprise/department_budget_test.go` 中 `TestDepartmentBudgetAPIRejectsTypeSwitch`：POST create 不应再拒绝不同类型；改名为同部门多类型创建成功并验证 admin action 写入两条 create。
  - [x] 如存在预算池 update controller，再补“改同一预算池 type 被拒绝”的 controller/service 测试；如不存在 update API，在测试或注释中明确本故事不新增该能力。
  - [x] 回归预算委派类型不匹配测试：不同类型 source/target budget 仍应返回 `ErrBudgetDelegationBudgetTypeMismatch`，同部门存在另一种类型池不影响本次显式选择的同类型委派。
  - [x] 回归 quota request capability：同部门混合 active pools 时 `Budgets` 包含所有 active pools，且提交/审批使用 request 中保存的 `department_budget_id`。

- [x] 更新前端测试与用户可见引导 (AC: 1, 3)
  - [x] 保持 `web/default/src/features/enterprise-organization/index.tsx` 的预算池创建表单允许选择 `balance` 或 `subscription`；不要在已有池类型与新建表单类型之间做前端禁用。
  - [x] 在预算池列表与选择控件中继续显示预算池类型、ID、部门名与状态，确保同部门多池并存时管理员、负责人、员工都能区分当前显式选择的池。
  - [x] 更新 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`：渲染同部门混合 `balance` / `subscription` 列表，并验证 `resolveBudgetSelection` / 预算池列表不按类型过滤掉另一类池。
  - [x] 若新增/调整文案，必须写入 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` 的 `translation` namespace；不要把 key 写到 JSON 根级。

- [x] 运行并记录验证 (AC: 1, 2, 3, 4)
  - [x] `go test ./service/enterprise -run 'DepartmentBudget|BudgetDelegation|QuotaRequest|QuotaAllocation'`
  - [x] `go test ./controller/enterprise -run 'DepartmentBudget|BudgetDelegation|QuotaRequest|QuotaAllocation'`
  - [x] `cd web/default && bun run typecheck`
  - [x] `cd web/default && bun run test:e2e`（或至少运行覆盖 `enterprise-organization` 的 bundled tests）
  - [x] 如 `go test ./service/enterprise` 因既有 SQLite 并发/唯一约束日志失败，必须在 Dev Agent Record 中如实记录失败范围，并提供通过的定向测试输出；不得宣称全量通过。

## Dev Notes

### 需求与业务上下文

- Story 7A.3 是 Epic 7 完成后的纠偏项：预算池类型不可变约束应从“部门级不可变”收敛到“单预算池实例不可变”。这是修复既有实现偏差，不是新增第三种预算池类型。[Source: `_bmad-output/planning-artifacts/epics.md` Story 7A.3]
- PRD V1.3A 已明确：同一部门可创建多个预算池，例如一个长期 `balance` + 一个月度 `subscription`；分配时管理员显式选择目标池。[Source: `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md` FR-9]
- 架构 V1.3A 明确：`DepartmentBudget.Type` 继续是单个预算池的单值，因此单个预算池实例仍只能是 `balance` 或 `subscription`；但同一部门下允许不同类型预算池并存。类型兼容校验只作用于当前操作选中的预算池，不应扩展为“部门级单类型”限制。[Source: `_bmad-output/planning-artifacts/architecture.md` Integration Points #8]
- 本故事不触碰 relay、billing expression、logs 或 upstream protocol。预算仍通过部门预算池 → allocation → user subscription wallet 的既有链路承接。[Source: `CLAUDE.md` Rule 7 / Architecture Relay Boundary]

### 当前实现状态（已读 UPDATE 文件）

- `service/enterprise/department_budget.go`
  - 当前 `DepartmentBudgetService.Create` 在创建前调用 `latestBudget(departmentId, tenantId)`，如果最新预算池类型与新建类型不同，就返回 `ErrDepartmentBudgetTypeImmutable`。这正是部门级类型锁定 bug，应删除或绕开该 create-time 检查。
  - `ListByDepartment` 已经能返回 scope 内多个预算池并按 `usage_ratio/remaining/type/status` 排序；它不要求部门单类型。
  - `GetByDepartment` 返回 latest budget，主要是旧 summary / 默认选中兼容入口；修复 create 后不要把它当成类型约束来源。
  - `GetDetail` 按 `tenant_id + department_id + id` 定位预算池，天然是单池视角，应保留。

- `service/enterprise/budget_delegation.go`
  - `lockDelegationBudgets` 已按 source/target 的显式 budget id 和 department id 锁定两条预算池。
  - `validateBudgetDelegationType` 当前要求 sourceBudget.Type == targetBudget.Type，并只接受 `balance/subscription`。这属于“当前显式选择池”的兼容性校验，符合 AC3；不要放宽为跨类型委派，除非产品另行变更。
  - 不要新增按部门 `latestBudget` 推断 source/target 类型的逻辑。

- `service/enterprise/quota_request.go`
  - `GetCapability` 已通过 `DepartmentBudgetService.ListByDepartment(..., IncludeDescendants=false)` 获取当前部门预算池，并筛选 `item.DepartmentId == departmentId` + `Status == active` 后返回 `Budgets`。这应在同部门多类型并存时返回两类池。
  - `Submit` 保存 `DepartmentBudgetId` 与 `BudgetMode`；`ensureRequesterContext` 按提交的 `DepartmentBudgetId + TenantId + DepartmentId` 校验预算池存在且 active。该设计符合显式选池，不要改成默认取 latest budget。
  - `decide` 审批通过后调用 `QuotaAllocationService.createTx`，继续复用 allocation/wallet 生命周期。

- `service/enterprise/quota_allocation.go`
  - `createTx` 按 `DepartmentBudgetId + TenantId + DepartmentId` 锁定当前选中预算池，并将 `budget.Type/CycleType/...` 写入 allocation 快照。
  - `reserveBudgetQuota` 根据 budget 自身类型走 `balance` remaining 扣减或 `subscription` allocated_total/cycle_quota 校验；符合单池视角。
  - 不应在 allocation 创建中加入部门级类型检查。

- `service/enterprise/budget_capacity.go`
  - `reserveDepartmentBudgetQuota`、`buildBudgetReservationUpdate`、`buildBudgetCreditUpdate`、`buildBudgetDelegationReverseTargetUpdate` 都基于传入的单个 `DepartmentBudget` 类型分支，符合本故事目标。
  - 这些函数是预算铁律核心，不要为了放宽 create 约束而放宽 `remaining >= committed_quota` 或 `allocated_total + committed_quota <= cycle_quota`。

- `controller/enterprise/department_budget_test.go`
  - 当前 `TestDepartmentBudgetAPIRejectsTypeSwitch` 把“新建不同类型预算池”当作 type switch 拒绝，应随本故事修正为成功用例。
  - `TestDepartmentBudgetAPIWorkflow` 与 list/detail 测试可继续复用，但建议补两条 create action 的审计断言。

- `service/enterprise/department_budget_test.go`
  - 当前 `TestCreateDepartmentBudgetKeepsExistingTypeImmutable` 也把“新建不同类型预算池”当作 immutable 错误，应改为同部门多类型成功。
  - 已有 `TestGetDepartmentBudgetReturnsLatest` 证明服务历史上允许 DB 中存在混合类型预算池；修复 create 后应将该状态变成正常路径，而不是只能通过直接 DB seed 达成。

- `web/default/src/features/enterprise-organization/index.tsx`
  - `createBudgetSchema` 已允许表单在 `balance` / `subscription` 间选择，并分别校验字段。
  - `DepartmentBudgetPanel` 已使用 `budgetListQuery` + `effectiveBudgetId` + `selectedBudgetId` 显式选择预算池；allocation、delegation、quota request 表单都提交具体 budget id。前端结构已经接近 AC3，要防止新增“已有池类型不同则禁用创建”的逻辑。
  - `quotaRequestCapabilityQuery` 使用后端 capability 返回的 `budgets`，员工申请要继续显式选择 `department_budget_id`。

### Previous Story Intelligence

- Story 7A.1 学到：企业治理工作台的 action label / i18n 必须完整，不要让内部 action key 直接露出；新增文案必须进入全部 Default locale，并位于 `translation` namespace。[Source: `_bmad-output/implementation-artifacts/7a-1-complete-enterprise-governance-i18n-and-action-label-mapping.md`]
- Story 7A.2 学到：普通员工侧入口应复用既有 `quota request` workflow，不新增后端 API 或拆分缓存命名空间；提交成功后要失效治理时间线与通知缓存，保持审批 / 通知闭环可见。[Source: `_bmad-output/implementation-artifacts/7a-2-move-employee-quota-request-entry-to-balance-wallet-user-flow.md`]
- Story 7A.2 review 修复过普通用户读取自身部门成员关系权限问题；本故事如改 quota request capability 或员工申请测试，必须保留“普通员工只能读自身可提交范围，不扩大治理权限面”的边界。
- Story 7A.2 验证记录显示 `go test ./controller/enterprise ./service/enterprise` 中 service 包曾因既有 SQLite 并发/唯一约束测试日志失败；本故事应优先运行更精确的 `-run` 定向测试并如实记录结果。

### Git Intelligence Summary

- 最近提交 `de419ee9f feat(story-7A.2)` 修改了钱包侧员工申请入口、`controller/enterprise/department_membership.go`、`router/enterprise-router.go`、企业组织/钱包前端测试与所有 Default locale。说明当前分支已经有 quota request capability 与钱包侧入口联动，7A.3 不应破坏该入口。
- 最近提交 `5d9f57715 feat(story-7A.1)` 修改了企业组织工作台、治理时间线、通知投递与多语言。说明本故事任何新增前端文案都要同步 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` 并回归 i18n key 测试。
- 当前 sprint-status 中 `7a-3-relax-budget-pool-type-locking-to-per-budget-invariance` 仍是 backlog；create-story 完成后应更新为 `ready-for-dev`。

### Architecture Compliance

- 继续遵守 Router → Controller → Service → Model 分层；Controller 只解参、映射错误、写 admin action；业务规则放 service。[Source: `CLAUDE.md`; `_bmad-output/planning-artifacts/architecture.md`]
- 所有 JSON marshal/unmarshal 必须使用 `common.Marshal` / `common.Unmarshal` 等 wrapper；本故事预计不需要新增 JSON 处理，如必须新增快照/审计 payload，必须走 `common/json.go`。[Source: `CLAUDE.md` Rule 1]
- 数据库代码必须兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+；优先 GORM，不新增数据库专属 SQL。[Source: `CLAUDE.md` Rule 2]
- DTO 可选标量继续使用指针 + `omitempty`，避免显式 `0` / `false` 被吞。[Source: `CLAUDE.md` Rule 6]
- 严禁触碰 `relay/**`、`pkg/billingexpr/**`、`pkg/cachex/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`、所有 `relay/channel/**`，除非先升级架构评审。本故事没有触碰理由。[Source: `_bmad-output/planning-artifacts/architecture.md` Enforcement Guidelines]
- `new-api` 与 `QuantumNous` 是受保护标识，不得删除、替换或改名。[Source: `CLAUDE.md` Rule 5]

### Library / Framework Requirements

- 后端：Go + Gin + GORM v2；不需要新增 Go 依赖。
- 前端 Default：React 19 + TypeScript + TanStack Query + i18next；使用 Bun 运行脚本。
- 本故事不需要 web research 或依赖升级；不要借机升级 React、GORM、Gin、Rsbuild 或新增表单/状态库。

### File Structure Requirements

Expected UPDATE files:

- `service/enterprise/department_budget.go` — 修正 create-time type guard。
- `service/enterprise/department_budget_test.go` — 修改部门级 type immutable 旧测试，新增多类型并存用例。
- `controller/enterprise/department_budget_test.go` — 修改 API 旧拒绝测试，验证 POST create 多类型成功。
- `service/enterprise/budget_delegation_test.go` — 如现有用例不足，补 explicit selected budget type compatibility 回归。
- `service/enterprise/quota_request_test.go` — 如现有用例不足，补 capability 返回混合 active pools。
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx` — 补预算池列表 / 选择器混合类型渲染。
- `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` — 仅当新增文案时更新。

Likely no changes needed:

- `service/enterprise/quota_allocation.go`、`service/enterprise/budget_capacity.go`、`service/enterprise/budget_delegation.go`、`service/enterprise/quota_request.go` 的核心逻辑已按显式 budget id 工作；除非测试暴露问题，不要重写。
- `dto/enterprise/*` 预计不需要改。
- `model/enterprise/department_budget.go` 预计不需要 schema 改动。

### Testing Requirements

Minimum targeted validation:

```bash
go test ./service/enterprise -run 'DepartmentBudget|BudgetDelegation|QuotaRequest|QuotaAllocation'
go test ./controller/enterprise -run 'DepartmentBudget|BudgetDelegation|QuotaRequest|QuotaAllocation'
cd web/default && bun run typecheck
cd web/default && bun run test:e2e
```

Recommended assertions:

- 同部门 `balance -> subscription` 创建成功。
- 同部门 `subscription -> balance` 创建成功。
- `ListByDepartment` 返回同部门混合类型预算池，并保留 type/status/usage_ratio 展示。
- `GetByDepartment` 的 latest 兼容行为不再作为部门类型锁定依据。
- 预算委派仍拒绝本次显式选择的 source/target budget 类型不一致。
- quota request capability 返回当前部门所有 active budget pools，混合类型不被过滤。
- quota request submit / approve 后 allocation 使用 request 保存的 `department_budget_id`。
- 通知队列/投递失败不回滚 quota request / allocation 核心事务（如已有测试覆盖则确认仍通过）。

### Anti-Patterns to Avoid

- 不要把 `ErrDepartmentBudgetTypeImmutable` 简单删掉并在未来 update 场景失去“单池不可变”语义；应收窄其使用边界。
- 不要在前端根据“当前 latest budget type”禁用另一种类型创建；这会把后端修复重新变成 UI 层 bug。
- 不要让 allocation、quota request、delegation 自动选择“部门最新预算池”；所有链路都必须显式选择 budget id。
- 不要放宽预算铁律或委派类型兼容性：本故事只允许“同部门多池不同类型并存”，不允许“单次操作跨类型混用”。
- 不要新增预算池编辑/删除能力来解决 AC2；AC2 的正确落点是确认不存在可改型入口，或已有入口拒绝改型。
- 不要新增 relay / billing expression 字段来追踪预算池类型。

### References

- [Source: `_bmad-output/planning-artifacts/epics.md`#Story 7A.3]
- [Source: `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`#FR-9]
- [Source: `_bmad-output/planning-artifacts/architecture.md`#Integration Points / 预算池类型约束边界（V1.3A）]
- [Source: `service/enterprise/department_budget.go`#DepartmentBudgetService.Create]
- [Source: `service/enterprise/budget_delegation.go`#validateBudgetDelegationType]
- [Source: `service/enterprise/quota_request.go`#GetCapability / ensureRequesterContext]
- [Source: `service/enterprise/quota_allocation.go`#createTx]
- [Source: `service/enterprise/budget_capacity.go`#reserveDepartmentBudgetQuota]
- [Source: `_bmad-output/implementation-artifacts/7a-2-move-employee-quota-request-entry-to-balance-wallet-user-flow.md`#Senior Developer Review]

## Dev Agent Record

### Agent Model Used

gpt-5.5

### Debug Log References

- 2026-06-02: BMAD create-story workflow executed for Story 7A.3 with YOLO mode. Existing backlog story was upgraded to ready-for-dev after re-reading skill, discovery protocol, template, checklist, sprint status, epics, PRD, architecture, previous story, recent git history, and relevant source/test files.
- 2026-06-02: BMAD dev-story workflow executed. Removed create-time `latestBudget` type comparison from `DepartmentBudgetService.Create`; no budget update API/UI was added.
- 2026-06-02: Targeted backend validations passed: `go test ./service/enterprise -run 'DepartmentBudget|BudgetDelegation|QuotaRequest|QuotaAllocation'`; `go test ./controller/enterprise -run 'DepartmentBudget|BudgetDelegation|QuotaRequest|QuotaAllocation'`.
- 2026-06-02: Frontend validations passed from `web/default`: `bun run typecheck`; `bun run test:e2e`. `test:e2e` emitted an existing Rsbuild optional dependency warning for `supports-color` from `debug/src/node.js`, but all bundled tests passed.
- 2026-06-02: Full Go regression `go test ./...` was attempted and failed in existing unrelated `model` package setup with missing SQLite tables such as `abilities`, `tasks`, `tokens`, `channels`, and `user_subscriptions`; targeted service/controller enterprise tests passed.

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- Story scope is constrained to correcting budget pool type-lock granularity; no relay, billing expression, schema, or new budget type work is required.
- Current primary bug is identified in `DepartmentBudgetService.Create` create-time `latestBudget` type comparison.
- Prior 7A.2 learnings are included: preserve quota request workflow, employee self-scope, governance cache invalidation, and locale `translation` namespace rules.
- Removed the department-level create-time budget type lock. Same department can now create `balance` and `subscription` pools in either order while preserving per-type create parameter validation.
- Confirmed `ErrDepartmentBudgetTypeImmutable` is no longer used to reject new mixed-type pools in the same department; no update API/UI was introduced, so single-pool type mutation remains outside this story scope.
- Added backend regressions for mixed budget create order, API create audit actions, explicit selected-budget delegation mismatch, and quota request capability returning mixed active pools.
- Added frontend regression that renders same-department mixed `balance` / `subscription` budget pools and verifies explicit `resolveBudgetSelection` keeps the selected budget by ID without type filtering.
- Reviewed quota delegation/request/allocation paths: they continue to lock and validate by explicit budget IDs (`source/target budget id`, `department_budget_id + department_id`) rather than department-level budget type.
- No new user-visible copy was added, so locale JSON files were not changed.

### File List

- `_bmad-output/implementation-artifacts/7a-3-relax-budget-pool-type-locking-to-per-budget-invariance.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/tests/test-summary.md`
- `service/enterprise/department_budget.go`
- `service/enterprise/department_budget_test.go`
- `service/enterprise/budget_delegation_test.go`
- `service/enterprise/quota_request_test.go`
- `controller/enterprise/department_budget_test.go`
- `tests/api/enterprise_department_budget_test.go`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`

## Senior Developer Review (AI)

_Reviewer: hth on 2026-06-02_

### Review Scope

- Story file: `_bmad-output/implementation-artifacts/7a-3-relax-budget-pool-type-locking-to-per-budget-invariance.md`
- Git changes reviewed: story document, sprint status, test summary, `service/enterprise/department_budget.go`, changed backend tests, changed API test, and `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`.
- Source review deliberately excluded `_bmad/`, `_bmad-output/` implementation planning text from code-quality findings except for story/sprint/test-summary sync checks.

### Findings

- **HIGH:** 0 confirmed.
- **MEDIUM:** 0 confirmed.
- **LOW:** 0 confirmed.

No auto-fix patch was required beyond review-workflow bookkeeping (story status, sprint status, File List, Change Log, and this review note).

### AC Validation

1. **AC1 同部门 `balance` 后创建 `subscription` 成功：Implemented.** `DepartmentBudgetService.Create` no longer calls `latestBudget` to block a different new pool type; create-time validation now branches only by the requested new pool type in `service/enterprise/department_budget.go:135`. Service, controller, and API regressions cover mixed create success.
2. **AC2 单预算池实例类型不可变：Implemented within current product surface.** Review found no budget update API/UI route capable of changing an existing budget pool type. The story correctly did not add update CRUD; `ErrDepartmentBudgetTypeImmutable` is no longer misused for same-department mixed creation.
3. **AC3 显式选中预算池校验：Implemented.** Delegation still compares selected source/target budget instances only; quota request `GetCapability` returns all active same-department pools and `ensureRequesterContext` validates the submitted `department_budget_id + department_id`; allocation `createTx` locks and snapshots the submitted budget id. Frontend selection remains by budget id and the mixed-pool list renders type + department + id.
4. **AC4 通知失败不回滚核心事务：Implemented / unchanged.** Quota request submit/decision enqueue notifications only after transaction success; allocation governance notification enqueue likewise happens after successful core mutation. Existing frontend governance notification tests still render delivery failure reason and resend action.

### Task Audit

- All checked tasks have implementation evidence in the changed service/controller/API/frontend tests or in existing unchanged explicit-budget service paths.
- The story File List was missing changed `_bmad-output/implementation-artifacts/tests/test-summary.md` and `tests/api/enterprise_department_budget_test.go`; this review fixed the File List documentation gap.

### Validation Checklist

- [x] Story file loaded from `_bmad-output/implementation-artifacts/7a-3-relax-budget-pool-type-locking-to-per-budget-invariance.md`.
- [x] Story Status verified as reviewable (`review`) before review and updated to `done` after no critical issues remained.
- [x] Epic and Story IDs resolved as 7A.3 / `7a-3-relax-budget-pool-type-locking-to-per-budget-invariance`.
- [x] Story Context / Dev Notes reviewed from the story file; no separate story-context file was found.
- [x] Epic/architecture docs located: `_bmad-output/planning-artifacts/architecture.md` found; PRD/epic references were already embedded in story Dev Notes.
- [x] Architecture/standards loaded: `CLAUDE.md` and BMAD config loaded.
- [x] Tech stack documented: Go/Gin/GORM backend, React/TypeScript/Bun frontend.
- [x] MCP/web doc search not needed because no dependency or external API behavior changed.
- [x] Acceptance Criteria cross-checked against implementation.
- [x] File List reviewed and corrected for completeness.
- [x] Tests identified and mapped to ACs; generated API test and existing e2e coverage verified.
- [x] Code quality review performed on changed source/test files.
- [x] Security review performed on changed source/test files; no auth/tenant expansion or injection issue found.
- [x] Outcome: Approve.
- [x] Review notes appended under `Senior Developer Review (AI)`.
- [x] Change Log updated with review entry.
- [x] Status updated to `done`.
- [x] Sprint status synced to `done`.
- [x] Story saved successfully.

### Validation Run

- `go test ./service/enterprise -run 'DepartmentBudget|BudgetDelegation|QuotaRequest|QuotaAllocation'` — passed.
- `go test ./controller/enterprise -run 'DepartmentBudget|BudgetDelegation|QuotaRequest|QuotaAllocation'` — passed.
- `go test ./tests/api -run 'TestEnterpriseDepartmentBudgetAPIAllowsMixedTypeCreatesForDepartment' -count=1` — passed.
- `cd web/default && bun run typecheck` — passed.
- `cd web/default && bun run test:e2e` — passed; existing optional `supports-color` warning from `debug/src/node.js` remained non-blocking.

### Outcome

Approved. Story 7A.3 is marked `done`; sprint tracking is synced.

## Change Log

- 2026-06-02 15:20:00 +0800: 创建 Story 7A.3 上下文文档，纳入 backlog。
- 2026-06-02: BMAD create-story 重新生成并增强 Story 7A.3，上下文补齐到 ready-for-dev，明确实现文件、测试范围、前序故事教训与反模式。
- 2026-06-02: BMAD dev-story 实现完成：放宽同部门新建混合类型预算池，补后端/前端回归测试并更新状态为 review。
- 2026-06-02: BMAD story-automator-review 完成：对照 AC、已完成任务、File List 与 git 实际变更进行对抗审查，未发现剩余阻断或需自动修复的 HIGH/MEDIUM 问题；补齐 review 记录并更新状态为 done。

## Create-Story Checklist Validation

- [x] Story ID、Story Key、用户故事与 AC 从 Epic 7A.3 提取并保持一致。
- [x] 已纳入 PRD V1.3A 对“同部门多预算池、单池类型不可变”的产品约束。
- [x] 已纳入 Architecture V1.3A 对预算池类型约束边界、显式选池和 relay/billing 禁触碰边界的要求。
- [x] 已读取并总结会被修改的 UPDATE 文件当前状态、变化点和必须保留的行为。
- [x] 已纳入 Story 7A.1 / 7A.2 的可执行教训，特别是 i18n、普通用户权限、quota request workflow 和治理通知闭环。
- [x] 已给出防止 LLM dev agent 犯错的反模式清单。
- [x] 已给出后端、前端与定向测试要求。
- [x] 状态已设置为 `ready-for-dev`。
