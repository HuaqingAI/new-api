---
baseline_commit: 287a4f69f1d7ceb9fc0c5a0c08ff73a7bc7e95d5
---

# Story 7A.1: 补齐企业治理工作台 i18n 与动作标签映射

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员或负责人,
I want 企业治理工作台中的标题、表头、状态值和动作标签都以当前语言完整显示,
so that 我不会在治理流程中看到英文占位、内部 action key 或难以理解的系统标识。

## Acceptance Criteria

1. **Given** 用户以非英文语言访问企业组织页中的治理区域
   **When** 页面渲染治理时间线、通知投递、员工额度申请区域、表头、空状态、按钮和状态文案
   **Then** 所有文案都通过 `t(...)` 和 locale JSON 显示当前语言
   **And** 不出现遗漏翻译的英文源字符串。
2. **Given** 治理时间线存在已知 `action_type` 记录
   **When** UI 渲染 `Action` 列或通知投递动作摘要
   **Then** `action_type` 必须映射成用户可读标签
   **And** 不直接显示 `enterprise.organization.*`、`enterprise.dingtalk.*`、`enterprise.alert.*` 等内部 key。
3. **Given** 治理时间线或通知投递中出现未知 `action_type`
   **When** 页面展示 action 列
   **Then** UI 必须展示明确兜底文案（例如“Unknown governance action” + 安全截断/调试辅助信息）
   **And** 不能把完整内部 key 作为主要用户可见标签。
4. **Given** 通知投递和审批结果列表包含状态、动作、失败原因、重发按钮或空状态
   **When** 页面在 `zh` 等非英文 locale 下渲染
   **Then** 状态值与按钮动作必须与当前 locale 一致
   **And** 测试覆盖至少一个非英文 locale 的治理区域输出。

## Tasks / Subtasks

- [x] 盘点企业组织治理工作台的 i18n 和 action label 缺口 (AC: 1, 2, 3, 4)
  - [x] 检查 `web/default/src/features/enterprise-organization/index.tsx` 中治理时间线、通知投递、员工额度申请、审批按钮、空状态、表格表头、状态 badge、toast 文案是否全部走 `t(...)`。
  - [x] 检查 `governanceActionLabel`、`governanceDeliveryStatusLabel`、`enterpriseBudgetStatusLabel` 以及 quota request / allocation / delegation 表格中所有状态和动作文案，确认不会直接回退到后端 key 或裸状态字符串。
  - [x] 以 `service/enterprise/admin_action.go` 和 `service/enterprise/governance_notification.go` 的常量为事实源，列出当前 UI 可能收到的全部治理相关 action type；不要猜测或改写后端常量。
  - [x] 盘点 `web/default/src/i18n/locales/{en,zh,fr,ja,ru,vi}.json` 是否包含治理区域使用的英文源字符串，重点包括 `Governance Timeline and Notification Delivery`、`Employee Quota Requests`、`Target Budget Pool`、`Trace ID`、`Action`、`Notification Status`、`Final failed`、`Resend`、所有 action label 和未知动作兜底文案。

- [x] 补齐 action label 映射，并为未知 action type 提供安全兜底 (AC: 2, 3)
  - [x] 在 `governanceActionLabel` 中补齐已知治理动作映射，至少覆盖：
    - `enterprise.organization.quota_request.submit` → `Quota request submitted`
    - `enterprise.organization.quota_request.approve` → `Quota request approved`
    - `enterprise.organization.quota_request.reject` → `Quota request rejected`
    - `enterprise.organization.quota_allocation.create` → `Allocation created`
    - `enterprise.organization.quota_allocation.reclaim` → `Allocation reclaimed`
    - `enterprise.organization.quota_allocation.cancel` → `Allocation cancelled`
    - `enterprise.organization.quota_allocation.revoke` → `Allocation revoked`
    - `enterprise.organization.budget_delegation.create` → `Budget delegation created`
    - `enterprise.organization.budget_delegation.supersede` → `Budget delegation adjusted`
    - `enterprise.organization.budget_delegation.revoke` → `Budget delegation revoked`
    - `enterprise.organization.budget_delegation.reject` → `Budget delegation rejected`
    - `enterprise.organization.department_budget.create` → `Department budget created`
    - `enterprise.organization.department_budget.reject` → `Department budget rejected`
    - 如 timeline 会投影负责人授权、成员维护、钉钉同步或告警重发，也要在同一 helper 或等价集中 helper 中覆盖对应常量。
  - [x] 未知 action type 不再 `return t(actionType)`；改为稳定兜底，例如 `t('Unknown governance action')`，必要时在次级 text / tooltip 展示经过截断的 action suffix，避免主列泄露完整内部 key。
  - [x] 保持映射集中在 helper 中，不在 JSX 表格 cell 内散落 `if/else`，防止后续 action 新增时再次漏翻译。

- [x] 补齐 Default 前端 locale JSON (AC: 1, 2, 3, 4)
  - [x] 在 `web/default/src/i18n/locales/en.json` 保持英文源字符串 key 与英文值一致或项目既有格式一致。
  - [x] 在 `zh.json` 补齐自然中文翻译，确保治理工作台不再显示英文标题、表头、空状态、按钮和动作标签。
  - [x] 在 `fr.json`、`ja.json`、`ru.json`、`vi.json` 同步新增相同 key；可使用简洁可理解翻译，禁止缺 key 导致 fallback 到英文。
  - [x] 如运行 `bun run i18n:sync` 会重排 locale，确认只接受与本故事相关的 locale 变更，不引入无关文案删除。

- [x] 增加回归测试锁定非英文输出和未知 action 兜底 (AC: 1, 2, 3, 4)
  - [x] 更新 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`，新增治理区域在 `zh` locale 下渲染的断言，覆盖标题、`Action` 表头、`Notification Status` 表头、状态 badge、`Resend` 按钮和空状态。
  - [x] 添加包含多个已知 action type 的 timeline fixture，断言每个 action 都显示可读标签，而不是 `enterprise.organization.*` 内部 key。
  - [x] 添加未知 action type fixture，断言主 label 为兜底文案，并显式 `doesNotMatch` 完整内部 key。
  - [x] 保留现有英文 locale 测试，确保英文源字符串作为 key 的项目约定不被破坏。

- [x] 执行验证并记录结果 (AC: 1, 2, 3, 4)
  - [x] 在 `web/default/` 下运行定向测试：`bun test src/features/enterprise-organization/enterprise-organization.test.tsx`。
  - [x] 运行 `bun run typecheck`。
  - [x] 如修改 locale 同步或新增大量 key，运行 `bun run i18n:sync` 并检查 diff。
  - [x] 不需要运行 Go 测试，除非实现过程中误触后端文件；本故事预期为 Default 前端和 locale 修复。

## Dev Notes

### 目标与边界

- 这是 Epic 7 完成后的体验纠偏 / bug 修复，核心是“前端投影与翻译完整性”，不是重新定义治理事实、审批流或通知投递模型。[Source: _bmad-output/planning-artifacts/epics.md:1141-1164]
- 后端 `action_type` 是治理审计和通知追溯事实的一部分，不要为了显示中文而修改后端常量、payload、DB 记录或 API contract。[Source: service/enterprise/admin_action.go:12-43; service/enterprise/governance_notification.go:15-23]
- Story 7.6 已建立治理时间线和治理通知 delivery；7A.1 只修 UI 文案 / label 映射，必须保持 7.6 的事实源和幂等通知语义。[Source: _bmad-output/implementation-artifacts/7-6-audit-governance-actions-and-send-dingtalk-notifications.md:76-86]
- 普通员工额度申请入口迁移属于 Story 7A.2；本故事只能补齐企业组织工作台内现有申请/审批区域文案，不要把入口迁移到余额 / 钱包页面。[Source: _bmad-output/planning-artifacts/epics.md:1165-1188]
- 预算池类型约束修正属于 Story 7A.3；本故事不要修改预算池创建、类型校验或 allocation service。[Source: _bmad-output/planning-artifacts/epics.md:1190-1213]

### 当前实现状态（已读 UPDATE 文件）

- `web/default/src/features/enterprise-organization/index.tsx` 是主要修改点。当前 `GovernanceActivityCard` 用 `governanceActionLabel(item.action_type, t)` 渲染 action 列，`GovernanceNotificationDeliveryList` 渲染通知投递状态和重发按钮。[Source: web/default/src/features/enterprise-organization/index.tsx:3492-3609]
- 当前 `governanceActionLabel` 已覆盖 quota request、quota allocation、budget delegation create/supersede，但未知 action 会 `return t(labels[actionType] ?? actionType)`，如果 locale 缺 key，就会显示完整内部 key；这是本故事必须修复的主要风险。[Source: web/default/src/features/enterprise-organization/index.tsx:3685-3701]
- `governanceDeliveryStatusLabel` 对 `pending/sent/resent/failed/final_failed` 有映射，但 default 分支返回原始 status；若新增未知状态，可能裸露后端状态字符串。若本故事范围允许，应同步改为安全兜底。[Source: web/default/src/features/enterprise-organization/index.tsx:3703-3721]
- `QuotaRequestTable` 已使用 `Target Budget Pool`、`Requested Quota`、`Approved Quota`、`Approve`、`Reject` 等英文源字符串；这些 key 必须存在于所有 Default locale。[Source: web/default/src/features/enterprise-organization/index.tsx:3737-3899]
- `enterprise-organization.test.tsx` 已有英文渲染测试，覆盖 `Governance Timeline and Notification Delivery`、`Quota request approved`、`Final failed`、`Resend`，但缺少非英文 locale、完整 action 矩阵和未知 action 兜底测试。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx:506-545]
- `api.ts` / `types.ts` 只传递 `action_type: string`，无需变更 API 类型；action label 应保持前端 helper 负责。[Source: web/default/src/features/enterprise-organization/api.ts; web/default/src/features/enterprise-organization/types.ts]

### 后端 action type 事实源

- `service/enterprise/governance_notification.go` 的治理通知动作常量覆盖 quota request submit/approve/reject 与 allocation create/reclaim/cancel/revoke。[Source: service/enterprise/governance_notification.go:15-23]
- `service/enterprise/admin_action.go` 的 admin action 常量还包括成员维护、部门管理员/负责人授权、钉钉配置/同步、usage report、department budget、budget delegation、alert rule / delivery resend 等；timeline projection 若聚合这些事实，前端需要可读标签或安全兜底。[Source: service/enterprise/admin_action.go:12-43]
- 已确认 7A.1 预期不修改 `service/enterprise/admin_action.go`、`service/enterprise/governance_notification.go`、`model/enterprise/**`、`controller/enterprise/**` 或 `router/enterprise-router.go`；这些文件只作为事实源引用。

### i18n 规则与前端约束

- Default 前端使用 `i18next`，locale 文件为 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，翻译 key 是英文源字符串；新增前端文案必须进入这些 JSON。[Source: CLAUDE.md#Internationalization (i18n); _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md:519-529]
- 不要把 Default 前端改成命名空间式 key（例如 `enterprise.governance.action.approve`）；本项目 Default 主题约定是 `t('English source string')`。[Source: CLAUDE.md#Internationalization (i18n)]
- 如新增未知动作兜底文案，推荐英文源字符串如 `Unknown governance action`、`Unknown delivery status`、`Internal action suffix: {{suffix}}`，并同步所有 locale。
- 翻译补齐应局限在企业治理相关 key。不要删除或重写既有 unrelated locale key，避免影响其它页面。

### 架构与代码边界

- Default 前端是企业能力完整交付面，使用 React 19、TypeScript、Rsbuild、TanStack Query、Base UI、Tailwind；本故事不新增依赖。[Source: _bmad-output/planning-artifacts/architecture.md:130-135; _bmad-output/planning-artifacts/architecture.md:363-397]
- 企业组织 feature 位于 `web/default/src/features/enterprise-organization/`；若拆 helper，优先留在该 feature 内，不要污染全局 `lib/`。[Source: _bmad-output/planning-artifacts/architecture.md:601-615]
- TanStack queryKey 继续保持 `['enterprise', 'organization', ...]` 命名空间；本故事不应改变 queryKey 或 mutation invalidation 策略。[Source: _bmad-output/planning-artifacts/architecture.md:700-720]
- 本故事不得触碰 `relay/**`、`pkg/billingexpr/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json` 或 `relay/channel/**`。[Source: _bmad-output/planning-artifacts/architecture.md:745-760]
- 所有 JSON / DB / Go 三库兼容规则仍然有效，但本故事预期无需新增 Go JSON 或 DB 代码。[Source: CLAUDE.md#Rules]

### Previous Story Intelligence

- Story 7.6 完成了治理时间线投影、治理通知 delivery / dispatch / API、Default 企业组织页治理活动视图和 i18n；验证中 `web/default` 定向测试与 typecheck 通过，但 service 全包测试受 sandbox / 既有断言影响未全量通过。[Source: _bmad-output/implementation-artifacts/7-6-audit-governance-actions-and-send-dingtalk-notifications.md:164-174]
- 7.6 明确的防误点继续适用：不要新建万能审计表复制所有事实；不要把治理通知塞进风险告警专用 delivery；不要让通知失败回滚核心治理事务；不要复制直接 `encoding/json` marshal 模式。[Source: _bmad-output/implementation-artifacts/7-6-audit-governance-actions-and-send-dingtalk-notifications.md:187-196]
- 7.6 文件清单显示本故事可能触达的前端文件已经在上一故事中被修改：`web/default/src/features/enterprise-organization/{api.ts,types.ts,index.tsx,enterprise-organization.test.tsx}` 与所有 Default locale JSON。[Source: _bmad-output/implementation-artifacts/7-6-audit-governance-actions-and-send-dingtalk-notifications.md:222-231]

### Git Intelligence Summary

- 最近提交包括 BMAD 技能升级、Epic 7A follow-up planning artifacts、Epic 7 orchestration/retrospective 修复；说明当前任务来源是 Epic 7 完成后的规划纠偏，不是新的企业能力大改。[Source: git log --oneline -5]
- 当前工作树已有 BMAD/story-automator 与 planning artifacts 修改；实现本故事时不要回滚无关变更，尤其不要清理 `_bmad-output/story-automator/**` 或 `.claude/skills/**` 中与本故事无关的文件。[Source: initial gitStatus]

### Latest Technical Information

- 本故事不需要引入或升级外部库；继续使用仓库已固定的 React 19、TanStack Query、i18next、Base UI 与 Bun 工具链即可。[Source: CLAUDE.md#Tech Stack; _bmad-output/planning-artifacts/architecture.md:130-135]
- 因为无新增依赖、无浏览器 API 变更、无外部服务 API，本次未把 Web research 结果写入技术要求；实现者应优先遵循仓库当前 package versions 和 existing tests。

### Project Structure Notes

- 预计修改文件：
  - `web/default/src/features/enterprise-organization/index.tsx`
  - `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - `web/default/src/i18n/locales/en.json`
  - `web/default/src/i18n/locales/zh.json`
  - `web/default/src/i18n/locales/fr.json`
  - `web/default/src/i18n/locales/ja.json`
  - `web/default/src/i18n/locales/ru.json`
  - `web/default/src/i18n/locales/vi.json`
- 只读参考文件：
  - `service/enterprise/admin_action.go`
  - `service/enterprise/governance_notification.go`
  - `_bmad-output/planning-artifacts/epics.md`
  - `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`
  - `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md`
  - `_bmad-output/planning-artifacts/architecture.md`
  - `_bmad-output/implementation-artifacts/7-6-audit-governance-actions-and-send-dingtalk-notifications.md`
- 明确不改：
  - `relay/**`
  - `pkg/billingexpr/**`
  - `model/log.go`
  - `controller/log.go`
  - `router/relay-router.go`
  - `docs/openapi/relay.json`
  - `relay/channel/**`
  - 后端 action type 常量和数据库事实表

### References

- [Source: _bmad-output/planning-artifacts/epics.md:1141-1164]
- [Source: _bmad-output/planning-artifacts/epics.md:1165-1213]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md:446-465]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md:519-529]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md:240-261]
- [Source: _bmad-output/planning-artifacts/architecture.md:363-397]
- [Source: _bmad-output/planning-artifacts/architecture.md:601-615]
- [Source: _bmad-output/planning-artifacts/architecture.md:700-720]
- [Source: _bmad-output/planning-artifacts/architecture.md:745-760]
- [Source: _bmad-output/implementation-artifacts/7-6-audit-governance-actions-and-send-dingtalk-notifications.md:76-86]
- [Source: _bmad-output/implementation-artifacts/7-6-audit-governance-actions-and-send-dingtalk-notifications.md:164-174]
- [Source: service/enterprise/admin_action.go:12-43]
- [Source: service/enterprise/governance_notification.go:15-23]
- [Source: web/default/src/features/enterprise-organization/index.tsx:3492-3721]
- [Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx:506-545]
- [Source: CLAUDE.md#Internationalization (i18n)]
- [Source: CLAUDE.md#Rules]

## Senior Developer Review (AI)

_Reviewer: hth on 2026-06-02_

### Outcome

Approve / Done. 复核 Story 7A.1 的 AC、任务、文件清单与实际 git 变更后，已自动修复发现的 HIGH/MEDIUM 问题，无剩余 CRITICAL issue。

### Findings Fixed

- **HIGH** 通知投递独立记录没有展示动作摘要，AC2 对“通知投递动作摘要”要求未完全满足；已在 `GovernanceNotificationDeliveryList` 中展示 `governanceActionLabel(item.action_type, t)`，避免 delivery-only 记录裸露或缺失 action label。
- **HIGH** timeline 外通知筛选使用 `notificationsByTrace.has(item.trace_id)`，该 map 由通知自身构建，导致没有 timeline 的通知记录永远不会渲染；已改为基于 `timelineTraceIds` 判断，空 timeline 或孤立 delivery 现在会正确显示。
- **MEDIUM** `Attempt {{count}}/{{max}}` 使用了 `t(...)` 但 locale JSON 未覆盖，非英文下会 fallback 英文；已补齐 `en/zh/fr/ja/ru/vi` 翻译，并加入多语言 key 覆盖测试。

### Validation

- `bun test src/features/enterprise-organization/enterprise-organization.test.tsx` — 44 pass / 0 fail
- `bun run typecheck` — passed (`tsc -b`)
- `bun run i18n:sync` — passed and regenerated sync report

### Checklist Summary

- Story file loaded and status verified as reviewable.
- AC1-AC4 cross-checked against implementation.
- File List compared with git changes; story-owned source/locale/test files reviewed.
- Tests mapped to ACs; missing delivery-only case added.
- Code quality and security review completed on changed source files.
- Story status updated to `done`; sprint status synced.



### Agent Model Used

GPT-5.5

### Debug Log References

- 2026-06-02 16:50:16 +0800: 执行 `bmad-dev-story` workflow；读取 skill、checklist、config、sprint status 与 Story 7A.1，并设置 baseline commit `287a4f69f1d7ceb9fc0c5a0c08ff73a7bc7e95d5`。
- 2026-06-02 16:50:16 +0800: 以 `service/enterprise/admin_action.go` 与 `service/enterprise/governance_notification.go` 为事实源盘点治理 action type；确认本次仅修改 Default 前端与 locale，不修改后端事实源。
- 2026-06-02 16:50:16 +0800: 先新增治理 i18n/action label 回归测试并确认失败；随后补齐 `governanceActionLabel` 已知动作映射、未知 action 兜底、未知 delivery status 兜底。
- 2026-06-02 16:50:16 +0800: 补齐 `en/zh/fr/ja/ru/vi` Default locale 中治理工作台相关 key，并运行 `bun run i18n:sync` 生成同步报告。
- 2026-06-02 16:50:16 +0800: 验证通过：`bun test src/features/enterprise-organization/enterprise-organization.test.tsx`（43 pass）、`bun run typecheck`（tsc -b 通过）。
- 2026-06-02: 执行 `bmad-story-automator-review` workflow；读取 review skill、workflow、instructions、checklist，复核 Story 7A.1 的 AC、文件清单、git 变更和实现。
- 2026-06-02: Review 发现通知投递独立记录未展示 action 摘要、timeline 外通知筛选条件错误、`Attempt {{count}}/{{max}}` locale 缺 key；已自动修复并补充回归测试。
- 2026-06-02: Review 验证通过：`bun test src/features/enterprise-organization/enterprise-organization.test.tsx`（44 pass）、`bun run typecheck`（tsc -b 通过）、`bun run i18n:sync`。
- 2026-06-02 16:40:19 +0800: 按用户要求执行 `bmad-create-story` workflow；读取 `.claude/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`。
- 2026-06-02 16:40:19 +0800: 已执行 `resolve_customization.py`，workflow 配置解析成功：无 prepend / append steps，persistent facts 为 `file:{project-root}/**/project-context.md`，仓库内未发现匹配文件。
- 2026-06-02 16:40:19 +0800: 已读取 `_bmad/bmm/config.yaml`，确认 `project_name=new-api`、`user_name=hth`、`communication_language=Chinese`、`document_output_language=Chinese`、`implementation_artifacts=_bmad-output/implementation-artifacts`。
- 2026-06-02 16:40:19 +0800: 已完整读取 sprint status，Story 7A.1 canonical story key 为 `7a-1-complete-enterprise-governance-i18n-and-action-label-mapping`，当前已是 `ready-for-dev`；本次保留该状态并补强 story context。
- 2026-06-02 16:40:19 +0800: 已执行 discover-inputs：加载 epics、architecture、PRD shard、addendum、UX（仅 agent-platform UX，与本故事不直接相关）。
- 2026-06-02 16:40:19 +0800: 已读取前序 Story 7.6 并提取治理时间线 / 通知 delivery / Default 前端接入 / 测试验证 / 防误点。
- 2026-06-02 16:40:19 +0800: 已读取实现相关 UPDATE 文件：`web/default/src/features/enterprise-organization/index.tsx`、`enterprise-organization.test.tsx`、`service/enterprise/admin_action.go`、`service/enterprise/governance_notification.go`；另由探索代理检查 `api.ts`、`types.ts` 和 locale JSON 覆盖风险。
- 2026-06-02 16:40:19 +0800: 已按 checklist 自检，补强了 action type 全量映射、未知 action 兜底、非英文 locale 测试、前端-only 边界和不修改后端事实源要求。

### Completion Notes List

- 已完成企业治理工作台 i18n 与 action label 缺口盘点，并按后端常量事实源补齐 UI 可读标签。
- 已扩展 `governanceActionLabel` 覆盖 quota request、quota allocation、budget delegation、department budget、成员维护、负责人/管理员授权、钉钉同步、usage report、alert rule / delivery resend 等治理动作；未知 action 现在显示 `Unknown governance action`，不再把完整内部 key 作为主 label。
- 已将未知通知投递状态改为 `Unknown delivery status` 兜底，避免直接裸露后端状态字符串。
- 已补齐 Default 前端 `en/zh/fr/ja/ru/vi` locale 中治理标题、表头、状态、按钮、空状态与 action label key；中文治理区域渲染测试覆盖标题、Action、Notification Status、Final failed、Resend 和未知兜底。
- 已新增 action 矩阵、未知 action 兜底、非英文 locale 与多语言 key 覆盖测试；定向测试与 typecheck 均通过。
- Review 自动修复通知投递列表：独立通知记录现在展示 action 摘要并使用集中 `governanceActionLabel` 映射；timeline 外通知筛选改为基于 timeline trace id 集合，避免空 timeline 时误吞投递记录。
- Review 补齐 `Attempt {{count}}/{{max}}` 在 `en/zh/fr/ja/ru/vi` locale 中的翻译，并纳入多语言 key 覆盖测试。
- Review 后 `bun test src/features/enterprise-organization/enterprise-organization.test.tsx`、`bun run typecheck`、`bun run i18n:sync` 均通过，Story 状态更新为 `done`。
- 已将 Story 7A.1 从简版上下文扩展为完整开发指导，状态保持 `ready-for-dev`。
- 已明确本故事是 Default 前端 i18n/action label 修复，不修改后端 action type、数据库事实、治理 workflow 或通知投递语义。
- 已列出必须覆盖的 action type、locale JSON、测试断言和验证命令。
- 已记录前序 Story 7.6 的事实源边界和可复用实现上下文。

### File List

- `_bmad-output/implementation-artifacts/7a-1-complete-enterprise-governance-i18n-and-action-label-mapping.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/i18n/locales/_reports/_sync-report.json`
- `web/default/src/i18n/locales/_reports/ja.untranslated.json`
- `web/default/src/i18n/locales/_reports/ru.untranslated.json`

## Change Log

- 2026-06-02: 完成 Story 7A.1 story-automator review 自动修复；修复独立通知投递 action 摘要与筛选逻辑，补齐 Attempt locale key，新增回归测试并将状态更新为 `done`。
- 2026-06-02 16:50:16 +0800: 完成 Story 7A.1 dev-story 实现，补齐治理 action label 映射、安全兜底、多语言 locale 与回归测试；状态更新为 `review`。
- 2026-06-02 16:40:19 +0800: 重新执行 BMAD create-story workflow，补强 Story 7A.1 为完整开发上下文文档并保持 `ready-for-dev`。
