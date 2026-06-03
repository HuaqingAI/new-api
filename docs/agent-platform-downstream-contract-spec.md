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

状态：**AP-6.2 schema frozen，后端已实现，UI 未完全产品化**

当前已实现接口：

- `GET /api/agent-platform/clients`
- `POST /api/agent-platform/clients`
- `GET /api/agent-platform/clients/:id`
- `PUT /api/agent-platform/clients/:id`

当前冻结字段：

| 字段 | 等级 | 语义 | 默认值 / 派生 | 校验要求 |
| --- | --- | --- | --- | --- |
| `client_id` | MUST response | 平台生成的稳定 opaque id，是下游 OAuth 与 open capability 授权时使用的 client id。 | create 时由服务端生成；客户端不得提交或推导。 | 非空；稳定；不得把 `slug`、名称或数据库自增 id 当成公开契约。 |
| `slug` | MUST request/response | 管理侧可读标识，用于运维检索与人工沟通。 | create 必填；runtime 归一为小写 trim 后保存。 | 非空；全局唯一；不得作为 OAuth `client_id` 使用。 |
| `display_name` | MUST request/response | 管理端展示名称。 | create 必填。 | 非空。 |
| `client_type` | MUST request/response | 客户端类型，例如 `desktop`、`cli`、`server`。 | create 必填；runtime 归一为小写 trim 后保存。 | 非空；用于治理解释，不创建平行 client object model。 |
| `status` | SHOULD request, MUST response | client 集成状态。 | create 未传时：若 `contract_version` 与 `capabilities` 非空则为 `active`，否则为 `invalid_integration`。 | 只允许 `active`、`disabled`、`invalid_integration`；其他值为 invalid input。 |
| `allowed_grant_types` | SHOULD | OAuth grant allowlist。 | 空值表示未声明；authorize/token 会按 OAuth 子契约拒绝不满足的 grant。 | 必须是字符串数组；建议包含 `authorization_code`，按需包含 `refresh_token`、`client_credentials`。 |
| `redirect_uris` | SHOULD for OAuth | OAuth callback allowlist。 | 空值表示未配置 callback。 | 必须是字符串数组；authorize 时 exact match，无 wildcard、host suffix 或 scheme fallback。 |
| `allowed_scopes` | SHOULD | 下游可请求 scope allowlist。 | 空值表示没有授权 scope。 | 必须是字符串数组；authorize/token 请求的每个 scope 都必须存在。 |
| `contract_version` | SHOULD request, MUST response | 客户端声明遵循的公共契约版本。 | 空值会使默认状态进入 `invalid_integration`。 | AP-6 当前签核版本为 `2026-06`；不得由 extension 改写。 |
| `capabilities` | SHOULD | 客户端能力声明 JSON，例如 discovery、skill invoke、knowledge query 需求。 | 空值会使默认状态进入 `invalid_integration`。 | 必须是合法 JSON；当前 runtime 不冻结更细子 schema，consumer signoff 按存在性治理。 |
| `extensions` | EXTENSION | 下游私有治理字段。 | 默认为空。 | 必须是合法 JSON object；顶层 key 必须 namespaced 且包含 `.`，例如 `cherry_studio.owner`、`codex.profile`；不得重写核心字段语义。 |
| `allow_client_credentials` | SHOULD | 是否允许该 client 使用 client credentials。 | create 默认 `false`；update 使用显式布尔指针以保留 `false`。 | 只有为 `true` 且 `allowed_grant_types` 包含 `client_credentials` 时，token endpoint 才可执行该 grant。 |

当前 identity 模型：

- 使用稳定 opaque `client_id`
- 支持 namespaced extensions
- `contract_version` 是正式治理字段
- `capabilities` 是正式治理字段
- Codex second-consumer 复用同一套 client registration schema，不新增平行 client object model

当前缺口：

- 当前 Agent Platform 页没有真正的 `Clients` 工作区
- 运维人员还不能只靠 web/default 完成完整 onboarding
- 当前 AP-6.2 最小闭环由 API / fixture 支撑；完整自助 Clients 工作区拆入后续产品化 epic

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
  "extensions": {
    "cherry_studio.owner": "ops"
  },
  "allow_client_credentials": false
}
```

说明：

- 这是后端当前可接受的 control-plane registration 形态
- `capabilities`、`extensions` 为 JSON 结构，且 `extensions` 顶层 key 必须 namespaced
- 当前还没有完整的 UI 管理界面承载它

最小 onboarding checklist：

| 步骤 | 等级 | 当前支撑面 | 完成判定 |
| --- | --- | --- | --- |
| registration | MUST | API / OpenAPI / fixture | 已创建 client，`client_id` 稳定返回，`slug`、`display_name`、`client_type` 可读。 |
| redirect/callback | MUST for OAuth | API / OAuth runtime tests | `redirect_uris` 包含 exact callback，authorize mismatch 明确失败。 |
| allowed scopes | MUST for scoped access | API / OAuth runtime tests | `allowed_scopes` 覆盖下游请求 scope。 |
| contract version | MUST for active integration | API / conformance | `contract_version` 非空并与当前签核版本一致。 |
| capabilities | MUST for active integration | API / conformance | `capabilities` 非空且可由运营解释。 |
| consent / OAuth prerequisites | SHOULD | OAuth runtime / 后续 UX | 当前 JSON authorize 能记录 consent；完整用户可见 consent 页面属于后续产品化。 |
| mock fixture | SHOULD | `tests/agentplatform/conformance/` | 使用 synthetic client/resource/token，不含真实凭据或企业数据。 |

状态语义：

| 状态 | 语义 | 进入条件 | 下游解释 |
| --- | --- | --- | --- |
| `active` | 集成配置满足当前最小契约，可参与 OAuth / open capability 流程。 | create 时 `contract_version` 与 `capabilities` 非空且未显式传其他状态。 | 可以继续执行授权、发现、详情或调用流程。 |
| `disabled` | 管理员显式停用 client。 | API update 设置 `status=disabled`。 | 不应重试；需管理员重新启用。 |
| `invalid_integration` | 注册存在但配置不完整或契约不满足。 | 缺 `contract_version`、缺 `capabilities`，或 OAuth 时缺 grant / callback / scope 等配置。 | 非 generic failure；运营应修复 registration、redirect/callback、scope、contract 或 capability 字段。 |

AP-6.2 不要求完整 Clients 产品化工作区。当前必须在 API / fixture / signoff 文档中完成最小闭环，不得把现有 Agent Platform shell 描述成完整自助 onboarding UI。

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

Status: frozen for signoff

Owner story: `ap-6-1-freeze-oauth-token-revoke-and-callback-wire-contract`

Source of truth:

- DTO: `dto/agentplatform/oauth.go`
- Runtime: `controller/agentplatform/oauth.go`; `service/agentplatform/oauth_authorize.go`; `service/agentplatform/oauth_token.go`
- OpenAPI: `docs/openapi/api.json`
- Conformance fixtures: `tests/agentplatform/conformance/fixtures.go`
- Runtime tests: `controller/agentplatform/oauth_test.go`; `service/agentplatform/oauth_authorize_test.go`; `service/agentplatform/oauth_token_test.go`

Current AP-6.1 freezes the wire capability that exists today. `GET /api/agent-platform/oauth/authorize` currently returns a JSON authorization result in the standard API envelope. It does not yet provide the final operator-friendly consent page or browser `302` redirect callback product flow. A future browser UX MAY wrap this same core contract, but MUST NOT change the authorize/token/revoke field semantics without a versioned contract update.

Auth plane boundary:

- Downstream clients MUST NOT read dashboard session cookies, relay tokens, upstream provider OAuth tokens, refresh token hashes, platform internal grant rows, or provider secrets.
- Downstream clients MUST rely only on the AP auth plane authorization result, AP bearer token, refresh token when issued, and the public fields documented here.
- Dashboard session is only the current platform user identity source for authorize; it is not a downstream credential.

Authorize request: `GET /api/agent-platform/oauth/authorize`

| Field | Level | Semantics |
| --- | --- | --- |
| `client_id` | MUST | Registered Agent Platform integration client id. Client must exist, be `active`, have non-empty `contract_version`, non-empty `capabilities_json`, include `authorization_code` in `allowed_grant_types_json`, and allow every requested scope. |
| `redirect_uri` | MUST | Callback URI. It MUST exactly match one entry in `agent_platform_clients.redirect_uris_json`. No wildcard, host suffix, scheme fallback, or fuzzy normalization is supported in AP-6.1. |
| `scope` | MUST | Space-delimited scopes. Every scope MUST exist in `allowed_scopes_json`; missing or extra scopes are rejected as non-retryable configuration/request errors. |
| `state` | SHOULD | Client correlation value returned unchanged in the JSON authorize response. |
| `code_challenge` | MUST | PKCE S256 challenge for authorization-code exchange. |
| `code_challenge_method` | MUST | Only `S256` is supported. `plain` PKCE is not in AP-6.1 P0. |

Authorize success response data:

| Field | Level | Semantics |
| --- | --- | --- |
| `client_id` | MUST | Echoes the authorized client id. |
| `contract_version` | MUST | Client contract version used for this grant. |
| `scope` | MUST | Authorized scope string. |
| `state` | SHOULD | Echoes request `state`; empty string when absent. |
| `authorization_code` | MUST | Short-lived code used only with `grant_type=authorization_code`. |
| `redirect_uri` | MUST | Echoes the exact accepted redirect URI. |
| `consent_recorded` | MUST | `true` when `agent_platform_authorization_grants.consented_at` was set for this grant. |

Token request: `POST /api/agent-platform/oauth/token`

| Field | Level | Semantics |
| --- | --- | --- |
| `client_id` | MUST | Registered integration client id. Client must be active and contract-valid. |
| `grant_type` | SHOULD | One of `authorization_code`, `refresh_token`, `client_credentials`. Empty value is normalized to `authorization_code` by current runtime. |
| `code` | MUST for `authorization_code` | Authorization code returned by authorize. |
| `code_verifier` | MUST for `authorization_code` | PKCE verifier; S256 hash MUST match the stored `code_challenge`. |
| `redirect_uri` | MUST for `authorization_code` | MUST still exactly match the registered redirect allowlist. |
| `refresh_token` | MUST for `refresh_token` | Opaque `rt_` token. Runtime stores only HMAC hash and rotates on use. |
| `scope` | SHOULD for `client_credentials` | Requested client-credentials scopes; when absent, runtime uses allowed scopes. |

Token success response data:

| Field | Level | Semantics |
| --- | --- | --- |
| `access_token` | MUST | First-party HS256 AP bearer token. It is not a relay token and not an upstream provider token. |
| `token_type` | MUST | `Bearer`. |
| `expires_in` | MUST | Access-token TTL in seconds. Current runtime: `300`. |
| `refresh_token` | MUST | Opaque refresh token returned for successful exchanges; synthetic examples only in fixtures. |
| `refresh_expires_in` | MUST | Refresh-token TTL in seconds. Current runtime target: 30 days (`2592000`). |
| `scope` | MUST | Granted scope string. |
| `contract_version` | MUST | Contract version embedded into access token claims. |
| `grant_id` | MUST | Public grant identifier used for diagnostics and audit correlation; not a database row id contract. |

Grant types:

- `authorization_code` MUST validate code, verifier, redirect URI, grant status `authorized`, non-expired code, and non-revoked grant. Success changes grant status to `issued`.
- `refresh_token` MUST rotate the refresh token: create a new token, revoke the old token, set old `last_used_at`, and preserve grant continuity.
- `client_credentials` MAY be used only when `agent_platform_clients.allow_client_credentials = true` and `allowed_grant_types_json` includes `client_credentials`. It creates no end-user consent semantics.

Revoke request: `POST /api/agent-platform/oauth/revoke`

| Field | Level | Semantics |
| --- | --- | --- |
| `client_id` | MUST | Client owning the token/grant. |
| `token` | MUST | Access token or refresh token to revoke. |
| `token_type_hint` | SHOULD | `access_token` or `refresh_token`; runtime also treats `rt_` prefix as refresh-token evidence. |

Revoke success response data:

| Field | Level | Semantics |
| --- | --- | --- |
| `revoked` | MUST | `true` on successful revoke. Refresh-token revoke revokes refresh token and grant; access-token revoke revokes grant and increments `grant.token_version`. |

Consent, grant, client state, and audit:

- Consent fact source is `agent_platform_authorization_grants.consented_at` and the grant record. AP-6.1 does not claim a complete user-visible consent page.
- Grant state MUST at least distinguish `authorized`, `issued`, and `revoked`.
- Access token validity converges through `grant.token_version` and `revoked_at`; refresh token validity converges through token hash, expiry, rotation, and revocation.
- Successful revoke with an actor SHOULD write `agentplatform.oauth.revoke` admin action.
- Audit payload MUST NOT include bearer tokens, refresh token plaintext, refresh token hash, provider secret, tenant secret, or upstream OAuth token.

Error and retryability semantics:

| Scenario | Client interpretation | Retryable |
| --- | --- | --- |
| malformed request, missing required field, bad grant type | request/config error | no |
| redirect mismatch | integration configuration error | no |
| scope mismatch or missing scope | integration configuration / permission error | no |
| unsupported PKCE method or bad verifier | request/auth error | no |
| inactive client, missing contract version, missing capabilities, invalid integration | integration configuration error | no |
| expired access token | refresh-token exchange may be attempted when a valid refresh token is available | yes, via refresh only |
| expired/revoked/reused refresh token | login/authorization must restart | no |
| revoked grant | same token cannot recover; authorization must restart | no |
| platform/internal failure | apply common platform retry policy | yes |

Downstream examples:

- Legal callback: `redirect_uri=cherrystudio://oauth/callback` exactly matches one registered allowlist entry and returns JSON `authorization_code`.
- Callback mismatch: `redirect_uri=https://evil.example/callback` does not exactly match and returns a non-retryable authorization failure.
- Scope mismatch: requesting `ap.admin.write` when only `ap.resources.read` is allowed returns a non-retryable authorization failure.
- Inactive or invalid integration client: inactive status, empty `contract_version`, empty `capabilities_json`, or missing `authorization_code` grant support returns a non-retryable client authorization failure.

### 11.2 Client Registration Contract

Status: frozen

Owner story: `ap-6-2-freeze-client-registration-schema-and-onboarding-flow`

Source of truth:

- DTO: `dto/agentplatform/client.go`
- Runtime: `controller/agentplatform/client.go`; `service/agentplatform/client.go`
- OpenAPI: `docs/openapi/api.json`
- Runtime tests: `controller/agentplatform/client_test.go`; `service/agentplatform/client_test.go`
- Consumer signoff: `docs/agent-platform-consumer-signoff.md`

AP-6.2 client registration schema freeze covers:

- field levels and semantics for `client_id`, `slug`, `display_name`, `client_type`, `status`, `allowed_grant_types`, `redirect_uris`, `allowed_scopes`, `contract_version`, `capabilities`, `extensions`, and `allow_client_credentials`
- `client_id` as stable opaque id and `slug` as management-readable identifier
- namespaced `extensions` keys that cannot override core contract semantics
- `invalid_integration`, `active`, and `disabled` status semantics
- onboarding checklist split between current API / fixture support and future UI productization

Invalid configurations that must resolve to explicit onboarding state instead of generic failure include missing `contract_version`, missing `capabilities`, disabled client, missing `authorization_code` grant support, redirect mismatch, scope mismatch, and non-namespaced extensions. Runtime returns invalid input for malformed schema and exposes persisted incomplete client records as `invalid_integration`.

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

Status: provided for frozen OAuth and open capability surfaces; AP-6.5 / AP-6.6 dependent cases are explicit pending fixtures

Owner story: `ap-6-7-provide-mock-fixture-and-contract-conformance-suite`

Source of truth artifacts:

- Fixture catalog and conformance assertions: `tests/agentplatform/conformance/`
- Runtime handler conformance coverage: `controller/agentplatform/open_capabilities_test.go`
- OpenAPI public path/schema source: `docs/openapi/api.json`

Local / CI commands:

- `GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance`
- `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`

Synthetic fixture coverage:

- OAuth authorize success / token success / refresh rotation success / revoke success
- OAuth expired token / revoked grant / missing scope / redirect mismatch / invalid PKCE / invalid client / invalid integration
- discovery empty / success / multi-resource / contract mismatch filtered or rejected
- detail visible + callable / visible but not callable / contract invalid / resource revoked / resource offline
- refresh fresh / stale / revoked / offline / observed ETag-version mismatch / TTL over 300 seconds non-compliance diagnostics
- Skill invoke sync success / contract invalid / timeout / upstream provider failure
- Knowledge query retrieval success with items and citations / provider offline / upstream provider failure / no provider-native field leakage
- model discovery default / no default / multiple defaults / default disabled / provider offline / unavailable / account-tenant mismatch: pending with AP-6.5 reason because enterprise model discovery is not frozen
- error matrix states: pending with AP-6.6 reason because the client state matrix artifact is not frozen

Safety constraints:

- Fixtures use synthetic IDs such as `client_cherry_mock`, `tenant_demo`, `acct_demo`, `res_skill_demo`, `res_knowledge_demo`, `res_agent_demo`, and `model_demo_default`.
- Fixtures must not contain real access tokens, refresh tokens, provider secrets, tenant secrets, real user information, or real enterprise data.
- OAuth token examples use synthetic placeholders only; provider-native Knowledge fields and provider config are not exposed in conformance payloads.
- Model discovery pending fixtures must not be satisfied by `/api/models`, `/api/user/models`, or `/v1/models`.

### 11.8 Consumer Signoff

Status: blocked

Owner story: `ap-6-8-complete-cherry-studio-first-and-codex-second-signoff`

Signoff artifact:

- `docs/agent-platform-consumer-signoff.md`

Current coverage matrix:

| Consumer / domain | Status | Evidence | Notes |
| --- | --- | --- | --- |
| Cherry Studio OAuth authorize/token/revoke/callback/allowlist | signed off | `docs/openapi/api.json`; `tests/agentplatform/conformance/`; `controller/agentplatform/oauth_test.go`; `service/agentplatform/oauth_authorize_test.go`; `service/agentplatform/oauth_token_test.go` | Current wire contract is JSON authorize plus token/revoke APIs. Final browser redirect UX is productization over the same OAuth contract, not a separate core protocol. |
| Cherry Studio resource discovery/detail/refresh | signed off | `docs/openapi/api.json`; `tests/agentplatform/conformance/`; `controller/agentplatform/open_capabilities_test.go` | Covers TTL, freshness, ETag, revoked/offline/stale convergence, visibility/callable state, diagnostics, and contract compatibility. |
| Cherry Studio Skill invoke and Knowledge query | signed off | `tests/agentplatform/conformance/`; `controller/agentplatform/open_capabilities_test.go` | Covers success, timeout, upstream failure, contract invalid, provider offline, standardized items/citations, and provider-native field non-leakage. |
| Cherry Studio enterprise model discovery | blocked | `tests/agentplatform/conformance/fixtures.go` pending AP-6.5 fixtures | AP-6.5 is not frozen; this must not be satisfied by `/api/models`, `/api/user/models`, or `/v1/models`. |
| Cherry Studio error/client state matrix | blocked | `tests/agentplatform/conformance/fixtures.go` pending AP-6.6 fixture | AP-6.6 is not frozen; pending fixture cannot be treated as completed signoff. |
| Codex second-consumer review | signed off | `docs/agent-platform-consumer-signoff.md` | Codex can reuse `client`, `contract_version`, `capabilities`, open capability resources, error envelope, OpenAPI, and fixture/conformance assets. |

Extension namespaces:

- `extensions.cherry_studio`
- `extensions.codex`

Conclusion:

- Cherry Studio overall first-consumer signoff remains `blocked` until AP-6.5 and AP-6.6 are frozen.
- Codex second-consumer review does not require a parallel resource model, parallel core protocol trunk, or independent open capability endpoint; no parallel core protocol is required.
- No `docs/openapi/api.json` update is required for this story because it adds only signoff documentation and conformance drift checks, not a new public API or schema.

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
