# Story 2.2: 实现浏览器委托授权主路径

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 下游集成负责人,
I want 平台先交付最小可用的浏览器委托授权路径,
so that 消费者能安全访问开放能力面，而不用发明自定义授权流程。

## Acceptance Criteria

1. **Given** 某浏览器型委托消费者被授权  
   **When** 它执行 MVP 下游授权流程  
   **Then** 平台支持 authorization code + PKCE  
   **And** 不要求消费者使用 MVP 未冻结的 grant type。
2. **Given** 某最终用户尚未登录或尚未 consent  
   **When** 浏览器委托流程开始  
   **Then** 平台能够串联既有登录态与 consent 记录  
   **And** 审计链能关联 user、client、scope 与 grant 结果。

## Tasks / Subtasks

- [x] 在 Agent Platform bounded context 中落地最小 delegated-auth 主模型 (AC: 1, 2)
  - [x] 新增 `model/agentplatform/authorization_grant.go`（或等价命名），定义 `agent_platform_authorization_grants` 表，至少覆盖：`grant_id`、`client_id`、可空 `user_id`、`grant_type`、`scope_text`、`status`、`token_version`、`contract_version`、`consented_at`、`revoked_at`、`created_at`、`updated_at`。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#350-362]
  - [x] 新增最小 consent 事实源（可复用 grant 表字段，也可加单独 consent 记录），确保浏览器委托链路能明确表达“用户已经同意哪些 scope 给哪个 client”。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#444-447; _bmad-output/planning-artifacts/architecture-agent-platform.md#990-992]
  - [x] 当前故事不要求 refresh token 哈希存储或 token rotation 细节，那属于 2.3；这里的目标是把 delegated-auth 主路径的 grant/consent 事实源先建好。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.3]

- [x] 实现 `/api/agent-platform/oauth/authorize` 浏览器授权入口 (AC: 1, 2)
  - [x] 扩展 `controller/agentplatform`、`service/agentplatform` 和 `router/agentplatform-router.go`（或 `router/api-router.go` 下专门的 OAuth 子路由注册），提供 `/api/agent-platform/oauth/authorize` 的最小实现。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#561-564; _bmad-output/planning-artifacts/architecture-agent-platform.md#444-447]
  - [x] 该入口必须校验：`client_id` 存在且有效、grant type 包含 `authorization_code`、PKCE 参数齐全、requested scopes 在 client 允许范围内；不满足时返回可诊断错误，而不是静默回退。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#345-348; _bmad-output/planning-artifacts/ux-agent-platform.md#3-1-客户端注册]
  - [x] 若用户未登录，复用现有 `new-api` 登录/session 体系完成登录前置，不在 2.2 新造第二套登录系统。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#444-446; router/api-router.go; controller/oauth.go]

- [x] 实现最小 consent + code exchange 主路径 (AC: 1, 2)
  - [x] 在授权页/服务逻辑中记录 consent 结果，并生成短生命周期 authorization code；authorization code 至少要绑定：client、user、scope、PKCE challenge / method、contract_version、过期时间。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#990-1002]
  - [x] 实现 `/api/agent-platform/oauth/token` 的最小 code exchange 路径，校验 authorization code + PKCE verifier，并返回后续数据面可用的最小 token 响应结构；此故事可以先返回短期 access token，refresh token 细节放到 2.3。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#447-450; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.3]
  - [x] 当前故事不实现 `client_credentials` 真正发放逻辑，只保留后续 2.3/2.4 所需的模型与边界约束。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#450]

- [x] 保持 2.2 的范围，不提前做 2.3/2.4 的事情 (AC: 1, 2)
  - [x] 本故事不实现 refresh token 哈希存储/轮换、token version revoke cache、open-capabilities discovery/detail/invoke、client instance、或 UI 完整 consent 页面体验。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.3; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.4]
  - [x] 也不增加非 MVP grant type；仍然只聚焦 browser delegated auth 的 authorization code + PKCE 主路径。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#440]

- [x] 补齐 model/service/controller 测试与最小管理面合同 (AC: 1, 2)
  - [x] 新增 `model/agentplatform/*_test.go`，覆盖 grant/consent 记录、authorization code 约束字段、过期时间和用户/client 绑定关系。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#350-362]
  - [x] 新增 `service/agentplatform/*_test.go`，覆盖：PKCE 校验、未登录/未 consent 拒绝、scope 超出 client 允许范围拒绝、已禁用 client 拒绝。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#UJ-1; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#UJ-3]
  - [x] 新增 `controller/agentplatform/*_test.go`，覆盖 authorize/token 成功路径和错误输入；若 OpenAPI 管理面需要暴露这些路由，同步更新 `docs/openapi/api.json`。[Source: docs/openapi/api.json]

## Dev Notes

- `2.2` 的目标不是完整 OAuth 子系统，而是把“浏览器委托授权主路径”打通。也就是说：client 已存在、用户已有登录体系、scope 已有声明，当前故事只负责把它们串成最小的 `authorize -> consent -> code -> token` 主链。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#444-447]
- 现有仓库已经有 OAuth 登录相关的 controller/service 和 PKCE 生成经验（如 Codex OAuth），这些可以复用为实现参考，但 Agent Platform bearer auth 仍是新的第一方授权平面，不能直接复用旧的登录回调语义。[Source: controller/oauth.go; service/codex_oauth.go; _bmad-output/planning-artifacts/architecture-agent-platform.md#106-109]
- 2.2 还是 control/auth plane 的故事，不需要去碰 `/api/open-capabilities/**` 数据面；后者要等 2.4 以后再接 discovery/detail/invoke。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.4]
- PKCE、client 状态、scope 范围、contract version 都应该在服务层做明确校验，而不是让 controller 层或前端自己猜测授权失败原因。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#129]
- 仍然遵守跨库和 JSON wrapper 约束：grant/consent/code 相关的结构化字段如果需要 JSON，一律走 `TEXT` + `common.*` 路线；不要引入数据库专属 JSON 类型。[Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go; AGENTS.md#Rule 2]

### Project Structure Notes

- 预计新增文件：
  - `model/agentplatform/authorization_grant.go`
  - `service/agentplatform/oauth_authorize.go`
  - `service/agentplatform/oauth_token.go`
  - `controller/agentplatform/oauth.go`
  - `dto/agentplatform/oauth.go`
  - 对应 `*_test.go`
- 预计修改文件：
  - `model/agentplatform/migration.go`
  - `router/agentplatform-router.go` 或 `router/api-router.go`
  - `docs/openapi/api.json`
- 保持不改或只读依赖：
  - `relay/**`
  - `docs/openapi/relay.json`
  - `web/default/**`
  - `model/agentplatform/refresh_token*.go`（留给 2.3）
  - `model/agentplatform/client_instance*.go`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.2: 实现浏览器委托授权主路径]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#341-379]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#440-450]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#990-1002]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#UJ-1]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#UJ-3]
- [Source: controller/oauth.go]
- [Source: service/codex_oauth.go]
- [Source: _bmad-output/implementation-artifacts/ap-2-1-register-clients-and-capability-declarations.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 对 Agent Platform 的 `2.2 -> ap-2-2-*` key 映射沿用已修正逻辑；本 story 文件由 manual create-story takeover 补齐，用于避免 create-story 在 Epic 2 范围内空转。
- 已加载 `epics-agent-platform.md`、`architecture-agent-platform.md`、`prd-agent-platform-2026-05-31/prd.md`，并参考现有 `controller/oauth.go`、`service/codex_oauth.go` 作为授权主路径实现参照。
- 当前仓库在 `2.1` 后已经具备 client registration 主模型，因此 `2.2` 的重点是 delegated auth 主链，而不是重新做 client 配置或 token 生命周期全套。

### Completion Notes List

- 已落地最小 delegated-auth 主路径：`/api/agent-platform/oauth/authorize` 与 `/api/agent-platform/oauth/token`。
- 已实现 authorization code + PKCE 校验、consent/grant 记录和最小 token exchange 返回结构。
- 已明确禁止在本故事里提前实现 refresh token 全套、client credentials 真正发放和 open-capabilities 数据面。
- 已完成 OAuth authorize/token 相关 model/service/controller 定向测试，并通过 `GOCACHE=/private/tmp/go-build-cache go test ./model/agentplatform ./service/agentplatform ./controller/agentplatform` 验证。

### File List

- `_bmad-output/implementation-artifacts/ap-2-2-implement-browser-delegated-auth-main-flow.md`
- `model/agentplatform/authorization_grant.go`
- `service/agentplatform/oauth_authorize.go`
- `service/agentplatform/oauth_authorize_test.go`
- `service/agentplatform/oauth_token.go`
- `service/agentplatform/oauth_token_test.go`
- `controller/agentplatform/oauth.go`
- `controller/agentplatform/oauth_test.go`
- `dto/agentplatform/oauth.go`
- `model/agentplatform/migration.go`
- `router/agentplatform-router.go`


## Change Log

- 2026-06-01: 完成浏览器委托授权主路径、authorization code + PKCE、consent/grant 记录与定向测试，并将故事推进为 done。
