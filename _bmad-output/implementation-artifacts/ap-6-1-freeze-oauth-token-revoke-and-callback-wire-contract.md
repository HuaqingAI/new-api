---
baseline_commit: 33ad91e09
---

# Story 6.1: 冻结 OAuth、Token、Revoke 与 Callback Wire Contract

Status: done

> Sprint key 映射：本故事来自 Agent Platform Epic 6，真实 sprint-status key 为 `ap-6-1-freeze-oauth-token-revoke-and-callback-wire-contract`。本文件按用户要求落地为 `_bmad-output/implementation-artifacts/ap-6-1-*.md`。注意：同一 epic 中既有 6.7 / 6.8 artifact 使用了 `6-7-*`、`6-8-*` 文件名，但 sprint key 仍是 `ap-6-*`；实现和状态同步必须以 sprint-status 中的 `ap-6-*` key 为准。

## Story

As a 客户端集成人员,
I want 获得明确的 OAuth / token / revoke / callback / allowlist wire contract,
so that 下游可以稳定实现企业登录与授权回调，而不是依赖当前后端行为猜测。

## Acceptance Criteria

1. **Given** 下游客户端准备接入授权流程  
   **When** 查阅 AP-6 公共契约  
   **Then** 文档明确 `authorize`、`token`、`refresh`、`revoke` 的请求、响应、错误码、redirect/callback 行为和 allowlist 规则  
   **And** 明确当前 JSON authorize 能力与最终浏览器 redirect UX 的关系。

2. **Given** 用户 consent、grant 或 client 状态影响授权  
   **When** 下游触发授权或 token 刷新  
   **Then** 契约明确对应状态、错误语义、可重试性和审计要求  
   **And** 不要求下游读取 dashboard session 或 relay token。

## Tasks / Subtasks

- [x] 冻结 OAuth / token / revoke 公共契约文档 (AC: 1, 2)
  - [x] 更新 `docs/agent-platform-downstream-contract-spec.md#11.1 OAuth / Token / Revoke Wire Contract`，将状态从 `pending freeze` 改为可签核状态，并补齐 `authorize`、`token`、`refresh_token`、`client_credentials`、`revoke` 的请求 / 响应 / 错误 / retryable 语义。
  - [x] 在同一文档中明确当前 `GET /api/agent-platform/oauth/authorize` 返回 JSON authorization code 的实现事实，以及最终浏览器 redirect callback UX 的目标关系：当前 AP-6.1 冻结 wire capability，不得伪称已完成完整 consent 页面或 302 redirect 产品流。
  - [x] 为每个字段标注 `MUST` / `SHOULD` / `MAY` / `EXTENSION`，尤其是 `client_id`、`redirect_uri`、`scope`、`state`、`code_challenge`、`code_challenge_method`、`authorization_code`、`access_token`、`refresh_token`、`token_type`、`expires_in`、`refresh_expires_in`、`contract_version`、`grant_id`、`token_type_hint`、`revoked`。
  - [x] 明确下游不得读取 dashboard session、relay token、upstream provider OAuth token、refresh token hash 或平台内部 grant 表；下游只依赖 AP auth plane 返回的 bearer token 和公开字段。

- [x] 冻结 callback / redirect URI allowlist 规则 (AC: 1)
  - [x] 记录 `redirect_uri` 必须与 `agent_platform_clients.redirect_uris_json` 完全匹配；不要实现 wildcard、host suffix、scheme fallback 或模糊归一化，除非先更新架构与测试。
  - [x] 记录 `code_challenge_method` 当前仅支持 `S256`；plain PKCE 不进入 AP-6.1 P0。
  - [x] 记录 authorize 必须校验 client 为 `active`、`contract_version` 非空、`capabilities_json` 非空、授权模式包含 `authorization_code`、scope 均在 `allowed_scopes_json` 中。
  - [x] 增加下游示例：合法 callback、callback mismatch、scope mismatch、inactive / invalid integration client 的响应语义。

- [x] 对齐 DTO、Controller、Service、OpenAPI 三件套 (AC: 1, 2)
  - [x] 以 `dto/agentplatform/oauth.go` 的 `OAuthAuthorizeRequest`、`OAuthAuthorizeResponse`、`OAuthTokenRequest`、`OAuthTokenResponse`、`OAuthRevokeRequest`、`OAuthRevokeResponse` 为唯一 DTO source，不要在文档、fixture 或测试里手写一套漂移字段。
  - [x] 校验 `controller/agentplatform/oauth.go` 返回的 success envelope 与 `docs/openapi/api.json` 中 `AgentPlatformOAuthAuthorizeResponse`、`AgentPlatformOAuthTokenResponse`、`AgentPlatformOAuthRevokeResponse` 一致。
  - [x] 若发现 OpenAPI 缺字段、错误 schema 不明确或 required 列表不完整，更新 `docs/openapi/api.json` 并扩展 `tests/agentplatform/conformance/fixtures_test.go` 的 anti-drift 断言。
  - [x] 若无需改 Go runtime，也要在 completion notes 中说明原因：本故事冻结既有实现，不重写 auth plane。

- [x] 扩展 OAuth fixture / conformance 覆盖 (AC: 1, 2)
  - [x] 更新 `tests/agentplatform/conformance/fixtures.go`，确保 OAuth fixture 至少覆盖 authorize success、token success、refresh rotation、revoke success、expired / revoked grant、missing scope、redirect mismatch、invalid PKCE、invalid client / invalid integration。
  - [x] 更新 `tests/agentplatform/conformance/fixtures_test.go`，断言 OAuth fixture 的 method、path、required fields、OpenAPI schema ref、pending reason 与 contract spec 11.1 保持一致。
  - [x] 禁止 fixture 包含真实 access token、refresh token、provider secret、tenant secret、真实企业数据或真实用户数据；只能使用 synthetic placeholder。
  - [x] 复用 `common.Marshal` / `common.Unmarshal`，不得直接调用 `encoding/json` marshal / unmarshal。

- [x] 补强 controller/service 行为验证 (AC: 1, 2)
  - [x] 扩展或确认 `controller/agentplatform/oauth_test.go` 覆盖完整 authorize -> token -> refresh -> revoke workflow，并验证 bad verifier、refresh token reuse、revoke 后刷新失败。
  - [x] 扩展或确认 `service/agentplatform/oauth_authorize_test.go` 覆盖 redirect URI allowlist、scope allowlist、PKCE S256、inactive / invalid integration client。
  - [x] 扩展或确认 `service/agentplatform/oauth_token_test.go` 覆盖 JWT access token TTL 300 秒、refresh token 30 天、refresh token hash 存储、refresh rotation、grant/token revoke、`client_credentials` 仅在 client 显式开启时可用、wrong verifier 失败。
  - [x] 如改动错误映射，更新 `writeOAuthError` 的测试或增加 controller 级断言，避免把 callback / scope / revoked grant 全部变成不可诊断的 generic failure。

- [x] 记录 consent、grant、client 状态与审计要求 (AC: 2)
  - [x] 在 contract spec 中说明当前 consent 记录事实源来自 `agent_platform_authorization_grants.consented_at` / grant 记录；当前没有完整用户可见 consent 页面，不得在 AP-6.1 中假装完成 UI。
  - [x] 明确 grant 状态至少区分 `authorized`、`issued`、`revoked`，refresh token 使用后轮换并撤销旧 token，access token 通过 `grant.token_version` 和 `revoked_at` 收敛。
  - [x] 明确 revoke 成功时应写入 `agentplatform.oauth.revoke` admin action；审计 payload 不得包含 bearer token、refresh token 明文、refresh token hash 或 provider secret。
  - [x] 明确 error / retryability：invalid request / bad PKCE / redirect mismatch / scope mismatch 为不可重试配置或请求错误；token expired 可由 refresh 重试；revoked grant 不可通过同一 token 重试；platform/internal failure 才可按通用策略重试。

- [x] 同步 AP-6 signoff 资产 (AC: 1, 2)
  - [x] 更新 `docs/agent-platform-consumer-signoff.md` 中 OAuth / token / revoke / callback / allowlist 行，引用本故事冻结结果、OpenAPI 路径、controller/service 测试和 conformance fixture。
  - [x] 如果 OAuth 域完成冻结，Cherry Studio first-consumer 的 OAuth 子域可以从 `ready for signoff` 推进到 `signed off`；但不要因 AP-6.1 完成而把 AP-6.5 model discovery 或 AP-6.6 error/client-state matrix 误标为 signed off。
  - [x] Codex second-consumer review 继续复用同一 auth plane；不得修改 `service/codex_*`、`controller/codex_*`、Codex channel affinity 或 relay channel 逻辑来完成本故事。

## Dev Notes

### Story Source and Scope

- Story 6.1 属于 Agent Platform Epic 6 “下游公共契约冻结与接入签核”。Epic 6 不是建设新 runtime plane，而是把 AP-1 到 AP-5 已完成能力冻结成下游可签核的公共契约。[Source: `_bmad-output/planning-artifacts/epics-agent-platform.md#Epic 6: 下游公共契约冻结与接入签核`; `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP6-AD-1`]
- 本故事只冻结 auth plane 的 wire contract：`/api/agent-platform/oauth/authorize`、`/api/agent-platform/oauth/token`、`/api/agent-platform/oauth/revoke`，以及 callback / redirect URI allowlist、PKCE、grant、refresh、revoke、consent 可见性。不要借此重写 discovery、Skill invoke、Knowledge query、model discovery、错误/状态矩阵或 web/default 全量 Clients 工作区。[Source: `_bmad-output/planning-artifacts/epics-agent-platform.md#Story 6.1: 冻结 OAuth、Token、Revoke 与 Callback Wire Contract`; `docs/agent-platform-downstream-contract-spec.md#11.1 OAuth / Token / Revoke Wire Contract`]
- 当前 authorize 实现返回 JSON authorization code，而不是最终完整浏览器 302 callback / consent page UX。AP-6.1 必须把这种关系讲清楚：当前冻结的是可用 wire capability 和契约边界，完整 operator-friendly OAuth 产品流不应被伪装成已完成。[Source: `docs/agent-platform-downstream-contract-spec.md#4.1 OAuth / 企业登录态`; `_bmad-output/planning-artifacts/ux-agent-platform.md#3.7 OAuth / consent 可见性`]

### Existing Runtime Behavior to Preserve

- `OAuthAuthorizeRequest` 读取 query/form 字段：`client_id`、`redirect_uri`、`scope`、`state`、`code_challenge`、`code_challenge_method`。成功响应字段为 `client_id`、`contract_version`、`scope`、`state`、`authorization_code`、`redirect_uri`、`consent_recorded`。[Source: `dto/agentplatform/oauth.go`; `controller/agentplatform/oauth.go`]
- `OAuthTokenRequest` 支持 `authorization_code`、`refresh_token`、`client_credentials`。空 `grant_type` 会被 normalize 为 `authorization_code`。成功响应字段为 `access_token`、`token_type`、`expires_in`、`refresh_token`、`refresh_expires_in`、`scope`、`contract_version`、`grant_id`。[Source: `dto/agentplatform/oauth.go`; `service/agentplatform/oauth_token.go#Exchange`]
- Access token 是 HS256 JWT，TTL 为 5 分钟，claims 包含 `client_id`、可选 `user_id`、`scope`、`contract_version`、`token_version`、`grant_id`，issuer 为 `new-api/agent-platform`。Refresh token 是 `rt_` 前缀 opaque token，存储为 HMAC hash，TTL 为 30 天，刷新时创建新 token 并撤销旧 token。[Source: `service/agentplatform/oauth_token.go#accessTokenTTL`; `#refreshTokenTTL`; `#generateJWTAccessToken`; `#exchangeRefreshToken`]
- Revoke 支持 refresh token 或 access token：refresh token revoke 会撤销 refresh token 和 grant；access token revoke 会撤销 grant 并 increment `token_version`，从而使旧 access token 校验失败。带 actor 时写入 `agentplatform.oauth.revoke` admin action。[Source: `service/agentplatform/oauth_token.go#Revoke`; `#revokeGrant`]
- Authorize 当前要求已登录 dashboard session 作为平台用户身份来源，但下游不能读取或复用 dashboard session；下游只接收 authorization code 和后续 bearer token。[Source: `controller/agentplatform/oauth.go#OAuthAuthorize`; `_bmad-output/planning-artifacts/architecture-agent-platform.md#Two auth planes`]

### Architecture Guardrails

- Auth plane、control plane、open capability plane 必须保持分离：control plane `/api/agent-platform/**` 使用 dashboard session；auth plane `/api/agent-platform/oauth/**` 处理授权和 token；open capability plane `/api/open-capabilities/**` 使用 Agent Platform bearer token。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#API & Communication Patterns`]
- Agent Platform bearer token 是 first-party token，不得复用 relay token、dashboard session、上游 OAuth provider token 或 `service/codex_*` token。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#Authentication & Security`]
- `client_credentials` 只能在 `agent_platform_clients.allow_client_credentials = true` 且 allowed grant types 包含 `client_credentials` 时启用；它不创建 end-user consent 语义。[Source: `service/agentplatform/oauth_token.go#exchangeClientCredentials`; `_bmad-output/planning-artifacts/architecture-agent-platform.md#Authorization flow design`]
- 所有公开契约变更必须同步三件套：`docs/agent-platform-downstream-contract-spec.md`、`docs/openapi/api.json`、`tests/agentplatform/conformance/`。三者不一致时 AP-6 story 不得 done。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP6-AD-3`; `docs/agent-platform-downstream-contract-spec.md#3A.1 Source of Truth`]
- 不修改 `relay/**`、`/v1/**`、`docs/openapi/relay.json`、`service/codex_*`、`controller/codex_*` 或 existing upstream login provider 逻辑。本故事中的 OAuth 是 Agent Platform downstream auth plane，不是仓库已有 OAuth login provider 或 Codex channel 维护任务。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP-6 Implementation Handoff`; `docs/project-overview.md`]

### Files to Read Before Editing

- `docs/agent-platform-downstream-contract-spec.md`：当前 4.1、5.1、6.2、6.3、11.1 已描述 OAuth 现状，但 11.1 仍是 `pending freeze`；实现应在这里收敛签核级契约，不要新增平行文档。[Source: `docs/agent-platform-downstream-contract-spec.md`]
- `docs/openapi/api.json`：当前已包含 `/api/agent-platform/oauth/authorize`、`/token`、`/revoke` 以及 `AgentPlatformOAuth*` schemas；实现必须确认 required fields 与 DTO 一致。[Source: `docs/openapi/api.json`]
- `dto/agentplatform/oauth.go`、`controller/agentplatform/oauth.go`、`service/agentplatform/oauth_authorize.go`、`service/agentplatform/oauth_token.go`：当前 runtime source of truth。
- `controller/agentplatform/oauth_test.go`、`service/agentplatform/oauth_authorize_test.go`、`service/agentplatform/oauth_token_test.go`：当前可复用测试基础。
- `tests/agentplatform/conformance/fixtures.go`、`tests/agentplatform/conformance/fixtures_test.go`：6.7 已建立的 fixture/conformance 入口，本故事应扩展它，而不是另建不可维护的 mock。
- `docs/agent-platform-consumer-signoff.md`：6.8 已建立签核 artifact；本故事完成后需要更新 OAuth 域证据与状态。

### Previous Story Intelligence

- 6.7 已建立 `tests/agentplatform/conformance/`，并在 review 中修复过 fixture method 漂移、OAuth revoke typed response 缺失、OpenAPI data schema 漂移。AP-6.1 必须继续用 DTO + OpenAPI + fixture 自动断言防漂移，不要只更新 markdown。[Source: `_bmad-output/implementation-artifacts/6-7-provide-mock-fixture-and-contract-conformance-suite.md#Senior Developer Review (AI)`]
- 6.8 已建立 `docs/agent-platform-consumer-signoff.md` 和 `TestConsumerSignoffArtifactAlignsWithContractSources`。AP-6.1 更新签核状态时必须保持 AP-6.5 / AP-6.6 blocker 不被误标为完成。[Source: `_bmad-output/implementation-artifacts/6-8-complete-cherry-studio-first-and-codex-second-signoff.md#Completion Notes List`; `tests/agentplatform/conformance/fixtures_test.go#TestConsumerSignoffArtifactAlignsWithContractSources`]
- 最近提交 `e8585fedb` 和 `33ad91e09` 分别改动了 OAuth/conformance/OpenAPI/signoff 文档。实现前读这些文件，避免覆盖刚修复的 AP-6 三件套。[Source: git log `e8585fedb`; `33ad91e09`]

### Testing Requirements

- 最小后端验证：
  - `GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance ./service/agentplatform ./controller/agentplatform`
- 如修改 OpenAPI、DTO、controller、service、router 或 conformance，运行受影响包测试；如触碰共享 auth/middleware/model 行为，扩大到：
  - `GOCACHE=/private/tmp/go-build-cache go test ./...`
- 已知历史全量回归风险：6.7 / 6.8 记录过 `go test ./...` 在未触碰包中存在既有失败，主要集中于 `model` 测试库缺表、`relay/channel/claude` 文件内容转换断言、`relay/helper` stream scanner status 断言。若全量仍失败，记录为既有无关失败，不要掩盖本故事目标包结果。[Source: `_bmad-output/implementation-artifacts/6-7-provide-mock-fixture-and-contract-conformance-suite.md#Debug Log References`; `_bmad-output/implementation-artifacts/6-8-complete-cherry-studio-first-and-codex-second-signoff.md#Completion Notes List`]
- 若新增 web/default OAuth / consent 可见性 UI，必须运行：
  - `cd web/default && bun test src/features/agent-platform/agent-platform.test.tsx`
  - `cd web/default && bun run typecheck`
  - `cd web/default && bun run i18n:sync`
  AP-6.1 默认不要求 UI 产品化，除非实现时决定补签核展示。

### Project Structure Notes

- Agent Platform 后端继续遵循 `Router -> Controller -> Service -> Model`，相关代码位于 `router/agentplatform-router.go`、`controller/agentplatform/`、`service/agentplatform/`、`model/agentplatform/`、`dto/agentplatform/`。[Source: `AGENTS.md#Architecture`; `_bmad-output/planning-artifacts/architecture-agent-platform.md#Complete Project Directory Structure`]
- 文档和签核资产属于 `docs/agent-platform-*`；conformance 属于 `tests/agentplatform/conformance/`；story/context 才属于 `_bmad-output/implementation-artifacts/`。不要把运行时或测试依赖放进 `_bmad-output`。
- JSON 业务代码必须使用 `common.Marshal` / `common.Unmarshal` / `common.DecodeJson` 等 wrapper；可以引用 `encoding/json.RawMessage` 类型，但不要直接调用标准库 marshal/unmarshal。[Source: `AGENTS.md#Rule 1: JSON Package`]
- 如果新增或修改 DB 模型 / 迁移，必须保持 SQLite、MySQL >= 5.7.8、PostgreSQL >= 9.6 兼容；优先使用 GORM，避免 DB-specific raw SQL。[Source: `AGENTS.md#Rule 2: Database Compatibility`]
- 不删除、不替换、不弱化受保护项目和组织标识，包括 `new-api` 与 `QuantumNous` 相关路径、metadata、版权、文档与包路径。[Source: `AGENTS.md#Rule 5: Protected Project Information`]

### References

- [Source: `_bmad-output/planning-artifacts/epics-agent-platform.md#Story 6.1: 冻结 OAuth、Token、Revoke 与 Callback Wire Contract`]
- [Source: `_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-5`; `#FR-6`; `#FR-13`; `#FR-14`; `#FR-15`]
- [Source: `_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#5. AP-6 的具体定位`; `#5.3 AP-6 MVP Exit Gate`]
- [Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#Authentication & Security`; `#API & Communication Patterns`; `#AP-6 Architectural Decisions`; `#AP-6 Implementation Handoff`]
- [Source: `_bmad-output/planning-artifacts/ux-agent-platform.md#3.7 OAuth / consent 可见性`; `#3.8 下游契约签核视图`]
- [Source: `docs/agent-platform-downstream-contract-spec.md#4.1 OAuth / 企业登录态`; `#6.2 OAuth Authorize Query Contract`; `#6.3 OAuth Token Exchange`; `#11.1 OAuth / Token / Revoke Wire Contract`]
- [Source: `dto/agentplatform/oauth.go`; `controller/agentplatform/oauth.go`; `service/agentplatform/oauth_authorize.go`; `service/agentplatform/oauth_token.go`]
- [Source: `controller/agentplatform/oauth_test.go`; `service/agentplatform/oauth_authorize_test.go`; `service/agentplatform/oauth_token_test.go`; `tests/agentplatform/conformance/fixtures.go`; `tests/agentplatform/conformance/fixtures_test.go`]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-03 17:44 +0800：执行 BMAD create-story workflow；已读取用户指定 `.agents/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`。
- 2026-06-03 17:44 +0800：`resolve_customization.py` 因当前 `python3` 缺少 `tomllib` 失败；已按技能 fallback 手动读取 `.agents/skills/bmad-create-story/customize.toml`，未发现 `_bmad/custom/bmad-create-story*.toml` override。
- 2026-06-03 17:44 +0800：已读取 `_bmad/bmm/config.yaml`，确认 user 为 `hth`，communication/document language 为 Chinese，planning/implementation artifacts 均位于 `_bmad-output`。
- 2026-06-03 17:44 +0800：已读取完整 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认 `ap-6-1-freeze-oauth-token-revoke-and-callback-wire-contract: backlog`，并发现既有 6.7 / 6.8 文件名与 sprint key 的 `ap-6-*` 前缀不一致。
- 2026-06-03 17:44 +0800：已分析 Agent Platform epic、PRD、addendum、architecture、UX、downstream contract spec、OAuth DTO/controller/service/tests、OpenAPI、conformance suite、6.7 / 6.8 previous story artifacts 和最近相关 git commits。
- 2026-06-03 17:50 +0800：执行 BMAD dev-story workflow；读取 `.agents/skills/bmad-dev-story/SKILL.md`、`checklist.md`、`_bmad/bmm/config.yaml`、完整 story 与 sprint-status。`project-context.md` 未发现；workflow customization 无 prepend/append。
- 2026-06-03 17:50 +0800：按 TDD 先扩展 `tests/agentplatform/conformance/fixtures_test.go`，确认 `go test ./tests/agentplatform/conformance` 因缺少 `oauth_refresh_rotation_success` 等 AP-6.1 OAuth fixtures 失败。
- 2026-06-03 17:52 +0800：目标验证通过：`GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance ./service/agentplatform ./controller/agentplatform`。
- 2026-06-03 17:52 +0800：全量回归 `GOCACHE=/private/tmp/go-build-cache go test ./...` 失败于既有无关问题：根包缺 `web/classic/dist`、`model` 测试库缺表、`relay/channel/claude` 文件内容转换断言、`relay/helper` stream scanner status 断言；目标 AP-6.1 包均通过。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- Validation checklist applied manually：story 包含 BDD AC、任务分解、文件位置、现有 runtime 行为、架构约束、前置/后续 story intelligence、测试入口、禁止范围、安全约束与三件套防漂移要求。
- 本 story artifact 明确 AP-6.1 默认是 contract-freeze / conformance / documentation story，不要求重写 auth plane 或产品化完整 browser consent UI。
- `docs/agent-platform-downstream-contract-spec.md#11.1` 已从 `pending freeze` 推进为 `frozen for signoff`，补齐 authorize/token/refresh/client_credentials/revoke 字段等级、JSON authorize 与未来 browser redirect UX 边界、allowlist、consent/grant/client 状态、审计和 retryability 语义。
- 未重写 Go auth plane runtime；本故事冻结既有实现并通过 OpenAPI/conformance/tests 防漂移。新增 runtime 测试只补强 inactive / invalid integration client 与 plain PKCE 拒绝断言。
- OAuth fixture 覆盖已扩展到 authorize success、token success、refresh rotation、revoke success、expired/revoked grant、missing scope、redirect mismatch、invalid PKCE、invalid client / invalid integration；fixture 使用 synthetic placeholder，未加入真实 token/secret/user/tenant 数据。
- `docs/openapi/api.json` 已对齐 DTO 与 current runtime：authorize response `state` required，token request `grant_type` optional，token response `refresh_token`、`refresh_expires_in`、`grant_id` required。
- `docs/agent-platform-consumer-signoff.md` 已将 Cherry Studio OAuth 子域推进为 `signed off`，但 Cherry Studio overall first-consumer 仍保持 `blocked`，AP-6.5 model discovery 与 AP-6.6 error/client-state matrix 未误标完成；未修改 Codex channel、relay channel affinity 或 `service/codex_*` / `controller/codex_*`。
- Definition of Done checklist 手动通过：任务均完成，AC 1/2 覆盖，目标包测试通过，文件列表更新；全量 `go test ./...` 存在已记录的无关既有失败。

### File List

- `_bmad-output/implementation-artifacts/ap-6-1-freeze-oauth-token-revoke-and-callback-wire-contract.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `docs/agent-platform-downstream-contract-spec.md`
- `docs/agent-platform-consumer-signoff.md`
- `docs/openapi/api.json`
- `service/agentplatform/oauth_authorize_test.go`
- `tests/agentplatform/conformance/fixtures.go`
- `tests/agentplatform/conformance/fixtures_test.go`

## Change Log

- 2026-06-03：创建 Story 6.1 context artifact，状态设为 `ready-for-dev`。
- 2026-06-03：实现 AP-6.1 OAuth/token/revoke/callback/allowlist contract freeze，扩展 conformance fixture、防漂移测试、OpenAPI schema freeze 与 consumer signoff OAuth 子域签核状态；story 状态设为 `done`。
