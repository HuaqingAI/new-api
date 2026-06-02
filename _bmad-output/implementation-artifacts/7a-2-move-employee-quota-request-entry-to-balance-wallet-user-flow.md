# Story 7A.2: 将员工额度申请入口迁移到余额 / 钱包用户场景

Status: backlog

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 普通员工,
I want 在余额不足时从我的钱包 / 余额界面直接发起额度申请,
so that 我不需要进入企业治理工作台也能完成申请。

## Acceptance Criteria

1. **Given** 普通员工额度不足或需要主动申请额度  
   **When** 其查看自己的余额 / 钱包区域  
   **Then** 页面提供明确可达的“申请额度”入口  
   **And** 不依赖 `Enterprise Organization` 管理菜单。
2. **Given** 员工点击申请入口  
   **When** 打开表单页或弹框  
   **Then** 仍必须显式选择目标部门与目标预算池  
   **And** 提交后继续复用现有 quota request API / workflow。
3. **Given** 负责人或管理员进入企业组织治理工作台  
   **When** 处理申请  
   **Then** 现有审批、治理时间线和通知投递视图保持可用  
   **And** 不因入口迁移而丢失治理闭环能力。

## Tasks / Subtasks

- [ ] 识别用户侧余额 / 钱包最佳入口，并设计不破坏现有治理工作台的接入方式 (AC: 1, 2, 3)
  - [ ] 盘点当前用户余额、钱包、充值或额度相关前端 feature 与路由，确认最适合放置“申请额度”入口的页面或组件。
  - [ ] 决定采用独立页面还是弹框表单，并记录为什么这一路径更符合普通用户心智。
  - [ ] 保持企业组织页继续作为审批 / 时间线 / 通知状态的治理工作台，不把治理视图迁出。[Source: architecture.md#Information Architecture; CC proposal]

- [ ] 在用户侧入口复用现有 quota request workflow (AC: 1, 2, 3)
  - [ ] 复用现有 `/api/enterprise/quota-requests`、`/api/enterprise/quota-requests/capability/:department_id` 等接口，不新发明第二套申请 API。
  - [ ] 表单仍要求显式选择目标部门与目标预算池，避免多部门成员关系下的审批归属歧义。
  - [ ] 确保普通用户只能看到自己可提交的部门与预算池能力，不扩大治理权限面。[Source: controller/enterprise/quota_request.go; service/enterprise/quota_request.go]

- [ ] 调整前端导航与测试，验证普通用户可达性 (AC: 1, 2, 3)
  - [ ] 更新用户侧页面组件、路由与交互测试。
  - [ ] 验证普通用户无需看到企业组织菜单，也能到达申请入口。
  - [ ] 回归企业组织页中的审批表格、治理时间线、通知状态，确保负责人流程不退化。

## Dev Notes

- 这是产品入口迁移，不应改变 quota request 的后端状态机、审批权限模型或审计通知链路。[Source: service/enterprise/quota_request.go; Story 7.5 / 7.6]
- 当前企业组织页里集成了申请表与审批表；后续实现要把“员工申请入口”和“负责人审批工作台”解耦，但仍允许负责人在治理工作台查看请求列表。[Source: web/default/src/features/enterprise-organization/index.tsx]
- 需要特别注意普通用户是否已有与余额/钱包相关的既有模块，优先复用而不是新造页面骨架。

### Project Structure Notes

- 预计会修改：
  - 用户余额 / 钱包相关 feature（待开发时最终确认）
  - `web/default/src/features/enterprise-organization/index.tsx`
  - quota request 相关测试与可能的 route 文件
- 需要保持 queryKey 与 enterprise feature 命名一致，避免把 quota request 状态分裂成两套缓存命名。

### References

- [Source: /Users/hq-it/.codex/worktrees/ab4c/new-api/_bmad-output/planning-artifacts/epics.md#story-7a2-将员工额度申请入口迁移到余额--钱包用户场景]
- [Source: /Users/hq-it/.codex/worktrees/ab4c/new-api/_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#uj-6-员工额度不足时发起申请并由部门负责人审批]
- [Source: /Users/hq-it/.codex/worktrees/ab4c/new-api/_bmad-output/planning-artifacts/architecture.md#information-architecture]
- [Source: /Users/hq-it/.codex/worktrees/ab4c/new-api/_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-02-enterprise-governance-followup-cc.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-02 15:20:00 +0800: 基于已批准的 Correct Course 结果创建 Story 7A.2，目标是把员工申请入口迁移到普通用户余额 / 钱包场景。

### Completion Notes List

- 已明确该故事是产品入口重构，不重写 quota request 后端流程。
- 已保存关键约束：普通用户入口迁出，负责人审批工作台保留。
- 已标记余额 / 钱包相关前端落点需要在开发时进一步定位。

### File List

- `_bmad-output/implementation-artifacts/7a-2-move-employee-quota-request-entry-to-balance-wallet-user-flow.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

- 2026-06-02 15:20:00 +0800: 创建 Story 7A.2 上下文文档，纳入 backlog。
