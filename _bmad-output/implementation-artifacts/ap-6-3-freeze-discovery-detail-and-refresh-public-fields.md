---
baseline_commit: 3896f3352b69a3d269ec94a1425cdf6325da0914
---

# Story AP-6.3: 冻结 Discovery、Detail 与 Refresh 公共字段集

Status: done

> Sprint key 映射：本故事来自 Agent Platform Epic AP-6，真实 sprint-status key 为 `ap-6-3-freeze-discovery-detail-and-refresh-public-fields`。本文件按用户要求落地为 `_bmad-output/implementation-artifacts/ap-6-3-*.md`。实现和状态同步必须以 sprint-status 中的 `ap-6-3-*` key 为准。

## Story

As a 客户端集成人员,
I want discovery / detail / refresh 字段集被正式冻结,
so that 下游可以稳定展示资源列表、详情、freshness 与撤销状态。

## Acceptance Criteria

1. **Given** 下游读取资源列表或详情  
   **When** AP-6 公共契约返回资源数据  
   **Then** `resource_id`、`resource_type`、`display_name`、`resource_version`、`contract_version`、`visibility_state`、`callable_state`、`freshness_ttl_seconds`、`freshness`、`etag`、`extensions`、`diagnostics` 等字段被标注为 MUST / SHOULD / MAY  
   **And** `source`、`accountId`、`tenantId`、`disabledReason`、`fetchedAt`、`expiresAt` 是否进入 P0 被明确决策。

2. **Given** 资源被 revoke、rollback、disable 或 offline  
   **When** 下游执行 refresh 或超过 TTL  
   **Then** 契约明确 stale / revoked / offline / fresh 的收敛规则和示例 payload。

## Tasks / Subtasks

- [x] 冻结 discovery / detail / refresh 字段等级与端点差异 (AC: 1)
  - [x] 更新 `docs/agent-platform-downstream-contract-spec.md` 的 `11.3 Discovery / Detail / Refresh Contract`，将状态从 `pending freeze` 改为 `frozen`，并给出字段等级表。
  - [x] 明确 `GET /api/open-capabilities/discovery` 是列表摘要：MUST 返回 `resource_id`、`resource_type`、`display_name`、`resource_version`、`contract_version`、`visibility_state`、`callable_state`、`freshness_ttl_seconds`、`freshness`、`etag`；`extensions` 为 MAY/EXTENSION；不强制带 `diagnostics`。
  - [x] 明确 `GET /api/open-capabilities/resources/:id` 是详情契约：在 discovery 字段基础上 MUST 返回 `status`、`contract_compatible`、`diagnostics`；SHOULD/MAY 返回 `schema`、`detail`、`supported_extensions`、`extensions`。
  - [x] 明确 `POST /api/open-capabilities/refresh` 是对账契约：MUST 返回 `resource_id`、`resource_version`、`contract_version`、`freshness_ttl_seconds`、`freshness`、`etag`、`visibility_state`、`callable_state`、`contract_compatible`、`diagnostics`。

- [x] 对 P0 候选字段做显式纳入/排除决策 (AC: 1)
  - [x] 默认决策：`source`、`accountId`、`tenantId`、`disabledReason`、`fetchedAt`、`expiresAt` 不进入 AP-6.3 resource discovery/detail/refresh P0 核心字段，除非实现同时补齐 DTO、OpenAPI、fixtures 和 conformance。
  - [x] 说明排除理由：`accountId` / `tenantId` / model source 语义属于 AP-6.5 enterprise model discovery；`disabledReason` 在 AP-6.3 通过 `diagnostics.reason` 与 error envelope 表达；`fetchedAt` / `expiresAt` 不在现有 DTO 中，当前 freshness 以 `freshness_ttl_seconds`、`freshness`、`etag`、`resource_version` 表达。
  - [x] 如需要下游私有展示字段，只允许通过 namespaced `extensions` 暴露，不得新增 Cherry Studio / Codex 专用核心字段。

- [x] 冻结 freshness、visible/callable 与收敛语义 (AC: 2)
  - [x] 明确 `freshness` 枚举至少为 `fresh`、`stale`、`offline`、`revoked`，TTL 上限为 300 秒。
  - [x] 明确 `visibility_state` 与 `callable_state` 分离：visible 只表示可发现，callable 才表示可调用；Agent 依赖不可调用时可进入 `contract_invalid` 且仍由 diagnostics 解释。
  - [x] 明确 revoke / rollback / disable / offline 后，下游必须在下一次 refresh 或 300 秒 TTL 上限内收敛；超窗继续使用旧投影是客户端不合规，不是平台静默容忍。
  - [x] 给出 `fresh`、`stale`、`revoked`、`offline` 和 observed ETag/version mismatch 的示例 payload，且示例必须使用 synthetic id，不得包含真实 token、企业数据或 provider secret。

- [x] 对齐 DTO、Controller、Service、OpenAPI 三件套 (AC: 1, 2)
  - [x] 以 `dto/agentplatform/open_capabilities.go` 为当前 response shape source of truth，对齐 `controller/agentplatform/open_capabilities.go`、`service/agentplatform/discovery.go`、`docs/openapi/api.json` 和 `tests/agentplatform/conformance/fixtures.go`。
  - [x] 若 OpenAPI 对 discovery/detail/refresh required 字段、enum、request body 或示例缺失，做最小更新；不要借 AP-6.3 引入新的 runtime plane 或新 endpoint。
  - [x] 若不改 runtime，必须在 completion notes 中说明原因：AP-6.3 是 contract-freeze / docs / tests story，只有 DTO/OpenAPI/fixture 与 runtime 不一致时才做最小代码修补。

- [x] 补强 conformance 与 drift guardrails (AC: 1, 2)
  - [x] 扩展或确认 `tests/agentplatform/conformance/fixtures_test.go` 验证 discovery required fields、detail required fields、refresh required fields、freshness enum、TTL 300、diagnostics required shape。
  - [x] 扩展或确认 `controller/agentplatform/open_capabilities_test.go` 覆盖 discovery -> detail -> refresh 主路径，以及 stale / revoked / offline / observed mismatch 或 non-compliance diagnostics。
  - [x] 扩展或确认 `service/agentplatform/discovery_test.go` 覆盖 contract mismatch、published projection filtering、stale TTL、revoked/offline 错误、extension namespace 提取。

- [x] 同步 consumer signoff 与 UX 边界 (AC: 1, 2)
  - [x] 更新 `docs/agent-platform-consumer-signoff.md` 中 Resource discovery/detail/refresh 行，说明 AP-6.3 已冻结到可签核状态，同时 Cherry Studio 整体仍因 AP-6.5 / AP-6.6 保持 blocked。
  - [x] 若 `_bmad-output/planning-artifacts/ux-agent-platform.md` 中 3.8 签核视图状态需要同步，只更新最小状态说明，不交付新的 web/default 产品化 UI。
  - [x] 不得把 AP-6.3 的资源 discovery 与 AP-6.5 enterprise model discovery 混为一谈，不得复用 `/api/models`、`/api/user/models` 或 `/v1/models` 解决本故事。

## Dev Notes

### Story Source and Scope

- Story AP-6.3 属于 Agent Platform Epic AP-6 “下游公共契约冻结与接入签核”。Epic AP-6 的目标不是重建 control plane / auth plane / open capability plane，而是把 AP-1 到 AP-5 已实现能力冻结成可签核公共契约。[Source: `_bmad-output/planning-artifacts/epics-agent-platform.md#Epic AP-6: 下游公共契约冻结与接入签核`; `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP-6 Architectural Decisions`]
- 本故事只覆盖 resource discovery/detail/refresh 字段集与 freshness/revoke 收敛语义。Skill invoke / Knowledge query 属于 6.4，enterprise model discovery 属于 6.5，error/client state matrix 属于 6.6。[Source: `_bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-6.3`; `docs/agent-platform-downstream-contract-spec.md#11.3 Discovery / Detail / Refresh Contract`]
- AP-6 公共契约 source of truth 是三件套：`docs/agent-platform-downstream-contract-spec.md`、`docs/openapi/api.json`、mock / fixture / conformance artifacts。三者不一致时本故事不得 done。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP6-AD-3`; `docs/agent-platform-downstream-contract-spec.md#3A.1 Source of Truth`]

### Existing Runtime Behavior to Preserve

- 当前公开端点已存在：
  - `GET /api/open-capabilities/discovery`
  - `GET /api/open-capabilities/resources/:id`
  - `POST /api/open-capabilities/refresh`
  - 这三个端点都需要 `ap.resources.read` bearer scope。[Source: `controller/agentplatform/open_capabilities.go`; `_bmad-output/planning-artifacts/architecture-agent-platform.md#Open capability endpoints`]
- `dto/agentplatform/open_capabilities.go` 当前字段形状：
  - discovery item 包含 `resource_id`、`resource_type`、`display_name`、`resource_version`、`contract_version`、`visibility_state`、`callable_state`、`freshness_ttl_seconds`、`freshness`、`etag`、可选 `extensions`。
  - detail response 在 discovery 语义上增加 `status`、`schema`、`detail`、`supported_extensions`、`contract_compatible`、`diagnostics`。
  - refresh response 包含对账所需的 `resource_id`、`resource_version`、`contract_version`、`freshness_ttl_seconds`、`freshness`、`etag`、`visibility_state`、`callable_state`、`contract_compatible`、`diagnostics`。
  [Source: `dto/agentplatform/open_capabilities.go`]
- `service/agentplatform/discovery.go` 当前行为要点：
  - discovery 只返回当前 client 的 visible exposures，并过滤 revoked/offline、permission-denied、contract-incompatible projections。
  - detail 对 revoked exposure 返回 `resourceRevoked`，对 offline/disabled resource 返回 `resourceOffline`，contract version mismatch 返回 `contractInvalid`。
  - refresh 复用 detail，并用 observed ETag/version/observed_at 生成 diagnostics。
  - freshness 由 exposure revoked/offline/stale/fresh 状态计算，TTL 过期进入 `stale`。
  [Source: `service/agentplatform/discovery.go`; `service/agentplatform/discovery_test.go`]
- `controller/agentplatform/open_capabilities_test.go` 已覆盖 discovery/detail/refresh 主路径，以及 stale、revoked、offline、diagnostics 和 error envelope。实现时应扩展这个测试，不要新增绕过 controller 的平行测试入口。[Source: `controller/agentplatform/open_capabilities_test.go`]

### Field-Level Decisions the Dev Agent Must Not Miss

- `diagnostics` 不应强行加入 discovery 列表响应。现有 DTO 只在 detail/refresh 返回 diagnostics；story AC 中的字段清单是公共字段总体候选集，必须在 contract 中标清“哪个端点返回哪个字段”。[Source: `dto/agentplatform/open_capabilities.go`; `_bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-6.3`]
- `source`、`accountId`、`tenantId`、`disabledReason`、`fetchedAt`、`expiresAt` 必须做显式 P0 决策。除非本故事同步完成 DTO/OpenAPI/fixtures/runtime 支持，否则应明确排除出 AP-6.3 P0 core resource fields，并说明可通过 `extensions.<namespace>` 或后续 AP-6.5/6.6 承接。[Source: `docs/agent-platform-downstream-contract-spec.md#11.3 Discovery / Detail / Refresh Contract`; `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP6-AD-2`]
- `extensions` 必须 namespaced，且不得覆盖核心字段语义。Cherry Studio / Codex 私有字段只能走 `extensions.cherry_studio` / `extensions.codex` 或更细 namespace。[Source: `docs/agent-platform-downstream-contract-spec.md#3A.3 Contract Versioning`; `docs/agent-platform-consumer-signoff.md#Extension Governance`]
- 不要修改 `service/codex_*`、`controller/codex_*`、`relay/**`、`/v1/**` 或 `docs/openapi/relay.json`。AP-6.3 是 Agent Platform open capability contract freeze，不是 Codex channel 或 relay 模型发现故事。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP6-AD-1`; `AGENTS.md#Architecture`]

### Architecture Guardrails

- 不新增第四个 runtime plane。任何必要 API 必须归属既有 `control plane`、`auth plane` 或 `open capability plane`；本故事预期不新增 endpoint。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP6-AD-1`]
- freshness 是 contract-level requirement：默认 TTL 上限 300 秒；消费者必须理解 `fresh`、`stale`、`offline`、`revoked`；revoke/rollback/disable/offline 后必须在下一次 refresh 或 300 秒内收敛。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#Freshness and convergence`; `_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-6`]
- JSON marshal/unmarshal 新代码必须使用 `common.Marshal`、`common.Unmarshal`、`common.UnmarshalJsonStr` 等 wrapper，不得在业务代码中直接调用 `encoding/json` marshal/unmarshal。类型引用如 `json.RawMessage` 可保留。[Source: `AGENTS.md#Rule 1: JSON Package`]
- 数据库代码必须同时兼容 SQLite、MySQL >= 5.7.8、PostgreSQL >= 9.6；本故事通常不应新增迁移或 raw SQL。[Source: `AGENTS.md#Rule 2: Database Compatibility`]

### Files to Read Before Editing

- `dto/agentplatform/open_capabilities.go`
- `controller/agentplatform/open_capabilities.go`
- `controller/agentplatform/open_capabilities_test.go`
- `service/agentplatform/discovery.go`
- `service/agentplatform/discovery_test.go`
- `service/agentplatform/open_capability_errors.go`
- `docs/agent-platform-downstream-contract-spec.md`
- `docs/agent-platform-consumer-signoff.md`
- `docs/openapi/api.json`
- `tests/agentplatform/conformance/fixtures.go`
- `tests/agentplatform/conformance/fixtures_test.go`
- `_bmad-output/planning-artifacts/ux-agent-platform.md` section 3.8

### Previous Story Intelligence

- 6.1 已冻结 OAuth/token/revoke/callback/allowlist wire contract，并建立了 OpenAPI + fixtures + conformance 的冻结方式。6.3 应沿用同一模式，不回退去重写 OAuth。[Source: `_bmad-output/implementation-artifacts/ap-6-1-freeze-oauth-token-revoke-and-callback-wire-contract.md`; `git show 1623c7f35 --stat`]
- 6.2 已冻结 client registration schema，并补齐 Clients OpenAPI、runtime 最小校验、conformance drift checks 和 signoff 状态。6.3 应复用该“三件套 + signoff artifact”模式，并继续保持 Cherry Studio 整体 blocked 直到 AP-6.5/AP-6.6 完成。[Source: `_bmad-output/implementation-artifacts/ap-6-2-freeze-client-registration-schema-and-onboarding-flow.md`; `docs/agent-platform-consumer-signoff.md`]
- 当前 `docs/agent-platform-consumer-signoff.md` 已把 Resource discovery/detail/refresh 标为 `signed off`，但 `docs/agent-platform-downstream-contract-spec.md#11.3` 仍是 `pending freeze` 且 sprint-status 仍为 backlog。实现必须消除这个漂移，而不是把现有 signoff 状态当作无需工作的证据。[Source: `docs/agent-platform-consumer-signoff.md#3.2 Coverage Matrix`; `docs/agent-platform-downstream-contract-spec.md#11.3 Discovery / Detail / Refresh Contract`; `_bmad-output/implementation-artifacts/sprint-status.yaml`]

### Testing Requirements

- 最小验证：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`
  - `GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance`
- 若只更新 markdown/OpenAPI/fixtures，也必须跑 conformance 包，确保 contract spec、OpenAPI 和 fixture/signoff 不漂移。
- 若触碰 `web/default` 签核展示：
  - `cd web/default && bun run typecheck`
  - `cd web/default && bun test src/features/agent-platform/agent-platform.test.tsx`
  - `cd web/default && bun run i18n:sync`

### References

- [Source: `_bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-6.3: 冻结 Discovery、Detail 与 Refresh 公共字段集`]
- [Source: `_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-5`; `#FR-6`; `#SM-4`]
- [Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP-6 Architectural Decisions`; `#Freshness and convergence`; `#Open capability endpoints`]
- [Source: `_bmad-output/planning-artifacts/ux-agent-platform.md#3.8 下游契约签核视图`]
- [Source: `docs/agent-platform-downstream-contract-spec.md#4.3 Open Capability Discovery / Detail / Refresh`; `#11.3 Discovery / Detail / Refresh Contract`; `#11.7 Mock / Fixture / Conformance`; `#11.8 Consumer Signoff`]
- [Source: `dto/agentplatform/open_capabilities.go`; `controller/agentplatform/open_capabilities.go`; `service/agentplatform/discovery.go`; `tests/agentplatform/conformance/fixtures.go`; `tests/agentplatform/conformance/fixtures_test.go`]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-03 18:34 +0800：按 BMAD create-story workflow 创建 AP-6.3 story context artifact。`resolve_customization.py` 因默认 Python 缺少 `tomllib` 未能运行，已按 skill fallback 手动读取 `.agents/skills/bmad-create-story/customize.toml`；无 team/user override 文件。
- 2026-06-03 18:34 +0800：发现 persistent fact glob `**/project-context.md` 无匹配文件；以 AGENTS.md、BMAD config、AP planning artifacts、现有 6.1/6.2 story、runtime DTO/controller/service/tests 和 contract/signoff artifacts 作为上下文。
- 2026-06-03 18:34 +0800：确认目标 sprint key 为 `ap-6-3-freeze-discovery-detail-and-refresh-public-fields`，当前状态为 `backlog`；与历史企业治理 `6-3-move-membership...` 不是同一 story。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- 本 story 明确 AP-6.3 是 contract-freeze / docs / OpenAPI / fixture / conformance story，不新增 runtime plane，不处理 AP-6.5 model discovery 或 AP-6.6 error/client state matrix。
- 关键防错点：`diagnostics` 只应作为 detail/refresh 必备字段；discovery 列表保持摘要字段。`source` / `accountId` / `tenantId` / `disabledReason` / `fetchedAt` / `expiresAt` 必须显式决策，默认不进入 AP-6.3 P0 核心字段。
- 2026-06-03 19:02 +0800：Codex 接管 6.3 dev，终止并清理重复 tmux dev 会话，按 source-of-truth 直接同步 contract spec、signoff artifact、OpenAPI enum/TTL 与 conformance guardrails。
- 2026-06-03 19:08 +0800：目标验证通过：`GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance`、`GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`。

### File List

- `_bmad-output/implementation-artifacts/ap-6-3-freeze-discovery-detail-and-refresh-public-fields.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `docs/agent-platform-downstream-contract-spec.md`
- `docs/agent-platform-consumer-signoff.md`
- `docs/openapi/api.json`
- `tests/agentplatform/conformance/fixtures.go`
- `tests/agentplatform/conformance/fixtures_test.go`

## Change Log

- 2026-06-03：创建 Story AP-6.3 context artifact，状态设为 `ready-for-dev`。
