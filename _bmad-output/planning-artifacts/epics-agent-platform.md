---
stepsCompleted: [1, 2, 3, 4]
inputDocuments:
  - "_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md"
  - "_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md"
  - "_bmad-output/planning-artifacts/architecture-agent-platform.md"
  - "_bmad-output/planning-artifacts/sprint-change-proposal-2026-05-31-agent-platform.md"
---

# Agent Platform - Epic Breakdown

## Overview

本文档基于 Agent Platform 最新 PRD、addendum、architecture 及重规划提案，输出可直接进入后续实现规划的 Epic 与 Story 拆分结果。

## Requirements Inventory

### Functional Requirements

FR1: 平台必须为 `Skill`、`Knowledge`、`Agent` 提供统一的资源治理框架，确保三类资源拥有稳定身份与一致治理语义。

FR2: 平台必须为所有资源定义统一生命周期，至少覆盖 `draft`、`published`、`disabled`、`revoked`、`offline`、`deprecated`，并支持显式的 publish/disable/revoke/offline/rollback 动作。

FR3: 平台必须区分内部资源定义与面向客户端的发布投影，使同一资源能够按客户端差异化暴露，而不删除源定义。

FR4: 平台必须允许管理员注册并维护下游客户端，包括契约版本与能力声明。

FR5: 平台必须提供共享的开放能力标准层，覆盖 discovery、detail、invoke/query、refresh，而不是为每个客户端单独设计私有协议。

FR6: 平台必须定义统一的 `TTL`、`Freshness`、`Version` / `ETag`、强制刷新、撤销与收敛语义，MVP 阶段 `TTL` 上限为 5 分钟。

FR15: 平台必须定义契约版本、兼容规则与废弃策略，保证后续客户端可以复用同一核心契约而不失控漂移。

FR7: 平台必须提供 `Skill` 的后台管理能力，包括创建、编辑、查看、发布、禁用、撤销。

FR8: 平台必须定义 `Skill` 的 discovery/detail/invoke 契约及其标准化错误行为。

FR9: 平台必须提供 `Knowledge` 的后台管理能力，包括元数据、版本、状态与发布治理。

FR10: 平台必须将 `Knowledge` 的 MVP 消费形态冻结为单一标准化 `retrieval` 模式，返回结构化引用结果，而不是承诺泛化 RAG 或 runtime 能力。

FR11: 平台必须提供 `Agent` 定义、版本、状态、描述信息的管理能力。

FR12: 平台必须允许 `Agent` 声明其依赖的 `Skill` / `Knowledge`，让客户端能理解组合边界，而不要求平台提供服务端运行时编排。

FR13: 所有开放能力接口必须返回统一错误 envelope，并具备稳定的错误类别与可重试语义。

FR14: 平台必须记录并展示创建、修改、发布、撤销、回滚、重试及失败消费路径的审计与诊断信息。

### NonFunctional Requirements

NFR1: 契约稳定性。任何 breaking change 都必须伴随明确的契约版本变更与废弃策略。

NFR2: 最小授权与可解释性。平台必须区分 `visible` 与 `callable`，授权失败与兼容失败必须可诊断。

NFR3: 可审计性。治理动作必须形成完整事件链，支持查询、回滚与撤销诊断。

NFR4: 缓存收敛性。资源状态变化必须通过 `refresh` 或在 300 秒上限内完成收敛，超窗继续使用旧投影必须可诊断为不合规。

NFR5: 可诊断性。平台必须支持通过请求标识、资源版本、客户端标识和发布投影状态定位问题。

NFR6: 仓库兼容性。管理面 API 必须进入 `docs/openapi/api.json`；JSON 编解码必须使用 `common/json.go`；数据模型与迁移必须兼容 SQLite、MySQL、PostgreSQL。

### Additional Requirements

- 必须在现有 `new-api` monorepo 内实现，不能新建独立 starter 或拆出新服务。
- Agent Platform 实现需落在新的 bounded context 路径中，遵循现有 Router -> Controller -> Service -> Model 分层。
- 数据模型采用 shared registry + typed detail，而不是一个巨型多态表。
- 必须明确分离 control plane、auth plane、open capability plane。
- `Knowledge` 在 MVP 中冻结为 provider-backed 的 `retrieval` 契约，不负责 ingestion、embedding、完整 RAG 生命周期。
- `Knowledge` provider 集成必须通过统一的 `http_retrieval` adapter 语义完成，具体引擎名称不得成为公共契约字段。
- `Agent` 在 MVP 中冻结为 definition/template 资源，不引入服务端执行 runtime、流程状态或 orchestration engine。
- freshness / convergence 语义属于契约要求，不是实现建议。
- contract version 与 namespaced extension 治理属于一等架构约束。
- `client_instance` 仅作为 post-MVP 扩展点，MVP 不应要求实例级路由或持久化模型。
- 下游授权模型冻结为浏览器委托的 authorization code + PKCE，以及受控开启的 client credentials。
- 第二个验证消费者优先级冻结为 `Codex`；`cc switch` 作为后续验证对象保留在 roadmap 中。
- refresh token 必须哈希存储；access token 使用短期 JWT，并结合 token version 做撤销校验。
- 管理 UI 以 `web/default` 为 MVP 主交付面；`web/classic` 对等支持不在 MVP 范围。
- 管理 UI 需包含 overview、clients、skills、knowledge、agents、publishing、audit/diagnostics 等功能域。
- open capability 路由位于 `/api/open-capabilities/**`；control plane 路由位于 `/api/agent-platform/**`；OAuth 路由位于 `/api/agent-platform/oauth/**`。
- 未经后续架构修订批准，不得改动 `relay/**` 协议面。
- 必须继续保护 `new-api` 与 `QuantumNous` 标识。

### UX Design Requirements

未提供独立 UX 设计文档。以下可执行 UX 要求来自 architecture 与 addendum：

UX-DR1: 管理体验必须落在 `web/default`，并显式提供 Overview、Clients、Skills、Knowledge、Agents、Publishing、Audit & Diagnostics 功能域。

UX-DR2: 导航顺序必须强化控制面工作流：Overview -> Clients -> Skills -> Knowledge -> Agents -> Publishing -> Audit & Diagnostics。

UX-DR3: UI 必须清晰区分 `visible` 与 `callable`、发布状态、freshness 状态和依赖有效性，不能隐式处理。

UX-DR4: 客户端注册、发布/撤销、consent 可见性与诊断等管理动作必须能在浏览器管理控制面完成，但管理 UI 不能演变成开放能力数据面的 bearer token 消费者。
UX-DR5: 实施前必须有一份轻量 UX 文档冻结关键交互：客户端注册、发布/撤销、visible but not callable 状态解释、审计 drill-down 与失败重试反馈。

### FR Coverage Map

FR1: Epic 1 - 统一资源注册与治理基线。
FR2: Epic 1 - 生命周期、版本与回滚治理动作。
FR3: Epic 1 - 发布投影与按客户端暴露治理。
FR4: Epic 2 - 客户端注册与下游主体管理。
FR5: Epic 2 - 开放能力标准层。
FR6: Epic 2 - freshness、refresh、revoke 与收敛契约。
FR15: Epic 2 - 契约版本与扩展治理。
FR7: Epic 3 - Skill 管理。
FR8: Epic 3 - Skill 调用契约。
FR9: Epic 4 - Knowledge 管理。
FR10: Epic 4 - retrieval-only Knowledge 契约。
FR11: Epic 5 - Agent 定义管理。
FR12: Epic 5 - Agent 依赖声明与边界校验。
FR13: Epic 2 - 开放能力接口统一错误 envelope。
FR14: Epic 1 - 治理审计与诊断基线，并在 Epic 2-5 中由领域流程补强。

## Epic List

### Epic 1: 资源治理基线

平台管理员与资源发布者可以在一个统一治理框架中管理 `Skill`、`Knowledge`、`Agent` 三类资源的注册、生命周期、版本、发布投影与审计基础。

**FRs covered:** FR1, FR2, FR3, FR14

### Epic 2: 客户端接入与开放能力标准层

平台管理员可以注册下游客户端、发放受控访问能力、发布标准化能力接口，并强制 freshness、契约版本与错误语义在多个消费者之间保持一致。

**FRs covered:** FR4, FR5, FR6, FR15, FR13, FR14

### Epic 3: Skill 库管理与调用

资源发布者与下游消费者可以围绕 `Skill` 完成管理、发布、发现、详情读取与调用闭环，而无需引入面向单客户端的私有协议分支。

**FRs covered:** FR7, FR8, FR13, FR14

### Epic 4: Knowledge 库管理与检索契约

资源发布者与下游消费者可以把 `Knowledge` 作为 provider-backed 的 `retrieval` 契约进行治理与消费，并清晰识别其调用边界。

**FRs covered:** FR9, FR10, FR13, FR14

### Epic 5: Agent 定义库与依赖边界

资源发布者与下游消费者可以管理和发布 `Agent` 定义、查看其依赖关系并校验组合边界，同时避免平台被错误拉入服务端运行时范围。

**FRs covered:** FR11, FR12, FR13, FR14

## Epic 1: 资源治理基线

先建立 Agent Platform 的 bounded context 基础，让后续所有面向客户端的能力都建立在统一资源模型、统一发布模型和统一审计模型之上，而不是在各领域重复造治理底座。

### Story 1.1: 建立共享资源注册表与稳定资源身份

As a 平台管理员,
I want `Skill`、`Knowledge`、`Agent` 都通过统一注册表进入平台并拥有稳定资源身份,
So that 平台在三类资源之间拥有稳定资源身份和一致治理语义。

**Acceptance Criteria:**

**Given** Agent Platform bounded context 被引入
**When** 资源持久化层被搭建
**Then** 系统为 `Skill`、`Knowledge`、`Agent` 建立共享 registry 模型
**And** 每个资源都拥有稳定的 `resource_id` 与不可混淆的 `resource_type`。

**Given** 资源需要被后续 detail、publish、audit 流程复用
**When** 平台读取 registry 记录
**Then** 共享治理字段与资源身份字段足以独立支持这些流程
**And** 后续 typed detail 的演进不会改变已有资源身份。

### Story 1.2: 为三类资源落地 typed detail 持久化结构

As a 平台管理员,
I want `Skill`、`Knowledge`、`Agent` 的专属字段通过 typed detail 结构落地,
So that 平台不需要依赖巨型多态 payload 也能保持资源差异化表达。

**Acceptance Criteria:**

**Given** 不同资源类型存在专属字段
**When** 资源被持久化
**Then** 系统将共享治理字段与类型专属 detail 字段分开存储
**And** 不把所有资源强行塞进一个巨型多态 detail payload。

**Given** 项目仓库约束生效
**When** 新增 model、DTO 或 migration
**Then** JSON 编解码使用 `common/json.go`
**And** schema 与迁移保持 SQLite、MySQL、PostgreSQL 三库兼容。

### Story 1.3: 实现统一生命周期、版本与可回滚治理动作

As a 资源发布者,
I want 资源与资源版本按照统一生命周期流转，并通过显式治理动作改变状态,
So that publish、disable、revoke、offline、rollback 的行为是一致且可解释的。

**Acceptance Criteria:**

**Given** 某资源已存在于 registry 中
**When** 发布者发起治理动作改变状态
**Then** 平台强制使用统一生命周期词汇 `draft`、`published`、`disabled`、`revoked`、`offline`、`deprecated`
**And** 状态变化由显式动作触发，而不是由隐式数据写入推导。

**Given** 某资源历史版本已经发布过
**When** 有权限的操作者执行 rollback
**Then** 平台恢复到已知历史版本
**And** rollback 不依赖手工修改数据库。

**Given** 某次生命周期变更成功或失败
**When** 记录审计事件
**Then** 审计事件至少包含操作者、对象、动作、前后状态、请求标识与结果
**And** 该事件链后续可用于 publish/revoke/rollback 诊断。

### Story 1.4: 建立与客户端注册解耦的发布投影基线

As a 平台管理员,
I want 在不依赖真实客户端记录的前提下先定义 projection/exposure 语义,
So that Epic 1 可以独立交付发布模型基线，而不会前向依赖 Epic 2 的客户端接入实现。

**Acceptance Criteria:**

**Given** 某资源定义已存在且准备进入发布流程
**When** 平台设计发布投影骨架
**Then** 平台先定义 projection 对象、visibility/callable 语义与状态转移规则
**And** 这些规则可以在测试桩或占位客户端场景中被验证。

**Given** Epic 2 后续引入正式客户端记录
**When** projection 需要绑定目标客户端
**Then** Epic 1 的投影语义可直接复用
**And** 不需要返工资源生命周期与审计基线。

### Story 1.5: 建立发布投影与 exposure 模型

As a 平台管理员,
I want 发布投影独立于源资源定义进行管理,
So that 同一个资源可以针对不同客户端拥有不同的 visible/callable 状态。

**Acceptance Criteria:**

**Given** 某资源定义已存在但尚未暴露给客户端
**When** 下游客户端执行 discovery
**Then** 该资源不会出现在开放能力面中
**And** 控制面中的源定义数据保持完整。

**Given** 某资源已经发布给客户端 A，但未发布给客户端 B
**When** 查询 exposure 状态
**Then** 平台能够表达按客户端区分的 visibility 与 callable state
**And** exposure 不能从 registry 状态自动推断。

**Given** 某客户端上的发布投影被 revoke
**When** 该资源对其他客户端仍然有效
**Then** 平台只撤销目标客户端相关的 published projection
**And** 不删除或破坏底层资源定义。

### Story 1.6: 在 `web/default` 交付管理控制面骨架

As a 平台管理员,
I want 在 `web/default` 中拥有专门的 Agent Platform 控制面骨架,
So that 后续客户端管理、资源治理、发布流程和诊断能力都落在一个清晰的管理面中。

**Acceptance Criteria:**

**Given** Agent Platform 管理 UI 被引入
**When** 在 `web/default` 中接入导航
**Then** UI 提供 Overview、Clients、Skills、Knowledge、Agents、Publishing、Audit & Diagnostics 功能域
**And** 导航顺序遵循 architecture 中定义的信息架构。

**Given** MVP 范围已冻结
**When** 确定前端实现面
**Then** Agent Platform 管理能力落在 `web/default`
**And** `web/classic` 不作为 MVP 对等交付目标。

**Given** 某治理对象同时存在 visible 与 callable 语义
**When** UI 渲染状态摘要或标签
**Then** 界面显式区分这两个状态
**And** 不用模糊状态文案掩盖兼容失败或不可调用原因。

**Given** 团队准备进入前端实施
**When** 设计管理端关键流程
**Then** 已存在一份轻量 UX 文档覆盖客户端注册、发布/撤销、状态解释与审计 drill-down
**And** 前后端对关键操作名称与状态标签使用一致术语。

## Epic 2: 客户端接入与开放能力标准层

建立可复用的客户端契约平面，让首个消费者通过标准层完成接入，同时保证后续消费者无需重新定义平台核心模型。

### Story 2.1: 建立下游客户端注册与能力声明模型

As a 平台管理员,
I want 以统一对象模型注册并维护下游客户端的契约与能力信息,
So that 新消费者接入不再依赖一次性集成记录或特判结构。

**Acceptance Criteria:**

**Given** 某个新的下游消费者需要接入
**When** 管理员创建客户端记录
**Then** 平台保存稳定 client identity、status、allowed scopes、supported grant types、contract version 与 capability declarations
**And** Cherry Studio 可以作为首个消费者被表示，而不需要专属对象形状。

**Given** 某客户端配置不完整或处于禁用状态
**When** 它尝试进入发布或调用路径
**Then** 平台阻止该流程并返回可诊断错误
**And** 缺失能力声明会被视为无效集成状态。

**Given** 客户端需要保存扩展元数据
**When** 这些元数据被存储
**Then** 平台允许 namespaced extension 字段
**And** 这些扩展字段不能重定义核心契约语义。

### Story 2.2: 实现浏览器委托授权主路径

As a 下游集成负责人,
I want 平台先交付最小可用的浏览器委托授权路径,
So that 消费者能安全访问开放能力面，而不用发明自定义授权流程。

**Acceptance Criteria:**

**Given** 某浏览器型委托消费者被授权
**When** 它执行 MVP 下游授权流程
**Then** 平台支持 authorization code + PKCE
**And** 不要求消费者使用 MVP 未冻结的 grant type。

**Given** 某最终用户尚未登录或尚未 consent
**When** 浏览器委托流程开始
**Then** 平台能够串联既有登录态与 consent 记录
**And** 审计链能关联 user、client、scope 与 grant 结果。

### Story 2.3: 实现 token 生命周期与撤销机制

As a 下游集成负责人,
I want 平台对已发放 token 提供可审计、可撤销、可轮换的生命周期治理,
So that 消费者能安全访问开放能力面，而不用发明自定义授权流程。

**Acceptance Criteria:**

**Given** 某受信任的机器型消费者被显式允许
**When** 它申请 token 访问
**Then** 平台可以通过可选的 client credentials 路径授予访问
**And** 是否开启由每个 client 单独控制，而非全局默认。

**Given** 平台发放 refresh token
**When** refresh token 被存储或轮换
**Then** refresh token 必须以哈希形式存储并支持单独撤销
**And** access token 采用短期 JWT，并结合当前 token version 校验。

### Story 2.4: 发布开放能力标准接口与统一错误语义

As a 客户端集成人员,
I want 通过共享开放能力层访问 discovery、detail、invoke/query、refresh,
So that 我能通过一套稳定协议完成接入，而不是面向不同消费者写不同接口。

**Acceptance Criteria:**

**Given** Agent Platform 资源已经发布
**When** 下游客户端访问开放能力面
**Then** 它可以通过 `/api/open-capabilities/**` 使用 discovery、detail、invoke/query、refresh 语义
**And** 这些路由与 control plane、OAuth 路由保持边界分离。

**Given** 任意开放能力调用成功或失败
**When** 平台返回响应
**Then** 平台对 `Skill`、`Knowledge`、`Agent` 与客户端校验路径统一使用标准 envelope
**And** 错误返回包含稳定类别与可重试语义，而不是依赖字符串猜测。

**Given** 某消费者需要私有扩展字段
**When** 平台返回契约数据
**Then** 扩展字段以 namespaced 方式承载
**And** 核心 discovery/detail/invoke/refresh 语义不被改变。

### Story 2.5: 强制 freshness、revoke 与契约版本收敛规则

As a 客户端集成人员,
I want freshness、`ETag` / `Version`、refresh 和 revoke 行为具备明确契约语义,
So that 客户端可以一致地收敛发布投影，平台也能诊断陈旧状态问题。

**Acceptance Criteria:**

**Given** 客户端拿到某个 published projection
**When** 它读取契约元数据
**Then** 投影包含面向消费者可读的 `TTL`、`freshness`、`version`、`etag` 等信息
**And** freshness 状态至少支持 `fresh`、`stale`、`offline`、`revoked`。

**Given** 某投影已经超过 `TTL` 且尚未完成对账
**When** 客户端继续使用旧投影
**Then** 该投影必须被视为 `stale`
**And** 超过收敛窗口继续使用会被诊断为客户端不合规，而不是静默 best effort。

**Given** 资源发生 revoke、rollback、disable 或 offline
**When** 平台或客户端执行 refresh
**Then** 收敛必须在下一次 refresh 或 300 秒上限内完成
**And** 平台能解释问题来自缓存时序、刷新时序还是客户端实现错误。

**Given** 平台引入 breaking contract change
**When** 新契约准备发布
**Then** 平台必须要求显式 contract version 变更
**And** 兼容判断基于客户端声明的支持版本与能力集合完成。

## Epic 3: Skill 库管理与调用

把 `Skill` 做成第一个完整跑通的资源闭环，作为控制面治理与标准调用能力的最清晰样板。

### Story 3.1: 在控制面提供 Skill 管理能力

As a 资源发布者,
I want 在控制面中创建、编辑、查看、发布、禁用、撤销 `Skill`,
So that `Skill` 的治理不再依赖静态配置或手工改库。

**Acceptance Criteria:**

**Given** 发布者进入 Agent Platform 的 Skill 功能区
**When** 创建或更新某个 `Skill`
**Then** 平台保存稳定身份、版本、状态与描述信息
**And** 正常治理动作可通过控制面完成，而不是依赖手工配置。

**Given** 某个 `Skill` 发生治理动作
**When** 状态变化被持久化
**Then** publish/disable/revoke 等动作都可被审计
**And** UI 能清晰展示当前状态与历史记录。

**Given** 某个 `Skill` 尚未发布给目标客户端
**When** 该客户端执行 discovery
**Then** 该 `Skill` 不会出现在 discovery 结果中
**And** 该行为遵循 Epic 1 中共享 exposure 规则。

### Story 3.2: 定义 Skill 契约 detail、schema 与调用准备度

As a 资源发布者,
I want 每个 `Skill` 都具备明确的 schema、invoke mode 与版本化契约 detail,
So that 客户端无需猜测请求和返回结构就能准备调用。

**Acceptance Criteria:**

**Given** 某个 `Skill` detail 被创建
**When** detail 被保存
**Then** 平台保存输入 schema、输出 schema、timeout/binding metadata 与版本化契约 detail
**And** 这些字段存放在 typed `Skill` detail 结构中，而不是消费者私有文档里。

**Given** 某个 `Skill` 契约不完整或不合法
**When** 发布者尝试发布该 `Skill`
**Then** 平台阻止发布并返回 contract validation error
**And** 失败原因可从控制面反馈和审计记录中定位。

**Given** 某客户端读取已发布的 `Skill` detail
**When** detail 响应返回
**Then** 契约信息足以支持调用前准备
**And** 可选扩展字段不能替代核心 invoke 契约。

### Story 3.3: 通过开放能力层暴露 Skill 的 discovery/detail/invoke

As a 客户端集成人员,
I want 通过共享能力层发现、查看并调用 `Skill`,
So that 首个消费者和后续消费者都能走同一条集成路径。

**Acceptance Criteria:**

**Given** 客户端已授权且目标 `Skill` 已对其发布
**When** 客户端执行 discovery 与 detail
**Then** 它可以发现该 `Skill` 并读取标准化契约数据
**And** 平台按客户端规则强制执行 visibility/callability 校验。

**Given** 客户端调用某个 `Skill`
**When** 调用请求合法且已授权
**Then** 平台通过标准化 invoke 面执行业务调用
**And** 成功与失败都使用统一响应/错误语义。

**Given** 调用因参数校验、权限不足、资源撤销、资源下线、超时或上游失败而失败
**When** 平台返回失败响应
**Then** 失败会映射到稳定的标准错误类别
**And** 结果可以被审计和诊断，而不是只能依赖零散日志。

## Epic 4: Knowledge 库管理与检索契约

将 `Knowledge` 做成受治理、可发布、边界明确的资源类型，同时严格控制 MVP 不滑向“完整 RAG 平台”。

### Story 4.1: 建立带 provider 元数据的 Knowledge 管理能力

As a 资源发布者,
I want 管理 `Knowledge` 资源、版本、发布状态与 provider 绑定元数据,
So that `Knowledge` 成为平台治理对象，而不要求平台承担 ingestion 或索引职责。

**Acceptance Criteria:**

**Given** 发布者创建或编辑 `Knowledge` 资源
**When** 定义被保存
**Then** 平台保存该资源的元数据、状态、版本与发布信息
**And** 即便 provider 侧 ingestion/indexing 在平台外部，治理仍然有效。

**Given** 某个 `Knowledge` 需要 provider-backed 配置
**When** 元数据被持久化
**Then** provider type 与 provider binding config 作为平台治理的一部分被保存
**And** 外部 provider 不会变成控制面生命周期的 source of truth。

**Given** 某个 `Knowledge` 被发布或撤销
**When** 查询审计与可见性数据
**Then** 平台能追踪其发布历史、撤销历史与客户端可见范围
**And** 它复用与其他资源一致的 exposure 模型。

### Story 4.2: 将 Knowledge 的 MVP 契约冻结为标准化 retrieval

As a 客户端集成人员,
I want `Knowledge` 的契约语义在 MVP 中被明确收束,
So that 我只需要围绕一个清晰的 `retrieval` 模式完成消费，而不用猜测它是不是泛化 RAG 接口。

**Acceptance Criteria:**

**Given** 某个 `Knowledge` 资源被定义为 MVP 资源
**When** 其契约 detail 被保存
**Then** `knowledge_mode` 被明确标记为 `retrieval`
**And** direct answer generation、多轮 grounding 等模式不作为 MVP 标准能力出现。

**Given** 某个 `Knowledge` 资源绑定了具体 provider
**When** 平台保存其对外契约
**Then** 公共 detail/query 契约仍然通过统一 `http_retrieval` 语义表达
**And** 不暴露 `LightRAG` 或其它 provider 的专属字段作为平台标准字段。

**Given** 某客户端请求 `Knowledge` detail
**When** detail 响应返回
**Then** detail 中包含 retrieval query shape 与结果结构预期
**And** 结果语义覆盖片段、来源元数据与排序信息。

**Given** 某客户端不支持目标 `Knowledge` 要求的 retrieval 模式
**When** 平台在调用前执行兼容性校验
**Then** 平台可以将该资源判定为 visible but not callable
**And** 返回明确的兼容性或契约错误原因。

### Story 4.3: 通过共享能力层暴露 Knowledge 检索行为

As a 客户端集成人员,
I want 像使用其他资源一样，通过共享标准层发现并查询 `Knowledge`,
So that retrieval 能成为可复用的平台能力，而不是消费者特例。

**Acceptance Criteria:**

**Given** 客户端已授权且某个 `Knowledge` 已对其发布
**When** 客户端执行 discovery 与 detail
**Then** 该资源会通过共享能力层返回标准化元数据
**And** exposure 规则继续决定其 visibility 与 callability。

**Given** 客户端发起 retrieval query
**When** 查询合法且资源可调用
**Then** 平台返回结构化 retrieval 结果，其中包含来源元数据与排序信息
**And** 响应使用统一 success/error 语义。

**Given** 查询因为契约错误、权限问题、撤销、下线、超时或 provider 失败而失败
**When** 平台返回失败结果
**Then** 失败映射到共享错误 envelope
**And** 诊断信息不会泄露 provider secret 或敏感配置。

### Story 4.4: 接入首个 `http_retrieval` provider 适配器

As a 平台管理员,
I want 通过 provider-neutral 的 `http_retrieval` adapter 接入首个外部检索引擎,
So that 平台可以验证 Knowledge 契约闭环，而不会把公共接口绑定到单一知识引擎实现。

**Acceptance Criteria:**

**Given** 团队在 `LightRAG`、`FastGPT`、`RAGFlow` 等候选中选择首个试点 provider
**When** 适配器被实现
**Then** 平台通过统一 adapter 接口完成 query 调用、错误映射与 citation 归一化
**And** 切换到另一个 `http_retrieval` provider 不要求修改公共 API 契约。

**Given** provider 发生超时、鉴权失败或响应结构变化
**When** 平台处理失败
**Then** 平台能把失败归因到 provider 适配层
**And** 对客户端保持稳定的错误 envelope 与诊断语义。

## Epic 5: Agent 定义库与依赖边界

把 `Agent` 纳入平台作为受治理、可发布、可解释依赖边界的定义对象，同时守住“不进入服务端 runtime”的架构红线。

### Story 5.1: 在控制面提供 Agent 定义管理能力

As a 资源发布者,
I want 创建、更新、发布、撤销并查看 `Agent` 定义,
So that `Agent` 成为一等治理资源，而不是被误当成平台必须执行的运行时对象。

**Acceptance Criteria:**

**Given** 发布者管理某个 `Agent` 资源
**When** 创建或更新其定义
**Then** 平台保存稳定身份、版本、状态与描述信息
**And** 该管理模型明确把 `Agent` 视为 definition/template 对象。

**Given** 某个 `Agent` 定义发生状态变化
**When** 执行 publish、revoke、rollback 等动作
**Then** 这些动作像其他资源一样被审计
**And** UI/控制面不会暗示平台拥有超出定义边界的服务端 runtime 能力。

**Given** MVP 边界已冻结
**When** 为 `Agent` 做实现决策
**Then** 平台不会在本 Story 中引入 execution state、workflow state 或 multi-step orchestration
**And** 这些能力保持明确 out of scope。

### Story 5.2: 实现 Agent 依赖声明与边界校验

As a 资源发布者,
I want 让 `Agent` 显式声明其 `Skill` 与 `Knowledge` 依赖,
So that 平台与消费者都能在使用前理解其组合边界。

**Acceptance Criteria:**

**Given** 某个 `Agent` 定义正在编辑
**When** 发布者声明依赖项
**Then** 平台保存对 `Skill` 和 `Knowledge` 的依赖引用
**And** 这些依赖在控制面中可被查看与检查。

**Given** 某个依赖缺失、被撤销、不兼容或未发布给目标客户端
**When** 平台校验该 `Agent` 的发布或可用性
**Then** 平台阻止它被视为稳定可调用资源
**And** 校验失败会明确指出是哪条依赖边界不满足。

**Given** 某客户端请求 `Agent` detail
**When** detail 响应返回
**Then** 客户端能够理解该 `Agent` 的依赖边界与组合元数据
**And** 不需要自行猜测平台并不拥有的隐式 runtime 行为。

### Story 5.3: 通过 discovery/detail 发布 Agent 定义而不越界到 runtime

As a 客户端集成人员,
I want 通过标准能力层发现并读取 `Agent` 定义,
So that 我可以消费可复用的 `Agent` 元数据，而不要求平台替我执行该 `Agent`。

**Acceptance Criteria:**

**Given** 某个 `Agent` 定义已经对客户端发布
**When** 客户端执行 discovery 与 detail
**Then** 该 `Agent` 通过与其他资源一致的标准能力层暴露
**And** 返回数据能清晰表达其定义与依赖边界。

**Given** 客户端或平台需要判断该 `Agent` 是否 callable
**When** 依赖状态或兼容规则不满足
**Then** 平台可以将其判定为 visible but not callable
**And** 判定原因使用共享、可诊断的语义，而不是泛化失败提示。

**Given** MVP 边界需要被最终验证
**When** 本 Epic 完成
**Then** 首个 `Agent` 验证闭环覆盖 publish -> discovery -> detail -> dependency/reference-boundary validation -> audit trail
**And** 不要求平台交付任何服务端 `Agent` runtime 或 orchestration engine。
