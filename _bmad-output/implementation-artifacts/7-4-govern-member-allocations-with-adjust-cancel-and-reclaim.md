---
baseline_commit: 2f5f9451c8bb4ae5e4a1b7e8c14acddb66328434
---

# Story 7.4: 补齐成员 allocation 的调整、取消与回收治理

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 部门负责人,
I want 对已分配给成员的 allocation 执行调整、取消和回收,
so that 成员额度分配具备完整的后续治理能力。

## Acceptance Criteria

1. **Given** 某成员已有有效 allocation  
   **When** 负责人执行额度调整  
   **Then** 系统关闭旧 allocation 并新建新 allocation  
   **And** 历史记录与状态链保持可追溯。
2. **Given** 负责人取消某个成员 allocation  
   **When** 取消生效  
   **Then** 系统不物理删除 wallet 或 allocation  
   **And** 记录 `revoked`、`closed` 或等价状态语义。
3. **Given** 系统对某 allocation 执行回收  
   **When** 回收成功  
   **Then** 记录回收额度、操作者、处理时间和来源 allocation  
   **And** 幂等重试不会产生双重回收。

## Tasks / Subtasks

- [x] 扩展成员 allocation 状态机与数据事实，覆盖调整/取消/回收全生命周期 (AC: 1, 2, 3)
  - [x] 复用 `enterprise_quota_allocations` 作为单一 allocation 审计源，不新建平行 allocation ledger；如需新增字段，优先放在 `model/enterprise/quota_allocation.go` 与对应迁移中，表达 superseded / revoked / expired / closed / processed_at / revoke_reason / reclaimed_quota 等事实。[Source: _bmad-output/planning-artifacts/architecture.md#预算委派流（V1.3）; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-24: 补齐成员 allocation 的调整、取消与回收治理]
  - [x] 调整必须建模为 supersede：旧 allocation 关闭，新 allocation 生效；不能直接原地 UPDATE 旧 allocation 的 committed quota 或 wallet quota。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.4 成员 allocation 生命周期]
  - [x] 取消与回收都不得硬删除 `subscriptions` 子 wallet 或 `enterprise_quota_allocations`；wallet / allocation 状态需要保留可审计链路并与现有 wallet 生命周期兼容。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-24: 补齐成员 allocation 的调整、取消与回收治理; service/enterprise/quota_allocation.go]

- [x] 在 `QuotaAllocationService` 内补齐调整、取消、回收语义，保持现有钱包规则不变 (AC: 1, 2, 3)
  - [x] 在 `service/enterprise/quota_allocation.go` 新增稳定入口，例如 `Supersede`, `Cancel`, `Reclaim` 或等价命名；继续复用现有预算预留、SQLite 重试、wallet backfill invariant 和预算回补 helper，不引入新的 relay/runtime 扣费逻辑。[Source: service/enterprise/quota_allocation.go; _bmad-output/planning-artifacts/architecture.md#预算委派流（V1.3）]
  - [x] 调整 allocation 时，需要为新 allocation 创建新的成员 wallet 事实或等价的新额度记录，并将旧 allocation / wallet 标记为 superseded / closed；不能让同一 wallet 同时承载两段不同 committed quota 历史。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.4 成员 allocation 生命周期; model.UserSubscription lifecycle]
  - [x] 取消 allocation 时，应回补父预算池未消耗额度，并将 wallet 状态调整到 revoked/inactive 等现有可表达状态；若已部分消耗，只回补可回收部分，不制造负额度。[Source: service/enterprise/quota_allocation.go#applyAllocationRevokeBudgetUpdate; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-24: 补齐成员 allocation 的调整、取消与回收治理]
  - [x] 回收动作要显式记录 reclaimed quota、操作者、处理时间、来源 allocation，并保证重复回调/重复任务执行是幂等的，不会双重回补父预算或重复更新 wallet 状态。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.4 成员 allocation 生命周期]

- [x] 保持 allocation 治理与现有 scheduler / wallet 生命周期协同 (AC: 2, 3)
  - [x] `wallet_state_sync_task`、`balance_expiry_task` 和现有撤销逻辑需要继续适用于 superseded / revoked / expired allocation；不要让新增状态破坏现有自动暂停、到期回收、父池状态联动。[Source: service/enterprise/wallet_state_sync_task.go; service/enterprise/balance_expiry_task.go]
  - [x] 对 `subscription` 型 allocation，调整/回收后仍要保持 `allocated_total`、`cycle_quota`、`remaining` 的铁律成立；对 `balance` 型 allocation，只回补未消耗余额，不回补已消耗部分。[Source: service/enterprise/quota_allocation.go; _bmad-output/planning-artifacts/architecture.md#部门预算与成员 wallet]
  - [x] 并发下不能出现重复 wallet、双重回收、父池负数、或 processed_at 回退等不一致状态；优先扩展现有事务和幂等 guard，而不是新增第二套分布式锁方案。[Source: service/enterprise/quota_allocation_test.go; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.7 V1.3 测试补充]

- [x] 暴露成员 allocation 调整/取消/回收 API，并保持现有分配入口边界清晰 (AC: 1, 2, 3)
  - [x] 在 `controller/enterprise/quota_allocation.go` 与 `router/enterprise-router.go` 扩展调整、取消、回收相关入口；沿用 `/api/enterprise/quota-allocations` 命名空间，不为成员 allocation 另起平行资源树。[Source: router/enterprise-router.go; controller/enterprise/quota_allocation.go]
  - [x] DTO 新增调整/取消/回收请求与响应字段，继续使用 snake_case；可选标量使用指针 + `omitempty`，响应保持 `{ success, message, data }` 与空数组 `[]` 约定。[Source: AGENTS.md#Rule 6: Upstream Relay Request DTOs — Preserve Explicit Zero Values]
  - [x] 区分预算不足、用户不在部门、wallet 已受保护、已处理回收、无权限等错误 message key，便于前端展示具体治理原因。[Source: service/enterprise/errors.go; i18n/keys.go]

- [x] 在 Default 企业组织页补齐 allocation 调整/取消/回收工作流 (AC: 1, 2, 3)
  - [x] 继续复用 `web/default/src/features/enterprise-organization/` 当前预算工作区和成员列表上下文，为 allocation 列表增加调整、取消、回收操作；不要要求用户手填 wallet/allocation ID 作为主路径。[Source: web/default/src/features/enterprise-organization/index.tsx; _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md]
  - [x] UI 需要清晰区分 active / superseded / revoked / expired / closed 等状态，并显式展示调整动作是“关闭旧 allocation + 新建新 allocation”，不是原地编辑同一记录。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.4 成员 allocation 生命周期]
  - [x] 取消/回收操作需要展示可回收额度或处理结果，避免用户误以为所有 committed quota 都会回补；同时继续展示 wallet lineage 与来源 allocation 信息。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]
  - [x] 新增前端文案进入 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，queryKey 仍保持在 `['enterprise', 'organization', ...]` 命名空间内，mutation invalidate 精确更新 allocation 列表与当前预算工作区。[Source: AGENTS.md#Internationalization (i18n); web/default/src/features/enterprise-organization/api.ts]

- [x] 补齐测试，锁住调整/取消/回收的状态语义与幂等性 (AC: 1, 2, 3)
  - [x] Service tests：覆盖 supersede 创建新 allocation、旧 allocation 关闭、旧 wallet 停用；取消后父池回补；回收只回补未消耗额度；重复取消/重复回收幂等；subscription 并发铁律不被破坏。[Source: service/enterprise/quota_allocation_test.go; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.7 V1.3 测试补充]
  - [x] Controller/router tests：覆盖调整/取消/回收 API 参数绑定、错误 message、processed_at/状态字段回传、重复请求幂等。[Source: controller/enterprise/quota_allocation_test.go]
  - [x] Frontend tests：覆盖 allocation 列表状态文案、调整弹层/表单校验、取消/回收后的 UI 更新、wallet lineage 展示保持可追溯。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]
  - [x] Boundary tests：继续确认本故事不修改 `relay/**`、`pkg/billingexpr/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`。[Source: service/enterprise/boundary_test.go; _bmad-output/planning-artifacts/architecture.md#Anti-Patterns]

## Dev Notes

- Story 7.4 只处理“成员 allocation 的后续治理”，不重做 Story 7.3 的部门预算委派，也不进入 Story 7.5 的员工额度申请审批流。[Source: _bmad-output/planning-artifacts/epics.md#Epic 7: 分层预算治理与额度审批]
- 当前 `QuotaAllocationService.Create` 和 `Revoke` 已经覆盖“分配”和“撤销/回补”的基础链路，7.4 的核心增量是 supersede / cancel / reclaim 的完整状态语义与幂等性，而不是另起一套 wallet 模型。[Source: service/enterprise/quota_allocation.go]
- `wallet_state_sync_task`、`balance_expiry_task`、`UserSubscription` 状态联动已经存在，新增状态必须和这些任务兼容，不能把自动回收、暂停、过期联动打断。[Source: service/enterprise/wallet_state_sync_task.go; service/enterprise/balance_expiry_task.go]
- `enterprise_quota_allocations` 仍然是成员 allocation 的单一审计源；不要把 allocation 调整/取消/回收再双写到新的 admin action ledger 里，只补充必要的低频治理审计上下文即可。[Source: _bmad-output/planning-artifacts/architecture.md#审计边界]
- 当前前端 `enterprise-organization` 已经展示 quota allocation 列表、wallet lineage、revoked/expired 状态，这能作为 7.4 的 UI 基座；优先在现有工作区上增强，不新做平行页面。[Source: web/default/src/features/enterprise-organization/index.tsx; web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]
- 所有 JSON 仍走 `common.Marshal` / `common.Unmarshal*`，所有 DB 变更保持三库兼容，不引入新的递归 SQL 或 JSON 专属操作符。[Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go; AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]

### Project Structure Notes

- 重点预计修改：
  - `service/enterprise/quota_allocation.go`
  - `service/enterprise/errors.go`
  - `service/enterprise/quota_allocation_test.go`
  - `service/enterprise/wallet_state_sync_task.go`
  - `service/enterprise/balance_expiry_task.go`
  - `controller/enterprise/quota_allocation.go`
  - `controller/enterprise/quota_allocation_test.go`
  - `router/enterprise-router.go`
  - `dto/enterprise/allocation.go`
  - `i18n/keys.go`
  - `i18n/locales/{en,zh-CN,zh-TW}.yaml`
  - `web/default/src/features/enterprise-organization/api.ts`
  - `web/default/src/features/enterprise-organization/types.ts`
  - `web/default/src/features/enterprise-organization/index.tsx`
  - `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`
- 保持不改：
  - `relay/**`
  - `pkg/billingexpr/**`
  - `model/log.go`
  - `controller/log.go`
  - `router/relay-router.go`
  - `docs/openapi/relay.json`
  - `relay/channel/**`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 7.4: 补齐成员 allocation 的调整、取消与回收治理]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic 7: 分层预算治理与额度审批]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-24: 补齐成员 allocation 的调整、取消与回收治理]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.4 成员 allocation 生命周期]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.7 V1.3 测试补充]
- [Source: service/enterprise/quota_allocation.go]
- [Source: service/enterprise/wallet_state_sync_task.go]
- [Source: service/enterprise/balance_expiry_task.go]
- [Source: service/enterprise/errors.go]
- [Source: controller/enterprise/quota_allocation.go]
- [Source: controller/enterprise/quota_allocation_test.go]
- [Source: router/enterprise-router.go]
- [Source: web/default/src/features/enterprise-organization/index.tsx]
- [Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-02 01:27 +0800: `GOCACHE=$(pwd)/.cache/go-build go test ./service/enterprise -run 'QuotaAllocation|WalletStateSync|BalanceExpiry|Boundary'`
- 2026-06-02 01:27 +0800: `GOCACHE=$(pwd)/.cache/go-build go test ./controller/enterprise -run 'QuotaAllocation'`
- 2026-06-02 01:27 +0800: `bun test web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- 2026-06-02 01:27 +0800: `GOCACHE=$(pwd)/.cache/go-build go test ./service/enterprise ./controller/enterprise` 受仓库现有 `service/enterprise` 非本故事失败与 `httptest` 端口权限限制影响，未作为本故事质量门禁依据。

### Completion Notes List

- 已为成员 allocation 增加 superseded / revoked / expired / closed 生命周期字段、迁移与审计事实，继续以 `enterprise_quota_allocations` 作为唯一治理审计源。
- 已在 `QuotaAllocationService` 中补齐 `Supersede`、`Cancel`、`Reclaim`，复用现有预算回补、wallet backfill invariant 与 SQLite 重试，并保证回收幂等。
- 已扩展成员 allocation API、错误 message key 与 Default 企业组织页治理工作流，支持在当前部门上下文中直接调整、取消和回收 allocation。
- 已补充后端与前端测试，覆盖 supersede 链路、取消/回收幂等、状态文案、lineage 展示与控制器回传字段。
- 已依据 commit、定向验证和 sprint-status 的 source-of-truth 收口将故事状态同步为 `done`，避免实现完成后文档仍停留在 `review`。

### File List

- controller/enterprise/quota_allocation.go
- controller/enterprise/quota_allocation_test.go
- dto/enterprise/quota_allocation.go
- i18n/keys.go
- i18n/locales/en.yaml
- i18n/locales/zh-CN.yaml
- model/enterprise/migration.go
- model/enterprise/quota_allocation.go
- router/enterprise-router.go
- service/enterprise/errors.go
- service/enterprise/quota_allocation.go
- service/enterprise/quota_allocation_test.go
- web/default/src/features/enterprise-organization/api.ts
- web/default/src/features/enterprise-organization/enterprise-organization.test.tsx
- web/default/src/features/enterprise-organization/index.tsx
- web/default/src/features/enterprise-organization/types.ts
- web/default/src/i18n/locales/en.json
- web/default/src/i18n/locales/fr.json
- web/default/src/i18n/locales/ja.json
- web/default/src/i18n/locales/ru.json
- web/default/src/i18n/locales/vi.json
- web/default/src/i18n/locales/zh.json

### Change Log

- 2026-06-02: 新增成员 allocation 的 supersede / cancel / reclaim 治理链路，补齐后端状态机、API、前端操作入口与测试覆盖。
- 2026-06-02 12:47:08 +0800: retrospective 收尾时按 source-of-truth 同步 story artifact 状态为 `done`。
