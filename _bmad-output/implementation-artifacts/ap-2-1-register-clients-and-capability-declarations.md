# Story 2.1: 建立下游客户端注册与能力声明模型

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 平台管理员,
I want 以统一对象模型注册并维护下游客户端的契约与能力信息,
so that 新消费者接入不再依赖一次性集成记录或特判结构。

## Acceptance Criteria

1. **Given** 某个新的下游消费者需要接入  
   **When** 管理员创建客户端记录  
   **Then** 平台保存稳定 client identity、status、allowed scopes、supported grant types、contract version 与 capability declarations  
   **And** Cherry Studio 可以作为首个消费者被表示，而不需要专属对象形状。
2. **Given** 某客户端配置不完整或处于禁用状态  
   **When** 它尝试进入发布或调用路径  
   **Then** 平台阻止该流程并返回可诊断错误  
   **And** 缺失能力声明会被视为无效集成状态。
3. **Given** 客户端需要保存扩展元数据  
   **When** 这些元数据被存储  
   **Then** 平台允许 namespaced extension 字段  
   **And** 这些扩展字段不能重定义核心契约语义。

## Tasks / Subtasks

- [x] 在 Agent Platform bounded context 中落地正式 client 主模型 (AC: 1, 2, 3)
  - [x] 新增 `model/agentplatform/client.go`，定义 `agent_platform_clients` 表，至少覆盖：稳定 `client_id`、`slug`、`display_name`、`client_type`、`status`、`allowed_grant_types_json`、`redirect_uris_json`、`allowed_scopes_json`、`contract_version`、`capabilities_json`、`extensions_json`、`allow_client_credentials`、`created_at`、`updated_at`。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Client model]
  - [x] `client_id` 必须是稳定 opaque id，不得复用数据库自增主键；`slug` / `display_name` 是管理面可读字段，不能取代 `client_id` 的身份职责。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#860]
  - [x] 在 `model/agentplatform/migration.go` 注册 client 表迁移，并保持 SQLite、MySQL、PostgreSQL 兼容；结构化字段继续走 `TEXT` + `common.Marshal` / `common.UnmarshalJsonStr` 路线。[Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go; AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]

- [x] 实现客户端能力声明与无效集成状态判定 (AC: 1, 2, 3)
  - [x] 在服务层明确哪些字段决定“invalid integration”：至少包括缺失 `contract_version`、缺失 capability declarations、grant types 与 redirect URIs 不匹配等。该状态必须是可诊断业务状态，而不是只靠 UI 侧猜测。[Source: _bmad-output/planning-artifacts/ux-agent-platform.md#3-1-客户端注册]
  - [x] `capabilities_json` 与 `extensions_json` 要求做基础结构校验：capabilities 表达核心契约边界，extensions 只允许 namespaced 元数据，不得重定义 grant/scope/contract 等核心语义。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#205; _bmad-output/planning-artifacts/architecture-agent-platform.md#503-606]
  - [x] 当前故事不实现 OAuth token/grant 颁发，也不做 open-capabilities 调用；这里只需要把 client registration 与 capability declaration 作为 control plane 事实源建好。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.2; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.3]

- [x] 提供最小客户端注册控制面 API (AC: 1, 2, 3)
  - [x] 新增 `dto/agentplatform/client.go`、`service/agentplatform/client.go`、`controller/agentplatform/client.go`（命名可等价），提供最小 CRUD：创建 client、读取单个 client、分页/列表查询、更新 client 配置与状态。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Requirements -> Components Mapping]
  - [x] 路由至少包括 `/api/agent-platform/clients` 和 `/api/agent-platform/clients/:id`；控制面仍使用 `/api/agent-platform/**` + `AdminAuth()`，不要提前把 client registration 路由混入 OAuth authorize/token 或 open-capability 数据面。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#549-564; router/agentplatform-router.go]
  - [x] 当前 story 要让 Cherry Studio 能以统一 client 模型被录入，但不要为 Cherry Studio 单独创建特殊表/特殊字段；若需要消费者特定字段，只能通过 namespaced extensions 表达。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#UJ-1; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#2-3-与-Cherry-Studio-的边界]

- [x] 让 1.4/1.5 的 exposure target 具备向真实 client 迁移的兼容路径 (AC: 1, 2)
  - [x] 当前 `agent_platform_resource_exposures` 里若仍使用 stub target 表达，至少要建立“可映射到正式 client identity”的兼容路径；`2.1` 可以通过 service 侧解析/校验，让 exposure target 与新 client 主模型具备对齐机制，但不要在这一步破坏 `1.4/1.5` 已有 exposure 语义。[Source: _bmad-output/implementation-artifacts/ap-1-4-establish-projection-baseline-decoupled-from-client-registration.md; _bmad-output/implementation-artifacts/ap-1-5-build-publication-projection-and-exposure-model.md]
  - [x] 本故事不要求一次性把所有旧 exposure target 数据迁移成正式 `client_id`，但至少要防止新增 exposure 继续使用无法映射的自由文本 target 结构。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Client vs client_instance separation]

- [x] 锁定 2.1 的边界，不提前做 2.2/2.3/开放能力数据面 (AC: 1, 2, 3)
  - [x] 本故事不实现 authorization code + PKCE、refresh token、token version、client credentials 真正发放、OAuth consent、`/api/open-capabilities/**`、`client_instance` 模型或 UI 完整表单体验。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.2; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.3; _bmad-output/planning-artifacts/architecture-agent-platform.md#348-370]
  - [x] `allow_client_credentials` 只作为注册配置位落地，不意味着 2.1 就要实现真实 token grant 路径。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Client model]

- [x] 补齐 model/service/controller 测试与管理面合同 (AC: 1, 2, 3)
  - [x] 新增 `model/agentplatform/client_test.go`，覆盖：`client_id` 稳定生成、TEXT 字段无数据库默认值、capabilities/extensions 存储与解析、invalid integration 关键字段的基础约束。[Source: model/agentplatform/resource_test.go; model/enterprise/text_default_test.go]
  - [x] 新增 `service/agentplatform/client_test.go`，覆盖：创建有效 client、拒绝缺少 capability/contract 的无效配置、禁用 client 时返回可诊断状态、extensions 命名空间校验。[Source: _bmad-output/planning-artifacts/ux-agent-platform.md#3-1-客户端注册]
  - [x] 新增 `controller/agentplatform/client_test.go`，覆盖最小 client CRUD API 的成功读写和错误输入；若管理面合同扩展，同步更新 `docs/openapi/api.json`。[Source: docs/openapi/api.json]

## Dev Notes

- `2.1` 的目标不是发 token，而是先把“什么是一个正式 client”建模清楚。只有这样后续 `2.2/2.3` 的授权和 token 生命周期才不会建立在一堆临时字段上。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-4]
- 当前 exposure 层已经有 per-target 语义，但 `2.1` 要开始让 target 与正式 client identity 对齐；这一步要追求兼容路径，而不是粗暴推翻 `1.4/1.5` 的投影模型。[Source: _bmad-output/implementation-artifacts/ap-1-4-establish-projection-baseline-decoupled-from-client-registration.md; _bmad-output/implementation-artifacts/ap-1-5-build-publication-projection-and-exposure-model.md]
- namespaced extensions 是显式架构约束，不是 UI 约定。服务层就应该保护这个边界，避免客户端配置把核心契约语义塞进 extensions 里“假装支持”。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#205; _bmad-output/planning-artifacts/architecture-agent-platform.md#503-606]
- `client_instance` 仍然不是 2.1 的目标；MVP 当前只需要产品级 principal `client`。不要在这个故事中提前把设备/安装实例概念做进去。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Client model; _bmad-output/planning-artifacts/architecture-agent-platform.md#Client vs client_instance separation]
- 继续沿用 `TEXT` + `common.*` JSON wrapper 的策略，所有结构化配置字段都不要引入数据库专属 JSON 类型或直接 `encoding/json` 业务调用。[Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go; AGENTS.md#Rule 2]

### Project Structure Notes

- 预计新增文件：
  - `model/agentplatform/client.go`
  - `service/agentplatform/client.go`
  - `controller/agentplatform/client.go`
  - `dto/agentplatform/client.go`
  - 对应 `*_test.go`
- 预计修改文件：
  - `model/agentplatform/migration.go`
  - `router/agentplatform-router.go`
  - `service/agentplatform/exposure.go`（若要建立 target→client 对齐的兼容路径）
  - `docs/openapi/api.json`
- 保持不改或只读依赖：
  - `relay/**`
  - `docs/openapi/relay.json`
  - `web/default/**`
  - `oauth/**`
  - `model/agentplatform/client_instance*.go`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.1: 建立下游客户端注册与能力声明模型]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-4]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#UJ-1]
- [Source: _bmad-output/planning-artifacts/ux-agent-platform.md#3-1-客户端注册]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Client model]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#205]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#503-606]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#549-564]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Client vs client_instance separation]
- [Source: _bmad-output/implementation-artifacts/ap-1-5-build-publication-projection-and-exposure-model.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 对 Agent Platform 的 `2.1 -> ap-2-1-*` key 映射沿用已修正逻辑；本 story 文件由 manual create-story takeover 补齐，用于避免 create-story 停留在 Epic 1 exposure 基线上空转。
- 已加载 `epics-agent-platform.md`、`architecture-agent-platform.md`、`prd-agent-platform-2026-05-31/prd.md` / `ux-agent-platform.md`，并参考已完成的 `ap-1-4`、`ap-1-5` 投影相关产物作为前序上下文。
- 当前仓库已经具备 registry、typed detail、lifecycle 与 exposure 基线，因此 `2.1` 的焦点是正式 client 主模型和 capability declaration，而不是重新定义前面的控制面资源模型。

### Completion Notes List

- 已落地正式 `agent_platform_clients` 主模型，补齐稳定 `client_id`、contract version、allowed scopes、grant types、capabilities 和 namespaced extensions 持久化。
- 已实现最小 client 注册管理 API：创建、读取、列表、更新，并把缺失 capability/contract 的对象标记为 `invalid_integration`。
- 已保留 `allow_client_credentials` 作为配置位，但没有提前实现真实 token grant。
- 已完成 client registration 相关 model/service/controller 定向测试，并通过 `GOCACHE=/private/tmp/go-build-cache go test ./model/agentplatform ./service/agentplatform ./controller/agentplatform` 验证。

### File List

- `_bmad-output/implementation-artifacts/ap-2-1-register-clients-and-capability-declarations.md`
- `model/agentplatform/client.go`
- `service/agentplatform/client.go`
- `service/agentplatform/client_test.go`
- `controller/agentplatform/client.go`
- `controller/agentplatform/client_test.go`
- `dto/agentplatform/client.go`
- `model/agentplatform/migration.go`
- `router/agentplatform-router.go`


## Change Log

- 2026-06-01: 完成 client 主模型、capability declaration、invalid integration 判定与定向测试，并将故事推进为 done。
