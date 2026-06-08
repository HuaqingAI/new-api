---
baseline_commit: 56b663567d9b458b22c5d9ac9331f21f7f38bdd1
---

# Story 7B.4: 统一一次性企业 wallet 与非正过期时间展示语义

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员或员工,
I want 一次性企业 wallet 和无过期时间场景用业务语义展示,
so that 我不会看到 1970 时间、错误的“已过期”判断或过于技术化的文案。

## Acceptance Criteria

1. **Given** 企业派生 wallet 的 `cycle_type = never`
   **When** 页面展示周期信息
   **Then** 使用“一次性额度”或等价业务文案
   **And** 不再仅显示 `No Reset`。

2. **Given** `end_time <= 0` 或 `expires_at <= 0`
   **When** 页面展示过期信息
   **Then** 不得格式化为 epoch 时间
   **And** 必须展示“永不过期”或“未设置过期时间”等业务语义，并避免误判为已过期。

## Tasks / Subtasks

- [x] 建立企业 wallet 显示语义 helper，避免分散判断 (AC: 1, 2)
  - [x] 在现有企业组织或钱包 feature 内新增/调整纯函数，用于判断企业派生 wallet 的周期语义、过期语义和活跃/过期状态。
  - [x] `cycle_type === 'never'` 在企业 wallet / 派生 wallet 场景返回 `t('One-time quota')` 或同等业务文案，不再复用普通订阅计划的 `No Reset` 作为主展示。
  - [x] `end_time <= 0`、`expires_at <= 0`、缺失时间值不得传入 `new Date(value * 1000)` 或通用 timestamp formatter 形成 1970/epoch 展示。
  - [x] 未知 `cycle_type` 主文案应使用本地化未知周期文案，必要时在辅助位置展示 code；不要把内部 code 当用户主文案。

- [x] 修正钱包页企业分配 wallet 卡片的非正 `end_time` 行为 (AC: 2)
  - [x] 更新 `web/default/src/features/wallet/components/subscription-plans-card.tsx` 中 `getRemainingDays`、`isExpired`、状态 badge、Until/Expired at 行和 next reset 展示。
  - [x] 企业分配 wallet 且 `end_time <= 0` 时必须显示 active/可用语义，不得因为 `(end_time || 0) < now` 被误判为 expired。
  - [x] 企业分配 wallet 且无过期时间时，时间行显示 `Never expires` / `No expiry set` 的本地化文案，而不是 `1970-*` 或 `Invalid Date`。
  - [x] 普通订阅计划仍按既有 `end_time` 语义展示；不要把所有普通订阅的 `end_time=0` 静默当作有效长期订阅，除非现有数据契约明确如此。

- [x] 修正企业组织预算详情和派生 wallet 表格的周期/过期展示 (AC: 1, 2)
  - [x] 更新 `DepartmentBudgetDetailTable` 的 `Cycle / Expiry` 列：`wallet.cycle_type === 'never'` 展示一次性额度；`wallet.expires_at <= 0` 展示永不过期/未设置过期时间语义。
  - [x] 更新 `DepartmentBudgetOverviewCard` / `DepartmentBudgetStatusCard` 的 `Cycle Type`、`Expires At (optional)`：对 balance/一次性预算池避免显示技术化 `No Reset` 和 `-`，优先展示业务语义。
  - [x] 保留 `next_reset_time > 0` 时的正常下一次重置时间展示；不要把有效正时间覆盖成“永不过期”。
  - [x] 不改变预算池、allocation、quota request 的后端状态机和存储字段。

- [x] 补齐 i18n 资源 (AC: 1, 2)
  - [x] 在 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` 添加本故事新增文案。
  - [x] 推荐 key 包含：`One-time quota`、`Never expires`、`No expiry set`、`Unknown cycle type`，以及 UI 实际使用的等价英文源字符串。
  - [x] 继续使用 flat JSON 和英文源字符串 key；不要新增 namespace，不要依赖 i18next fallback 让非英文语言显示英文源字符串。
  - [x] 不删除、改名或替换受保护 `new-api`、`QuantumNous`、版权、品牌和 i18n sync allowlist 条目。

- [x] 扩展回归测试覆盖企业 wallet 时间语义 (AC: 1, 2)
  - [x] 扩展 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`：`cycle_type='never'`、`expires_at=0` 的预算详情和派生 wallet 不出现 `No Reset`、`1970`、epoch 日期或裸 `never` code，并显示一次性/永不过期业务文案。
  - [x] 扩展 `web/default/src/features/wallet/components/subscription-plans-card.test.tsx`：企业分配 wallet `end_time=0` 或负数时不被判为 expired，显示 active/长期有效语义，且不出现 `1970`。
  - [x] 测试普通订阅计划的 `No Reset` 仍可作为普通 plan reset-period 文案出现，避免误伤订阅计划购买卡片。
  - [x] 增加 zh locale 断言，确认新增业务文案有翻译且不回退英文源字符串。

- [x] 运行并记录验证 (AC: 1, 2)
  - [x] `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx`
  - [x] `cd web/default && bun test src/features/wallet/components/subscription-plans-card.test.tsx`
  - [x] `cd web/default && bun run typecheck`
  - [x] `cd web/default && bun run i18n:sync`
  - [x] 未修改视觉布局；本次未运行浏览器截图检查。

## Dev Notes

### 需求与业务上下文

- Epic 7B 是 Epic 7 / 7A 完成后的企业治理 follow-up，目标是修正治理通知解耦、预算池可辨识性、i18n 残留、一次性 wallet 展示和金额视角；本故事只处理一次性企业 wallet 与非正过期时间的显示语义。[Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Epic 7B]
- Story 7B.4 明确要求 `cycle_type = never` 展示“一次性额度”或等价业务文案，不再仅显示 `No Reset`；`end_time <= 0` 或 `expires_at <= 0` 不得格式化为 epoch 时间，并且不得误判为已过期。[Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Change J]
- 架构 V1.3B 锁定企业 wallet 显示语义：显示层必须先解释业务语义再格式化时间；`cycle_type = never` 表示一次性/无周期重置，`end_time <= 0`、`expires_at <= 0` 表示永不过期或未设置过期时间。[Source: `_bmad-output/planning-artifacts/architecture.md`#Data Flow item 10]
- 2026-06-03 correct-course 的成功标准包含“一次性企业 wallet 与非正过期时间不再显示 1970/错误过期状态/技术化文案”。实现验收应以用户可见 UI 为准，不只改 helper。[Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Success Criteria]

### 当前实现状态（已读 UPDATE 文件）

- `web/default/src/features/wallet/components/subscription-plans-card.tsx`
  - 当前 `getRemainingDays` 对 `end_time` 缺失/0 返回 0；列表渲染里 `const isExpired = (subscription?.end_time || 0) < now` 会把 `end_time=0` 的企业分配 wallet 判为 expired。
  - 当前 Until/Cancelled at/Expired at 行直接执行 `new Date((subscription?.end_time || 0) * 1000).toLocaleString()`，`end_time <= 0` 会产生 1970 附近日期或无业务意义的时间。
  - 7B.3 已新增 `getSubscriptionCardTitle`、`getManagedSubscriptionNote`、`getBillingPreferenceLabel`、`getSubscriptionSourceLabel` 并完成 i18n 覆盖。7B.4 应在这些 helper 附近补显示语义 helper，避免在 JSX 中继续散落时间判断。
  - 普通 subscription plan 的 `formatResetPeriod(plan, t)` 仍返回 `t('No Reset')`，购买卡片用它来隐藏 reset benefit；7B.4 不应全局删除普通订阅计划的 `No Reset` 语义。

- `web/default/src/features/enterprise-organization/index.tsx`
  - `DepartmentBudgetDetailTable` 的 `Cycle / Expiry` 列当前展示 `formatBudgetCycleType(wallet.cycle_type, t)`，并用 `wallet.next_reset_time ? ... : wallet.expires_at ? ... : '-'` 处理时间；`expires_at=0` 只显示 `-`，没有业务语义。
  - `DepartmentBudgetOverviewCard` / `DepartmentBudgetStatusCard` 的 `Cycle Type` 当前通过 `formatBudgetCycleType` 展示；`formatBudgetCycleType('never')` 返回 `t('No Reset')`，与本故事要求冲突。
  - `Expires At (optional)` 当前用 `budget.expires_at ? formatTimestamp(budget.expires_at) : '-'`，`expires_at=0` 显示 `-`，仍缺少“永不过期/未设置过期时间”语义。
  - `formatBudgetCycleType` 对未知 cycle type 返回裸 `cycleType || '-'`；如本故事触碰该 helper，应同步修复为本地化 fallback，避免重引入 7B.3 已避免的内部 code 外露问题。

- `web/default/src/lib/format.ts`
  - 通用 `formatTimestamp(timestamp)` 仅对 `timestamp === -1` 返回硬编码 `Never`，其他值调用 `formatTimestampToDate`；`formatTimestampToDate(0)` 返回 `-`。
  - 这个通用 formatter 没有业务上下文和 i18n 参数。7B.4 不应把企业 wallet 业务语义塞进通用 formatter，除非能保证不破坏其他调用；更稳妥是在企业/wallet feature 层先判断非正时间。

- `web/default/src/features/subscriptions/lib/format.ts`
  - `formatResetPeriod` 对普通订阅计划的 reset period 返回 `t('No Reset')`。这是订阅计划上下文，不等同于企业派生 wallet 的“一次性额度”。本故事应区分两个上下文。

- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - 现有 `renders balance budget details without subscription-only fields` 断言 `cycle_type='never'` 时出现 `No Reset`。7B.4 需要更新该断言为一次性/无重置业务语义，并增加非正 expires 断言。
  - 现有 `renders budget detail wallet lineage with revoked and expired states` 覆盖 monthly + next_reset_time 正值路径，但没有覆盖 `cycle_type='never'`、`expires_at=0` 或 `expires_at<0`。

- `web/default/src/features/wallet/components/subscription-plans-card.test.tsx`
  - 当前仅测试 title、managed note、billing preference/source fallback 和 locale 覆盖，没有渲染企业 wallet `end_time<=0` 的状态/日期行为。
  - 7B.4 应增加可渲染或 helper 级测试；如果组件完整渲染需要较多 API mock，优先导出纯 helper 覆盖 `isExpired`、date line、remaining days、expiry label。

### Previous Story Intelligence

- Story 7B.3 刚完成企业治理与钱包相关 i18n 收尾，新增/修改了 `subscription-plans-card.tsx`、企业组织表格、全部 locale 和测试。7B.4 修改同一批文件时必须保留 7B.3 的稳定映射：未知 status/source/preference 不得回退裸内部 code。[Source: `_bmad-output/implementation-artifacts/7b-3-complete-quota-request-governance-wallet-i18n-coverage.md`#Completion Notes List]
- Story 7B.3 明确指出 `end_time <= 0` 的业务语义属于 7B.4，不应在 i18n 故事里大改；因此本故事要补上该遗留点，并继续同步 locale。[Source: `_bmad-output/implementation-artifacts/7b-3-complete-quota-request-governance-wallet-i18n-coverage.md`#当前实现状态]
- Story 7B.2 建立预算池展示 helper/component 复用策略。7B.4 同样应偏向共享 helper，而不是钱包页、预算详情、预算概览各写一套条件判断。[Source: `_bmad-output/implementation-artifacts/7b-2-redesign-budget-pool-selection-and-selected-summary.md`#Completion Notes List]
- Story 7B.1 已要求治理动作与通知投递状态解耦。7B.4 不涉及 notification delivery，不要借时间语义修复改动治理动作结果或通知重试行为。[Source: `_bmad-output/implementation-artifacts/7B-1-decouple-governance-action-result-from-notification-delivery.md`#Completion Notes List]

### Git Intelligence Summary

- 最近提交 `56b663567 feat(story-7B.3): 完成额度申请与治理通知相关 i18n 收尾` 修改了企业组织、钱包 subscription card、钱包 quota request card、全部 Default locale 和相关测试。7B.4 的主要修改面与该提交重叠，开发前应先读当前文件而不是依据旧 story 猜测。
- 前序提交 `cf67bde3b feat(story-7B.2): 重构额度申请预算池选择与已选摘要` 新增预算池展示 helper；`553d9ac0f feat(story-7B.1)` 修改治理 timeline/notification 展示。本故事不应回退这些实现。
- 当前工作树可能存在 story automator、测试摘要或其他未提交文件。Dev agent 不得回退、删除或重写与本故事无关的改动。

### Architecture Compliance

- 本故事是 Default 前端显示层修复，不改 Go 数据库、migration、router、service、relay、billing expression、quota request DTO 或上游 provider request DTO。[Source: `_bmad-output/planning-artifacts/architecture.md`#No Relay-Path Department Deduction; `AGENTS.md`#Rule 6]
- Default 前端继续使用 React 19、TypeScript、i18next、Base UI/shadcn-style primitives、Tailwind 和 lucide icons；不新增依赖。[Source: `_bmad-output/planning-artifacts/architecture.md`#Frontend Architecture; `web/default/package.json`]
- 前端 i18n 使用 `useTranslation()` / `t('English source string')`，translation files 是 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` flat JSON。[Source: `AGENTS.md`#Internationalization]
- 企业预算治理底层仍以 quota 作为存储和约束单位；本故事只改周期/过期展示，不引入金额换算，不提前实现 7B.5。[Source: `_bmad-output/planning-artifacts/architecture.md`#Data Flow item 11]
- 受保护标识 `new-api`、`QuantumNous` 不得删除、替换、改名或从版权/元数据/i18n sync literal allowlist 中移除。[Source: `AGENTS.md`#Rule 5]

### Library / Framework Requirements

- 不新增日期库；继续使用现有 `dayjs`/`formatTimestamp`/浏览器日期能力，但企业 wallet 的非正时间必须先在 feature helper 层判断。
- 不新增 i18n 库、不新增翻译服务、不引入运行时翻译 API。只使用现有 `i18next`、`react-i18next` 和 locale JSON。
- 使用现有 Node test / React server render 测试模式；如导出纯 helper，优先用 `node:test` 精准覆盖矩阵。
- 不需要 web research 或依赖升级；本故事依赖版本已由 `web/default/package.json` 固定。

### File Structure Requirements

Expected UPDATE files:

- `web/default/src/features/wallet/components/subscription-plans-card.tsx` - 企业分配 wallet active/expired 判断、时间行、剩余天数和显示 helper。
- `web/default/src/features/wallet/components/subscription-plans-card.test.tsx` - 企业分配 wallet `end_time<=0` 语义、locale 覆盖和不出现 1970/Expired 的测试。
- `web/default/src/features/enterprise-organization/index.tsx` - `formatBudgetCycleType`、预算详情/预算概览 `Cycle / Expiry` 和 `Expires At` 显示。
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx` - `cycle_type=never`、`expires_at<=0` 预算/派生 wallet 回归测试。
- `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` - 新增业务语义文案。

Possible helper file:

- `web/default/src/features/enterprise-organization/quota-request-budget-display.ts` 或新增同 feature 小 helper - 如果希望企业组织表格复用 `formatBudgetCycleType`/expiry helper，放在企业 feature 内，不要塞进通用 `lib/format.ts`。

Likely no changes needed:

- `controller/enterprise/*`、`service/enterprise/*`、`dto/enterprise/*` - 不改变 API 或后端状态语义。
- `model/enterprise/*`、`model/main.go` - 无 schema 变更。
- `relay/**`、`pkg/billingexpr/**` - 不涉及 relay 或计费表达式。
- `web/classic/**` - 7B follow-up 来自 Default 前端钱包/治理显示问题；除非用户另行要求，不扩展 Classic。

### Project Structure Notes

- 推荐实现为“语义 helper + UI 调用”：例如 `isEnterpriseAllocationWallet(sub)`、`hasPositiveTimestamp(value)`、`getEnterpriseWalletExpiryDisplay(...)`、`getEnterpriseWalletStatus(...)`。helper 返回 label/status，不让 JSX 自己判断非正时间。
- `No Reset` 可以继续存在于普通订阅 plan reset period 场景；企业派生 wallet 的主展示必须使用“一次性额度/无周期重置”类业务文案。不要全局把 `No Reset` 翻译改成“一次性额度”，否则会误伤订阅计划购买页。
- 有效正时间保持现有日期展示；只有 `<=0`、空值、缺失值需要业务 fallback。
- 状态判断要区分业务 status 与时间派生 status：`status='cancelled'` 仍应显示 Cancelled；`status='active' && end_time<=0` 应显示 Active/长期有效，而不是 Expired。
- 不要把 `formatTimestamp` 改成对所有 `0` 返回“永不过期”；通用 formatter 没有业务上下文，其他模块可能需要 `-`。

### Testing Requirements

Minimum targeted validation:

```bash
cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx
cd web/default && bun test src/features/wallet/components/subscription-plans-card.test.tsx
cd web/default && bun run typecheck
cd web/default && bun run i18n:sync
```

Recommended assertions:

- `cycle_type='never'` 的企业预算/派生 wallet 渲染“一次性额度”或等价业务文案，不出现 `No Reset` 作为主周期文案。
- `expires_at=0`、`expires_at=-1` 的企业预算/派生 wallet 显示“永不过期/未设置过期时间”，不出现 `1970`、`1969`、`Invalid Date` 或裸 `-`。
- 企业分配 wallet `source_type='enterprise_allocation'`、`status='active'`、`end_time=0` 显示 Active 和长期有效语义，不显示 Expired。
- 企业分配 wallet `end_time=-1` 与 `end_time=0` 行为一致或按业务语义区分，但都不得格式化为 epoch。
- 普通订阅 plan 的 reset period 仍可返回 `No Reset`，购买卡片隐藏 reset benefit 的既有逻辑不被破坏。
- zh locale 下新增语义文案不回退英文源字符串；所有 supported locale 都有新增 key。

### Anti-Patterns to Avoid

- 不要把 `end_time || 0` 直接用于过期判断或 `new Date(... * 1000)`。
- 不要把 `expires_at <= 0` 传入日期格式化后显示 `1970-*`、`1969-*`、`Invalid Date` 或 `-` 作为业务结论。
- 不要全局改 `No Reset` 翻译来完成企业 wallet 文案；普通订阅计划上下文仍需要原 reset-period 语义。
- 不要让未知 `cycle_type` 回退裸内部 code。
- 不要改后端状态枚举、审批/分配事务、delivery retry、query invalidation 或 wallet lifecycle 存储逻辑。
- 不要只更新 `en.json`；必须同步 zh/fr/ru/ja/vi 并运行 i18n sync。
- 不要删除、改名或替换受保护的 `new-api`、`QuantumNous`、版权、包名和品牌信息。

### References

- [Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Change J]
- [Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Success Criteria]
- [Source: `_bmad-output/planning-artifacts/architecture.md`#Data Flow item 10]
- [Source: `AGENTS.md`#Internationalization]
- [Source: `AGENTS.md`#Rule 5]
- [Source: `web/default/src/features/wallet/components/subscription-plans-card.tsx`#subscription list render]
- [Source: `web/default/src/features/enterprise-organization/index.tsx`#DepartmentBudgetDetailTable]
- [Source: `web/default/src/features/enterprise-organization/index.tsx`#DepartmentBudgetOverviewCard]
- [Source: `web/default/src/features/enterprise-organization/index.tsx`#formatBudgetCycleType]
- [Source: `web/default/src/lib/format.ts`#formatTimestamp]
- [Source: `web/default/src/features/subscriptions/lib/format.ts`#formatResetPeriod]
- [Source: `_bmad-output/implementation-artifacts/7b-3-complete-quota-request-governance-wallet-i18n-coverage.md`#Previous Story Intelligence]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-03: BMAD create-story workflow executed for Story 7B.4 in YOLO mode.
- 2026-06-03: Read `.agents/skills/bmad-create-story/SKILL.md`, `discover-inputs.md`, `template.md`, and `checklist.md`.
- 2026-06-03: Resolved workflow customization with `_bmad/scripts/resolve_customization.py`; no activation prepend/append steps, persistent facts glob `file:{project-root}/**/project-context.md`.
- 2026-06-03: Loaded `_bmad/bmm/config.yaml`: `user_name=hth`, communication/document language Chinese, planning artifacts `_bmad-output/planning-artifacts`, implementation artifacts `_bmad-output/implementation-artifacts`.
- 2026-06-03: Persistent facts glob found no `project-context.md` files.
- 2026-06-03: Loaded sprint status and confirmed story key `7b-4-normalize-one-time-wallet-and-non-positive-expiry-display` previous status `backlog`.
- 2026-06-03: Loaded relevant Epic 7B source from `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`, architecture V1.3B data flow items, previous story 7B.3, git history, current wallet/enterprise organization implementation files, tests, and formatting helpers.
- 2026-06-03: Checklist validation applied: story includes exact AC, current-state notes for UPDATE files, previous story intelligence, i18n guardrails, protected identifier constraints, file boundaries, anti-patterns, and targeted tests.
- 2026-06-03: Story-automator review workflow loaded `.agents/skills/bmad-story-automator-review/{SKILL.md,workflow.yaml,instructions.xml,checklist.md}` and reviewed implementation against git reality.
- 2026-06-03: Auto-fixed review findings: incomplete story File List/status, missing locale keys for `One-time quota` / `No expiry set` / `Unknown cycle type`, and non-positive wallet expiry label consistency.
- 2026-06-03: Validation passed: `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx src/features/wallet/components/subscription-plans-card.test.tsx` (60 pass).
- 2026-06-03: Validation passed: `cd web/default && bun run typecheck`.
- 2026-06-03: Validation passed: `cd web/default && bun run i18n:sync`; sync report has missingCount=0/extrasCount=0 for all locales. Existing ja/ru untranslatedCount=21 remains outside this story's added keys.

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- Story scope is intentionally Default frontend display-layer only; no backend API, database, relay, billing expression, quota request workflow, or notification delivery changes are required.
- Implemented feature-layer timestamp helpers so enterprise allocation wallets with `end_time <= 0` remain active and render semantic expiry text without epoch dates.
- Updated enterprise budget cycle/expiry display so `cycle_type='never'` renders `One-time quota`, non-positive expiry renders `Never expires`, and unknown cycle codes do not leak as primary UI text.
- Added locale coverage for `One-time quota`, `No expiry set`, and `Unknown cycle type` across en/zh/fr/ja/ru/vi while preserving ordinary subscription `No Reset` semantics.
- Added regression tests covering enterprise allocation wallet non-positive expiry, ordinary subscription `end_time=0`, zh locale behavior, enterprise budget one-time quota display, unknown cycle fallback, and no `1970` / raw code output.

### Change Log

- 2026-06-03: Implemented and reviewed Story 7B.4; auto-fixed review findings and marked story done.

### Senior Developer Review (AI)

- Reviewer: GPT-5 Codex on 2026-06-03.
- Outcome: Approved after automatic fixes.
- Findings fixed:
  - MEDIUM: Story File List omitted actual changed frontend implementation/test/locale files, making review traceability incomplete.
  - HIGH: Locale coverage for new UI strings was incomplete; tests failed for missing `One-time quota` and `No expiry set`.
  - MEDIUM: Non-positive expiry display needed a constrained helper path to avoid epoch formatting while preserving ordinary subscription expired semantics.
- Verification:
  - `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx src/features/wallet/components/subscription-plans-card.test.tsx`
  - `cd web/default && bun run typecheck`
  - `cd web/default && bun run i18n:sync`

### File List

- `_bmad-output/implementation-artifacts/7b-4-normalize-one-time-wallet-and-non-positive-expiry-display.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/features/wallet/components/subscription-plans-card.tsx`
- `web/default/src/features/wallet/components/subscription-plans-card.test.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/i18n/locales/zh.json`
