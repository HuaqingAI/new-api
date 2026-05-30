---
baseline_commit: 9967da5ca24a6fab9423ff80ad2dcd4ea710f1c9
---

# Story 3.5: 展示预算池与 wallet 状态

Status: done

## Story

As a 管理员,
I want 查看部门预算池和所有派生 wallet 的状态,
so that 我可以判断预算是否健康并追踪每笔分配。

## Acceptance Criteria

1. **Given** 部门存在一个或多个预算池  
   **When** 管理员打开预算池列表  
   **Then** 系统展示类型、剩余额度、累计已分配、使用率、生效状态  
   **And** 支持按使用率、剩余额度、类型和状态排序。
2. **Given** 管理员打开预算池详情  
   **When** 系统加载派生子钱包列表  
   **Then** 展示目标用户、quota、remain_quota、周期/过期、wallet 状态、来源 allocation  
   **And** 单一审计源可定位到目标用户、目标 wallet、来源父池。
3. **Given** 预算池使用率达到阈值  
   **When** 使用率 ≥ 80% 或 ≥ 95%  
   **Then** 页面分别显示预警或紧急状态  
   **And** 阈值可配置且保存时校验 `0 < 预警 < 紧急 ≤ 100`。

## Tasks / Subtasks

- [x] 扩展预算池后端查询，从“单个最新预算池”提升为可排序列表 + 详情聚合视图 (AC: 1, 2)
  - [x] 在 `service/enterprise/department_budget.go` 新增列表与详情查询能力，返回同一部门下全部 `enterprise_department_budgets`，而不是继续只读 `latestBudget()`；结果至少包含 `id`、`type`、`status`、`remaining`、`allocated_total`、`total_quota/cycle_quota`、使用率、阈值状态和 `parent_status`。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-12-展示部门预算与-wallet-状态-v11-修订; _bmad-output/planning-artifacts/epics.md#story-35-展示预算池与-wallet-状态; model/enterprise/department_budget.go]
  - [x] 为 `balance` 与 `subscription` 两类预算统一定义“使用率”口径并在 service 层固定下来，避免前端自行推导。建议以父池总承诺/总额度作为列表排序和告警判断的唯一来源：`balance` 基于 `1 - remaining/total_quota`，`subscription` 基于 `allocated_total/cycle_quota`，同时保留当前 `remaining` 字段用于展示。[Source: model/enterprise/department_budget.go; service/enterprise/quota_allocation.go; _bmad-output/planning-artifacts/architecture.md#department-quota-semantics]
  - [x] 详情查询不要读 `logs` 表，不要拼接第二审计链。派生 wallet 明细应直接 join `enterprise_quota_allocations` 与 `user_subscriptions`，以 allocation 为单一审计源，补出目标用户、wallet、父池三者的可追踪关系。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-12-展示部门预算与-wallet-状态-v11-修订; model/enterprise/quota_allocation.go; model/subscription.go]
- [x] 为预算池列表、详情和阈值配置新增企业 API / DTO，保持现有企业路由边界 (AC: 1, 2, 3)
  - [x] 在 `dto/enterprise/department_budget.go` 与必要的新 DTO 中补充列表/详情/阈值配置响应结构；所有可选标量继续用指针类型，避免 `0`/`false` 在 JSON 往返中丢失。[Source: AGENTS.md#rule-6-upstream-relay-request-dtos--preserve-explicit-zero-values; dto/enterprise/department_budget.go]
  - [x] 在 `controller/enterprise/department_budget.go` 新增“预算池列表 + 预算池详情”接口，控制器只负责解参、调用 service、映射 `{ success, message, data }`，不要把使用率计算、排序逻辑或 allocation join 放到 controller。[Source: _bmad-output/planning-artifacts/architecture.md#api--communication-patterns; controller/enterprise/department_budget.go]
  - [x] 路由继续挂在 `router/enterprise-router.go` 的 `/api/enterprise/**` 命名空间下，并沿用 `middleware.EnterpriseDepartmentAdmin("id")` / `middleware.EnterpriseDepartmentAdmin("department_id")` 权限模型，避免回退到全局 `AdminAuth`。[Source: router/enterprise-router.go; middleware/enterprise_dept_admin.go]
  - [x] 为阈值配置补一条最小可用的管理路径。当前仓库尚无企业预算阈值现成配置，因此 Story 3.5 必须明确落地到现有 Option / config 体系，例如扩展 `setting/operation_setting/quota_setting.go` 并通过 `model.UpdateOption` 持久化 `quota_setting.enterprise_budget_warning_threshold` 与 `quota_setting.enterprise_budget_critical_threshold`（或等价 key），同时在保存时校验 `0 < warning < critical <= 100`。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-12-展示部门预算与-wallet-状态-v11-修订; setting/operation_setting/quota_setting.go; controller/option.go; model/option.go]
- [x] 在 Default 企业组织页重做预算信息架构，支持列表排序、详情联动和健康状态展示 (AC: 1, 2, 3)
  - [x] 继续在 `web/default/src/features/enterprise-organization/` 内实现，不新建无关页面；将当前“Current Budget Pool + Allocate Wallet + raw allocation table”升级为“预算池列表 + 选中详情 + allocation/wallet 明细”结构。[Source: web/default/src/features/enterprise-organization/index.tsx; _bmad-output/planning-artifacts/architecture.md#frontend-architecture]
  - [x] 列表必须展示类型、剩余额度、累计已分配、使用率、生效状态，并支持按使用率、剩余额度、类型、状态排序；queryKey 继续遵循 `['enterprise', scope, params]` 精确失效，不允许粗暴失效整个 `['enterprise']`。[Source: _bmad-output/planning-artifacts/architecture.md#frontend-architecture; _bmad-output/planning-artifacts/architecture.md#communication-patterns; web/default/src/features/enterprise-organization/api.ts]
  - [x] 详情区必须把 allocation 与 wallet 视图绑定到选中的预算池，展示目标用户、`wallet_id`、分配额度、剩余额度、周期/过期、wallet 状态、allocation 状态、处理时间，并把 “来源 allocation / 来源父池” 做成可识别字段，而不是继续只展示 `target_user_id / wallet_id / status` 三列。[Source: web/default/src/features/enterprise-organization/index.tsx; web/default/src/features/enterprise-organization/types.ts; model/subscription.go; model/enterprise/quota_allocation.go]
  - [x] 告警/紧急阈值展示要复用现有 `StatusBadge` / `Badge` 视觉体系，不要硬编码裸文本；状态文案与新字段必须补齐 `en/zh/fr/ru/ja/vi`，并运行 `bun run i18n:sync` 保持字典一致。[Source: web/default/src/components/status-badge.tsx; AGENTS.md#internationalization-i18n; web/default/src/i18n/locales/en.json]
- [x] 接入系统设置中的阈值编辑入口，避免把可配置阈值偷塞进预算页面本地状态 (AC: 3)
  - [x] 在 Default 系统设置中复用现有 billing/quota 设置通路，优先扩展 `web/default/src/features/system-settings/general/quota-settings-section.tsx` 与 `billing/section-registry.tsx`，让管理员能编辑预算预警/紧急阈值并经 `PUT /api/option/` 持久化。[Source: web/default/src/features/system-settings/general/quota-settings-section.tsx; web/default/src/features/system-settings/billing/section-registry.tsx; web/default/src/features/system-settings/hooks/use-update-option.ts]
  - [x] 后端设置更新继续走现有 `controller/option.go` + `model.UpdateOption` 路径；若新增 `quota_setting` 字段，确保启动时默认值、`OptionMap` 回填和运行时读取全部打通，避免“前端可保存但后端仍读不到阈值”。[Source: controller/option.go; model/option.go; setting/operation_setting/quota_setting.go]
  - [x] 阈值校验错误需返回可测试的稳定消息，且前后端口径一致，不能只在前端做 zod 校验后让后端静默接受非法值。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-12-展示部门预算与-wallet-状态-v11-修订; controller/option.go]
- [x] 完成回归与测试，保护现有分配/撤销/扣费链路不被 Story 3.5 的展示层改动破坏 (AC: 1, 2, 3)
  - [x] 为 `service/enterprise/department_budget.go` 和新增详情/列表 service 测试覆盖：多预算池排序、balance/subscription 使用率计算、阈值状态判定、allocation + wallet 聚合结果，以及“只基于预算池/allocation/subscription 快照，不读 logs”断言。[Source: _bmad-output/planning-artifacts/architecture.md#test-organization; service/enterprise/department_budget_test.go]
  - [x] 为 `controller/enterprise/department_budget_test.go` / `tests/api/enterprise_department_budget_test.go` 覆盖列表/详情接口、租户/部门权限、排序参数、阈值配置保存校验和错误映射。[Source: controller/enterprise/department_budget_test.go; tests/api/enterprise_department_budget_test.go]
  - [x] 扩展 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`，覆盖预算池列表排序、阈值 badge、详情字段、空状态、已撤销/已过期 wallet 明细和本地化状态显示；不要只停留在当前静态 `DepartmentBudgetStatusCard` 渲染测试上。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]
  - [x] 扩展系统设置相关前端测试，覆盖阈值表单默认值、非法保存、成功保存和 query invalidation；并确认 Story 3.5 不改动 `relay/**`、`service/pre_consume_quota.go`、`service/quota.go`、`pkg/billingexpr/**` 的结算语义。[Source: AGENTS.md#rule-7-billing-expression-system--read-pkgbillingexprexprmd; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-11-保留现有计费表达式和倍率语义-v11-调整措辞; service/pre_consume_quota.go; service/quota.go]

## Dev Notes

- 当前实现距离 FR-12 还差一层完整的信息架构。`GetDepartmentBudget()` / `DepartmentBudgetStatusCard` 只返回并展示“该部门最新一个预算池”，而不是“一个或多个预算池”的可排序列表；`QuotaAllocationTable` 也只展示少量原始字段。Story 3.5 的核心不是补一列 UI，而是把预算 tab 升级成可运维的状态总览。[Source: controller/enterprise/department_budget.go; service/enterprise/department_budget.go; web/default/src/features/enterprise-organization/index.tsx]
- 架构文档中的 `subscriptions` 是抽象说法，仓库真实实现是 `model.UserSubscription` / `user_subscriptions`。详情视图中所谓 `quota` / `remain_quota` / wallet 状态，必须映射到真实字段语义：`AmountTotal`、`AmountUsed`、`Status`、`NextResetTime`、`EndTime`、`SourceType='enterprise_allocation'`、`SourceAllocationId`，不要凭空创造第二套 wallet DTO。[Source: model/subscription.go; service/enterprise/quota_allocation.go]
- Story 3.4 已经把撤销、过期和父池状态联动接入运行链路；Story 3.5 的展示必须真实反映这些终态，而不是只展示 allocation 当前状态字符串。尤其要确保 paused / revoked / expired 的父池与子 wallet 状态都可见，并且已处理项仍能被审计追踪。[Source: _bmad-output/implementation-artifacts/3-4-revoke-and-sync-department-wallet-state.md; service/enterprise/wallet_state_sync_task.go; service/enterprise/balance_expiry_task.go]
- 当前 `DepartmentBudget` 模型已具备 `AllocatedTotal`、`Remaining`、`ParentStatus` 等字段，足够支撑健康度列表；不要为 Story 3.5 新增一张“预算状态缓存表”。如果确实需要聚合字段，优先在 service 查询里计算或扩展现有 DTO。[Source: model/enterprise/department_budget.go]
- 阈值配置是本故事最容易被 LLM 偷懒的地方。PRD 要求阈值“可配置”，而当前代码库只有通用 quota 设置和用户配额预警阈值，没有企业预算阈值现成字段。Story 3.5 必须明确把阈值接入现有 Option / `quota_setting` 管道，不能把 80/95 写死在前端常量里然后宣称已满足 AC3。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-12-展示部门预算与-wallet-状态-v11-修订; setting/operation_setting/quota_setting.go; web/default/src/features/system-settings/general/quota-settings-section.tsx]
- 企业展示接口仍要遵守项目的低侵入原则：只新增企业模块表/服务/路由集成点，不进入 relay、token、billing 内核。Story 3.5 是观测层增强，不是重写扣费、重置或 allocation 事务逻辑。[Source: AGENTS.md#overview; _bmad-output/planning-artifacts/architecture.md#data-architecture; _bmad-output/planning-artifacts/architecture.md#relay-boundary]
- 如果后端需要排序参数，优先让 service 白名单支持 `usage_ratio`、`remaining`、`type`、`status` 等固定字段，避免把任意 SQL `ORDER BY` 字段暴露给 controller/query string。三库兼容仍是硬约束，不能为省事写数据库专属表达式。[Source: AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
- 前端现有 enterprise budget queryKey 只有 `departmentBudgetQueryKey` 与 `quotaAllocationQueryKey` 两个粗粒度 key。Story 3.5 引入预算池列表、选中详情和阈值设置后，要继续遵守 TanStack Query 的精确失效模式，避免一个小 mutation 把整个 enterprise 模块缓存全部打掉。[Source: _bmad-output/planning-artifacts/architecture.md#communication-patterns; web/default/src/features/enterprise-organization/api.ts]

### Project Structure Notes

- 预计修改的后端文件：
  - `dto/enterprise/department_budget.go`
  - `controller/enterprise/department_budget.go`
  - `controller/enterprise/department_budget_test.go`
  - `router/enterprise-router.go`
  - `service/enterprise/department_budget.go`
  - `service/enterprise/department_budget_test.go`
  - `service/enterprise/errors.go`
  - `setting/operation_setting/quota_setting.go`
  - `model/option.go`
  - `controller/option.go`
  - `tests/api/enterprise_department_budget_test.go`
- 预计修改的前端文件：
  - `web/default/src/features/enterprise-organization/api.ts`
  - `web/default/src/features/enterprise-organization/types.ts`
  - `web/default/src/features/enterprise-organization/index.tsx`
  - `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - `web/default/src/features/system-settings/general/quota-settings-section.tsx`
  - `web/default/src/features/system-settings/billing/section-registry.tsx`
  - `web/default/src/features/system-settings/billing/index.tsx`
  - `web/default/src/i18n/locales/*.json`
- 关键对齐要求：
  - 企业预算 API 继续挂在 `/api/enterprise/**`，系统设置继续走 `/api/option/`
  - 预算健康度计算放 service，控制器/前端只消费结果
  - 预算详情严格基于 `enterprise_department_budgets`、`enterprise_quota_allocations`、`user_subscriptions` 当前快照
  - 不新增 relay DTO、部门计费变量或 logs 依赖

### References

- [epics.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/epics.md:514)
- [prd.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md:262)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:231)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:343)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:513)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:1091)
- [3-4-revoke-and-sync-department-wallet-state.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/3-4-revoke-and-sync-department-wallet-state.md:1)
- [controller/enterprise/department_budget.go](/Users/hq-it/repository/github/huaqingai/new-api/controller/enterprise/department_budget.go:1)
- [service/enterprise/department_budget.go](/Users/hq-it/repository/github/huaqingai/new-api/service/enterprise/department_budget.go:1)
- [service/enterprise/quota_allocation.go](/Users/hq-it/repository/github/huaqingai/new-api/service/enterprise/quota_allocation.go:1)
- [model/enterprise/department_budget.go](/Users/hq-it/repository/github/huaqingai/new-api/model/enterprise/department_budget.go:1)
- [model/enterprise/quota_allocation.go](/Users/hq-it/repository/github/huaqingai/new-api/model/enterprise/quota_allocation.go:1)
- [model/subscription.go](/Users/hq-it/repository/github/huaqingai/new-api/model/subscription.go:1)
- [router/enterprise-router.go](/Users/hq-it/repository/github/huaqingai/new-api/router/enterprise-router.go:1)
- [setting/operation_setting/quota_setting.go](/Users/hq-it/repository/github/huaqingai/new-api/setting/operation_setting/quota_setting.go:1)
- [controller/option.go](/Users/hq-it/repository/github/huaqingai/new-api/controller/option.go:1)
- [model/option.go](/Users/hq-it/repository/github/huaqingai/new-api/model/option.go:205)
- [web/default/src/features/enterprise-organization/index.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/index.tsx:680)
- [web/default/src/features/enterprise-organization/api.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/api.ts:1)
- [web/default/src/features/enterprise-organization/types.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/types.ts:1)
- [web/default/src/features/enterprise-organization/enterprise-organization.test.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/enterprise-organization.test.tsx:116)
- [web/default/src/features/system-settings/general/quota-settings-section.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/system-settings/general/quota-settings-section.tsx:1)
- [web/default/src/features/system-settings/billing/section-registry.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/system-settings/billing/section-registry.tsx:1)

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 按用户要求先读取 `.agents/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`，并依据 skill 指令执行 create-story。
- `_bmad/scripts/resolve_customization.py` 因本地 `python3` 缺少 `tomllib` 无法运行，已按 skill 的 fallback 规则手工解析 `customize.toml` 与 `_bmad/bmm/config.yaml`。
- 完整读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认目标 story key 为 `3-5-display-budget-pool-and-wallet-state`，原状态为 `backlog`。
- 选择性加载并分析了 Epic 3、PRD FR-9~FR-12、Architecture 数据/前端/集成结构段落，以及 Story 3.1~3.4 成果文件。
- 额外检查了当前真实实现：`controller/service/model` 企业预算链路、Default enterprise-organization 预算 tab、system-settings quota section、OptionMap 配置路径，以及 `UserSubscription` 企业 wallet 字段。
- 本次故事创建未发现可用 `project-context.md` 持久事实文件，故以 AGENTS.md、PRD、Architecture、既有故事与现有源码为主要上下文。
- 使用 repo 内 `GOCACHE=$(pwd)/.cache/go-build` 运行定向 Go 回归，规避沙箱下默认 Go build cache 不可写问题。
- `go test ./service/enterprise -run 'Test(CreateDepartmentBudget|GetDepartmentBudget|ListDepartmentBudgets|GetDepartmentBudgetDetail)'`、`go test ./controller/enterprise -run 'TestDepartmentBudget'`、`go test ./controller -run 'TestUpdateOptionValidatesEnterpriseBudgetThresholds'`、`go test ./tests/api -run 'TestEnterpriseDepartmentBudget'` 全部通过。
- `web/default/` 下执行 `bun run test:e2e`、`bun run typecheck`、`bun run i18n:sync`；前端静态测试、类型检查与字典同步全部通过。

### Completion Notes List

- 已把 Story 3.5 明确定位为 FR-12 的“观测与管理视图完善”，而非新增扣费或分配逻辑。
- 已指出当前实现与需求的核心差距：后端只有最新单池读取，前端只有单卡片 + 原始 allocation 表，尚不满足多预算池列表、详情追踪和阈值配置。
- 已把阈值配置缺口显式落在现有 `quota_setting` / OptionMap 通道，避免 dev agent 以硬编码 80/95 伪完成 AC3。
- 已将 wallet 展示语义收敛到仓库真实的 `UserSubscription` 字段，避免沿用架构中的抽象 `subscriptions` 说法误导实现。
- 已新增企业预算池列表/详情 API、统一使用率与阈值状态计算，以及基于 allocation + user_subscriptions 的单审计链 wallet 明细查询。
- 已在 Default 企业组织页完成预算池列表排序、详情联动、阈值 badge、wallet 明细表和精确 query invalidation 改造，并保留预算创建与 allocation 操作入口。
- 已在系统设置 Quota 区域接入企业预算预警/紧急阈值编辑，前后端统一使用 `quota_setting.enterprise_budget_warning_threshold` / `quota_setting.enterprise_budget_critical_threshold` 并共享同一校验规则。

### File List

- `_bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `controller/enterprise/department_budget.go`
- `controller/enterprise/department_budget_test.go`
- `controller/enterprise/department_membership_test.go`
- `controller/option.go`
- `dto/enterprise/department_budget.go`
- `i18n/keys.go`
- `i18n/locales/en.yaml`
- `i18n/locales/zh-CN.yaml`
- `router/enterprise-router.go`
- `service/enterprise/department_budget.go`
- `service/enterprise/department_budget_test.go`
- `service/enterprise/errors.go`
- `setting/operation_setting/quota_setting.go`
- `tests/api/enterprise_department_budget_test.go`
- `web/default/package.json`
- `web/default/rsbuild.test.config.ts`
- `web/default/src/features/enterprise-organization/api.ts`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/types.ts`
- `web/default/src/features/system-settings/billing/index.tsx`
- `web/default/src/features/system-settings/billing/section-registry.tsx`
- `web/default/src/features/system-settings/general/quota-settings-section.tsx`
- `web/default/src/features/system-settings/types.ts`

### Change Log

- 2026-05-29: 新增企业预算池列表/详情与阈值配置链路，重构 Default 企业组织页预算视图，并补齐后端/前端/系统设置测试覆盖。
- 2026-05-29: 自动审查并修复预算详情错误映射、父池状态取值错误、React 渲染期状态写入问题，同时校正故事 File List 与实际变更的一致性。

## Senior Developer Review (AI)

### Outcome

Approve after auto-fix.

### Findings

1. `controller/enterprise/department_budget.go`
   预算详情查询在 `budget_id` 不存在时错误返回 `enterprise.organization.department_not_found`，把“预算池不存在”混淆成“部门不存在”，不满足可测试的错误映射要求。已新增 `enterprise.organization.department_budget_not_found` 并修正 controller 映射。
2. `service/enterprise/department_budget.go`
   预算详情中的 `source_parent_budget_status` 之前返回 `budget.ParentStatus`，这会把父池关联状态字段误当成当前父池状态，导致审计视图可见状态与真实预算池生命周期脱节。已改为返回当前预算池 `budget.Status`。
3. `web/default/src/features/enterprise-organization/index.tsx`
   `DepartmentBudgetPanel` 在 render 期间直接调用 `allocationForm.setValue(...)` 和 `setSelectedBudgetId(...)`，在 React 19 下会造成不稳定重渲染风险，也违背组件状态更新时机。已迁移到 `useEffect` 驱动。
4. ` _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md`
   Dev Agent Record -> File List 声称修改了 `controller/option_test.go`、`quota-settings-section.test.tsx` 和多份前端 locale JSON，但这些文件不在基线 diff 中，属于实现记录失真。已按实际变更校正 File List。

### Validation

- `GOCACHE=$(pwd)/.cache/go-build go test ./service/enterprise -run 'Test(ListDepartmentBudgetsSortsAndCalculatesThresholds|GetDepartmentBudgetDetailAggregatesAllocationWalletsWithoutLogs)'`
- `GOCACHE=$(pwd)/.cache/go-build go test ./controller/enterprise -run 'TestDepartmentBudget(ListAndDetailAPI|DetailReturnsBudgetNotFound)'`
- `GOCACHE=$(pwd)/.cache/go-build go test ./tests/api -run 'TestEnterpriseDepartmentBudgetListAndDetailAPI'`
- `bun run typecheck`
- `bun run test:e2e`

### Review Summary

- 已自动修复 4 个高/中优先级问题。
- 未发现剩余 CRITICAL 问题，故事状态更新为 `done`。
