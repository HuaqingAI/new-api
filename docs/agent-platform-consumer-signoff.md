# Agent Platform Consumer Signoff

Status: blocked
contract_version: 2026-06
owner_story: ap-6-8-complete-cherry-studio-first-and-codex-second-signoff
updated_at: 2026-06-03

## 1. Scope

本文记录 AP-6 公共契约的 consumer signoff 结果，用于验证 Cherry Studio first consumer 与 Codex second consumer 是否能复用同一条 Agent Platform 下游公共契约。

公共契约 source of truth 仍是三件套：

- `docs/agent-platform-downstream-contract-spec.md`
- `docs/openapi/api.json`
- `tests/agentplatform/conformance/`，并包含 runtime handler 证据 `controller/agentplatform/open_capabilities_test.go`
- OAuth runtime handler 与 allowlist 证据：`controller/agentplatform/oauth_test.go`、`service/agentplatform/oauth_authorize_test.go`、`service/agentplatform/oauth_token_test.go`

本签核 artifact 不新增新的业务主干、DTO、controller、service、router 或数据库对象。AP-6.1 对 `docs/openapi/api.json` 做 OAuth schema freeze；AP-6.2 对现有 Clients API 做 schema freeze：补齐 `GET/POST /api/agent-platform/clients` 与 `GET/PUT /api/agent-platform/clients/{id}` 的 OpenAPI path/schema，使其对齐当前 DTO 与 runtime。AP-6.5 model discovery 与 AP-6.6 error/client state matrix 尚未冻结，不能通过提前写入 OpenAPI 伪造签核完成。

## 2. Signoff Status Rules

本 artifact 只使用以下状态值：

- `draft`
- `ready for signoff`
- `signed off`
- `blocked`

当前总体状态为 `blocked`，原因是 AP-6.5 enterprise model discovery contract 与 AP-6.6 error/client state matrix 仍未冻结。已冻结或已有实现证据的域可以单独记录为 `signed off` 或 `ready for signoff`，但 Cherry Studio 整体 first-consumer signoff 不能标记为 `signed off`。

## 3. Cherry Studio First-Consumer Signoff

### 3.1 Result

Status: blocked

Cherry Studio 可以基于当前 AP-6 公共契约接入 OAuth、resource discovery/detail/refresh、Skill invoke 与 Knowledge query 的基础闭环。OAuth 子域已完成 AP-6.1 wire contract freeze，可以标记为 `signed off`。Cherry Studio 的完整 first-consumer signoff 仍被 AP-6.5 与 AP-6.6 阻塞，因此总体不得标记为 `signed off`。

Client registration / onboarding 已完成 AP-6.2 client registration schema freeze。Cherry Studio 可以复用现有 `client_id`、`slug`、`client_type`、`allowed_grant_types`、`redirect_uris`、`allowed_scopes`、`contract_version`、`capabilities`、`extensions`、`allow_client_credentials` 字段；无效配置必须落到 `invalid_integration`、`disabled` 或 OAuth allowlist/scope/callback 明确失败，不得只呈现 generic failure。

Cherry Studio 私有展示字段只能进入 `extensions.cherry_studio` namespace。对 client registration，具体顶层 key 使用 `extensions.cherry_studio.*` 形式的点分 namespaced key。当前 fixture 已使用 `extensions.cherry_studio` 验证资源 extension 路径，client registration runtime 使用点分 namespaced key 验证，且没有新增 Cherry Studio 专属核心字段、特殊 endpoint 或特殊资源模型。

### 3.2 Coverage Matrix

| Domain | Status | Evidence | Signoff notes |
| --- | --- | --- | --- |
| Client registration / onboarding | signed off | `docs/agent-platform-downstream-contract-spec.md`; `docs/openapi/api.json`; `service/agentplatform/client_test.go`; `controller/agentplatform/client_test.go`; `tests/agentplatform/conformance/fixtures_test.go` | AP-6.2 client registration schema freeze 已覆盖字段等级、示例 payload、OpenAPI path/schema、`invalid_integration` 状态、namespaced extension 拒绝、`allow_client_credentials` 显式布尔字段；完整 Clients 自助工作区仍属于后续 UI productization。 |
| OAuth authorize/token/revoke/callback/allowlist | signed off | `docs/openapi/api.json`; `tests/agentplatform/conformance/fixtures.go`; `controller/agentplatform/oauth_test.go`; `service/agentplatform/oauth_authorize_test.go`; `service/agentplatform/oauth_token_test.go` | JSON authorize response 是当前 wire contract；最终浏览器 redirect UX 属于后续产品化层，不能改变 authorize/token/revoke 的核心 JSON contract。Callback 与 redirect URI 严格匹配 registered client allowlist；fixture 覆盖 authorize success、token success、refresh rotation、revoke success、expired/revoked grant、missing scope、redirect mismatch、invalid PKCE、invalid client / invalid integration。 |
| Resource discovery/detail/refresh | signed off | `docs/openapi/api.json`; `tests/agentplatform/conformance/fixtures.go`; `controller/agentplatform/open_capabilities_test.go` | 覆盖 `resource_id`、`resource_type`、`display_name`、`resource_version`、`contract_version`、TTL、freshness、ETag、visibility/callable、diagnostics、revoked/offline/stale 收敛规则。 |
| Skill invoke | signed off | `tests/agentplatform/conformance/fixtures.go`; `controller/agentplatform/open_capabilities_test.go` | 覆盖 sync invoke success、timeout、upstream failure、contract invalid；provider-native 字段不得外泄。 |
| Knowledge query | signed off | `tests/agentplatform/conformance/fixtures.go`; `controller/agentplatform/open_capabilities_test.go` | 覆盖 standardized items/citations、provider offline、upstream failure；provider-native 字段不得外泄。 |
| Enterprise model discovery | blocked | `tests/agentplatform/conformance/fixtures.go` pending fixtures | AP-6.5 未冻结；不得复用 `/api/models`、`/api/user/models` 或 `/v1/models` 伪造默认模型、模型不可用、account/tenant source 语义。 |
| Error matrix and client state matrix | blocked | `tests/agentplatform/conformance/fixtures.go` pending fixture | AP-6.6 未冻结；不得把 pending fixture 当成完整客户端状态签核。 |
| Mock fixture and conformance | ready for signoff | `tests/agentplatform/conformance/`; `controller/agentplatform/open_capabilities_test.go` | 覆盖 OAuth、open capability、Skill、Knowledge、error envelope，并对 AP-6.5 / AP-6.6 缺口保留 explicit pending reason。 |

### 3.3 Local Verification

本地验证命令：

```bash
GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance ./service/agentplatform ./controller/agentplatform
```

`TestConsumerSignoffArtifactAlignsWithContractSources` 与 `TestConsumerSignoffCoverageMatrixMatchesFixtures` 用于发现 consumer signoff artifact、contract spec、OpenAPI 路径和 fixture/conformance 状态的明显漂移。只要 AP-6.5 / AP-6.6 仍是 pending fixture，Cherry Studio 整体状态就必须保持 `blocked`。

`./service/agentplatform` 与 `./controller/agentplatform` 包测试覆盖 Clients registration/onboarding、OAuth authorize/token/revoke、PKCE、redirect URI allowlist、scope allowlist、refresh rotation、grant revoke 与 bearer error convergence。`./tests/agentplatform/conformance` 负责把这些 runtime 证据索引到签核 artifact 与公共契约三件套。

## 4. Codex Second-Consumer Review

### 4.1 Result

Status: signed off

Codex second-consumer review 的范围是 AP-6 下游公共契约复用评审，不是仓库已有 Codex channel、Codex OAuth upstream、credential refresh 或 relay channel affinity 维护任务。

Codex 可以复用同一份 AP-6 公共契约中的：

- `client`
- AP-6.2 client registration schema freeze
- `contract_version`
- `capabilities`
- open capability resource model
- standard error envelope
- `docs/openapi/api.json`
- `tests/agentplatform/conformance/` fixture/conformance 资产

结论：Codex 不需要新增平行资源主模型、平行核心协议主干或独立 open capability endpoint；no parallel core protocol is required.

### 4.2 Extension Governance

Codex 如需额外字段，只允许使用以下治理路径：

- `extensions.codex`
- 向后兼容 optional field
- 提升 `contract_version`

Codex 私有字段不得改写核心字段语义，不得新增平行核心协议主干。Codex second-consumer 仍复用同一 registration schema，不新增平行 client object model。

## 5. UI Boundary

AP-6.2 不交付完整 `web/default` Clients 产品化工作区。当前 onboarding 最小闭环由 API / fixture 支撑：registration、redirect/callback、allowed scopes、contract version、capabilities、OAuth prerequisites 和 mock fixture 都可被文档、OpenAPI、runtime tests 与 conformance 解释。后续产品化可在 `web/default/src/features/agent-platform/` 增加 Clients 工作区和 signoff 状态视图，但该 UI 必须直接表达 `draft`、`ready for signoff`、`signed off`、`blocked`，并展示当前 `contract_version` 与核心契约域覆盖情况，不得暗示当前已具备完整自助 onboarding UI。

## 6. Final Conclusion

- Cherry Studio first-consumer signoff: `blocked`
- Client registration / onboarding signoff: `signed off`
- Cherry Studio OAuth subdomain signoff: `signed off`
- Codex second-consumer review: `signed off`
- Extension namespaces used or reserved: `extensions.cherry_studio`, `extensions.codex`
- OpenAPI change: OAuth schema freeze plus AP-6.2 existing Clients API schema freeze
- Protocol conclusion: no parallel core protocol, no parallel resource model, and no independent open capability endpoint are required
