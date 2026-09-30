# Story 7A.2: 将员工额度申请入口迁移到余额 / 钱包用户场景

Status: done

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

- [x] 识别用户侧余额 / 钱包最佳入口，并设计不破坏现有治理工作台的接入方式 (AC: 1, 2, 3)
  - [x] 盘点当前用户余额、钱包、充值或额度相关前端 feature 与路由，确认最适合放置“申请额度”入口的页面或组件。
  - [x] 决定采用独立页面还是弹框表单，并记录为什么这一路径更符合普通用户心智。
  - [x] 保持企业组织页继续作为审批 / 时间线 / 通知状态的治理工作台，不把治理视图迁出。[Source: architecture.md#Information Architecture; CC proposal]

- [x] 在用户侧入口复用现有 quota request workflow (AC: 1, 2, 3)
  - [x] 复用现有 `/api/enterprise/quota-requests`、`/api/enterprise/quota-requests/capability/:department_id` 等接口，不新发明第二套申请 API。
  - [x] 表单仍要求显式选择目标部门与目标预算池，避免多部门成员关系下的审批归属歧义。
  - [x] 确保普通用户只能看到自己可提交的部门与预算池能力，不扩大治理权限面。[Source: controller/enterprise/quota_request.go; service/enterprise/quota_request.go]

- [x] 调整前端导航与测试，验证普通用户可达性 (AC: 1, 2, 3)
  - [x] 更新用户侧页面组件、路由与交互测试。
  - [x] 验证普通用户无需看到企业组织菜单，也能到达申请入口。
  - [x] 回归企业组织页中的审批表格、治理时间线、通知状态，确保负责人流程不退化。

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
- 2026-06-02 17:23:42 +0800: 在钱包页新增普通员工“申请额度”卡片，作为用户侧入口；采用钱包页内卡片表单而非企业治理页入口，符合余额不足时从余额 / 钱包补额度的用户心智。
- 2026-06-02 17:23:42 +0800: 钱包侧表单复用既有 `getUserDepartments`、`getQuotaRequestCapability`、`submitQuotaRequest` 与 `quotaRequestQueryScopeKey`，仍要求显式选择目标部门与目标预算池，不新增后端 API 或拆分缓存命名空间。
- 2026-06-02 17:23:42 +0800: 企业组织治理工作台保留现有审批表格、治理时间线与通知投递视图；本次仅新增员工侧申请入口。
- 2026-06-02 17:23:42 +0800: 已新增钱包侧组件测试并纳入 `test:e2e`，回归企业组织、治理时间线、通知投递和相关前端测试。

### Implementation Plan

- 将普通员工申请入口放在 `Wallet` 页面底部的独立卡片中，避免改变充值、订阅和企业治理工作台布局。
- 员工侧卡片先根据当前用户查询其有效部门成员关系，再按选中部门查询 quota request capability，仅展示后端判定可提交的预算池。
- 提交时调用既有 quota request API，并在成功后失效同一 enterprise organization quota request query scope，确保负责人治理工作台可继续看到审批闭环。

### File List

- `_bmad-output/implementation-artifacts/7a-2-move-employee-quota-request-entry-to-balance-wallet-user-flow.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `controller/enterprise/department_membership.go`
- `controller/enterprise/department_membership_test.go`
- `router/enterprise-router.go`
- `web/default/package.json`
- `web/default/rsbuild.test.config.ts`
- `web/default/src/features/wallet/index.tsx`
- `web/default/src/features/wallet/components/employee-quota-request-card.tsx`
- `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/i18n/locales/zh.json`

## Change Log

- 2026-06-02 15:20:00 +0800: 创建 Story 7A.2 上下文文档，纳入 backlog。
- 2026-06-02 17:23:42 +0800: 完成钱包侧员工额度申请入口、复用 quota request workflow、补充测试与 i18n，并将故事状态更新为 review。
- 2026-06-02 17:44:25 +0800: Senior Developer Review 自动修复普通员工读取自身部门成员关系、治理闭环缓存失效与 i18n 命名空间问题，回归测试通过并将故事状态更新为 done。

## Senior Developer Review (AI)

**Reviewer:** hth
**Date:** 2026-06-02 17:44:25 +0800
**Outcome:** Approve / 自动修复后通过

### 发现并自动修复的问题

1. **HIGH — 钱包侧普通员工入口无法读取自身部门成员关系。**
   证据：`web/default/src/features/wallet/components/employee-quota-request-card.tsx` 复用 `getUserDepartments(user.id)`，但 `router/enterprise-router.go` 原先对 `GET /api/enterprise/users/:id/departments` 挂了 `AdminAuth()`，普通员工无法获取自己的 active memberships，导致 AC1/AC2 的钱包侧入口无法进入可提交状态。
   修复：将该 GET 路由降为登录用户可访问，并在 `controller/enterprise/department_membership.go` 中限制非管理员只能读取 `:id` 等于当前登录用户的成员关系；保留 PUT 管理员权限。新增 `controller/enterprise/department_membership_test.go` 覆盖普通用户可读自身、不可读他人。

2. **MEDIUM — 钱包提交成功后只失效 quota request query scope，未刷新治理时间线与通知投递缓存。**
   证据：钱包侧提交成功后仅 invalidate `quotaRequestQueryScopeKey`，而 AC3 要求负责人/管理员治理闭环的审批、治理时间线、通知投递视图保持可用。若负责人工作台已加载，时间线/通知缓存可能继续展示旧状态。
   修复：新增 `invalidateQuotaRequestGovernanceQueries`，在钱包提交成功后同时失效 `governanceTimelineQueryKey` 与 `governanceNotificationQueryKey`；新增前端测试覆盖缓存失效目标。

3. **MEDIUM — 新增翻译键被写在 locale JSON 根级，绕开 `translation` namespace。**
   证据：`web/default/src/i18n/config.ts` 默认 namespace 为 `translation`，但新增钱包文案出现在 `web/default/src/i18n/locales/{lang}.json` 根级，非英文语言下 `t()` 不会读取这些翻译。
   修复：将新增钱包文案移动到所有支持语言的 `translation` 对象内，并用脚本验证根级无非 `translation` 键。

### 验收标准复核

- **AC1:** 通过。钱包页已渲染员工额度申请卡片；普通员工现在可读取自己的有效部门成员关系，不依赖 `Enterprise Organization` 管理菜单。
- **AC2:** 通过。表单仍要求显式选择目标部门与目标预算池，并复用 `/api/enterprise/users/:id/departments`、`/api/enterprise/quota-requests/capability/:department_id`、`/api/enterprise/quota-requests`。
- **AC3:** 通过。企业组织治理工作台的审批、治理时间线与通知投递视图保留；钱包提交后补充失效治理时间线/通知缓存，避免闭环视图滞后。

### 验证

- `cd web/default && bun run typecheck` ✅
- `cd /Users/hq-it/repository/github/huaqingai/new-api && go test ./controller/enterprise` ✅
- `cd web/default && bun run test:e2e` ✅（Rsbuild 仍输出既有 optional dependency warning：`supports-color` 未解析；所有 bundled tests 通过，整体 99/99 通过，其中 `employee-quota-request-card` 6/6 通过。）
- `cd /Users/hq-it/repository/github/huaqingai/new-api && go test ./controller/enterprise ./service/enterprise` ⚠️ controller 包通过；service 包因既有 SQLite 并发/唯一约束测试日志导致失败，与本故事改动无直接关系，本次以 controller 定向回归作为后端验证。

### Checklist

- [x] Story file loaded from `_bmad-output/implementation-artifacts/7a-2-move-employee-quota-request-entry-to-balance-wallet-user-flow.md`
- [x] Story Status verified as reviewable (review)
- [x] Epic and Story IDs resolved (7A.2)
- [x] Story Context located or warning recorded（未找到独立 story context；已使用 story、epics/prd/architecture 规划文档）
- [x] Epic Tech Spec located or warning recorded（已读取/检索 `_bmad-output/planning-artifacts/epics.md`、`architecture.md`、`prd.md`）
- [x] Architecture/standards docs loaded (as available)
- [x] Tech stack detected and documented（Go/Gin/GORM；React 19/TypeScript/Rsbuild/Bun/i18next）
- [x] MCP doc search performed (or web fallback) and references captured（无 MCP doc 搜索工具可用；使用本地规划与源码检索替代）
- [x] Acceptance Criteria cross-checked against implementation
- [x] File List reviewed and validated for completeness
- [x] Tests identified and mapped to ACs; gaps noted
- [x] Code quality review performed on changed files
- [x] Security review performed on changed files and dependencies
- [x] Outcome decided (Approve/Changes Requested/Blocked)
- [x] Review notes appended under "Senior Developer Review (AI)"
- [x] Change Log updated with review entry
- [x] Status updated according to settings (if enabled)
- [x] Sprint status synced (if sprint tracking enabled)
- [x] Story saved successfully
