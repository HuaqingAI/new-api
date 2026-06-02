# Agent Platform 下游接入技术规格说明

状态：draft / AP-6 contract-freeze candidate  
版本：v0.2  
更新日期：2026-06-02  
对应规划：AP-6 下游公共契约冻结与接入签核  
适用对象：Cherry Studio 企业 Provider、Codex 侧消费者、以及其它经过授权的 Open Capability 下游接入方

## 1. 文档目的

本文档用于给下游接入方提供一份**基于当前实现状态**的技术规格说明，帮助下游明确：

- 当前 `new-api` Agent Platform 已经提供了哪些后端契约；
- 哪些能力已经可以接入使用；
- 哪些能力虽然有后端基础，但尚未完全产品化或未冻结为最终公共契约；
- 下游在 Phase 0/Phase 1 接入时可以依赖什么，不能假设什么。

本文在 v0.1 “当前实现状态说明”的基础上，升级为 AP-6 的公共契约冻结候选稿。

它仍是 draft，但用途从“解释当前后端做到了什么”升级为“定义下游签核需要冻结什么”。AP-6 story 进入 done 前，本文需要与 `docs/openapi/api.json`、mock / fixture / conformance test artifacts 保持一致。

## 2. 总体结论

### 2.1 当前总体能力判断

当前 Agent Platform 对下游的可用性可以概括为：

- **客户端注册能力**：后端已具备，前端管理工作区未完全产品化
- **浏览器委托 OAuth 登录能力**：后端已具备
- **Token exchange / refresh / revoke**：后端已具备
- **开放能力 discovery / detail / refresh**：已具备
- **Skill invoke**：已具备，但依赖上游资源版本与 binding 配置完整
- **Knowledge query**：已具备，但依赖 provider-backed retrieval 配置完整
- **Agent detail 读取**：已具备，按 definition/template 语义暴露
- **面向运营者的完整接入控制台流程**：尚未完全产品化

### 2.2 一句话判断

当前系统已经超过“只有概念 API”的阶段，  
但还没有完全进入“对下游 fully frozen、operator-friendly、可直接运营接入”的最终形态。

更准确地说：

- **后端接入底座已经有了**
- **下游正式公共契约文档之前没有独立收敛**
- **本文件就是当前版本的第一份签核级下游口径草案**

## 3. 规格来源

本规格基于以下事实源整理：

- 当前后端代码：
  - `router/`
  - `controller/agentplatform/`
  - `dto/agentplatform/`
  - `service/agentplatform/`
  - `middleware/agent_platform_bearer.go`
- 当前 Agent Platform 已完成 stories：
  - `_bmad-output/implementation-artifacts/ap-2-1-register-clients-and-capability-declarations.md`
  - `_bmad-output/implementation-artifacts/ap-2-2-implement-browser-delegated-auth-main-flow.md`
  - `_bmad-output/implementation-artifacts/ap-2-3-implement-token-lifecycle-and-revocation.md`
  - `_bmad-output/implementation-artifacts/ap-2-4-publish-open-capability-standard-apis-and-error-envelope.md`
  - `_bmad-output/implementation-artifacts/ap-2-5-enforce-freshness-revoke-and-contract-version-convergence.md`

## 3A. AP-6 签核原则

### 3A.1 Source of Truth

AP-6 公共契约以三件套为准：

1. `docs/agent-platform-downstream-contract-spec.md`
2. `docs/openapi/api.json`
3. mock / fixture / conformance test artifacts

三者不一致时，相关 AP-6 story 不得标记为 done。

### 3A.2 Field Requirement Levels

本文后续字段必须按以下等级标注：

- `MUST`：下游和平台都必须支持，缺失视为契约不合格。
- `SHOULD`：强烈建议支持；如不支持必须说明原因和替代路径。
- `MAY`：可选扩展字段，不影响核心契约。
- `EXTENSION`：只能出现在 `extensions.<namespace>` 下，不得改写核心字段语义。

### 3A.3 Contract Versioning

- breaking change 必须提升 `contract_version`。
- additive optional fields 可保持当前 contract version。
- 客户端未知字段必须安全忽略。
- Cherry Studio / Codex 私有字段必须进入 namespaced extensions。

## 4. 能力核对结果

### 4.1 OAuth / 企业登录态

状态：**部分满足下游需求，后端契约可用**

当前已实现：

- `GET /api/agent-platform/oauth/authorize`
- `POST /api/agent-platform/oauth/token`
- `POST /api/agent-platform/oauth/revoke`
- 强制 PKCE `S256`
- 浏览器委托授权路径基于现有平台登录态

当前已覆盖：

- authorization code 颁发
- token exchange
- refresh token exchange
- revoke 路径
- client 校验
- scope 校验
- redirect URI 校验

当前未完全产品化的部分：

- 没有完整的前端授权管理工作区
- 没有正式对外发布的 host allowlist / callback policy 文档
- `authorize` 当前返回的是 JSON 结果，而不是完整浏览器跳转协议型 UX 规范
- consent 事实源已有，但没有独立、完整的用户可见 consent 页面契约

判断：

- 如果下游需要的是“先把浏览器委托登录接起来”，当前后端已可支撑
- 如果下游需要的是“完整企业 OAuth 产品流”，当前仍属于后端基础已具备、产品化流程未完成

### 4.2 客户端注册

状态：**后端已实现，UI 未完全产品化**

当前已实现接口：

- `GET /api/agent-platform/clients`
- `POST /api/agent-platform/clients`
- `GET /api/agent-platform/clients/:id`
- `PUT /api/agent-platform/clients/:id`

当前支持字段：

| 字段 | 状态 | 说明 |
| --- | --- | --- |
| `slug` | 已支持 | 管理面可读标识 |
| `display_name` | 已支持 | 展示名称 |
| `client_type` | 已支持 | 客户端类型 |
| `status` | 已支持 | 当前状态 |
| `allowed_grant_types` | 已支持 | grant types JSON |
| `redirect_uris` | 已支持 | redirect URI 列表 |
| `allowed_scopes` | 已支持 | scope 列表 |
| `contract_version` | 已支持 | 一等字段 |
| `capabilities` | 已支持 | capability declarations |
| `extensions` | 已支持 | namespaced 扩展 |
| `allow_client_credentials` | 已支持 | 配置位存在 |

当前 identity 模型：

- 使用稳定 opaque `client_id`
- 支持 namespaced extensions
- `contract_version` 是正式治理字段
- `capabilities` 是正式治理字段

当前缺口：

- 当前 Agent Platform 页没有真正的 `Clients` 工作区
- 运维人员还不能只靠 web/default 完成完整 onboarding
- 还没有正式的下游示例 payload 文档

判断：

- 下游可以依赖当前 client registration 后端模型
- 但运营侧仍需要通过 API 或临时工具录入 client

### 4.3 Open Capability Discovery / Detail / Refresh

状态：**已实现**

当前接口：

- `GET /api/open-capabilities/discovery`
- `GET /api/open-capabilities/resources/:id`
- `POST /api/open-capabilities/refresh`

当前 bearer scope：

- discovery / detail / refresh 需要 `ap.resources.read`

当前已返回的关键字段：

| 字段 | 状态 |
| --- | --- |
| `resource_id` | 已支持 |
| `resource_type` | 已支持 |
| `display_name` | 已支持 |
| `resource_version` | 已支持 |
| `contract_version` | 已支持 |
| `visibility_state` | 已支持 |
| `callable_state` | 已支持 |
| `freshness_ttl_seconds` | 已支持 |
| `freshness` | 已支持 |
| `etag` | 已支持 |
| `extensions` | 已支持 |
| `supported_extensions` | detail 中支持 |
| `contract_compatible` | detail/refresh 中支持 |
| `diagnostics` | detail/refresh 中支持 |

下游可据此实现：

- 资源发现
- 资源详情加载
- freshness 对账
- visible / callable 分离

判断：

- 这是当前实现里最成熟的一块

### 4.4 Skill Invoke

状态：**已实现，但依赖控制面资源版本与 binding 配置**

当前接口：

- `POST /api/open-capabilities/skills/:id/invoke`

当前 bearer scope：

- `ap.skills.invoke`

当前运行前提：

- 资源必须已发布
- resource version 必须存在
- resource version 关联的 Skill detail / binding config 必须完整

当前 binding 模型：

- 从 Skill detail 中读取 `method`、`url`、`headers`
- 服务端代表下游发起实际上游调用

判断：

- 这不是占位路由了
- 但它是否可用取决于上游对 Skill 资源定义是否填写完整

### 4.5 Knowledge Query

状态：**已实现，但依赖 provider binding**

当前接口：

- `POST /api/open-capabilities/knowledge-bases/:id/query`

当前 bearer scope：

- `ap.knowledge.query`

当前约束：

- `knowledge_mode` 必须是 `retrieval`
- 必须存在 provider-backed binding config
- provider adapter 会校验 binding 并执行 query

当前下游可依赖：

- query 路由存在
- retrieval-only 约束已生效
- 返回 `items` 与 `citations`

当前不能完全假设：

- Knowledge provider onboarding 已完全产品化
- 所有 provider 生态都已经成熟

判断：

- 对下游来说，这已经不是纯概念接口
- 但仍然强依赖上游 Knowledge 资源配置质量

### 4.6 Agent Detail / Definition Read

状态：**已实现，按 definition/template 暴露**

当前接口：

- `GET /api/open-capabilities/agents/:id`

当前 bearer scope：

- `ap.agents.read`

当前行为：

- Agent 作为定义对象暴露
- detail 返回依赖和兼容性边界
- 不承诺服务端 Agent runtime

判断：

- 这与当前架构方向一致
- 下游应把 Agent 视为 definition/template，而不是远端 runtime

## 5. 当前服务端路由面

### 5.1 Control Plane

Base path：`/api/agent-platform`

当前已存在：

- OAuth
  - `GET /oauth/authorize`
  - `POST /oauth/token`
  - `POST /oauth/revoke`
- Resources
  - `GET /resources`
  - `POST /resources`
  - `GET /resources/:id`
- Skills
  - `GET /skills`
  - `POST /skills`
  - `GET /skills/:id`
  - `PUT /skills/:id`
- Knowledge
  - `GET /knowledge-bases`
  - `POST /knowledge-bases`
  - `GET /knowledge-bases/:id`
  - `PUT /knowledge-bases/:id`
- Agents
  - `GET /agents`
  - `POST /agents`
  - `GET /agents/:id`
  - `PUT /agents/:id`
- Clients
  - `GET /clients`
  - `POST /clients`
  - `GET /clients/:id`
  - `PUT /clients/:id`
- Version / Exposure / Lifecycle
  - `POST /resources/:id/versions`
  - `GET /resources/:id/versions/:version`
  - `GET /resources/:id/exposures`
  - `POST /resources/:id/exposures`
  - `GET /resources/:id/exposures/:target`
  - `PUT /resources/:id/exposures/:target`
  - `POST /resources/:id/exposures/:target/revoke`
  - `POST /resources/:id/publish`
  - `POST /resources/:id/disable`
  - `POST /resources/:id/revoke`
  - `POST /resources/:id/offline`
  - `POST /resources/:id/versions/:version/rollback`

### 5.2 Open Capabilities

Base path：`/api/open-capabilities`

当前已存在：

- `GET /discovery`
- `GET /resources/:id`
- `POST /refresh`
- `POST /skills/:id/invoke`
- `POST /knowledge-bases/:id/query`
- `GET /agents/:id`

## 6. 当前 DTO / 契约形态

### 6.1 Client Registration DTO

当前 create payload 形态：

```json
{
  "slug": "cherry-studio",
  "display_name": "Cherry Studio",
  "client_type": "desktop",
  "status": "active",
  "allowed_grant_types": ["authorization_code", "refresh_token"],
  "redirect_uris": ["cherrystudio://oauth/callback"],
  "allowed_scopes": [
    "ap.resources.read",
    "ap.skills.invoke",
    "ap.knowledge.query",
    "ap.agents.read"
  ],
  "contract_version": "2026-06",
  "capabilities": {
    "discovery": true
  },
  "extensions": {},
  "allow_client_credentials": false
}
```

说明：

- 这是后端当前可接受的 control-plane registration 形态
- `capabilities`、`extensions` 为 JSON 结构
- 当前还没有完整的 UI 管理界面承载它

### 6.2 OAuth Authorize Query Contract

当前 query 参数：

- `client_id`
- `redirect_uri`
- `scope`
- `state`
- `code_challenge`
- `code_challenge_method`

当前 JSON 返回字段：

- `client_id`
- `contract_version`
- `scope`
- `state`
- `authorization_code`
- `redirect_uri`
- `consent_recorded`

重要说明：

- 如果下游期待传统浏览器 302 redirect callback 体验，当前实现应视为“服务能力已存在”，而不是“最终 UX 契约已冻结”

### 6.3 OAuth Token Exchange

当前 JSON 请求字段：

- `client_id`
- `grant_type`
- `code`
- `code_verifier`
- `redirect_uri`
- `refresh_token`
- `scope`

当前 token 返回字段：

- `access_token`
- `token_type`
- `expires_in`
- `refresh_token`
- `refresh_expires_in`
- `scope`
- `contract_version`
- `grant_id`

### 6.4 Discovery Item

当前 discovery item 字段：

- `resource_id`
- `resource_type`
- `display_name`
- `resource_version`
- `contract_version`
- `visibility_state`
- `callable_state`
- `freshness_ttl_seconds`
- `freshness`
- `etag`
- `extensions`

与下游诉求相比，以下字段目前**没有作为标准 discovery 公共字段冻结**：

- `source`
- `accountId`
- `tenantId`
- `disabledReason`
- `fetchedAt`
- `expiresAt`

判断：

- 当前 discovery/detail 已经可用
- 但如果 Cherry Studio 企业 Provider 需要你给的那套更丰富字段矩阵，还需要再做一轮公共契约冻结

## 7. 统一错误 Envelope

当前已存在统一错误 envelope 方向，最小结构可概括为：

```json
{
  "success": false,
  "error": {
    "code": "permissionDenied",
    "message": "permission denied",
    "retryable": false,
    "request_id": "req_xxx",
    "resource_id": "res_xxx",
    "resource_version": "1.0.0"
  }
}
```

当前已明确覆盖的错误类别：

| Code | 当前含义 |
| --- | --- |
| `permissionDenied` | 权限不足、client 无效、grant 被撤销、未授权访问 |
| `resourceRevoked` | 资源已撤销 |
| `resourceOffline` | 资源下线 / 禁用 |
| `quotaOrRateLimited` | 预留分类，当前代码层并未完全覆盖所有调用路径 |
| `timeout` | Skill invoke 超时 |
| `upstreamFailed` | 上游业务失败 / HTTP 失败 |
| `contractInvalid` | 契约不合法、binding 无效、资源 detail 不完整 |

注意：

- 与你给的下游需求相比，当前 envelope 还不是完整的最终版企业 provider 规格
- 但方向和基础字段已经有了

## 8. 对你提供的下游需求的逐项判断

### 8.1 已满足或基本满足

- OAuth authorize/token/revoke 路径存在
- PKCE `S256` 存在
- client registration 存在
- access token / refresh token / revoke 存在
- open capability discovery / detail / refresh 存在
- 统一错误 envelope 基础存在
- visible / callable 分离存在
- stale / offline / revoked freshness 语义存在
- Skill invoke 路由存在
- Knowledge query 路由存在
- Agent definition read 路由存在

### 8.2 部分满足

- 企业登录产品流：后端能跑，但前端运营工作流未产品化
- 模型发现：当前 Agent Platform 并未单独冻结你要求的 enterprise model discovery 公共规格
- Knowledge metadata：后端资源模型存在，但还未整理成正式下游公共文档
- Skill metadata 与 invoke 契约：后端可支撑，但尚未抽成正式下游规格包
- 诊断与 trace：有部分字段，但还未形成签核级对外说明

### 8.3 尚未满足或尚未冻结为公共规格

- 没有一份正式下游可直接签核的 enterprise provider integration spec
- 当前 Agent Platform UI 没有完整 `Clients` 工作区
- 资源 authoring / exposure / auth onboarding 还没有完整运营流
- host allowlist / mock server / fixture 分发机制未正式冻结
- 模型发现还没有冻结成你需求中那样的正式契约

## 9. 给下游团队的当前接入建议

下游团队当前可以安全推进：

- OAuth + PKCE 接入原型
- token lifecycle 处理
- open capability discovery 集成
- detail 加载
- refresh / stale / revoked 处理

下游团队当前不应直接假设：

- 授权 UX 已定型
- 运营人员能完全通过当前 UI 完成 client onboarding
- model discovery 已冻结成最终公共协议
- 当前 web/default Agent Platform UI 已覆盖全部后端能力

## 10. 建议的下一份正式冻结文档

建议后续继续输出一个更强约束版本的下游契约文档，专门冻结：

- OAuth authorize/token/revoke wire protocol
- client registration schema
- discovery/detail/refresh response schema
- skill invoke request/response schema
- knowledge query request/response schema
- agent detail schema
- 统一错误 envelope
- freshness / revoke / retry 语义

## 11. AP-6 待冻结契约清单

### 11.1 OAuth / Token / Revoke Wire Contract

Status: pending freeze

Owner story: `ap-6-1-freeze-oauth-token-revoke-and-callback-wire-contract`

Must freeze:

- authorize request / response
- browser redirect vs JSON authorize 行为
- token exchange
- refresh
- revoke
- callback / redirect URI allowlist
- consent visibility
- auth-related error codes

### 11.2 Client Registration Contract

Status: pending freeze

Owner story: `ap-6-2-freeze-client-registration-schema-and-onboarding-flow`

Must freeze:

- client fields
- capabilities schema
- extensions schema
- invalid integration reasons
- onboarding checklist
- API-first vs UI-required steps

### 11.3 Discovery / Detail / Refresh Contract

Status: pending freeze

Owner story: `ap-6-3-freeze-discovery-detail-and-refresh-public-fields`

Must freeze:

- resource fields and MUST / SHOULD / MAY level
- freshness semantics
- visible / callable semantics
- diagnostics object
- optional `source` / `accountId` / `tenantId` / `disabledReason` / `fetchedAt` / `expiresAt` decision

### 11.4 Skill Invoke and Knowledge Query Contracts

Status: pending freeze

Owner story: `ap-6-4-freeze-skill-invoke-and-knowledge-query-specs`

Must freeze:

- Skill metadata and schema fields
- sync invoke request / response
- Knowledge retrieval query request / response
- citations / items shape
- provider failure mapping
- P0 exclusion for async / cancel / task status if not supported

### 11.5 Enterprise Model Discovery Contract

Status: missing / must add

Owner story: `ap-6-5-freeze-enterprise-model-discovery-contract`

This is not equivalent to `/api/models`, `/api/user/models`, or `/v1/models`.

Must freeze:

- `modelId`
- `providerStableId`
- `displayName`
- `isDefault`
- `status`
- `disabledReason`
- `capabilities`
- `accountId` / `tenantId` source semantics
- no default / multiple default / default disabled states

### 11.6 Error Matrix and Client State Matrix

Status: pending freeze

Owner story: `ap-6-6-freeze-error-code-and-client-state-matrices`

Must freeze mappings from platform errors to client states:

- `permissionDenied` -> `loginExpired` or `noAssignedResource` depending on auth/resource context
- `resourceRevoked` -> `revoked`
- `resourceOffline` -> `offline`
- `quotaOrRateLimited` -> retryable quota/rate limited state
- `timeout` -> retryable timeout state
- `upstreamFailed` -> provider/platform failure state
- `contractInvalid` -> visible-but-not-callable / integration invalid state

### 11.7 Mock / Fixture / Conformance

Status: missing / must add

Owner story: `ap-6-7-provide-mock-fixture-and-contract-conformance-suite`

Must provide synthetic fixtures for:

- OAuth success / expired / revoked
- discovery empty / success
- detail visible but not callable
- refresh stale / revoked / offline
- Skill invoke success / contract invalid / upstream failed
- Knowledge query success / provider offline
- model discovery default / no default / disabled
- error matrix states

### 11.8 Consumer Signoff

Status: pending

Owner story: `ap-6-8-complete-cherry-studio-first-and-codex-second-signoff`

Must record:

- Cherry Studio first-consumer signoff result
- Codex second-consumer review result
- any extension namespaces used
- no parallel core protocol created

## 12. 当前 readiness verdict

对于“授权给客户端登录并接入下游 newapi enterprise provider 的能力”这一问题，当前结论是：

- 后端接入底座：**usable**
- 下游公共契约成熟度：**partial**
- 运营/控制台工作流完整度：**not yet complete**

换句话说：

- 现在已经不是“完全没做”
- 但也还不能说“下游只看一份完全冻结的正式公共规格就能无歧义接入全部能力”

## 13. 审阅建议

你可以重点带着下面三个问题看这份文档：

1. Cherry Studio 这类下游当前最关心的是不是 OAuth + discovery + invoke/query 的基础闭环？
2. 他们是否必须在 Phase 0 就拿到“模型发现”的正式公共规格，而不是后续补冻结？
3. 他们是否接受“后端能力先接入、运营控制台后补齐”的推进顺序？

如果这三个问题里有任何一个答案是否定的，就应该继续发起下一轮 CC，把下游公共契约进一步冻结。  
