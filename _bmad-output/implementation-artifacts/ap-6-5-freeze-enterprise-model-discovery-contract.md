---
baseline_commit: b9c268b832682b5beca6a860a869e15ab52d3f79
---

# Story AP-6.5: 冻结 Enterprise Model Discovery 公共契约

Status: done

> Sprint key 映射：本故事来自 Agent Platform Epic AP-6，真实 sprint-status key 为 `ap-6-5-freeze-enterprise-model-discovery-contract`。本文件按用户要求落地为 `_bmad-output/implementation-artifacts/ap-6-5-*.md`。实现和状态同步必须以 sprint-status 中的 `ap-6-5-*` key 为准。

## Story

As a 客户端集成人员,
I want 获得企业模型发现、默认模型和模型状态的公共契约,
so that Cherry Studio P0 可以展示可用模型并解释不可用原因。

## Acceptance Criteria

1. **Given** 下游请求可用模型列表  
   **When** 平台返回 enterprise model discovery 响应  
   **Then** 契约明确 `modelId`、`providerStableId`、`displayName`、`isDefault`、`status`、`disabledReason`、`capabilities`、`accountId` / `tenantId` 来源语义  
   **And** 明确该契约与现有 `/api/models`、`/api/user/models`、`/v1/models` 的关系，不能用现有管理或 relay 接口模糊替代。

2. **Given** 出现无默认模型、多默认模型、默认模型不可用、模型被禁用或 provider 不可用  
   **When** 下游读取模型状态  
   **Then** 契约明确状态矩阵、错误语义和 UI 展示建议。

## Tasks / Subtasks

- [x] 定义 enterprise model discovery downstream projection surface (AC: 1)
  - [x] 在既有 open capability plane 下定义只读 model discovery endpoint，而不是复用 `/api/models`、`/api/user/models` 或 `/v1/models`。
  - [x] 明确鉴权 scope、response envelope 和与现有 AP-6 contract version 的关系。
  - [x] 说明该 endpoint 是 downstream public projection，不是 dashboard 管理接口或 relay 兼容接口。

- [x] 冻结 enterprise model discovery 公共字段与状态语义 (AC: 1, 2)
  - [x] 冻结 `modelId`、`providerStableId`、`displayName`、`isDefault`、`status`、`disabledReason`、`capabilities`、`accountId`、`tenantId`。
  - [x] 明确 no default / multiple default / default disabled / provider offline / model unavailable / account-tenant mismatch 的公共语义与示例。
  - [x] 若某些字段暂由平台投影填充而非真实企业数据源，也必须在 contract 中说明来源与限制。

- [x] 对齐 DTO / Controller / Service / OpenAPI / Fixture (AC: 1, 2)
  - [x] 新增最小 DTO、controller、service 和 router 注册，使 enterprise model discovery 成为真实公共投影接口。
  - [x] 最小更新 `docs/openapi/api.json`，补充 request/response schema。
  - [x] 更新 `tests/agentplatform/conformance/fixtures.go`，把 model-discovery pending fixtures 转为真实 fixtures，并加入 drift guardrails。

- [x] 补强 runtime 与 conformance tests (AC: 1, 2)
  - [x] 扩展 `tests/agentplatform/conformance/fixtures_test.go` 校验 enterprise model discovery required fields、default-model state 和 `/api/models` 非等价约束。
  - [x] 为 controller/service 增加覆盖：default model、no default、multiple defaults、disabled / offline / unavailable、account/tenant mismatch。

- [x] 同步 signoff 与 UX 边界 (AC: 2)
  - [x] 更新 `docs/agent-platform-consumer-signoff.md` 与 contract spec 中 enterprise model discovery 条目。
  - [x] 保持 Cherry Studio overall first-consumer signoff 仅在 6.5 与 6.6 都完成后解除 blocked；6.6 完成后由 6.8 统一收敛为 signed off。

## Dev Notes

### Scope and Constraints

- AP-6.5 不能用 `/api/models`、`/api/user/models` 或 `/v1/models` 充当完成证据；必须形成 Agent Platform 下游公共投影。
- 架构 amendment 已明确 AP-6 不新增第四个 runtime plane，因此 model discovery 应归属既有 plane；最小实现优先考虑 open capability plane 下的新只读投影路径。
- 本故事要补真实 endpoint + docs/OpenAPI/fixtures/tests；如果只写文档，仍然会停留在“说明性规格”。

### Current Evidence

- 当前 conformance 中 model discovery 仍全部是 pending fixtures。
- signoff artifact 仍把 enterprise model discovery 标记为 `blocked`。
- 现有 `/api/models`、`/api/user/models`、`/v1/models` 属于 dashboard / relay / legacy surfaces，不能直接等价为下游公共契约。

### Testing Requirements

- `GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance`
- `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Completion Notes List

- 2026-06-03 19:41 +0800：基于 AP-6 architecture amendment、contract gap matrix、pending conformance fixtures 与现有 router/model surfaces 创建 AP-6.5 story artifact；结论是必须新增真实 downstream model discovery projection，而非复用 `/api/models`、`/api/user/models` 或 `/v1/models`。
- 2026-06-03 19:58 +0800：新增 `GET /api/open-capabilities/models` downstream public projection，使用 client namespaced extension `model_discovery.config` 生成 enterprise model discovery contract，并冻结 default/no-default/multiple-default/default-disabled/provider-offline/unavailable/account-tenant-mismatch 公共语义。
- 2026-06-03 19:59 +0800：目标验证通过：`GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance`、`GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`。
- 2026-06-03：orchestration-5 source-of-truth review 自动修复 default model 不在有效列表时仍返回 `default_state=resolved` 的合同漂移；新增 `default_unavailable`，同步 service、fixture、OpenAPI、contract spec 与 conformance 防漂移测试。

### File List

- `_bmad-output/implementation-artifacts/ap-6-5-freeze-enterprise-model-discovery-contract.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `dto/agentplatform/open_capabilities.go`
- `service/agentplatform/model_discovery.go`
- `service/agentplatform/model_discovery_test.go`
- `controller/agentplatform/open_capabilities.go`
- `controller/agentplatform/open_capabilities_test.go`
- `router/open_capabilities_router.go`
- `docs/openapi/api.json`
- `docs/agent-platform-downstream-contract-spec.md`
- `docs/agent-platform-consumer-signoff.md`
- `tests/agentplatform/conformance/fixtures.go`
- `tests/agentplatform/conformance/fixtures_test.go`

## Senior Developer Review (AI) - Orchestration 5 Source-of-Truth Sweep

Reviewer: GPT-5 Codex
Date: 2026-06-03
Outcome: Approve after auto-fix

### Findings Fixed

- [HIGH] `model_discovery_model_unavailable` fixture 和 runtime 把 configured default 不在有效模型列表的场景返回为 `default_state=resolved`，但 contract 对 `resolved` 的定义是“exactly one effective default model is present”。这会让下游把不可用默认模型误判为可用默认模型。已新增 `default_unavailable`，并同步 `service/agentplatform/model_discovery.go`、OpenAPI enum、contract spec、fixture 与 `TestModelDiscoveryUnavailableDefaultIsNotResolved`。

### Verification

- `GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance ./service/agentplatform ./controller/agentplatform` 通过。

## Change Log

- 2026-06-03：创建 Story AP-6.5 context artifact，状态设为 `ready-for-dev`。
- 2026-06-03：source-of-truth review 修复 default unavailable 状态漂移，新增 conformance 防回归断言并保持 story done。
