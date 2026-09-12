---
title: "new-api 通用 Agent 能力平台 PRD"
status: draft
created: 2026-05-31
updated: 2026-05-31
source_change_proposal: "../../sprint-change-proposal-2026-05-31-agent-platform.md"
revision_notes:
  - "V1.0 (2026-05-31): 基于 Correct Course 结果新建独立产品线 PRD。定义通用 Agent 能力平台，Cherry Studio 企业扩展版作为首个验证消费者；AP-6 明确为既有 OpenAPI / 开放能力体系的标准扩展层，而非 Cherry Studio 专属适配。"
  - "V1.1 (2026-05-31): 按 BMAD PRD 标准补充主/次用户、角色责任边界、闭环式 MVP、术语语义、跨切 NFR、成功指标与决策闸门，并将 Agent MVP 边界收敛为定义与引用对象。"
  - "V1.2 (2026-05-31): 锁定 Knowledge MVP 为 retrieval 单形态，量化缓存/撤销收敛窗口，统一 MVP/SM 口径，并补充 Agent 发现/引用验证指标。"
  - "V1.3 (2026-05-31): 根据 implementation-readiness 与 LightRAG 调研结论，进一步冻结 Knowledge 为 provider-neutral 的 `http_retrieval` 契约，确定 Codex 为第二验证消费者优先级，并要求实现前提供轻量 UX 说明。"
  - "V1.4 (2026-06-02): Correct Course 重处理后，AP-6 被重新定义为 AP-1 到 AP-5 完成后的下游公共契约冻结与接入签核 follow-up epic，并作为 Cherry Studio / Codex 正式接入前的 MVP exit gate。"
---

# PRD: new-api 通用 Agent 能力平台

## 0. 文档目的

本文档面向产品、架构、研发、测试与后续 Epic 拆分，定义 `new-api` 在企业治理底座之上新增的独立产品线：**通用 Agent 能力平台**。本文档使用 Glossary 统一术语，按 Feature 分组需求，并用全局稳定 `FR-*` 标识能力边界；实现细节、模型选型和更深的技术机制放入 `addendum.md` 或后续 architecture 文档，不在本 PRD 主体重复展开。

本 PRD 以 Cherry Studio 企业扩展版作为首个验证消费者，但不把 Cherry Studio 视为唯一消费者或标准定义者。PRD 中所有术语、旅程、指标和约束都以可复用于后续 Codex、cc switch 及其它下游 agent 工具为前提。

## 1. 愿景

`new-api` 要从“AI API gateway + 企业治理底座”继续演进为“通用 Agent 能力平台”。它不是给单一下游客户端补若干接口，而是成为上游统一 control plane，让平台管理员和资源发布者在一个后台里治理 `Skill`、`Knowledge`、`Agent` 三类资源，并把这些资源按统一标准发布给多个下游客户端。

这个平台真正提供的价值不是“存了多少资源”，而是把**资源主数据、发布版本、客户端可见性、可调用状态、错误语义、审计与回滚**统一起来，让新客户端接入不再重复造协议，让资源发布不再靠手工改配置，让问题定位不再停留在“到底是资源坏了、客户端不支持、权限不对，还是版本漂了”的猜测阶段。

当前阶段最需要被解决的不是抽象完整性，而是三个可验证的业务痛点：

- 新客户端接入成本过高：每接一个消费者，都要重复解释资源模型、可见性规则和错误语义。
- 资源生命周期不可控：资源发布后谁可见、谁可调、何时下线、何时回滚缺少统一治理动作与证据。
- 故障定位成本过高：当下游调用失败时，平台方无法快速判断是资源状态、客户端能力、权限配置还是契约版本导致。

## 2. 目标用户

### 2.1 主用户与次用户

- **主用户：平台管理员 / AI 平台运营者**
  - 关心新客户端接入成本、资源治理效率、发布安全性和审计追踪。
  - 希望在不改数据库、不手工拼配置的前提下完成资源治理。

- **次用户：客户端集成人员**
  - 当前首位代表是 Cherry Studio 企业扩展版集成方。
  - 关心是否存在稳定、可版本协商、可诊断的标准接入层，而不是每接一个平台能力都拿到一份新的私有约定。

### 2.2 Jobs To Be Done

- 作为平台管理员，我要能注册一个客户端并声明它支持的能力边界，这样我可以控制它能看到什么、能调用什么。
- 作为平台管理员，我要能发布、撤销和回滚资源版本，这样我可以在不动底层数据的情况下治理资源暴露面。
- 作为资源发布者，我要能维护资源元数据、版本和状态，这样我可以让资源以受控方式对外可见。
- 作为客户端集成人员，我要能基于一个稳定标准接入平台，这样我不需要为每个消费者重复实现资源发现和错误处理逻辑。
- 作为故障处理人，我要能快速定位调用失败的责任边界，这样我可以在资源、权限、客户端能力、版本之间做出明确判断。

### 2.3 角色矩阵与责任边界

- **平台管理员**
  - 目标：治理资源、管理客户端、执行发布与回滚、查看审计与诊断。
  - 可执行动作：注册客户端、配置可见范围、发布/撤销资源、查看审计与错误链路。
  - 不负责：资源内容正确性、客户端 UI 呈现质量、第三方客户端的最终用户交互。

- **资源发布者**
  - 目标：提交和维护 `Skill` / `Knowledge` / `Agent` 的定义、版本和说明。
  - 可执行动作：创建草稿、更新元数据、提交发布、查看自身资源状态。
  - 不负责：覆盖平台级可见性策略、绕过客户端能力声明、直接修改已发布客户端视图。

- **客户端集成人员**
  - 目标：让下游客户端按统一标准接入并稳定消费资源。
  - 可执行动作：声明客户端能力、实现 discovery/detail/invoke/refresh 协议、透传标准错误。
  - 不负责：定义平台资源主数据、跳过权限和可见性规则、私自扩展核心标准字段语义。

- **企业租户管理员**
  - 目标：在平台允许范围内决定本租户对哪些资源开放和启用。
  - 可执行动作：在授权后启用或停用租户范围资源，查看租户级可见资源。
  - 不负责：定义跨租户规则、修改资源主数据。MVP 阶段如无自助入口，可由平台管理员代办 [ASSUMPTION]。

- **终端使用者**
  - 目标：在 Cherry Studio、Codex 等下游工具中发现并调用平台暴露的资源。
  - 可执行动作：消费客户端已暴露且可调用的资源。
  - 不负责：绕过客户端策略直接访问平台管理面，或解释平台内部治理状态。

### 2.4 非目标用户（MVP）

- 需要本地离线资源副本执行的终端用户
- 需要复杂多租户 SaaS 运营能力的运营角色
- 需要全流程知识库采集、向量化、RAG 编排的知识工程团队
- 需要把远端 `Skill` 全量镜像为本地 MCP 工具的本地 agent 运行时

### 2.5 关键用户旅程

- **UJ-1. 林涛将 Cherry Studio 接入平台并跑通首个 Skill 闭环。**
  - **Persona + context：** 林涛是企业内部 AI 平台管理员，目标是在一周内让 Cherry Studio 企业扩展版接入平台并验证首个可用资源。
  - **Entry state：** 管理后台可登录，Cherry Studio 已申请接入，平台中已存在一个待发布 `Skill` 草稿。
  - **Path：**
    1. 林涛在客户端管理页创建 Cherry Studio 客户端记录，并填写 `client_type`、支持的契约版本和能力声明。
    2. 他在 `Skill` 库中选择一个草稿版本，补齐输入输出 `Schema` 与版本说明。
    3. 他将该版本发布给 Cherry Studio 的可见范围，并保存 `TTL` / freshness 规则。
    4. Cherry Studio 通过标准 discovery 接口发现该 `Skill`，再通过 detail 获取契约。
    5. Cherry Studio 发起一次 invoke，平台返回成功结果并留下审计记录。
  - **Climax：** 林涛在后台和 Cherry Studio 侧都看到同一个 `Skill` 已可发现且可调用，证明首个闭环打通。
  - **Resolution：** Cherry Studio 成为首个验证消费者，后续其它客户端可以复用同一标准接入。
  - **Evidence of success：** 客户端注册记录、资源发布记录、discovery 结果、invoke 成功响应、后台审计事件五者一致。
  - **Failure branch：** 若 Cherry Studio 发现资源但调用失败，平台必须能明确判断失败归因于权限、资源状态、客户端能力或契约版本。

- **UJ-2. 乔安发布一个新的 Knowledge 版本并在异常后快速回滚。**
  - **Persona + context：** 乔安是知识资源发布者，负责把一个 `Knowledge` 资源的新版本开放给指定客户端。
  - **Entry state：** `Knowledge` 资源已有一个稳定发布版本，乔安正在准备新版本。
  - **Path：**
    1. 乔安更新 `Knowledge` 元数据和调用形态。
    2. 她提交发布后，平台生成新的对外发布版本。
    3. Cherry Studio 在调用时反馈新版本结果异常。
    4. 乔安与平台管理员定位到是资源版本问题，而非客户端能力问题。
    5. 平台管理员执行回滚，客户端重新发现旧版本。
  - **Climax：** 回滚后客户端恢复到上一个可用版本，无需手工改数据库或删除主数据。
  - **Resolution：** 问题版本保留审计记录，旧版本恢复可用。
  - **Evidence of success：** 前后两个发布版本均可追踪，客户端可见版本变化、后台存在回滚审计链。
  - **Failure branch：** 若回滚后客户端仍调用异常，平台必须能继续区分是缓存未收敛、撤销规则不清还是客户端未刷新。

- **UJ-3. 魏然定位“可见但不可调用”的客户端故障。**
  - **Persona + context：** 魏然是客户端集成人员，负责处理 Cherry Studio 中“资源看得见但调不通”的故障单。
  - **Entry state：** Cherry Studio 已注册并能 discovery 到某个 `Knowledge`，但 invoke 失败。
  - **Path：**
    1. 魏然根据请求 ID 查到失败调用。
    2. 他看到标准错误 envelope，确认是 `contractInvalid` 还是 `permissionDenied`。
    3. 他回到平台后台核对客户端声明能力、资源状态、版本和可调用状态。
    4. 平台给出一致的错误分类和审计证据。
  - **Climax：** 魏然能在 30 分钟内把故障明确归因到一个责任边界，而不是在平台和客户端之间反复猜测。
  - **Resolution：** 问题被转交给正确的处理方：资源发布者、平台管理员或客户端开发者。
  - **Evidence of success：** 错误码、资源状态、客户端能力声明、审计链路一致且可查询。

- **UJ-4. 何青发布一个 Agent 定义供客户端发现与引用。**
  - **Persona + context：** 何青是 `Agent` 设计者，需要让客户端发现一个具备明确依赖边界的 `Agent`。
  - **Entry state：** 平台中已存在该 `Agent` 依赖的 `Skill` 和 `Knowledge` 资源。
  - **Path：**
    1. 何青在 `Agent` 库创建 `Agent` 定义并声明依赖关系。
    2. 她发布该 `Agent` 的一个版本给指定客户端。
    3. 客户端 discovery 到该 `Agent`，并读取其依赖和能力边界。
  - **Climax：** 客户端能够理解“这是什么 Agent、依赖什么资源、可被如何引用”，而不要求平台同时提供复杂 workflow runtime。
  - **Resolution：** `Agent` 在 MVP 中以“定义与引用对象”进入标准层，不承诺复杂执行引擎；MVP 验证目标是客户端能发现、读取并引用该定义。
  - **Evidence of success：** `Agent` 详情返回依赖关系、版本和可见状态，客户端能够按契约消费该定义。

## 3. Glossary

- **Resource** — 平台统一治理的能力对象。MVP 内包含 `Skill`、`Knowledge`、`Agent` 三类。
- **Definition** — 资源主数据定义，供平台内部维护和版本管理使用。
- **Version** — 某个资源定义的可识别版本。版本是发布、回滚与审计的基本单位。
- **Published Projection** — 面向某类客户端暴露的资源发布视图。客户端只消费投影，不直接消费平台内部定义。
- **Client** — 消费开放能力标准层的下游工具，如 Cherry Studio、Codex、cc switch。
- **Client Registration** — 客户端进入平台并声明自身契约版本、能力边界和标识信息的动作。
- **Publish** — 把某个资源版本生成可供客户端消费的 `Published Projection`。
- **Visible** — 客户端可以发现资源元数据。
- **Callable** — 客户端在当前权限、状态和契约条件下可以实际调用资源。
- **Enable** — 在平台允许范围内，对某个租户或客户端打开资源使用。
- **Revoke** — 撤销资源或资源投影的可持续使用资格；撤销后不应再被视为可调用资源。
- **Offline** — 资源因上游不可用、依赖异常或治理动作而暂时不可调用。
- **Freshness** — 资源缓存新鲜度状态，如 `fresh`、`stale`、`offline`、`revoked`。
- **Knowledge Mode** — `Knowledge` 对客户端暴露的标准调用形态。V1 MVP 固定为 `retrieval`。
- **HTTP Retrieval Provider** — 通过统一 HTTP 适配边界接入的平台外部检索引擎。它可以是 LightRAG、FastGPT、RAGFlow 或其它实现，但不改变平台公共契约。
- **Open Capability Layer** — 建立在既有 OpenAPI / open 能力体系上的标准扩展层，用于 discovery、detail、invoke、refresh 和错误语义。
- **Contract Version** — 标准能力层的版本标识，用于客户端与平台之间做兼容性协商。

## 4. Features

### 4.1 统一资源控制面

**Description：** 平台必须把 `Skill`、`Knowledge`、`Agent` 放入同一个治理框架，使平台管理员和资源发布者能按统一规则维护资源定义、版本、发布投影和状态语义。这里的统一不是把三类资源压成一个弱语义列表，而是确保它们都具备稳定 ID、可审计状态、可回滚版本和面向客户端的发布视图。实现 UJ-1、UJ-2、UJ-4。

#### FR-1: 统一管理 Skill / Knowledge / Agent 三类资源

系统必须提供统一的资源治理框架，对 `Skill`、`Knowledge`、`Agent` 三类资源进行主数据管理。

**Consequences (testable)：**
- 每类资源都有稳定 `resourceId`，且资源类型不可混淆。
- 所有资源都能关联版本、状态、发布记录和客户端可见性。
- 平台管理员可在一个后台导航结构中查看三类资源，但不要求三类资源共用完全相同字段。

#### FR-2: 定义统一且可解释的生命周期语义

系统必须为所有资源定义统一生命周期语义，至少覆盖 `draft`、`published`、`disabled`、`revoked`、`offline`、`deprecated`，并明确 `publish`、`disable`、`revoke`、`offline`、`rollback` 的业务规则。

**Consequences (testable)：**
- 任一状态变更都必须由明确动作触发，并留下审计记录。
- `Visible` 不自动等于 `Callable`；当资源可见但不可调用时，平台能给出明确原因。
- 回滚必须恢复到已知历史版本，而不是依赖手工编辑数据库。

#### FR-3: 区分资源主数据与客户端发布投影

系统必须区分平台内部维护的资源 `Definition` 与对客户端公开的 `Published Projection`。

**Consequences (testable)：**
- 未发布资源不会被客户端发现。
- 同一资源可针对不同客户端拥有不同可见范围或发布状态。
- 撤销客户端投影不会强制删除资源主数据。

### 4.2 客户端注册与开放能力标准层

**Description：** 平台必须把“接一个新客户端”收敛成标准动作：注册客户端、声明能力边界、按统一标准发现资源、读取详情、执行调用并理解错误。该层的目标是减少重复协议成本，而不是堆出一个只对 Cherry Studio 可用的特例面。实现 UJ-1、UJ-3。

#### FR-4: 管理下游客户端注册信息

平台管理员必须能够注册和维护下游客户端信息，包括客户端标识、状态、接入说明、支持的 `Contract Version` 和能力声明。

**Consequences (testable)：**
- Cherry Studio 可以作为首个验证消费者被平台注册。
- 新客户端使用相同对象模型注册，不需要为每个客户端单独创建一套平台内对象体系。
- 客户端未注册或能力声明缺失时，不得进入正式可调用流程。

#### FR-5: 在既有 OpenAPI / open 能力体系上扩展统一标准层

系统必须通过统一标准层对外暴露资源 discovery、detail、invoke 和 refresh 语义，而不是为每个客户端分别设计私有协议。

**Consequences (testable)：**
- AP-6 输出的是标准能力面，而非 Cherry Studio 专用接口。
- 新客户端只要实现同一核心契约即可接入。
- 客户端私有扩展不得改变核心标准的基础语义。

#### FR-6: 定义统一刷新、缓存与撤销语义

系统必须为下游客户端提供统一的 `TTL`、`Freshness`、`Version` / `ETag`、强制刷新和撤销处理规则。MVP 默认单个 `Published Projection` 的 `TTL` 上限为 5 分钟；平台管理员必须能够对指定客户端或资源触发强制刷新。

**Consequences (testable)：**
- 客户端能区分 `fresh`、`stale`、`offline`、`revoked` 四类状态；超过 `TTL` 且未拿到匹配 `ETag` / `Version` 的投影必须进入 `stale`。
- 资源被 `revoke`、`rollback`、`disable` 或转为 `offline` 后，客户端必须在下一次 `refresh` 或 5 分钟 `TTL` 上限内完成收敛；超过该窗口仍继续使用旧投影视为客户端实现不合规。
- 平台必须能解释“客户端还在用旧版本”是由缓存策略、刷新时序还是错误实现导致，并在审计/诊断中留下对应原因。

#### FR-15: 定义契约版本、兼容性与废弃策略

平台必须定义 `Contract Version`、breaking change 规则、向后兼容要求和废弃策略，以保证后续支持 Codex、cc switch 等客户端时不会让契约漂移失控。

**Consequences (testable)：**
- 任何 breaking change 都必须伴随契约版本变更。
- 客户端可以声明自己支持的契约版本与能力集合，平台能据此做兼容性判断。
- 对未知可选字段、额外 namespaced 扩展字段和已废弃字段，客户端与平台都存在预期行为。

### 4.3 Skill 库管理与执行契约

**Description：** `Skill` 是 MVP 中最先完成端到端闭环验证的资源类型。平台必须允许发布者维护 `Skill` 的元数据、输入输出 `Schema`、版本与状态，并让客户端按统一契约发现和调用该 `Skill`。实现 UJ-1、UJ-2。

#### FR-7: 提供 Skill 库后台管理能力

平台管理员或资源发布者必须能够创建、编辑、查看、发布、禁用和撤销 `Skill`。

**Consequences (testable)：**
- `Skill` 必须具有稳定 ID、版本、状态与输入输出 `Schema`。
- 发布前后的状态变化可追踪，并能关联到操作者和时间。
- 管理员不需要手工改配置或数据库来完成正常治理动作。

#### FR-8: 提供 Skill 调用契约

平台必须定义 `Skill` 的 discovery、detail、invoke 契约及其错误语义。

**Consequences (testable)：**
- `Skill` 调用至少支持参数校验失败、权限不足、资源撤销、资源下线、超时和上游失败的明确映射。
- 客户端在调用前无需猜测 `Schema`。
- 同一 `Skill` 对不同客户端的可见性和可调用性必须符合平台规则。

### 4.4 Knowledge 库管理与调用形态

**Description：** `Knowledge` 在平台中是被治理和发布的资源，而不是整个 RAG 管道本身。MVP 关注的是资源元数据、调用形态、状态、版本和面向客户端的可消费契约。实现 UJ-2、UJ-3。

#### FR-9: 提供 Knowledge 库后台管理能力

平台管理员或资源发布者必须能够维护 `Knowledge` 资源的元数据、状态、版本和发布关系。

**Consequences (testable)：**
- 平台能区分不同 `Knowledge` 资源及其版本。
- `Knowledge` 的发布记录、撤销记录和可见客户端范围可追踪。
- `Knowledge` 资源治理不依赖其底层 ingestion、embedding 或索引构建流程是否纳入 MVP。

#### FR-10: 明确定义 Knowledge 调用形态

平台必须为 `Knowledge` 资源定义可识别的调用形态。V1 MVP 固定只支持单一标准化 `retrieval` 形态：客户端提交查询条件，平台返回结构化引用结果（片段、来源元数据、排序信息）；MVP 不承诺直接答案生成、多轮 chat grounding 或其它 `Knowledge` 模式。

**Consequences (testable)：**
- 管理后台和开放能力层都将 `Knowledge Mode` 明确标识为 `retrieval`，且 detail 结果能暴露该模式所需的查询输入和返回的引用结果结构。
- 客户端可以根据 `retrieval` 元数据与有序引用结果做正确消费，而不是靠专有解释。
- 若某客户端不支持 `retrieval` 模式，平台必须在可见性或调用前检查阶段将其判定为“可见但不可调”，并给出明确原因或契约错误。
- provider 选择通过统一的 `http_retrieval` 适配边界完成，平台公共契约中不得出现 `LightRAG` 等具体引擎专属字段或专属模式。

### 4.5 Agent 库管理与组合边界

**Description：** `Agent` 在 MVP 中进入平台时，优先被视为“可治理、可发布、可引用的定义对象”，而不是复杂 server-side workflow runtime。平台要让客户端理解 `Agent` 是什么、依赖什么、边界在哪里，而不是在 MVP 阶段承诺完整编排执行引擎。实现 UJ-4。

#### FR-11: 提供 Agent 库后台管理能力

平台必须允许资源发布者维护 `Agent` 定义、版本、状态和描述信息。

**Consequences (testable)：**
- `Agent` 具备稳定 ID、版本、状态和说明信息。
- `Agent` 发布、撤销和回滚行为可被审计。
- MVP 中 `Agent` 不要求等同于复杂运行时实例。

#### FR-12: 允许 Agent 声明依赖关系

`Agent` 必须能够声明其依赖的 `Skill` / `Knowledge` 能力，以形成可解释的资源组合边界。

**Consequences (testable)：**
- 后台可查看 `Agent` 依赖关系。
- 客户端获取 `Agent` 详情时，能理解其能力边界和依赖对象。
- 当依赖关系不满足平台规则时，`Agent` 不得被错误地视为稳定可调用资源。

**Out of Scope：**
- MVP 不要求平台提供复杂 `Agent` workflow runtime 或多步 server-side orchestration。

### 4.6 审计、错误与诊断

**Description：** 该平台若不能让平台方知道“谁改了什么、客户端为什么失败、资源为什么不可调”，就只是一个资源清单，而不是治理平台。审计、错误和诊断不是附属功能，而是 MVP 闭环的一部分。实现 UJ-1、UJ-2、UJ-3。

#### FR-13: 统一错误 envelope 与错误语义

平台所有开放能力接口必须返回统一错误 envelope，用于跨 `Skill` / `Knowledge` / `Agent` / `Client` 复用，并明确主要错误类别的语义和可重试性。

**Consequences (testable)：**
- 至少可映射：`permissionDenied`、`resourceRevoked`、`resourceOffline`、`quotaOrRateLimited`、`timeout`、`upstreamFailed`、`contractInvalid`。
- 平台对主要错误类别给出固定的状态码、错误码和是否可重试语义。
- 客户端可以基于标准 envelope 做稳定处理，而不是依赖字符串猜测。

#### FR-14: 提供后台审计与诊断可见性

平台必须记录并展示创建、修改、发布、撤销、回滚、重试等关键治理动作，并能支持调用失败的诊断。

**Consequences (testable)：**
- 后台能查询发布、撤销、失败和重试状态。
- 审计事件至少能关联操作者、对象、动作、前后状态、请求标识和结果。
- 诊断信息不能泄露敏感凭据。

## 5. Cross-Cutting NFRs And Governance

- **NFR-1 契约稳定性**
  - 核心标准层必须具备明确版本语义和废弃策略。
  - 不允许在不升版本的情况下引入 breaking change。

- **NFR-2 权限与最小授权**
  - 平台必须区分“可见”与“可调用”。
  - 客户端、租户和资源级规则必须可解释，且拒绝原因可诊断。

- **NFR-3 审计与可追责**
  - 关键治理动作必须留痕，且可被查询与导出。
  - 回滚、撤销、重试等动作必须形成完整事件链。

- **NFR-4 缓存与收敛**
  - 资源状态变化后的客户端收敛行为必须有统一规则。
  - 平台必须能判断问题来自状态本身还是客户端未按规则刷新。

- **NFR-5 诊断性**
  - 平台必须支持通过请求标识、资源版本、客户端标识定位故障。
  - “资源可见但不可调用”必须是可解释场景，而非模糊异常。

- **NFR-6 平台兼容性约束**
  - 管理 API 进入 `docs/openapi/api.json`。
  - 所有 JSON marshal/unmarshal 继续使用 `common/json.go`。
  - 数据模型和迁移必须兼容 SQLite、MySQL、PostgreSQL。

## 6. 非目标（Explicit）

- 不在 MVP 中交付复杂多租户 SaaS 运营平台。
- 不在 MVP 中交付知识库 ingestion、embedding、RAG 全链路。
- 不在 MVP 中把远端资源镜像为本地 MCP 主数据。
- 不在 MVP 中交付复杂 `Agent` workflow runtime。
- 不在 MVP 中支持多个企业账号同时并存消费。
- 不在 MVP 中为单一消费者维护一套独立私有协议。

## 7. MVP Scope

### 7.1 MVP 闭环定义

MVP 必须至少跑通以下闭环，而不是只完成一组后台页面或接口清单：

1. 平台管理员可以注册一个下游客户端，并声明其契约版本与能力边界。
2. 资源发布者可以至少发布 1 个 `Skill`、1 个 `Knowledge`（固定 `retrieval` 形态）资源，并发布 1 个引用这些能力的 `Agent` 定义给该客户端。
3. 该客户端可以按统一标准发现、读取详情并调用上述 `Skill` 与 `Knowledge`，同时发现并读取该 `Agent` 的依赖与引用边界。
4. 平台可以记录调用结果、失败原因、资源版本和责任边界。
5. 平台管理员可以下线、撤销或回滚资源，并让客户端在统一收敛窗口内感知变化。

### 7.2 In Scope

- `Skill`、`Knowledge`、`Agent` 三类资源的主数据管理
- 客户端注册、能力声明和可见性治理
- 统一发布、撤销、回滚、状态、版本、缓存和错误语义
- Cherry Studio 企业扩展版首个验证接入
- 开放能力标准层
- 后台管理页面与后端管理逻辑
- `Agent` 以定义与引用对象形式进入 MVP，包括依赖声明、发布和可发现边界

### 7.3 Out of Scope For MVP

- 多租户运营：当前优先解决统一资源治理与首个客户端闭环，而非完整运营体系。
- 本地执行副本：当前不为本地 agent runtime 复制整套远端资源主数据。
- 知识库底层采集与索引构建：当前只治理 `Knowledge` 的元数据、版本和调用形态。
- `Knowledge` 的 `qa`、`chat-context reference` 等扩展形态：当前不作为 MVP 标准调用模式交付。
- 绑定某个唯一外部知识引擎作为平台标准件：当前只冻结 `http_retrieval` 契约，不冻结唯一 provider。
- 复杂工作流运行时：当前不承诺 server-side `Agent` orchestration engine。
- 客户端间双向同步冲突解决：当前优先保证单向治理与可诊断性。

## 8. Success Metrics

**Primary**

- **SM-1：** Cherry Studio 从客户端注册完成到首个 `Skill` 端到端可调用闭环完成，内部试点周期不超过 10 个工作日。验证 FR-4、FR-5、FR-7、FR-8。 [ASSUMPTION]
- **SM-2：** 在首个试点中，至少 1 个 `Skill` 和 1 个 `Knowledge`（固定 `retrieval` 形态）资源完成“发布 -> discovery -> detail -> invoke -> 审计留痕”闭环。验证 FR-3、FR-5、FR-8、FR-10、FR-14。
- **SM-3：** 90% 的失败调用都能映射到标准错误 envelope 中的已知错误类别，并在 30 分钟内定位责任边界。验证 FR-13、FR-14。

**Secondary**

- **SM-4：** 资源发布、撤销或回滚后，后台在 1 分钟内可见对应治理动作和结果，客户端在下一次 `refresh` 或 5 分钟 `TTL` 上限内收敛到正确状态。验证 FR-2、FR-6、FR-14。
- **SM-5：** 第二个消费者 `Codex` 接入时，不需要重定义资源主模型，也不需要复制一套新的私有协议主干。验证 FR-1、FR-5、FR-15。
- **SM-6：** 在首个试点中，至少 1 个 `Agent` 定义完成“发布 -> discovery -> detail -> 依赖/引用边界校验 -> 审计留痕”验证，客户端能够读取其依赖的 `Skill` / `Knowledge` 边界而不要求平台提供复杂 runtime。验证 FR-11、FR-12、FR-14。
- **SM-7：** 首个 `Knowledge` provider 从 LightRAG、FastGPT、RAGFlow 等候选中任选其一接入时，平台对外 detail/query 契约与错误 envelope 无需变更。验证 FR-5、FR-10、FR-13、FR-15。

**Counter-metrics (do not optimize)**

- **SM-C1：** 试点期间“客户端可见但平台无法解释为何不可调用”的案例数必须为 0。对冲 SM-1、SM-3。
- **SM-C2：** 未升 `Contract Version` 就引入 breaking change，或因切换 `Knowledge` provider 而改变公共契约的次数必须为 0。对冲 SM-5、SM-7。
- **SM-C3：** 为首个消费者引入只服务单一客户端且不可复用的核心协议分支次数必须为 0。对冲 SM-1。
- **SM-C4：** 为了让 `Agent` 看起来“可执行”而提前引入复杂 server-side orchestration runtime 的次数必须为 0。对冲 SM-6。

## 9. Constraints And Guardrails

- 管理 API 进入 `docs/openapi/api.json`。
- 不直接修改 `relay/**` 协议面，除非架构升级审批。
- 所有 JSON marshal/unmarshal 继续使用 `common/json.go`。
- 数据模型和迁移必须兼容 SQLite、MySQL、PostgreSQL。
- 继续保护 `new-api` 与 `QuantumNous` 标识。
- PRD 主体定义能力边界，不直接规定“统一基表还是 typed detail 表”等实现选型。

## 10. 决策闸门（进入 Architecture Freeze 前必须拍板）

- **DG-1：核心标准层是否允许有限客户端扩展，边界如何约束？**
  - 必须在 architecture 冻结前明确：核心契约是否保持严格通用，并允许通过 namespaced 扩展字段承载客户端私有信息。

- **DG-2：第二个验证消费者优先级已定为 Codex。**
  - `cc switch` 继续保留在后续验证序列中，但不会阻塞当前 PRD、architecture 与 epics 的实施规划。

- **DG-3：审计与诊断达到什么粒度，才算可上线的治理能力？**
  - 必须明确最低发布门槛，例如请求标识、前后状态、回滚链路、错误类别和责任边界是否缺一不可。

## 11. 假设索引

- [ASSUMPTION] MVP 阶段如无租户自助治理入口，平台管理员可代办租户范围启用动作。
- [ASSUMPTION] `SM-1` 的周期目标以内部试点环境为测量基线，而非外部正式商用 SLA。
