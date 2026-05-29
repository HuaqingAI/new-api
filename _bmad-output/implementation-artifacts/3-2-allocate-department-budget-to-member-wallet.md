---
baseline_commit: 8be5dda92af0d2f1d15744dde01944dc32a550d7
---

# Story 3.2: 将部门预算分配为成员 wallet

Status: done

## Story

As a 管理员,
I want 从部门预算池向成员分配 wallet,
so that 成员 API 消费可以优先使用部门分配额度。

## Acceptance Criteria

1. **Given** 部门预算池有可分配额度且目标用户属于该部门  
   **When** 管理员提交分配额度  
   **Then** 系统在目标用户名下创建一行 `subscriptions` 子钱包  
   **And** 子钱包带 `source_type='enterprise_allocation'` 与 `source_allocation_id`。
2. **Given** 创建 wallet 子分配  
   **When** 系统写入分配结果  
   **Then** `enterprise_quota_allocations` 记录父预算池、目标用户、目标 wallet、金额、类型快照、周期/过期快照、操作者和状态  
   **And** 该记录是配额分配审计单一来源，不再双写 `enterprise_admin_actions`。
3. **Given** 用户已有主钱包或自有钱包  
   **When** 部门 wallet 创建成功  
   **Then** 部门 wallet 默认排在主钱包之前参与现有 wallet 优先级  
   **And** 用户可调整顺序但不能自行删除部门 wallet。

## Tasks / Subtasks

- [x] 新增部门分配流水模型与迁移 (AC: 1, 2)
  - [x] 新增 `model/enterprise/quota_allocation.go`，定义 `enterprise_quota_allocations`，至少包含 `tenant_id`、`department_budget_id`、`department_id`、`target_user_id`、`wallet_id`、`actor_id`、`committed_quota`、`budget_type_snapshot`、`cycle_type_snapshot`、`cycle_started_at_snapshot`、`custom_seconds_snapshot`、`expires_at_snapshot`、`reason`、`before_budget_snapshot`、`after_budget_snapshot`、`status`、`created_at`、`updated_at`；JSON 快照字段如需存储结构化数据，使用 `TEXT` + `common.Marshal/common.UnmarshalJsonStr`，不要引入数据库专属 JSON 类型。[Source: _bmad-output/planning-artifacts/architecture.md#department-quota-semantics; _bmad-output/planning-artifacts/architecture.md#allocation-ledger; AGENTS.md#rule-1-json-package--use-commonjsongo; AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
  - [x] 将 `enterprise_quota_allocations` 接入 `model/enterprise/migration.go`，保持 SQLite / MySQL / PostgreSQL 三库兼容；不要依赖 `ALTER COLUMN`、JSONB、MySQL 专属函数或 PostgreSQL 专属操作符。[Source: model/enterprise/migration.go; AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
  - [x] 为 `model.UserSubscription` 增加 Story 3.2 所需的企业 wallet 元数据列与排序列，至少覆盖 `source_type`、`source_allocation_id`、以及实现“默认排在主钱包之前且可调序”所需的稳定排序字段；迁移必须兼容现有数据与三库，不得破坏既有订阅订单/套餐逻辑。[Source: _bmad-output/planning-artifacts/architecture.md#department-quota-semantics; _bmad-output/planning-artifacts/architecture.md#fr-10--以-wallet-子分配方式承接部门预算扣减-v11-重写; model/subscription.go; model/main.go]
- [x] 实现 quota allocation 服务与事务创建逻辑 (AC: 1, 2)
  - [x] 新增 `service/enterprise/quota_allocation.go`，提供创建分配的输入校验、用户存在性校验、目标用户是否属于该部门校验、父预算池状态校验，并复用当前企业分层组织，不要把企业逻辑散落到通用订阅 service 中。[Source: _bmad-output/planning-artifacts/epics.md#story-32-将部门预算分配为成员-wallet; router/enterprise-router.go; service/enterprise/department_budget.go]
  - [x] 创建分配时必须在单个数据库事务中完成三件事：更新父池、创建子 wallet、写入 `enterprise_quota_allocations`；任一步失败必须整体回滚，不允许出现父池已扣减但 wallet/ledger 缺失的状态。[Source: _bmad-output/planning-artifacts/architecture.md#data-flow; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-10-以-wallet-子分配方式承接部门预算扣减-v11-重写]
  - [x] `balance` 父池分配时按架构要求减少 `enterprise_department_budgets.remaining`，并用现有 GORM 原子表达式模式更新，后续 Story 3.3 再补足高并发铁律防线；本故事至少保证单事务内不会写出明显不一致快照。[Source: _bmad-output/planning-artifacts/architecture.md#concurrency-strategy; _bmad-output/planning-artifacts/architecture.md#data-flow]
  - [x] `subscription` 父池分配时不要发明新的周期系统；子 wallet 周期字段必须与当前 `NormalizeResetPeriod` / `ResetDueSubscriptions` 兼容，后续周期重置继续复用现有 `service/subscription_reset_task.go`。[Source: model/subscription.go#normalizeresetperiod; model/subscription.go#resetduesubscriptions-resets-subscriptions-whose-next_reset_time-has-passed; service/subscription_reset_task.go]
  - [x] 明确 Story 3.2 只实现“分配创建 + 排序参与 + 单一审计源”，不实现撤销、过期回收、父池状态联动任务或并发 50 压测护栏；这些分别属于 Story 3.3 与 3.4。[Source: _bmad-output/planning-artifacts/epics.md#story-33-强制预算铁律与并发安全; _bmad-output/planning-artifacts/epics.md#story-34-撤销和联动部门-wallet-状态]
- [x] 扩展现有订阅扣费链路以使用企业 wallet 元数据与优先级 (AC: 1, 3)
  - [x] 修改 `model/subscription.go` 中的查询与预扣费选择逻辑，使 relay 继续只走现有 `PreConsumeUserSubscription` / `PostConsumeUserSubscriptionDelta` 路径，但能够按新增排序字段优先消费企业 wallet，而不是继续按 `end_time desc, id desc` 的旧顺序“碰运气”。[Source: _bmad-output/planning-artifacts/architecture.md#no-relay-path-department-deduction; model/subscription.go#preconsumeusersubscription-pre-consumes-from-any-active-subscription-total-quota]
  - [x] 在 `GetAllActiveUserSubscriptions` / `GetAllUserSubscriptions` 返回中暴露前端排序与来源渲染所需字段，让用户可见部门 wallet 来源，同时保持旧订阅 UI/接口兼容。[Source: controller/subscription.go#getsubscriptionself; web/default/src/features/subscriptions/types.ts; web/default/src/features/subscriptions/api.ts]
  - [x] 禁止用户通过现有“删除订阅”路径删除 `source_type='enterprise_allocation'` 的钱包；若现有 admin 删除入口也能命中这类记录，需要在 Story 3.2 先加保护或显式限制，避免 AC 3 被当前硬删除能力破坏。[Source: _bmad-output/planning-artifacts/epics.md#story-32-将部门预算分配为成员-wallet; model/subscription.go#admindeleteusersubscription-hard-deletes-a-user-subscription; web/default/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.tsx]
- [x] 暴露企业 quota allocation API，且不双写 admin action (AC: 1, 2)
  - [x] 新增 `dto/enterprise/quota_allocation.go`，对分配请求中的可选标量继续使用指针类型并保留显式零值语义；控制器继续使用 `common.UnmarshalBodyReusable`。[Source: AGENTS.md#rule-6-upstream-relay-request-dtos--preserve-explicit-zero-values; controller/enterprise/department_budget.go]
  - [x] 新增 `controller/enterprise/quota_allocation.go` 和路由 `POST /api/enterprise/quota-allocations`，必要时补充列表/详情接口以支撑 Story 3.5 的状态展示，但本故事至少要有创建接口与最小查询能力供前端刷新分配结果。[Source: _bmad-output/planning-artifacts/architecture.md#api-style; _bmad-output/planning-artifacts/architecture.md#data-flow]
  - [x] 将新路由注册到 `router/enterprise-router.go`，继续走 `middleware.EnterpriseDepartmentAdmin` 或更严的企业权限守卫；分配动作绝不能调用 `service/enterprise/admin_action.go` 双写审计。[Source: router/enterprise-router.go; middleware/enterprise_dept_admin.go; _bmad-output/planning-artifacts/architecture.md#allocation-ledger; _bmad-output/planning-artifacts/architecture.md#data-flow]
- [x] 扩展 Default 前端的预算 tab 与用户订阅视图 (AC: 1, 3)
  - [x] 在 `web/default/src/features/enterprise-organization/` 现有预算池 tab 中增加“分配 wallet”入口，而不是新开无关页面；复用当前 TanStack Query query key 组织方式。[Source: web/default/src/features/enterprise-organization/index.tsx; web/default/src/features/enterprise-organization/api.ts; _bmad-output/planning-artifacts/architecture.md#frontend-architecture]
  - [x] 在 `web/default/src/features/enterprise-organization/api.ts`、`types.ts` 中新增 quota allocation 的请求/响应类型与 query key，前端创建成功后刷新预算池状态与目标用户 wallet 视图。[Source: web/default/src/features/enterprise-organization/api.ts; _bmad-output/planning-artifacts/architecture.md#state--data-fetching]
  - [x] 在用户订阅/钱包相关前端视图中显式标识 `enterprise_allocation` 来源、默认优先顺序，并提供“可调序但不可删除”的交互基础；若当前 UI 没有排序接口，需要在 Story 3.2 中把最小 reorder 入口一起落地，否则 AC 3 不成立。[Source: web/default/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.tsx; web/default/src/features/subscriptions/types.ts; _bmad-output/planning-artifacts/architecture.md#department-quota-semantics]
  - [x] 补齐 `en/zh/fr/ru/ja/vi` 文案，并运行 `bun run i18n:sync` 保持字典一致。[Source: AGENTS.md#internationalization-i18n; web/default/package.json]
- [x] 完成验证与回归检查 (AC: 1, 2, 3)
  - [x] Go 测试至少覆盖 `model/subscription.go` 的排序/来源扩展、`service/enterprise/quota_allocation.go` 的事务创建与失败回滚、`controller/enterprise/quota_allocation.go` 的权限与错误映射。[Source: _bmad-output/planning-artifacts/architecture.md#test-organization]
  - [x] 增加 `tests/api` 或等价回归用例，验证创建分配后能查到 `source_type='enterprise_allocation'` 的 wallet，且不会额外产生 `enterprise_admin_actions` 记录。[Source: _bmad-output/planning-artifacts/architecture.md#allocation-ledger; tests/api/enterprise_department_budget_test.go]
  - [x] 前端至少执行 `bun run i18n:sync`、`bun run typecheck`，并补充企业组织预算 tab / 订阅展示相关组件测试，覆盖创建成功、错误提示、来源标记与不可删除行为。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx; web/default/package.json]
  - [x] 手工确认未创建任何部门 budget / wallet 时，现有用户额度、订阅购买、订阅重置、relay 计费行为不变；Story 3.2 不得往 `relay/**`、OpenAI 兼容请求 DTO 或计费表达式系统引入部门字段。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-10-以-wallet-子分配方式承接部门预算扣减-v11-重写; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-11-保留现有计费表达式和倍率语义-v11-调整措辞]

## Dev Notes

- Story 3.2 的核心不是新增一张“企业钱包表”，而是把部门分配落进当前真正参与扣费的 `model.UserSubscription` / `user_subscriptions` 链路。架构文档多处用 `subscriptions` 作抽象表述，但仓库真实实现的预扣费入口是 `model.PreConsumeUserSubscription`，它当前直接按 `UserSubscription` 扣减；如果 dev agent 另外造一张平行表而不扩展这条链路，AC 1 和 AC 3 都不会成立。[Source: model/subscription.go#preconsumeusersubscription-pre-consumes-from-any-active-subscription-total-quota; service/funding_source.go; _bmad-output/planning-artifacts/architecture.md#department-quota-semantics]
- 当前 `UserSubscription` 只有 `Source string` 这类旧字段，没有 `source_type`、`source_allocation_id`、排序字段，也没有“企业 wallet 不可删除”的约束。Story 3.2 必须把这些元数据与行为限制补进去，否则只能把企业分配伪装成普通 admin 订阅，后续无法做单一审计源、状态联动和前端区分。[Source: model/subscription.go; web/default/src/features/subscriptions/types.ts; controller/subscription.go]
- 当前预扣费选择顺序是 `end_time asc, id asc`，与架构要求的“部门 wallet 默认在主钱包前”不一致。因此 Story 3.2 不只是新增分配接口，还必须改订阅查询/排序与展示层；否则企业 wallet 创建出来后可能不会被优先消耗。[Source: model/subscription.go#preconsumeusersubscription-pre-consumes-from-any-active-subscription-total-quota; _bmad-output/planning-artifacts/architecture.md#department-quota-semantics]
- `enterprise_department_budgets` 目前只有 `Remaining`，没有架构草案里提到的 `allocated_total`。Story 3.2 可以沿用现有模型增量扩展，但必须在故事实现中明确 subscription 父池如何记录“已承诺额度”，并与后续 Story 3.3 的铁律校验兼容；不要临时把 `remaining` 混用成所有语义。[Source: model/enterprise/department_budget.go; _bmad-output/planning-artifacts/architecture.md#data-flow]
- 配额分配是业务事实本身，因此**不要**沿用 Story 3.1 的 `enterprise_admin_actions` 审计模式。当前预算池创建走 `writeAdminAction` 是对的，但 Story 3.2 的 quota allocation 必须只写 `enterprise_quota_allocations`，否则会直接违反 Epic/架构要求。[Source: controller/enterprise/department_budget.go; service/enterprise/admin_action.go; _bmad-output/planning-artifacts/epics.md#story-32-将部门预算分配为成员-wallet; _bmad-output/planning-artifacts/architecture.md#allocation-ledger]
- 权限守卫继续复用企业模块既有模式。`middleware.EnterpriseDepartmentAdmin` 已能从 query/body 解析 `tenant_id`；新分配接口要么在路径里携带预算池/部门 ID 并复用该守卫，要么在 controller 内先解析预算池再做同等权限校验，但不要退回到普通 `AdminAuth` 放大全局权限。[Source: middleware/enterprise_dept_admin.go; router/enterprise-router.go]
- Story 3.2 仍然不能碰 relay 协议表面和计费表达式系统。企业预算只通过 wallet 选择顺序影响“扣哪个订阅”，而不是改变“这次请求算多少 quota”。`pkg/billingexpr/expr.md` 的语义在本故事里应保持原样，不新增部门变量。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-11-保留现有计费表达式和倍率语义-v11-调整措辞; _bmad-output/planning-artifacts/architecture.md#no-relay-path-department-deduction]
- 现有前端预算 tab 已经在 `enterprise-organization` feature 内，不需要新 feature。优先在这个 tab 中加入分配表单/列表，并在现有订阅展示里补“来源”和“顺序”信息，减少前后端割裂。[Source: web/default/src/features/enterprise-organization/index.tsx; web/default/src/features/enterprise-organization/api.ts]
- Story 3.1 的一个未决实现风险会影响 Story 3.2：预算池服务当前 `latestBudget` 只取一个最新 budget，而 PRD 允许“同一部门多个预算池，分配时显式选择目标池”。Story 3.2 若要按架构做 `POST /api/enterprise/quota-allocations`，就需要显式传 `department_budget_id` 并让预算池查询从“单一 latest”演进到“可枚举并按 ID 定位”。不要继续把“某部门只有一个预算池”当成隐含前提。[Source: service/enterprise/department_budget.go; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-9-创建部门预算池-v11-修订]

### Project Structure Notes

- 预计新增后端文件：
  - `model/enterprise/quota_allocation.go`
  - `service/enterprise/quota_allocation.go`
  - `controller/enterprise/quota_allocation.go`
  - `dto/enterprise/quota_allocation.go`
  - `service/enterprise/quota_allocation_test.go`
  - `controller/enterprise/quota_allocation_test.go`
- 预计新增或修改的现有后端文件：
  - `model/enterprise/migration.go`
  - `model/subscription.go`
  - `model/main.go`
  - `router/enterprise-router.go`
  - `controller/subscription.go`
  - `service/funding_source.go`（仅在需要同步返回展示/来源信息时；不要改资金来源语义）
- 预计新增或修改的前端文件：
  - `web/default/src/features/enterprise-organization/api.ts`
  - `web/default/src/features/enterprise-organization/types.ts`
  - `web/default/src/features/enterprise-organization/index.tsx`
  - `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - `web/default/src/features/subscriptions/types.ts`
  - `web/default/src/features/subscriptions/api.ts`
  - `web/default/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.tsx`
  - `web/default/src/i18n/locales/*.json`
- 对齐要点：
  - 企业代码继续集中在 `model/service/controller/enterprise`
  - 订阅扣费路径继续走现有 `model/subscription.go`
  - 低频管理动作与配额分配审计源严格分离

### References

- [epics.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/epics.md:422)
- [prd.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md:217)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:251)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:272)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:1149)
- [model/subscription.go](/Users/hq-it/repository/github/huaqingai/new-api/model/subscription.go:236)
- [model/subscription.go](/Users/hq-it/repository/github/huaqingai/new-api/model/subscription.go:770)
- [model/subscription.go](/Users/hq-it/repository/github/huaqingai/new-api/model/subscription.go:1072)
- [service/funding_source.go](/Users/hq-it/repository/github/huaqingai/new-api/service/funding_source.go:1)
- [service/subscription_reset_task.go](/Users/hq-it/repository/github/huaqingai/new-api/service/subscription_reset_task.go:1)
- [model/enterprise/department_budget.go](/Users/hq-it/repository/github/huaqingai/new-api/model/enterprise/department_budget.go:1)
- [service/enterprise/department_budget.go](/Users/hq-it/repository/github/huaqingai/new-api/service/enterprise/department_budget.go:1)
- [controller/enterprise/department_budget.go](/Users/hq-it/repository/github/huaqingai/new-api/controller/enterprise/department_budget.go:1)
- [service/enterprise/admin_action.go](/Users/hq-it/repository/github/huaqingai/new-api/service/enterprise/admin_action.go:1)
- [router/enterprise-router.go](/Users/hq-it/repository/github/huaqingai/new-api/router/enterprise-router.go:1)
- [middleware/enterprise_dept_admin.go](/Users/hq-it/repository/github/huaqingai/new-api/middleware/enterprise_dept_admin.go:1)
- [web/default/src/features/enterprise-organization/index.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/index.tsx:1)
- [web/default/src/features/enterprise-organization/api.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/api.ts:1)
- [web/default/src/features/subscriptions/types.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/subscriptions/types.ts:1)
- [web/default/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.tsx:1)

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story created from Epic 3 / Story 3.2 sources in `_bmad-output/planning-artifacts/`.
- Manual workflow resolution used because `_bmad/scripts/resolve_customization.py` requires Python 3.11+, but local `python3` lacks `tomllib`.
- Reviewed Story 3.1, current department budget implementation, current subscription/wallet flow, enterprise middleware/router, and latest commits ending at `8be5dda9`.
- Critical implementation gap identified and documented: current repository has no `enterprise_quota_allocations`, no enterprise wallet metadata on `UserSubscription`, and no wallet priority ordering support, so Story 3.2 must extend those existing primitives instead of assuming they already exist.
- Verified story 3.2 code paths with repo-local `GOCACHE` to avoid sandboxed system cache write failures; skipped unrelated `service/enterprise` DingTalk connectivity tests that require binding an `httptest` port, and instead ran the story-specific `Test(CreateQuotaAllocation|DepartmentBudget)` subset.
- Installed `web/default` dependencies with Bun using writable temp/cache directories, then fixed `enterprise-organization/index.tsx` form typing so `bun x tsc -b` passes under the story changes.

### Completion Notes List

- Story context explicitly distinguishes architecture abstraction (`subscriptions`) from the repository’s actual runtime primitive (`model.UserSubscription` / `user_subscriptions`) to prevent the dev agent from building an unused parallel wallet table.
- Added guardrails for single-source audit, relay non-intrusion, tenant-aware department permissions, and no-double-write behavior.
- Captured the hidden AC dependency that wallet priority and “cannot self-delete” require changes in both backend subscription logic and the existing subscription management UI/API.
- Flagged the mismatch between PRD “multiple budget pools per department” and current `latestBudget` implementation so Story 3.2 can select a concrete parent budget by ID instead of inheriting an incorrect single-pool assumption.
- Closed the remaining migration gap for SQLite upgrades by adding explicit `user_subscriptions` compatibility columns (`source_type`, `source_allocation_id`, `sort_order`, `is_primary`) and a regression test covering legacy-table backfill.
- Added the missing user-facing reorder path for self subscription wallets so enterprise allocations are not only displayed as managed/non-deletable, but can also be reprioritized by the member to satisfy AC 3.
- Validated backend story coverage with `go test ./model ./controller/enterprise ./tests/api` and `go test ./service/enterprise -run 'Test(CreateQuotaAllocation|DepartmentBudget)'`, and validated frontend typing plus locale synchronization with `bun x tsc -b` and `bun run i18n:sync`.

### File List

- `_bmad-output/implementation-artifacts/3-2-allocate-department-budget-to-member-wallet.md`
- `controller/enterprise/quota_allocation.go`
- `controller/enterprise/quota_allocation_test.go`
- `controller/subscription.go`
- `dto/enterprise/quota_allocation.go`
- `i18n/keys.go`
- `i18n/locales/en.yaml`
- `i18n/locales/zh-CN.yaml`
- `middleware/enterprise_dept_admin.go`
- `model/enterprise/migration.go`
- `model/enterprise/quota_allocation.go`
- `model/main.go`
- `model/subscription.go`
- `model/subscription_enterprise_wallet_test.go`
- `router/api-router.go`
- `router/enterprise-router.go`
- `service/enterprise/errors.go`
- `service/enterprise/quota_allocation.go`
- `service/enterprise/quota_allocation_test.go`
- `tests/api/enterprise_quota_allocation_test.go`
- `web/default/src/features/enterprise-organization/api.ts`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/types.ts`
- `web/default/src/features/subscriptions/api.ts`
- `web/default/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.test.tsx`
- `web/default/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.tsx`
- `web/default/src/features/subscriptions/types.ts`
- `web/default/src/features/wallet/components/subscription-plans-card.test.tsx`
- `web/default/src/features/wallet/components/subscription-plans-card.tsx`

### Change Log

- 2026-05-29: 完成 Story 3.2，实现企业预算分配 ledger、企业 wallet 事务创建、订阅优先级/删除保护、企业 quota allocation API、预算 tab 分配入口、用户自助 reorder 入口，以及后端/前端回归验证。
- 2026-05-29: 执行 story-automator review，修复非过期企业 wallet 被错误视为 inactive、quota allocation 列表按 budget_id 绕过部门校验，以及父预算池扣减缺少 `RowsAffected` 护栏的问题；补充对应回归测试并完成状态同步。

## Senior Developer Review (AI)

### Outcome

- Changes requested issues已修复，本次复核结果为通过。

### Findings Fixed

1. `model/subscription.go` 将 `end_time = 0` 的企业 wallet 排除在 active 查询之外，导致成功分配出的永久 wallet 不会参与 `GetAllActiveUserSubscriptions` / `PreConsumeUserSubscription`，直接破坏 AC 1 与 AC 3。已统一改为 `status='active' AND (end_time = 0 OR end_time > now)`。
2. `controller/enterprise/quota_allocation.go` 的列表接口仅依赖请求里的 `department_id` 通过中间件校验，但真实查询只按 `department_budget_id` 取数，存在用任意可管理部门 ID 读取其他预算池分配记录的越权面。已在 controller 中先按 budget 反查权属部门并校验与请求一致。
3. `service/enterprise/quota_allocation.go` 更新父预算池时只检查 `Error`，未检查原子扣减的 `RowsAffected`；在并发或陈旧快照场景下可能继续创建 wallet/ledger。已补上 `RowsAffected == 0 => ErrQuotaAllocationQuotaExceeded` 护栏。

### Verification

- `GOCACHE=$(pwd)/.tmp/go-build-cache go test ./model -run 'Test(GetAllUserSubscriptionsOrdersEnterpriseWalletFirst|AdminDeleteUserSubscriptionRejectsEnterpriseWallet|GetAllActiveUserSubscriptionsIncludesNonExpiringEnterpriseWallet|EnsureUserSubscriptionTableSQLiteAddsEnterpriseWalletColumns)'`
- `GOCACHE=$(pwd)/.tmp/go-build-cache go test ./controller/enterprise -run 'Test(QuotaAllocationAPIWorkflow|QuotaAllocationAPIRejectsUserOutsideDepartment|QuotaAllocationListRejectsMismatchedDepartmentID)'`
- `GOCACHE=$(pwd)/.tmp/go-build-cache go test ./service/enterprise -run 'Test(CreateQuotaAllocationCreatesWalletAndLedger|CreateQuotaAllocationRejectsUserOutsideDepartment|CreateQuotaAllocationRollsBackOnLedgerFailure|GetBudgetDepartmentReturnsBudgetOwnerDepartment)'`
- `cd web/default && bun run typecheck`
- `cd web/default && bun run test:e2e`
