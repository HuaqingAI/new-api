---
baseline_commit: cf67bde3b1dcb544d7994e76672f5a2c1fe3262e
---

# Story 7B.3: 完成额度申请与治理通知相关 i18n 收尾

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 非英文语言环境下的管理员或员工,
I want quota request、allocation、delivery 状态、动作标签和表头完整本地化,
so that 我不会看到英文残留或回退到内部 key。

## Acceptance Criteria

1. **Given** 用户查看 quota request、allocation、timeline、delivery 和 wallet 相关界面
   **When** 当前语言不是英文
   **Then** 截图可见字段全部存在 locale 资源
   **And** action/status/delivery 文案都通过稳定映射层输出，不依赖遗漏的英文 source key。

## Tasks / Subtasks

- [x] 审计 Default 前端企业治理可见文案和状态输出面 (AC: 1)
  - [x] 扫描 `web/default/src/features/enterprise-organization/**`、`web/default/src/features/wallet/**` 中 quota request、allocation、timeline、delivery、wallet 相关 `t(...)`、状态映射和直接字符串输出。
  - [x] 建立本故事需要覆盖的可见字段清单：表头、按钮、空态、toast、form label、placeholder、action label、request/allocation/wallet status、delivery status、budget type、cycle/source/fallback 文案。
  - [x] 明确哪些英文品牌/专有名词可以保持原样（如 DingTalk、Budget #id、Trace ID），其余截图可见英文残留必须进入 locale。

- [x] 收敛 quota request / allocation / wallet 状态映射，不再裸露内部 status (AC: 1)
  - [x] 调整 `enterpriseBudgetStatusLabel` 或拆分更明确的 helper，使 `submitted`、`approved`、`rejected`、`fulfilled`、`active`、`paused`、`revoked`、`expired`、`superseded`、`closed`、`cancelled` 等状态都有稳定本地化 label。
  - [x] 默认分支不能返回裸 `status`；未知状态应返回本地化的未知状态文案，并可在辅助位置显示 code 用于排障，不能把内部 code 当作用户主文案。
  - [x] `QuotaRequestTable` 的 request status、`QuotaAllocationTable` 的 allocation status、`DepartmentBudgetDetailTable` 的 wallet/allocation status、预算池选择 helper 的 status label 必须复用同一稳定映射或同等覆盖范围。
  - [x] 保持状态含义不变，不修改 quota request submit/approve/reject、allocation adjust/cancel/reclaim、wallet lifecycle 后端状态机。

- [x] 补齐 governance action / delivery 映射与 locale 覆盖 (AC: 1)
  - [x] 检查 `governanceActionLabel` 的 action map，覆盖当前后端会产生的 quota request、quota allocation、budget delegation、department budget、alert delivery action type。
  - [x] 检查 `governanceDeliveryStatusLabel`，覆盖 `pending`、`sent`、`resent`、`failed`、`final_failed`、`unconfigured`，并保留 7B.1 的语义：`unconfigured` 是非阻塞/未配置状态，不显示为业务失败，不提供 resend。
  - [x] 若新增 fallback 文案，使用稳定英文源字符串进入 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，不要把 `enterprise.organization.*` action key 直接展示给用户。
  - [x] 错误原因 `error_reason` 可作为后端返回的诊断文本展示，但其 label、容器标题和状态都必须本地化；不要把通知投递错误覆盖到治理动作结果。

- [x] 补齐 quota request、allocation、delivery、wallet 表格和摘要的可见文案 (AC: 1)
  - [x] `QuotaRequestTable` 的 `Requester`、`Target Department`、`Target Budget Pool`、`Requested Quota`、`Approved Quota`、`Fulfillment`、`Status`、`Actions` 及 approve/reject 输入提示必须确认在所有 locale 存在。
  - [x] `QuotaRequestTable` 目标预算池不要只显示裸 `#id`；优先复用 7B.2 的预算池展示 helper 能拿到的信息，无法拿到完整对象时也使用本地化身份文案 `Budget #{{budgetId}}`。
  - [x] `QuotaAllocationTable` 和 `DepartmentBudgetDetailTable` 的 allocation、wallet、source、cycle/expiry、processed-at 等表头和状态必须确认 locale 覆盖。
  - [x] 钱包侧 `EmployeeQuotaRequestCard` 与 `SubscriptionPlansCard` 中企业派生 wallet 的 title、managed note、billing preference、active/expired/cancelled 文案必须确认 locale 覆盖；不得引入硬编码英文说明。

- [x] 同步 Default 前端 i18n 资源 (AC: 1)
  - [x] 更新 `web/default/src/i18n/locales/en.json` 作为英文源，并同步 `zh`、`fr`、`ru`、`ja`、`vi`。
  - [x] 使用 flat JSON 结构，key 继续是英文源字符串；不要引入新 namespace。
  - [x] 运行 `cd web/default && bun run i18n:sync`，确认新增 key 排序和缺失检测稳定。
  - [x] 保持受保护 `new-api`、`QuantumNous`、版权和品牌相关 key 不被删除、改名或替换。

- [x] 扩展回归测试，验证非英文 locale 不回退内部 key (AC: 1)
  - [x] 扩展 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`：切换到 `zh` 或另一个非英文语言后渲染 quota request、quota allocation、governance timeline、delivery list，断言常见 action/status/delivery 文案已本地化，且不出现 `enterprise.organization.quota_request.submit`、`final_failed`、`unconfigured`、`submitted` 等裸内部值。
  - [x] 扩展 `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx`：验证钱包 quota request 表单和预算池 option/summary 在非英文语言下使用 locale 文案。
  - [x] 扩展 `web/default/src/features/wallet/components/subscription-plans-card.test.tsx`：验证企业派生 wallet title/note/status 走 `t(...)`，并避免新增英文硬编码。
  - [x] 如添加独立 helper，优先导出纯函数并用 Node test 覆盖映射矩阵。

- [x] 运行并记录验证 (AC: 1)
  - [x] `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx`
  - [x] `cd web/default && bun test src/features/wallet/components/employee-quota-request-card.test.tsx`
  - [x] `cd web/default && bun test src/features/wallet/components/subscription-plans-card.test.tsx`
  - [x] `cd web/default && bun run typecheck`
  - [x] `cd web/default && bun run i18n:sync`
  - [x] `cd web/default && bun run test:e2e`（如耗时不可接受，记录未运行原因）

## Dev Notes

### 需求与业务上下文

- Epic 7B 是 Epic 7 / 7A 完成后的企业治理 follow-up，修正通知语义耦合、预算池可辨识性、i18n 残留、一次性 wallet 展示语义和金额视角；本故事只处理额度申请、治理通知和 wallet 相关可见文案的 i18n 收尾。[Source: `_bmad-output/planning-artifacts/epics.md`#Epic 7B]
- Story 7B.3 的验收明确要求非英文语言下 quota request、allocation、timeline、delivery、wallet 相关截图可见字段全部存在 locale 资源，action/status/delivery 通过稳定映射层输出，不依赖遗漏英文 source key。[Source: `_bmad-output/planning-artifacts/epics.md`#Story 7B.3]
- 2026-06-03 correct-course 将成功标准写明为“本次截图可见的 quota request / allocation / delivery 英文残留全部完成 i18n”。实现时要以用户可见面为准，不只跑静态 sync。[Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Success Criteria]
- 架构要求 Default 前端新增文案必须进入 i18n，英文 key 为源字符串；Default i18n 使用 flat JSON，不引入命名空间前缀。[Source: `_bmad-output/planning-artifacts/architecture.md`#Internationalization; `AGENTS.md`#Internationalization]
- 治理通知投递是核心治理事务之后的旁路动作；无通知通道/未配置必须展示为非阻塞状态，不能把审批成功或 allocation 创建成功呈现为失败。[Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Change M]

### 当前实现状态（已读 UPDATE 文件）

- `web/default/src/features/enterprise-organization/quota-request-budget-display.ts`
  - 7B.2 新增预算池展示 helper，当前导出 `enterpriseBudgetStatusLabel`、`formatBudgetType`、`getQuotaRequestBudgetDisplayText`。
  - 当前 `enterpriseBudgetStatusLabel` 覆盖 `active`、`paused`、`revoked`、`expired`、`superseded`、`closed`、`cancelled`，但默认分支返回裸 `status || '-'`。7B.3 必须修复这个默认回退，且覆盖 quota request 的 `submitted`、`approved`、`rejected`、`fulfilled` 等状态。
  - `formatBudgetType` 只区分 `balance` 和其他；如果后端出现未知 budget type，当前会显示 `Subscription Budget`。若触碰该 helper，应避免未知类型被误标为 subscription。

- `web/default/src/features/enterprise-organization/quota-request-budget-display-components.tsx`
  - 7B.2 新增 `QuotaRequestBudgetOption` 和 `QuotaRequestBudgetSummary`，钱包入口和企业组织工作台共同使用。
  - 组件依赖 `getQuotaRequestBudgetDisplayText` 的本地化输出。修复 status/type fallback 时要保证这两个组件同步受益，不要在两个入口重复写映射。

- `web/default/src/features/enterprise-organization/index.tsx`
  - `QuotaRequestTable` 已有表头和按钮使用 `t(...)`，但 status 通过 `enterpriseBudgetStatusLabel(item.status, t)`，当前 helper 对 request status 覆盖不足。
  - `QuotaRequestTable` 的 Target Budget Pool 当前只输出 `#{item.department_budget_id}`；7B.3 至少应改为 `t('Budget #{{budgetId}}', { budgetId })`，如能从现有预算池列表传入详情则可复用 7B.2 summary，但不要新增后端接口。
  - `GovernanceActivityCard` 使用 `governanceActionLabel` 和 `governanceDeliveryStatusLabel`，已覆盖 7B.1 的 `unconfigured` 显示为 `Not configured`。必须保留 `unconfigured` neutral/grey 样式和不显示 resend 的行为。
  - `GovernanceNotificationDeliveryList` 只有 `final_failed` 显示 resend；7B.3 不应改变 delivery 重试语义。
  - `DepartmentBudgetDetailTable` 的 wallet/allocation status 也走 `enterpriseBudgetStatusLabel`，会受 helper fallback 修复影响。

- `web/default/src/features/wallet/components/employee-quota-request-card.tsx`
  - 钱包额度申请入口已经复用企业组织的 quota request API、query key 和 7B.2 预算池展示组件。
  - 当前可见文案均大体走 `t(...)`，但新增或变更文案必须同步所有 locale。
  - 保留部门切换时清空 `department_budget_id` 的行为；本故事不改变 quota request 提交流程。

- `web/default/src/features/wallet/components/subscription-plans-card.tsx`
  - 企业派生 wallet title 使用 `getSubscriptionCardTitle`，无 plan title 时显示 `Enterprise Allocation Wallet · Subscription #id`。
  - `getManagedSubscriptionNote` 返回 `Managed by department · Cannot be deleted by user`。
  - 组件中 active/cancelled/expired、source、priority、remaining、billing preference 都是 wallet 可见字段，7B.3 要保证 locale 覆盖和测试覆盖。
  - `end_time <= 0` 的业务语义主要属于 Story 7B.4；7B.3 不应在本故事大改时间语义，但若新增“未知/未设置”文案，必须同步 locale。

- `web/default/scripts/sync-i18n.mjs`
  - `bun run i18n:sync` 以 `en` 为基准重排所有 locale，并检测疑似未翻译值。
  - 脚本内 `BRAND_AND_LITERAL_KEYS` 包含受保护或专有名词，开发者不得借 i18n 清理删除 `new-api`、`QuantumNous`、版权、品牌和元数据。

### Previous Story Intelligence

- Story 7B.2 已完成预算池富选项和已选摘要，新增共享 helper/component，并让钱包入口和企业组织工作台共用预算池展示口径。7B.3 必须在这些 helper 上补齐 status/type i18n，而不是复制第二套展示逻辑。[Source: `_bmad-output/implementation-artifacts/7b-2-redesign-budget-pool-selection-and-selected-summary.md`#Completion Notes List]
- Story 7B.2 的 senior review 已要求实际 option/summary 组件测试覆盖部门、Budget ID、预算类型、remaining、status。7B.3 扩展测试时应保留这些断言，并增加非英文 locale / 内部 key 不外露断言。[Source: `_bmad-output/implementation-artifacts/7b-2-redesign-budget-pool-selection-and-selected-summary.md`#Senior Developer Review]
- Story 7B.1 已完成治理动作结果与通知投递结果解耦；`unconfigured` delivery 状态是非阻塞状态，不能回退成 `final_failed` 或审批失败，也不能显示 resend。[Source: `_bmad-output/implementation-artifacts/7B-1-decouple-governance-action-result-from-notification-delivery.md`#Completion Notes List]
- Story 7A.1 已处理企业治理工作台 i18n 与动作标签映射第一轮收尾；7B.3 是针对 quota request / allocation / delivery / wallet 截图残留的 follow-up，不要回退旧 action label 或引入内部 key 展示。[Source: `_bmad-output/implementation-artifacts/7a-1-complete-enterprise-governance-i18n-and-action-label-mapping.md`#Dev Notes]
- Story 7A.2 已把员工 quota request 入口移到钱包用户流，并要求提交成功后精确失效 quota request、governance timeline、notification query。7B.3 不要把入口迁回企业组织工作台，也不要扩大缓存失效范围。[Source: `_bmad-output/implementation-artifacts/7a-2-move-employee-quota-request-entry-to-balance-wallet-user-flow.md`#Senior Developer Review]

### Git Intelligence Summary

- 最近提交 `cf67bde3b feat(story-7B.2): 重构额度申请预算池选择与已选摘要` 新增 `quota-request-budget-display.ts`、`quota-request-budget-display-components.tsx`，并改动 `enterprise-organization/index.tsx`、钱包 quota request 卡片、相关测试和全部 Default locale。7B.3 应先基于这些新 helper 做 i18n 收敛。
- 前一提交 `553d9ac0f feat(story-7B.1): 解耦治理动作结果与通知投递结果` 改动治理 timeline/notification 展示。7B.3 修改同一区域时必须保留治理动作与 delivery 分层。
- 当前工作树存在与本故事无关的未提交改动（BMAD story automator、hooks、测试摘要、story-automator 输出等）。Dev agent 不得回退、删除或重写这些无关变更。

### Architecture Compliance

- Default 前端继续使用 React 19、TypeScript、TanStack Query、i18next、Base UI/shadcn-style primitives、Tailwind 和 lucide icons；不新增依赖。[Source: `web/default/package.json`; `_bmad-output/planning-artifacts/architecture.md`#Frontend Architecture]
- 前端 i18n 使用 `useTranslation()` / `t('English source string')`，translation files 是 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` flat JSON。[Source: `AGENTS.md`#Internationalization]
- 本故事是 Default 前端 i18n/显示层修复，不改 Go 数据库、migration、router、service、relay、billing expression、quota request DTO 或上游 provider request DTO。[Source: `_bmad-output/planning-artifacts/architecture.md`#No Relay-Path Department Deduction; `AGENTS.md`#Rule 6]
- Query key 必须保持 feature-scoped 精确模式；本故事正常不需要改 query invalidation。如果测试触碰 submit 成功路径，保留 `quotaRequestQueryScopeKey`、`governanceTimelineQueryKey`、`governanceNotificationQueryKey` 的精确失效。[Source: `web/default/src/features/enterprise-organization/api.ts`]
- 受保护标识 `new-api`、`QuantumNous` 不得删除、替换、改名或从版权/元数据/i18n sync literal allowlist 中移除。[Source: `AGENTS.md`#Rule 5]

### Library / Framework Requirements

- 不新增 i18n 库、不新增翻译服务、不引入运行时翻译 API。只使用现有 `i18next`、`react-i18next` 和 locale JSON。
- 使用现有 Node test / React server render 测试模式；现有测试通过 `I18nextProvider` 和 `renderToStaticMarkup` 验证 UI 文案。
- 使用现有 `StatusBadge`、`Badge`、`formatNumber`、`formatQuota`、`formatTimestamp` 和预算池展示 helper；不要新建平行 UI 基础组件。
- 不需要 web research 或依赖升级；本故事依赖版本已由 `web/default/package.json` 固定。

### File Structure Requirements

Expected UPDATE files:

- `web/default/src/features/enterprise-organization/quota-request-budget-display.ts` - 状态/type/预算池展示 helper 的稳定 i18n fallback。
- `web/default/src/features/enterprise-organization/quota-request-budget-display-components.tsx` - 如 helper 输出结构变化，同步 option/summary 展示。
- `web/default/src/features/enterprise-organization/index.tsx` - quota request table、governance action/delivery、allocation/wallet table 可见文案和映射使用点。
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx` - 非英文 locale、action/status/delivery 不裸露内部值测试。
- `web/default/src/features/wallet/components/employee-quota-request-card.tsx` - 仅在发现钱包入口可见文案缺口时更新。
- `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx` - 钱包 quota request 非英文文案测试。
- `web/default/src/features/wallet/components/subscription-plans-card.tsx` - 仅在发现企业 wallet 可见文案缺口时更新。
- `web/default/src/features/wallet/components/subscription-plans-card.test.tsx` - 企业 wallet title/note/status i18n 测试。
- `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` - locale 资源同步。

Likely no changes needed:

- `controller/enterprise/*`、`service/enterprise/*`、`dto/enterprise/*` - 本故事不改变后端 API 或状态语义。
- `model/enterprise/*`、`model/main.go` - 无 schema 变更。
- `relay/**`、`pkg/billingexpr/**` - 本故事不涉及 relay 或计费表达式。
- `web/classic/**` - Story 7B.3 明确来自 Default 前端截图残留；除非用户另行要求，不扩展 Classic。

### Project Structure Notes

- 如果需要共享映射，把 helper 放在 `web/default/src/features/enterprise-organization/quota-request-budget-display.ts` 或同 feature 下的小 helper 文件中；不要把企业治理状态映射塞进通用 `lib`，除非已有调用面明确跨 feature。
- `enterpriseBudgetStatusLabel` 目前被预算池、allocation、wallet、quota request、timeline 共用。修改时要确认所有调用场景的状态词不会互相误标；必要时拆成 `enterpriseGovernanceStatusLabel` / `quotaRequestStatusLabel` 等更具体 helper，但要避免重复映射。
- `QuotaRequestItem` 当前没有完整 budget object，只有 `department_budget_id`。若只为了本地化身份文案，不要新增 API；可传入当前预算池列表做 best-effort 匹配，或至少使用 `t('Budget #{{budgetId}}')`。
- 对未知后端 code 的 UI 策略：用户主 label 使用本地化“未知状态/未知动作/未知投递状态”，必要时在辅助 text 或 tooltip 中显示 code 供排障；不要把 code 当作未翻译 label。
- `error_reason`、`trace_id`、`source_type`、`source_id`、`Budget #id` 属于数据或诊断值，不是 locale key；但它们周围的 label 和状态必须本地化。

### Testing Requirements

Minimum targeted validation:

```bash
cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx
cd web/default && bun test src/features/wallet/components/employee-quota-request-card.test.tsx
cd web/default && bun test src/features/wallet/components/subscription-plans-card.test.tsx
cd web/default && bun run typecheck
cd web/default && bun run i18n:sync
```

Recommended assertions:

- `enterpriseBudgetStatusLabel('submitted', zhT)` 返回本地化“已提交”语义，不返回 `submitted`。
- `enterpriseBudgetStatusLabel('unrecognized_status', zhT)` 返回本地化未知状态文案，不返回裸 code。
- `governanceActionLabel('enterprise.organization.quota_request.submit', zhT)` 返回本地化“额度申请已提交”语义，不返回 action key。
- `governanceDeliveryStatusLabel('unconfigured', zhT)` 返回本地化“未配置”语义，且 delivery list 不显示 resend。
- `governanceDeliveryStatusLabel('final_failed', zhT)` 返回本地化最终失败语义，且 final_failed 仍允许 resend。
- `QuotaRequestTable` 在非英文 locale 下不出现 `submitted`、`approved`、`rejected`、`fulfilled` 等裸状态。
- `QuotaRequestTable` Target Budget Pool 显示 `Budget #id` 的本地化模板或复用预算池 summary，而不是裸 `#id`。
- 钱包 quota request option/summary 和企业组织工作台 option/summary 在非英文 locale 下显示本地化 type/status/remaining。
- 企业派生 wallet title/note/status 通过 locale 输出；测试避免新增硬编码英文说明。
- `bun run i18n:sync` 后 locale 文件无缺失 key；对 zh/ja/ru 等非拉丁语言不留下明显英文句子，专有名词除外。

### Anti-Patterns to Avoid

- 不要只把缺失英文 key 追加到 `en.json`，却不补 zh/fr/ru/ja/vi。
- 不要依赖 i18next fallback，让非英文语言缺 key 后显示英文源字符串。
- 不要让 `enterpriseBudgetStatusLabel`、`governanceActionLabel`、`governanceDeliveryStatusLabel` 默认返回裸内部 code。
- 不要把 `unconfigured` 通知状态重新当作业务失败，也不要让它显示 resend。
- 不要改后端状态枚举、审批/分配事务、delivery retry 语义或 query invalidation 范围来完成 i18n。
- 不要用机器生成的大段低质量翻译覆盖已有人工翻译；新增 key 要简短准确，保留已有翻译质量。
- 不要删除、改名或替换受保护的 `new-api`、`QuantumNous`、版权、包名和品牌信息。

### References

- [Source: `_bmad-output/planning-artifacts/epics.md`#Story 7B.3]
- [Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Change I]
- [Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Change M]
- [Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Success Criteria]
- [Source: `_bmad-output/planning-artifacts/architecture.md`#Internationalization]
- [Source: `AGENTS.md`#Internationalization]
- [Source: `web/default/src/features/enterprise-organization/quota-request-budget-display.ts`#enterpriseBudgetStatusLabel]
- [Source: `web/default/src/features/enterprise-organization/index.tsx`#GovernanceActivityCard / QuotaRequestTable / DepartmentBudgetDetailTable]
- [Source: `web/default/src/features/wallet/components/employee-quota-request-card.tsx`#EmployeeQuotaRequestCard]
- [Source: `web/default/src/features/wallet/components/subscription-plans-card.tsx`#getSubscriptionCardTitle / getManagedSubscriptionNote]
- [Source: `web/default/scripts/sync-i18n.mjs`#i18n sync behavior]
- [Source: `_bmad-output/implementation-artifacts/7b-2-redesign-budget-pool-selection-and-selected-summary.md`#Senior Developer Review]
- [Source: `_bmad-output/implementation-artifacts/7B-1-decouple-governance-action-result-from-notification-delivery.md`#Completion Notes List]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-03 14:51:03 +0800: BMAD create-story workflow executed for Story 7B.3 in YOLO mode.
- 2026-06-03: Read `.agents/skills/bmad-create-story/SKILL.md`, `discover-inputs.md`, `template.md`, and `checklist.md`.
- 2026-06-03: Workflow resolver failed because local `python3` lacks Python 3.11 `tomllib`; customization was resolved manually per skill fallback.
- 2026-06-03: Loaded `_bmad/bmm/config.yaml`: `user_name=hth`, communication/document language Chinese, planning artifacts `_bmad-output/planning-artifacts`, implementation artifacts `_bmad-output/implementation-artifacts`.
- 2026-06-03: Persistent facts glob `file:{project-root}/**/project-context.md` found no files.
- 2026-06-03: Input discovery loaded relevant sections from `epics.md`, `architecture.md`, `sprint-change-proposal-2026-06-03.md`, previous story `7b-2-redesign-budget-pool-selection-and-selected-summary.md`, git history, sprint status, current enterprise organization / wallet implementation files, tests, locale files, and `sync-i18n.mjs`.
- 2026-06-03: Confirmed story key from sprint status: `7b-3-complete-quota-request-governance-wallet-i18n-coverage`, previous status `backlog`.
- 2026-06-03: Checklist validation applied: story includes exact AC, current-state notes for UPDATE files, previous story intelligence, i18n mapping guardrails, protected identifier constraints, file boundaries, anti-patterns, and targeted tests.
- 2026-06-03 15:11:06 +0800: Senior developer review workflow executed in auto-fix mode for Story 7B.3.
- 2026-06-03: Reviewed actual git changes against story AC and File List; found story artifact had not documented the frontend implementation files.
- 2026-06-03: Auto-fixed wallet subscription display fallback so billing preference and source values no longer expose raw internal codes such as `enterprise_allocation` or unknown enum values.
- 2026-06-03: Auto-fixed missing locale coverage for enterprise wallet source/fallback fields and `Subscription Priority` across `en`, `zh`, `fr`, `ja`, `ru`, and `vi`.
- 2026-06-03: Fixed TypeScript test fixture typing for unknown budget/status payload coverage.
- 2026-06-03: Validation passed: targeted enterprise organization test, wallet quota request test, subscription plans test, typecheck, i18n sync, and full `test:e2e`.

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- Story scope is intentionally Default frontend i18n/display-layer only; no backend API, database, relay, or billing expression changes are required.
- `enterpriseBudgetStatusLabel` now covers quota request, allocation, wallet, and budget lifecycle statuses used by this story, and falls back to localized `Unknown status` instead of raw status codes.
- `formatBudgetType` now distinguishes `balance` and `subscription`, and falls back to localized `Unknown budget type`.
- `QuotaRequestTable` now renders target budget identity through the localized `Budget #{{budgetId}}` template instead of a bare `#id`.
- Governance action and delivery mapping behavior was preserved, including neutral `unconfigured` delivery status and resend only for `final_failed`.
- Wallet quota request option/summary and form labels are covered by zh locale regression tests and do not expose unknown type/status codes.
- Enterprise allocation wallet title, managed note, source, priority, and billing preference labels are covered by locale resources and regression tests.
- Senior review auto-fix added subscription wallet source/billing preference fallback helpers and completed missing locale keys across all supported Default frontend languages.
- No backend API, database, relay, billing expression, or protected branding changes were made.

### File List

- `_bmad-output/implementation-artifacts/7b-3-complete-quota-request-governance-wallet-i18n-coverage.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `web/default/src/features/enterprise-organization/quota-request-budget-display.ts`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx`
- `web/default/src/features/wallet/components/subscription-plans-card.tsx`
- `web/default/src/features/wallet/components/subscription-plans-card.test.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`

## Senior Developer Review (AI)

Reviewer: hth
Date: 2026-06-03

### Outcome

Approved after auto-fix. No critical issues remain.

### Findings

- HIGH fixed: Story File List only listed the story artifact and sprint status, while implementation changed Default frontend source, tests, and locale files. File List and review notes were updated to match actual git changes.
- MEDIUM fixed: `SubscriptionPlansCard` rendered `source_type` / `source` directly in the wallet list, so enterprise wallets displayed raw `enterprise_allocation` in a user-visible `Source` field. Added `getSubscriptionSourceLabel` and localized source/fallback labels.
- MEDIUM fixed: `getBillingPreferenceLabel` fell back to returning unknown internal preference codes. Unknown values now render localized `Unknown billing preference`.
- MEDIUM fixed: `Subscription Priority` was used in UI but missing from all locale resources. Added translations across `en`, `zh`, `fr`, `ja`, `ru`, and `vi`.
- LOW fixed: Unknown budget/status regression test used enum-invalid fixture overrides that failed `tsc -b`. The test now models abnormal API payloads via an explicit boundary cast.

### Validation

- Passed: `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx`
- Passed: `cd web/default && bun test src/features/wallet/components/employee-quota-request-card.test.tsx`
- Passed: `cd web/default && bun test src/features/wallet/components/subscription-plans-card.test.tsx`
- Passed: `cd web/default && bun run typecheck`
- Passed: `cd web/default && bun run i18n:sync`
- Passed: `cd web/default && bun run test:e2e`
