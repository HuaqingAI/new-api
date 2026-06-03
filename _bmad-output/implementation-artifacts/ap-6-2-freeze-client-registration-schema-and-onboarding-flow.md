---
baseline_commit: 1623c7f35af3be29aafe128a2775d2bb6c9f52bb
---

# Story 6.2: 冻结 Client Registration Schema 与 Onboarding 最小流程

Status: done

> Sprint key 映射：本故事来自 Agent Platform Epic 6，真实 sprint-status key 为 `ap-6-2-freeze-client-registration-schema-and-onboarding-flow`。本文件按用户要求落地为 `_bmad-output/implementation-artifacts/ap-6-2-*.md`。实现和状态同步必须以 sprint-status 中的 `ap-6-2-*` key 为准。

## Story

As a 平台管理员,
I want 有一套可签核的 client registration schema 与最小 onboarding 流程,
so that 运维人员可以为 Cherry Studio / Codex 注册客户端并解释每个字段的治理含义。

## Acceptance Criteria

1. **Given** 管理员创建或维护下游客户端  
   **When** 使用 AP-6 规格  
   **Then** `client_id`、`slug`、`client_type`、`allowed_grant_types`、`redirect_uris`、`allowed_scopes`、`contract_version`、`capabilities`、`extensions`、`allow_client_credentials` 均有字段语义、示例和校验规则  
   **And** 无效配置会映射到明确 onboarding 状态，而不是只显示通用失败。

2. **Given** `web/default` 当前只有 Agent Platform shell  
   **When** 评估 AP-6 最小交付面  
   **Then** 明确哪些 onboarding 动作必须在 UI 完成，哪些可暂由 API / fixture 支撑  
   **And** 若完整 Clients 工作区超出 AP-6，则拆入后续产品化 epic。

## Tasks / Subtasks

- [x] 冻结 client registration schema 文档 (AC: 1)
  - [x] 更新 `docs/agent-platform-downstream-contract-spec.md` 中 client registration / onboarding 相关章节，明确 `client_id`、`slug`、`display_name`、`client_type`、`status`、`allowed_grant_types`、`redirect_uris`、`allowed_scopes`、`contract_version`、`capabilities`、`extensions`、`allow_client_credentials` 的字段等级、语义、默认值与校验要求。
  - [x] 明确 `client_id` 是稳定 opaque id，`slug` 是管理侧可读标识，`extensions` 只能使用 namespaced keys，不得重写核心契约语义。
  - [x] 明确哪些字段属于 MUST/SHOULD/MAY/EXTENSION，并补示例 payload，避免下游靠 controller/runtime 猜字段。

- [x] 冻结 onboarding 最小流程与状态语义 (AC: 1, 2)
  - [x] 在 contract spec 和/或 consumer signoff 资产中定义最小 onboarding checklist：registration、redirect/callback、allowed scopes、contract version、capabilities、consent/OAuth prerequisites、mock fixture 使用方式。
  - [x] 明确 `invalid_integration`、`active`、`disabled` 等 client 状态的语义，以及哪些无效配置会进入 `invalid_integration` 而不是 generic failure。
  - [x] 明确 AP-6.2 不要求完整 Clients 产品化工作区；若 UI 尚未完成，必须说明当前由 API / fixture 支撑的最小闭环，不得伪称已具备完整自助 onboarding UI。

- [x] 对齐 DTO、Controller、Service、OpenAPI 三件套 (AC: 1)
  - [x] 以 `dto/agentplatform/client.go` 为 DTO source of truth，对齐 `controller/agentplatform/client.go`、`service/agentplatform/client.go` 和 `docs/openapi/api.json` 中 Clients 相关 schema。
  - [x] 若发现 OpenAPI 缺少 required fields、枚举、示例或响应 data schema，对 `docs/openapi/api.json` 做最小更新，保持与现有 runtime 一致。
  - [x] 如无需改 runtime，也要在 completion notes 中说明原因：本故事冻结既有 registration/onboarding 契约，不重写 Clients 业务逻辑。

- [x] 补强 client registration / onboarding 测试覆盖 (AC: 1, 2)
  - [x] 扩展或确认 `service/agentplatform/client_test.go` 覆盖有效 client 创建、`invalid_integration`、non-namespaced extension 拒绝、`allow_client_credentials`、contract/capability 缺失语义。
  - [x] 扩展或确认 `controller/agentplatform/client_test.go` 覆盖 create/get/update/list 流程，并验证 invalid input、status update、clients 列表最小 onboarding 数据可读取。
  - [x] 若对 OpenAPI 或 consumer signoff 增加防漂移约束，补充 `tests/agentplatform/conformance/fixtures_test.go` 或相邻测试，防止 registration/onboarding 契约与 fixture 漂移。

- [x] 同步 AP-6 signoff / UX 边界说明 (AC: 2)
  - [x] 更新 `docs/agent-platform-consumer-signoff.md` 中 client registration / onboarding 相关状态，说明 AP-6.2 是否已冻结到可签核状态，以及仍由后续故事承担的 UI/productization 缺口。
  - [x] 对齐 `_bmad-output/planning-artifacts/ux-agent-platform.md` 中 3.6 Clients onboarding / 3.8 signoff 要求到当前冻结结果，确保不会误导运营认为完整 Clients 工作区已完成。
  - [x] 明确 Codex second-consumer 仍复用同一 registration schema，不新增平行 client object model。

## Dev Notes

### Story Source and Scope

- Story 6.2 属于 Agent Platform Epic 6 “下游公共契约冻结与接入签核”。它聚焦冻结 client registration schema 与 onboarding 最小闭环，而不是重建 control plane 或完整前端工作区。[Source: `_bmad-output/planning-artifacts/epics-agent-platform.md#Story 6.2`; `_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#5.2 2026-06-02 Correct Course 重处理后的定位`]
- 本故事默认是 contract-freeze / docs / tests / signoff story。只有在冻结现状时发现 runtime 与 DTO/OpenAPI 不一致，才做最小代码修补；不要顺手扩展大块 `web/default` Clients 产品化 UI。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP-6 Implementation Handoff`; `_bmad-output/planning-artifacts/ux-agent-platform.md#3.6 AP-6 Clients onboarding`]

### Existing Runtime Behavior to Preserve

- 当前 Clients API surface 已存在：
  - `GET /api/agent-platform/clients`
  - `POST /api/agent-platform/clients`
  - `GET /api/agent-platform/clients/:id`
  - `PUT /api/agent-platform/clients/:id`
  [Source: `controller/agentplatform/client.go`; `controller/agentplatform/client_test.go`]
- `service/agentplatform/client.go` 当前行为要点：
  - `slug` / `display_name` / `client_type` 缺失会直接视为 invalid input
  - `contract_version` 或 `capabilities` 缺失时，若未显式设置状态，则自动进入 `invalid_integration`
  - `extensions` 必须是 namespaced；plain keys 会触发 `ErrInvalidClientInput`
  - `allow_client_credentials` 是显式布尔治理字段
  [Source: `service/agentplatform/client.go`; `service/agentplatform/client_test.go`]
- `dto/agentplatform/client.go` 已定义当前对外字段形状，应作为 schema freeze 的 DTO source of truth。[Source: `dto/agentplatform/client.go`]

### Architecture Guardrails

- AP-6 仍然遵循三件套 source of truth：`docs/agent-platform-downstream-contract-spec.md`、`docs/openapi/api.json`、mock / fixture / conformance artifacts。三者不一致时本故事不得 done。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP6-AD-3`]
- Clients registration 是 Agent Platform control plane 范畴，不得引入平行协议主干，也不得把 `/api/models`、relay surface 或 Codex channel 逻辑混入本故事。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP-6 Architectural Decisions`; `AGENTS.md#Architecture`]
- 不修改 `service/codex_*`、`controller/codex_*`、`relay/**`、`/v1/**` 或 `docs/openapi/relay.json`。Codex second-consumer 只复用相同 client schema，不需要专门的 client object model。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP6-AD-6`; `AGENTS.md#Rule 5: Protected Project Information`]

### Files to Read Before Editing

- `dto/agentplatform/client.go`
- `controller/agentplatform/client.go`
- `service/agentplatform/client.go`
- `service/agentplatform/client_test.go`
- `controller/agentplatform/client_test.go`
- `docs/agent-platform-downstream-contract-spec.md`
- `docs/agent-platform-consumer-signoff.md`
- `docs/openapi/api.json`
- `_bmad-output/planning-artifacts/ux-agent-platform.md` sections 3.6 / 3.8

### Previous Story Intelligence

- 6.1 已冻结 OAuth/token/revoke/callback/allowlist 子域，并把 Cherry Studio OAuth 子域推进到 `signed off`，但整体 first-consumer 仍然 blocked。6.2 必须衔接这一事实，继续冻结 registration/onboarding，而不是回退去重写 OAuth 契约。[Source: `_bmad-output/implementation-artifacts/ap-6-1-freeze-oauth-token-revoke-and-callback-wire-contract.md`]
- 5.4 已把 `web/default` Agent Platform shell 收口成真实控制面页面，但并没有完整 Clients 工作区，因此 6.2 需要清晰区分“当前 API / fixture 支撑的最小 onboarding”与“后续 UI productization”。[Source: `_bmad-output/implementation-artifacts/ap-5-4-stabilize-agent-platform-web-default-integration.md`; `_bmad-output/planning-artifacts/ux-agent-platform.md#3.6 AP-6 Clients onboarding`]

### Testing Requirements

- 最小验证：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`
- 若修改 OpenAPI / conformance / signoff 防漂移逻辑：
  - `GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance`
- 若触碰前端 onboarding 状态展示：
  - `cd web/default && bun run typecheck`
  - `cd web/default && bun test src/features/agent-platform/agent-platform.test.tsx`
  - `cd web/default && bun run i18n:sync`

### References

- [Source: `_bmad-output/planning-artifacts/epics-agent-platform.md#Story 6.2: 冻结 Client Registration Schema 与 Onboarding 最小流程`]
- [Source: `_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-4`; `#FR-5`; `#FR-15`; `#SM-1`; `#SM-5`]
- [Source: `_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#5.2 2026-06-02 Correct Course 重处理后的定位`; `#5.3 AP-6 MVP Exit Gate`]
- [Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP-6 Architectural Decisions`; `#AP-6 Implementation Handoff`]
- [Source: `_bmad-output/planning-artifacts/ux-agent-platform.md#3.6 AP-6 Clients onboarding`; `#3.8 下游契约签核视图`]
- [Source: `dto/agentplatform/client.go`; `controller/agentplatform/client.go`; `service/agentplatform/client.go`; `controller/agentplatform/client_test.go`; `service/agentplatform/client_test.go`]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-03 18:00 +0800：Story Automator create-step 多次尝试通过 Codex/Claude 子会话创建 6.2 story artifact，但会话长期停留在上下文聚合阶段，未写出 `_bmad-output/implementation-artifacts/ap-6-2-*.md` 或更新 sprint-status。
- 2026-06-03 18:00 +0800：基于 `epics-agent-platform.md`、AP-6 PRD/addendum、architecture、UX、现有 client controller/service/DTO/tests，以及已完成的 6.1 story，手工恢复生成本 story artifact，避免 create 阶段持续阻塞整个 automator 队列。
- 2026-06-03 18:14 +0800：红阶段补强 client service/controller/conformance 测试后，`go test ./service/agentplatform ./controller/agentplatform` 因 client extension 示例未使用点分 namespaced key 失败，`go test ./tests/agentplatform/conformance` 因 OpenAPI 缺 Clients path/schema 与 signoff 未登记 AP-6.2 冻结失败。
- 2026-06-03 18:17 +0800：目标验证通过：`GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`；`GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance`。
- 2026-06-03 18:17 +0800：全量 `GOCACHE=/private/tmp/go-build-cache go test ./...` 未通过，失败来自当前工作区前置/既有问题：根包缺 `web/classic/dist` embed 产物，`model` 包测试缺 SQLite 表迁移，`relay/channel/claude` 与 `relay/helper` 存在非本故事范围断言失败。

### Completion Notes List

- 手工恢复创建了 AP-6.2 story context artifact，并将目标范围收敛为 client registration schema freeze、onboarding 最小流程、DTO/Controller/Service/OpenAPI 对齐、测试补强与 signoff 资产同步。
- 本 story 默认不要求完整 Clients 工作区产品化 UI；若需 UI，仅要求明确 onboarding 哪些仍由 API / fixture 支撑。
- `docs/agent-platform-downstream-contract-spec.md` 已将 Client Registration Contract 从 pending 改为 frozen，并冻结 `client_id`、`slug`、`display_name`、`client_type`、`status`、`allowed_grant_types`、`redirect_uris`、`allowed_scopes`、`contract_version`、`capabilities`、`extensions`、`allow_client_credentials` 的等级、语义、默认值、示例与校验要求。
- 最小 onboarding checklist 已明确 registration、redirect/callback、allowed scopes、contract version、capabilities、consent/OAuth prerequisites、mock fixture 的当前支撑面；AP-6.2 明确只由 API / fixture / signoff 文档支撑，不声称已完成完整 `web/default` Clients 自助工作区。
- OpenAPI 原缺少现有 Clients API path/schema，本故事补齐 `GET/POST /api/agent-platform/clients` 与 `GET/PUT /api/agent-platform/clients/{id}`，并增加 `AgentPlatformClientCreateRequest`、`AgentPlatformClientUpdateRequest`、`AgentPlatformClientItem`、`AgentPlatformClientListResponse`。
- Runtime 只做最小契约修补：Clients service 现在拒绝非数组 grant/redirect/scope、空字符串数组项和未知 client status；未重写 Clients 业务流程或引入新 UI/新 client object model。
- `docs/agent-platform-consumer-signoff.md` 已将 Client registration / onboarding 子域标为 signed off，同时保留 Cherry Studio 整体仍因 AP-6.5/AP-6.6 blocked；Codex second-consumer 继续复用同一 registration schema。

### File List

- `_bmad-output/implementation-artifacts/ap-6-2-freeze-client-registration-schema-and-onboarding-flow.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/planning-artifacts/ux-agent-platform.md`
- `controller/agentplatform/client_test.go`
- `docs/agent-platform-consumer-signoff.md`
- `docs/agent-platform-downstream-contract-spec.md`
- `docs/openapi/api.json`
- `service/agentplatform/client.go`
- `service/agentplatform/client_test.go`
- `tests/agentplatform/conformance/fixtures_test.go`

## Change Log

- 2026-06-03：创建 Story 6.2 context artifact，状态设为 `ready-for-dev`。
- 2026-06-03：冻结 AP-6.2 client registration schema 与 onboarding 最小流程，补齐 Clients OpenAPI schema、runtime 最小校验、测试覆盖、signoff/UX 边界说明，状态推进到 `done`。
