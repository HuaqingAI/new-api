---
baseline_commit: b87a374f9642172ae447076df8b7ddb1f34a5338
---

# Story 3.4: 撤销和联动部门 wallet 状态

Status: done

## Story

As a 管理员,
I want 撤销部门分配并在父预算池状态变化时批量联动子 wallet,
so that 未消费额度按规则失效或回收且审计可追溯。

## Acceptance Criteria

1. **Given** 管理员撤销 `balance` 型子分配  
   **When** wallet 仍有未消费余额  
   **Then** 父池回收金额等于 `min(wallet.remain_quota, max(0, allocated_quota - consumed))`  
   **And** allocation 状态标记为 `revoked`，wallet 状态标记为 inactive/revoked，不物理删除。
2. **Given** `balance` 型子 wallet 到达 `expires_at`  
   **When** `balance_expiry_task` 执行  
   **Then** 父池回收金额等于 `min(wallet.remain_quota_at_expiry, max(0, allocated_quota - consumed))`  
   **And** allocation 状态标记为 `expired`，wallet 状态失效。
3. **Given** 同一 `balance` 型过期或撤销事件已处理  
   **When** `balance_expiry_task` 或撤销逻辑重复执行  
   **Then** 父池 `remaining` 不会再次增加  
   **And** allocation 保留 `processed_at` 或等价幂等标记用于审计。
4. **Given** 管理员撤销 `subscription` 型子分配  
   **When** 撤销生效  
   **Then** 父池释放承诺额，子 wallet 状态标记为 `revoked`  
   **And** 下个周期不再 reset，该周期未用部分自然失效。
5. **Given** 父预算池暂停、过期或撤销  
   **When** `wallet_state_sync_task` 执行  
   **Then** 系统批量同步子 wallet 状态  
   **And** controller 层不逐个手工更新子 wallet。

## Tasks / Subtasks

- [x] 扩展父池 / allocation / wallet 状态模型与幂等审计字段 (AC: 1, 2, 3, 4, 5)
  - [x] 在 `model/enterprise/quota_allocation.go` 中把状态枚举从当前 `active/cancelled` 扩展到至少 `active/paused/revoked/expired`，并增加 `processed_at`（或等价的已处理时间戳组合）以支持撤销/过期幂等审计；迁移继续走 GORM + SQLite 补列模式，不引入数据库专属语法。[Source: _bmad-output/planning-artifacts/architecture.md#allocation-ledger; _bmad-output/planning-artifacts/epics.md#story-34-撤销和联动部门-wallet-状态; model/enterprise/quota_allocation.go; model/main.go; AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
  - [x] 在 `model/enterprise/department_budget.go` 中把父池持久状态扩展到可表达 `paused` / `revoked`，供后台任务扫描；不要把“父池当前状态”只保留在 controller 参数或临时变量里，因为 `wallet_state_sync_task` 必须基于数据库中的父池状态批量联动。[Source: _bmad-output/planning-artifacts/architecture.md#revocation-policy; _bmad-output/planning-artifacts/architecture.md#data-flow; model/enterprise/department_budget.go]
  - [x] 复核 `model.UserSubscription` 的现有 `Status` / `EndTime` / `NextResetTime` 语义，确保被撤销或过期的企业 wallet 会自动被现有 active 查询和 reset 逻辑跳过；不要新增平行 wallet 表，也不要依赖架构文档中的抽象名 `subscriptions` 去假设仓库里存在另一套运行时实体。[Source: model/subscription.go; _bmad-output/planning-artifacts/architecture.md#department-quota-semantics]
- [x] 实现 allocation 撤销事务与余额回收 / 承诺释放逻辑 (AC: 1, 3, 4)
  - [x] 在 `service/enterprise/quota_allocation.go` 增加明确的撤销入口（可放在同文件或拆到 `wallet_state_sync.go`，但企业配额逻辑必须继续集中在 `service/enterprise/`），事务内锁定 allocation、父池和目标 wallet，避免“父池已回收/释放但 wallet 状态未改”或相反的半成品状态。[Source: _bmad-output/planning-artifacts/architecture.md#source-organization; service/enterprise/quota_allocation.go]
  - [x] `balance` 型撤销按当前仓库真实字段计算未消费额度。仓库运行时 wallet 是 `model.UserSubscription`，没有 PRD 文案里的 `remain_quota` 列；实现时应基于 `AmountTotal` / `AmountUsed` 或现有 helper 得到未消费值，再映射到 `min(unspent, max(0, committed_quota - consumed))` 的产品规则，而不是发明第二套钱包余额字段。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-10-以-wallet-子分配方式承接部门预算扣减-v11-重写; model/subscription.go; service/enterprise/quota_allocation.go]
  - [x] `subscription` 型撤销必须原子释放父池 `allocated_total`，并把 `remaining` 重新收敛到 `cycle_quota - allocated_total` 语义；撤销后的子 wallet 不能再被下个周期 `ResetDueSubscriptions` 重置，但当前周期已发出未消费部分也不能回收回父池。[Source: _bmad-output/planning-artifacts/architecture.md#department-quota-semantics; _bmad-output/planning-artifacts/architecture.md#revocation-policy; service/subscription_reset_task.go]
  - [x] 撤销逻辑重复执行时必须是幂等 no-op：父池 `remaining` / `allocated_total` 不能二次变化，allocation 只保留第一次处理的 `processed_at` 与最终状态；不要把重复执行实现成“再加一次余额然后覆盖状态”。[Source: _bmad-output/planning-artifacts/epics.md#story-34-撤销和联动部门-wallet-状态]
- [x] 新增后台任务与启动入口，承接过期回收和父池状态联动 (AC: 2, 3, 5)
  - [x] 新增 `service/enterprise/balance_expiry_task.go`，只处理企业 `balance` 型 allocation；识别条件优先基于 `enterprise_quota_allocations.budget_type_snapshot='balance'` 或等价 join，不要依赖仓库中并不存在的 `is_subscription` 列。[Source: _bmad-output/planning-artifacts/architecture.md#data-flow; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-10-以-wallet-子分配方式承接部门预算扣减-v11-重写; model/subscription.go]
  - [x] 新增 `service/enterprise/wallet_state_sync_task.go`，按父池状态批量同步关联 allocation 和 `user_subscriptions`；实现应以批量 update / 分页扫描为主，controller 不能在请求路径里逐条循环更新所有子 wallet。[Source: _bmad-output/planning-artifacts/architecture.md#data-flow; _bmad-output/planning-artifacts/architecture.md#revocation-policy]
  - [x] 为企业任务增加明确启动入口。当前仓库只有 `service.StartSubscriptionQuotaResetTask()`，没有企业 scheduler 或这两个 task 的 bootstrap；Story 3.4 必须新增 `service/enterprise/scheduler.go` 或等价入口，并从 `main.go` 在 `common.IsMasterNode` 约束下启动，保持与现有任务模型一致。[Source: _bmad-output/planning-artifacts/architecture.md#file-organization-patterns; _bmad-output/planning-artifacts/architecture.md#data-flow; main.go; service/subscription_reset_task.go]
- [x] 暴露最小可用的撤销 API，并保持企业权限边界 (AC: 1, 4, 5)
  - [x] 在 `controller/enterprise/quota_allocation.go` / `router/enterprise-router.go` 增加撤销接口，优先沿用现有 quota allocation 路由族与 `middleware.EnterpriseDepartmentAdmin` 权限模型；不要退化成全局 `AdminAuth`，也不要在 controller 里内嵌父池扫描和批量子 wallet 更新逻辑。[Source: controller/enterprise/quota_allocation.go; router/enterprise-router.go; middleware/enterprise_dept_admin.go]
  - [x] 如果撤销接口需要新的请求 DTO，继续遵循可选标量使用指针类型的项目规则；虽然此故事主要是管理端 mutation，但 DTO 形状仍要和项目现有 `common.UnmarshalBodyReusable` / `{ success, message, data }` 契约对齐。[Source: AGENTS.md#rule-6-upstream-relay-request-dtos--preserve-explicit-zero-values; controller/enterprise/department_budget.go]
  - [x] 明确本故事只需要最小可用的 child revoke 能力和后台 sync，避免为了“父池暂停/撤销”另起一整套新页面；若需补充父池状态变更 API，优先在现有部门预算模块内最小扩展，而不是新开 feature。[Source: web/default/src/features/enterprise-organization/index.tsx; _bmad-output/planning-artifacts/architecture.md#frontend-architecture]
- [x] 更新预算 tab 的状态展示与撤销交互，为 Story 3.5 铺路但不重做信息架构 (AC: 1, 4, 5)
  - [x] 在 `web/default/src/features/enterprise-organization/types.ts`、`api.ts`、`index.tsx` 中补充 revoke action、processed/status 字段展示，以及需要的 query invalidation；继续复用当前预算 tab 和 `QuotaAllocationTable`，不要新建无关页面。[Source: web/default/src/features/enterprise-organization/types.ts; web/default/src/features/enterprise-organization/api.ts; web/default/src/features/enterprise-organization/index.tsx]
  - [x] 补齐前端 i18n 文案，使 `revoked` / `expired` / `paused` / 撤销按钮 / 失败提示在 `en/zh/fr/ru/ja/vi` 都可渲染；若新增 key，运行 `bun run i18n:sync`。[Source: AGENTS.md#internationalization-i18n; web/default/src/i18n/locales/en.json; web/default/src/i18n/locales/zh.json]
  - [x] 预算 tab 只需要提供最小可执行的撤销入口与状态可视化。完整“预算池与 wallet 状态总览”仍由 Story 3.5 承接，因此不要在 3.4 里重构整个预算页表格或汇总卡布局。[Source: _bmad-output/planning-artifacts/epics.md#story-35-展示预算池与-wallet-状态]
- [x] 完成回归与幂等验证，保护现有计费链路零侵入 (AC: 1, 2, 3, 4, 5)
  - [x] 在 `service/enterprise/quota_allocation_test.go` 增加 `balance` 撤销、`subscription` 撤销、重复撤销 no-op、父池状态同步、过期回收幂等用例；新增 `service/enterprise/wallet_state_sync_test.go` / `balance_expiry_task_test.go` 或同目录等价测试文件，覆盖 `IsMasterNode` 之外的可直接运行函数。[Source: _bmad-output/planning-artifacts/architecture.md#test-organization]
  - [x] 在 `controller/enterprise/quota_allocation_test.go` 与 `tests/api/enterprise_quota_allocation_test.go` 覆盖撤销接口的权限、错误映射、重复调用、列表状态刷新和“未物理删除 wallet”断言。[Source: controller/enterprise/quota_allocation_test.go; tests/api/enterprise_quota_allocation_test.go]
  - [x] 前端至少补充预算 tab 组件测试，覆盖撤销按钮、状态 badge、幂等提示和刷新行为；继续沿用当前 feature 内测试布局，不引入新的前端 harness。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]
  - [x] 回归确认 `relay/**`、`service/funding_source.go`、`pkg/billingexpr/**`、上游请求 DTO 和普通用户钱包行为不因本故事改变；Story 3.4 是企业 wallet 生命周期管理，不是重写扣费公式或 relay 路径。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-11-保留现有计费表达式和倍率语义-v11-调整措辞; _bmad-output/planning-artifacts/architecture.md#no-relay-path-department-deduction; AGENTS.md#rule-7-billing-expression-system--read-pkgbillingexprexprmd]

## Dev Notes

- 当前仓库真实运行时钱包实体是 `model.UserSubscription`，而不是架构文档抽象描述里的 `subscriptions` 表字段集合。Story 3.4 必须在现有 `AmountTotal` / `AmountUsed` / `Status` / `NextResetTime` 语义上完成撤销与过期处理，不能另外造一张“企业 wallet 状态表”然后希望 relay 自动感知。[Source: model/subscription.go; service/enterprise/quota_allocation.go]
- `service/enterprise/quota_allocation.go` 目前只有 `Create()` 路径，没有 revoke path；`model/enterprise/quota_allocation.go` 目前也只有 `active/cancelled` 两态。这是 Story 3.4 的首要实现缺口，不能靠 controller 层临时写 SQL 绕过去。[Source: service/enterprise/quota_allocation.go; model/enterprise/quota_allocation.go]
- 架构要求 `wallet_state_sync_task` 和 `balance_expiry_task`，但当前仓库里这两个文件和启动入口都不存在；`main.go` 目前只启动 `StartSubscriptionQuotaResetTask()`。因此 Story 3.4 不是“补几行状态判断”，而是要真正把企业后台任务接到 master-node 启动链路里。[Source: main.go; service/subscription_reset_task.go; _bmad-output/planning-artifacts/architecture.md#data-flow]
- `balance_expiry_task` 的识别逻辑不能直接照抄架构里的 `is_subscription=false`，因为当前 `user_subscriptions` 模型没有这个字段。最稳妥的实现是基于 allocation 的 `BudgetTypeSnapshot` / 父池类型来识别 balance 分配，再联到 wallet 处理过期与回收。[Source: model/subscription.go; model/enterprise/quota_allocation.go; _bmad-output/planning-artifacts/architecture.md#department-quota-semantics]
- 现有 `ResetDueSubscriptions()` 只会处理 active subscription；因此 revoke/expire enterprise wallet 的最低风险做法是让 wallet 脱离 active 集合，而不是给 reset task 再塞一层企业特殊分支。Story 3.4 应尽量复用既有过滤语义，而不是复制一份重置逻辑到企业模块。[Source: model/subscription.go; service/subscription_reset_task.go]
- Story 3.3 已把订阅型父池语义收敛到 `allocated_total`。Story 3.4 的 subscription revoke 必须和这个语义兼容，释放承诺额时同步回写 `remaining = cycle_quota - allocated_total`，否则 Story 3.5 的状态展示会继续读到错误剩余额度。[Source: _bmad-output/implementation-artifacts/3-3-enforce-budget-invariants-and-concurrency-safety.md; service/enterprise/quota_allocation.go]
- 前端当前 `QuotaAllocationTable` 只展示 `target_user_id / committed_quota / wallet_id / status / created_at`，没有 revoke action，也没有 processed/expired 信息。Story 3.4 需要最小扩展这张表，但不要重构整个预算页面，因为 Story 3.5 还要承接完整状态展示。[Source: web/default/src/features/enterprise-organization/index.tsx]
- 本故事仍然不能碰 `relay/**`、`service/funding_source.go`、`service/billing_session.go`、`pkg/billingexpr/**` 的协议和结算语义。企业预算生命周期只能通过 wallet 状态影响“这个钱包是否还会被选中”，不能把部门判断注入模型计费或表达式上下文。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-11-保留现有计费表达式和倍率语义-v11-调整措辞; service/funding_source.go; service/billing_session.go; pkg/billingexpr/expr.md]

### Project Structure Notes

- 预计新增后端文件：
  - `service/enterprise/wallet_state_sync_task.go`
  - `service/enterprise/balance_expiry_task.go`
  - `service/enterprise/wallet_state_sync_test.go`
  - `service/enterprise/balance_expiry_task_test.go`
- 预计修改的后端文件：
  - `service/enterprise/quota_allocation.go`
  - `service/enterprise/errors.go`
  - `controller/enterprise/quota_allocation.go`
  - `controller/enterprise/quota_allocation_test.go`
  - `router/enterprise-router.go`
  - `model/enterprise/quota_allocation.go`
  - `model/enterprise/department_budget.go`
  - `model/enterprise/migration.go`
  - `model/main.go`
  - `main.go`
  - `model/subscription.go`
  - `tests/api/enterprise_quota_allocation_test.go`
- 预计修改的前端文件：
  - `web/default/src/features/enterprise-organization/api.ts`
  - `web/default/src/features/enterprise-organization/types.ts`
  - `web/default/src/features/enterprise-organization/index.tsx`
  - `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - `web/default/src/i18n/locales/*.json`
- 对齐要求：
  - 企业生命周期逻辑继续集中在 `model/service/controller/enterprise`
  - 运行时 wallet 选择与 reset 继续复用 `model/subscription.go` / `service/subscription_reset_task.go`
  - 后台批量任务通过单一启动入口挂到 `main.go`，避免把 enterprise task 启动散落到多个包里

### References

- [epics.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/epics.md:479)
- [prd.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md:242)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:251)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:282)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:1149)
- [3-3-enforce-budget-invariants-and-concurrency-safety.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/3-3-enforce-budget-invariants-and-concurrency-safety.md:1)
- [service/enterprise/quota_allocation.go](/Users/hq-it/repository/github/huaqingai/new-api/service/enterprise/quota_allocation.go:1)
- [service/enterprise/errors.go](/Users/hq-it/repository/github/huaqingai/new-api/service/enterprise/errors.go:1)
- [controller/enterprise/quota_allocation.go](/Users/hq-it/repository/github/huaqingai/new-api/controller/enterprise/quota_allocation.go:1)
- [controller/enterprise/quota_allocation_test.go](/Users/hq-it/repository/github/huaqingai/new-api/controller/enterprise/quota_allocation_test.go:1)
- [service/enterprise/quota_allocation_test.go](/Users/hq-it/repository/github/huaqingai/new-api/service/enterprise/quota_allocation_test.go:1)
- [model/enterprise/department_budget.go](/Users/hq-it/repository/github/huaqingai/new-api/model/enterprise/department_budget.go:1)
- [model/enterprise/quota_allocation.go](/Users/hq-it/repository/github/huaqingai/new-api/model/enterprise/quota_allocation.go:1)
- [model/subscription.go](/Users/hq-it/repository/github/huaqingai/new-api/model/subscription.go:1)
- [model/main.go](/Users/hq-it/repository/github/huaqingai/new-api/model/main.go:199)
- [main.go](/Users/hq-it/repository/github/huaqingai/new-api/main.go:117)
- [service/subscription_reset_task.go](/Users/hq-it/repository/github/huaqingai/new-api/service/subscription_reset_task.go:1)
- [router/enterprise-router.go](/Users/hq-it/repository/github/huaqingai/new-api/router/enterprise-router.go:1)
- [middleware/enterprise_dept_admin.go](/Users/hq-it/repository/github/huaqingai/new-api/middleware/enterprise_dept_admin.go:1)
- [web/default/src/features/enterprise-organization/api.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/api.ts:1)
- [web/default/src/features/enterprise-organization/index.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/index.tsx:1000)
- [web/default/src/features/enterprise-organization/types.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/types.ts:1)
- [pkg/billingexpr/expr.md](/Users/hq-it/repository/github/huaqingai/new-api/pkg/billingexpr/expr.md:1)

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 按 `bmad-create-story` 工作流手工解析 skill 配置，因为本地 `python3` 缺少 `tomllib`，无法运行 `_bmad/scripts/resolve_customization.py`。
- 读取了 Epic 3 / Story 3.4、PRD FR-10 / FR-11、Architecture V1.1、Sprint Status，以及已完成的 Story 3.2 / 3.3 故事文件。
- 检查了当前实现中的关键文件：`service/enterprise/quota_allocation.go`、`service/enterprise/errors.go`、`controller/enterprise/quota_allocation.go`、`router/enterprise-router.go`、`model/enterprise/department_budget.go`、`model/enterprise/quota_allocation.go`、`model/subscription.go`、`service/subscription_reset_task.go`、`main.go`、`web/default/src/features/enterprise-organization/*`。
- 额外确认了一个关键现实差异：架构文档要求的 `wallet_state_sync_task` / `balance_expiry_task` / 企业 scheduler 入口在当前仓库中尚不存在，Story 3.4 必须显式补齐这些文件和启动挂载点。
- 本次故事创建未发现任何 `project-context.md` 持久事实文件，已改为以 `AGENTS.md`、PRD、Architecture 和既有实现为主要上下文来源。
- 当前基线 commit 为 `b87a374f9642172ae447076df8b7ddb1f34a5338`（Story 3.3 完成后）。

### Completion Notes List

- 已把 Story 3.4 从“抽象的撤销需求”收敛到当前仓库真实结构：`model.UserSubscription` 是运行时 wallet，`enterprise_quota_allocations` 是审计源，撤销/过期必须在这两者上落地。
- 已明确指出当前缺失的企业后台任务与启动入口，避免 dev agent 误以为只需补 controller 或 SQL。
- 已将 `balance` 与 `subscription` 两种撤销语义拆开，并和 Story 3.3 的 `allocated_total` 语义对齐。
- 已为前端范围设边界：本故事只需在现有预算 tab 增加最小 revoke/status 能力，完整状态总览留给 Story 3.5。
- 已实现企业 allocation 的手动撤销、余额型过期回收、父池状态驱动的 wallet 批量联动，以及 `processed_at` 幂等审计字段。
- 已把企业 scheduler 接到 `main.go` 的 master-node 启动链路，并补齐 service/controller/API/前端测试覆盖撤销权限、重复调用和状态刷新。
- 已验证 Story 3.4 相关 Go 测试、预算页前端测试和 `web/default` 构建通过；全包后端回归中仅剩与本故事无关的 `dingtalk_connectivity_test` 因沙箱禁止本地端口监听失败。

### File List

- `_bmad-output/implementation-artifacts/3-4-revoke-and-sync-department-wallet-state.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `controller/enterprise/quota_allocation.go`
- `controller/enterprise/quota_allocation_test.go`
- `dto/enterprise/quota_allocation.go`
- `main.go`
- `model/enterprise/department_budget.go`
- `model/enterprise/quota_allocation.go`
- `model/main.go`
- `router/enterprise-router.go`
- `service/enterprise/balance_expiry_task.go`
- `service/enterprise/balance_expiry_task_test.go`
- `service/enterprise/errors.go`
- `service/enterprise/quota_allocation.go`
- `service/enterprise/quota_allocation_test.go`
- `service/enterprise/scheduler.go`
- `service/enterprise/wallet_state_sync_task.go`
- `service/enterprise/wallet_state_sync_test.go`
- `tests/api/enterprise_quota_allocation_test.go`
- `web/default/src/features/enterprise-organization/api.ts`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/types.ts`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/i18n/locales/zh.json`

### Change Log

- 2026-05-29: 为企业 quota allocation 增加手动撤销入口、`processed_at` 审计字段、余额型回收公式和订阅型承诺释放逻辑，确保重复撤销/过期处理保持幂等。
- 2026-05-29: 新增 `balance_expiry_task`、`wallet_state_sync_task` 和企业 scheduler 启动入口，并在 `main.go` 的 `IsMasterNode` 链路下接入。
- 2026-05-29: 扩展 controller/API/前端预算 tab，补充撤销按钮、`processed_at` / 状态展示、query invalidation 与六语种文案。
- 2026-05-29: 补齐 service/controller/API/前端测试，验证 balance revoke、subscription revoke、重复调用 no-op、父池状态联动、过期回收、权限边界和 UI 状态刷新。
- 2026-05-29: AI 代码审查后修复 paused 子 wallet 在父池 revoked/expired 与过期任务路径中无法终态处理的问题，并把预算/分配状态展示改为本地化标签。

## Senior Developer Review (AI)

### Reviewer

hth

### Date

2026-05-29 16:31:46 +0800

### Outcome

Approve

### Findings

1. `service/enterprise/wallet_state_sync_task.go` 只扫描 `active` allocation，导致父预算池先 `paused`、后 `revoked/expired` 时，子 allocation 和 wallet 会卡在 paused，无法满足 AC4/AC5 的最终撤销联动要求。已修复为 revoked/expired 同步同时处理 `active` 与 `paused` 子项，并补充回归测试。
2. `service/enterprise/balance_expiry_task.go` 只处理 `active` 的 balance allocation，导致已 paused 且到期的余额型 wallet 不会进入 `expired` 回收路径，违背 AC2/AC3。已修复为过期任务覆盖 `active/paused`，并补充回归测试。
3. `web/default/src/features/enterprise-organization/index.tsx` 直接渲染后端原始状态字符串，Story 声称补齐 revoked/expired/paused 状态展示与多语种文案，但 UI 实际仍显示裸枚举。已修复为统一映射到本地化标签，组件测试同步更新。

### Fixes Applied

- 扩展 `QuotaAllocationService.Revoke()` 对 `active/paused` wallet 的终态更新。
- 扩展 `SyncWalletStates()` 与 `ExpireBalanceAllocations()` 的状态筛选条件，覆盖 paused 子项。
- 新增 paused→revoked 与 paused→expired 回归测试。
- 统一预算池与 allocation 表格状态文案为本地化显示。

### Validation

- `GOCACHE=$(pwd)/.cache/go-build go test ./service/enterprise ./controller/enterprise ./tests/api -run 'QuotaAllocation|WalletState|BalanceExpiry'`
- `bun test src/features/enterprise-organization/enterprise-organization.test.tsx`
- `bun run build`
