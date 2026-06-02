# Story 7A.1: 补齐企业治理工作台 i18n 与动作标签映射

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员或负责人,
I want 企业治理工作台中的标题、表头和动作标签都以当前语言完整显示,
so that 我不会在治理流程中看到英文占位或内部动作标识。

## Acceptance Criteria

1. **Given** 用户以非英文语言访问企业组织页中的治理区域  
   **When** 页面渲染 `Governance Timeline and Notification Delivery`、`Employee Quota Requests`、表头、空状态与按钮文案  
   **Then** 所有文案都通过 i18n key 显示当前语言  
   **And** 不应回落成遗漏翻译的英文。
2. **Given** 治理时间线存在 `action_type` 记录  
   **When** UI 渲染 `Action` 列  
   **Then** 原始 `enterprise.organization.*` action key 必须映射成用户可读标签  
   **And** 若出现未知 action type，也要有明确兜底展示，而不是直接泄露内部 key。
3. **Given** 通知投递和审批结果列表包含状态、动作或原因字段  
   **When** 页面展示这些字段  
   **Then** 文案、状态值与按钮动作必须保持与当前 locale 一致  
   **And** 对应测试覆盖应验证中文等非英文输出。

## Tasks / Subtasks

- [ ] 盘点企业组织治理工作台中所有未覆盖或映射不完整的文案与动作标签 (AC: 1, 2, 3)
  - [ ] 检查 `web/default/src/features/enterprise-organization/index.tsx` 中治理时间线、通知投递、员工额度申请、审批按钮、空状态、表格表头等所有 `t(...)` 文案。
  - [ ] 检查 `governanceActionLabel` 与相关格式化 helper，确认所有已落地 `action_type` 都有明确 label 映射。
  - [ ] 盘点当前 locale JSON 中是否缺少 `Governance Timeline and Notification Delivery`、`Employee Quota Requests`、`Action`、`Target Budget Pool`、审批/通知相关文案。[Source: web/default/src/features/enterprise-organization/index.tsx; web/default/src/i18n/locales/*.json]

- [ ] 修复治理区域 i18n 缺口与 action label 映射 (AC: 1, 2, 3)
  - [ ] 在 `web/default/src/i18n/locales/{en,zh,fr,ja,ru,vi}.json` 补齐缺少的 enterprise governance 文案。
  - [ ] 为 `governanceActionLabel` 增加 Story 7.1 / 7.3 / 7.4 / 7.5 / 7.6 已引入 action type 的完整映射，避免 UI fallback 到内部 key。
  - [ ] 确保空状态、表头、按钮和 cell 文案一致使用英文源字符串作为 key，符合前端 i18n 规则。[Source: AGENTS.md#Internationalization (i18n)]

- [ ] 补齐回归测试并验证多语言显示 (AC: 1, 2, 3)
  - [ ] 更新 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`，增加治理区域在非英文 locale 下的断言或最小 label 覆盖。
  - [ ] 如需要，补充治理 action label helper 的单测或页面测试 fixture，覆盖未知 action type 的兜底行为。
  - [ ] 运行 `bun test` 定向测试、`bun run typecheck`、必要时 `bun run i18n:sync`，确保 locale 结构与静态 key 同步。

## Dev Notes

- 这是明确 bug 修复，不改动企业治理的业务边界，只补齐 UI 文案与 label 投影。
- 前端 i18n 规则是“英文源字符串作为 key”，而不是新增命名式 key；修复时应继续沿用现有项目做法。[Source: AGENTS.md#Internationalization (i18n)]
- `governanceActionLabel` 当前是治理时间线 action 可读性的核心入口，修复应优先集中在这里，而不是在渲染层散落 if/else。[Source: web/default/src/features/enterprise-organization/index.tsx]
- 不要修改后端 `action_type` 常量本身，避免破坏 Story 7.6 已形成的审计事实和通知追溯链路。[Source: service/enterprise/admin_action.go; model/enterprise/governance_notification_delivery.go]

### Project Structure Notes

- 主要落点预计在：
  - `web/default/src/features/enterprise-organization/index.tsx`
  - `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - `web/default/src/i18n/locales/*.json`
- 若需要拆 helper，优先放在 `web/default/src/features/enterprise-organization/` 范围内，避免污染通用层。

### References

- [Source: /Users/hq-it/.codex/worktrees/ab4c/new-api/_bmad-output/planning-artifacts/epics.md#story-7a1-补齐企业治理工作台-i18n-与动作标签映射]
- [Source: /Users/hq-it/.codex/worktrees/ab4c/new-api/_bmad-output/planning-artifacts/architecture.md#information-architecture]
- [Source: /Users/hq-it/.codex/worktrees/ab4c/new-api/_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-02-enterprise-governance-followup-cc.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-02 15:20:00 +0800: 基于已批准的 Correct Course 结果创建 Story 7A.1，上下文聚焦企业治理工作台 i18n 与 action label 漏映射问题。

### Completion Notes List

- 已把问题定性为明确 bug，而不是需求变更。
- 已明确不能修改后端 action type 事实源，只能改前端投影和 locale。
- 已把测试范围限定在 enterprise-organization feature 和 locale 同步流程。

### File List

- `_bmad-output/implementation-artifacts/7a-1-complete-enterprise-governance-i18n-and-action-label-mapping.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

- 2026-06-02 15:20:00 +0800: 创建 Story 7A.1 上下文文档，准备进入开发。
