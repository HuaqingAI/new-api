# Story 7A.3: 放宽预算池类型约束至单池不可变

Status: backlog

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
   **Then** 系统仍然拒绝  
   **And** 继续保持“单预算池实例类型不可变”。
3. **Given** 同一部门存在多个不同类型预算池  
   **When** 进行预算委派、员工申请或成员分配  
   **Then** 系统按当前显式选中的预算池做类型兼容性校验  
   **And** 不再把整个部门视为单一预算类型。

## Tasks / Subtasks

- [ ] 修正预算池创建约束粒度 (AC: 1, 2)
  - [ ] 审查 `service/enterprise/department_budget.go` 中基于 `latestBudget` 的类型锁定逻辑，移除“部门级单类型”限制。
  - [ ] 保留“单预算池实例创建后不可跨类型切换”的规则，不引入预算池 update / mutation 越权改型能力。
  - [ ] 确保已有错误语义 `ErrDepartmentBudgetTypeImmutable` 只用于真正的“改单池类型”场景，而不是拦截新建不同类型预算池。

- [ ] 审查下游链路在多类型并存下的行为 (AC: 1, 2, 3)
  - [ ] 核查预算委派仍按 source/target budget 的显式类型一致性校验，不因为同部门存在多池而误判。
  - [ ] 核查 quota request capability / 提交表单在同部门多池并存时，能正确列出并区分池类型。
  - [ ] 核查成员 allocation 创建、预算池列表与 detail view 在同部门多池并存时不依赖“部门单类型”隐式前提。

- [ ] 增加后端与前端测试覆盖 (AC: 1, 2, 3)
  - [ ] 补充 `service/enterprise/department_budget_test.go`，验证同部门可创建不同类型预算池。
  - [ ] 补充相关 controller / frontend tests，验证预算池选择、委派、申请和分配在多类型并存时工作正常。
  - [ ] 回归 Story 7.3 / 7.5 / 7.6 的关键路径，避免修正类型约束后破坏既有治理链路。

## Dev Notes

- PRD 已明确“同一部门可创建多个预算池（例如一个长期 balance + 一个月度 subscription）”；当前实现与该句冲突，因此这是需求实现偏差修复，不是新增需求。[Source: prd.md#FR-9]
- 当前问题根源在 `DepartmentBudgetService.Create`：它把“预算池实例不可切换”误实现成了“部门已有池类型不可变”。[Source: service/enterprise/department_budget.go]
- 预算委派与 allocation 继续可以要求“本次操作的 source/target budget 类型兼容”，但这个约束不应上升成部门级全局单类型。
- 本故事不要求补齐预算池编辑/删除能力；只修正类型锁定粒度。

### Project Structure Notes

- 主要落点预计在：
  - `service/enterprise/department_budget.go`
  - `service/enterprise/department_budget_test.go`
  - `controller/enterprise/department_budget_test.go`
  - `web/default/src/features/enterprise-organization/index.tsx`
  - `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`

### References

- [Source: /Users/hq-it/.codex/worktrees/ab4c/new-api/_bmad-output/planning-artifacts/epics.md#story-7a3-放宽预算池类型约束至单池不可变]
- [Source: /Users/hq-it/.codex/worktrees/ab4c/new-api/_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-9-创建部门预算池-v11-修订]
- [Source: /Users/hq-it/.codex/worktrees/ab4c/new-api/_bmad-output/planning-artifacts/architecture.md#预算池类型约束边界-v13a]
- [Source: /Users/hq-it/.codex/worktrees/ab4c/new-api/_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-02-enterprise-governance-followup-cc.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-02 15:20:00 +0800: 基于已批准的 Correct Course 结果创建 Story 7A.3，目标是修正预算池类型约束粒度。

### Completion Notes List

- 已明确该故事不是新增预算池类型，而是校正“同部门多池并存”的约束边界。
- 已记录和 Story 7.3 / 7.5 的耦合点，避免后续修复时破坏预算委派和员工申请流程。
- 已标记“预算池编辑/删除能力”不在本故事范围内。

### File List

- `_bmad-output/implementation-artifacts/7a-3-relax-budget-pool-type-locking-to-per-budget-invariance.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

- 2026-06-02 15:20:00 +0800: 创建 Story 7A.3 上下文文档，纳入 backlog。
