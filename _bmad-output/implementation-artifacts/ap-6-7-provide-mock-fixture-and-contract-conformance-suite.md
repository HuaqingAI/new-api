---
baseline_commit: 0a90354a060d0747d3234452cb5a0334c507446c
---

# Story AP-6.7: 提供 Mock、Fixture 与 Contract Conformance 套件

Status: done

> Sprint key 映射：本故事来自 Agent Platform Epic AP-6，规划与 sprint-status key 为 `ap-6-7-provide-mock-fixture-and-contract-conformance-suite`。本文档按 AP 命名空间规范落地为 `ap-6-7-*` 文件名；实现和状态同步以 `ap-6-7-*` 为真实 sprint key。

## Story

As a 下游集成方,
I want 获得 mock / fixture / conformance 测试资产,
so that 我可以在真实环境前验证 OAuth、discovery、detail、refresh、Skill invoke、Knowledge query 和 model discovery。

## Acceptance Criteria

1. **Given** 下游开发者未接入真实平台环境  
   **When** 使用 AP-6 fixture  
   **Then** 可以获得成功、撤销、下线、stale、contract mismatch、permission denied、provider failure、model disabled 等代表性 payload  
   **And** fixture 不包含真实 token、provider secret 或敏感租户数据。

2. **Given** 下游完成实现  
   **When** 运行 conformance tests  
   **Then** 可以验证字段解析、错误映射、状态矩阵、TTL/refresh 收敛和 model discovery 默认模型处理。

## Tasks / Subtasks

- [x] 建立 AP-6 fixture 与 mock 资产位置，并复用真实 DTO / handler 行为 (AC: 1)
  - [x] 在仓库内新增清晰的 AP-6 fixture/conformance 目录，建议优先使用 `tests/agentplatform/conformance/` 或 `tests/fixtures/agentplatform/`；不要放入 `relay/**`、`web/classic/**` 或临时脚本目录。
  - [x] fixture payload 必须从当前 `dto/agentplatform/*`、`service/agentplatform/*` 和 OpenAPI 契约派生，不允许手写一套与真实返回字段漂移的平行 JSON。
  - [x] 使用 synthetic IDs，例如 `client_cherry_mock`、`tenant_demo`、`acct_demo`、`res_skill_demo`、`model_demo_default`；禁止真实 access token、refresh token、provider secret、tenant secret、真实用户信息或真实企业数据。

- [x] 覆盖开放能力和 OAuth 的代表性成功/失败 payload (AC: 1, 2)
  - [x] OAuth fixture 至少覆盖 authorize/token 成功、expired token、revoked grant/token、missing scope 或 permission denied。
  - [x] discovery fixture 至少覆盖 empty、success、多资源类型、contract mismatch 被过滤或拒绝的行为。
  - [x] detail fixture 至少覆盖 visible + callable、visible but not callable、contract invalid、resource revoked、resource offline。
  - [x] refresh fixture 至少覆盖 fresh、stale、revoked、offline、observed etag/version mismatch、TTL 超 300 秒后的 non-compliance diagnostics。
  - [x] Skill invoke fixture 至少覆盖 sync success、contract invalid、timeout、upstream/provider failure。
  - [x] Knowledge query fixture 至少覆盖 retrieval success、items/citations、provider offline/upstream failure、provider-native 字段不外泄。

- [x] 覆盖 enterprise model discovery fixture 与默认模型状态 (AC: 1, 2)
  - [x] 先确认 AP-6.5 已冻结 model discovery 公共契约；若尚未实现，conformance 应以 `pending/skip with reason` 记录，不得发明最终字段。
  - [x] fixture 至少覆盖 default model、no default model、multiple default models、default disabled、provider offline、model unavailable、account / tenant mismatch。
  - [x] 明确 `modelId`、`providerStableId`、`displayName`、`isDefault`、`status`、`disabledReason`、`capabilities`、`accountId` / `tenantId` 来源语义。
  - [x] 不得把 `/api/models`、`/api/user/models` 或 `/v1/models` 的响应直接伪装成 enterprise model discovery 签核结果。

- [x] 实现 contract conformance 自动化测试入口 (AC: 2)
  - [x] 提供可由 CI 或本地运行的 Go 测试入口，优先复用 `controller/agentplatform/open_capabilities_test.go` 中的 in-memory SQLite setup、bearer token、stub HTTP client 和 stub knowledge provider 模式。
  - [x] conformance 至少断言字段解析、required field presence、error envelope shape、错误码到客户端状态矩阵映射、freshness / ETag / TTL 收敛、model discovery 默认模型处理。
  - [x] 对 AP-6.6 尚未冻结的错误/状态矩阵，测试必须依赖 AP-6.6 输出的矩阵 artifact；若矩阵缺失，本故事实现应先补充阻塞说明或将相关 conformance case 标记为显式 pending。
  - [x] 测试输出应能告诉下游“哪个契约项失败”，不要只返回 generic test failed。

- [x] 同步三件套 source of truth (AC: 1, 2)
  - [x] 更新 `docs/agent-platform-downstream-contract-spec.md` 的 11.7，列出 fixture/conformance 目录、运行方式、覆盖矩阵和安全约束。
  - [x] 更新 `docs/openapi/api.json`，确保 mock / conformance 覆盖的公开接口和 schema 与 OpenAPI 一致；若只新增测试资产而不新增 API，也要确认现有 `/api/open-capabilities/**` schema 未漂移。
  - [x] 将 fixture/conformance 文件纳入 AP-6 source-of-truth 清单；三者不一致时，本故事不得标记完成。

## Dev Notes

### Story Source and Scope

- Story AP-6.7 属于 Agent Platform Epic AP-6 “下游公共契约冻结与接入签核”。AP-6 的目标不是重建 control plane / auth plane / open capability plane，而是把 AP-1 到 AP-5 已完成能力冻结成下游可签核公共契约，并补齐 model discovery、错误/状态矩阵、mock fixture、conformance tests 与 onboarding 缺口。[Source: `_bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-6.7`; `_bmad-output/planning-artifacts/architecture-agent-platform.md#V1.4 Architecture Amendment: AP-6 Public Contract Freeze`]
- 本故事不新增第四个 runtime plane。所有实现必须归属既有三类 surface：control plane `/api/agent-platform/**`、auth plane `/api/agent-platform/oauth/**`、open capability plane `/api/open-capabilities/**`。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP6-AD-1`]
- 本故事优先交付 fixture、mock 和 conformance 资产；只有在契约验证暴露当前代码缺口时，才做最小代码补齐。不要借此重写 OAuth、open capability、Skill、Knowledge、Agent 或 relay 实现。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP-6 Implementation Handoff`]

### Hard Dependencies

- AP-6.5 model discovery 契约是 model discovery fixture/conformance 的前置。当前 `docs/agent-platform-downstream-contract-spec.md` 明确 11.5 状态为 missing / must add，不能用现有 `/api/models`、`/api/user/models`、`/v1/models` 替代。[Source: `docs/agent-platform-downstream-contract-spec.md#11.5 Enterprise Model Discovery Contract`]
- AP-6.6 错误码矩阵与客户端状态矩阵是 conformance 错误映射测试的前置。当前 sprint-status 列出 `ap-6-6-freeze-error-code-and-client-state-matrices: backlog` 且未找到对应 story artifact；实现时必须先确认矩阵 artifact 是否已存在。[Source: `_bmad-output/implementation-artifacts/sprint-status.yaml`; `docs/agent-platform-downstream-contract-spec.md#11.6 Error Matrix and Client State Matrix`]
- AP-6 公共契约 source of truth 是三件套：`docs/agent-platform-downstream-contract-spec.md`、`docs/openapi/api.json`、mock / fixture / conformance test artifacts。三者不一致时 AP-6 story 不得 done。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP6-AD-3`; `docs/agent-platform-downstream-contract-spec.md#3A.1 Source of Truth`]

### Existing Code to Reuse

- Open capability DTO 当前在 `dto/agentplatform/open_capabilities.go`，字段包括 `resource_id`、`resource_type`、`display_name`、`resource_version`、`contract_version`、`visibility_state`、`callable_state`、`freshness_ttl_seconds`、`freshness`、`etag`、`extensions`、`diagnostics` 等。Fixture 必须覆盖这些真实字段，不要改成 camelCase 或消费者私有命名。[Source: `dto/agentplatform/open_capabilities.go`]
- 统一错误 envelope 当前在 `service/agentplatform/open_capability_errors.go`，错误码常量包括 `permissionDenied`、`resourceRevoked`、`resourceOffline`、`quotaOrRateLimited`、`timeout`、`upstreamFailed`、`contractInvalid`。Conformance 应直接断言这些常量，不要复制字符串到独立实现里。[Source: `service/agentplatform/open_capability_errors.go`]
- `controller/agentplatform/open_capabilities_test.go` 已有高价值 setup：in-memory SQLite、`apmodel.Migrate`、synthetic Cherry Studio client、published Skill/Knowledge/Agent、client_credentials token、stub skill HTTP client、stub knowledge provider，以及 discovery/detail/refresh/invoke/query/agent detail 测试。优先抽取或复用这些模式来生成 conformance，而不是另建不可维护的 mock server。[Source: `controller/agentplatform/open_capabilities_test.go`]
- `service/agentplatform/discovery.go` 已实现 freshness 常量 `fresh`、`stale`、`offline`、`revoked` 和 diagnostics 计算。TTL/refresh conformance 应以这里的行为为准。[Source: `service/agentplatform/discovery.go`]
- 当前 OpenAPI 已包含 `/api/agent-platform/resources` 和 `/api/open-capabilities/discovery`、`/resources/{id}`、`/refresh`、`/skills/{id}/invoke`、`/knowledge-bases/{id}/query`、`/agents/{id}` 等路径。实现时必须检查 schema 与 fixture 是否一致。[Source: `docs/openapi/api.json`]

### Security and Data Hygiene

- Fixture 只允许 synthetic data，不允许真实 token、provider secret、tenant secret、真实用户信息或真实企业数据。OAuth signing key、refresh token、provider secret 不得进入浏览器 payload、日志、fixture 或审计样例。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#Secret handling`; `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP6-AD-5`]
- 所有 JSON marshal/unmarshal 代码遵守项目规则：业务代码使用 `common.Marshal` / `common.Unmarshal` / `common.DecodeJson` 等 wrapper；可以引用 `encoding/json` 的 `RawMessage` 类型，但不要直接调用标准库 marshal/unmarshal。[Source: `AGENTS.md#Rule 1: JSON Package`; `dto/agentplatform/open_capabilities.go` 当前仅作为类型引用使用 `json.RawMessage`]
- 本故事不修改 `relay/**`、`/v1/**` 或 `docs/openapi/relay.json`。Enterprise model discovery 是 Agent Platform 下游公共投影，不是 relay model list 的重命名。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP-6 Implementation Handoff`; `_bmad-output/planning-artifacts/architecture-agent-platform.md#AP6-AD-2`]

### Project Structure Notes

- 后端仍遵循 `Router -> Controller -> Service -> Model`。Agent Platform 相关业务代码继续放在 `router/agentplatform-router.go`、`router/open_capabilities_router.go`、`controller/agentplatform/`、`service/agentplatform/`、`model/agentplatform/`、`dto/agentplatform/`。[Source: `AGENTS.md#Architecture`; `_bmad-output/planning-artifacts/architecture-agent-platform.md#Complete Project Directory Structure`]
- 测试组织建议：已有 first-wave 目标包括 `service/agentplatform/discovery_test.go`、`service/agentplatform/knowledge_query_test.go`、`controller/agentplatform/*_test.go`。本故事的 conformance 可以新增 dedicated package，但应复用这些领域 service/controller，而不是把测试写成纯静态 JSON 比对。[Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#Testing organization`]
- 若新增文档或 fixture 索引，优先放在 `docs/agent-platform-*` 或 `tests/**` 下；不要把签核资产散落到 `_bmad-output` 作为运行时依赖。`_bmad-output` 只记录 story/context，不应成为测试套件的代码入口。

### Previous Story Intelligence

- Agent Platform Epic AP-1 retro 指出自动化最容易在 create/dev 早期漂移回旧 enterprise 域。实现本故事时必须先锁定 Agent Platform 文件树和 AP-6 source-of-truth，不要泛化到 enterprise governance 或 relay 代码。[Source: `_bmad-output/implementation-artifacts/ap-epic-1-retro-2026-06-01.md#Significant Discoveries`]
- Epic AP-1 retro 还要求后续故事复用 `visible` / `callable`、projection / exposure、resource version 与 lifecycle 词汇，不允许另起近义模型。Fixture 和 conformance 的状态命名必须沿用这些词汇。[Source: `_bmad-output/implementation-artifacts/ap-epic-1-retro-2026-06-01.md#Team Agreements`]
- 最近相关实现持续使用 `controller/service/dto/model/agentplatform` 和 `web/default/src/features/agent-platform` 作为主实现面。当前 story 不应把 conformance 逻辑放入 `enterprise` 或 `oauth` 杂项目录。[Source: recent git log: `ap-5-*` Agent Platform stories; `_bmad-output/implementation-artifacts/ap-epic-1-retro-2026-06-01.md`]

### Testing Requirements

- 推荐最小验证命令：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`
  - `GOCACHE=/private/tmp/go-build-cache go test ./tests/...`，如果新增 dedicated conformance package
  - `go test ./...`，若本故事修改共享 DTO、middleware、router 或 OpenAPI 生成/校验逻辑
- 若新增前端签核状态展示，再运行：
  - `cd web/default && bun run typecheck`
  - `cd web/default && bun test src/features/agent-platform/agent-platform.test.tsx`
  - `cd web/default && bun run i18n:sync`
- Conformance 测试必须至少覆盖失败路径；只覆盖 success fixture 不满足 AC。

### References

- [Source: `_bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-6.7: 提供 Mock、Fixture 与 Contract Conformance 套件`]
- [Source: `_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-5`; `#FR-6`; `#FR-15`]
- [Source: `_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#5.2 2026-06-02 Correct Course 重处理后的定位`; `#5.3 AP-6 MVP Exit Gate`]
- [Source: `_bmad-output/planning-artifacts/architecture-agent-platform.md#API & Communication Patterns`; `#AP-6 Architectural Decisions`; `#AP-6 Implementation Handoff`]
- [Source: `_bmad-output/planning-artifacts/ux-agent-platform.md#3.8 下游契约签核视图`; `#3.10 客户端状态矩阵展示`]
- [Source: `docs/agent-platform-downstream-contract-spec.md#11.7 Mock / Fixture / Conformance`]
- [Source: `dto/agentplatform/open_capabilities.go`; `service/agentplatform/open_capability_errors.go`; `service/agentplatform/discovery.go`; `controller/agentplatform/open_capabilities_test.go`]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-03 03:37 +0800：执行 BMAD create-story workflow；已读取 `.claude/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`。
- 2026-06-03 03:37 +0800：`resolve_customization.py` 因 Python `tomllib` 缺失失败，已按技能回退规则手动读取 `.claude/skills/bmad-create-story/customize.toml`；未发现 team/user override。
- 2026-06-03 03:37 +0800：已读取 `_bmad/bmm/config.yaml`，确认 communication/document language 为 Chinese，planning artifacts 和 implementation artifacts 均位于 `_bmad-output`。
- 2026-06-03 03:37 +0800：已读取完整 `sprint-status.yaml`，确认真实 sprint key 为 `ap-6-7-provide-mock-fixture-and-contract-conformance-suite: backlog`。
- 2026-06-03 03:37 +0800：已分析 `epics-agent-platform.md`、Agent Platform PRD/addendum、architecture、UX、downstream contract spec、现有 Agent Platform DTO/service/controller 测试和 OpenAPI 路径。
- 2026-06-03 03:45 +0800：执行 BMAD dev-story workflow；已读取 `.claude/skills/bmad-dev-story/SKILL.md` 和 `checklist.md`。
- 2026-06-03 03:45 +0800：`resolve_customization.py` 因 Python `tomllib` 缺失失败；已按技能回退规则读取 `.claude/skills/bmad-dev-story/customize.toml`，未发现 team/user override，activation prepend/append 为空。
- 2026-06-03 03:45 +0800：已读取 `_bmad/bmm/config.yaml`；communication/document language 为 Chinese，user 为 hth，implementation artifacts 位于 `_bmad-output/implementation-artifacts`。
- 2026-06-03 03:45 +0800：未发现 `project-context.md`；已读取完整 story、完整 sprint-status，并确认真实 sprint key 为 `ap-6-7-provide-mock-fixture-and-contract-conformance-suite`。
- 2026-06-03 03:45 +0800：确认 AP-6.5 model discovery 与 AP-6.6 error/client state matrix 仍为 backlog，未发现对应 artifact；相关 fixture/conformance 必须显式 pending，不发明最终契约。
- 2026-06-03 03:52 +0800：新增 `tests/agentplatform/conformance` typed fixture catalog 与 conformance assertions；fixture 从 Agent Platform DTO 和 service 常量构建，覆盖 synthetic data 安全扫描、required cases、pending reason、OpenAPI path 一致性。
- 2026-06-03 03:52 +0800：扩展 `controller/agentplatform/open_capabilities_test.go`，复用 in-memory SQLite、bearer token、stub HTTP client、stub knowledge provider，覆盖 TTL non-compliance、revoked/offline、skill timeout、knowledge upstream failure 与 provider-native 字段不外泄。
- 2026-06-03 03:52 +0800：更新 `docs/agent-platform-downstream-contract-spec.md` 11.7 和 `docs/openapi/api.json`，使 fixture/conformance/docs/OpenAPI 三件套一致。
- 2026-06-03 03:52 +0800：验证通过：`GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance`、`GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`、`GOCACHE=/private/tmp/go-build-cache go test ./tests/...`。
- 2026-06-03 03:52 +0800：执行全量 `GOCACHE=/private/tmp/go-build-cache go test ./...` 未通过；失败集中在既有无关包：`model` 缺少测试表初始化、`relay/channel/claude` 文件内容转换断言、`relay/helper` stream status 断言。本故事目标包和新增 conformance 测试均通过。
- 2026-06-03 10:01 +0800：执行 BMAD story-automator review workflow；已读取指定 `SKILL.md`、`workflow.yaml`、`instructions.xml`、`checklist.md`，并按自动修复模式处理。
- 2026-06-03 10:01 +0800：Review 发现并修复 3 个问题：fixture 错误用例 HTTP method 元数据漂移、OAuth revoke 缺少 typed response DTO、OpenAPI data schema 未覆盖 OAuth typed response 与 open-capabilities detail/refresh diagnostics 字段。
- 2026-06-03 10:01 +0800：Review 验证通过：`GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance ./service/agentplatform ./controller/agentplatform ./tests/...`。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- Validation checklist applied manually：story 明确 AC、前置依赖、复用文件、禁止范围、安全约束、测试入口和三件套一致性要求。
- AP-6 fixture/conformance 目录已建立在 `tests/agentplatform/conformance/`，包含 typed fixture catalog、synthetic data 安全扫描、coverage assertions、OpenAPI path conformance。
- OAuth、discovery、detail、refresh、Skill invoke、Knowledge query 代表性成功/失败 payload 已覆盖；fixture 使用 DTO 和 service 常量生成，避免平行 JSON 字段漂移。
- AP-6.5 model discovery 和 AP-6.6 error/client state matrix 均仍未冻结；相关 default/no default/disabled/provider/account-tenant/error matrix case 已显式 pending with reason，且禁止以 `/api/models`、`/api/user/models` 或 `/v1/models` 替代。
- Controller handler conformance 已复用现有 in-memory SQLite、bearer token、stub skill HTTP client、stub knowledge provider 模式，覆盖 TTL/ETag/version diagnostics、revoked/offline、timeout/upstream failure 和 provider-native 字段不外泄。
- AP-6 source of truth 三件套已同步：contract spec 11.7、OpenAPI OAuth/public open-capabilities paths、fixture/conformance artifacts。
- Review 自动修复后，conformance 会校验 OAuth/OpenAPI response data schema 必填字段，避免只检查 path 存在导致三件套再次漂移。

### Senior Developer Review (AI)

Reviewer: hth on 2026-06-03 10:01 +0800

Outcome: Approve / done

#### Findings Fixed

- [MEDIUM] `tests/agentplatform/conformance/fixtures.go` 中 discovery/detail/OAuth bearer 失败 fixture 被统一标成 `POST`，与真实 `GET` API surface 不一致。已为 error fixture 增加 method 参数，并修正对应 fixture。
- [MEDIUM] OAuth revoke 成功 payload 在 controller 使用匿名 `gin.H`，fixture/OpenAPI 只能手写响应形状，容易和真实 DTO 漂移。已新增 `dtoagentplatform.OAuthRevokeResponse`，并让 controller 与 fixture 复用该 DTO。
- [MEDIUM] `docs/openapi/api.json` 只新增 OAuth 路径通用 envelope，未把 authorize/token/revoke 的 `data` 绑定到 typed response schema；open-capabilities detail/refresh schema 也缺少真实返回的 `contract_compatible` 与 `diagnostics`。已补齐 OpenAPI schema，并扩展 conformance 测试校验 required data fields 与 `$ref` component。

#### Review Checklist

- [x] Story file loaded from `_bmad-output/implementation-artifacts/ap-6-7-provide-mock-fixture-and-contract-conformance-suite.md`
- [x] Story Status verified as reviewable (`review`)
- [x] Epic and Story IDs resolved (`AP-6.7`, sprint key `ap-6-7-provide-mock-fixture-and-contract-conformance-suite`)
- [x] Story Context located or warning recorded
- [x] Epic Tech Spec located or warning recorded
- [x] Architecture/standards docs loaded as available
- [x] Tech stack detected and documented
- [x] MCP doc search not required; local project artifacts and source files were sufficient
- [x] Acceptance Criteria cross-checked against implementation
- [x] File List reviewed and validated for completeness
- [x] Tests identified and mapped to ACs
- [x] Code quality review performed on changed files
- [x] Security review performed on changed files and fixture data
- [x] Outcome decided: Approve
- [x] Review notes appended under "Senior Developer Review (AI)"
- [x] Change Log updated with review entry
- [x] Status updated to `done`
- [x] Sprint status synced
- [x] Story saved successfully

### File List

- `_bmad-output/implementation-artifacts/ap-6-7-provide-mock-fixture-and-contract-conformance-suite.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `tests/agentplatform/conformance/fixtures.go`
- `tests/agentplatform/conformance/fixtures_test.go`
- `controller/agentplatform/oauth.go`
- `controller/agentplatform/open_capabilities_test.go`
- `docs/agent-platform-downstream-contract-spec.md`
- `docs/openapi/api.json`
- `dto/agentplatform/oauth.go`
- `service/agentplatform/open_capability_errors.go`

## Change Log

- 2026-06-03：创建 Story AP-6.7 context artifact，状态设为 `ready-for-dev`。
- 2026-06-03：实现 AP-6 mock fixture / conformance suite，补齐 frozen OAuth/open-capabilities 覆盖；AP-6.5/AP-6.6 依赖项以 pending reason 固化；更新 OpenAPI 与 downstream contract spec，状态设为 `review`。
- 2026-06-03：完成 story-automator review，自动修复 fixture method、OAuth revoke DTO、OpenAPI response schema / diagnostics 漂移，并将状态设为 `done`。
