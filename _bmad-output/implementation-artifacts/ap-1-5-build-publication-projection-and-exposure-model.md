# Story AP-1.5: 建立发布投影与 exposure 模型

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 平台管理员,
I want 发布投影独立于源资源定义进行管理,
so that 同一个资源可以针对不同客户端拥有不同的 visible/callable 状态。

## Acceptance Criteria

1. **Given** 某资源定义已存在但尚未暴露给客户端  
   **When** 下游客户端执行 discovery  
   **Then** 该资源不会出现在开放能力面中  
   **And** 控制面中的源定义数据保持完整。
2. **Given** 某资源已经发布给客户端 A，但未发布给客户端 B  
   **When** 查询 exposure 状态  
   **Then** 平台能够表达按客户端区分的 visibility 与 callable state  
   **And** exposure 不能从 registry 状态自动推断。
3. **Given** 某客户端上的发布投影被 revoke  
   **When** 该资源对其他客户端仍然有效  
   **Then** 平台只撤销目标客户端相关的 published projection  
   **And** 不删除或破坏底层资源定义。

## Tasks / Subtasks

- [x] 把 `1.4` 的 projection baseline 扩展成真正的 per-target exposure 模型 (AC: 1, 2, 3)
  - [x] 在已有 `model/agentplatform/exposure.go` 基础上补足“同一 `resource_id` + `resource_version` 对多个 target 共存”的约束，确保 target A、target B 的 exposure 行独立存在、独立更新，不会互相覆盖。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-1.5; _bmad-output/planning-artifacts/architecture-agent-platform.md#337-339]
  - [x] exposure 仍然必须独立于 registry/status 存在：definition 状态更新不等于 target exposure 自动变化；反之 revoke 某个 target 的 projection 也不能删除 resource definition 或破坏其他 target 的 published projection。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Definition vs exposure separation; _bmad-output/planning-artifacts/architecture-agent-platform.md#319-339]
  - [x] 若 `1.4` 使用的是 stub target 模型，本故事应把 target 维度提升到真正可区分“client A / client B”的投影粒度，但仍不引入 Epic AP-2 的完整 client registration 主模型。[Source: _bmad-output/implementation-artifacts/ap-1-4-establish-projection-baseline-decoupled-from-client-registration.md]

- [x] 实现真正的 per-target publish / revoke / read 语义 (AC: 1, 2, 3)
  - [x] 在 `service/agentplatform` 中明确提供针对 target 的发布投影管理入口：create published projection、读取单 target exposure、列出某资源所有 exposures、revoke 单 target exposure。这里的“publish”是 exposure 语义，不是重做 1.3 的 lifecycle。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#974-984; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-3]
  - [x] 实现后必须满足：资源若未对某 target 发布，该 target 不应在 discovery 视角中被视为可见；资源若已对 A 发布、未对 B 发布，A/B 查询 exposure 时状态必须不同。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-1.5]
  - [x] revoke 单 target exposure 时，只更新该 exposure 行的 visibility/callable/revoked_at 及必要审计，不影响同资源对其它 target 的 published projection。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#337-339]

- [x] 扩展控制面 API，表达“同一个资源对不同目标的不同暴露状态” (AC: 1, 2, 3)
  - [x] 扩展 `controller/agentplatform`、`dto/agentplatform` 与 `router/agentplatform-router.go`，提供最小管理 API 支持：按 target 创建/更新 exposure、列出某资源的 exposure 列表、按 target revoke、按 target 查看详情。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#555; _bmad-output/planning-artifacts/architecture-agent-platform.md#Requirements -> Components Mapping]
  - [x] 这些 API 仍属于 control plane；本故事不要提前实现 `/api/open-capabilities/discovery`，但要让 exposure 数据结构足以供后续 discovery/detail 直接消费。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#577; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-5]
  - [x] 如果当前 API 中 target 还叫 placeholder/stub key，本故事要把字段含义写清楚：它现在表达“发布目标标识”，后续 Epic AP-2 可将其替换为正式 client identity，而不是重新推翻 exposure 模型。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Client vs client_instance separation]

- [x] 锁定 1.5 的边界，不提前做真实 discovery/OAuth/client 管理 (AC: 1, 2, 3)
  - [x] 本故事不实现真实 `agent_platform_clients`、grant/token、OAuth、open-capabilities discovery/detail/invoke、真实 freshness 收敛执行或 UI；重点仅是把 exposure 模型做成“可直接承接这些能力”的稳定基线。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-4; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#6-暂不进入-MVP-的内容]
  - [x] 也不在本故事里引入 `client_instance` 级别的复杂 revoke/telemetry 语义；1.5 只需把 per-target projection 语义建立起来。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Client vs client_instance separation]

- [x] 补齐模型、服务、控制器测试与合同，证明 exposure 的 per-target 独立性真实成立 (AC: 1, 2, 3)
  - [x] 新增 `model/agentplatform/*_test.go`，覆盖：同一资源对多个 target 的 exposure 并存、单 target revoke 不影响其它 target、以及 resource definition 不因 exposure revoke 被删除。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#337-339]
  - [x] 新增 `service/agentplatform/*_test.go`，覆盖：A 已发布/B 未发布、A revoke/B 保持 published、非法 target 或非法状态更新被拒绝。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-1.5]
  - [x] 新增 `controller/agentplatform/*_test.go`，覆盖最小曝光管理 API 的成功读写与错误输入；若管理面合同变化，同步更新 `docs/openapi/api.json`。[Source: docs/openapi/api.json]

## Dev Notes

- `1.5` 不是重新发明 exposure，而是把 `1.4` 的“projection baseline”做成真正可表达“对 A 有效、对 B 无效”的 per-target 模型。[Source: _bmad-output/implementation-artifacts/ap-1-4-establish-projection-baseline-decoupled-from-client-registration.md]
- 这一层最大的坑是把 exposure 状态重新简化回 definition 状态，或者让 revoke 删除 definition。无论实现上多方便，这两种做法都会直接违背 `1.5` AC。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-1.5; _bmad-output/planning-artifacts/architecture-agent-platform.md#Definition vs exposure separation]
- `visible` / `callable` 仍然必须作为两个独立维度保留，不要退回单一字段或“published 即 callable”的偷懒实现。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#337-339]
- 即使当前还没有真实 client registration，对 target 的抽象也必须足够稳定，后续 Epic AP-2 才能直接把 `client_id` / capability declaration 接上，而不是返工 exposure 表结构。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Client vs client_instance separation]
- 1.5 仍不负责 open capability plane 的 discovery 实现，但 exposure 表的数据字段和状态语义要让后续 `/api/open-capabilities/discovery` 能直接消费“只返回 published projections”这一原则。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#577]

### Project Structure Notes

- 预计新增文件：
  - `service/agentplatform/exposure_lifecycle.go`（若需要把 per-target publish/revoke 与 baseline CRUD 分开）
  - 对应 `*_test.go`
- 预计修改文件：
  - `model/agentplatform/exposure.go`
  - `service/agentplatform/exposure.go`
  - `controller/agentplatform/exposure.go`
  - `dto/agentplatform/exposure.go`
  - `router/agentplatform-router.go`
  - `docs/openapi/api.json`
- 保持不改或只读依赖：
  - `relay/**`
  - `docs/openapi/relay.json`
  - `web/default/**`
  - `oauth/**`
  - `model/agentplatform/client*.go`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-1.5: 建立发布投影与 exposure 模型]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#319-339]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#337-339]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#555]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#577]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Definition vs exposure separation]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Client vs client_instance separation]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-3]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-5]
- [Source: _bmad-output/implementation-artifacts/ap-1-4-establish-projection-baseline-decoupled-from-client-registration.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 对 Agent Platform 的 `1.5 -> ap-1-5-*` key 映射沿用已修正逻辑；本 story 文件由 manual create-story takeover 补齐，用于避免 create-story 停留在 `1.4` exposure baseline 上下文中空转。
- 已加载 `epics-agent-platform.md`、`architecture-agent-platform.md`、`prd-agent-platform-2026-05-31/prd.md`，并参考已完成的 `ap-1-4` projection baseline 产物作为前序实现上下文。
- 当前仓库已经具备 exposure baseline 和 per-target stub target 字段，因此 `1.5` 的关注点应严格收敛到“真正的 per-target exposure 独立性”，而不是重新定义 projection。

### Completion Notes List

- 已把 `1.4` 的 exposure baseline 扩展成真正的 per-target exposure 模型，支持同一资源对多个 target 并存投影。
- 已实现 per-target revoke，不会删除底层 resource definition，也不会影响其他 target 的 published projection。
- 已补 exposure 管理 API 的 per-target 读写与 revoke 语义，并保持 control plane 边界，不提前引入真实 discovery/client/OAuth。
- 已完成 exposure 独立性相关 model/service/controller 定向测试，并通过 `GOCACHE=/private/tmp/go-build-cache go test ./model/agentplatform ./service/agentplatform ./controller/agentplatform` 验证。

### File List

- `_bmad-output/implementation-artifacts/ap-1-5-build-publication-projection-and-exposure-model.md`
- `model/agentplatform/exposure.go`
- `model/agentplatform/exposure_test.go`
- `service/agentplatform/exposure.go`
- `service/agentplatform/exposure_test.go`
- `controller/agentplatform/exposure.go`
- `controller/agentplatform/exposure_test.go`
- `dto/agentplatform/exposure.go`
- `model/agentplatform/migration.go`
- `router/agentplatform-router.go`


## Change Log

- 2026-06-01: 完成 per-target exposure 独立性、单 target revoke 语义与定向测试，并将故事推进为 done。
