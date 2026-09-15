---
status: approved
project: new-api
created: 2026-06-03
workflow: bmad-correct-course
mode: batch
trigger_source: _bmad-output/implementation-artifacts/investigations/enterprise-governance-quota-request-followup-investigation.md
related_proposals:
  - _bmad-output/planning-artifacts/sprint-change-proposal-2026-06-02-enterprise-governance-followup-cc.md
recommended_scope: moderate
selected_path: direct-adjustment
approved: 2026-06-03
approved_by: hth
proposed_changes_to:
  - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md
  - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md
  - _bmad-output/planning-artifacts/epics.md
  - _bmad-output/planning-artifacts/architecture.md
  - _bmad-output/implementation-artifacts/sprint-status.yaml
---

# Sprint Change Proposal: 企业治理额度申请后续排障与体验修正

## 1. Issue Summary

### Trigger

本次纠偏由 2026-06-03 的专项排障结论触发，输入文档为：

- `_bmad-output/implementation-artifacts/investigations/enterprise-governance-quota-request-followup-investigation.md`

排障结论确认，这不是“审批主链路失败”，而是 Epic 7 / Epic 7A 完成后，企业治理额度申请闭环在真实使用下暴露出新的产品耦合、展示语义和体验完整性问题。

### Triggering Story Context

本次问题主要落在以下既有故事的后续质量范围内：

- `Story 7.5` 支持员工额度申请、单步审批与自动分配
- `Story 7.6` 建立治理动作审计时间线与钉钉通知
- `Story 7A.1` 补齐企业治理工作台 i18n 与动作标签映射
- `Story 7A.2` 将员工额度申请入口迁移到余额 / 钱包用户场景
- `Story 7A.3` 放宽预算池类型约束至单池不可变

### Core Problem

问题类型归类：

- 技术实现耦合导致的产品误导
- 已交付功能的体验完整性缺口
- 已修正 follow-up 的覆盖边界不足

清晰问题陈述：

1. quota request 审批成功与治理通知投递失败目前没有被产品语义分离，导致“通知配置缺失/投递失败”会被用户误读为“审批失败”。
2. quota request 的目标预算池选择虽然已经迁移到钱包入口，但选项信息仍然过弱，用户难以确认自己选中了哪个预算池。
3. 企业治理与钱包相关界面仍有一批 quota request / allocation / delivery 文案未完全 i18n 化，说明 7A.1 的修正范围不足以覆盖当前模块。
4. 一次性企业钱包与非正过期时间的展示语义没有产品化，`cycle_type = never`、`end_time <= 0`、`expires_at <= 0` 仍会落入通用时间格式化逻辑，产生误导。
5. 预算池、allocation、quota request 等界面当前主要只展示 quota 数值，不够直观；用户需要补充金额（`$`）视角，并允许在额度输入表单里切换单位。

### Evidence

- 审批主链路成功：`enterprise_quota_requests.status = fulfilled`，并已创建 `enterprise_quota_allocations`。
- 可见失败来自治理通知投递：`enterprise_governance_notification_deliveries.status = final_failed`，原因为 `enterprise alert rule not found`。
- 通知通道当前仍依赖 alert rule：`service/enterprise/governance_notification_dispatch.go`
- quota request 预算池选项与已选摘要信息不足：
  - `web/default/src/features/wallet/components/employee-quota-request-card.tsx`
  - `web/default/src/features/enterprise-organization/index.tsx`
- 一次性钱包与非正时间显示问题：
  - `web/default/src/features/wallet/components/subscription-plans-card.tsx`
  - `web/default/src/features/enterprise-organization/index.tsx`

## 2. Impact Analysis

### Checklist Summary

- `1.1` Triggering story identified: `[x] Done`
- `1.2` Core problem defined: `[x] Done`
- `1.3` Evidence collected: `[x] Done`
- `2.1` Current epic still viable: `[x] Done`
- `2.2` Epic-level changes needed: `[x] Done`
- `2.3` Remaining epics reviewed: `[x] Done`
- `2.4` New epic required: `[x] Done`
- `2.5` Priority resequencing needed: `[x] Done`
- `3.1` PRD conflict checked: `[x] Done`
- `3.2` Architecture conflict checked: `[x] Done`
- `3.3` UI/UX impact checked: `[x] Done`
- `3.4` Secondary artifacts checked: `[x] Done`
- `4.1` Direct adjustment evaluated: `[x] Viable`
- `4.2` Rollback evaluated: `[x] Not viable`
- `4.3` MVP review evaluated: `[x] Not viable`
- `4.4` Recommended path selected: `[x] Done`
- `6.4` `sprint-status.yaml` update: `[!] Action-needed after approval`

### Epic Impact

对当前触发 epic 的判断：

- `Epic 7` 保持 `done`，不建议回滚。
- `Epic 7A` 保持 `done`，不建议重写已验收历史。
- 本次问题说明 `Epic 7A` 已解决“入口迁移 / 部分 i18n / 类型约束”第一轮纠偏，但没有覆盖“通知语义解耦、预算池可辨识性、一次性钱包展示语义”的第二轮质量缺口。

建议的 epic 级变更：

- 新增 `Epic 7B`，而不是重开 `Epic 7A`。

原因：

- `7A` 已经完成并写入 `sprint-status.yaml`，重开会混淆历史完成定义。
- 本次问题是新的 follow-up 范围，不是对 7A 原始结论的否定。
- 以 `7B` 承接更利于 backlog、验收和回顾拆分。

### PRD Impact

PRD 当前与本次结论的冲突/缺口主要有 4 处：

1. `FR-25` 只要求“显式选择目标预算池”，但没有要求预算池选项必须可辨识，也没有要求选中后显示已选摘要。
2. `FR-26` 虽然写了“通知失败不会阻塞核心治理事务”，但没有要求“治理动作成功”和“通知投递状态”在产品展示上显式分离。
3. 派生 wallet / budget detail 的一次性与非正过期时间语义没有被要求为业务语义展示，导致当前实现容易退回通用日期格式化。
4. 预算池和分配相关流程没有要求支持“额度 / 金额”双视角展示，因此当前 UI 只能看到 quota，不能直观看到与钱包一致的金额语义。

### Architecture Impact

架构主方向保持成立，不需要重写，但需要补充 4 条明确边界：

1. 治理动作状态与通知投递状态是两个独立状态面，不应共用“失败”语义。
2. quota request 表单不只要“能选预算池”，还要“可识别预算池”。
3. 企业派生 wallet 的“无周期 / 未过期 / 未设置过期时间”必须在显示层单独建模，不能落回 `new Date(0)` 风格逻辑。
4. 预算治理相关前端显示应支持 quota 与金额（$）双视角，但底层后端仍保持 quota 作为存储和约束单位。

### UI/UX Impact

本次影响显著且集中在 4 个界面：

- 钱包 quota request 入口表单
- 企业组织工作台 quota request 表单
- 企业治理时间线 / 通知投递区域
- 钱包卡片与预算池派生 wallet 列表

当前仓库没有单独的新 `new-api` UX 规格文档，因此本次 UX 变更应先通过 `epics.md` 与 `architecture.md` 固化。

### Secondary Artifact Impact

本次 follow-up 还会影响：

- `web/default/src/i18n/locales/*.json`
- quota request / governance / wallet 相关前端测试
- `sprint-status.yaml` 中的新 epic / story 条目
- 预算池/分配/申请表单与列表的显示层状态和列定义

## 3. Recommended Approach

### Option Evaluation

#### Option 1: Direct Adjustment

- 结论：`Viable`
- Effort：`Medium`
- Risk：`Medium`

说明：

- 不需要回滚已完成能力。
- 可以通过“补充规划工件 + 新增 follow-up epic/story + 实施修复”完成。
- 不触碰 relay / billing expression 主链路。

#### Option 2: Potential Rollback

- 结论：`Not viable`
- Effort：`High`
- Risk：`High`

说明：

- 审批与分配主链路是正确的，问题不在需要回退的基础方案，而在通知耦合和展示层语义。

#### Option 3: PRD MVP Review

- 结论：`Not viable`
- Effort：`Medium`
- Risk：`Low`

说明：

- 这不是 MVP 目标错误，也不需要缩减范围。
- 当前问题属于现有 MVP 范围内的语义补完和可靠性修正。

### Selected Path

选择：`Direct Adjustment`

执行方式：

1. 在规划层追加 `Epic 7B`，承接本次 follow-up。
2. 在 PRD / Addendum / Architecture 中补齐“通知解耦、预算池可辨识性、一次性 wallet 语义、额度/金额双视角”。
3. 后续由 Product Owner / Developer 按新 stories 实施。

### Scope Classification

`Moderate`

原因：

- 需要新增 epic / stories
- 需要修改 PRD / addendum / architecture
- 涉及后端 dispatch 语义、前端多个界面、i18n 和测试
- 但不需要根本性重规划，也不需要回滚既有 epic

## 4. Detailed Change Proposals

### 4.1 PRD 修改提案

Artifact:
`_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`

#### Change A: 补强 FR-25 的预算池可辨识性要求

Section:
`FR-25 / 可验证结果`

OLD:

```markdown
- 员工提交申请时必须显式选择目标部门和目标预算池模式/池子，避免多部门成员关系下的审批归属歧义。
```

NEW:

```markdown
- 员工提交申请时必须显式选择目标部门和目标预算池；预算池选项必须至少同时展示部门名、预算池标识或名称、预算类型、当前可分配/剩余额度与状态，避免多部门成员关系下的审批归属歧义和误选。
- 员工完成选择后，界面必须显示“已选请求范围/已选预算池摘要”，让提交前可再次确认目标预算池。
```

Rationale:
当前问题不是“没有预算池选择”，而是“预算池虽然可选但不可辨识”。

#### Change B: 补强 FR-26 的治理动作与通知投递分离语义

Section:
`FR-26 / 可验证结果`

OLD:

```markdown
- 通知失败不会阻塞核心治理事务提交，但必须记录投递状态和错误原因。
```

NEW:

```markdown
- 治理动作结果与通知投递结果必须独立记录和独立展示。审批通过、自动分配成功等核心治理动作，不得因为通知未配置或通知投递失败而在产品上被呈现为治理动作失败。
- 通知失败不会阻塞核心治理事务提交，但必须记录投递状态和错误原因；“未配置治理通知通道/无可用通道”必须与“投递失败”区分。
```

Rationale:
当前最大的误导是把通知失败显示成审批失败。

#### Change C: 补强派生 wallet 的一次性 / 非正过期时间语义

Section:
`FR-12 / 部门预算池及派生 wallet 状态展示`

OLD:

```markdown
（当前没有对 cycle_type = never、end_time <= 0、expires_at <= 0 的展示语义给出明确要求。）
```

NEW:

```markdown
- 派生 wallet 与预算详情视图必须区分“周期型重置”“一次性额度”“永不过期”与“未设置过期时间”等业务语义。
- 对于 `cycle_type = never`、`end_time <= 0`、`expires_at <= 0` 的场景，界面不得显示 epoch 时间、不得默认渲染为已过期，必须先映射为业务语义后再展示。
```

Rationale:
这是本次用户直观看到的第二类误导。

#### Change D: 补强预算池与额度流程的单位/金额双视角要求

Section:
`FR-9 / FR-12 / FR-25 相关显示与输入要求`

OLD:

```markdown
（当前只要求展示 quota 数值，没有要求表单支持单位切换，也没有要求预算池列表/分配记录同步显示金额视角。）
```

NEW:

```markdown
- 预算池创建、成员 allocation、预算委派、额度申请等额度输入表单，应支持在“额度（quota）”与“金额（$）”视角之间切换；该切换首先是显示/输入层能力，不改变后端底层 quota 存储模型。
- 预算池列表、预算池详情、成员 allocation 记录、额度申请记录和相关治理记录，应在保留原 quota 数值的同时补充金额（$）显示，便于用户快速理解。
- 如果当前上下文缺少可靠金额换算信息，则界面至少保留 quota 原值，并对金额显示做显式兜底，而不是隐藏原 quota。
```

Rationale:
这是轻量增强，不改底层模型，但能显著提升企业治理界面的可理解性。

### 4.2 Addendum 修改提案

Artifact:
`_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md`

#### Change E: 追加 V1.3B 纠偏说明

Section:
`新增 V1.3B 修订`

OLD:

```markdown
（当前没有 2026-06-03 这轮 follow-up 说明。）
```

NEW:

```markdown
V1.3B (2026-06-03): 根据 enterprise governance quota request follow-up investigation 追加纠偏。
- 明确治理动作结果与通知投递结果必须分离展示，未配置治理通知通道不应被显示为审批失败。
- 明确 quota request 预算池选择必须可辨识，提交前必须显示已选预算池摘要。
- 明确一次性企业 wallet 与非正过期时间必须按业务语义展示，不得退回通用 epoch/expired 文案。
- 明确预算池与额度相关表单支持 quota / 金额双视角切换，预算池列表和分配记录补充金额显示，但不改变底层 quota 语义。
```

Rationale:
把本次 follow-up 与 2026-06-02 的 7A 纠偏区分开，避免语义混在一起。

### 4.3 Epics 修改提案

Artifact:
`_bmad-output/planning-artifacts/epics.md`

#### Change F: 在 Epic 7A 后追加新 Epic 7B，而不是重开 7A

Section:
`Epic 7A 之后新增 Epic 7B`

OLD:

```markdown
Epic 7A 是当前最后一个企业治理 follow-up epic。
```

NEW:

```markdown
## Epic 7B: 企业治理通知解耦、钱包语义与额度显示修正

在 Epic 7 与 Epic 7A 完成后，团队继续修正企业治理额度申请闭环中的语义耦合、预算池可辨识性、i18n 残留、一次性 wallet 展示问题，以及预算治理中的金额显示直观性，确保审批成功不会再被通知状态误导，且钱包/预算池展示符合业务语义。
```

Rationale:
保留 7A 的完成历史，同时为本次新增问题建立独立 backlog。

#### Change G: 新增 Story 7B.1

OLD:

```markdown
（无对应后续 story。）
```

NEW:

```markdown
### Story 7B.1: 解耦治理动作结果与通知投递结果

**Requirements:** follow-up to FR26

As a 管理员或负责人,
I want 审批/分配结果与治理通知投递状态在后端语义和前端展示上独立存在,
So that 我不会把“未配置通知通道”或“投递失败”误判为“审批失败”。

**Acceptance Criteria:**

**Given** quota request 审批已成功并已创建 allocation
**When** 系统没有可用治理通知通道或相关 alert rule
**Then** 请求与 allocation 结果仍显示成功
**And** 通知区域显示“未配置/未投递/待配置”一类独立状态，而不是 `final_failed` 误导业务结论。

**Given** 通知投递真实失败
**When** 前端展示治理时间线
**Then** 审批动作结果与投递结果分别可见
**And** 错误原因仅归属通知投递记录，不覆盖治理动作结论。
```

#### Change H: 新增 Story 7B.2

OLD:

```markdown
（无对应后续 story。）
```

NEW:

```markdown
### Story 7B.2: 重构额度申请预算池选择与已选摘要

**Requirements:** follow-up to FR25

As a 普通员工或负责人,
I want 在 quota request 表单里清楚区分各个可选预算池，并在提交前再次确认已选目标,
So that 我不会因为预算池名称过弱或信息缺失而误选。

**Acceptance Criteria:**

**Given** 用户打开钱包 quota request 表单或企业组织工作台 quota request 表单
**When** 系统展示预算池选项
**Then** 每个选项至少显示部门名、预算池标识或名称、预算类型、剩余额度与状态
**And** 不再只有近似的单行短标签。

**Given** 用户已选择预算池
**When** 表单进入提交前状态
**Then** 页面显示“已选请求范围/已选预算池摘要”
**And** 摘要中包含部门和具体预算池身份，而不是只显示部门。
```

#### Change I: 新增 Story 7B.3

OLD:

```markdown
（无对应后续 story。）
```

NEW:

```markdown
### Story 7B.3: 完成额度申请与治理通知相关 i18n 收尾

**Requirements:** follow-up to FR25, FR26, NFR8

As a 非英文语言环境下的管理员或员工,
I want quota request、allocation、delivery 状态、动作标签和表头完整本地化,
So that 我不会看到英文残留或回退到内部 key。

**Acceptance Criteria:**

**Given** 用户查看 quota request、allocation、timeline、delivery 和 wallet 相关界面
**When** 当前语言不是英文
**Then** 截图可见字段全部存在 locale 资源
**And** action/status/delivery 文案都通过稳定映射层输出，不依赖遗漏的英文 source key。
```

#### Change J: 新增 Story 7B.4

OLD:

```markdown
（无对应后续 story。）
```

NEW:

```markdown
### Story 7B.4: 统一一次性企业 wallet 与非正过期时间展示语义

**Requirements:** follow-up to FR12, FR25

As a 管理员或员工,
I want 一次性企业 wallet 和无过期时间场景用业务语义展示,
So that 我不会看到 1970 时间、错误的“已过期”判断或过于技术化的文案。

**Acceptance Criteria:**

**Given** 企业派生 wallet 的 `cycle_type = never`
**When** 页面展示周期信息
**Then** 使用“一次性额度”或等价业务文案
**And** 不再仅显示 `No Reset`。

**Given** `end_time <= 0` 或 `expires_at <= 0`
**When** 页面展示过期信息
**Then** 不得格式化为 epoch 时间
**And** 必须展示“永不过期”或“未设置过期时间”等业务语义，并避免误判为已过期。
```

#### Change K: 新增 Story 7B.5

OLD:

```markdown
（无对应后续 story。）
```

NEW:

```markdown
### Story 7B.5: 为预算池与分配记录补充金额视角和单位切换

**Requirements:** follow-up to FR9, FR12, FR25

As a 管理员或员工,
I want 在额度表单里切换“额度 / 金额”视角，并在预算池与分配记录中同时看到 quota 和金额,
So that 我能用更直观的金额视角理解预算和钱包额度，同时保留原 quota 语义。

**Acceptance Criteria:**

**Given** 管理员在创建预算池、创建 allocation、创建 delegation，或员工提交 quota request
**When** 填写额度字段
**Then** 表单支持在 quota 与金额（$）视角之间切换
**And** 该切换不改变后端底层 quota 存储模型。

**Given** 用户查看预算池列表、预算池详情、allocation 记录或 quota request 记录
**When** 页面展示额度相关列
**Then** 在保留 quota 原值的同时补充金额（$）显示
**And** 显示方式尽量与钱包额度语义保持一致。
```

### 4.4 Architecture 修改提案

Artifact:
`_bmad-output/planning-artifacts/architecture.md`

#### Change L: 扩写“额度申请审批流（V1.3）”

Section:
`额度申请审批流（V1.3）`

OLD:

```markdown
员工从余额 / 钱包相关 UI 发起额度申请 → `POST /api/enterprise/quota-requests` → 记录目标部门、目标预算池模式/池子、申请额度、申请原因与幂等键 → 系统按目标部门的有效负责人集合路由单步审批 → 审批通过后调用既有 `quota_allocation.go` 完成自动分配 → 写 request / approval / fulfillment 审计与通知状态。重复审批或重复回调不得产生重复分配。企业组织工作台负责审批、治理时间线与通知状态展示。
```

NEW:

```markdown
员工从余额 / 钱包相关 UI 发起额度申请 → quota request 表单加载“可辨识预算池选项”（部门名、预算池标识或名称、预算类型、剩余额度、状态）并在选中后显示已选预算池摘要 → `POST /api/enterprise/quota-requests` → 记录目标部门、目标预算池、申请额度、申请原因与幂等键 → 系统按目标部门的有效负责人集合路由单步审批 → 审批通过后调用既有 `quota_allocation.go` 完成自动分配 → 写 request / approval / fulfillment 审计与通知状态。重复审批或重复回调不得产生重复分配。企业组织工作台负责审批；治理动作结果与通知投递状态必须独立展示。
```

Rationale:
把“可辨识预算池”和“治理动作/通知状态分离”两项补到架构级行为约束。

#### Change M: 新增“治理通知投递语义（V1.3B）”

OLD:

```markdown
（当前没有独立规则显式约束“无通知通道配置”与“投递失败”的语义区别。）
```

NEW:

```markdown
9. **治理通知投递语义（V1.3B）**：治理通知投递是核心治理事务之后的旁路动作。审批成功、allocation 创建成功等治理结果必须保持独立事实；当系统没有可用治理通知通道或缺少所需配置时，应记录为“未配置/跳过/待配置”一类非阻塞状态，而不是把治理动作标记成失败。真实投递失败仍需可重试、可追踪，并与治理动作结果分层展示。
```

#### Change N: 新增“企业 wallet 显示语义（V1.3B）”

OLD:

```markdown
（当前没有独立规则显式约束 `cycle_type = never` 与 `end_time/expires_at <= 0` 的显示语义。）
```

NEW:

```markdown
10. **企业 wallet 显示语义（V1.3B）**：企业派生 wallet 的显示层必须先解释业务语义再格式化时间。`cycle_type = never` 表示一次性/无周期重置；`end_time <= 0`、`expires_at <= 0` 代表永不过期或未设置过期时间的业务状态，不得直接传入日期格式化逻辑，也不得默认标记为已过期。
```

#### Change O: 新增“额度单位与金额视角（V1.3B）”

OLD:

```markdown
（当前没有独立规则显式约束预算池、allocation、quota request 的金额显示视角与单位切换。）
```

NEW:

```markdown
11. **额度单位与金额视角（V1.3B）**：企业预算治理继续以 quota 作为底层存储和约束单位，但前端显示层应支持 quota / 金额（$）双视角。预算池创建、分配、委派、额度申请表单允许用户切换输入/理解视角；预算池列表、allocation 记录、quota request 记录和相关详情在保留 quota 原值的同时补充金额显示。该能力优先作为显示层增强，不引入新的后端计费单位。
```

### 4.5 UI/UX 变更说明

当前没有单独的 `new-api` UX 规格文件，因此本次作为临时 UX delta 记录：

- 钱包 quota request 卡片：
  - 预算池下拉从“弱标签”改为“富选项行”
  - 选中后增加预算池摘要卡
- 企业组织工作台 quota request 区：
  - 与钱包入口共用同一预算池辨识模型
  - 保留治理审批处理能力，不再承担“唯一用户入口”角色
- 治理时间线 / delivery 区：
  - 同一 trace 下分离“业务动作结果”和“通知投递结果”
  - “未配置通道”与“投递失败”区别呈现
- 钱包与派生 wallet 列表：
  - `cycle_type = never` 显示为业务文案
  - 非正过期时间不再渲染 epoch / expired 误导
- 预算池与额度相关表单/列表：
  - 表单支持 quota / 金额双视角切换
  - 列表和记录保留 quota，同时补金额（$）显示

## 5. Implementation Handoff

### Scope

`Moderate`

### Handoff Recipients

- `Product Owner / Product Lead`
  - 确认 `Epic 7B` 是否按本提案立项
  - 确认“未配置治理通知通道”的最终产品文案和状态命名
  - 确认一次性 wallet 的最终中文业务文案
  - 确认金额视角采用“表单切换 + 列表双显示”的最小实现，不引入新的底层单位

- `Developer`
  - 实现 `Story 7B.1` 到 `Story 7B.5`
  - 为治理 dispatch、quota request 表单、wallet 显示语义补回归测试
  - 完成相关 locale 资源和前端状态映射收尾

### Success Criteria

1. quota request 审批成功时，即使没有治理通知通道，也不会再被 UI 呈现为审批失败。
2. quota request 的预算池选择在钱包入口和企业组织工作台入口都具备可辨识性，且提交前能看到已选摘要。
3. 本次截图可见的 quota request / allocation / delivery 英文残留全部完成 i18n。
4. 一次性企业 wallet 与 `end_time <= 0` / `expires_at <= 0` 的场景不再出现 epoch 时间或错误过期判断。
5. 预算池、allocation 和 quota request 相关界面都能在保留 quota 的同时显示金额（$），且额度表单支持单位切换。

### Suggested PR Split

1. `fix: separate governance action success from notification delivery state`
2. `fix: improve quota request budget-pool selection and governance i18n coverage`
3. `fix: normalize enterprise one-time wallet and non-positive expiry semantics`
4. `feat: add quota amount toggle and dollar display for budget governance`

## 6. Sprint Status Update Plan

批准后应追加到：
`_bmad-output/implementation-artifacts/sprint-status.yaml`

建议新增：

```yaml
  epic-7b: backlog
  7b-1-decouple-governance-action-result-from-notification-delivery: backlog
  7b-2-redesign-budget-pool-selection-and-selected-summary: backlog
  7b-3-complete-quota-request-governance-wallet-i18n-coverage: backlog
  7b-4-normalize-one-time-wallet-and-non-positive-expiry-display: backlog
  7b-5-add-quota-amount-toggle-and-dollar-display-for-budget-governance: backlog
  epic-7b-retrospective: optional
```

## 7. Final Recommendation

本次建议正式归类为：

- `Correct Course`
- `Direct Adjustment`
- `Moderate scope`

明确结论：

- 不回滚 `Epic 7` 或 `Epic 7A`
- 不重做 MVP 范围
- 通过新增 `Epic 7B` 承接本次 follow-up
- 先更新规划工件，再进入实现

## 8. Approval Request

提案文档已生成，但尚未进入 approved 状态。

批准后下一步应执行：

1. 更新 `epics.md`
2. 更新 `architecture.md` / `prd.md` / `addendum.md`
3. 更新 `sprint-status.yaml`
4. 创建 `Epic 7B` 对应 story 文件并进入实现
