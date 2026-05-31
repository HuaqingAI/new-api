---
status: draft
project: new-api
created: 2026-05-31
workflow: bmad-correct-course
mode: batch
trigger_source: user request on 2026-05-31 to re-plan downstream agent capabilities as a generic platform, with Cherry Studio enterprise as the first consumer
recommended_scope: major
proposed_changes_to:
  - _bmad-output/planning-artifacts/briefs/brief-new-api-2026-05-27/brief.md
  - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md
  - _bmad-output/planning-artifacts/epics.md
  - _bmad-output/planning-artifacts/architecture.md
  - _bmad-output/implementation-artifacts/sprint-status.yaml
proposed_new_artifacts:
  - _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md
  - _bmad-output/planning-artifacts/epics-agent-platform.md
  - _bmad-output/planning-artifacts/architecture-agent-platform.md
---

# Sprint Change Proposal: 通用 Agent 能力平台化重规划

## 1. Issue Summary

### Trigger

本次 Correct Course 由 2026-05-31 的新需求说明触发。触发源不是实现缺陷，而是规划边界判断发生了变化：

- 现有 Cherry Studio 相关输入文档，主要把上游需求表述为“给下游客户端冻结 API 契约”。
- 用户明确补充：这批需求必须是**通用的下游 agent 工具能力**，Cherry Studio 企业扩展版只是第一位消费者，后续还要服务 Codex、cc switch skill 库等其它下游。
- 用户进一步指出：`skill 库 / 知识库 / agent 库` 不是“只有 API 的资源”，其上游必须同时具备**前端管理页面、后端管理逻辑、生命周期治理和可见性控制**。

### Core Problem

问题类型：**战略重规划 / 产品边界修正**。

当前规划的根本问题不是“接口字段不全”，而是把一个应当作为**上游能力平台（control plane）**来规划的领域，误缩减成了“面向单一下游客户端的 API 契约整理”。

如果继续沿着 API-only 路径推进，会导致以下系统性偏差：

- 把 Cherry Studio 误当成唯一消费方，损失后续服务 Codex / cc switch / 其它 agent 工具的通用性。
- 把 Skill / Knowledge / Agent 误当成“可拉取对象列表”，遗漏上游作为 source of truth 所需的管理后台、发布、版本、撤销、授权、审计、缓存和状态治理。
- 把“平台能力扩展”留在模糊 roadmap 层，不足以形成可执行的 PRD / Epics / Architecture。
- 后续真正实施时，前端页面、后台权限、资源生命周期、客户端绑定、错误状态矩阵等隐含工作会集中暴露，造成 epic 粒度失真。

### Evidence

1. 当前 Cherry Studio 输入文本把 Phase 0 重点放在 OAuth、模型发现、知识库/技能调用契约、缓存撤销和错误 envelope，默认上游已经具备模型、知识库、技能等资源及其 source of truth。
2. 现有产品简报的“主题 B：平台能力扩展”仍然是 `DeepWiki / WebChat / 图像视频入口` 叙事，尚未把“通用 agent 能力平台”定义为正式产品线。
3. 现有企业管控 PRD 明确把 `DeepWiki / 企业知识库 RAG / WebChat 增强 / 多模态入口` 排除在 V1 之外，说明这些内容无法直接硬塞进当前企业治理 PRD。
4. 当前 `epics.md` 没有任何一个 epic 为 Skill 库、Knowledge 库、Agent 库的上游管理逻辑、前端页面、客户端绑定和发布治理负责。
5. 当前 `architecture.md` 也没有为“通用下游资源平台”定义统一资源模型、库级状态机、客户端可见性、版本语义和撤销传播路径。

## 2. Impact Analysis

### 2.1 Triggering Story Assessment

- 触发故事：不是某一个 story 的验收失败。
- 更准确地说，这是对“平台扩展线”的重新定性：从“未来某些能力扩展”变为“当前必须建立的通用 agent 能力平台规划”。

结论：不能用修补单个 story 的方式处理，必须提升到 PRD / Epics / Architecture 层重规划。

### 2.2 Epic Impact Assessment

#### Existing Enterprise Governance Epics

现有企业治理 Epics 1-6 仍然有效，它们解决的是：

- 企业组织 / 部门模型
- 钉钉登录与同步
- 部门预算与 wallet
- 部门用量看板
- 风险事件与告警
- 企业治理视图重构

这些 epic 提供的是**企业基础设施底座**，不是通用 agent 能力平台本身。

结论：

- 不建议回写或推翻已有 Epic 1-6。
- 不建议把新需求拆散塞入 Epic 2-5 的尾部。
- 需要新开一条规划主线。

#### New Epic-Level Changes Required

建议新增一组独立的“Agent Platform” epics，而不是继续扩充现有企业治理 epics：

1. **AP-1 平台资源域模型与控制面基础**
2. **AP-2 客户端注册、授权、可见性与发布治理**
3. **AP-3 Skill 库管理与执行契约**
4. **AP-4 Knowledge 库管理与检索契约**
5. **AP-5 Agent 库管理与组合契约**
6. **AP-6 开放能力标准扩展层与首个消费者验证**

如后续范围继续扩大，可再拆：

7. **AP-7 运维观测、缓存撤销、诊断与回放工具**

### 2.3 PRD Impact

当前企业治理 PRD 与这批需求存在显式边界冲突：

- 当前 PRD 把知识库、DeepWiki、WebChat 增强等排除在 V1 范围之外。
- 当前 PRD 的核心目标用户是企业 IT / 管理员 / 合规负责人。
- 本次新需求的核心对象是：**平台管理员、资源维护者、客户端集成人员、下游 agent 工具消费者**。

因此不建议把这批需求直接追加到现有企业治理 PRD 主体中。

建议：

- 保留当前 PRD 作为“企业治理基础设施 PRD”。
- **新建独立 PRD：`prd-agent-platform-2026-05-31/prd.md`**。
- 在旧 brief / PRD 中只补一条 roadmap 级联动说明，避免文档职责混乱。

### 2.4 Architecture Impact

当前架构文档缺少以下关键设计面：

- 统一资源抽象：`model / knowledgeBase / skill / agent`
- 稳定 ID、版本、状态、撤销、缓存 TTL、可见性范围
- 资源主数据与“对客户端发布投影”分离
- Skill / Knowledge / Agent 的共性控制面与差异化字段
- 客户端注册与 entitlement
- 发布、下线、撤销、灰度、审计
- 通用错误 envelope 与状态矩阵
- 管理后台页面信息架构

结论：

- 现有 `architecture.md` 不应被少量增补后继续承担这条新产品线的设计。
- 需要新建 **`architecture-agent-platform.md`**，或者至少在批准后拆出独立架构文档。

### 2.5 UX / Frontend Impact

这批需求不是“纯接口”工作，必须显式纳入前端管理面：

- Skill 库页面：列表、详情、Schema、版本、发布状态、可见客户端
- Knowledge 库页面：资源元数据、调用模式、来源、状态、版本、发布状态
- Agent 库页面：定义、依赖、能力组合、版本、发布状态、可见客户端
- 客户端注册页：Cherry Studio、Codex、cc switch 等客户端的接入配置、授权状态、TTL / cache / revoke 策略
- 发布中心：草稿 / 已发布 / 已停用 / 已撤销 / 已过期资源总览
- 审计与任务页：同步、发布、撤销、重试、诊断

结论：

- 这些页面不属于现有企业组织页或部门用量页的 follow-up。
- 需要新的前端 feature 集合与导航结构。

### 2.6 Technical / Operational Impact

新增的技术域至少包括：

- 后端模型、服务、控制器、OpenAPI
- 前端管理台
- 统一发布状态机
- 客户端资源投影
- 资源版本控制
- 资源撤销与缓存失效
- Trace / requestId / error envelope
- Mock / fixture / contract tests

同时必须继续遵守项目护栏：

- 三库兼容（SQLite / MySQL / PostgreSQL）
- 管理 API 走 `docs/openapi/api.json`
- 不触碰 `relay/**` 的协议面，除非架构审批升级
- JSON 继续走 `common/json.go`

## 3. Recommended Approach

### Selected Path

**Hybrid / Major Replan**

不是在现有企业治理 PRD 里继续“补几个 FR”，而是：

1. 保留现有企业治理规划作为底座；
2. 将本次需求提升为一条新的“通用 Agent 能力平台”规划主线；
3. 以 Cherry Studio 企业扩展版为首个验证消费者设计，但所有上游对象、状态机、发布流程、错误语义都保持通用，并落在既有 OpenAPI / 开放能力体系的标准扩展层上；
4. 在平台 MVP 中同时规划 API、后台管理页面、资源生命周期和客户端集成。

### Why This Path

这是风险最低、后续可复用性最高的路径，因为它：

- 不破坏已经完成的企业治理能力；
- 不把“通用平台”错误降级为“单客户端协议适配”；
- 能正确承接用户强调的前端 + 后端 + 管理逻辑；
- 允许 Cherry Studio 先落地，但不会把领域模型绑定到 Cherry 专有语义或某个单独渠道的专属适配；
- 为 Codex / cc switch 后续接入预留统一资源模型和客户端层。

### Alternatives Considered

#### Option 1: 直接扩写当前企业治理 PRD / Epics

不推荐。

原因：

- 文档目标用户和范围已经不同。
- 会让“企业治理基础设施”和“agent 能力平台”混在一起。
- 后续拆 story 时会很难解释哪些是治理底座，哪些是平台资源域。

#### Option 2: 继续把 Cherry Studio 文本当成接口契约文档

不推荐。

原因：

- 这只能冻结消费者视角的 API 期望。
- 无法回答上游资源从哪里创建、谁管理、如何发布、如何撤销、如何审计。

#### Option 3: 等 Cherry Studio 实现后再反推平台

不推荐。

原因：

- 会把第一个消费者的特殊约束固化成平台模型。
- 后续接入 Codex / cc switch 时会暴露严重抽象问题。

### Effort / Risk / Timeline

- Effort: High
- Risk: High
- Scope classification: Major
- Timeline impact: 明显增加，需要独立排期，不应作为旧 Epic 6 的附属补丁处理

## 4. Detailed Change Proposals

### Proposal 1: 重写产品简报中“平台能力扩展”的主叙事

Artifact: `_bmad-output/planning-artifacts/briefs/brief-new-api-2026-05-27/brief.md`

OLD:

```markdown
### 🧠 主题 B：平台能力扩展

| 能力 | 描述 |
|---|---|
| 代码库 DeepWiki | ... |
| WebChat 增强 | ... |
| 图像/视频生成 | ... |
```

NEW:

```markdown
### 🧠 主题 B：通用 Agent 能力平台

| 能力 | 描述 |
|---|---|
| Skill 库 | 上游统一管理技能元数据、输入输出 Schema、版本、发布状态、客户端可见性 |
| Knowledge 库 | 上游统一管理知识资源元数据、调用模式、状态、来源、客户端发布语义 |
| Agent 库 | 上游统一管理 Agent 定义、依赖能力、版本、发布和撤销 |
| 客户端集成层 | Cherry Studio 企业扩展版为首个消费者，后续支持 Codex、cc switch skill 库等 |
| 发布与治理 | 统一处理 draft/published/disabled/revoked/offline、TTL、缓存、审计、错误 envelope |
```

Rationale:

- 把“平台扩展”从几个功能点，升级成正式产品线。
- DeepWiki / WebChat / 多模态以后可以作为 Skill / Agent / Knowledge 的具体内容形态，而不是平台定义本身。

### Proposal 2: 保留当前企业治理 PRD，不再硬塞新需求进入其主体

Artifact: `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`

OLD:

```markdown
V1 不追求新增模型能力或复杂知识库，而是先把组织身份、部门归属、部门预算、用量聚合与告警链路打通。
```

NEW:

```markdown
企业治理 PRD 继续只覆盖组织、登录、预算、用量和风险治理。
通用 agent 能力平台（Skill 库 / Knowledge 库 / Agent 库 / 下游客户端集成）作为独立规划主线，单独建立 PRD 与架构文档，不混入本 PRD 的 FR 主体。
```

Rationale:

- 保护现有 PRD 的职责边界。
- 避免把两类产品目标混成一个文档。

### Proposal 3: 新建独立 PRD《通用 Agent 能力平台》

New Artifact: `_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md`

该 PRD 应覆盖的最小需求域：

1. **统一资源抽象**
   - `model`
   - `knowledgeBase`
   - `skill`
   - `agent`

2. **统一资源生命周期**
   - `draft`
   - `published`
   - `disabled`
   - `revoked`
   - `offline`
   - `deprecated`

3. **客户端接入与发布语义**
   - 客户端注册
   - 资源发布范围
   - 强制刷新
   - 缓存 TTL / ETag / version
   - 资源撤销传播

4. **三个库的管理面**
   - Skill 库管理
   - Knowledge 库管理
   - Agent 库管理

5. **首个验证消费者**
   - Cherry Studio 企业扩展版

6. **后续消费者约束**
   - Codex
   - cc switch skill 库

7. **非目标**
   - 不把远端资源复制成本地主数据副本供客户端离线执行
   - 不在 P0 把远端技能直接注册为本地 MCP 工具
   - 不在 P0 同时支持多企业账号并存

Rationale:

- 需要一个真正可拆 epic / story 的产品文档，而不是仅靠下游契约清单。

### Proposal 4: 新建独立 Epics 文档，而不是污染现有 `epics.md`

New Artifact: `_bmad-output/planning-artifacts/epics-agent-platform.md`

建议的 Epic 结构：

#### Epic AP-1: 平台资源域模型与控制面基础

目标：

- 定义统一资源主数据、稳定 ID、状态机、版本、审计和客户端投影。

#### Epic AP-2: 客户端注册、授权、可见性与发布治理

目标：

- 管理 Cherry Studio / Codex / cc switch 等客户端注册信息、可见资源范围、发布状态和撤销策略。

#### Epic AP-3: Skill 库管理与执行契约

目标：

- 提供 Skill 库的后台 CRUD、Schema 管理、版本、发布、禁用、调用契约和错误语义。

#### Epic AP-4: Knowledge 库管理与检索契约

目标：

- 提供 Knowledge 库的后台 CRUD、调用形态定义（retrieval / qa / context-reference）、发布和状态治理。

#### Epic AP-5: Agent 库管理与组合契约

目标：

- 提供 Agent 的定义、依赖关系、版本管理、发布状态和客户端消费契约。

#### Epic AP-6: 开放能力标准扩展层与首个消费者验证

目标：

- 基于既有 OpenAPI / 开放能力体系扩展统一标准，而不是为某一个渠道做专属适配。
- 以 Cherry Studio 为第一验证消费者，验证通用平台契约、缓存策略、错误状态矩阵和可观测性。
- 确保后续 Codex、cc switch skill 库等消费者复用同一标准接入面。

Rationale:

- 现有 `epics.md` 是企业治理 breakdown，不适合混入另一条产品主线。

### Proposal 5: 新建独立架构文档《Agent Platform Architecture》

New Artifact: `_bmad-output/planning-artifacts/architecture-agent-platform.md`

该架构文档至少要回答：

1. 统一资源模型是“单表多态”还是“基表 + typed detail 表”
2. 资源主数据与“面向客户端的发布投影”是否分离
3. 客户端接入采用 pull、push，还是 pull + revoke event 混合
4. 统一错误 envelope 如何跨 model / knowledge / skill / agent 复用
5. Skill / Knowledge / Agent 的状态、版本、TTL、ETag、刷新、撤销如何统一
6. 后台页面的信息架构和导航如何落位
7. 管理 API 命名和表前缀如何设计，同时遵守项目跨库与 JSON 约束

Rationale:

- 这条主线的架构复杂度已经超出“在现有 enterprise architecture 上补一段”的程度。

### Proposal 6: 现有 `sprint-status.yaml` 暂不直接写入

Artifact: `_bmad-output/implementation-artifacts/sprint-status.yaml`

OLD:

```yaml
development_status:
  epic-1: done
  ...
  epic-6: backlog
```

NEW:

```yaml
# No immediate change until the new PRD / epics set is approved.
# After approval, add a new planning track:
#   agent-platform-epic-1 ... agent-platform-epic-6
```

Rationale:

- 当前 proposal 仍是 draft。
- 在没有批准独立 PRD / Epics 命名方式前，不应直接污染实现状态文件。

## 5. Proposed MVP Shape

为避免再次失控，建议把“通用 Agent 能力平台”MVP 收束为以下边界：

### In Scope

- 上游 Skill / Knowledge / Agent 三个库的**主数据管理**
- 上游发布、禁用、撤销、版本与客户端可见性治理
- Cherry Studio 企业扩展版的首个消费契约验证
- 既有 OpenAPI / 开放能力体系上的标准扩展层
- 统一错误 envelope、状态矩阵、缓存与刷新语义
- 后台管理页面和后端管理逻辑

### Out of Scope

- 真正的多租户 SaaS 运营平台
- 本地执行引擎或复杂工作流编排器
- 把远端能力全部镜像为本地 MCP 工具
- 多客户端同时在线双向同步冲突解决
- 知识库内容采集 / 向量化 / RAG 引擎全流程

## 6. Implementation Handoff

### Scope Classification

**Major**

因为它要求：

- 新产品线级别 PRD
- 新架构文档
- 新 epics 文档
- 后台管理面规划
- 下游消费者契约重定义

### Handoff Recipients

- **Product Manager / Planning Owner**
  - 负责将“通用 agent 能力平台”从 roadmap 提升为正式 PRD
  - 负责确认 Cherry Studio 是 first validating consumer，而不是唯一 consumer

- **Solution Architect**
  - 负责统一资源模型、发布投影、客户端接入、状态机与错误语义
  - 负责明确与现有 enterprise 模块的边界

- **Frontend / UX Owner**
  - 负责 Skill / Knowledge / Agent / Client / Publish 等后台页面的信息架构

- **Backend Owner**
  - 负责模型、服务、控制器、OpenAPI、审计、状态机与缓存撤销链路

- **Client Integration Owner**
  - 负责基于既有 OpenAPI / 开放能力体系定义标准扩展层
  - 负责 Cherry Studio 首个验证接入，以及未来 Codex / cc switch 的抽象复用

### Success Criteria

批准后的下一阶段至少需要产出：

1. 独立 PRD：`prd-agent-platform-2026-05-31/prd.md`
2. 独立 Epics 文档：`epics-agent-platform.md`
3. 独立架构文档：`architecture-agent-platform.md`
4. 清晰的 MVP 边界：不是 API-only，而是 API + 管理后台 + 生命周期治理
5. 明确的 consumer strategy：Cherry Studio first validating consumer, Codex / cc switch later
6. 明确的 integration strategy：不是渠道专属适配，而是既有 OpenAPI / 开放能力的标准扩展

## 7. Checklist Status

| Item | Status | Notes |
| ---- | ------ | ----- |
| 1.1 Triggering story identified | Done | Trigger is the strategic shift from API-only Cherry integration to a generic downstream agent platform. |
| 1.2 Core problem defined | Done | Current plan under-specifies the upstream control plane and over-focuses on downstream API contracts. |
| 1.3 Evidence collected | Done | User clarification + existing brief / PRD / epics / architecture conflicts identified. |
| 2.1 Current epic assessed | Done | Existing enterprise governance epics remain valid as foundation. |
| 2.2 Epic-level changes determined | Done | New platform epics are required. |
| 2.3 Remaining epics reviewed | Done | New work should not be hidden inside existing enterprise epics. |
| 2.4 New epics needed | Done | AP-1 through AP-6 recommended. |
| 2.5 Epic sequencing considered | Done | New planning track should be created after PRD / architecture approval. |
| 3.1 PRD conflict check | Done | Existing enterprise PRD conflicts with this new scope and should not absorb it directly. |
| 3.2 Architecture conflict check | Done | Existing architecture lacks generic platform decisions. |
| 3.3 UX conflict check | Done | API-only planning omits required admin pages and workflows. |
| 3.4 Other artifact impact | Done | OpenAPI, sprint-status, client contract docs, fixtures all affected. |
| 4.1 Direct adjustment | Not viable | Scope is too large for a minor extension of existing planning docs. |
| 4.2 Rollback | N/A | No code rollback needed; issue is planning direction. |
| 4.3 MVP review | Viable | Recommended as part of new PRD scoping. |
| 4.4 Recommended path | Done | Major replan with a separate product line is recommended. |
| 5.1 Issue summary | Done | This proposal. |
| 5.2 Epic/artifact adjustments | Done | This proposal. |
| 5.3 Recommended path rationale | Done | This proposal. |
| 5.4 MVP impact and action plan | Done | New independent planning track required. |
| 5.5 Handoff plan | Done | PM / Architect / FE / BE / Client integration roles identified. |
| 6.1 Final review | Done | Proposal internally reviewed for consistency with current artifacts. |
| 6.2 Proposal accuracy | Done | Proposal reflects both user clarification and current repo planning state. |
| 6.3 Explicit user approval | Action-needed | Pending user approval before mutating PRD / epics / sprint-status. |
| 6.4 Update sprint-status.yaml | N/A | Deferred until approval. |
| 6.5 Confirm next steps | Action-needed | Need user confirmation on whether to convert this proposal into new PRD / epics / architecture docs now. |

## 8. Final Recommendation

本次需求不应继续以“Cherry Studio 上游接口需求清单”的形式单独推进。

建议正式决策为：

- **上游定位：** 通用 Agent 能力平台
- **首个下游：** Cherry Studio 企业扩展版（首个验证消费者）
- **后续下游：** Codex、cc switch skill 库等
- **接入方式：** 在既有 OpenAPI / open 能力上做标准扩展，不为某个渠道单独建一套私有协议
- **规划方式：** 新建独立 PRD / Epics / Architecture，而不是扩写当前企业治理 PRD
- **MVP 原则：** 不做 API-only，必须同时规划后台页面、后端管理逻辑、资源生命周期和客户端可见性治理

---

_Draft only. Awaiting approval before applying changes to PRD / epics / sprint status._
