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

本签核 artifact 不新增公开 API、DTO、controller、service、router 或数据库对象，因此 `docs/openapi/api.json` 无需变更。OpenAPI 的职责仍是描述已冻结的 OAuth 与 open capability wire surface；AP-6.5 model discovery 与 AP-6.6 error/client state matrix 尚未冻结，不能通过提前写入 OpenAPI 伪造签核完成。

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

Cherry Studio 可以基于当前 AP-6 公共契约接入 OAuth、resource discovery/detail/refresh、Skill invoke 与 Knowledge query 的基础闭环。Cherry Studio 的完整 first-consumer signoff 被 AP-6.5 与 AP-6.6 阻塞，因此不得标记为 `signed off`。

Cherry Studio 私有展示字段只能进入 `extensions.cherry_studio`。当前 fixture 已使用 `extensions.cherry_studio` 验证 namespaced extension 路径，且没有新增 Cherry Studio 专属核心字段、特殊 endpoint 或特殊资源模型。

### 3.2 Coverage Matrix

| Domain | Status | Evidence | Signoff notes |
| --- | --- | --- | --- |
| OAuth authorize/token/revoke/callback/allowlist | ready for signoff | `docs/openapi/api.json`; `tests/agentplatform/conformance/fixtures.go`; `controller/agentplatform/oauth_test.go`; `service/agentplatform/oauth_authorize_test.go`; `service/agentplatform/oauth_token_test.go` | JSON authorize response 是当前 wire contract；最终浏览器 redirect UX 属于后续产品化层，不能改变 authorize/token/revoke 的核心 JSON contract。Callback 与 redirect URI 校验沿用 registered client allowlist。 |
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

本故事新增 `TestConsumerSignoffArtifactAlignsWithContractSources`，用于发现 consumer signoff artifact、contract spec、OpenAPI 路径和 fixture/conformance 状态的明显漂移。只要 AP-6.5 / AP-6.6 仍是 pending fixture，Cherry Studio 整体状态就必须保持 `blocked`。

`./service/agentplatform` 与 `./controller/agentplatform` 包测试覆盖 OAuth authorize/token/revoke、PKCE、redirect URI allowlist、scope allowlist、refresh rotation、grant revoke 与 bearer error convergence。`./tests/agentplatform/conformance` 负责把这些 runtime 证据索引到签核 artifact 与公共契约三件套。

## 4. Codex Second-Consumer Review

### 4.1 Result

Status: signed off

Codex second-consumer review 的范围是 AP-6 下游公共契约复用评审，不是仓库已有 Codex channel、Codex OAuth upstream、credential refresh 或 relay channel affinity 维护任务。

Codex 可以复用同一份 AP-6 公共契约中的：

- `client`
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

Codex 私有字段不得改写核心字段语义，不得新增平行核心协议主干。

## 5. UI Boundary

本故事只交付文档签核 artifact 与可重复防漂移验证，不新增 `web/default` UI。等价展示路径是本 artifact 的 status、coverage matrix 与 blocker 列表；后续产品化可在 `web/default/src/features/agent-platform/` 增加 signoff 状态视图，但该 UI 必须直接表达 `draft`、`ready for signoff`、`signed off`、`blocked`，并展示当前 `contract_version` 与核心契约域覆盖情况。

## 6. Final Conclusion

- Cherry Studio first-consumer signoff: `blocked`
- Codex second-consumer review: `signed off`
- Extension namespaces used or reserved: `extensions.cherry_studio`, `extensions.codex`
- OpenAPI change: not required, because no public API/schema changed in this story
- Protocol conclusion: no parallel core protocol, no parallel resource model, and no independent open capability endpoint are required
