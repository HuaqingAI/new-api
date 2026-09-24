---
status: approved-applied
project: new-api
created: 2026-05-28
approved: 2026-05-28
applied: 2026-05-28
workflow: bmad-correct-course
mode: batch
trigger_source: _bmad-output/planning-artifacts/implementation-readiness-report-2026-05-28.md
recommended_scope: minor
applied_to:
  - _bmad-output/planning-artifacts/epics.md
---

# Sprint Change Proposal: 修正 Story 验收口径缺口

## 1. Issue Summary

### Trigger

本次 Correct Course 由 Implementation Readiness 报告触发。报告结论为 **NEEDS WORK**：PRD、Architecture 与 FR 覆盖整体充分，但 `epics.md` 中少数 Story 的 Acceptance Criteria 没有完整继承 PRD / Architecture 中的关键验收约束。

### Core Problem

问题类型：原需求理解在 Story 拆分阶段被弱化，不是战略转向、不是技术不可行，也不是 MVP 范围变化。

核心问题：

- Story 1.4 在 Epic 1 中引用 Epic 5 才引入的风险事件权限，形成前向依赖。
- Story 3.3 没有保留 PRD FR10 对并发、混合计费模型和预算一致性的量化验收。
- Story 3.4 没有保留 PRD FR10 对 `balance` 过期回收公式和幂等性的验收。
- Story 4.1 偏向内部聚合任务，缺少管理员可观察的 API/查询完成边界。
- 没有独立 UX 文档，复杂后台页面只能依赖 Story AC 约束 UI 行为，因此关键页面/空态/错误态需要写得更具体。

### Evidence

- Readiness report: `_bmad-output/planning-artifacts/implementation-readiness-report-2026-05-28.md`
- Current epics: `_bmad-output/planning-artifacts/epics.md`
- PRD baseline: `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`
- Architecture baseline: `_bmad-output/planning-artifacts/architecture.md`

## 2. Impact Analysis

### Epic Impact

| Epic | Impact | Required Change |
| ---- | ------ | --------------- |
| Epic 1 | Story 1.4 has a forward dependency on Epic 5 risk events. | Reword Story 1.4 AC to cover enterprise organization/admin-action audit permissions only. |
| Epic 2 | No impact. | No change. |
| Epic 3 | Story 3.3 and 3.4 are weaker than PRD FR10 / Architecture V1.1. | Strengthen concurrency, iron-rule, balance-expiry, and idempotency ACs. |
| Epic 4 | Story 4.1 lacks a query/API completion boundary. | Add enterprise usage API/snapshot acceptance. |
| Epic 5 | No structural impact. Existing Story 5.1 already owns risk-field visibility. | No change, but keep risk-event visibility checks here rather than in Epic 1. |

### PRD Impact

No PRD changes are required. The PRD already contains the stronger requirements:

- FR10 defines wallet sub-allocation, relay non-involvement, concurrency expectations, balance expiry recovery, and idempotency.
- FR13/FR14/FR20 define multi-department duplicate-counting and disclaimer behavior.
- NFR1/NFR5/NFR7 define cross-DB compatibility, consistency, and performance gates.

### Architecture Impact

No architecture changes are required. Architecture already supports the proposed story edits:

- Enterprise APIs under `/api/enterprise/**`
- `enterprise_usage_snapshots`
- `enterprise_quota_allocations`
- `wallet_state_sync_task`
- `balance_expiry_task`
- no relay path modification
- Default/Classic frontend split and i18n guardrails

### UX Impact

No standalone UX artifact exists. This proposal does not create a full UX spec. Instead, it strengthens story-level UX acceptance where the readiness report identified risk:

- Story 1.1 empty-state behavior for organization page before sync.
- Story 4.1 query/API completion boundary so Story 4.2 dashboard behavior is implementable.
- Existing Story 4.2/4.4/5.6 disclaimer requirements remain load-bearing.

### Technical Impact

No code has been implemented yet, so no rollback is needed. The change is limited to planning artifacts, primarily `epics.md`.

## 3. Recommended Approach

### Selected Path

Option 1: **Direct Adjustment**

### Rationale

The PRD and Architecture are not in conflict. The issue is localized to Story acceptance criteria. Directly patching `epics.md` preserves the current epic structure, keeps the MVP intact, and prevents implementation agents from under-testing the highest-risk areas.

### Alternatives Considered

- Potential Rollback: Not applicable. No implementation work needs rollback.
- PRD MVP Review: Not needed. MVP remains achievable and scope does not need reduction.
- New Epic: Not needed. All FRs already have 100% epic coverage.

### Effort and Risk

- Effort: Low
- Risk: Low if story ACs are patched before sprint planning
- Timeline impact: Minimal; this should be done before `bmad-sprint-planning`

## 4. Detailed Change Proposals

### Proposal 1: Story 1.1 Add Empty State Acceptance

Story: `Story 1.1: 查看企业部门树`

Section: Acceptance Criteria

OLD:

```markdown
**Given** 企业模块已启用
**When** 管理员打开企业组织页面
**Then** 系统显示部门树、部门名称、父子关系、状态、外部来源标识和同步异常状态
**And** 支持至少 3 层部门结构，部门停用或删除后不物理删除，历史名称可追溯。
```

NEW:

```markdown
**Given** 企业模块已启用
**When** 管理员打开企业组织页面
**Then** 系统显示部门树、部门名称、父子关系、状态、外部来源标识和同步异常状态
**And** 支持至少 3 层部门结构，部门停用或删除后不物理删除，历史名称可追溯。

**Given** 企业模块已启用但尚未完成任何部门同步或手工创建
**When** 管理员打开企业组织页面
**Then** API 返回空数组而非 `null`
**And** Default 前端显示空状态和可执行入口，引导有权限的管理员进入钉钉配置或同步流程。
```

Rationale: 让 Epic 1 在钉钉同步前也能独立验收，并补齐缺少独立 UX 文档时最容易遗漏的空态。

### Proposal 2: Story 1.4 Remove Forward Dependency

Story: `Story 1.4: 记录企业组织管理审计`

Section: Acceptance Criteria

OLD:

```markdown
**Given** 管理员查看企业组织操作记录
**When** 按时间、动作类型或对象筛选
**Then** 系统返回分页列表
**And** 普通用户不能查看其他部门风险事件或管理员专用字段。
```

NEW:

```markdown
**Given** 管理员查看企业组织操作记录
**When** 按时间、动作类型或对象筛选
**Then** 系统返回分页列表
**And** 每条记录至少包含 action_id、actor_id、action_type、object_type、object_id、created_at 和 diff_summary。

**Given** 普通用户或无企业管理权限用户访问企业组织操作记录
**When** 请求 `enterprise_admin_actions` 列表或详情
**Then** 系统拒绝访问并返回 `error.enterprise.permission.admin_required`
**And** 不暴露管理员专用审计字段。
```

Rationale: 风险事件属于 Epic 5，不能成为 Epic 1 的验收依赖。此处只验证企业组织管理审计的权限边界。

### Proposal 3: Story 3.3 Restore Quantitative Concurrency Acceptance

Story: `Story 3.3: 强制预算铁律与并发安全`

Section: Acceptance Criteria

OLD:

```markdown
**Given** 多个管理员或任务并发分配同一父预算池
**When** 分配请求同时提交
**Then** 系统使用事务和原子 update 防止父池余额被超扣
**And** 最终 `enterprise_quota_allocations` 与父池剩余额度一致。
```

NEW:

```markdown
**Given** 多个管理员或后台任务并发分配同一父预算池
**When** ≥50 个分配请求同时提交，且请求总额超过父池 `total_quota` 或 `cycle_quota`
**Then** 系统只允许不违反铁律的请求成功
**And** 失败请求返回 `enterprise_budget_insufficient`，并区分 `balance_remaining_insufficient` 与 `subscription_cycle_allocated_exceeded`
**And** 数据库中不出现父池负余额、重复 wallet、重复 allocation 或 `Σ children.committed_quota` 超过父池额度的状态。

**Given** 已存在部门预算子 wallet，且系统同时处理固定计费模型与 `tiered_expr` 模型请求
**When** ≥50 并发负载连续运行 ≥10 分钟
**Then** 每个子 wallet 满足 `consumed = quota - remain_quota`
**And** 父池铁律始终成立，`enterprise_quota_allocations`、`subscriptions` 子 wallet 与父池快照一致
**And** `relay/**` 中不新增部门查询、部门变量或企业 DTO 字段。
```

Rationale: PRD FR10 / NFR5 的核心风险是并发与一致性。Story 必须给实现和测试代理明确的量化验收目标。

### Proposal 4: Story 3.4 Restore Balance Expiry Formula and Idempotency

Story: `Story 3.4: 撤销和联动部门 wallet 状态`

Section: Acceptance Criteria

OLD:

```markdown
**Given** 管理员撤销 `balance` 型子分配
**When** wallet 仍有未消费余额
**Then** 父池按规则回收未消费部分
**And** wallet 状态标记为 revoked，不物理删除。
```

NEW:

```markdown
**Given** 管理员撤销 `balance` 型子分配
**When** wallet 仍有未消费余额
**Then** 父池回收金额等于 `min(wallet.remain_quota, max(0, allocated_quota - consumed))`
**And** allocation 状态标记为 `revoked`，wallet 状态标记为 inactive/revoked，不物理删除。

**Given** `balance` 型子 wallet 到达 `expires_at`
**When** `balance_expiry_task` 执行
**Then** 父池回收金额等于 `min(wallet.remain_quota_at_expiry, max(0, allocated_quota - consumed))`
**And** allocation 状态标记为 `expired`，wallet 状态失效。

**Given** 同一 `balance` 型过期或撤销事件已处理
**When** `balance_expiry_task` 或撤销逻辑重复执行
**Then** 父池 `remaining` 不会再次增加
**And** allocation 保留 processed_at 或等价幂等标记用于审计。
```

Rationale: "按规则回收"不可测试。PRD 已给出公式和幂等要求，Story 必须继承。

### Proposal 5: Story 4.1 Add API Completion Boundary

Story: `Story 4.1: 聚合部门用量快照`

Section: Acceptance Criteria

OLD:

```markdown
**Given** `logs` 累计 100 万条且部门数 ≤ 100
**When** 查询近 30 天部门用量首屏
**Then** 系统 P95 响应目标 < 3 秒
**And** 默认查询不得依赖逐条扫描全量日志。
```

NEW:

```markdown
**Given** 部门用量聚合任务执行完成
**When** 管理员通过 `/api/enterprise/usage/department-summary` 查询同一时间范围
**Then** API 从 `enterprise_usage_snapshots` 或等价聚合表返回部门汇总数据
**And** 每行至少包含 dept_id、dept_name、window_start、window_end、request_count、prompt_tokens、completion_tokens、quota、user_count 和 model_distribution。

**Given** 聚合范围内存在未归属用户消费
**When** 查询 `/api/enterprise/usage/department-summary`
**Then** 响应包含 `dept_id = null`、`dept_name = "未归属"` 的行
**And** 数组字段缺省返回 `[]`，不返回 `null`。

**Given** `logs` 累计 100 万条且部门数 ≤ 100
**When** 查询近 30 天部门用量首屏
**Then** 系统 P95 响应目标 < 3 秒
**And** 默认查询不得依赖逐条扫描全量日志。
```

Rationale: Story 4.1 需要让下游 Story 4.2 有可消费的 API 和字段边界，否则它只是内部任务实现。

## 5. Checklist Execution Summary

| Checklist Item | Status | Notes |
| -------------- | ------ | ----- |
| 1.1 Triggering story identified | Done | Triggered by readiness review across Story 1.4, 3.3, 3.4, 4.1. |
| 1.2 Core problem defined | Done | Story-level acceptance drift from PRD/Architecture. |
| 1.3 Evidence gathered | Done | Readiness report + PRD + Architecture + current Epics. |
| 2.1 Current epic impact | Done | Epic 1/3/4 affected. |
| 2.2 Epic-level changes | Done | No epic scope change required. |
| 2.3 Remaining epics reviewed | Done | Epic 2/5 unaffected structurally. |
| 2.4 New/obsolete epics | Done | No new or obsolete epics. |
| 2.5 Epic order/priority | Done | No resequencing required. |
| 3.1 PRD conflicts | Done | No PRD conflict. PRD is source of stronger ACs. |
| 3.2 Architecture conflicts | Done | No architecture conflict. Architecture already supports fixes. |
| 3.3 UI/UX impact | Action-needed | No UX doc exists; story-level UX AC must be strengthened. |
| 3.4 Secondary artifacts | Done | No CI/deployment changes. Sprint status only changes if sprint plan already exists. |
| 4.1 Direct Adjustment | Viable | Low effort, low risk. |
| 4.2 Rollback | Not viable | No implementation to roll back. |
| 4.3 PRD MVP Review | Not viable | MVP unchanged. |
| 4.4 Recommended path | Done | Direct Adjustment. |
| 5.1-5.5 Proposal components | Done | Included in this document. |
| 6.1-6.5 Final review/handoff | Pending user approval | Requires explicit approval before applying edits. |

## 6. Implementation Handoff

### Scope Classification

Minor.

Reason: This proposal modifies existing Story acceptance criteria only. It does not change PRD scope, Architecture decisions, Epic order, data model direction, or MVP.

### Handoff Recipients

- Product Owner / planning agent: approve story text changes and keep traceability intact.
- Developer agent: after approval, update `epics.md` exactly as proposed before sprint planning.
- Test architecture agent: later derive ATDD/load/concurrency tests from Story 3.3 and Story 3.4.

### Success Criteria

- `epics.md` no longer contains Epic 1 forward dependency on risk events.
- Story 3.3 includes PRD-level concurrency and budget invariant acceptance.
- Story 3.4 includes exact balance-expiry recovery formula and idempotency acceptance.
- Story 4.1 exposes an observable API/snapshot completion boundary.
- Readiness status can be rechecked without the same major findings.

## 7. Approval Request

Please review this proposal.

Options:

- Continue / approve: apply the proposed `epics.md` edits.
- Edit: revise one or more proposed changes before applying.
- Skip: leave planning artifacts unchanged.

## 8. Approval and Application Log

- 2026-05-28: User approved the proposal with `approve`.
- 2026-05-28: Applied proposed edits to `_bmad-output/planning-artifacts/epics.md`.
- 2026-05-28: Correct Course workflow complete. Issue addressed: story acceptance criteria drift from PRD/Architecture. Change scope: minor. Artifacts modified: `_bmad-output/planning-artifacts/epics.md`, `_bmad-output/planning-artifacts/sprint-change-proposal-2026-05-28.md`. Routed to: sprint planning / story validation.
