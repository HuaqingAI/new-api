---
status: approved
project: new-api
created: 2026-06-02
workflow: bmad-correct-course
mode: batch
trigger_source: Epic 7 post-implementation product validation and bug review
recommended_scope: moderate
selected_path: hybrid
approved: 2026-06-02
approved_by: hth
handoff_scope: moderate
handoff_recipients:
  - Product Owner / Product Lead
  - Developer
proposed_changes_to:
  - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md
  - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md
  - _bmad-output/planning-artifacts/epics.md
  - _bmad-output/planning-artifacts/architecture.md
---

# Sprint Change Proposal: Epic 7 企业治理后续纠偏

## 1. Issue Summary

### Trigger

本次变更由 Epic 7 完成后的本地产品验证触发，不是开发阶段阻塞，也不是底层架构失效。

触发问题主要来自企业组织页中的治理闭环交付在真实使用视角下暴露出两类问题：

1. **明确缺陷（Bug）**
   - 企业组织页中 `Governance Timeline and Notification Delivery`、`Employee Quota Requests`、`Action` 等治理相关标题、表头和内容存在 i18n / label 映射缺失或不完整的现象，导致页面局部仍显示英文或内部动作标识。

2. **需求与产品实现偏移（Change）**
   - 员工额度申请虽然从后端权限模型上支持普通成员提交，但前端主入口被放在 `Enterprise Organization` 管理工作台内，不符合普通用户的使用心智。
   - 预算池虽然设计上保持“单预算池单类型（balance / subscription 二选一）”，但当前实现额外施加了“同一部门后续所有预算池类型必须与第一个预算池一致”的约束，与“预算池单类型”相比收紧过度。

### Triggering Story Context

- 触发能力主要来自 Epic 7 的 Story 7.5（员工额度申请）与 Story 7.6（治理时间线与通知）。
- 同时关联 Story 3.x / 7.3 的预算池与预算委派模型，因为预算池类型约束是通过现有 `DepartmentBudgetService` 继承下来的。

### Core Problem

问题类型：**已交付能力的产品化入口修正 + 需求边界澄清 + 局部前端缺陷修复**

清晰问题陈述：

1. 当前企业治理工作台中的部分治理模块未完全产品化，存在可见文案和 label 映射缺口。
2. 员工额度申请被实现为“治理工作台中的一个卡片”，而不是“普通用户围绕余额不足的自助动作”，导致入口设计不合理。
3. 当前预算池模型把“预算池单类型”进一步实现成“部门预算池类型不可变”，这比需求原意更严格，限制了一个部门同时拥有一次性池和周期池的可能性。

### Evidence

#### Bug Evidence

- 前端治理区域标题与表格位于：
  - `web/default/src/features/enterprise-organization/index.tsx:2957`
  - `web/default/src/features/enterprise-organization/index.tsx:3523`
  - `web/default/src/features/enterprise-organization/index.tsx:3550`
- `Action` 列内容通过 `governanceActionLabel(item.action_type, t)` 映射，若映射不全会回退到原始 action key：
  - `web/default/src/features/enterprise-organization/index.tsx:3571`
  - `web/default/src/features/enterprise-organization/index.tsx:3685`

#### Product / UX Evidence

- 企业组织模块作为管理工作台入口定义于：
  - `web/default/src/hooks/use-sidebar-data.ts:138`
- 员工额度申请 UI 目前被直接放在企业组织工作区卡片中：
  - `web/default/src/features/enterprise-organization/index.tsx:2955`
- 文档对企业能力承载位置的描述是“企业组织模块内部承载治理能力”，但没有要求员工申请必须通过管理工作台入口：
  - `_bmad-output/planning-artifacts/architecture.md:384`
  - `_bmad-output/planning-artifacts/epics.md:133`

#### Requirement Drift Evidence

- 当前预算池创建逻辑中，如果部门已有预算池，则后续预算池类型不能与现有类型不同：
  - `service/enterprise/department_budget.go:143-150`
- 该逻辑返回 `ErrDepartmentBudgetTypeImmutable`，错误名本身就体现了“部门级预算池类型不可变”：
  - `service/enterprise/errors.go`
- PRD 原文仅明确“部门预算池类型一经创建不可在 balance 与 subscription 间切换；如需变更，必须先停用旧池并新建”，它约束的是**单个预算池的类型切换**，而不是**一个部门下所有预算池都必须同类型**：
  - `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md:249`

## 2. Impact Analysis

### Checklist Summary

- `1.1 Triggering story`：`[x] Done`
- `1.2 Core problem`：`[x] Done`
- `1.3 Evidence`：`[x] Done`
- `2.1~2.5 Epic impact`：`[x] Done`
- `3.1~3.4 Artifact impact`：`[x] Done`
- `4.1~4.4 Path forward`：`[x] Done`

### Epic Impact

本次变更不推翻 Epic 7 已完成定义，但会影响 Epic 7 follow-up 的收尾质量和后续治理能力的产品化方向。

#### 对已完成 Epic 的影响

- **Epic 7 保持 done**
  - 原有故事完成定义仍成立：负责人权限、descendant scope、预算委派、allocation 治理、quota request、governance timeline 都已落地。
  - 本次问题不要求回滚 Epic 7。

#### 对后续规划的影响

需要补一个 **Epic 7 follow-up / Correct Course** 层级的后续工作，范围至少包括：

1. 企业治理工作台 i18n / label 完整性修复。
2. 员工额度申请入口从“管理工作台内部卡片”调整为“普通用户余额场景入口”。
3. 部门预算池类型约束从“部门级不可变”收敛回“单预算池单类型，但同部门允许多个不同类型预算池并存”。

### PRD Impact

PRD 当前对以下两点不够明确，因此需要增量补充：

1. **员工额度申请的前端入口语义**
   - 当前 PRD 强调员工可在所属部门上下文中申请额度，但没有明确其是否必须位于企业组织管理工作台内。
   - 建议在 PRD 或 addendum 中补清：员工入口应贴近余额/钱包场景，而不是依赖管理工作台菜单。

2. **预算池类型约束粒度**
   - 当前 PRD 对“预算池类型不可切换”的约束更像是单个预算池生命周期约束。
   - 需要明确写出：一个部门是否允许多个不同类型预算池并存。
   - 根据本次产品结论，建议明确为：**允许并存；禁止同一预算池实例跨类型切换**。

### Architecture Impact

架构不需要重写，但需要补充两个明确规则：

1. **员工入口与治理工作台分层**
   - 治理审批、时间线、通知等仍放企业组织工作台。
   - 员工申请入口从用户自己的余额/钱包上下文发起，走相同 quota request API。

2. **预算池类型约束边界**
   - `DepartmentBudget.Type` 继续保留单值。
   - 但 `DepartmentBudgetService.Create` 不应再按“部门已有池类型”锁死新增池类型。
   - 下游 allocation / delegation 继续要求“单条链路 source/target 类型一致”，但这不应扩展成“部门级全局单类型”。

### UX Impact

这是本次变更里最重要的一块。

#### 当前问题

- 企业组织页是管理工作台，适合负责人/管理员。
- 员工额度申请是成员自助行为，入口放在这里不自然。
- 结果是后端支持普通成员，但前端主入口不可达或不符合使用路径。

#### 建议方向

- 在钱包余额或额度不足相关区域增加“申请额度”入口。
- 点击后进入单独表单页或弹框。
- 申请时仍然需要显式选择：
  - 目标部门
  - 目标预算池
- 审批、治理时间线、通知投递仍留在企业组织管理工作台内。

### Technical / Secondary Artifact Impact

需要更新的内容包括：

- `web/default/src/features/enterprise-organization/**`
- 钱包/余额相关前端模块（需后续确认具体落点）
- i18n locale JSON
- enterprise organization tests
- PRD / addendum / epics / architecture 文档

目前不要求变更：

- relay 主链路
- billing expression
- enterprise quota request 后端状态机主体
- governance timeline / notification 后端模型

## 3. Recommended Approach

### Selected Path

**Hybrid**

即：

- **Bug 部分** 直接修复。
- **Change 部分** 通过 Correct Course 文档补清需求和实施方向，再进入后续 backlog。

### Option Evaluation

#### Option 1: Direct Adjustment

- 对 i18n / label 缺失：可直接调整，`[Viable]`
- 对员工入口迁移：涉及 UX 与页面结构变更，不能只算小修
- 对预算池类型约束：涉及需求解释与后端行为修改，也不能只算小修

结论：只适合处理 bug，不足以覆盖本次全部问题。

#### Option 2: Potential Rollback

- 不需要 rollback。
- 当前能力本身有价值，问题出在入口与约束设计。

结论：`[Not viable]`

#### Option 3: PRD MVP Review

- 不需要重定义 MVP。
- 这是治理产品化的后续 refinement，不是 MVP 方向错误。

结论：`[Not viable]`

### Recommended Hybrid

1. 将 **i18n / label 缺失** 视为 bug，直接进入修复 backlog。
2. 将 **员工申请入口重构** 视为产品变更，走 CC。
3. 将 **预算池类型约束收紧过度** 视为需求实现偏差，走 CC 并明确规则后实施。

### Scope Classification

**Moderate**

原因：

- 不需要回滚已完成 Epic。
- 需要改 PRD / addendum / epics / architecture 的表述。
- 需要新增或调整后续 backlog。
- 前后端都会有后续实现变更，但不触及底层主链路。

## 4. Detailed Change Proposals

### 4.1 PRD 修改提案

Artifact:
`_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`

#### Change A: 明确员工额度申请入口不是管理工作台唯一入口

OLD:

```markdown
当员工额度不足时，允许其在所属部门上下文中发起额度申请。
```

NEW:

```markdown
当员工额度不足时，允许其从自身余额 / 钱包相关入口发起额度申请。
申请时仍必须显式选择所属部门上下文与目标预算池。
企业组织工作台保留审批与治理视图，但不作为普通员工唯一入口。
```

Rationale：把“员工自助入口”与“管理员治理工作台”解耦。

#### Change B: 明确预算池类型约束粒度

OLD:

```markdown
部门预算池类型一经创建不可在 `balance` 与 `subscription` 间切换；如需变更，必须先停用旧池并新建。
```

NEW:

```markdown
单个部门预算池实例一经创建，不可在 `balance` 与 `subscription` 间切换；如需变更，必须停用旧池并新建新池。
同一部门可并存多个预算池，不要求所有预算池必须共享同一种类型。
```

Rationale：把“单池不可变”与“部门级单类型”区分开。

### 4.2 Addendum 修改提案

Artifact:
`_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md`

新增一个小节，明确：

- 员工申请入口位于余额/钱包关联场景。
- 企业组织页保留审批和治理记录。
- 预算池类型是单池约束，不是部门级约束。
- 当一个部门存在多个预算池时，申请表和治理页都必须显式展示池类型与目标池选择。

### 4.3 Epics 修改提案

Artifact:
`_bmad-output/planning-artifacts/epics.md`

建议新增 follow-up story，而不是回写 7.5 / 7.6 完成定义：

#### New follow-up story candidate A

`Story 7.x: 将员工额度申请入口迁移到普通用户余额/钱包场景`

目标：

- 保留现有 quota request API
- 为普通用户增加可达入口
- 企业组织页继续承载审批与治理视图

#### New follow-up story candidate B

`Story 7.x: 修正企业治理工作台 i18n 与动作 label 映射`

目标：

- 修补治理时间线、通知投递、申请表、表头等缺失翻译
- 完整覆盖 `action_type` 到用户可读 label 的映射

#### New follow-up story candidate C

`Story 7.x: 放宽预算池类型约束至单池不可变而非部门级不可变`

目标：

- 修改 `DepartmentBudgetService.Create`
- 补测试
- 评估 delegation / request / list 视图对多类型并存的呈现

### 4.4 Architecture 修改提案

Artifact:
`_bmad-output/planning-artifacts/architecture.md`

新增或修订规则：

1. 员工申请入口为用户侧入口，审批与治理在企业组织工作台。
2. 部门预算池允许多池并存，但每个池实例类型不可切换。
3. budget delegation 仍要求 source/target 同类型，但不要求部门全局单类型。

## 5. Implementation Handoff

### Scope

**Moderate**

### Handoff Recipients

- **Product Owner / Product Lead**
  - 确认员工入口迁移方案是否采用“余额页按钮 + 弹框”还是“余额页按钮 + 独立表单页”
  - 确认预算池多类型并存是否作为明确规则写回 PRD

- **Developer**
  - 修复治理工作台 i18n / action label 缺失
  - 在需求确认后实现员工入口迁移
  - 放宽部门级预算池类型约束，并补回归测试

### Success Criteria

1. 企业组织页治理相关标题、表头和 action label 不再出现未翻译英文或内部 action key。
2. 普通用户无需进入企业组织管理菜单，也能从余额/钱包场景发起额度申请。
3. 同一部门允许创建多个不同类型预算池；限制仅保留在单预算池实例层面。
4. 预算委派、申请审批、预算列表和相关测试在多预算池类型并存下保持正确。

## 6. Checklist Record

- `1.1` `[x] Done`：触发故事主要关联 7.5 / 7.6
- `1.2` `[x] Done`：问题被归类为 bug + change
- `1.3` `[x] Done`：代码与文档证据已收集
- `2.1` `[x] Done`：Epic 7 不需要回滚
- `2.2` `[x] Done`：需要新增 follow-up 变更项
- `2.3` `[x] Done`：主要影响后续治理产品化，不波及旧 epic 定义
- `2.4` `[x] Done`：不需要废弃 existing epic，但需要新增后续 work
- `2.5` `[x] Done`：后续优先级应高于一般 UI polish
- `3.1` `[x] Done`：PRD 需补清入口与预算池类型约束粒度
- `3.2` `[x] Done`：Architecture 需补清入口分层和预算池约束边界
- `3.3` `[x] Done`：UI/UX 影响明显
- `3.4` `[x] Done`：次级影响落在前端与测试
- `4.1` `[x] Viable`：Bug 可直接修，但不足以覆盖全部变更
- `4.2` `[x] Not viable`：不需要 rollback
- `4.3` `[x] Not viable`：不需要重定义 MVP
- `4.4` `[x] Done`：推荐 Hybrid
- `5.1` `[x] Done`
- `5.2` `[x] Done`
- `5.3` `[x] Done`
- `5.4` `[x] Done`
- `5.5` `[x] Done`
- `6.1` `[x] Done`
- `6.2` `[x] Done`
- `6.3` `[x] Done`：用户已明确批准本次 CC 方案
- `6.4` `[N/A]`：当前先输出 Proposal，不直接改 sprint-status

## 7. Final Recommendation

本次结论应正式归类为：

- **Bug**
  - 企业治理工作台 i18n / action label 映射缺失

- **Change**
  - 员工额度申请入口从企业组织管理工作台迁出，改为普通用户余额/钱包相关入口
  - 预算池类型约束从“部门级不可变”收敛为“单池不可变”

建议按 Correct Course 流程批准后，进入：

1. backlog 重排
2. 需求文档补清
3. 后续实现

## 8. Approval and Handoff

### Approval

- 审批结果：`Approved`
- 审批时间：`2026-06-02`
- 审批人：`hth`

### Handoff Decision

本次变更范围为 `Moderate`，按 Correct Course 流程移交给：

- **Product Owner / Product Lead**
  - 确认员工额度申请的新入口形态
  - 确认预算池类型约束的需求边界
  - 决定 follow-up backlog 的优先级

- **Developer**
  - 修复企业治理工作台 i18n / action label 缺失
  - 在需求确认后调整员工申请入口
  - 在需求确认后修正预算池类型约束实现并补测试

### Immediate Next Step

建议下一步进入两段式执行：

1. 先更新规划工件：PRD / addendum / epics / architecture
2. 再基于更新后的规划创建 follow-up stories 并实施

### Workflow Completion

- Issue addressed: Epic 7 企业治理工作台 post-implementation bug + change
- Change scope: Moderate
- Artifacts modified: Sprint Change Proposal
- Routed to: Product Owner / Product Lead + Developer

Correct Course workflow complete, hth!
