---
baseline_commit: fc3d5f203421d8e2e4b60b5b6fb28c9f67a49b22
---

# Story AP-1.4: 建立与客户端注册解耦的发布投影基线

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 平台管理员,
I want 在不依赖真实客户端记录的前提下先定义 projection/exposure 语义,
so that Epic AP-1 可以独立交付发布模型基线，而不会前向依赖 Epic AP-2 的客户端接入实现。

## Acceptance Criteria

1. **Given** 某资源定义已存在且准备进入发布流程  
   **When** 平台设计发布投影骨架  
   **Then** 平台先定义 projection 对象、visibility/callable 语义与状态转移规则  
   **And** 这些规则可以在测试桩或占位客户端场景中被验证。
2. **Given** Epic AP-2 后续引入正式客户端记录
   **When** projection 需要绑定目标客户端  
   **Then** Epic AP-1 的投影语义可直接复用
   **And** 不需要返工资源生命周期与审计基线。

## Tasks / Subtasks

- [x] 在现有 registry/version/lifecycle 基线上建立 projection/exposure 数据骨架 (AC: 1, 2)
  - [x] 新增 `model/agentplatform/exposure.go`（或等价命名），定义 `agent_platform_resource_exposures` 表，至少覆盖：`resource_id`、`resource_version`、占位客户端标识、`visibility_state`、`callable_state`、`freshness_ttl_seconds`、`etag`、`extensions_json`、`published_at`、`revoked_at`。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#319-339]
  - [x] 在 `model/agentplatform/migration.go` 中注册 exposure 表迁移，并继续保持 Agent Platform bounded context 的集中迁移模式。[Source: model/agentplatform/migration.go; model/main.go]
  - [x] 本故事中的“目标客户端”必须支持占位或测试桩形式，不依赖 Epic AP-2 的正式 `agent_platform_clients` 记录；可以使用 placeholder client key / stub target 字段，但不要反向引入真实 client registration 模型依赖。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-1.4]

- [x] 落 visibility / callable 状态语义与最小状态转移规则 (AC: 1)
  - [x] 明确并实现投影层自己的状态字段：`visibility_state` 与 `callable_state` 必须独立表达，不能从 resource definition 或 lifecycle 状态隐式推断。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Definition vs exposure separation; _bmad-output/planning-artifacts/architecture-agent-platform.md#337-339]
  - [x] 至少支持这些基础语义：visible but not callable、callable for one target but not another、projection revoke 不删除底层 definition。即使当前目标只是占位客户端，这些语义也必须在模型/service 层成立。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#337-339]
  - [x] 当前故事不要求完整 freshness/revoke 收敛实现，但 `freshness_ttl_seconds` 与 `etag` 字段要先纳入投影模型，作为后续 2.x 与 5.x 的扩展锚点。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#592-595]

- [x] 提供不依赖真实 client registration 的最小 projection 管理 API (AC: 1, 2)
  - [x] 扩展 `service/agentplatform`、`dto/agentplatform`、`controller/agentplatform` 和 `router/agentplatform-router.go`，支持最小的 projection/exposure 创建、读取、列表查询和状态更新能力。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#555; _bmad-output/planning-artifacts/architecture-agent-platform.md#Requirements -> Components Mapping]
  - [x] 这些 API 只属于 control plane，不属于 `/api/open-capabilities/**`；controller 不得直接把 exposure 逻辑当成开放消费面接口暴露。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Control plane vs open plane separation]
  - [x] 若当前故事为了占位客户端引入 `client_key` / `client_scope` 之类的 stub 字段，必须把它们建模为“projection target placeholder”，方便 Epic AP-2 将来替换成正式 client 记录，而不是把占位字段散落在 extensions 里不可控增长。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-1.4; _bmad-output/planning-artifacts/architecture-agent-platform.md#Client vs client_instance separation]

- [x] 明确 1.4 与 1.5 / 2.x 的边界，避免把 projection baseline 过早做成完整发布系统 (AC: 1, 2)
  - [x] 本故事只定义 projection/exposure 语义基线和最小 stub-target 管理，不实现真实客户端注册、OAuth grant、token 验证、open-capability discovery/detail、projection freshness 收敛后台任务或 UI。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-1.5; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#5-AP-6-的具体定位; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#6-暂不进入-MVP-的内容]
  - [x] 也不要在 1.4 就引入 `agent_platform_clients` 的完整主模型；那属于 Epic AP-2 的职责。1.4 的职责是保证 projection 语义独立于“是否已有真实客户端记录”。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-1.4; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-4]

- [x] 补 projection baseline 的模型、服务、控制器测试与合同 (AC: 1, 2)
  - [x] 新增 `model/agentplatform/*_test.go`，覆盖 exposure 表迁移、visible/callable 双状态分离、占位客户端目标持久化，以及 revoke 只影响 projection 不删除 resource definition。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#337-339; _bmad-output/implementation-artifacts/ap-1-3-implement-unified-lifecycle-version-and-rollback-actions.md]
  - [x] 新增 `service/agentplatform/*_test.go`，覆盖 stub-target projection 的创建/读取、非法状态迁移拒绝、以及后续 Epic AP-2 可以在不改 projection 语义的前提下替换 target 绑定的兼容性假设。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-1.4]
  - [x] 新增 `controller/agentplatform/*_test.go`，覆盖最小 exposure 管理 API 的成功读写与非法输入场景；若新增管理面接口，同步更新 `docs/openapi/api.json`，但不要改 `docs/openapi/relay.json`。[Source: docs/openapi/api.json; AGENTS.md#Overview]

## Dev Notes

- `1.4` 的关键不是“把真实客户端做完”，而是先让 projection/exposure 成为独立对象。这样 `1.5` 才能做真正的 exposure 模型，`2.x` 才能接真实 client registration，而不需要回头拆 definition/lifecycle 逻辑。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-1.4]
- projection 层必须独立于 definition 状态存在，不能根据 `resource.status` 自动推断 `visible` / `callable`。架构文档已经明确：definition tables own canonical metadata，exposure tables own per-client visibility/callability/freshness。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Definition vs exposure separation]
- 当前故事里即便没有真实 client 记录，也要把“目标对象”抽象出来，否则后续 Epic AP-2 接 client 注册时仍然会被迫推翻 exposure 模型。一个受控的 stub target 比直接把 client 信息混入 extensions 更可演进。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-1.4; _bmad-output/planning-artifacts/architecture-agent-platform.md#Client vs client_instance separation]
- `visibility_state` 和 `callable_state` 必须是两个独立字段，不能退回到一个模糊 `status`。这是后续“visible but not callable”诊断链条成立的前提。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#337-339]
- 1.4 还不负责 open-capability 消费面，也不负责 freshness/revoke 的真实收敛执行，但 `freshness_ttl_seconds` / `etag` 这些字段值得在 exposure 表先占位，这样后续 2.x 不用再做破坏性迁移。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#592-595]
- 所有新增结构化字段仍然要遵守 1.2 的跨库/JSON 规则：`TEXT` + `common.*`，不引入 JSONB 或数据库专属运算符。[Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go; AGENTS.md#Rule 2]

### Project Structure Notes

- 预计新增文件：
  - `model/agentplatform/exposure.go`
  - `service/agentplatform/exposure.go`
  - `controller/agentplatform/exposure.go`
  - `dto/agentplatform/exposure.go`
  - 对应 `*_test.go`
- 预计修改文件：
  - `model/agentplatform/migration.go`
  - `router/agentplatform-router.go`
  - `docs/openapi/api.json`
- 保持不改或只读依赖：
  - `relay/**`
  - `docs/openapi/relay.json`
  - `web/default/**`
  - `oauth/**`
  - `model/agentplatform/client*.go`（若尚未正式引入）

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-1.4: 建立与客户端注册解耦的发布投影基线]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#319-339]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#337-339]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#592-595]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Definition vs exposure separation]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Client vs client_instance separation]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-3]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-4]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#5-AP-6-的具体定位]
- [Source: _bmad-output/implementation-artifacts/ap-1-3-implement-unified-lifecycle-version-and-rollback-actions.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 对 Agent Platform 的 `1.4 -> ap-1-4-*` key 映射沿用已修正逻辑；本 story 文件由 manual create-story takeover 补齐，用于避免 create-story 停留在 `1.3` lifecycle 上下文中空转。
- 已加载 `epics-agent-platform.md`、`architecture-agent-platform.md`、`prd-agent-platform-2026-05-31/prd.md` / `addendum.md`，并参考已完成的 `ap-1-1`、`ap-1-2`、`ap-1-3` 产物作为前序上下文。
- 当前仓库已经具备 registry、typed detail 和 lifecycle/action audit 基线，因此 `1.4` 的关注点应严格收敛到 projection/exposure 语义与 stub-target 管理，而不是回头做 client registration。

### Completion Notes List

- 已补齐 `agent_platform_resource_exposures` 基线模型，明确 projection/exposure 独立于 definition 状态存在。
- 已实现 stub-target 基线：`client_key` / `client_scope`、`visibility_state` / `callable_state`、TTL / ETag / extensions 持久化。
- 已补 exposure 管理 API：创建、查询、读取、更新投影，不依赖真实 client registration。
- 已完成 exposure 相关 model/service/controller 定向测试，并通过 `GOCACHE=/private/tmp/go-build-cache go test ./model/agentplatform ./service/agentplatform ./controller/agentplatform` 验证。

### File List

- `_bmad-output/implementation-artifacts/ap-1-4-establish-projection-baseline-decoupled-from-client-registration.md`
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

- 2026-06-01: 完成 projection/exposure 语义基线、stub-target 管理 API 与定向测试，并将故事推进为 done。
