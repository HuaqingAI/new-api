---
baseline_commit: 9a596fe0e4b0d5b64277a7fbb65b9ff93d8e6c1d
---

# Story 7B.5: 为预算池与分配记录补充金额视角和单位切换

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员或员工,
I want 在额度表单里切换“额度 / 金额”视角，并在预算池与分配记录中同时看到 quota 和金额,
so that 我能用更直观的金额视角理解预算和钱包额度，同时保留原 quota 语义。

## Acceptance Criteria

1. **Given** 管理员在创建预算池、创建 allocation、创建 delegation，或员工提交 quota request  
   **When** 填写额度字段  
   **Then** 表单支持在 quota 与金额（$）视角之间切换  
   **And** 该切换不改变后端底层 quota 存储模型。

2. **Given** 用户查看预算池列表、预算池详情、allocation 记录或 quota request 记录  
   **When** 页面展示额度相关列  
   **Then** 在保留 quota 原值的同时补充金额（$）显示  
   **And** 显示方式尽量与钱包额度语义保持一致。

## Tasks / Subtasks

- [x] 建立企业治理额度显示与输入 helper，复用现有钱包换算语义 (AC: 1, 2)
  - [x] 在 `web/default/src/features/enterprise-organization/` 下新增小型 helper 或扩展既有 `quota-request-budget-display.ts`，封装 quota 原值、金额显示、quota <-> amount 输入转换和单位标签。
  - [x] 复用 `web/default/src/lib/format.ts` 中现有 `formatQuota`、`parseQuotaFromDollars`、`quotaUnitsToDollars`；不要新增第二套 `500000 = $1` 常量或手写换算。
  - [x] helper 输出必须能同时展示 raw quota 和金额/钱包显示值，例如主行保留 `1,000,000 quota`，辅助行展示 `≈ $2.00` 或现有 `formatQuota(1000000)` 结果。
  - [x] 如果系统配置不是 USD 显示，金额辅助显示仍应与钱包 `formatQuota` 语义一致；不要硬编码所有场景都显示 `$` 后导致与钱包余额不一致。

- [x] 为四类额度输入表单增加 quota / 金额视角切换 (AC: 1)
  - [x] 预算池创建表单：`total_quota` 和 `cycle_quota` 都支持切换输入视角；提交给 `createBudget` 的 payload 仍为 quota 数值字段。
  - [x] 成员 allocation 创建表单：`committed_quota` 支持切换输入视角；提交给 `createQuotaAllocation` 的 payload 仍为 `committed_quota` quota 单位。
  - [x] 预算 delegation 创建/调整表单：`committed_quota` 和 supersede draft 支持切换或至少显示同步金额预览；提交给 `createBudgetDelegation` / `supersedeBudgetDelegation` 仍为 quota 单位。
  - [x] quota request 表单：企业组织工作台和钱包入口 `EmployeeQuotaRequestCard` 的 `requested_quota` 支持切换输入视角；提交给 `submitQuotaRequest` 仍为 `requested_quota` quota 单位。
  - [x] 负责人审批 quota request 时的 `approvedQuota` draft 支持切换或金额预览；审批 payload 仍为 `approved_quota` quota 单位。
  - [x] 输入视角切换不得清空用户已输入的有效值；切换时应转换当前值并保持正整数 quota 约束。

- [x] 在预算池、wallet、allocation、delegation、quota request 展示中补充金额视角 (AC: 2)
  - [x] `DepartmentBudgetListCard` 的 `Remaining Quota`、`Allocated Total` 列保留 quota 原值并补充金额/钱包显示。
  - [x] `DepartmentBudgetOverviewCard` / `DepartmentBudgetStatusCard` 的 `Remaining Quota`、`Total Quota`、`Cycle Quota` 显示 raw quota + 金额辅助信息。
  - [x] `DepartmentBudgetDetailTable` 的派生 wallet `Quota`、`Remain Quota` 显示 raw quota + 金额辅助信息。
  - [x] `QuotaAllocationTable` 的 `Allocation Quota`、`Reclaimed Quota` 显示 raw quota + 金额辅助信息，并让 active allocation 的 supersede 输入能看到金额视角。
  - [x] `BudgetDelegationTable` 的 `Delegation Quota` 显示 raw quota + 金额辅助信息，并覆盖 active delegation supersede 输入。
  - [x] `QuotaRequestTable` 的 `Requested Quota`、`Approved Quota` 显示 raw quota + 金额辅助信息，并覆盖 approve draft 输入。
  - [x] `GovernanceActivityCard` 的 `Quota Change` 可以补金额辅助信息；若表格拥挤，至少不得破坏现有 raw quota 展示。
  - [x] `QuotaRequestBudgetOption` / `QuotaRequestBudgetSummary` 的 remaining 文案补充金额视角，使钱包入口和企业组织入口保持一致。

- [x] 补齐 i18n 资源 (AC: 1, 2)
  - [x] 在 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` 添加新增文案。
  - [x] 推荐 key 包含：`Quota view`、`Amount view`、`Quota units`、`Amount value`、`Equivalent amount`、`Raw quota`、`{{quota}} quota`、`Approx. {{amount}}`、`Stored as quota units`，以及 UI 实际使用的英文源字符串。
  - [x] 继续使用 flat JSON 和英文 source key；不要新增 namespace，不要依赖 fallback 让非英文语言显示英文源字符串。
  - [x] 不删除、改名或替换受保护 `new-api`、`QuantumNous`、版权、品牌和 i18n sync allowlist 条目。

- [x] 扩展回归测试 (AC: 1, 2)
  - [x] 扩展 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`，覆盖 budget list/detail/status、allocation table、delegation table、quota request table 的 raw quota + 金额辅助显示。
  - [x] 扩展 `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx`，覆盖钱包入口 quota request 表单出现单位切换/金额视角，并且仍提交 quota 单位 payload。
  - [x] 增加 helper 级测试，覆盖 `quotaUnitsToDollars` / `parseQuotaFromDollars` 复用路径、金额输入小数、非数字输入、切换后 quota 正整数化。
  - [x] 增加 zh locale 断言，确认新增单位切换与金额辅助文案有翻译且不回退英文源字符串。

- [x] 运行并记录验证 (AC: 1, 2)
  - [x] `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx`
  - [x] `cd web/default && bun test src/features/wallet/components/employee-quota-request-card.test.tsx`
  - [x] `cd web/default && bun run typecheck`
  - [x] `cd web/default && bun run i18n:sync`
  - [x] 如果控件布局发生明显变化，启动 Default 前端并用浏览器/截图检查桌面与移动宽度下表格、输入切换和按钮文字不重叠。（本环境监听本地端口失败：`listen EPERM 127.0.0.1:4177`；已用 typecheck 与静态组件渲染测试覆盖。）

## Dev Notes

### 需求与业务上下文

- Epic 7B 是 Epic 7 / 7A 完成后的企业治理 follow-up，目标是修正治理通知解耦、预算池可辨识性、i18n 残留、一次性 wallet 展示语义，以及预算治理中的金额显示直观性。[Source: `_bmad-output/planning-artifacts/epics.md`#Epic 7B]
- Story 7B.5 明确要求在预算池创建、allocation、delegation、quota request 的额度输入中支持 quota / 金额视角切换；底层仍以 quota 存储和约束。[Source: `_bmad-output/planning-artifacts/epics.md`#Story 7B.5]
- 2026-06-03 correct-course 把“预算池、allocation、quota request 当前主要只展示 quota，不够直观；用户需要补充金额（$）视角，并允许额度输入表单切换单位”列为核心问题之一。[Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Core Problem]
- 架构 V1.3B 明确：企业预算治理继续以 quota 作为底层存储和约束单位；前端显示层支持 quota / 金额双视角；该能力优先作为显示层增强，不引入新的后端计费单位。[Source: `_bmad-output/planning-artifacts/architecture.md`#Data Flow item 11]

### 当前实现状态（已读 UPDATE 文件）

- `web/default/src/lib/format.ts`
  - 已提供 `formatQuota(quota)`、`parseQuotaFromDollars(amount)`、`quotaUnitsToDollars(units)`。
  - `formatQuota` 会走 `formatQuotaWithCurrency` 和系统 currency display 配置；这比手写 `$${quota / 500000}` 更符合钱包余额语义。
  - `parseQuotaFromDollars` 在 tokens-only/raw quota 模式下会返回 raw quota；在 currency/custom 模式下会按配置换回 quota。7B.5 必须复用这条路径。

- `web/default/src/lib/currency.ts`
  - 明确定义 currency display 规则：`formatQuotaWithCurrency` 用于 raw quota values，先把 tokens/quota 转为 USD，再按显示配置输出。
  - 文档里写明 `quotaPerUnit` 是 tokens per USD，默认配置来自 system config；不要在企业治理 feature 中复制配置读取逻辑。

- `web/default/src/features/wallet/constants.ts`
  - 存在 `QUOTA_PER_DOLLAR = 500000`，但 7B.5 更应复用 `lib/format.ts` 的高层 helper。只有 helper 无法满足测试时才考虑引用该常量，且不得新增重复常量。

- `web/default/src/features/enterprise-organization/index.tsx`
  - 预算池创建表单直接用 `Input inputMode='numeric'` 写 `total_quota` / `cycle_quota`。
  - 成员 allocation 创建表单直接用 `Input inputMode='numeric'` 写 `committed_quota`。
  - delegation 创建表单和 `BudgetDelegationTable` supersede draft 直接写 `committed_quota`。
  - 企业组织工作台 quota request 表单直接写 `requested_quota`；`QuotaRequestTable` 的 approve draft 直接写 `approvedQuota`。
  - `DepartmentBudgetListCard`、`DepartmentBudgetOverviewCard`、`DepartmentBudgetStatusCard`、`DepartmentBudgetDetailTable`、`QuotaAllocationTable`、`BudgetDelegationTable`、`QuotaRequestTable` 当前多数额度列只显示 `formatNumber(...)` 或裸数值。
  - `GovernanceActivityCard` 的 `Quota Change` 当前只显示 `formatNumber(item.quota_delta)`；它是治理审计视图，可作为补充范围但不要牺牲主表格可读性。

- `web/default/src/features/enterprise-organization/quota-request-budget-display.ts`
  - 7B.2 已建立 `getQuotaRequestBudgetDisplayText`，预算池选项和已选摘要复用它。
  - 当前 `remainingLabel` 是 `Remaining {{remaining}}`，只显示 raw quota 格式化数值；7B.5 应扩展这里，避免钱包入口和企业组织入口各写一套金额辅助文案。

- `web/default/src/features/enterprise-organization/quota-request-budget-display-components.tsx`
  - `QuotaRequestBudgetOption` / `QuotaRequestBudgetSummary` 是钱包入口和企业组织 quota request 表单共用组件。补金额显示时优先改这里和 display helper。

- `web/default/src/features/wallet/components/employee-quota-request-card.tsx`
  - 钱包入口 quota request 表单复用企业接口 `submitQuotaRequest`，payload 仍发送 `requested_quota`。
  - 当前 `requested_quota` 是单一 numeric input。7B.5 应让这里和企业组织工作台使用同一套单位切换控件/逻辑，避免员工入口与管理工作台行为不一致。

### Previous Story Intelligence

- Story 7B.4 刚完成企业 wallet 时间语义修正，改动面覆盖 `enterprise-organization/index.tsx`、`subscription-plans-card.tsx`、locale 和测试。7B.5 修改同一批企业组织文件时必须保留 7B.4 的 `cycle_type='never'`、`expires_at<=0`、`end_time<=0` 业务语义，不能重新引入 `No Reset`、epoch 时间或错误 expired 判断。[Source: `_bmad-output/implementation-artifacts/7b-4-normalize-one-time-wallet-and-non-positive-expiry-display.md`#Completion Notes List]
- Story 7B.3 已完成 quota request、allocation、delivery、wallet 相关 i18n 和未知 code fallback。7B.5 新增单位切换文案时必须继续覆盖 en/zh/fr/ru/ja/vi，未知 status/type/action 不得回退裸内部 code。[Source: `_bmad-output/implementation-artifacts/7b-3-complete-quota-request-governance-wallet-i18n-coverage.md`#Completion Notes List]
- Story 7B.2 建立预算池可辨识选项和已选摘要，相关代码集中在 `quota-request-budget-display.ts` 与 `quota-request-budget-display-components.tsx`。7B.5 应扩展这些 helper/component，而不是在钱包入口复制显示逻辑。[Source: `_bmad-output/implementation-artifacts/7b-2-redesign-budget-pool-selection-and-selected-summary.md`#Completion Notes List]
- Story 7B.1 解耦治理动作与通知投递状态。7B.5 不涉及 notification delivery，不要借金额显示修复去改治理通知、resend、alert rule 或 delivery status 语义。[Source: `_bmad-output/implementation-artifacts/7B-1-decouple-governance-action-result-from-notification-delivery.md`#Completion Notes List]

### Git Intelligence Summary

- 最近提交 `9a596fe0e feat(story-7B.4): 统一一次性钱包与非正过期时间展示语义` 修改了企业组织和钱包显示语义。开发前读当前文件，不要依据旧 story 猜测。
- 前序提交 `56b663567 feat(story-7B.3)`、`cf67bde3b feat(story-7B.2)`、`553d9ac0f feat(story-7B.1)` 分别覆盖 i18n、预算池选择、通知解耦；7B.5 不应回退这些实现。
- 当前工作树存在与本故事无关的未提交改动，主要在 `.agents/skills/bmad-story-automator/**`、`.claude/**`、`.codex/hooks.json` 和 story automator 输出。Dev agent 不得回退、删除或重写这些无关改动。

### Architecture Compliance

- 本故事是 Default 前端显示层和输入视角增强；不改 Go 后端数据库、migration、controller、service、DTO、relay、billing expression 或 quota request API contract。[Source: `_bmad-output/planning-artifacts/architecture.md`#Data Flow item 11]
- 所有提交到后端的字段仍是现有 quota 字段：`total_quota`、`cycle_quota`、`committed_quota`、`requested_quota`、`approved_quota`、`new_committed_quota`。金额视角只在前端转换为 quota 后提交。[Source: `web/default/src/features/enterprise-organization/types.ts`#DepartmentBudgetItem; #QuotaAllocationItem; #QuotaRequestItem; #BudgetDelegationItem]
- Default 前端继续使用 React 19、TypeScript、i18next、Base UI/shadcn-style primitives、Tailwind 和 lucide icons；不新增依赖。[Source: `_bmad-output/planning-artifacts/architecture.md`#Frontend Architecture; `web/default/package.json`]
- 前端 i18n 使用 `useTranslation()` / `t('English key')`，translation files 是 flat JSON：`web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`。[Source: `AGENTS.md`#Internationalization]
- 受保护标识 `new-api`、`QuantumNous` 不得删除、替换、改名或从版权/元数据/i18n sync literal allowlist 中移除。[Source: `AGENTS.md`#Rule 5]

### Library / Framework Requirements

- 不新增 currency、number-format、date、form 或 state management 依赖。
- 输入控件优先复用现有 `FormField`、`Input`、`Select`、`Button`、`Label` 或已有 UI primitive。单位切换可用 segmented control 风格按钮/选择器，但必须是稳定尺寸，不能造成表单行跳动。
- 金额显示优先用现有 `formatQuota`，金额输入转换用 `parseQuotaFromDollars` / `quotaUnitsToDollars`。不要手写 `quota / 500000` 作为主要实现。
- 所有新增 UI 文案走 `t(...)`；不要在 JSX 中直接写未翻译的中文或英文。

### File Structure Requirements

Expected UPDATE files:

- `web/default/src/features/enterprise-organization/index.tsx` - 预算池、allocation、delegation、quota request 表单和表格的单位切换、金额辅助显示。
- `web/default/src/features/enterprise-organization/quota-request-budget-display.ts` - 预算池选项/摘要的 remaining raw quota + 金额显示 helper。
- `web/default/src/features/enterprise-organization/quota-request-budget-display-components.tsx` - 预算池选项/摘要渲染金额辅助信息。
- `web/default/src/features/wallet/components/employee-quota-request-card.tsx` - 钱包入口 quota request 表单单位切换。
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx` - 企业组织预算治理金额视角测试。
- `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx` - 钱包入口单位切换与提交 payload 测试。
- `web/default/src/i18n/locales/{en,zh,fr,ja,ru,vi}.json` - 新增单位切换/金额辅助文案。

Possible NEW file:

- `web/default/src/features/enterprise-organization/quota-amount-display.tsx` 或 `quota-amount-input.tsx` - 如果多个表单/表格共享较多 JSX，新增 feature-local 小组件比把逻辑塞在 `index.tsx` 更清晰。

Likely no changes needed:

- `controller/enterprise/**`、`service/enterprise/**`、`dto/enterprise/**`、`model/enterprise/**` - 本故事不改变 API 或后端状态语义。
- `relay/**`、`pkg/billingexpr/**` - 不涉及 relay 计费或表达式系统。
- `web/classic/**` - 7B follow-up 来自 Default 前端钱包/治理显示问题；除非用户另行要求，不扩展 Classic。

### Project Structure Notes

- 推荐实现为“纯 helper + 小型输入组件 + 表格显示组件”：
  - `formatEnterpriseQuotaAmount(quota, t)` 返回 `{ quotaLabel, amountLabel, combinedLabel }`。
  - `QuotaAmountDisplay` 在主行展示 raw quota，辅助行展示 `formatQuota(quota)` 或 `Approx. {{amount}}`。
  - `QuotaAmountInput` 接收 quota 字符串和 setter，内部维护当前视角 `quota | amount`，切换时用 `quotaUnitsToDollars` / `parseQuotaFromDollars` 转换。
- 输入最终落入 react-hook-form 的字段仍必须是 quota 单位字符串/数字。不要把金额值写进 form 的 `total_quota`、`committed_quota` 等字段后直接提交。
- 金额小数输入需要明确 rounding：提交 quota 时用 `parseQuotaFromDollars` 的整数结果；显示视角可以保留合理小数，但不要生成 NaN、Infinity 或空字符串提交。
- `approved_quota = 0` 在表格展示中可能表示尚未批准；继续保留 `item.approved_quota || '-'` 的业务语义，但当有正数 approved quota 时应显示 raw quota + 金额。
- 不要全局改变 `formatNumber`，否则会影响所有数字表格。金额辅助只作用在企业治理 quota 相关位置。
- 表格列不要只把列名改成 `Amount` 后丢掉 quota；AC 明确“保留 quota 原值，同时补充金额显示”。
- 移动端/窄宽度下表格已有横向最小宽度压力。金额辅助信息建议用同一单元格的两行文本，而不是新增过多列。

### Testing Requirements

Minimum targeted validation:

```bash
cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx
cd web/default && bun test src/features/wallet/components/employee-quota-request-card.test.tsx
cd web/default && bun run typecheck
cd web/default && bun run i18n:sync
```

Recommended assertions:

- `formatEnterpriseQuotaAmount(1000000)` 或等价 helper 同时输出 raw quota `1,000,000` 和钱包金额/显示值，且不手写依赖 locale 的不可测字符串。
- 预算池列表和详情中 `remaining=1000000`、`allocated_total=500000`、`wallet.quota=250000` 同时渲染 raw quota 与金额辅助信息。
- allocation、delegation、quota request 表格中正数 quota 显示 raw quota + 金额；`approved_quota=0` 仍按未批准显示 `-`。
- quota / amount 输入切换后，提交 mutation payload 仍为 quota 单位。例如金额视角输入 `2` 时，payload 的 `requested_quota` 应等于 `parseQuotaFromDollars(2)` 的结果。
- 钱包入口 `EmployeeQuotaRequestCard` 与企业组织工作台 quota request 表单使用一致文案和换算逻辑。
- zh locale 下新增单位切换与金额辅助文案不回退英文源字符串；所有 supported locale 都有新增 key。
- i18n sync 报告新增 key 的 missingCount/extrasCount 为 0。

### Anti-Patterns to Avoid

- 不要新增后端字段、migration、API 参数或数据库金额列。
- 不要把金额视角当成新的计费单位；后端、事务、预算铁律和审批仍只认 quota。
- 不要在多个组件中复制 `quota / 500000`、`amount * 500000` 逻辑；复用 `parseQuotaFromDollars` / `quotaUnitsToDollars`。
- 不要把 `formatQuota` 全局替换成 raw number；钱包和其他模块已经依赖它的 currency display 语义。
- 不要只改预算池创建表单，遗漏 allocation、delegation、quota request 和审批 draft。
- 不要新增过多金额列导致表格拥挤；优先在同一 quota 单元格内增加辅助行。
- 不要只更新 `en.json`；必须同步 zh/fr/ru/ja/vi 并运行 i18n sync。
- 不要删除、改名或替换受保护的 `new-api`、`QuantumNous`、版权、包名和品牌信息。

### References

- [Source: `_bmad-output/planning-artifacts/epics.md`#Story 7B.5]
- [Source: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`#Core Problem]
- [Source: `_bmad-output/planning-artifacts/architecture.md`#Data Flow item 11]
- [Source: `web/default/src/lib/format.ts`#Quota Formatting]
- [Source: `web/default/src/lib/currency.ts`#Currency Formatting Library]
- [Source: `web/default/src/features/wallet/constants.ts`#QUOTA_PER_DOLLAR]
- [Source: `web/default/src/features/enterprise-organization/index.tsx`#budget/allocation/delegation/quota request forms]
- [Source: `web/default/src/features/enterprise-organization/quota-request-budget-display.ts`#getQuotaRequestBudgetDisplayText]
- [Source: `web/default/src/features/wallet/components/employee-quota-request-card.tsx`#requested_quota form]
- [Source: `_bmad-output/implementation-artifacts/7b-4-normalize-one-time-wallet-and-non-positive-expiry-display.md`#Previous Story Intelligence]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-03: BMAD create-story workflow executed for Story 7B.5 in YOLO mode.
- 2026-06-03: Read `.agents/skills/bmad-create-story/SKILL.md`, `discover-inputs.md`, `template.md`, and `checklist.md`.
- 2026-06-03: Tried `_bmad/scripts/resolve_customization.py --skill .agents/skills/bmad-create-story --key workflow`; local `python3` lacked Python 3.11 `tomllib`, so resolved customization manually from `.agents/skills/bmad-create-story/customize.toml`; no team/user overrides found.
- 2026-06-03: Loaded `_bmad/bmm/config.yaml`: `user_name=hth`, communication/document language Chinese, planning artifacts `_bmad-output/planning-artifacts`, implementation artifacts `_bmad-output/implementation-artifacts`.
- 2026-06-03: Persistent facts glob `file:{project-root}/**/project-context.md` found no files.
- 2026-06-03: Loaded sprint status and confirmed story key `7b-5-add-quota-amount-toggle-and-dollar-display-for-budget-governance` previous status `backlog`.
- 2026-06-03: Loaded relevant Epic 7B source from `_bmad-output/planning-artifacts/epics.md`, correct-course proposal `_bmad-output/planning-artifacts/sprint-change-proposal-2026-06-03.md`, architecture V1.3B data flow items, previous story 7B.4, git history, current enterprise organization and wallet quota request implementation files, tests, and quota/currency formatting helpers.
- 2026-06-03: Checklist validation applied: story includes exact AC, current-state notes for UPDATE files, previous story intelligence, i18n guardrails, protected identifier constraints, file boundaries, anti-patterns, and targeted tests.
- 2026-06-03: 执行 story-automator review workflow，读取 `bmad-story-automator-review` skill、workflow、instructions 和 checklist，按自动修复模式完成代码审查与修复。
- 2026-06-03: Review 发现并修复钱包入口未使用 quota/金额切换、金额视角新增文案未进入 locale、`BudgetStat` JSX 类型不匹配、治理 timeline 负向 quota_delta 被格式化为 0、story File List 未记录真实实现文件等问题。
- 2026-06-03: 执行 `cd web/default && bun run i18n:sync`，报告 missingCount/extrasCount 均为 0；ja/ru 存量 untranslatedCount 为 21，非本故事新增 key。
- 2026-06-03: 执行 `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx`，54 pass / 0 fail。
- 2026-06-03: 执行 `cd web/default && bun test src/features/wallet/components/employee-quota-request-card.test.tsx`，11 pass / 0 fail。
- 2026-06-03: 执行 `cd web/default && bun run typecheck`，通过。
- 2026-06-03: 执行 `git diff --check`，通过。
- 2026-06-03: 尝试 `cd web/default && bun run dev -- --host 127.0.0.1 --port 4177` 做浏览器检查，环境禁止监听本地端口，失败为 `listen EPERM 127.0.0.1:4177`。
- 2026-06-03: 再次执行 story-automator review workflow，复核 AC、任务、File List、i18n、表单输入和表格 action 区域；自动修复表格 `QuotaAmountInput` a11y prop、无效输入同步和窄列布局风险。
- 2026-06-03: 对当前未提交 follow-up patch 再次执行非交互 review，确认 diff 仅涉及 `quota-amount-controls.tsx`、`index.tsx`、`enterprise-organization.test.tsx` 的后续稳固修正；复验 `enterprise-organization` / `wallet` 测试、`typecheck` 与 `git diff --check`，未发现新增未修复问题。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- Story scope is intentionally Default frontend display/input-layer only; no backend API, database, relay, billing expression, quota request workflow, or notification delivery changes are required.
- Developer should reuse existing wallet quota/currency helpers instead of adding a new quota-to-dollar conversion path.
- 新增企业治理 quota/amount helper 与输入组件，所有提交字段仍保持 quota 单位。
- 预算池、派生 wallet、allocation、delegation、quota request、governance timeline、预算池选项/摘要均保留 raw quota 并补充钱包语义金额辅助显示。
- 钱包入口 `EmployeeQuotaRequestCard` 已与企业组织入口复用同一 quota/金额输入组件。
- i18n 已补齐 en/zh/fr/ru/ja/vi，新增文案不依赖 fallback。

### Senior Developer Review (AI)

#### Findings

1. **HIGH - 钱包入口 quota request 表单未实现金额视角切换。** `EmployeeQuotaRequestCard` 仍使用普通 numeric `Input`，不满足 AC1 对钱包入口 `requested_quota` 的要求。已改为复用 `QuotaAmountInput`，提交 payload 仍为 quota 单位。
2. **HIGH - 新增单位/金额文案未写入 locale。** `Quota view`、`Amount view`、`Approx. {{amount}}` 等 key 只出现在 JSX 中，非英文语言会回退英文源字符串，违反 i18n 任务。已补齐 en/zh/fr/ru/ja/vi 并运行 i18n sync。
3. **MEDIUM - `BudgetStat.value` 类型仍为 string。** 概览卡传入 `<QuotaAmountDisplay />` 后会导致 typecheck 失败。已将类型改为 `ReactNode`，badge 分支显式字符串化。
4. **MEDIUM - 负向治理额度变化被归零。** `formatEnterpriseQuotaAmount` 复用了非负输入 normalization，`GovernanceActivityCard` 的负向 `quota_delta` 会错误显示为 `0 quota`。已拆分显示 normalization，输入仍保持非负约束。
5. **MEDIUM - Story File List 与 git reality 不一致。** story 仅记录 story/sprint 文件，未记录实际前端实现、测试和 locale 文件。已补齐 File List。
6. **MEDIUM - 测试仍断言旧 raw-only 文案。** budget option/summary、企业表格和钱包入口测试未覆盖 raw quota + 金额辅助显示。已扩展 helper、企业组织、钱包入口和 zh locale 断言。
7. **MEDIUM - 表格 action 区域的额度输入没有实际 aria label。** `QuotaAllocationTable`、`BudgetDelegationTable`、`QuotaRequestTable` 使用了 `aria-label` prop，但 `QuotaAmountInput` 只接收 `ariaLabel`，导致内部 input 缺少可访问名称。已改为 `ariaLabel` 并增加 SSR 断言。
8. **MEDIUM - 金额视角无效输入可能和实际提交 quota 脱节。** 金额模式下输入非数字或负数时组件只更新可见 draft，不同步 form 字段，用户可能看到无效值但提交旧 quota。已让无效输入同步进入 form，由现有 zod 正整数校验阻止提交。
9. **LOW - 表格内单位切换控件和按钮在窄列中容易挤压。** active allocation/delegation supersede 与 approve draft 的 action 行原本单行排列，金额辅助文案会增加宽度压力。已改为可换行布局并固定输入控件宽度。

#### Review Outcome

Approve after automatic fixes. No CRITICAL issues remain.

#### Verification Rerun

- 2026-06-03: 针对后续未提交修补再次执行 story-automator review；当前剩余 diff 只包含金额模式无效输入同步、表格 action 区 `ariaLabel` 透传修正和窄列换行稳固。
- 复核 AC1 / AC2 涉及的预算池创建、allocation、delegation、quota request、审批 draft、预算池摘要、wallet/detail/table/timeline 等显示与输入面，未发现新的 CRITICAL / HIGH / MEDIUM 问题。
- 复验结果：`cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx` 54 pass，`cd web/default && bun test src/features/wallet/components/employee-quota-request-card.test.tsx` 11 pass，`cd web/default && bun run typecheck` 通过，`git diff --check` 通过。

### File List

- `_bmad-output/implementation-artifacts/7b-5-add-quota-amount-toggle-and-dollar-display-for-budget-governance.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/quota-amount-controls.tsx`
- `web/default/src/features/enterprise-organization/quota-request-budget-display.ts`
- `web/default/src/features/enterprise-organization/quota-request-budget-display-components.tsx`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/features/wallet/components/employee-quota-request-card.tsx`
- `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/vi.json`

## Change Log

- 2026-06-03: Story 7B.5 implemented and reviewed; added quota/amount input toggle, raw quota + amount display, wallet-entry parity, i18n resources, regression coverage, and sprint-status sync.
- 2026-06-03: Story 7B.5 senior review rerun; fixed quota amount input a11y prop forwarding, invalid amount draft synchronization, and action-cell wrapping stability.
- 2026-06-03: Story 7B.5 verification rerun confirmed the follow-up patch is review-clean; story remains `done` with no further action items.
