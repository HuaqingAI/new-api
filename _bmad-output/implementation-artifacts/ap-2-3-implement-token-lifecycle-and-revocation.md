# Story 2.3: 实现 token 生命周期与撤销机制

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 下游集成负责人,
I want 平台对已发放 token 提供可审计、可撤销、可轮换的生命周期治理,
so that 消费者能安全访问开放能力面，而不用发明自定义授权流程。

## Acceptance Criteria

1. **Given** 某受信任的机器型消费者被显式允许  
   **When** 它申请 token 访问  
   **Then** 平台可以通过可选的 client credentials 路径授予访问  
   **And** 是否开启由每个 client 单独控制，而非全局默认。
2. **Given** 平台发放 refresh token  
   **When** refresh token 被存储或轮换  
   **Then** refresh token 必须以哈希形式存储并支持单独撤销  
   **And** access token 采用短期 JWT，并结合当前 token version 校验。

## Tasks / Subtasks

- [x] 新增 refresh token 事实表与哈希存储模型 (AC: 2)
  - [x] 新增 `model/agentplatform/refresh_token.go`，定义 `agent_platform_refresh_tokens` 表，至少覆盖：`grant_id`、`refresh_token_hash`、`expires_at`、`rotated_from_id`、`revoked_at`、`last_used_at`、`created_at`、`updated_at`。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#OAuth / grant model]
  - [x] 在 `model/agentplatform/migration.go` 注册迁移，保持 SQLite / MySQL / PostgreSQL 同时兼容。[Source: AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]
  - [x] refresh token 不得明文落库；哈希必须走项目公共加密封装，而不是散落的自定义实现。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Secret handling; common/crypto.go]

- [x] 扩展 OAuth token 服务以覆盖 access/refresh/client credentials 生命周期 (AC: 1, 2)
  - [x] 将 `service/agentplatform/oauth_token.go` 从“最小 code exchange”升级为统一 token exchange 服务，支持 `authorization_code`、`refresh_token`、`client_credentials` 三条路径。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#440-487]
  - [x] access token 改为第一方签发的短期 JWT，claims 至少包含：`iss`、`sub`、`aud`、`exp`、`jti`、`client_id`、可选 `user_id`、`scope`、`contract_version`、`token_version`、`grant_id`。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#456-470]
  - [x] refresh token 使用时必须轮换：旧 token 标记为已撤销/已使用，新 token 哈希入库；任何 revoke 都要让后续 refresh 失败。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#484-487]
  - [x] `client_credentials` 只允许在 client 记录显式开启 `allow_client_credentials=true` 且 grant type 白名单包含 `client_credentials` 时发放。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#450; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.3]

- [x] 提供 revoke 与 token version 校验入口 (AC: 2)
  - [x] 新增 `/api/agent-platform/oauth/revoke`，至少支持 refresh token 与 access token 对应 grant 的撤销。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#563]
  - [x] grant/token revoke 时要更新 `authorization_grant.token_version` 或 grant 状态，使既有 JWT 在后续校验时失效，而不是只删除 refresh token。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#479-487]
  - [x] 保留最小 `ValidateAccessToken` 能力供后续 open-capabilities bearer auth 复用，但不要在本故事里提前实现整套 `/api/open-capabilities/**` 数据面。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.4]

- [x] 修正 OAuth auth plane 路由边界并补定向测试 (AC: 1, 2)
  - [x] `router/agentplatform-router.go` 中 `/api/agent-platform/oauth/**` 不能继续挂在 `AdminAuth()` 下；authorize 可基于已有登录 session，token/revoke 走 auth plane 语义。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#API & Communication Patterns]
  - [x] 更新 `service/agentplatform/oauth_token_test.go`、`controller/agentplatform/oauth_test.go`，覆盖：JWT access token 发放、refresh token 哈希存储与轮换、revoke 失效、client credentials 按 client 开关放行。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.3]

- [x] 保持 2.3 边界，不提前做 2.4/2.5 的数据面工作 (AC: 1, 2)
  - [x] 本故事不实现 `/api/open-capabilities/**` discovery/detail/invoke/refresh 正式接口。
  - [x] 本故事不引入 `client_instance` 主模型，也不扩展设备级 revoke。
  - [x] 本故事只补最小 openapi 记录，避免提前定义 2.4 才冻结的标准 envelope 细节。

## Dev Notes

### Previous Story Insights

- `2.1` 已经提供 `allow_client_credentials`、allowed grant types、allowed scopes、contract version 与 capabilities 事实源，因此 `2.3` 不需要重建 client 主模型。[Source: _bmad-output/implementation-artifacts/ap-2-1-register-clients-and-capability-declarations.md]
- `2.2` 已经落地 authorization code + PKCE 与 grant 事实表，因此 `2.3` 应该在现有 `authorization_grant` 基础上继续扩展 token lifecycle，而不是再造第二套授权记录。[Source: _bmad-output/implementation-artifacts/ap-2-2-implement-browser-delegated-auth-main-flow.md]

### Data Models

- `agent_platform_authorization_grants` 已存在：`grant_id`、`client_id`、可空 `user_id`、`grant_type`、`scope_text`、`status`、`token_version`、`contract_version`、`consented_at`、`revoked_at`。[Source: model/agentplatform/authorization_grant.go]
- 本故事新增 `agent_platform_refresh_tokens`，使用 `TEXT/varchar` + 时间字段，不依赖数据库专属 JSON/二进制类型。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#OAuth / grant model; AGENTS.md#Rule 2]

### API Notes

- Auth plane 路由固定在 `/api/agent-platform/oauth/**`：
  - `GET /authorize`
  - `POST /token`
  - `POST /revoke`
- 2.3 可以扩充 token/revoke 请求响应字段，但不要把 2.4 的标准错误 envelope 设计提前锁死在这里。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#520-570]

### Testing

- 推荐至少运行：
  - `GOCACHE=/private/tmp/go-build-cache go test ./model/agentplatform ./service/agentplatform ./controller/agentplatform`
- 验证重点：
  - refresh token 落库为哈希而非明文
  - refresh 轮换后旧 token 失效
  - revoke 后 access token 校验失败
  - `client_credentials` 仅在 client 显式启用时通过

### Project Structure Notes

- 预计新增文件：
  - `model/agentplatform/refresh_token.go`
  - 对应 `*_test.go`（如需要）
- 预计修改文件：
  - `service/agentplatform/oauth_token.go`
  - `controller/agentplatform/oauth.go`
  - `dto/agentplatform/oauth.go`
  - `router/agentplatform-router.go`
  - `docs/openapi/api.json`
- 保持不改或只读依赖：
  - `relay/**`
  - `web/default/**`
  - `oauth/**`
  - `model/agentplatform/client_instance*.go`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.3: 实现 token 生命周期与撤销机制]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#440-487]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#520-570]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#972-1002]
- [Source: _bmad-output/implementation-artifacts/ap-2-1-register-clients-and-capability-declarations.md]
- [Source: _bmad-output/implementation-artifacts/ap-2-2-implement-browser-delegated-auth-main-flow.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 继续作为总编排状态源，但 `2.3` 采用 manual create/dev takeover，避免 create-story / dev-story 子进程在 Agent Platform Epic 2 上继续漂移。
- 已加载 `epics-agent-platform.md`、`architecture-agent-platform.md`、`prd-agent-platform-2026-05-31/prd.md`，并基于 `2.1/2.2` 已有 client + delegated auth 实现继续扩展 token lifecycle。

### Completion Notes List

- 已新增 `agent_platform_refresh_tokens` 表，并使用 `common.GenerateHMAC` 存储 refresh token 哈希而非明文。
- 已将 OAuth token exchange 升级为统一生命周期服务，支持 `authorization_code`、`refresh_token` 轮换和按 client 开关控制的 `client_credentials`。
- 已将 access token 切换为短期 JWT，并提供最小 `ValidateAccessToken` 校验当前 `grant.token_version` / revoke 状态的能力。
- 已新增 `/api/agent-platform/oauth/revoke`，并修正 OAuth auth plane 路由不再挂在 `AdminAuth()` 下。
- 已完成并通过 `GOCACHE=/private/tmp/go-build-cache go test ./model/agentplatform ./service/agentplatform ./controller/agentplatform`。

### File List

- `_bmad-output/implementation-artifacts/ap-2-3-implement-token-lifecycle-and-revocation.md`
- `model/agentplatform/refresh_token.go`
- `model/agentplatform/migration.go`
- `service/agentplatform/oauth_token.go`
- `service/agentplatform/oauth_token_test.go`
- `controller/agentplatform/oauth.go`
- `controller/agentplatform/oauth_test.go`
- `dto/agentplatform/oauth.go`
- `router/agentplatform-router.go`

## Change Log

- 2026-06-01: 完成 refresh token 持久化、JWT access token、refresh rotation、revoke 路径、OAuth auth plane 路由修正与定向测试，并将故事推进为 done。
