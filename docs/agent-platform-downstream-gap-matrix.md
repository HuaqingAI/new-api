# Agent Platform 下游需求对照矩阵

状态：draft  
版本：v0.1  
更新日期：2026-06-01  
用途：用于下游接入评审、架构讨论、下一轮 Correct Course / story 拆解

## 1. 文档目的

本文档用于把“下游需求清单”和“当前 Agent Platform 实现状态”放到同一张表里对照，帮助团队快速判断：

- 哪些下游需求已经满足；
- 哪些需求只有后端基础、还未产品化；
- 哪些需求尚未冻结为正式公共契约；
- 下一轮应优先补什么。

本矩阵的判断基于：

- 你提供的《对上游 AI Infra 系统的接口需求清单》
- 当前代码实现
- 当前已完成的 Agent Platform 2.x stories

## 2. 结果说明

矩阵中的状态定义：

- `已满足`：当前实现已具备，且下游可直接依赖
- `部分满足`：有后端基础，但未产品化、未完全冻结或仍有限制
- `未满足`：当前没有稳定实现或没有足够明确的公共契约

## 3. 对照矩阵

| 编号 | 下游需求项 | 当前状态 | 当前实现依据 | 关键差距 | 建议动作 |
| --- | --- | --- | --- | --- | --- |
| P0-1 | OAuth / 企业登录态 | 部分满足 | 已有 `/api/agent-platform/oauth/authorize`、`/token`、`/revoke`，并支持 PKCE `S256` | 缺完整运营流、缺正式 callback/allowlist 文档、`authorize` 当前更像 JSON 能力接口而非完整浏览器产品流 | 冻结正式 OAuth wire contract，并补前端/运营工作流规格 |
| P0-2 | 模型发现 / 默认模型 / 模型状态 | 未满足 | 当前 Agent Platform 2.x 主要覆盖 client / auth / open-capabilities / skill / knowledge / agent，未冻结 enterprise model discovery 公共契约 | 你要求的 `modelId`、`providerStableId`、`isDefault`、`disabledReason`、状态矩阵等没有作为 Agent Platform 下游公共规格冻结 | 单独补一份 enterprise model discovery contract 或发起 CC |
| P0-3 | 远端知识库元数据与调用 | 部分满足 | `POST /api/open-capabilities/knowledge-bases/:id/query` 已存在，retrieval-only 已生效 | metadata 公共规格还不够完整；provider onboarding 未完全产品化；还没有正式公共文档冻结 | 冻结 knowledge public DTO 与 query contract 文档 |
| P0-4 | 远端技能元数据与执行 | 部分满足 | `POST /api/open-capabilities/skills/:id/invoke` 已实现，依赖 resource version 与 binding config | 没有单独的公共 skill invoke 规格文档；当前成功依赖上游控制面资源配置质量 | 冻结 skill metadata + invoke request/response spec |
| P0-5 | 资源缓存 / 状态 / 撤销 | 部分满足 | discovery/detail/refresh 已返回 `freshness`、`etag`、`resource_version`、`contract_compatible`、diagnostics | 还没有完全整理成下游签核级文档；部分下游期望字段如 `source`、`accountId`、`disabledReason` 未冻结 | 把 freshness / revoke / stale / retry 语义升格成正式对外契约 |
| 3.1 | OAuth 授权发起 | 部分满足 | `authorize` 支持 `client_id`、`redirect_uri`、`scope`、`state`、`code_challenge`、`code_challenge_method` | 缺明确的浏览器 redirect UX 说明；缺 allowlist/受控 host 规范 | 文档冻结，并补示例请求 |
| 3.2 | Token exchange / refresh / revoke | 已满足（后端） | token exchange、refresh、revoke 都已实现 | 对下游来说缺正式错误码与刷新策略说明文档 | 文档化即可 |
| 3.3 | 企业账号最小身份 DTO | 部分满足 | grant/token/client 维度已存在 `client_id`、`contract_version`；平台登录态存在 user/session 基础 | 没有明确的对下游“企业账号展示 DTO”公共规范，例如 `accountId`、`tenantId`、`enterpriseName` | 需要单独定义账户/租户展示 DTO |
| 4.1 | 企业模型发现接口 | 未满足 | 当前无独立 model discovery 公共契约冻结 | 下游要求的模型 stable id、默认模型、状态、禁用原因未正式承载 | 建议列为下一轮单独 story / CC 项 |
| 4.2 | 默认模型规则 | 未满足 | 当前无冻结规则 | 无“无默认/多默认/默认不可用”公共语义 | 单独补规格 |
| 4.3 | 模型调用错误映射 | 部分满足 | Open capability error envelope 方向已存在 | 主要针对 skill/knowledge/agent，未冻结 model domain 特有映射 | 需要补 model-specific error mapping |
| 5.1 | 资源 metadata 基础字段 | 部分满足 | 当前 resource discovery/detail 已有 `resource_id`、`resource_type`、`display_name`、`resource_version`、`freshness`、`etag` | 缺 `source`、`accountId`、`tenantId`、`disabledReason`、`fetchedAt`、`expiresAt` 等统一公共字段 | 冻结对下游最小字段集 |
| 5.2 | 刷新与撤销规则 | 部分满足 | refresh、stale、revoked、offline、contract_compatible 已有 | 缺正式下游文档说明；资源列表 TTL、强制刷新、撤销后列表策略未对外冻结 | 文档冻结 + 示例 |
| 6.1 | 知识库 metadata 接口 | 部分满足 | Knowledge list/get/query 后端基础存在 | metadata 公共字段仍未正式对下游冻结 | 提炼为正式 knowledge discovery/detail spec |
| 6.2 | 知识库 P0 调用形态三选一 | 已满足（方向） | 当前实现已收敛到 retrieval-only | 还未以面向下游的正式文档写清请求/响应 schema | 文档补齐 |
| 7.1 | 技能 metadata 接口 | 部分满足 | Skill list/get/version/invoke 后端基础存在 | 没有签核级 skill metadata 公共文档 | 文档冻结 |
| 7.2 | 技能执行接口 | 部分满足 | sync invoke 已存在 | async 模式、任务状态、取消接口尚未体现 | 明确 P0 是否只支持 sync |
| 8 | 统一错误 envelope | 部分满足 | `permissionDenied`、`resourceRevoked`、`resourceOffline`、`timeout`、`upstreamFailed`、`contractInvalid` 已存在 | 还未覆盖你需求中的所有客户端错误码表达，如 `loginExpired`、`noAssignedResource`、`networkFailed` 等公共语义 | 需要正式错误码映射矩阵 |
| 9 | 状态矩阵 | 部分满足 | `visible/callable`、`fresh/stale/offline/revoked` 已存在 | 还未把 `empty`、`loadFailed`、`loginExpired` 等客户端态完整冻结 | 需要补状态矩阵和映射规则 |
| 10 | 可观测性、性能和诊断 | 部分满足 | 已有 `request_id` / diagnostics 基础 | 还未冻结 `traceId/retryAfter/perf baseline` 为对下游公共承诺 | 建议后续单独冻结 |
| 11 | Mock / contract fixture | 未满足 | 当前 stories 和代码中未看到正式对下游发布的 mock server / fixture 套件 | 缺本地 contract fixture 交付物 | 建议列为 Phase 0 签核物 |
| 12 | 非接口范围确认 | 已满足（方向） | 当前架构已经明确不把 Agent 当 runtime，不把客户端当管理后台，不做本地主数据复制 | 缺一份面向下游的正式边界声明文档 | 直接沿用并下沉到正式对外规格 |

## 4. 能力域总结

### 4.1 已经比较成熟、可用于下游原型接入的能力

- OAuth + PKCE 基础链路
- token exchange / refresh / revoke
- client registration 后端模型
- open capability discovery / detail / refresh
- skill invoke
- knowledge query
- agent detail 读取
- freshness / revoked / offline / stale 的基础语义

### 4.2 当前最大 gap

最大的 gap 不是“后端完全没有”，而是：

1. **模型发现公共契约没有冻结**
- 这是你给的下游需求里最明显的空白

2. **很多能力只有后端基础，没有正式下游签核文档**
- client registration
- OAuth
- skill metadata / invoke
- knowledge metadata / query
- freshness / revoke

3. **运营产品化流程不足**
- 当前 UI 没有完整 Clients 工作区
- 当前 UI 没有完整下游 onboarding 工作流

4. **错误码、状态矩阵和诊断语义还没有完全固化成对下游承诺**

## 5. 建议优先级

### P1：必须先补

1. 冻结 OAuth / token / revoke 正式 wire contract
2. 冻结 client registration schema
3. 冻结 discovery/detail/refresh 公共字段集
4. 冻结 skill invoke / knowledge query 的正式 request/response schema

### P2：强烈建议尽快补

1. 冻结模型发现契约
2. 冻结错误码矩阵
3. 冻结状态矩阵
4. 给下游一套 mock / fixture

### P3：可后续推进

1. 完整 Clients 控制台工作区
2. 完整授权运营工作流
3. 更完整的 observability/perf baseline 承诺

## 6. 用于发起下一轮 CC 的建议结论

你如果要发起下一轮 CC，可以直接用下面这段结论：

> 当前 Agent Platform 已具备面向下游接入的后端基础能力，特别是在 OAuth、client registration、open capability discovery/detail/refresh、skill invoke、knowledge query 和 agent detail 方面已有真实实现。但这些能力尚未被完整冻结为下游可直接签核的公共契约，尤其缺少企业模型发现契约、完整错误码矩阵、状态矩阵和 mock fixture。下一轮 CC 不应再把问题定义为“是否需要这些能力”，而应聚焦于“如何把已有后端能力整理并冻结成正式下游接入规格，并补齐仍缺失的模型发现与产品化运营流程”。 

## 7. 审阅建议

建议你带着下面 4 个问题审这份矩阵：

1. Cherry Studio P0 最核心的阻塞到底是 OAuth / resource discovery / invoke/query，还是 model discovery？
2. 哪些项已经可以直接进入下游接入，不必等下一轮重做？
3. 哪些项必须先冻结成正式规格，否则实现会反复返工？
4. 下一轮 CC 是应该聚焦“公共契约冻结”，还是“控制台产品化”，或者两者一起推进？
