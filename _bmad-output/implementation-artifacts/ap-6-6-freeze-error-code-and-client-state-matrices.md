---
baseline_commit: 77132bd2d98be71d63640648657e4d588ea221fb
---

# Story 6.6: 冻结错误码矩阵与客户端状态矩阵

Status: done

> Sprint key 映射：本故事来自 Agent Platform Epic 6，真实 sprint-status key 为 `ap-6-6-freeze-error-code-and-client-state-matrices`。本文件按用户要求落地为 `_bmad-output/implementation-artifacts/ap-6-6-*.md`。实现和状态同步必须以 sprint-status 中的 `ap-6-6-*` key 为准。

## Story

As a 下游产品负责人,
I want 平台错误码与客户端展示状态有一张稳定矩阵,
so that 登录过期、空列表、加载失败、无可用资源、网络失败和资源不可调用等状态不会被客户端各自猜测。

## Acceptance Criteria

1. **Given** 平台返回 open capability error envelope  
   **When** 下游映射客户端状态  
   **Then** `permissionDenied`、`resourceRevoked`、`resourceOffline`、`quotaOrRateLimited`、`timeout`、`upstreamFailed`、`contractInvalid` 等错误码都有 HTTP/status、retryable、责任边界和 UI state 映射  
   **And** 下游需求中的 `loginExpired`、`noAssignedResource`、`networkFailed`、`empty`、`loadFailed` 等客户端态被纳入矩阵。

## Tasks / Subtasks

- [x] 冻结平台错误码矩阵 (AC: 1)
  - [x] 在 `docs/agent-platform-downstream-contract-spec.md#11.6` 中将状态从 `pending freeze` 改为 `frozen`。
  - [x] 明确每个 stable error code 的 retryable、责任边界、推荐客户端状态与典型触发场景。
  - [x] 区分 error envelope code 与非 error payload state，例如 `stale`、`revoked`、`offline`、`provider_offline`。

- [x] 冻结客户端状态矩阵 (AC: 1)
  - [x] 明确 `loginExpired`、`empty`、`loadFailed`、`networkFailed`、`noAssignedResource`、`visibleButNotCallable`、`stale`、`revoked`、`offline` 的定义与展示边界。
  - [x] 说明哪些状态来自 error envelope，哪些来自成功 payload / model discovery / discovery freshness。
  - [x] 补充对 resource discovery、skill invoke、knowledge query、model discovery 的矩阵归属示例。

- [x] 对齐 signoff / fixture / conformance (AC: 1)
  - [x] 将 `tests/agentplatform/conformance/fixtures.go` 中 `error_client_state_matrix` 从 pending fixture 转为真实 artifact。
  - [x] 扩展 `tests/agentplatform/conformance/fixtures_test.go` 校验 error-state matrix 覆盖与 contract/signoff 一致性。
  - [x] 更新 `docs/agent-platform-consumer-signoff.md` 中 Error matrix and client state matrix 行。

- [x] 最小 OpenAPI / runtime consistency check (AC: 1)
  - [x] 如需引用 stable error envelope 结构，只做最小文档级核对，不新增新的 runtime error plane。
  - [x] 不修改 `/v1/**` 或 relay surface。

## Dev Notes

### Scope

- AP-6.6 主要是 contract freeze + matrix artifact + fixture/conformance；通常不需要新增 runtime endpoint。
- 但必须让 pending `error_client_state_matrix` fixture 变成真实、可验证的 contract asset。

### Existing Evidence

- stable error codes 已存在于 `service/agentplatform/open_capability_errors.go`
- freshness / revoke / offline / provider_offline / model discovery 状态已分别存在于 resource/model payloads
- UX 最小要求已列出 `loginExpired`、`empty`、`loadFailed`、`networkFailed`、`noAssignedResource`、`visibleButNotCallable`、`stale`、`revoked`、`offline`

### Testing Requirements

- `GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance`
- `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Completion Notes List

- 2026-06-03 20:05 +0800：基于现有 error envelope、resource/model state 语义与 UX contract 要求创建 AP-6.6 story artifact，准备将 pending matrix fixture 转为真实签核资产。
- 2026-06-03 20:12 +0800：冻结 platform error-code matrix 与 payload-derived client-state matrix，将 `error_client_state_matrix` fixture 转为真实 conformance artifact，并同步 signoff/contract matrix 到 signed-off 状态。
- 2026-06-03 20:13 +0800：目标验证通过：`GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance`、`GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`。

### File List

- `_bmad-output/implementation-artifacts/ap-6-6-freeze-error-code-and-client-state-matrices.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `docs/agent-platform-downstream-contract-spec.md`
- `docs/agent-platform-consumer-signoff.md`
- `tests/agentplatform/conformance/fixtures.go`
- `tests/agentplatform/conformance/fixtures_test.go`

## Change Log

- 2026-06-03：创建 Story 6.6 context artifact，状态设为 `ready-for-dev`。
