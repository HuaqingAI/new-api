---
baseline_commit: 7ebc1da33d0bb5b6cc8f28dd15d28417fcf99288
---

# Story AP-6.4: 冻结 Skill Invoke 与 Knowledge Query Request/Response Spec

Status: done

> Sprint key 映射：本故事来自 Agent Platform Epic AP-6，真实 sprint-status key 为 `ap-6-4-freeze-skill-invoke-and-knowledge-query-specs`。本文件按用户要求落地为 `_bmad-output/implementation-artifacts/ap-6-4-*.md`。实现和状态同步必须以 sprint-status 中的 `ap-6-4-*` key 为准。

## Story

As a 客户端集成人员,
I want Skill invoke 与 Knowledge query 有正式请求/响应规格,
so that 下游可以在不阅读服务端代码的情况下实现调用和错误处理。

## Acceptance Criteria

1. **Given** 下游调用 Skill  
   **When** 查阅 AP-6 规格  
   **Then** Skill metadata、input schema、output schema、sync invoke 请求/响应、timeout、upstream failure 与 contract validation error 均有示例  
   **And** P0 明确是否只支持 sync invoke；若 async / cancel / task status 不进入 P0，必须显式声明。

2. **Given** 下游查询 Knowledge  
   **When** 查阅 AP-6 规格  
   **Then** retrieval query shape、items、citations、source metadata、排序信息、provider failure 与敏感配置保护均有示例  
   **And** 公共契约不暴露 LightRAG / FastGPT / RAGFlow 等 provider-native 字段。

## Tasks / Subtasks

- [x] 冻结 Skill invoke 公共 request/response 字段与 P0 边界 (AC: 1)
  - [x] 更新 `docs/agent-platform-downstream-contract-spec.md` 的 `11.4 Skill Invoke and Knowledge Query Contracts`，将状态从 `pending freeze` 改为 `frozen`。
  - [x] 明确 `POST /api/open-capabilities/skills/{id}/invoke` 的 request 至少包含标准 JSON payload，response MUST 返回 `resource_id`、`resource_version`、`contract_version`、`output`。
  - [x] 明确 Skill metadata 由已存在的 resource version / detail schema 提供：`invoke_schema`、`output_schema`、`invoke_mode`、`timeout_seconds` 为既有 metadata source of truth。
  - [x] 明确 AP-6.4 P0 只冻结 sync invoke；`async` / cancel / task status 不进入 P0，不得被现有 OpenAPI 暗示为已支持公共协议。

- [x] 冻结 Knowledge query 公共 request/response 字段与 provider-safe 边界 (AC: 2)
  - [x] 明确 `POST /api/open-capabilities/knowledge-bases/{id}/query` 的 request 至少包含 `query`；response MUST 返回 `resource_id`、`resource_version`、`contract_version`、`items`、`citations`。
  - [x] 明确 retrieval item / citation 的公共字段，允许 `metadata` 作为向后兼容扩展容器，但不得泄露 provider-native config。
  - [x] 明确 provider-specific 字段不得进入公共顶层 response，也不得出现 `provider_config`、provider endpoint、provider-native retrieval object。

- [x] 对齐 DTO / Controller / Service / OpenAPI / Fixture (AC: 1, 2)
  - [x] 以 `controller/agentplatform/open_capabilities.go`、`service/agentplatform/skill_invoke.go`、`service/agentplatform/knowledge_query.go`、`dto/agentplatform/resource_version.go` 为实现 source of truth，对齐 `docs/openapi/api.json` 与 `tests/agentplatform/conformance/fixtures.go`。
  - [x] 若当前 OpenAPI 对 skill invoke / knowledge query request/response shape 缺少 required 字段或示例，做最小补齐。
  - [x] 若 runtime 成功 payload 缺少契约冻结所需字段，做最小代码修补，不新增新 endpoint 或 runtime plane。

- [x] 补强 conformance 与 runtime tests (AC: 1, 2)
  - [x] 扩展 `tests/agentplatform/conformance/fixtures_test.go`，校验 skill invoke / knowledge query 的 OpenAPI required 字段、sync-only P0 边界、provider-native field non-leakage。
  - [x] 扩展 `controller/agentplatform/open_capabilities_test.go` 覆盖 skill invoke success、timeout、upstream failure、knowledge query success、provider failure、payload shape。
  - [x] 扩展 `service/agentplatform/skill_invoke_test.go`、`service/agentplatform/knowledge_query_test.go`，确认 contract-invalid、timeout / upstream failed、empty query 与 provider-safe behavior。

- [x] 同步 signoff artifact 但不提前完成 6.8 (AC: 1, 2)
  - [x] 更新 `docs/agent-platform-consumer-signoff.md` 中 Skill invoke / Knowledge query 的证据描述，使其与 AP-6.4 冻结内容一致。
  - [x] 保持 Cherry Studio overall first-consumer signoff 仍因 AP-6.5 / AP-6.6 为 `blocked`。

## Dev Notes

### Scope and Boundaries

- AP-6.4 只冻结 Skill invoke 与 Knowledge query 的公共 request/response spec，不处理 enterprise model discovery，也不完成错误码矩阵总表。
- 当前 runtime 已具备 skill invoke 与 retrieval-style knowledge query 的后端实现；本故事的主要目标是把已有行为冻结为正式公共契约，而不是新增全新工作流。
- AP-6.4 必须延续 AP-6.1 / AP-6.2 / AP-6.3 的同一模式：contract spec + OpenAPI + fixture/conformance + signoff evidence 同步。

### Runtime Facts to Preserve

- `POST /api/open-capabilities/skills/:id/invoke` 当前成功 payload 返回 `resource_id`、`resource_version`、`contract_version`、`output`。
- `POST /api/open-capabilities/knowledge-bases/:id/query` 当前成功 payload 返回 `resource_id`、`resource_version`、`contract_version`、`items`、`citations`。
- Skill runtime 通过 `SkillDef` 已有 `invoke_schema_json`、`output_schema_json`、`invoke_mode`、`timeout_seconds`、`binding_config_json`。
- Knowledge runtime 通过 `KnowledgeDef` 已有 `query_schema_json`、`citation_schema_json`、`freshness_rules_json`、`provider_capabilities_json`，但 provider config 不得暴露为下游公共字段。
- 当前 `KnowledgeQueryService` 只支持 `knowledge_mode=retrieval`；这是 AP-6.4 P0 的关键约束。

### Contract Decisions the Dev Agent Must Not Miss

- AP-6.4 P0 只冻结 sync invoke；即使底层模型允许 `async` 值存在于资源版本定义，也不能把 async/cancel/task-status 伪装为已支持公共协议。
- Knowledge query 公共响应允许标准化 `items`、`citations`、`metadata`，但不得输出 provider-native 顶层字段，也不得把 provider endpoint / config 泄露进 OpenAPI 示例、fixtures 或 runtime payload。
- 仍需沿用稳定 error envelope：`contractInvalid`、`timeout`、`upstreamFailed`、`resourceOffline`、`permissionDenied` 等由统一 error response 表达。

### Testing Requirements

- `GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance`
- `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Completion Notes List

- 2026-06-03 19:15 +0800：基于 AP-6 epic、architecture、现有 runtime/openapi/conformance 证据创建 AP-6.4 story artifact，并将后续执行固定回 Codex 优先路径。
- 2026-06-03 19:26 +0800：冻结 Skill invoke / Knowledge query 公共 request/response spec，新增正式 success DTO、OpenAPI required schema、sync-only / retrieval-only P0 说明，以及 provider-native non-leakage conformance guardrails。
- 2026-06-03 19:27 +0800：目标验证通过：`GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance`、`GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`。

### File List

- `_bmad-output/implementation-artifacts/ap-6-4-freeze-skill-invoke-and-knowledge-query-specs.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `dto/agentplatform/open_capabilities.go`
- `controller/agentplatform/open_capabilities.go`
- `controller/agentplatform/open_capabilities_test.go`
- `docs/agent-platform-downstream-contract-spec.md`
- `docs/agent-platform-consumer-signoff.md`
- `docs/openapi/api.json`
- `tests/agentplatform/conformance/fixtures.go`
- `tests/agentplatform/conformance/fixtures_test.go`

## Change Log

- 2026-06-03：创建 Story AP-6.4 context artifact，状态设为 `ready-for-dev`。
