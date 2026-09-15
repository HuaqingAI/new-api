---
status: approved
project: new-api
created: 2026-06-03
workflow: bmad-correct-course
mode: batch
---

# Sprint Change Proposal: 收紧预算池创建权限，并补齐预算池生命周期治理闭环

## 1. Issue Summary

### Problem

当前企业预算池存在两个耦合缺口：

1. 权限边界错误：部门管理员目前可以直接创建本部门预算池，而业务上预算池应由企业管理员统一建立；部门管理员只应在既有预算池基础上向下级部门分配本部门额度。
2. 生命周期闭环缺失：预算池创建后没有正式的暂停、恢复、扩缩容治理动作，导致“停用旧池并新建”只停留在口头约束层面。

### Trigger

- 触发问题的故事范围：`Story 3.1 创建部门预算池` 与后续 `Story 7A.3 放宽预算池类型约束至单池不可变`
- 触发方式：用户在真实使用中发现预算池创建后没有任何后续治理入口

### Evidence

- PRD 明确规定：单个预算池实例类型不可切换，如需变更必须“停用旧池并新建”
  - `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md:248-252`
- PRD / NFR 已多次出现“扩容”语义，说明容量变更不是凭空新增
  - `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md:118`
  - `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md:236`
  - `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md:533`
- 当前预算池创建接口挂在 `EnterpriseDepartmentAdmin("id")`，意味着部门管理员也可创建预算池
  - `router/enterprise-router.go:20`
  - `middleware/enterprise_dept_admin.go:15-20`
- 当前仓库中可直接落地的更高权限守卫是 `middleware.EnterpriseAdmin()`，并不存在独立的 `EnterpriseSystemAdmin()` 守卫；提案中的“系统管理员”必须映射到现有权限模型
  - `middleware/enterprise_admin.go:9-17`
- 当前预算委派界面和文案持续使用 “descendant / 后代部门” 话术，业务语义不够自然
  - `web/default/src/features/enterprise-organization/index.tsx:2838-2940`
- 现有后端已有预算容量增减 helper，可作为扩缩容实现基础
  - `service/enterprise/budget_capacity.go:91-109`
  - `service/enterprise/budget_capacity.go:113-131`
- `epics.md` 与 Story 3.1 只覆盖创建预算池，没有编辑/停用预算池故事
  - `_bmad-output/planning-artifacts/epics.md:439-462`
  - `_bmad-output/implementation-artifacts/3-1-create-department-budget-pool.md:15-49`
- 后端路由仅存在预算池 `GET/POST`
  - `router/enterprise-router.go:17-20`
- 控制器和服务仅实现 `Create/Get/List/Detail`
  - `controller/enterprise/department_budget.go:16-222`
  - `service/enterprise/department_budget.go:135-325`
- 预算池模型已存在 `active / paused / revoked / expired` 状态，可优先复用而不必新造一套状态词
  - `model/enterprise/department_budget.go:10-37`
- `wallet_state_sync_task` 已能扫描 `paused/revoked/expired` 父池并联动子 allocation / wallet
  - `service/enterprise/wallet_state_sync_task.go:18-85`
- 前端 API 与页面没有预算池停用/替换入口
  - `web/default/src/features/enterprise-organization/api.ts:324-378`
  - `web/default/src/features/enterprise-organization/index.tsx:2160-2183`
  - `web/default/src/features/enterprise-organization/index.tsx:2790-2821`
- 已有调查结论确认：这是“需求要求类型不可直接修改”与“停用旧池治理动作未实现”的叠加缺口
  - `_bmad-output/implementation-artifacts/investigations/budget-pool-editability-investigation.md`

## 2. Impact Analysis

### Checklist Summary

- `1.1` `[x] Done`：触发故事链已定位到 3.1 / 7A.3
- `1.2` `[x] Done`：问题类型属于“原始需求只做了创建，后续治理闭环遗漏”
- `1.3` `[x] Done`：已有代码、PRD 和调查文档证据
- `2.1` `[x] Done`：现有 Epic 7B 可继续承接
- `2.2` `[x] Done`：建议在现有 Epic 7B 下增补 / 收敛 follow-up stories，而不是重开 7A
- `2.3` `[x] Done`：未来预算委派、额度申请、预算池展示都会受益于该闭环
- `2.4` `[x] Done`：无需新增独立 epic
- `2.5` `[x] Done`：无需调整 epic 顺序，只需在 7B 收敛 story 定义
- `3.1` `[x] Done`：PRD 需补强“停用旧池并新建”的可执行治理要求
- `3.2` `[x] Done`：Architecture 需补预算池状态机、权限矩阵与联动语义
- `3.3` `[x] Done`：UI/UX 需补暂停/恢复/扩缩容入口、风险提示与替换引导
- `3.4` `[x] Done`：测试、审计和 sprint-status 需要同步更新
- `4.1` `[x] Viable`：直接在现有 Epic 7B 收敛 story，成本中等，风险可控
- `4.2` `[x] Not viable`：不建议直接开放“预算池任意编辑”
- `4.3` `[x] Done`：推荐先推进 `7B.7` 与收敛后的 `7B.6`，再视情况拆分 replacement phase
- `6.3` `[x] Done`：本次由用户明确要求继续移交 CC 收敛 story

### Epic Impact

本次问题不否定 `Epic 7A` 的结论。`7A.3` 解决的是“同部门可多池并存，但单池类型不可变”；本次缺口分成两类：

- 预算池应由企业管理员统一创建，而不是部门管理员自建
- 已创建预算池缺少暂停/恢复、扩缩容与后续替换闭环

建议：

- 不重开 `Epic 7A`
- 继续在 `Epic 7B` 下承接，但把 `7B.6 / 7B.7` 收敛成与当前权限模型、状态模型一致的故事定义

原因：

- `7B` 已经承接 quota request、预算池可辨识性、wallet 展示语义与治理显示层修正
- 本次问题同样属于“治理闭环缺口”，与 7B 的语义一致
- 不会混淆 7A 的历史完成定义

### PRD Impact

PRD 当前只明确写到：

- 预算池类型不可切换
- 如需变更，必须停用旧池并新建
- 铁律在“创建/扩容/变更子分配前”强制校验

但没有继续规定：

- “系统管理员”在当前系统中对应哪个真实权限角色
- 谁可以暂停 / 恢复 / 扩容 / 缩容预算池
- 哪些状态下允许这些动作
- 哪些容量字段允许原地扩缩容
- 扩容/缩容分别如何回写 `total_quota / cycle_quota / remaining / allocated_total`
- 缩容时何时应拒绝，以避免突破既有承诺或已消费事实
- 停用后子 wallet / allocation / delegation / quota request 的联动语义
- 前端是否必须提供“停用后新建”的替换引导

因此 PRD 需要补强一个 follow-up FR 或 addendum，至少明确：

1. “停用旧池并新建”不是口头约束，而是正式治理动作。
2. 容量字段允许受控扩缩容，但 `type` 仍不可原地修改。
3. 当前系统中的“预算池控制面”由 `EnterpriseAdmin` 权限承担，除非后续单独引入更细分的 enterprise-system-admin 角色。

### Architecture Impact

Architecture 需要补四类内容：

1. 权限矩阵
   - `EnterpriseAdmin`：可创建、暂停、恢复、扩容、缩容预算池
   - `EnterpriseDepartmentAdmin`：可查看预算池并做下级分配/成员分配，不可创建或修改预算池生命周期
2. 预算池状态机
   - 复用现有 `active / paused / revoked / expired`
   - 本期 story 只正式交付 `active <-> paused` 与 active 状态下的扩缩容
   - `revoked / expired` 继续沿用既有语义，不在本期重新发明新状态
3. 预算池扩缩容流
   - 管理员 UI -> resize action
   - `balance` 池增容：`total_quota += delta`, `remaining += delta`
   - `subscription` 池增容：`cycle_quota += delta`, `remaining += delta`
   - 缩容仅允许消减未承诺/未使用头寸，不能突破当前 `remaining` 与 `allocated_total` 语义
   - 所有校验与写入必须在同一事务内完成
4. 联动语义与权威边界
   - 运行时入口以父预算池 `status == active` 为同步校验权威
   - `wallet_state_sync_task` 继续承担对既有子 allocation / wallet 的收敛与补偿
   - 需要明确预算池暂停/恢复/扩缩容如何与已有 allocation revoke、delegation supersede、quota request active budget 校验协同

### Technical Impact

预计涉及：

- `router/enterprise-router.go`
- `controller/enterprise/department_budget.go`
- `service/enterprise/department_budget.go`
- `service/enterprise/admin_action.go`
- `service/enterprise/wallet_state_sync_task.go`（若需补边界）
- `web/default/src/features/enterprise-organization/api.ts`
- `web/default/src/features/enterprise-organization/types.ts`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `i18n/keys.go`
- `i18n/locales/{en,zh-CN,zh-TW}.yaml`
- `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`

## 3. Domain Invariants

为避免 story 在实现阶段再次发散，本提案先显式补充领域约束：

1. 预算池是部门预算控制面的聚合根；任一预算池实例的 `type` 一经创建不可原地修改。
2. 预算池生命周期必须映射到现有状态语义；本期不引入第四套近义状态词。
3. 预算池暂停后，不再允许其参与新的 allocation、delegation、quota request。
4. 预算池恢复后，可重新参与新的 allocation、delegation、quota request。
5. 预算池扩缩容只允许修改 `balance.total_quota` 或 `subscription.cycle_quota`，并同步维护 `remaining`。
6. 缩容不得突破既有已承诺 / 已分配 / 已使用事实；失败必须返回稳定错误并且无副作用。
7. 预算池生命周期动作不得物理删除历史 allocation、delegation、wallet、quota request 事实。
8. 所有预算池生命周期动作必须写入治理审计，并且具有可验证的前后快照。

## 4. Recommended Approach

### Selected Path

`Option 1: Direct Adjustment`

### Recommendation

在 `Epic 7B` 下保留两个落点，但收敛范围如下：

1. `Story 7B.6`：企业管理员预算池生命周期治理，**本期交付暂停、恢复、扩缩容**，并把“停用旧池并新建”明确为后续替换指引而不是本 story 内实现完整 replacement workflow。
2. `Story 7B.7`：收紧预算池创建权限，部门管理员只保留向下级部门分配本部门预算的能力，并把相关 UI/文案改成更贴近业务的话术。

而不是直接扩成“预算池可任意编辑”，也不是在当前 story 中硬塞完整 replacement cutover。

### Rationale

- 这与现有 PRD / NFR 更一致：类型仍不可直接修改，但容量扩缩容应被纳入正式治理能力
- 能先补齐“预算池控制面收口”和“容量日常调优”两条高频路径
- 风险低于把暂停/扩缩容/替换全部绑定在同一个 story 内
- 能复用现有 `paused/revoked/expired` 状态枚举、`wallet_state_sync_task` 和前端预算池列表/详情

### Effort / Risk

- Effort: Medium
- Risk: Medium

主要风险在于：

- 父池暂停后子 wallet / allocations 的终态联动边界
- 同部门多池并存时，默认选中 / active pool 校验与 quota request capability 的一致性
- 以 `EnterpriseAdmin` 映射“系统管理员”会带来比预想更宽的可操作面，后续若需更细粒度 enterprise role，还需再拆权限

## 5. Detailed Change Proposals

### Change A: 收敛 Story 7B.6 范围，先交付可测的生命周期动作

**OLD**

`Story 7B.6` 以“停用、扩缩容、替换”三类能力一起交付

**NEW**

收敛为：

`### Story 7B.6: 企业管理员补齐预算池暂停、恢复与扩缩容治理闭环`

建议 story 定义：

```md
### Story 7B.6: 企业管理员补齐预算池暂停、恢复与扩缩容治理闭环

**Requirements:** follow-up to FR9, FR10, FR12, FR23, FR25

As an 企业管理员（映射到当前 `EnterpriseAdmin` 权限）,
I want 对已创建预算池执行正式暂停、恢复、容量扩缩容，
So that 预算池生命周期保持集中治理，同时部门管理员只消费和分配既有预算，不自行造池。

**Acceptance Criteria:**

**Given** 某预算池已创建且当前不应再继续对外分配
**When** 企业管理员执行暂停动作
**Then** 系统将该预算池状态更新为 `paused` 并记录治理审计
**And** 前端不再把该池作为新的 allocation / delegation / quota request 可选 active 池。

**Given** 某预算池当前处于 `paused`
**When** 企业管理员执行恢复动作
**Then** 系统将该预算池状态恢复为 `active`
**And** 后续新建 allocation / delegation / quota request 可重新选择该池。

**Given** 某预算池已存在派生 wallet、allocation 或预算委派事实
**When** 企业管理员暂停该预算池
**Then** 系统按既有父池状态联动语义处理子项
**And** 不物理删除既有事实，历史链路仍可追溯。

**Given** 某预算池仍然有效且企业管理员只需要调整容量
**When** 企业管理员执行扩容或缩容
**Then** 系统允许在不改变 `type` 的前提下原地修改容量字段
**And** 缩容不得突破当前已承诺 / 已使用边界，失败时返回可测试错误且无副作用。

**Given** 企业管理员需要调整原预算池的类型
**When** 企业管理员查看当前预算池治理入口
**Then** UI 明确表达“类型不可编辑，需暂停旧池并新建新池”
**And** 本故事不要求实现完整 successor cutover workflow。
```

### Change B: PRD / Addendum 补强预算池治理要求与权限映射

**OLD**

```md
- 单个部门预算池实例一经创建不可在 `balance` 与 `subscription` 间切换；如需变更，必须先停用旧池并新建。
```

**NEW**

```md
- 单个部门预算池实例一经创建不可在 `balance` 与 `subscription` 间切换；如需变更，必须先停用旧池并新建。
- 当前系统中的预算池控制面由 `EnterpriseAdmin` 权限承担；`EnterpriseDepartmentAdmin` 不得直接创建或修改预算池生命周期。
- 系统必须提供正式的预算池暂停 / 恢复治理动作；暂停后该预算池不得再作为新的 allocation、budget delegation 或 quota request 的 active 目标池。
- 系统必须允许预算池容量受控扩缩容：`balance` 池调整 `total_quota`，`subscription` 池调整 `cycle_quota`；扩缩容不得打破既有 `remaining / allocated_total` 铁律。
- 预算池生命周期动作不得物理删除既有 allocation、delegation、wallet 或 request 事实；历史链路必须保持可追溯，并沿用既有父池状态联动语义。
- “停用旧池并新建”在本期至少必须落实为明确 UI/UX 引导与治理规则；若要实现完整 successor 切换流程，需单独立项或在后续 phase 明确定义。
```

### Change C: 收敛 Story 7B.7 为权限矩阵收口 + 文案修正

**OLD**

提案以“系统管理员”泛词描述创建权限收敛

**NEW**

建议 story 定义：

```md
### Story 7B.7: 收紧预算池创建权限并重命名下级部门预算分配语义

**Requirements:** follow-up to FR9, FR23, FR25

As an 企业管理员或部门管理员,
I want 由企业管理员统一创建预算池，而部门管理员只基于当前部门已有预算池继续向下级部门分配预算,
So that 预算池供给权与预算使用 / 分配权分离，治理边界更清晰，UI 文案也更符合业务心智。

**Acceptance Criteria:**

**Given** 普通部门管理员进入本部门预算工作区
**When** 页面渲染预算池治理入口
**Then** 不显示 “Create Budget Pool” 入口
**And** 其只能查看既有预算池并执行被授权的下级预算分配或成员分配动作。

**Given** 部门管理员直接调用 `POST /departments/:id/budget`
**When** 后端执行权限校验
**Then** 请求被拒绝并返回项目既有错误响应语义（HTTP 200 + `success=false`）
**And** 错误消息与 `EnterpriseAdmin` 权限要求一致。

**Given** 企业管理员进入任意部门预算工作区
**When** 需要为该部门建立预算池
**Then** 系统允许创建预算池
**And** 创建接口权限明确从 `EnterpriseDepartmentAdmin("id")` 收敛到 `EnterpriseAdmin()`。

**Given** 负责人或部门管理员把当前部门预算继续分给下级部门
**When** 页面展示该治理动作
**Then** UI 与文案不再强调“descendant / 后代部门”
**And** 改为“向下级部门分配预算”或等价更自然的话术。
```

### Change D: Architecture 增补权限矩阵、状态机与一致性约束

**OLD**

架构中只有预算池创建、allocation、delegation、wallet state sync、quota request 等流程，没有预算池生命周期治理流。

**NEW**

新增：

```md
#### RBAC Matrix

- `EnterpriseAdmin()`：
  - 可执行 Create Budget Pool
  - 可执行 Pause Budget Pool
  - 可执行 Resume Budget Pool
  - 可执行 Expand / Shrink Budget Pool
- `EnterpriseDepartmentAdmin()`：
  - 可查看预算池
  - 可执行下级预算分配、成员分配
  - 不可执行预算池 create / pause / resume / resize

#### Budget Pool State Machine

- Reuse: `active`, `paused`, `revoked`, `expired`
- In-sprint transitions:
  - `active -> paused`
  - `paused -> active`
- Out of scope for this story:
  - full replacement successor cutover
  - introducing new replacement-only status

#### Command Flow

企业管理员 UI
→ budget pool pause / resume / resize action
→ 权限校验（`EnterpriseAdmin()`）
→ 目标池身份校验（`tenant_id + department_id + budget_id`）
→ DB 事务内完成状态 / 容量校验与写入
→ 写 `enterprise_admin_actions`
→ 刷新 budget list / detail / active budget filters
→ `wallet_state_sync_task` 负责对子 allocation / wallet 的收敛与补偿

#### Consistency Rules

- 所有新建 allocation / delegation / quota request 继续以 `budget.status == active` 为同步入口校验权威
- `wallet_state_sync_task` 不是权限替代物，而是父池状态传播与对账补偿器
- 缩容必须在事务内校验新容量仍覆盖 `allocated_total` 与既有已使用事实
- 并发修改需保证不会把预算池压到负可用或让同一操作产生双重副作用
```

### Change E: 明确 replacement 的处理方式

**OLD**

把 replacement 直接放入 `7B.6` 当前 sprint 范围

**NEW**

- 本提案保留“停用旧池并新建”的治理原则
- 本 sprint 交付：
  - 预算池暂停 / 恢复
  - 预算池扩缩容
  - 类型不可编辑时的明确替换引导
- 本 sprint 不强制交付：
  - successor budget pool 指针切换
  - 旧池 / 新池引用迁移
  - 完整 cutover / rollback workflow

如后续需要真正的 replacement workflow，建议新增独立 story，单独定义：

- old_pool / new_pool 关系
- 切换时点
- 回滚策略
- 历史链路与审计模型

## 6. Implementation Handoff

### Scope

`Moderate`

这是一个明确的 backlog 收敛，不需要推翻现有 epic 结构，但需要：

- 更新 `epics.md` 中 `7B.6 / 7B.7` 的文本口径
- 更新 `_bmad-output/implementation-artifacts/sprint-status.yaml`
- 将 `7B.6 / 7B.7` 的跟踪状态调整为 `backlog`
- 后续由 dev 按 story 实施

### Handoff

- CC / PO 责任：确认本次采用“暂停 / 恢复 / 扩缩容 + 替换引导”而非“开放式编辑预算池”
- Dev 责任：按 story 实现预算池生命周期动作、联动与前端入口

### Success Criteria

- 预算池创建权从 `EnterpriseDepartmentAdmin` 收敛到 `EnterpriseAdmin`
- 预算池支持正式暂停与恢复
- 预算池支持受控扩缩容
- 暂停池不再参与新的 allocation / delegation / quota request
- 缩容不会突破已承诺 / 已使用边界
- 历史 wallet / allocation / delegation / request 事实不丢失
- 部门管理员 UI 不再显示预算池创建入口
- 前端明确区分“调整当前池容量”和“类型不可编辑，需停用旧池并新建新池”
- “后代部门委派 / descendant budget delegation” 话术被替换为更自然的“向下级部门分配预算”表述
- 后端、前端、i18n、测试、审计对同一权限模型与状态语义保持一致

## 7. Approved Next Steps

1. 更新 `Story 7B.6` 为企业管理员预算池暂停、恢复与扩缩容治理
2. 保持 `Story 7B.7`，但把“系统管理员”口径收敛为当前可实现的 `EnterpriseAdmin`
3. 在 `_bmad-output/implementation-artifacts/sprint-status.yaml` 中将 `7B.6 / 7B.7` 调整为 `backlog`
4. 后续由 dev 分 story 接手实现
