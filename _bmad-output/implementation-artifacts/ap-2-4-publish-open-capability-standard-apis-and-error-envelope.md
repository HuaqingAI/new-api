# Story 2.4: 发布开放能力标准接口与统一错误语义

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 客户端集成人员,
I want 通过共享开放能力层访问 discovery、detail、invoke/query、refresh,
so that 我能通过一套稳定协议完成接入，而不是面向不同消费者写不同接口。

## Acceptance Criteria

1. **Given** Agent Platform 资源已经发布  
   **When** 下游客户端访问开放能力面  
   **Then** 它可以通过 `/api/open-capabilities/**` 使用 discovery、detail、invoke/query、refresh 语义  
   **And** 这些路由与 control plane、OAuth 路由保持边界分离。
2. **Given** 任意开放能力调用成功或失败  
   **When** 平台返回响应  
   **Then** 平台对 `Skill`、`Knowledge`、`Agent` 与客户端校验路径统一使用标准 envelope  
   **And** 错误返回包含稳定类别与可重试语义，而不是依赖字符串猜测。
3. **Given** 某消费者需要私有扩展字段  
   **When** 平台返回契约数据  
   **Then** 扩展字段以 namespaced 方式承载  
   **And** 核心 discovery/detail/invoke/refresh 语义不被改变。

## Tasks / Subtasks

- [x] 建立开放能力层的统一 bearer 校验与标准 envelope (AC: 1, 2, 3)
  - [x] 新增面向 `/api/open-capabilities/**` 的 Agent Platform bearer 中间件，基于 `service/agentplatform/oauth_token.go` 的 JWT 校验和 scope 规则，而不是复用现有主 API 的 `TokenAuth()`。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Communication Patterns; _bmad-output/planning-artifacts/architecture-agent-platform.md#493-497]
  - [x] 新增统一 success/error envelope 映射，错误至少覆盖：`permissionDenied`、`resourceRevoked`、`resourceOffline`、`quotaOrRateLimited`、`timeout`、`upstreamFailed`、`contractInvalid`，并返回 `retryable` / `request_id` / `resource_id` / `resource_version` 等稳定字段。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#638-670]
  - [x] 所有开放能力层响应必须保持核心字段标准化，消费者私有扩展只能进入 namespaced `extensions` 对象。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Contract versioning and extension governance]

- [x] 实现 published projection 的 discovery/detail/refresh 最小闭环 (AC: 1, 2, 3)
  - [x] 新增 `service/agentplatform/discovery.go`（或等价命名），只返回已发布且对当前 client 可见的 exposure 投影，不从 registry/status 直接推断 discovery 结果。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#577-580; _bmad-output/implementation-artifacts/ap-1-5-build-publication-projection-and-exposure-model.md]
  - [x] detail 返回标准化资源契约数据，至少覆盖 resource 基本信息、resource version、contract version、TTL、freshness、etag、callability 状态和支持的 extension namespaces。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#577-608]
  - [x] refresh 返回当前投影的 version/etag/freshness 视角，供客户端做收敛对账，但本故事不需要完整实现 2.5 的“超时后强制 stale 合规诊断”全套策略。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#580-595]

- [x] 发布 `/api/open-capabilities/**` 路由骨架并守住资源边界 (AC: 1, 2, 3)
  - [x] 路由至少包含：`/api/open-capabilities/discovery`、`/api/open-capabilities/resources/:id`、`/api/open-capabilities/skills/:id/invoke`、`/api/open-capabilities/knowledge-bases/:id/query`、`/api/open-capabilities/agents/:id`、`/api/open-capabilities/refresh`。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#570-573]
  - [x] 在 `2.4` 范围内，`skills/:id/invoke` 与 `knowledge-bases/:id/query` 先提供标准 envelope + `contractInvalid` 占位行为，路由语义、边界和错误模型先固定下来。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Implementation order mandated by this architecture; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 3.3; #Story 4.3]
  - [x] `agents/:id` detail 只返回定义与依赖边界相关的标准化 detail，不越界承诺服务端 runtime。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Agent runtime boundary]

- [x] 衔接现有 exposure/resource/version/typed detail 模型 (AC: 1, 2, 3)
  - [x] open capability discovery/detail 直接消费 `1.5` 已建立的 per-target exposure 模型，继续沿用 `client_key` 表达目标 client identity，不重构 exposure 表。[Source: _bmad-output/implementation-artifacts/ap-1-5-build-publication-projection-and-exposure-model.md]
  - [x] detail 读取复用 `resource`、`resource_version` 已有 typed detail 快照事实源，不重新拼消费者私有 DTO。[Source: service/agentplatform/resource_version.go]
  - [x] 继续遵守 `common.*` JSON wrapper 与跨库兼容规则，不引入数据库专属 JSON 查询实现。[Source: AGENTS.md#Rule 1; AGENTS.md#Rule 2]

- [x] 补定向测试与最小 OpenAPI 记录 (AC: 1, 2, 3)
  - [x] 新增/更新 `service/agentplatform/discovery_test.go`、`controller/agentplatform/*_test.go`，覆盖：bearer scope 校验、published-only discovery、detail 正常返回、错误 envelope 稳定字段、refresh 最小对账结果。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Testing organization]
  - [x] 对 `/api/open-capabilities/**` 的最小管理文档更新 `docs/openapi/api.json`，但不提前冻结 3.x/4.x 的完整业务 payload 细节。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Docs organization]

## Dev Notes

### Previous Story Insights

- `1.5` 已经把 exposure 做成 per-target published projection，因此 `2.4` 的 discovery/detail 必须以 exposure 为 source of truth，而不是 resource status 本身。[Source: _bmad-output/implementation-artifacts/ap-1-5-build-publication-projection-and-exposure-model.md]
- `2.3` 已经完成 Agent Platform JWT access token、refresh token 与 revoke 逻辑，因此 `2.4` 可以直接在此基础上建立 bearer 校验中间件和 scope 判定。[Source: _bmad-output/implementation-artifacts/ap-2-3-implement-token-lifecycle-and-revocation.md]

### Architecture Constraints

- 开放能力层与 control plane / OAuth plane 严格分离：
  - control plane: `/api/agent-platform/**`
  - auth plane: `/api/agent-platform/oauth/**`
  - open capability plane: `/api/open-capabilities/**`
- `2.4` 是实现顺序中的第 3 步：先做 discovery/detail for published projections；Skill invoke / Knowledge query 的完整契约深化属于后续 stories，但路由与 envelope 现在就要固定。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1298-1330]

### Scope Model

- 资源读取：`ap.resources.read`
- Skill 调用：`ap.skills.invoke`
- Knowledge 检索：`ap.knowledge.query`
- Agent 读取：`ap.agents.read`
- 不允许任何 scope 隐式取得 control-plane 管理权限。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#493-497]

### Testing

- 推荐至少运行：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`
- 如果新增路由/中间件编译面扩大，再补：
  - `GOCACHE=/private/tmp/go-build-cache go test ./router/... ./middleware/...`

### Project Structure Notes

- 预计新增文件：
  - `service/agentplatform/discovery.go`
  - `service/agentplatform/open_capability_errors.go`
  - `controller/agentplatform/open_capabilities.go`
  - `dto/agentplatform/open_capabilities.go`
  - `middleware/agent_platform_bearer.go`
  - 对应 `*_test.go`
- 预计修改文件：
  - `router/api-router.go` 或新增相关 router 文件
  - `docs/openapi/api.json`
- 保持不改或只读依赖：
  - `relay/**`
  - `web/default/**`
  - provider-specific retrieval runtime（留给 4.4）

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.4: 发布开放能力标准接口与统一错误语义]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#570-608]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#638-670]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1298-1330]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#UJ-1]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#UJ-3]
- [Source: _bmad-output/implementation-artifacts/ap-1-5-build-publication-projection-and-exposure-model.md]
- [Source: _bmad-output/implementation-artifacts/ap-2-3-implement-token-lifecycle-and-revocation.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 继续作为总编排状态源，但 `2.4` 采用 manual create/dev takeover，避免 create-story / dev-story 子进程在 Agent Platform Epic 2 上继续漂移。
- 已加载 `epics-agent-platform.md`、`architecture-agent-platform.md`、`prd-agent-platform-2026-05-31/prd.md`，并对齐 `1.5` exposure 基线与 `2.3` JWT token 基线，准备实现 `/api/open-capabilities/**`。

### Completion Notes List

- 已新增 Agent Platform bearer 中间件，使用 2.3 的 JWT access token 做 scope 校验，不再复用主 API 的 `TokenAuth()`。
- 已实现 `/api/open-capabilities/discovery`、`/resources/:id`、`/refresh` 的最小 published projection 闭环，并统一返回 TTL / freshness / etag / contract version 语义。
- 已挂载 `/skills/:id/invoke`、`/knowledge-bases/:id/query`、`/agents/:id` 路由骨架，其中 invoke/query 当前返回稳定 `contractInvalid` envelope，占位等待 3.x/4.x 资源闭环继续填充。
- 已新增统一 open capability error envelope，稳定返回 `code`、`message`、`retryable`、`request_id`、`resource_id`、`resource_version`。
- 已补 `docs/openapi/api.json` 最小路径定义，并通过 `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform ./middleware ./router` 验证。

### File List

- `_bmad-output/implementation-artifacts/ap-2-4-publish-open-capability-standard-apis-and-error-envelope.md`
- `service/agentplatform/open_capability_errors.go`
- `service/agentplatform/discovery.go`
- `service/agentplatform/discovery_test.go`
- `middleware/agent_platform_bearer.go`
- `dto/agentplatform/open_capabilities.go`
- `controller/agentplatform/open_capabilities.go`
- `controller/agentplatform/open_capabilities_test.go`
- `router/open_capabilities_router.go`
- `router/api-router.go`
- `docs/openapi/api.json`

## Change Log

- 2026-06-01: 完成开放能力层 bearer auth、discovery/detail/refresh、统一错误 envelope、最小标准路由与定向测试，并将故事推进为 done。
