---
status: approved
project: new-api
created: 2026-06-01
approved: 2026-06-01
workflow: bmad-correct-course
mode: batch
trigger_source: enterprise governance follow-up brief provided by user on 2026-06-01
recommended_scope: moderate
selected_path: hybrid
proposed_changes_to:
  - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md
  - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md
  - _bmad-output/planning-artifacts/epics.md
  - _bmad-output/planning-artifacts/architecture.md
  - _bmad-output/implementation-artifacts/sprint-status.yaml
---

# Sprint Change Proposal: 分层预算治理与额度审批

## 1. Issue Summary

### Trigger

本次变更不是回滚既有企业能力，而是在当前企业板块基础上新增 follow-up 能力层，补齐“分层预算治理与额度审批”。

触发源不是单个 defect story，而是企业板块在完成基础交付后暴露出的治理闭环缺口：当前能力已经能承载组织树、部门预算池、成员额度分配、部门用量分析和告警，但仍明显偏“管理员直操工具”。

### Core Problem

问题类型：**已交付能力之上的新增治理需求与产品边界细化**，不是架构失效，不是 MVP 不可达，也不是既有 Epic 方向错误。

核心问题：

1. 缺少部门负责人权限模型，预算治理过度集中在全局管理员。
2. 缺少“包含子部门”的跨层查看能力，至少 summary / overview 场景无法按治理树聚合。
3. 缺少“上级预算池 -> 下级部门预算池”的分层治理能力，当前只有“部门 -> 成员 wallet”。
4. 现有预算分配虽然已有 revoke/expiry 基础，但缺少完整的调整、取消、回收闭环。
5. 缺少员工额度申请 -> 部门负责人审批 -> 自动分配的自助闭环。
6. 审计与通知链路还未覆盖上述治理动作，企业板块尚未形成完整治理闭环。

### Evidence

当前规划工件已经证明企业基础能力成立，但治理层不足：

- `PRD` 已覆盖部门预算池、成员 wallet 分配、部门用量和告警，但明确写明 “V1 只做预算配置和操作日志”，未覆盖单步额度审批。
- `epics.md` 中 `Epic 3` 的重点是“预算池 -> 成员 wallet”，没有“预算池 -> 下级部门预算池”。
- `architecture.md` 已有部门管理员权限和 wallet 生命周期联动，但没有“有效负责人优先级”“跨层 budget delegation”“额度申请审批”三套机制。
- `sprint-status.yaml` 显示企业基础 epics 已实质完成，因此本次问题应由 follow-up epic 承接，而不是重开旧 epic。

## 2. Impact Analysis

### Delivery Status Reality

根据 [_bmad-output/implementation-artifacts/sprint-status.yaml](D:/ProgramData/git/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/sprint-status.yaml)：

- `epic-1` 已完成。
- `epic-3` 已完成。
- `epic-5` 已完成。
- `epic-4`、`epic-6` 虽仍标记为 `in-progress`，但其 story 明细均已是 `done`。

结论：这次变更**不应回写或重开既有 Epic 1 / 3 / 4 / 6 的完成定义**，应通过新增 follow-up epic 承接。

### Checklist Summary

- `1.1 Triggering story`：`[x] Done`。无单一触发 story，本次由企业治理能力评审触发，核心基础来自 Epic 3 / Epic 4 / Epic 6。
- `1.2 Core problem`：`[x] Done`。属于新增治理需求，不是已选方案失败。
- `1.3 Evidence`：`[x] Done`。已有 PRD / epics / architecture / sprint-status 证据充分。
- `2.1~2.5 Epic impact`：`[x] Done`。应新增 Epic 7，不回滚旧 epic。
- `3.1~3.4 Artifact impact`：`[x] Done`。PRD、addendum、epics、architecture、sprint-status 均需增量补充。
- `4.1~4.4 Path forward`：`[x] Done`。推荐 `Hybrid`。

### Epic Impact

| Epic | 现状 | 本次影响 | 建议处理 |
| --- | --- | --- | --- |
| Epic 1 | 组织与部门基线已成立 | 不足以支撑负责人权限与跨层治理 | 保持已完成，Epic 7 follow-up 承接 |
| Epic 3 | 部门预算池与成员 wallet 已成立 | 缺少上级到下级预算池治理，缺少完整调整/取消/回收策略 | 保持已完成，Epic 7 follow-up 承接 |
| Epic 4 | 用量聚合与导出已成立 | 需补“包含子部门”查询作用域 | 保持既有成果，Epic 7 补跨层 scope |
| Epic 5 | 告警能力已成立 | 需补审批/治理动作的通知与审计联动 | 保持既有成果，Epic 7 复用通知能力 |
| Epic 6 | 树驱动治理视图已成立 | 可作为新治理闭环的 UI 容器 | 保持已完成，Epic 7 在其之上扩展 |

### PRD Impact

需要对 PRD 做**增量补充**，不推翻现有 FR-1~FR-20：

- 增加 `V1.3` revision note。
- 新增治理层用户旅程，至少补：
  - 上级部门负责人向下级部门委派预算；
  - 员工自主申请额度并经部门负责人单步审批。
- 新增 FR-21 ~ FR-26，承接本次 follow-up。
- 把“V1 只做预算配置和操作日志”的旧非目标，修正为“仍不做复杂审批流，但补齐单步额度申请审批闭环”。

### PRD Addendum Impact

需要在 `addendum.md` 中新增“V1.3 分层预算治理与额度审批”章节，补足：

- 部门负责人来源、优先级和覆盖关系；
- 包含子部门的 scope 语义、URL / API 参数与 UI 切换；
- 上级预算池向下级预算池分配的状态机；
- 成员 allocation 调整 / 取消 / 回收的审计建模；
- 申请 / 审批 / 自动分配 / 通知 / 幂等规则；
- 新的测试重点。

### Architecture Impact

不需要重写现有企业架构，但需要**补充 architecture 约束**，因为当前架构文档还没有以下内容：

- 有效部门负责人解析规则；
- 祖先到后代的 scope resolver；
- 部门预算池到部门预算池的 transfer/delegation ledger；
- 员工额度 request / approval workflow；
- 治理动作通知与 delivery trace。

建议：**补架构，不推翻架构**。继续保持：

- 不触碰 `relay/**`
- 不改计费表达式主链路
- 不改 `/v1/**` 协议表面
- 继续以新增表 / 新 service / 新 controller / 新路由承接

### UX Impact

不需要新增独立 UX 文档，但必须在 addendum / epics 中明确以下 UX 约束：

- “包含子部门”第一版优先落在 `usage / alerts / budget summary` 等汇总视图。
- 不要求第一版把所有明细列表都扩成整棵子树。
- 员工申请额度时必须选择一个明确的“部门 + 预算池模式/池子”上下文，避免多部门成员关系下审批归属歧义。
- 调整动作按“关闭旧分配 + 新建新分配”展示，不静默覆盖历史。

### Technical Impact

本次变更是**治理层增强**，不是底层重构：

- 继续复用现有 `enterprise_department_budgets`、`enterprise_quota_allocations`、`wallet_state_sync_task`、`balance_expiry_task`。
- 新增治理对象应优先落在 enterprise 模块自有表和 service 中。
- 继续遵循三库兼容、`common/json.go`、DTO pointer + `omitempty`、`enterprise_` 前缀等既有规则。

## 3. Recommended Approach

### Selected Path

**Hybrid**

具体含义：

- 不是 rollback；
- 不是重开旧 Epic；
- 也不是上升到重新定义 MVP；
- 而是通过新增 `Epic 7` 承接 follow-up 治理层能力，同时对 PRD / addendum / epics / architecture / sprint-status 做定点补充。

### Why Not Direct Adjustment Inside Existing Epics

不推荐直接改写旧 epic，原因是：

- 当前 `sprint-status` 已经体现旧 epic 基础能力完成；
- 把治理 follow-up 塞回旧 epic 会污染历史完成定义；
- 这次问题不是“Story 3.2 没做对”，而是“完成了基础分配后还需要更上一层的治理闭环”。

### Why Not Rollback

不推荐 rollback，原因是：

- 现有企业预算、用量、告警基础能力仍然正确；
- 问题出在治理层缺口，不在底层模型方向错误；
- rollback 会打断已有效的交付成果，收益很低。

### Why Not Full MVP Replan

不需要上升到更高层重规划，原因是：

- 现有 MVP 依然成立；
- 本次新增能力虽然重要，但仍然是企业模块的自然 follow-up；
- 核心架构边界仍稳定：enterprise 自治、relay/billing 不动。

### Scope Classification

**Moderate**

理由：

- 需要新增一个 follow-up epic；
- 需要扩充规划、架构和 sprint-status；
- 需要引入新的治理对象和审批流；
- 但不涉及主链路推翻，不需要 PM/Architect 级别的全面重规划。

### Effort and Risk

- Effort：`Medium`
- Risk：`Medium`
- Timeline impact：`Moderate`

## 4. Detailed Change Proposals

### 4.1 PRD

Artifact:
[_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md](D:/ProgramData/git/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md)

#### Change A: 增加 V1.3 revision note

OLD:

```markdown
revision_notes:
  - "V1.1 ..."
  - "V1.2 ..."
```

NEW:

```markdown
revision_notes:
  - "V1.1 ..."
  - "V1.2 ..."
  - "V1.3 (2026-06-01): 新增分层预算治理与额度审批 follow-up。补充部门负责人权限模型、包含子部门的查询作用域、上级预算池向下级部门预算池分配、成员 allocation 调整/取消/回收、员工额度申请与单步审批、审计与钉钉通知。保持既有企业预算/用量/告警基础能力与 relay/billing 主链路不变。"
```

Rationale：让本次变更成为明确的 follow-up 修订，而不是隐式需求漂移。

#### Change B: 修正非目标边界，避免与本轮 scope 冲突

OLD:

```markdown
- 部门级复杂审批流：V1 只做预算配置和操作日志。
```

NEW:

```markdown
- 不做复杂多级审批、代理审批、会签流。
- V1.3 增补员工额度申请、部门负责人单步审批、审批通过后自动分配、审计与通知闭环。
```

Rationale：保留“不做复杂审批”的边界，但显式允许本轮引入“简单审批”。

#### Change C: 新增治理层功能需求 FR-21 ~ FR-26

OLD:

```markdown
FR-20: 展示部门风险概览 ...
```

NEW:

```markdown
FR-21: 建立部门负责人权限模型。负责人来源同时支持钉钉同步与手动指定；推荐优先级为 `manual_deny_override > manual_grant > dingtalk_synced_owner`，以本地显式覆盖结果作为最终生效权限。
FR-22: 在用量、告警、预算 summary 等查询/汇总页面支持“包含子部门”作用域选项，第一版先覆盖 summary / overview 视图，不要求把所有明细列表扩成整棵子树。
FR-23: 支持上级部门预算池向任意后代部门预算池分配额度，不限制为直属下级；继续复用现有 `balance` / `subscription` 两类预算池能力，不引入新预算类型。
FR-24: 为成员额度分配补齐调整、取消、回收能力；调整优先按“关闭旧分配 + 新建新分配”建模，取消不做硬删除，明确 `active / superseded / revoked / expired / closed` 等状态语义。
FR-25: 当员工额度不足时，允许其在所属部门上下文中发起额度申请；员工自主选择基于现有企业预算池能力的模式或目标池，由部门负责人执行单步审批，审批通过后系统自动复用现有预算分配链路完成分配。
FR-26: 对部门负责人授权、跨层预算委派、成员 allocation 调整/取消/回收、员工申请审批、自动分配和回收等动作形成统一审计与钉钉通知闭环。
```

Rationale：这次不是局部 AC 调整，而是新的能力域，应该新增 FR 编号承接。

#### Change D: 新增 follow-up 用户旅程

建议新增至少 2 个 UJ：

- `UJ-5`：上级部门负责人把预算分配给下级部门预算池，下级负责人继续分配给成员。
- `UJ-6`：员工额度不足时提交申请，由部门负责人审批，系统自动分配额度并通知结果。

#### Change E: 多部门员工申请的归属规则

建议在 FR-25 下增加明确规则：

```markdown
多部门成员提交额度申请时，必须显式选择“目标部门 + 目标预算池/预算模式”；审批只路由到该目标部门的有效负责人集合，避免跨部门归属歧义。
```

Rationale：与 V1.1 的 N:N 成员关系保持一致。

### 4.2 PRD Addendum

Artifact:
[_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md](D:/ProgramData/git/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md)

#### Change A: 在现有第 7 节后新增第 8 节

OLD:

```markdown
### 7.6 企业页面统一展示规则
...
```

NEW:

```markdown
## 8. 分层预算治理与额度审批（V1.3 Follow-up）

### 8.1 部门负责人来源与生效优先级
- 来源同时支持钉钉同步与手动指定。
- 推荐生效顺序：`manual_deny_override > manual_grant > dingtalk_synced_owner`。
- 手动授权不是覆盖钉钉源数据，而是本地权限层覆盖；钉钉同步结果保留为来源事实。

### 8.2 包含子部门的查询作用域
- 第一版仅要求落在 `usage / alerts / budget summary` 等汇总视图。
- API 建议增加 `include_descendants` 布尔参数。
- 默认保持“仅当前部门”以兼容既有口径；用户可显式切换为“包含子部门”。
- 若性能需要，继续遵循“邻接表 + 可选路径缓存字段”的既有方向，不引入闭包表。

### 8.3 上级预算池到下级预算池的治理语义
- 允许祖先部门向任意后代部门预算池分配，不限直属下级。
- 继续复用 `balance / subscription` 两类预算池，不新增第三类预算类型。
- 历史调整按“关闭旧 transfer + 新建新 transfer”保留审计，不静默改写历史。

### 8.4 成员 allocation 生命周期
- 调整优先建模为 `supersede`：旧 allocation 关闭，新 allocation 生效。
- 取消不做硬删除，保留 `revoked / closed / expired / superseded` 等状态。
- 回收动作必须记录回收额度、操作者、处理时间和来源 allocation。

### 8.5 员工申请与单步审批
- 员工在所属部门上下文中提交申请。
- 员工必须选择目标部门和基于现有预算池能力的模式/目标池。
- 审批为单步，由目标部门的有效负责人集合处理；若无负责人，回退给 Admin。
- 若预算不足，本轮只做简单处理：审批人可驳回或按较小额度批准，不做升级审批链。
- 审批通过后直接复用现有预算分配能力完成自动分配，不引入新的额度生命周期规则。

### 8.6 审计与钉钉通知
- 对授权、预算下拨、成员分配调整、申请提交、审批结果、自动分配、回收等动作记录统一审计。
- 钉钉通知优先覆盖：申请提交、审批通过、审批拒绝、分配成功、分配回收/撤销。

### 8.7 测试补充
- 覆盖负责人优先级和覆盖关系。
- 覆盖 ancestor -> descendant scope 查询。
- 覆盖上级预算池向任意后代预算池分配的铁律与并发安全。
- 覆盖申请审批后的自动分配幂等性。
- 覆盖审批通知失败不影响核心事务成功。
```

Rationale：把产品口径、交互边界和实现约束收拢到 addendum，比散落到 AC 更可维护。

### 4.3 Epics

Artifact:
[_bmad-output/planning-artifacts/epics.md](D:/ProgramData/git/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/epics.md)

#### Change A: 新增 FR Inventory / Coverage Map 项

OLD:

```markdown
FR20: Epic 5 - 部门风险概览。
```

NEW:

```markdown
FR21: Epic 7 - 部门负责人权限模型。
FR22: Epic 7 - 包含子部门的查询作用域。
FR23: Epic 7 - 上级预算池向下级部门预算池分配。
FR24: Epic 7 - 成员 allocation 调整 / 取消 / 回收。
FR25: Epic 7 - 员工额度申请与单步审批。
FR26: Epic 7 - 审计与钉钉通知闭环。
```

#### Change B: 新增 Follow-up Epic 7

OLD:

```markdown
### Epic 6: 企业治理视图重构与账号标识修正
```

NEW:

```markdown
### Epic 7: 分层预算治理与额度审批

管理员与部门负责人可以在既有企业组织、预算、用量和告警基础上，完成跨层预算治理、成员额度治理、员工额度申请审批和治理动作审计通知，形成企业板块的治理闭环。

**FRs covered:** FR21, FR22, FR23, FR24, FR25, FR26
```

#### Change C: Epic 7 story 拆分

建议新增 6 个 stories：

1. `Story 7.1: 建立部门负责人权限模型与生效优先级`
2. `Story 7.2: 为预算/用量/告警 summary 增加包含子部门作用域`
3. `Story 7.3: 支持上级预算池向后代部门预算池分配`
4. `Story 7.4: 补齐成员 allocation 的调整、取消与回收治理`
5. `Story 7.5: 支持员工额度申请、单步审批与自动分配`
6. `Story 7.6: 建立治理动作审计时间线与钉钉通知`

建议每个 story 的重点如下：

##### Story 7.1

- 负责人来源：钉钉同步 + 手动指定。
- 生效优先级：`manual_deny_override > manual_grant > dingtalk_synced_owner`。
- 不扩展 `users.role`；继续落在 enterprise 部门角色体系。

##### Story 7.2

- 优先覆盖 `usage / alerts / budget summary`。
- API 参数使用 `include_descendants`。
- 第一版不要求所有 detail list 展开整棵子树。

##### Story 7.3

- 允许祖先预算池向任意后代预算池下拨。
- 继续使用 `balance / subscription`。
- 不限直属下级，但限制在同一祖先-后代树链内。
- 调整按“关闭旧 transfer + 新建新 transfer”留痕。

##### Story 7.4

- 面向成员 allocation 补齐 increase / decrease / close / revoke / reclaim。
- 不做硬删除。
- 复用既有 wallet 生命周期任务，不改 relay 主链路。

##### Story 7.5

- 员工选择目标部门和预算池模式/目标池发起申请。
- 审批为单步，由该部门有效负责人处理。
- 审批通过后复用现有 allocation service 自动分配。

##### Story 7.6

- 审计覆盖授权、transfer、adjust、request、approve、reclaim。
- 钉钉通知覆盖提交、审批结果、分配成功、撤销/回收。

Rationale：这 6 个 story 能完整覆盖你给出的能力域，并且避免把新治理需求硬塞回 Epic 3 / 4 / 6。

### 4.4 Sprint Status

Artifact:
[_bmad-output/implementation-artifacts/sprint-status.yaml](D:/ProgramData/git/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/sprint-status.yaml)

#### Mandatory Change

OLD:

```yaml
  epic-6: in-progress
  6-1-rebuild-enterprise-organization-as-tree-driven-governance-view: done
  6-2-rebuild-department-usage-as-tree-driven-analysis-view: done
  6-3-move-membership-and-budget-operations-into-department-context: done
  6-4-fix-dingtalk-username-strategy-and-controlled-rename: done
  6-5-unify-enterprise-user-display-rules: done
  epic-6-retrospective: optional
```

NEW:

```yaml
  epic-6: in-progress
  6-1-rebuild-enterprise-organization-as-tree-driven-governance-view: done
  6-2-rebuild-department-usage-as-tree-driven-analysis-view: done
  6-3-move-membership-and-budget-operations-into-department-context: done
  6-4-fix-dingtalk-username-strategy-and-controlled-rename: done
  6-5-unify-enterprise-user-display-rules: done
  epic-6-retrospective: optional

  epic-7: backlog
  7-1-establish-department-owner-role-model-and-precedence: backlog
  7-2-add-descendant-inclusive-scope-to-budget-usage-and-alert-summaries: backlog
  7-3-delegate-ancestor-budget-pools-to-descendant-department-pools: backlog
  7-4-govern-member-allocations-with-adjust-cancel-and-reclaim: backlog
  7-5-support-employee-quota-requests-single-step-approval-and-auto-allocation: backlog
  7-6-audit-governance-actions-and-send-dingtalk-notifications: backlog
  epic-7-retrospective: optional
```

#### Optional Cleanup

由于 `epic-4` 与 `epic-6` 的 story 明细已经全部是 `done`，若团队把 retrospective 视为可选，可顺手把 `epic-4` / `epic-6` 调整为 `done`。这不是本次变更的主目标，但属于状态卫生优化。

### 4.5 Architecture Supplement

Artifact:
[_bmad-output/planning-artifacts/architecture.md](D:/ProgramData/git/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md)

#### Recommended Supplement Areas

这次架构修改建议做“补充章节”，不重写原文：

1. **Authorization Model**
   - 明确有效负责人解析规则：
     `manual_deny_override > manual_grant > dingtalk_synced_owner`
   - 继续不扩展 `users.role`。

2. **Scope Resolver**
   - 为 `include_descendants` 增加 ancestor/descendant 解析规则。
   - 继续沿用邻接表建模；如性能需要，正式引入跨库兼容的路径缓存字段，不引入闭包表。

3. **Budget Delegation**
   - 新增“部门预算池 -> 部门预算池”的 transfer/delegation ledger。
   - 继续保留 `enterprise_quota_allocations` 负责“部门预算池 -> 成员 wallet”。
   - 不把新治理逻辑塞进 relay 或现有 subscription 结算主链路。

4. **Quota Request Workflow**
   - 新增员工 request / approval / fulfillment workflow。
   - 审批通过后的 fulfill 继续调用既有 allocation service。
   - 请求与审批要有幂等键或等价保护，避免重复审批重复分配。

5. **Governance Notifications**
   - 复用现有 `notify_*` / 钉钉机器人能力。
   - 通知失败不能影响预算治理核心事务提交。

Rationale：这几项都是本次能力的真实架构增量，单靠 PRD / epics 不足以约束实现。

## 5. Recommended Scope Direction

### P0

以下建议纳入 `Epic 7` 的首轮交付：

1. 新增 `Epic 7`，不回滚既有企业预算、用量、告警基础能力。
2. 建立部门负责人权限模型，明确钉钉同步与手动指定并存的优先级和覆盖关系。
3. 在 `usage / alerts / budget summary` 先落地“包含子部门”查询作用域。
4. 支持上级预算池向任意后代部门预算池分配，继续使用 `balance / subscription`。
5. 补齐成员 allocation 的调整、取消、回收能力，按审计友好方式建模。
6. 提供员工额度申请 -> 部门负责人单步审批 -> 自动分配闭环。
7. 建立治理动作审计与钉钉通知。

### P1 / Later

以下内容建议放到后续迭代，而不是压进本轮：

1. 把“包含子部门”扩展到所有 detail 列表和所有明细 drill-down。
2. 复杂多级审批、代理审批、会签流。
3. 预算不足时的升级审批链或自动升级路由。
4. 审批 SLA、催办、超时升级。
5. 更复杂的审批策略模板与自动化规则。

## 6. Implementation Handoff

### Recommended Handoff Type

**Moderate scope**

建议路由给：**Product Owner / Architect / Developer**

### Responsibilities

#### Product Owner

- 确认 FR-21 ~ FR-26 的最终口径。
- 确认“多部门员工申请时必须显式选择目标部门/池子”的假设是否接受。
- 确认第一版 `include_descendants` 只落在 summary 视图。

#### Architect

- 补充 architecture 中的负责人解析、descendant scope、budget delegation、request workflow、notification 约束。
- 决定 transfer ledger 与 request 表的最小可行 schema。
- 审核三库兼容与并发不变量。

#### Developer

- 按以下顺序实施 `Epic 7`：
  1. `7.1` 负责人模型
  2. `7.2` descendant scope
  3. `7.3` budget delegation
  4. `7.4` member allocation governance
  5. `7.5` request approval fulfillment
  6. `7.6` audit and DingTalk notification

#### QA

- 新增负责人优先级、跨层 scope、ancestor -> descendant transfer、审批幂等、通知失败隔离、审计完整性测试。

### Success Criteria

1. 新治理能力通过 `Epic 7` 承接，不污染旧 epic 完成定义。
2. 部门负责人可在自身治理范围内完成预算治理，不再完全依赖全局管理员。
3. 查询层面可选择“仅当前部门 / 包含子部门”。
4. 上级预算可下拨到任意后代部门预算池。
5. 员工可发起申请，部门负责人可单步审批并自动完成分配。
6. 调整、取消、回收、审批与通知均可追溯。

## 7. Final Recommendation

- **是否新增 Epic 7：是，强烈建议。**
- **推荐路径：Hybrid。**
- **scope classification：Moderate。**
- **本轮 P0：Epic 7 的 7.1 ~ 7.6。**
- **不建议本轮做的事：复杂多级审批、代理审批、会签流、新预算类型、新计费策略、relay/billing 主链路重写。**

本次 Correct Course 的本质，不是修补一个错误实现，而是在既有企业能力成立后，补齐真正可用的治理闭环。最稳妥的做法是：**保留现有基础能力与历史状态，引入一个新的 follow-up epic，把分层预算治理与额度审批作为治理层增强来做。**
