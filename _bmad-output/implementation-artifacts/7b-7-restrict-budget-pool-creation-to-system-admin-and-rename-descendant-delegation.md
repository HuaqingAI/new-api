---
created_at: 2026-06-03 23:25:50 +0800
baseline_commit: 990423635e8ac7aa8332765c5d6807667ed0306e
---

# Story 7B.7: 收紧预算池创建权限并重命名下级部门预算分配语义

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an 企业管理员或部门管理员,
I want 由企业管理员统一创建预算池，而部门管理员只基于当前部门已有预算池继续向下级部门分配预算,
so that 预算池供给权与预算使用/分配权分离，治理边界更清晰，UI 文案也更符合业务心智。

## Acceptance Criteria

1. **Given** 普通部门管理员进入本部门预算工作区  
   **When** 页面渲染预算池治理入口  
   **Then** 不显示 "Create Budget Pool" 入口  
   **And** 其只能查看既有预算池并执行被授权的下级预算分配或成员分配动作。

2. **Given** 部门管理员直接调用 `POST /departments/:id/budget`  
   **When** 后端执行权限校验  
   **Then** 请求被拒绝并返回项目既有错误响应语义（HTTP 200 + `success=false`）  
   **And** 错误消息与 `EnterpriseAdmin` 权限要求一致。

3. **Given** 企业管理员进入任意部门预算工作区  
   **When** 需要为该部门建立预算池  
   **Then** 系统允许创建预算池  
   **And** 创建接口权限明确从 `EnterpriseDepartmentAdmin("id")` 收敛到 `EnterpriseAdmin()`。

4. **Given** 负责人或部门管理员把当前部门预算继续分给下级部门  
   **When** 页面展示该治理动作  
   **Then** UI 与文案不再强调 "descendant / 后代部门"  
   **And** 改为"向下级部门分配预算"或等价更自然的话术。

## Tasks / Subtasks

- [x] 收紧预算池创建后端路由权限 (AC: 2, 3)
  - [x] 在 `router/enterprise-router.go` 将 `POST /api/enterprise/departments/:id/budget` 的中间件从 `middleware.EnterpriseDepartmentAdmin("id")` 改为 `middleware.EnterpriseAdmin()`。
  - [x] 保留 `GET /departments/:id/budget`、`GET /departments/:id/budgets`、`GET /departments/:id/budgets/:budget_id` 的 `EnterpriseDepartmentAdmin("id")` 权限；部门管理员仍必须能查看既有预算池。
  - [x] 不改变 `CreateDepartmentBudget` controller/service 的创建语义、DTO 字段、同部门混合类型能力、审计写入和失败审计；本故事只收紧创建入口权限。
  - [x] 确认非企业管理员请求创建预算池时返回 `common.ApiErrorI18n(c, i18n.MsgEnterprisePermissionAdminRequired)` 既有语义，即 HTTP 200、`success=false`、message 为 `error.enterprise.permission.admin_required`。

- [x] 调整后端和 API 级回归测试 (AC: 2, 3)
  - [x] 更新 `controller/enterprise/department_budget_test.go` 中权限路由测试：原 `TestDepartmentBudgetAPIDeniesNonDepartmentAdmin` 需要改为覆盖 `EnterpriseAdmin()`，断言普通部门管理员/普通用户创建预算池均得到 `error.enterprise.permission.admin_required`。
  - [x] 更新 `tests/api/enterprise_department_budget_test.go`：现有创建预算池 API workflow、invalid subscription failure audit、tenant scoped flow、mixed type creates 等用例必须使用 `common.RoleAdminUser` 登录创建预算池。
  - [x] 新增 API 断言：拥有 `DepartmentRole` 的普通用户仍可 `GET /departments/:id/budget(s)`，但 `POST /departments/:id/budget` 被拒绝为 `error.enterprise.permission.admin_required`。
  - [x] 保留/补充企业管理员创建成功后 `enterprise_admin_actions` 仍写入 `enterprise.organization.department_budget.create`，失败审计仍只覆盖创建校验失败，不因权限拒绝产生伪创建事实。

- [x] 在 Default 前端隐藏部门管理员的预算池创建入口 (AC: 1, 3)
  - [x] 在 `web/default/src/features/enterprise-organization/index.tsx` 复用 Story 7B.6 已建立的 `canManageBudgetLifecycle(role)` 或提取更明确的 `canManageBudgetControlPlane(role)`，让预算池创建表单只对 `ROLE.ADMIN` / root 等价角色渲染。
  - [x] 部门管理员视图必须继续显示预算池概览、预算池列表、成员 allocation 表单、下级部门预算分配表单、quota request 治理信息；不要把整个 `DepartmentBudgetPanel` 或 budget list 隐藏。
  - [x] create mutation、表单 state 和 `createDepartmentBudget` API 可以保留，但不可由非管理员 UI 触发；避免仅禁用按钮后仍展示"Create Budget Pool"文案。
  - [x] 确认 7B.6 lifecycle controls 仍只对管理员展示，部门管理员不会看到 pause/resume/resize/create 任一预算池控制面动作。

- [x] 将预算委派 UI 文案从 descendant 话术改为下级部门预算分配 (AC: 4)
  - [x] 在 `web/default/src/features/enterprise-organization/index.tsx` 替换 delegation 区域可见文案：`Budget Delegation To Descendant Department`、`Delegate from the current department budget pool to a descendant department budget pool...`、`Target Descendant Budget Pool`、`Choose a descendant budget pool`、`Delegate to descendant department`、空状态说明等。
  - [x] 推荐英文 source keys 使用业务自然话术，例如 `Allocate Budget To Subordinate Department`、`Allocate from the current department budget pool to a subordinate department budget pool, while keeping adjustment history visible.`、`Target Subordinate Budget Pool`、`Choose a subordinate budget pool`、`Allocate budget to subordinate department`。
  - [x] 中文翻译使用"向下级部门分配预算"、"目标下级预算池"等；避免"后代部门"、"委派到后代部门"出现在该治理动作的可见 UI 中。
  - [x] 保留数据模型和后端错误语义中的祖先/后代校验事实：目标仍必须是 source department 的 descendant，本故事只改用户心智文案，不改 `BudgetDelegationService.validateCreateDelegation` 的树关系规则。
  - [x] 不修改 include descendants 范围开关、usage/risk detail 关于 descendant tree 的分析型文案，除非它们直接描述预算分配动作；这些属于查询范围语义，不是本故事的 UI 命名目标。

- [x] 补齐前端 i18n 与 action label 覆盖 (AC: 1, 4)
  - [x] 更新 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` 中新增/替换的 delegation 文案，使用 flat JSON 英文 source key。
  - [x] 如果删除旧 source key 引用，运行 `bun run i18n:sync` 清理或同步 locale；不得让非英文语言回退显示英文源字符串。
  - [x] 保留治理 timeline action label 中 `Budget delegation created/adjusted/revoked/rejected` 的业务含义；如改 visible label，不要破坏 `enterprise.organization.budget_delegation.*` 到 label 的映射。
  - [x] 不删除、改名或替换受保护 `new-api`、`QuantumNous`、版权、品牌和 i18n sync allowlist 条目。

- [x] 增加前端测试，防止权限和文案回退 (AC: 1, 4)
  - [x] 扩展 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`：增加 helper/组件测试，断言部门管理员角色不会渲染 `Create Budget Pool`，管理员角色会渲染创建入口。
  - [x] 更新 delegation schema/test 名称与断言，从 "descendant target" 改为 "subordinate target" 或中文等价；schema 字段名可保持 `target_department_id` / `target_budget_id`。
  - [x] 更新空状态和 delegation rows 测试，断言新文案出现，并断言 budget delegation UI 不再包含 `Descendant` / `descendant department` / `后代部门` 这类旧动作话术。
  - [x] 保留 Story 7B.5 的 `QuotaAmountInput` / `QuotaAmountDisplay` 断言；下级部门预算分配仍以 quota 为底层单位并支持金额视角。

- [x] 运行验证并记录结果 (AC: 1, 2, 3, 4)
  - [x] 运行 `GOCACHE=/private/tmp/new-api-go-build go test ./controller/enterprise ./tests/api` 或至少覆盖 `controller/enterprise` 与 `tests/api` 中 enterprise budget 相关用例。
  - [x] 运行 `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx`。
  - [x] 运行 `cd web/default && bun run typecheck`。
  - [x] 运行 `cd web/default && bun run i18n:sync`。

## Dev Notes

### 需求与业务上下文

- Story 7B.7 来自 2026-06-03 correct-course：预算池创建权应从部门管理员收回到企业管理员；部门管理员只在既有预算池基础上做成员分配和向下级部门分配预算。[Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03-budget-pool-lifecycle-cc.md`#Issue Summary; #Change C]
- 当前系统没有独立 `EnterpriseSystemAdmin()`；本故事中的"系统管理员"必须映射为当前可实现的 `middleware.EnterpriseAdmin()` / `common.RoleAdminUser`，不要新增 role 或中间件。[Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03-budget-pool-lifecycle-cc.md`#Evidence; #Change C]
- 预算池供给权和使用/分配权要分离：`EnterpriseAdmin` 负责 create/pause/resume/resize，`EnterpriseDepartmentAdmin` 负责查看预算池、成员 allocation、下级部门 budget allocation/delegation。[Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03-budget-pool-lifecycle-cc.md`#Architecture Impact]
- PRD FR-9 仍要求同一部门可创建多个 balance/subscription 预算池，且单个预算池 type 不可原地切换；本故事不改变这些服务层不变量。[Source: `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`#FR-9]
- PRD FR-23 的技术事实仍是祖先部门可向任意后代部门预算池分配预算；本故事只把用户界面话术改成"向下级部门分配预算"，不改变祖先-后代树链校验。[Source: `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`#FR-23]

### 当前实现状态（已读 UPDATE 文件）

- `router/enterprise-router.go`
  - 当前 `POST /departments/:id/budget` 仍使用 `middleware.EnterpriseDepartmentAdmin("id")`，这是本故事的核心后端缺口。
  - `POST /departments/:id/budgets/:budget_id/pause|resume|resize` 已经使用 `middleware.EnterpriseAdmin()`，符合 7B.6 复审后的预算池控制面权限。
  - budget GET/list/detail、quota allocations、budget delegations 仍走 `EnterpriseDepartmentAdmin`，这些不应被收紧成企业管理员专用。

- `middleware/enterprise_admin.go`
  - `EnterpriseAdmin()` 仅检查 `c.GetInt("role") < common.RoleAdminUser`，失败时返回 `i18n.MsgEnterprisePermissionAdminRequired` 并 `Abort()`。
  - 错误响应遵循项目既有 API 语义：HTTP 200 + `success=false`，message 为 `error.enterprise.permission.admin_required`。

- `middleware/enterprise_dept_admin.go`
  - `EnterpriseDepartmentAdmin` 对 admin/root 直接放行；普通用户必须通过 `PermissionService.CanGovernDepartment`。
  - 它返回的是 `error.enterprise.permission.dept_admin_required`；创建预算池改为 `EnterpriseAdmin()` 后，测试必须同步断言 admin required，而不是 dept admin required。

- `controller/enterprise/department_budget.go`
  - `CreateDepartmentBudget` 已负责 DTO bind、tenant 归一、service create、成功/失败 admin action audit 和 response。
  - 本故事不需要改 controller 创建逻辑；权限应在 router middleware 层拦截。
  - 错误 mapping 已覆盖 budget invalid type/quota/cycle/type immutable/status/resize 等；本故事不新增后端业务错误。

- `service/enterprise/department_budget.go`
  - `Create` 已允许同一部门存在多个不同类型预算池，并保持单个 budget type 不可变。
  - 不要把权限逻辑下沉到 service；项目现有资源级权限在 router/controller middleware 层完成。

- `tests/api/enterprise_department_budget_test.go`
  - 多个 API workflow 当前用 `common.RoleCommonUser` + `DepartmentRole` 创建预算池，这些测试会在路由收紧后失败，需要改为 `common.RoleAdminUser` 创建。
  - 应新增/改造一个明确的部门管理员创建拒绝测试，避免未来误把 create route 放回 `EnterpriseDepartmentAdmin`。

- `web/default/src/features/enterprise-organization/index.tsx`
  - `DepartmentBudgetPanel` 中创建预算池表单当前无角色 gating；`Create Budget Pool` 按钮对部门管理员可见。
  - 7B.6 已增加 `canManageBudgetLifecycle` 并用于 pause/resume/resize controls；本故事可复用或重命名该权限 helper。
  - budget delegation 区域当前可见文案仍包含 `Budget Delegation To Descendant Department`、`Target Descendant Budget Pool`、`Choose a descendant budget pool`、`Delegate to descendant department` 和 descendant 空状态说明。
  - `descendantBudgetOptions` 变量名和 API 参数可保留；这是内部技术语义，不是用户可见文案。若改名，范围会扩大且更容易引入回归。

- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - 已有 `canManageBudgetLifecycle` 测试，断言 undefined/普通用户 false，admin/root true。
  - delegation schema 和空状态测试名称/断言仍使用 descendant guidance；需要同步为新业务话术。

- `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`
  - 现存 delegation 相关 key 覆盖所有语言，但多处翻译直接使用 descendant / 后代部门；本故事必须更新全部语言。
  - `Include descendants`、`Current scope: {{department}} and all descendant departments`、risk event descendant tree 等查询范围文案不属于预算分配动作，可保持。

### Previous Story Intelligence

- Story 7B.6 已完成 pause/resume/resize，并在 senior review 中修复了 lifecycle controls 对部门管理员可见的问题；本故事必须沿用相同权限口径，避免 create 表单成为唯一残留的部门管理员预算池控制面入口。[Source: `_bmad-output/implementation-artifacts/7b-6-close-budget-pool-lifecycle-gap-with-disable-and-replace.md`#Senior Developer Review]
- Story 7B.6 的路由已把 lifecycle actions 放到 `EnterpriseAdmin()`；本故事只需把 create route 对齐到同一控制面，不要重写 lifecycle service 或 UI。[Source: `router/enterprise-router.go`#RegisterEnterpriseRouter]
- Story 7B.5 已建立 `QuotaAmountInput` / `QuotaAmountDisplay` 金额视角；下级部门预算分配表单继续复用这套输入，不要回退成裸 quota 数字输入。[Source: `_bmad-output/implementation-artifacts/7b-5-add-quota-amount-toggle-and-dollar-display-for-budget-governance.md`#Completion Notes List]
- Story 7B.3 已补齐企业治理 i18n；本故事新增/替换 UI 文案必须同步 en/zh/fr/ru/ja/vi，不能只改中文或英文。[Source: `_bmad-output/implementation-artifacts/7b-3-complete-quota-request-governance-wallet-i18n-coverage.md`#Completion Notes List]
- Story 7B.2 建立预算池可辨识选项和 selected summary；下级部门预算池选择继续使用现有 budget option 展示，不要复制第二套 budget display helper。[Source: `_bmad-output/implementation-artifacts/7b-2-redesign-budget-pool-selection-and-selected-summary.md`#Completion Notes List]

### Git Intelligence Summary

- 最近提交 `ed68c17c1 feat(story-7B.6): 企业管理员补齐预算池暂停、恢复与扩缩容治理闭环` 修改了 router、controller/service、Default 前端、i18n 和 tests；7B.7 应在这些变更之上做小范围权限和文案收口。
- 最近提交 `990423635 test(story-7B.6): strengthen enterprise lifecycle guardrails` 增加 API guardrail tests，并修复 lifecycle controls 角色 gating；本故事应扩展同一测试文件，不要删除这些 guardrails。
- 当前 sprint status 中 `7b-7-restrict-budget-pool-creation-to-system-admin-and-rename-descendant-delegation` 是 backlog；create-story 完成后应标记为 ready-for-dev。

### Architecture Compliance

- 后端继续遵守 Router -> Controller -> Service -> Model 分层；权限收紧放在 `router/enterprise-router.go` middleware，不在 service 内读取用户角色。
- JSON marshal/unmarshal 继续使用 `common.*` wrapper；本故事预计不需要新增 JSON 序列化。
- 数据库必须兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+。本故事不应新增 migration 或 raw SQL。
- 企业管理路由继续使用 `/api/enterprise/**`，预算池创建 endpoint 仍保持 `/departments/:id/budget`，避免破坏前端 API client 与文档。
- mutation 后 TanStack Query invalidation 必须精确到 enterprise feature scope；不要新增 `invalidateQueries({ queryKey: ['enterprise'] })` 粗粒度全量失效。[Source: `_bmad-output/planning-artifacts/architecture.md`#Process Patterns]
- 配额分配事实只写 `enterprise_quota_allocations`，预算池创建仍属于低频管理动作并写 `enterprise_admin_actions`；本故事不改变审计源边界。
- 受保护标识 `new-api`、`QuantumNous` 不得删除、替换、改名或从版权/元数据/i18n sync literal allowlist 中移除。

### Library / Framework Requirements

- 后端不新增依赖；继续使用 Gin middleware、GORM、项目 i18n sentinel。
- 前端不新增依赖；继续使用 React 19、TypeScript、TanStack Query、react-hook-form/zod、Base UI/Tailwind 和 lucide icons。
- 如果新增可见按钮或提示，使用现有 button/form/card primitives 和 lucide icons；不要引入新的 UI 组件库。
- 文案替换必须走 `useTranslation()` + `t('English source key')`；不要硬编码中文字符串到组件。

### File Structure Requirements

Expected UPDATE files:

- `router/enterprise-router.go` - create budget route middleware from `EnterpriseDepartmentAdmin("id")` to `EnterpriseAdmin()`.
- `controller/enterprise/department_budget_test.go` - route/middleware permission unit test expectation changes.
- `tests/api/enterprise_department_budget_test.go` - API workflow login roles and department-admin create rejection coverage.
- `web/default/src/features/enterprise-organization/index.tsx` - create form role gating and budget delegation visible copy.
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx` - role gating and delegation copy regression tests.
- `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` - updated delegation copy translations.

Likely no changes needed:

- `service/enterprise/department_budget.go` - creation invariants already correct; no role logic here.
- `controller/enterprise/department_budget.go` - create handler semantics and audit should stay intact.
- `dto/enterprise/department_budget.go` - no DTO changes.
- `service/enterprise/budget_delegation.go` - keep descendant validation rule; user-facing copy changes only.
- `model/**`, migrations, `relay/**`, `pkg/billingexpr/**`, provider adapters - out of scope.
- `web/classic/**` - 7B follow-up is scoped to Default frontend unless explicitly requested.

### Anti-Patterns to Avoid

- 不要新增 `EnterpriseSystemAdmin`、新 role、role migration 或更细粒度权限模型；当前 story 明确映射到 `EnterpriseAdmin()`。
- 不要把预算池 GET/list/detail 改成 admin-only；部门管理员必须能查看既有预算池才能做成员分配和下级分配。
- 不要用前端隐藏替代后端权限；路由 middleware 是 AC2/AC3 权威。
- 不要为了改文案而改变 `BudgetDelegationService` 的祖先-后代校验、DTO 字段或数据库模型。
- 不要把 `include_descendants` 查询范围文案全部替换；本故事针对预算分配动作的 descendant 命名。
- 不要删除现有 budget lifecycle controls 或 7B.6 测试。
- 不要让部门管理员看到 disabled 的 create budget card；AC 要求不显示入口。
- 不要只更新 `en.json`/`zh.json`；Default 前端支持 en/zh/fr/ru/ja/vi。

### References

- [Source: `_bmad-output/planning-artifacts/epics.md`#Story 7B.7]
- [Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03-budget-pool-lifecycle-cc.md`#Change C]
- [Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03-budget-pool-lifecycle-cc.md`#Architecture Impact]
- [Source: `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`#FR-9]
- [Source: `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`#FR-23]
- [Source: `_bmad-output/planning-artifacts/architecture.md`#Data Flow items 3-4, 6, 11]
- [Source: `router/enterprise-router.go`#RegisterEnterpriseRouter]
- [Source: `middleware/enterprise_admin.go`#EnterpriseAdmin]
- [Source: `middleware/enterprise_dept_admin.go`#EnterpriseDepartmentAdmin]
- [Source: `web/default/src/features/enterprise-organization/index.tsx`#DepartmentBudgetPanel]
- [Source: `_bmad-output/implementation-artifacts/7b-6-close-budget-pool-lifecycle-gap-with-disable-and-replace.md`#Senior Developer Review]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-03 23:25:50 +0800: BMAD create-story workflow executed for Story 7B.7 in YOLO mode.
- 2026-06-03: Read `.agents/skills/bmad-create-story/SKILL.md`, `discover-inputs.md`, `template.md`, and `checklist.md`.
- 2026-06-03: Customization resolver failed because local `python3` lacks `tomllib`; followed skill fallback by reading `.agents/skills/bmad-create-story/customize.toml`, `_bmad/custom/bmad-create-story.toml`, and `_bmad/custom/bmad-create-story.user.toml`. No team/user overrides found.
- 2026-06-03: Persistent facts glob `file:{project-root}/**/project-context.md` found no files.
- 2026-06-03: Loaded `_bmad/bmm/config.yaml`: `user_name=hth`, communication/document language Chinese, planning artifacts `_bmad-output/planning-artifacts`, implementation artifacts `_bmad-output/implementation-artifacts`.
- 2026-06-03: Loaded full `_bmad-output/implementation-artifacts/sprint-status.yaml`; confirmed `7b-7-restrict-budget-pool-creation-to-system-admin-and-rename-descendant-delegation: backlog`.
- 2026-06-03: Loaded relevant Epic 7B source from `_bmad-output/planning-artifacts/epics.md`, correct-course proposal `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03-budget-pool-lifecycle-cc.md`, PRD FR-9/FR-23, architecture process/data-flow/enforcement sections, previous story 7B.6, and recent git history.
- 2026-06-03: Read current route, middleware, controller/service/test and Default frontend implementation files for budget creation permission and delegation copy.
- 2026-06-03: No web research was required because the story uses existing project libraries and no new external API/framework behavior.
- 2026-06-03: Checklist validation applied: story includes exact AC, current-state notes for UPDATE files, previous story intelligence, permission guardrails, i18n scope, file boundaries, anti-patterns, and targeted tests.
- 2026-06-03 23:30:12 +0800: BMAD dev-story workflow started; customization resolver failed because local `python3` lacks `tomllib`, so fallback customization loading was applied. No team/user overrides found.
- 2026-06-03 23:30:12 +0800: Captured baseline commit `990423635e8ac7aa8332765c5d6807667ed0306e`; story and sprint status moved to in-progress.
- 2026-06-03 23:36:08 +0800: First `GOCACHE=/private/tmp/new-api-go-build go test ./controller/enterprise ./tests/api` run passed `controller/enterprise` but exposed a `tests/api` lifecycle workflow expectation mismatch; adjusted the test to assert resize-below-committed while the budget is active.
- 2026-06-03 23:39:16 +0800: Final validation passed: `GOCACHE=/private/tmp/new-api-go-build go test ./controller/enterprise ./tests/api`, `bun test src/features/enterprise-organization/enterprise-organization.test.tsx`, `bun run typecheck`, and `bun run i18n:sync`.
- 2026-06-03 23:53:06 +0800: Senior Developer Review (AI) executed via `bmad-story-automator-review`; story/file-list/git reality cross-check completed, source review excluded `_bmad-output` except required story/status updates, one medium UI layout issue auto-fixed, and validation rerun.
- 2026-06-04 00:25:14 +0800: Follow-up Senior Developer Review (AI) executed via `bmad-story-automator-review`; verified 7B.7 commit/file-list reality, auto-fixed non-admin empty-state copy and stale descendant test naming, then reran frontend test, typecheck, and i18n sync.

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- Story status set to ready-for-dev.
- Scope intentionally limited to create-route permission, frontend create-entry visibility, and budget delegation user-facing copy.
- Critical guardrail: do not remove department-admin budget viewing or allocation/delegation ability while removing create/pause/resume/resize control-plane entry points.
- Critical guardrail: backend route permission is authoritative; frontend hiding is required but insufficient alone.
- Implemented budget-pool creation control-plane tightening by moving `POST /api/enterprise/departments/:id/budget` to `middleware.EnterpriseAdmin()` while preserving budget GET/list/detail department-admin access.
- Updated backend/controller and API regression coverage so department admins can still read budget pools but receive `error.enterprise.permission.admin_required` on create, without creating false success/reject audit facts.
- Hid the Default frontend budget-pool creation card for non-admin roles while preserving the budget overview, list, member allocation, subordinate allocation, quota request, and lifecycle admin controls.
- Reworded the budget delegation UI to "Allocate Budget To Subordinate Department" semantics across en/zh/fr/ru/ja/vi, while preserving backend descendant validation and analytical descendant-scope copy.
- Validation passed: Go enterprise controller/API tests, enterprise organization frontend test, TypeScript typecheck, and i18n sync.
- Senior review auto-fix: non-admin budget workspaces now render the budget overview in a full-width single-column grid after the create-budget card is hidden, avoiding a narrow empty control-plane column.
- Follow-up senior review auto-fix: budget list empty state now uses neutral enterprise-admin-created pool guidance instead of telling non-admin users to create the first pool; frontend tests now assert the neutral copy and no longer keep descendant wording in the delegation schema test name.

### File List

- `_bmad-output/implementation-artifacts/7b-7-restrict-budget-pool-creation-to-system-admin-and-rename-descendant-delegation.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `router/enterprise-router.go`
- `controller/enterprise/department_budget_test.go`
- `tests/api/enterprise_department_budget_test.go`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/vi.json`

## Change Log

- 2026-06-03: Created Story 7B.7 context document and marked sprint story ready-for-dev.
- 2026-06-03: Implemented Story 7B.7: restricted budget-pool creation to enterprise admins, preserved department-admin budget viewing/allocation surfaces, renamed visible budget delegation action copy to subordinate allocation semantics, updated six frontend locales, and added backend/API/frontend regression coverage.
- 2026-06-03: Senior Developer Review (AI) completed; fixed non-admin budget overview grid layout, reran required Go/frontend/i18n validations, and marked story done.
- 2026-06-04: Follow-up Senior Developer Review (AI) completed; fixed non-admin empty-state budget pool copy, removed stale descendant wording from a frontend test name, reran frontend test/typecheck/i18n validation, and kept story done.

## Senior Developer Review (AI)

### Review Summary

- Outcome: Approve after auto-fix.
- Critical issues remaining: 0.
- High issues remaining: 0.
- Medium issues found: 3 fixed.
- Git/story File List discrepancy: no source-code discrepancy requiring action; untracked story-automator artifacts remain outside application source review scope.

### Findings and Fixes

- [x] [AI-Review][Medium] When the create-budget card was hidden for non-admin roles, the parent grid still used the admin two-column layout, leaving the budget overview constrained to the former 360px create-card column on wide screens. Fixed `DepartmentBudgetPanel` to switch to `xl:grid-cols-1` for non-admin users while preserving the two-column admin layout. [`web/default/src/features/enterprise-organization/index.tsx`]
- [x] [AI-Review][Medium] The budget-pool list empty state still told users to "Create the first pool" even after creation was restricted to enterprise administrators. Replaced the empty-state guidance with neutral enterprise-admin-created pool copy and added a regression assertion. [`web/default/src/features/enterprise-organization/index.tsx`, `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`, `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`]
- [x] [AI-Review][Medium] A frontend test name still described delegation as a "descendant target" despite the story requiring subordinate allocation language in visible/testing surfaces. Renamed the test to subordinate target semantics. [`web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`]

### Validation

- [x] `GOCACHE=/private/tmp/new-api-go-build go test ./controller/enterprise ./tests/api` - passed.
- [x] `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx` - passed.
- [x] `cd web/default && bun run typecheck` - passed.
- [x] `cd web/default && bun run i18n:sync` - passed.
- [x] `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx` - passed on follow-up review.
- [x] `cd web/default && bun run typecheck` - passed on follow-up review.
- [x] `cd web/default && bun run i18n:sync` - passed on follow-up review.

### Checklist

- [x] Story status verified as reviewable before review.
- [x] Acceptance Criteria cross-checked against implementation.
- [x] File List reviewed against git changes.
- [x] Tests mapped to backend permission, API workflow, frontend visibility, and i18n copy behavior.
- [x] Code quality and security reviewed on changed source files.
- [x] Review notes appended and sprint status synced.
