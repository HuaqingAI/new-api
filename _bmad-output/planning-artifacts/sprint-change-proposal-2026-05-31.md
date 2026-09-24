---
status: approved-applied
project: new-api
created: 2026-05-31
approved: 2026-05-31
applied: 2026-05-31
workflow: bmad-correct-course
mode: incremental
trigger_source: enterprise interaction review and user feedback on 2026-05-31
recommended_scope: moderate
proposed_changes_to:
  - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md
  - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md
  - _bmad-output/planning-artifacts/epics.md
  - _bmad-output/implementation-artifacts/sprint-status.yaml
---

# Sprint Change Proposal: 企业治理视图重构与账号标识修正

## 1. Issue Summary

### Trigger

本次 Correct Course 由企业模块交互评审触发。触发源不是单个缺陷故事，而是企业组织、部门预算/资源池、组织用量等能力在完成后暴露出的系统性交互问题。

### Core Problem

问题类型：已交付能力的交互模式与账号标识策略需要后续纠偏，不属于架构失效，也不属于 MVP 不可达。

核心问题：

- 企业组织页和部门用量页仍偏“列表/表单 + detail drill-down”模式，缺少树驱动的治理视图。
- 当前治理操作仍过度依赖输入 `Department ID`、`User ID`，没有把“当前选中部门”作为页面一级状态。
- 组织树可以展示，但没有明确的折叠/展开、选中/高亮、默认展开和刷新恢复策略。
- 企业视图缺少直观的组织全局概览，部门上下文与右侧工作区承接不足。
- 钉钉同步创建用户时，默认 username 仍以 `dt_` 前缀和外部编号为主，导致成员列表、预算分配、用量排行、日志入口等场景可读性差。

### Evidence

- 用户反馈指出：企业用户管理、企业资源池等页面没有直观树形浏览、没有全局概览、树不能折叠、对象操作不能从部门数和用户列表进入、username 默认 `dt_编号` 不直观。
- 当前部门树组件是递归全展开展示，没有折叠状态，也没有页面级选中联动：`web/default/src/features/enterprise-organization/components/DepartmentTree.tsx`
- 当前企业组织页预算创建和 wallet 分配仍以 `Department ID` / `Target User ID` 输入为主：`web/default/src/features/enterprise-organization/index.tsx`
- 当前部门用量排行主要展示 `username`：`web/default/src/features/enterprise-usage/index.tsx`
- 当前钉钉同步/OAuth 创建用户时，默认 username 仍由 `dt_` 前缀策略生成：`service/enterprise/dingtalk_sync_helpers.go`、`service/enterprise/dingtalk_oauth.go`

## 2. Impact Analysis

### Delivery Status Reality

根据 `_bmad-output/implementation-artifacts/sprint-status.yaml`：

- `epic-1` 已标记为 `done`
- `epic-3` 已标记为 `done`
- `epic-4` 虽仍标记为 `in-progress`，但 `4-1` 到 `4-5` 已全部标记为 `done`

因此，本次变更不应回写旧 Epic 的完成定义，而应新增后续纠偏工作。

### Epic Impact

| Epic | 现状 | 影响 | 处理方式 |
| ---- | ---- | ---- | -------- |
| Epic 1 | done | 组织页交互骨架不足 | 不改完成状态；由 follow-up epic 承接 |
| Epic 2 | stories done | username 生成策略与账号治理不足 | 不改完成状态；由 follow-up epic 承接 |
| Epic 3 | done | 预算分配入口仍偏 ID 驱动 | 不改完成状态；由 follow-up epic 承接 |
| Epic 4 | stories done | 用量页骨架和用户展示规则不足 | 不改完成状态；由 follow-up epic 承接 |
| Epic 5 | backlog | 后续风险页也应复用统一用户展示规则 | 不调整本轮 scope，仅作为联动约束 |

### PRD Impact

PRD 需要补充 follow-up 修订，而不是推翻原有 FR：

- `FR-1`：增加企业组织页“左树右工作区”的页面骨架要求。
- `FR-6`：增加钉钉账号默认 username 策略和 username 受控修改能力。
- `FR-12`：增加预算池查看和分配必须从当前部门上下文进入。
- `FR-13` / `FR-14`：增加部门用量页“左树右分析区”的页面骨架要求。
- `FR-14` / `FR-17` / `FR-20`：增加“可读名称优先，username / ID 兜底”的统一展示规则。

### Architecture Impact

无需重做企业后端架构，也不需要回退现有预算池、wallet、用量聚合或 relay 边界。架构层需要补充的信息是：

- 企业治理视图的信息架构
- 树状态模型
- 页面一级状态与 URL 持久化策略
- username 生成、修改、缓存刷新和展示联动规则

这些更适合写入 PRD `addendum.md`，不需要重写现有 `architecture.md`。

### UX Impact

当前没有独立 UX 文档，因此本次变更必须在规划工件中显式补足以下 UX 约束：

- 企业组织页：左侧可折叠组织树，右侧当前部门工作区
- 部门用量页：左侧可折叠组织树，右侧当前部门分析区
- 当前选中部门是页面一级状态
- 树默认展开、选中、高亮、刷新保留策略
- 部门上下文驱动的成员维护、预算查看、资源池分配

### Technical Impact

本次 proposal 预计影响以下实现域：

- Default 前端企业组织页与部门用量页
- 企业后端接口的查询参数和返回结构（为树驱动状态和可读名称展示提供支撑）
- 钉钉用户创建 username 生成逻辑
- 用户修改 username 后的缓存刷新、查询和展示联动

不触碰：

- `relay/**`
- `pkg/billingexpr/**`
- 现有部门预算 wallet 模型
- 部门用量重复计入口径

## 3. Recommended Approach

### Selected Path

Option 1 / Hybrid：**新增 Follow-up Epic，最小补充 PRD 主文档和 Addendum，不回写旧 Epic 完成状态**

### Rationale

这是最符合当前事实状态的路径：

- 保留已完成 Epic 的历史真实性
- 不否定已实现能力，只承认它们在交互模式上存在后续纠偏
- 不需要回滚预算、wallet、用量聚合等已有后端方案
- 能把这次问题准确描述为“树驱动治理视图 + 账号标识策略修正”

### Alternatives Considered

- 修改旧 Epic：不推荐。会把历史完成状态与后续纠偏混在一起。
- Rollback：不适用。问题在交互模式和账号策略，不在底层能力方向。
- PRD MVP Review / 延后到更晚版本：可行但不推荐。会继续保留 ID 驱动治理和不可读 username 的明显体验缺口。

### Effort and Risk

- Effort: Medium
- Risk: Medium
- Timeline impact: Moderate；建议作为企业治理能力的后续迭代独立排期，而不是回写到旧 Epic 的“已完成”定义

## 4. Detailed Change Proposals

### Proposal 1: 在 `epics.md` 中新增 Follow-up Epic 6

Artifact: `_bmad-output/planning-artifacts/epics.md`

OLD:

```markdown
### Epic 5: 内容风险事件、告警与部门风险概览
```

NEW:

```markdown
### Epic 6: 企业治理视图重构与账号标识修正

管理员可以通过树驱动的治理视图完成企业组织、预算和用量管理，并确保企业账号标识在登录、查询、统计和分配场景中可读、可维护且一致。

**FRs covered:** follow-up to FR1, FR2, FR6, FR10, FR12, FR13, FR14
```

Rationale: 这次问题不是旧 Epic 失败，而是已交付能力的后续治理体验纠偏，应使用新的 epic 承接。

### Proposal 2: 为 Epic 6 新增 5 个 Follow-up Stories

Artifact: `_bmad-output/planning-artifacts/epics.md`

OLD:

```markdown
无对应 follow-up stories。
```

NEW:

```markdown
### Story 6.1: 企业组织页重构为树驱动治理视图
### Story 6.2: 部门用量页重构为树驱动分析视图
### Story 6.3: 部门上下文驱动的成员与预算操作
### Story 6.4: 钉钉账号 username 策略与受控修改
### Story 6.5: 企业页面统一用户标识展示规则
```

Story 6.1 重点：

- 左侧可折叠组织树
- 右侧当前部门工作区
- 当前选中部门是页面一级状态
- 刷新后恢复选中部门
- 默认展开根节点和当前选中路径

Story 6.2 重点：

- 左侧树与组织页采用一致状态模型
- 右侧承接用量概览、趋势、模型分布、用户排行、日志入口
- 右侧同时承接该部门维度下的说明/配置区

Story 6.3 重点：

- 成员维护、预算池查看、wallet 分配从当前部门上下文进入
- `Department ID` / `User ID` 不再作为主界面默认入口

Story 6.4 重点：

- 钉钉创建用户时默认 username 生成策略不再以 `dt_编号` 为主要结果
- 支持受控的 username 修改能力
- 明确缓存刷新、登录、查询和展示联动规则

Story 6.5 重点：

- 成员列表、预算分配、用量排行、日志入口、风险事件统一采用“可读名称优先，username / ID 兜底”的展示规则

Rationale: 这五个 stories 一起覆盖“信息架构、交互模式、账号标识”三层问题，避免继续拆成零散补丁。

### Proposal 3: 在 PRD 主体中追加 Follow-up 修订要求

Artifact: `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`

Sections to update:

- `FR-1`
- `FR-6`
- `FR-12`
- `FR-13`
- `FR-14`
- `FR-17`
- `FR-20`

OLD:

```markdown
FR-1/12/13/14 已覆盖树、预算、用量等功能能力，但没有显式约束树驱动页面骨架、页面一级状态和部门上下文操作。
FR-6 已覆盖钉钉登录与建号，但没有显式约束默认 username 策略和 username 修改治理。
```

NEW:

```markdown
追加 V1.x / follow-up 修订：

- 企业组织页采用“左侧组织树 + 右侧当前部门工作区”的结构。
- 部门用量页采用“左侧组织树 + 右侧当前部门分析区”的结构。
- 当前选中部门是页面一级状态，而不是附属 detail drill-down 状态。
- 组织树定义展开/折叠、选中/高亮、默认展开和刷新恢复策略。
- 部门成员维护、预算池查看、资源池分配从部门上下文进入，不再以输入 ID 为主路径。
- 钉钉同步或 OAuth 创建用户时，默认 username 生成策略必须优先产生可读且唯一的账号名，不能继续以 `dt_编号` 作为主要结果。
- 系统支持受控的 username 修改能力。
- 成员列表、预算分配、用量排行、日志入口、风险事件与告警页面统一采用“可读名称优先，username / ID 兜底”的展示规则。
```

Rationale: 主 PRD 只补原则，不推翻原 FR 完成事实。

### Proposal 4: 在 PRD Addendum 中新增“企业治理视图信息架构与交互骨架”章节

Artifact: `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md`

OLD:

```markdown
Addendum 中没有专门定义企业治理视图的信息架构和树状态模型。
```

NEW:

```markdown
## X. 企业治理视图信息架构与交互骨架（V1.x follow-up）

### X.1 企业组织页骨架
- 左侧：可折叠组织树
- 右侧：当前部门摘要、成员列表、预算池/资源池、相关操作区

### X.2 部门用量页骨架
- 左侧：可折叠组织树
- 右侧：当前部门的用量概览、趋势、模型分布、用户排行、日志入口、说明/配置区

### X.3 树状态模型
- 当前选中部门是页面一级状态
- 建议通过 URL `department_id` 持久化当前选中部门
- 默认展开根节点和当前选中路径
- 页面刷新后恢复当前选中部门

### X.4 操作入口约束
- 成员维护、预算池查看、资源池分配都从部门上下文进入
- ID 输入仅允许作为调试兜底，不作为主界面默认入口

### X.5 账号标识策略
- username 默认生成优先级
- 冲突处理
- 修改权限
- 缓存刷新
- 展示回退规则
```

Rationale: 交互骨架和状态模型比功能点更细，适合放入 addendum 作为后续 story 的直接上下文。

### Proposal 5: 在 `sprint-status.yaml` 中新增 Epic 6 和对应 stories

Artifact: `_bmad-output/implementation-artifacts/sprint-status.yaml`

OLD:

```yaml
development_status:
  epic-1: done
  epic-2: in-progress
  epic-3: done
  epic-4: in-progress
  epic-5: backlog
```

NEW:

```yaml
development_status:
  epic-1: done
  epic-2: in-progress
  epic-3: done
  epic-4: in-progress
  epic-5: backlog
  epic-6: backlog
  6-1-rebuild-enterprise-organization-as-tree-driven-governance-view: backlog
  6-2-rebuild-department-usage-as-tree-driven-analysis-view: backlog
  6-3-move-membership-and-budget-operations-into-department-context: backlog
  6-4-fix-dingtalk-username-strategy-and-controlled-rename: backlog
  6-5-unify-enterprise-user-display-rules: backlog
  epic-6-retrospective: optional
```

Rationale: 只有在 proposal 获批后才更新状态文件；本次 proposal 先把结构定义清楚。

## 5. Implementation Handoff

### Scope Classification

**Moderate**

原因：

- 不重做底层预算、wallet、用量聚合架构
- 需要调整规划工件、页面信息架构、后端展示契约和账号标识策略
- 涉及前后端联动，但边界集中在企业模块和用户标识治理

### Handoff Recipients

- Product Owner / Planning owner
  - 负责确认 Epic 6 与 story 粒度
  - 负责接受 PRD 主体和 addendum 的 follow-up 修订方式

- Developer / Frontend owner
  - 负责组织页和部门用量页的树驱动重构
  - 负责把成员、预算和分配操作迁移到部门上下文

- Developer / Backend owner
  - 负责补齐企业接口的上下文查询与返回结构
  - 负责 username 默认生成策略和 username 修改后的缓存/查询联动

### Success Criteria

- 企业组织页采用左树右工作区，当前选中部门为页面一级状态
- 部门用量页采用左树右分析区，当前选中部门为页面一级状态
- 成员维护、预算池查看、wallet 分配从部门上下文进入
- 主界面默认流程不再依赖输入 `Department ID` / `User ID`
- 钉钉新建账号默认 username 不再以 `dt_编号` 作为主要结果
- 企业页面统一按“可读名称优先，username / ID 兜底”展示用户

## 6. Checklist Status

| Item | Status | Notes |
| ---- | ------ | ----- |
| 1.1 Triggering story identified | Done | Trigger is enterprise interaction review, not a single story defect. |
| 1.2 Core problem defined | Done | Tree-driven governance view and username strategy gap identified. |
| 1.3 Evidence collected | Done | User feedback + current PRD/implementation evidence collected. |
| 2.1 Current epic assessed | Done | Existing epics are already completed or effectively completed. |
| 2.2 Epic-level change determined | Done | Do not rewrite old epics; add Epic 6. |
| 2.3 Remaining epics reviewed | Done | Epic 5 only receives display-rule linkage, not scope change. |
| 2.4 New epic needed | Done | Follow-up Epic 6 is recommended. |
| 2.5 Epic sequencing considered | Done | Epic 6 should run after current completed epics, without reopening them. |
| 3.1 PRD conflict check | Action-needed | PRD needs follow-up interaction and username policy additions. |
| 3.2 Architecture conflict check | Action-needed | Add addendum-level IA and state model constraints. |
| 3.3 UX conflict check | Action-needed | No dedicated UX doc; story/PRD must carry the interaction contract. |
| 3.4 Other artifacts impact | Action-needed | Username affects cache, login, log filters, and enterprise views. |
| 4.1 Direct adjustment | Viable | Via new follow-up epic and planning changes. |
| 4.2 Rollback | Not viable | No value; issue is not architectural rollback-worthy. |
| 4.3 MVP review | Partially viable | Possible to defer, but not preferred. |
| 4.4 Recommended path | Done | Hybrid direct adjustment selected. |
| 5.1 Issue summary | Done | This proposal. |
| 5.2 Epic/artifact adjustments | Done | Epic 6 + PRD/Addendum + sprint-status change defined. |
| 5.3 Recommended path rationale | Done | This proposal. |
| 5.4 MVP impact and action plan | Done | MVP remains achievable; follow-up interaction correction required. |
| 5.5 Handoff plan | Done | PO/DEV split defined. |
| 6.1 Final review | Done | Proposal reviewed and aligned to approved interaction direction. |
| 6.2 Proposal accuracy | Done | Proposal content updated to reflect approved tree-driven governance view and username strategy scope. |
| 6.3 Explicit user approval | Done | User approved with `yes` on 2026-05-31. |
| 6.4 Update sprint-status.yaml | Done | Epic 6 and follow-up stories added as backlog items. |
| 6.5 Confirm next steps | Done | Handoff completed to planning/PO and implementation owners. |

## 7. Final Handoff

### Applied Deliverables

- `epics.md`：新增 `Epic 6` 与 5 个 follow-up stories
- `sprint-status.yaml`：新增 `epic-6` 和对应 backlog stories
- `prd.md`：追加 V1.2 follow-up 修订说明
- `addendum.md`：新增企业治理视图信息架构与交互骨架章节

### Routed To

- Product Owner / Planning owner：确认 Epic 6 排期与 story 优先级
- Frontend owner：落地树驱动企业组织页与部门用量页
- Backend owner：补齐部门上下文接口契约与 username 策略联动

### Workflow Completion Summary

- Issue addressed: 企业治理视图仍偏 ID 驱动和表单/列表模式，缺少树驱动上下文；钉钉默认 username 不可读
- Change scope: Moderate
- Artifacts modified: `epics.md`、`sprint-status.yaml`、`prd.md`、`addendum.md`、本 proposal
- Routed to: PO / Frontend / Backend
