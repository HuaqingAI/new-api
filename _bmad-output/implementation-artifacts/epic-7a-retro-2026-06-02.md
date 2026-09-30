# Epic 7A Retrospective — 企业治理体验纠偏与预算池约束校正

日期：2026-06-02
项目：new-api
主持：Amelia（Senior Software Engineer）
参与角色：John（Product Manager）、Sally（UX Designer）、Winston（System Architect）、Murat（Test Architect）、Paige（Technical Writer）、hth（Project Lead，自动参与 / YOLO 模式）

> 本复盘按 BMAD retrospective workflow 自动执行。所有需要用户输入的确认、观察与决策均基于 sprint status、Story 7A.1–7A.3、Epic 7 回顾、PRD、架构文档与实际实现记录自主完成。

## 1. Epic Discovery 与完成状态

Amelia (Developer): "我先从 sprint-status 识别完成的 Epic。Epic 7A 的三个故事都已标记为 done；本次自动确认复盘对象为 Epic 7A。"

Alice (Product Owner): "状态上还有一个小偏差：`epic-7a` 自身仍停在 `in-progress`，但所有子故事已经完成。复盘结束后应把 epic 与 retrospective 状态同步为 done。"

### Sprint Status 事实

- `7a-1-complete-enterprise-governance-i18n-and-action-label-mapping`: done
- `7a-2-move-employee-quota-request-entry-to-balance-wallet-user-flow`: done
- `7a-3-relax-budget-pool-type-locking-to-per-budget-invariance`: done
- `epic-7a-retrospective`: optional → 本次复盘后应更新为 done
- `epic-7a`: in-progress → 所有故事完成后应同步为 done

## 2. Epic 目标与交付摘要

Epic 7A 是 Epic 7 完成后的治理体验与模型约束纠偏，核心目标不是新增大型治理能力，而是修正已交付能力中的三类偏差：

1. 企业治理工作台的 i18n、动作标签与通知投递显示不完整。
2. 员工额度申请入口放在治理工作台，不符合普通员工从余额 / 钱包补额度的使用心智。
3. 预算池类型不可变约束被错误扩大为部门级锁定，而 PRD / 架构要求是单预算池实例类型不可变。

### 交付结果

- 完成 3/3 个故事，故事级验收均通过。
- 未发现未解决的 HIGH / MEDIUM review defect。
- 治理体验、普通员工申请路径、预算池类型约束均已与规划纠偏目标对齐。
- 遗留风险主要集中在流程与回归基础设施：状态源同步、orchestration 记录陈旧、全量 Go regression 仍有既有噪音。

## 3. Story 深度分析

### Story 7A.1 — 补齐企业治理工作台 i18n 与动作标签映射

**交付内容**

- 以 `service/enterprise/admin_action.go` 与 `service/enterprise/governance_notification.go` 的 action type 常量为事实源，补齐治理 action label 映射。
- 未知 action 不再直接显示完整内部 key，改为安全兜底。
- 通知投递状态与 governance timeline / delivery-only records 渲染修复。
- 补齐 `en/zh/fr/ja/ru/vi` locale 的治理文案。
- 增加非英文 locale、action matrix、未知 action 与 delivery-only case 回归测试。

**Review 发现并修复**

- 通知投递独立记录原先未展示动作摘要。
- timeline 外通知筛选逻辑使用错误 map，可能吞掉孤立 delivery 记录。
- `Attempt {{count}}/{{max}}` locale 缺 key。

**验证证据**

- `bun test src/features/enterprise-organization/enterprise-organization.test.tsx` 通过。
- `bun run typecheck` 通过。
- `bun run i18n:sync` 通过。

### Story 7A.2 — 将员工额度申请入口迁移到余额 / 钱包用户场景

**交付内容**

- 在钱包页新增普通员工“申请额度”卡片。
- 采用钱包页内卡片表单，不新增后端 API。
- 复用既有 quota request workflow：部门选择、预算池 capability、提交申请与治理闭环。
- 普通员工仍必须显式选择目标部门与目标预算池。
- 企业组织页保留负责人 / 管理员审批、治理时间线与通知投递视图。

**Review 发现并修复**

- 普通员工无法读取自己的部门成员关系，导致钱包入口无法进入可提交状态。
- 钱包提交成功后只失效 quota request cache，治理时间线与通知 cache 可能滞后。
- 新增翻译键曾被写到 locale JSON 根级，绕开 `translation` namespace。

**验证证据**

- `cd web/default && bun run typecheck` 通过。
- `go test ./controller/enterprise` 通过。
- `cd web/default && bun run test:e2e` 通过。
- `go test ./controller/enterprise ./service/enterprise` 中 controller 通过；service 包存在既有 SQLite 并发 / 唯一约束噪音，未作为本故事阻断。

### Story 7A.3 — 放宽预算池类型约束至单池不可变

**交付内容**

- 移除 `DepartmentBudgetService.Create` 中基于 latest budget 的部门级类型锁定。
- 同一部门现在可创建 `balance` 与 `subscription` 两类预算池。
- 保留创建参数校验与单预算池实例类型不可变语义。
- 确认预算委派、员工申请、成员分配均按显式选中的 budget id 校验，而不是按部门整体类型推断。
- 增加服务、控制器、API 与前端预算池混合类型回归测试。

**Review 结论**

- 无 confirmed HIGH / MEDIUM / LOW defect。
- File List 与 story bookkeeping 已补齐。
- AC1–AC4 均有实现或保持不变的证据。

**验证证据**

- `go test ./service/enterprise -run 'DepartmentBudget|BudgetDelegation|QuotaRequest|QuotaAllocation'` 通过。
- `go test ./controller/enterprise -run 'DepartmentBudget|BudgetDelegation|QuotaRequest|QuotaAllocation'` 通过。
- `go test ./tests/api -run 'TestEnterpriseDepartmentBudgetAPIAllowsMixedTypeCreatesForDepartment' -count=1` 通过。
- `cd web/default && bun run typecheck` 通过。
- `cd web/default && bun run test:e2e` 通过。
- `go test ./...` 仍受既有 model package SQLite table setup 问题影响，非 7A.3 引入。

## 4. 复盘讨论摘要

Amelia (Developer): "我们先看做得好的地方。Epic 7A 的共同点很清晰：每个故事都围绕已经发现的产品 / 架构偏差做定点纠偏，没有把范围扩成新的大改。"

John (Product Manager): "产品侧最大收获是 7A.2。员工从钱包页申请额度，比进入企业治理工作台更符合任务场景；而审批闭环仍留在治理工作台，角色边界更清楚。"

Winston (System Architect): "架构侧最大收获是 7A.3。团队把‘类型不可变’重新收敛到单个预算池实例，没有破坏 delegation、quota request、allocation 这些显式 budget id 链路。这个边界非常重要。"

Murat (Test Architect): "质量上，review 找到的问题都具体、可复现、可修复。尤其是权限、缓存失效、i18n namespace 这些问题，如果不做 review，很容易形成真实用户路径故障。"

Sally (UX Designer): "体验纠偏很明显。治理工作台显示用户可读标签，员工在钱包里申请额度，这两个点都减少了用户理解负担。"

hth (Project Lead): "自动判断：本 Epic 最大亮点是纠偏效率高、验收证据清晰；最大短板是流程状态同步仍然落后于实现。"

## 5. 主要成功点

1. **纠偏目标聚焦**：三个故事都服务于 Epic 7 后续纠偏，不扩散到无关 relay、billing、log 或新预算类型能力。
2. **事实源边界清楚**：7A.1 使用后端 action 常量作为 UI label 映射事实源；7A.3 使用 PRD / architecture 对“单池不可变”的定义校正实现。
3. **角色体验更合理**：普通员工入口迁移到钱包；负责人 / 管理员治理闭环保留在企业组织页。
4. **Review 有真实价值**：7A.1 和 7A.2 review 均发现并修复了会影响用户路径的缺陷。
5. **显式选择预算池的架构模式得到巩固**：预算委派、额度申请与成员分配没有回退到“部门默认 / latest budget”推断。

## 6. 挑战与根因

### 挑战 1：源状态不同步

- 事实：所有 7A stories 已 done，但 `epic-7a` 仍为 `in-progress`；orchestration 文档也仍保留旧状态。
- 根因：story status、sprint status、story-automator orchestration 不是同一个自动同步事实源。
- 影响：复盘、规划与自动化判断需要额外交叉验证。

### 挑战 2：测试体系仍偏定向验证

- 事实：每个 story 的 targeted validation 足够强，但 full Go regression 仍有既有 model setup 噪音。
- 根因：企业线缺少稳定的统一 regression entry point，且全量测试中仍混入与当前 epic 无关的环境 / fixture 问题。
- 影响：故事级“通过”与仓库级“全绿”之间存在解释成本。

### 挑战 3：i18n 与 cache namespace 容易在 review 阶段才暴露

- 事实：7A.1 / 7A.2 都出现 locale 或 cache invalidation 相关 review 修复。
- 根因：前端跨页面治理闭环依赖 query invalidation、locale namespace 与多语言覆盖，单组件开发时容易漏看。
- 影响：后续企业治理功能需要更早把这些检查列入故事验收模板。

### 挑战 4：历史规划文本与实现纠偏之间存在追溯成本

- 事实：7A 本身来自 Correct Course / follow-up planning；部分历史 story wording 仍保留旧心智，需要读 addendum 与 7A stories 才能理解最终现实。
- 根因：规划纠偏文件、story 文件与架构 / PRD addendum 分散。
- 影响：新开发者进入时需要多跳阅读才能理解最终约束。

## 7. 前一次 retrospective 的跟进情况

Epic 7 retrospective 提出的关键改进包括：

- 强化 story-automator source-of-truth sync。
- 整理企业线统一 regression entry point。
- 保持 story review 对治理闭环的高强度验证。

### 跟进评估

- ✅ **Review 强度保持**：7A.1 / 7A.2 / 7A.3 均有 senior review 记录，且 review 修复了真实问题。
- ⏳ **统一 regression entry point 部分推进**：定向测试覆盖更完整，但 full Go regression 噪音仍未清理。
- ❌ **source-of-truth sync 未完全解决**：`epic-7a` 状态与 orchestration 状态仍需人工 / workflow 同步。

Amelia (Developer): "这不是个人失误，而是系统反馈：我们已经把 story 质量做好了，但状态与回归入口还没有同样成熟。"

## 8. 下一阶段准备

当前规划资料中未发现传统 Epic 8。Sprint status 显示后续 backlog 转向 `agent-platform-epic-1` 及其后续 agent-platform epics。

### 对后续工作的依赖影响

- Epic 7A 不阻塞 agent-platform backlog，但它暴露出的流程问题会影响任何后续自动化开发：状态同步、回归入口、文档追溯。
- 企业治理能力自身已具备继续作为产品能力维护的基础：钱包入口、审批闭环、通知投递、预算池多类型并存均已形成稳定边界。

## 9. Action Items

### Process Improvements

1. **同步 Epic 7A sprint status 与 retrospective 状态**
   - Owner: Amelia
   - Success criteria: `epic-7a` 与 `epic-7a-retrospective` 均为 `done`，comments / STATUS DEFINITIONS 保留。

2. **补齐 story-automator orchestration 状态同步机制或标注其为历史快照**
   - Owner: Amelia + Winston
   - Success criteria: 后续使用者不会把陈旧 orchestration 文件误认为当前执行状态。

3. **把 i18n namespace、query invalidation、普通用户自范围权限加入企业前端 review checklist**
   - Owner: Murat
   - Success criteria: 后续企业治理 story review 明确检查 locale `translation` namespace、所有支持语言 key、跨页面 cache invalidation、自身 / 他人权限边界。

### Technical / Quality Actions

1. **建立企业线稳定 regression entry point**
   - Owner: Murat + Amelia
   - Success criteria: 企业 controller/service/API/frontend 关键路径可通过单一命令或清晰命令组稳定验证，且不被无关 model fixture 噪音污染。

2. **隔离或修复 full Go regression 中既有 SQLite setup 噪音**
   - Owner: Amelia
   - Success criteria: 当前已知 `model` package missing-table / fixture 问题被单独记录、隔离或修复，不再影响企业故事验收表述。

3. **保留显式 budget id 选择作为企业预算链路的不变式**
   - Owner: Winston
   - Success criteria: 后续 delegation、quota request、allocation 相关 story 均不得回退为 latest budget / department default 推断。

### Documentation Actions

1. **验证 PRD / addendum / architecture 是否已反映最终实现**
   - Owner: Paige
   - Success criteria: 仅在发现与代码不一致时更新；若文档已经匹配实现，则记录“无需更新”。

2. **验证 README / configuration docs 是否存在企业治理入口或预算池约束的过时说明**
   - Owner: Paige
   - Success criteria: 若 README / config docs 未覆盖该功能或与代码一致，则不做无意义改动。

## 10. Readiness Assessment

- Testing & Quality: 故事级验证充分；统一 regression 入口仍需改进。
- Deployment: 未发现 story 记录中的部署阻断项；本复盘不假设外部生产状态。
- Stakeholder Acceptance: 从 BMAD story AC 与 review 记录看，Epic 7A 目标已满足。
- Technical Health: 企业治理核心边界更清晰；流程与测试基础设施存在可管理改进项。
- Unresolved Blockers: 无阻塞 Epic 7A 完成的产品 / 架构 blocker；存在状态同步与 full regression 噪音两类后续行动项。

## 11. Significant Discovery Check

本次复盘未发现需要重新定义后续 agent-platform backlog 的重大产品方向变更。

但发现两项对后续开发流程有影响的系统性问题：

1. **状态源同步不足**：story done 不自动保证 epic / orchestration done。
2. **回归入口不稳定**：targeted tests 与 full-suite 噪音之间仍需明确边界。

结论：无需修改下一业务 Epic 定义；需要强化流程与质量基础设施。

## 12. Commitments and Closure

Amelia (Developer): "Epic 7A 的产品纠偏完成了。我们保留了治理事实源、修正了员工入口、放宽了正确层级的预算池类型约束。"

Winston (System Architect): "架构方向是稳的：单池不可变，部门可多池，操作按显式 budget id。这个约束后续必须继续守住。"

Murat (Test Architect): "质量动作也明确：企业线需要稳定 regression 入口，不再让每个 story 重复解释既有 full-suite 噪音。"

Paige (Technical Writer): "我会按实现核对文档，只改真实偏差，不制造文档噪音。"

hth (Project Lead): "自动决策：接受本复盘结论，更新 sprint status，并进入文档差异核验。"

**Retrospective outcome:** Complete.
