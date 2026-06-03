# Story 1.2: 为三类资源落地 typed detail 持久化结构

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 平台管理员,
I want `Skill`、`Knowledge`、`Agent` 的专属字段通过 typed detail 结构落地,
so that 平台不需要依赖巨型多态 payload 也能保持资源差异化表达。

## Acceptance Criteria

1. **Given** 不同资源类型存在专属字段  
   **When** 资源被持久化  
   **Then** 系统将共享治理字段与类型专属 detail 字段分开存储  
   **And** 不把所有资源强行塞进一个巨型多态 detail payload。
2. **Given** 项目仓库约束生效  
   **When** 新增 model、DTO 或 migration  
   **Then** JSON 编解码使用 `common/json.go`  
   **And** schema 与迁移保持 SQLite、MySQL、PostgreSQL 三库兼容。

## Tasks / Subtasks

- [x] 在已落地的 shared registry 之上补资源版本与 typed detail 存储骨架 (AC: 1, 2)
  - [x] 新增 `model/agentplatform/resource_version.go`，定义 `agent_platform_resource_versions` 表，至少覆盖：`resource_id`、语义化 `version`、`contract_version`、`summary`、`schema_json`、`detail_json`、`status`、`created_by`、`published_at`、`created_at`、`updated_at`，并保持对 `agent_platform_resources` 的稳定引用关系。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Core tables]
  - [x] 新增 typed detail 模型文件：`model/agentplatform/skill_def.go`、`knowledge_def.go`、`agent_def.go`（或等价命名），分别对应 `agent_platform_skill_defs`、`agent_platform_knowledge_defs`、`agent_platform_agent_defs`，确保三类资源专属字段不再依赖单一 polymorphic 大表。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Typed detail shape; _bmad-output/planning-artifacts/architecture-agent-platform.md#Implementation Rules]
  - [x] 在 `model/agentplatform/migration.go` 中把这些新表纳入统一迁移入口，沿用 `1.1` 已建立的 Agent Platform bounded context 迁移路径，不在其他域里分散注册。[Source: model/agentplatform/migration.go; model/main.go]

- [x] 为三类资源建立最小但清晰的 typed detail 字段边界 (AC: 1)
  - [x] `Skill` typed detail 至少预留：invoke schema、output schema、invoke mode、timeout、binding config 的存储位；这些字段进入 `agent_platform_skill_defs`，而不是塞进 registry 主表或复用一份模糊 `detail_json` 解释所有类型。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Typed detail shape]
  - [x] `Knowledge` typed detail 至少预留：`knowledge_mode`、provider type、provider adapter key、provider binding config、normalized query/citation schema、freshness rules、provider capability declarations；MVP 中 `knowledge_mode` 仍应面向 `retrieval`，但本故事只负责 typed detail 持久化，不负责 provider 执行链路。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Typed detail shape; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-10]
  - [x] `Agent` typed detail 至少预留：manifest、dependency references、prompt/template metadata、compatibility metadata，并明确不包含 runtime execution state、workflow orchestration state 或长任务状态。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Typed detail shape; _bmad-output/planning-artifacts/architecture-agent-platform.md#Agent runtime boundary]

- [x] 落地服务层与 DTO，证明 typed detail 能独立于 registry 被创建和读取 (AC: 1, 2)
  - [x] 扩展 `dto/agentplatform` 与 `service/agentplatform`，为资源版本/typed detail 提供最小的 create + read 结构，至少支持：在已存在 registry 资源上创建一个版本记录，并绑定对应类型的 detail 数据。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Requirements -> Components Mapping]
  - [x] 服务层必须显式区分共享治理字段与 detail 字段的写入路径：registry 更新、version 写入、typed detail 写入应分层清晰，不允许 controller 直接拼接跨表 SQL，也不允许把三类 detail 全部塞回 `agent_platform_resources`。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Definition vs exposure separation; AGENTS.md#Architecture]
  - [x] 当前故事只需要 control plane 的最小管理接口，可以通过已有 `/api/agent-platform/resources` 体系增补版本/detail 管理路由，或引入 `/api/agent-platform/resources/:id/versions` 的最小实现；不要提前实现 publish/exposure/OAuth/open-capabilities 消费面。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#549-550; _bmad-output/planning-artifacts/architecture-agent-platform.md#561-573]

- [x] 锁定 JSON 存储和三库兼容策略，避免 typed detail 反向破坏仓库约束 (AC: 2)
  - [x] 业务 JSON 编解码一律走 `common.Marshal`、`common.Unmarshal`、`common.UnmarshalJsonStr`、`common.DecodeJson`，禁止在 typed detail 业务代码里直接用 `encoding/json` 做 marshal/unmarshal。[Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go]
  - [x] 所有 schema/detail/provider 配置字段默认使用 `TEXT` 存储字符串化 JSON，不使用 PostgreSQL `JSONB`、数据库专属 JSON 运算符或 MySQL-only 行为；如需默认空对象/数组，应用层做归一化，不在 `TEXT` 字段上声明数据库默认值。[Source: AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6; model/enterprise/text_default_test.go]
  - [x] 若新增 migration/索引/约束，确保 SQLite、MySQL、PostgreSQL 三库都能跑通；尤其避免在 SQLite 上使用不支持的列变更路径，必要时沿用 `model/enterprise/migration.go` 的补列/兼容策略。[Source: model/enterprise/migration.go; AGENTS.md#Rule 2]

- [x] 明确 Story 1.2 的边界，不提前实现 1.3+ 的行为语义 (AC: 1, 2)
  - [x] 本故事只负责 typed detail 持久化结构与最小读写，不实现完整 lifecycle governance、rollback、projection/exposure、OAuth grant、refresh token、knowledge provider adapter、invoke/query runtime、或 `web/default` 管理面。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.3; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.4; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.5; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#6-暂不进入-MVP-的内容]
  - [x] `detail_json` / `schema_json` 等字段可以存在于 version/detail 层，但不能把“统一生命周期词汇”“published projection”“visible/callable”这些 1.3/1.4/1.5 的语义提前塞进 1.2 的 controller/service 行为里。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-2; _bmad-output/planning-artifacts/architecture-agent-platform.md#Exposure model]

- [x] 补齐模型、服务、控制器测试，证明 typed detail 分层真的工作 (AC: 1, 2)
  - [x] 新增 `model/agentplatform/*_test.go`，覆盖：resource_version 与 typed detail 表迁移、TEXT 字段无数据库默认值、`resource_id` 绑定关系正确、三类资源 detail 结构不互相污染。[Source: model/agentplatform/resource_test.go; model/enterprise/text_default_test.go]
  - [x] 新增 `service/agentplatform/*_test.go`，覆盖：在 registry 资源上创建版本与 typed detail、非法资源类型/缺失 detail 输入校验、以及 detail 更新不改变 registry 身份。[Source: _bmad-output/implementation-artifacts/ap-1-1-establish-shared-resource-registry-and-stable-identity.md; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-1]
  - [x] 新增 `controller/agentplatform/*_test.go`（或等价 API 回归），覆盖最小管理 API 的成功读写与非法输入场景；若 OpenAPI 管理面合同被扩展，同步更新 `docs/openapi/api.json`。[Source: docs/openapi/api.json; _bmad-output/planning-artifacts/architecture-agent-platform.md#Requirements -> Components Mapping]

## Dev Notes

- Story 1.2 的真正目标是把 `1.1` 刚建立的 registry 基线扩展成“共享治理字段 + typed detail 字段分层存储”，而不是回退到一个 `detail_json` 大对象承载所有类型。架构文档已经把这点明确为 `CP-AP-9`。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#CP-AP-9]
- `agent_platform_resource_versions` 与三张 typed detail 表是后续 publish、audit、invoke/query 契约的基础，但本故事只做结构和最小管理读写，不碰完整 lifecycle/publish/exposure 语义。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Core tables; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.3]
- `Knowledge`、`Skill`、`Agent` 的 detail 结构应该各自清晰，而不是为了“统一”再造一个更大的 polymorphic payload。统一只发生在 registry / version 治理层，不发生在资源专属字段表达层。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Typed detail shape]
- 延续 `1.1` 的实现经验：所有管理面仍走 Agent Platform bounded context 自己的目录树与 `/api/agent-platform/**` 路由；不要回到 `enterprise` 或 `relay/**` 中借位实现。[Source: _bmad-output/implementation-artifacts/ap-1-1-establish-shared-resource-registry-and-stable-identity.md]
- 跨库兼容和 JSON wrapper 约束在 1.2 比 1.1 更关键，因为 typed detail 会天然引入更多结构化字段。凡是 schema/detail/provider 配置类数据，都优先走 `TEXT` + `common.*` JSON wrapper；不要为了便利引入 JSONB 或直接 `encoding/json` 业务调用。[Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go; AGENTS.md#Rule 2]
- `Agent` typed detail 不能因为有 manifest/dependencies 就被实现成 runtime orchestrator。服务端 workflow state、长任务状态、session state 仍然明确不在 MVP 范围内。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Agent runtime boundary]
- 本故事的最小完成态应该能支持下一个故事继续做 lifecycle/version governance，而不是停留在“表建好了但 service/controller 不可用”。因此最少要让版本和 detail 的创建/读取路径真实可测。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-2; _bmad-output/planning-artifacts/architecture-agent-platform.md#Requirements -> Components Mapping]

### Project Structure Notes

- 预计新增文件：
  - `model/agentplatform/resource_version.go`
  - `model/agentplatform/skill_def.go`
  - `model/agentplatform/knowledge_def.go`
  - `model/agentplatform/agent_def.go`
  - `model/agentplatform/*_test.go`
  - `service/agentplatform/resource_version.go`（或在 `resource.go` 中扩展版本/detail 行为）
  - `controller/agentplatform/resource_version.go`（或等价版本/detail controller）
  - `dto/agentplatform/resource_version.go`
- 预计修改文件：
  - `model/agentplatform/migration.go`
  - `docs/openapi/api.json`
  - 可能涉及 `router/agentplatform-router.go`
- 保持不改或只读依赖：
  - `relay/**`
  - `docs/openapi/relay.json`
  - `web/default/**`
  - `oauth/**`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.2: 为三类资源落地 typed detail 持久化结构]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#CP-AP-9]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Core tables]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Typed detail shape]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Agent runtime boundary]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Requirements -> Components Mapping]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-1]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-2]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-10]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#2-关键边界]
- [Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go]
- [Source: AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]
- [Source: model/agentplatform/resource.go]
- [Source: model/agentplatform/migration.go]
- [Source: _bmad-output/implementation-artifacts/ap-1-1-establish-shared-resource-registry-and-stable-identity.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 对 Agent Platform 的 `1.2 -> ap-1-2-*` key 映射沿用已修正逻辑；本 story 文件由 manual create-story takeover 补齐，用于避免 create-story 会话持续停留在 `1.1` 代码上下文中空转。
- 已加载 `epics-agent-platform.md`、`architecture-agent-platform.md`、`prd-agent-platform-2026-05-31/prd.md`、`addendum.md`，并参考已完成的 `ap-1-1` story artifact 作为前序实现上下文。
- 当前仓库在 `1.1` 后已经存在 `model/service/controller/dto/agentplatform` 的 registry 基线、最小 `/api/agent-platform/resources` 路由和 OpenAPI 管理面合同，因此 `1.2` 需要在该基线上继续扩版本/detail，而不是重建 registry。

### Completion Notes List

- 已在 `1.1` registry 基线之上补齐 `agent_platform_resource_versions` 与三张 typed detail 表：`skill_def`、`knowledge_def`、`agent_def`。
- 已补充版本/detail 的最小 control plane API：创建资源版本、按资源与版本读取 typed detail。
- 已坚持使用 `TEXT` + `common.Marshal` / `common.UnmarshalJsonStr` 处理 schema/detail/provider 配置类结构化字段，未引入 JSONB 或 relay 范围变更。
- 已完成 typed-detail 相关 model/service/controller 定向测试，并通过 `GOCACHE=/private/tmp/go-build-cache go test ./model/agentplatform ./service/agentplatform ./controller/agentplatform` 验证。

### File List

- `_bmad-output/implementation-artifacts/ap-1-2-implement-typed-detail-storage-for-skill-knowledge-agent.md`
- `model/agentplatform/resource_version.go`
- `model/agentplatform/skill_def.go`
- `model/agentplatform/knowledge_def.go`
- `model/agentplatform/agent_def.go`
- `model/agentplatform/migration.go`
- `model/agentplatform/resource_version_test.go`
- `service/agentplatform/resource_version.go`
- `service/agentplatform/resource_version_test.go`
- `controller/agentplatform/resource_version.go`
- `controller/agentplatform/resource_version_test.go`
- `dto/agentplatform/resource_version.go`
- `router/agentplatform-router.go`


## Change Log

- 2026-06-01: 完成 typed detail/resource version 持久化结构、最小管理 API 和定向测试，并将故事推进为 done。
