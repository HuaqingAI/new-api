# Story 1.1: 建立共享资源注册表与稳定资源身份

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 平台管理员,
I want `Skill`、`Knowledge`、`Agent` 都通过统一注册表进入平台并拥有稳定资源身份,
so that 平台在三类资源之间拥有稳定资源身份和一致治理语义。

## Acceptance Criteria

1. **Given** Agent Platform bounded context 被引入  
   **When** 资源持久化层被搭建  
   **Then** 系统为 `Skill`、`Knowledge`、`Agent` 建立共享 registry 模型  
   **And** 每个资源都拥有稳定的 `resource_id` 与不可混淆的 `resource_type`。
2. **Given** 资源需要被后续 detail、publish、audit 流程复用  
   **When** 平台读取 registry 记录  
   **Then** 共享治理字段与资源身份字段足以独立支持这些流程  
   **And** 后续 typed detail 的演进不会改变已有资源身份。

## Tasks / Subtasks

- [x] 为 Agent Platform 建立新的 bounded context 骨架并接入现有仓库主流程 (AC: 1, 2)
  - [x] 新增 `model/agentplatform/`、`service/agentplatform/`、`controller/agentplatform/`、`dto/agentplatform/` 目录及最小编译骨架，遵循现有 `Router -> Controller -> Service -> Model` 分层，不复用 `enterprise` 或 `relay` 包路径承载 Agent Platform 资源治理逻辑。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Implementation Rules; _bmad-output/planning-artifacts/architecture-agent-platform.md#Complete Project Directory Structure; AGENTS.md#Architecture]
  - [x] 通过新的 `router/agentplatform-router.go`（或等价拆分）把 Agent Platform control plane 注册到现有 `/api` 命名空间下，并由 `router/api-router.go` 接入；本故事只覆盖 `/api/agent-platform/resources` 及其最小必要管理路由，不提前实现 `/api/open-capabilities/**` 或 OAuth 路由。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Route naming; _bmad-output/planning-artifacts/architecture-agent-platform.md#Project Structure & Boundaries; router/api-router.go]
  - [x] 在 `model/main.go` 的现有迁移链路中接入 `model/agentplatform` 的迁移入口，模式对齐 `model/enterprise/migration.go` 的集中注册，不在启动流程里单独开第二套 DB 初始化通道。[Source: model/main.go; model/enterprise/migration.go]

- [x] 落地共享资源注册表模型与稳定资源身份生成规则 (AC: 1, 2)
  - [x] 新增 `model/agentplatform/resource.go`，定义 `agent_platform_resources` 表，至少覆盖：内部主键、稳定 `resource_id`、`resource_type`、`display_name`、`owner_user_id`、`status`、`latest_version`、`tenant_id`、`created_at`、`updated_at`，并确保 `resource_type` 仅允许 `skill | knowledge | agent` 三种枚举值。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Core tables; _bmad-output/planning-artifacts/architecture-agent-platform.md#Naming Patterns]
  - [x] `resource_id` 必须是稳定、不可混淆、对外可引用的 opaque ID，不能直接复用自增主键，也不能因为后续 typed detail、版本或 exposure 演进而改变；建议采用统一前缀（如 `res_`）+ 随机/派生唯一片段的生成方式，并为唯一性与查询建立索引。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Core tables; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-1]
  - [x] 共享治理字段必须能独立支撑后续 detail、publish、audit 流程：至少要能表达资源归属者、当前治理状态、最新版本引用位与租户预留字段；本故事不要把 typed detail JSON、大型 polymorphic payload 或 exposure 状态混入 registry 表。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#CP-AP-9; _bmad-output/planning-artifacts/architecture-agent-platform.md#Definition vs exposure separation; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#2-关键边界]

- [x] 提供最小可验证的资源注册写入/读取能力，证明统一注册表可用 (AC: 1, 2)
  - [x] 新增 `dto/agentplatform/resource.go`、`service/agentplatform/resource.go`、`controller/agentplatform/resource.go`（命名可等价），实现最小管理能力：创建资源、按 ID 查看资源、分页/列表查询资源；这些 API 只服务 control plane，不服务开放能力消费面。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Requirements -> Components Mapping; _bmad-output/planning-artifacts/architecture-agent-platform.md#Control plane vs open plane separation]
  - [x] 路由至少包括 `/api/agent-platform/resources` 和 `/api/agent-platform/resources/:id`；响应中必须返回稳定 `resource_id` 与明确 `resource_type`，从而证明三类资源共享同一 registry 模型而非三套彼此孤立的主表。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#549-550; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-3]
  - [x] 管理路由先复用现有后台鉴权模式（如 `middleware.AdminAuth()`）即可，但不得误接入 dashboard session 以外的开放能力 bearer 鉴权；后续更细粒度 AP 鉴权可在 Epic 2 引入。[Source: router/api-router.go; _bmad-output/planning-artifacts/architecture-agent-platform.md#Control plane vs open plane separation]

- [x] 锁定跨库迁移、状态词汇与后续演进边界，避免 Story 1.1 过度扩张 (AC: 2)
  - [x] 所有迁移和模型声明必须兼容 SQLite、MySQL、PostgreSQL：优先 GORM `AutoMigrate`，避免数据库专属 JSONB / 运算符；如未来需要 JSON 存储，一律通过 `TEXT` + `common.Marshal` / `common.UnmarshalJsonStr` 路线，不在本故事引入 `encoding/json` 直接编解码业务字段。[Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go; AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]
  - [x] 统一生命周期词汇只允许使用 `draft`、`published`、`disabled`、`revoked`、`offline`、`deprecated`；本故事可以先为 registry 记录默认 `draft`，但不要提前实现完整版本流转、rollback 操作或 exposure 语义，这些分别属于 Story 1.3 / 1.4 / 1.5。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-2; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.3; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.4; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.5]
  - [x] 不要在 Story 1.1 中实现 typed detail 表、Knowledge provider adapter、OAuth token/consent、open capability invoke/query、或 `web/default` 控制面页面；这些都超出共享 registry 的职责。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Project Structure & Boundaries; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#6-暂不进入-MVP-的内容]

- [x] 补齐模型、service、controller/route 层测试与文档合同，证明稳定身份不会被后续演进破坏 (AC: 1, 2)
  - [x] 新增 `model/agentplatform/resource_test.go`，覆盖：`resource_id` 唯一/稳定生成、`resource_type` 枚举校验、`draft` 默认状态、三库兼容字段声明与迁移入口接通。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1302-1307; model/enterprise/department_test.go; model/enterprise/text_default_test.go]
  - [x] 新增 `service/agentplatform/resource_test.go` 和 `controller/agentplatform/*_test.go`（或等价测试），覆盖：创建 `Skill` / `Knowledge` / `Agent` 三种资源时都进入同一 registry、列表/详情返回稳定 `resource_id`、以及后续更新共享字段不会重写身份值。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#UJ-1; _bmad-output/planning-artifacts/architecture-agent-platform.md#Requirements -> Components Mapping]
  - [x] 若本故事新增管理 API，请同步更新 `docs/openapi/api.json`（不是 `docs/openapi/relay.json`），并补充最小 contract 断言，确保 Agent Platform 属于管理面 API 而非 relay 协议面。[Source: AGENTS.md#Overview; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#2-2-与-relay--模型转发链路的边界]

## Dev Notes

- Story 1.1 的目标不是“把三类资源所有字段都做完”，而是先打下 **共享 registry + 稳定 identity** 这层基线，让后续 typed detail、版本、projection、audit 都能在同一资源主键之上演进。若在 1.1 就把 detail/exposure 混进主表，1.2~1.5 会被迫返工。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.1; _bmad-output/planning-artifacts/architecture-agent-platform.md#CP-AP-9]
- Agent Platform 是新的 bounded context，不是 enterprise 模块的旁支，也不是 relay 的内嵌扩展。后续实现必须落在 `model/service/controller/dto/agentplatform` 目录树，而不是把新逻辑散落进 `enterprise`、`relay/**`、`service/sensitive.go` 这类现有业务域中。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Implementation Rules; _bmad-output/planning-artifacts/architecture-agent-platform.md#Complete Project Directory Structure]
- 现有仓库已经有成熟的迁移接入模式：`model/main.go` 统一调用子域迁移器，`model/enterprise/migration.go` 在域内集中维护 GORM `AutoMigrate` 与必要的 SQLite/列补偿逻辑。Agent Platform 也应复制这种模式，而不是在 controller/service 启动时偷跑建表逻辑。[Source: model/main.go; model/enterprise/migration.go]
- `resource_id` 是跨后续故事、客户端投影和审计链路的关键引用锚点。它必须对外稳定、不可从 typed detail 演算、不可因 publish/rollback 变化而变化；因此实现时不要把 `resource_id` 与 `latest_version`、`client_id`、或某类资源内部 slug 绑定死。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-1; _bmad-output/planning-artifacts/architecture-agent-platform.md#Core tables]
- 本故事应该优先交付 control plane 的最小资源注册读写，而不是先做 UI 或 provider-specific 适配。架构文档已经明确：“Scaffold the core AP bounded context first... Do not start with UI-only work or provider-specific retrieval-provider wiring before the core control-plane and token model exists.”[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1507]
- Agent Platform 管理面路由属于 `/api/agent-platform/**`，开放消费面才属于 `/api/open-capabilities/**`。Story 1.1 不要提前把 control plane controller 直接拿去服务 open plane，也不要把 dashboard session 鉴权误用于开放能力 bearer auth。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Route naming; _bmad-output/planning-artifacts/architecture-agent-platform.md#Control plane vs open plane separation]
- 当前仓库的 `router/enterprise-router.go` 展示了一个成熟的后台管理路由注册模式：在 `router/api-router.go` 下挂子 router，复用现有认证中间件。Agent Platform 在 1.1 可采用相同接入方式，但路径和包名必须保持独立，避免 domain 污染。[Source: router/api-router.go; router/enterprise-router.go]
- 本故事还不需要 `web/default` 页面，但 UI 方向上的术语和信息架构已经冻结：后续 1.6 及之后的前端实现必须围绕 Overview / Clients / Skills / Knowledge / Agents / Publishing / Audit & Diagnostics 构建，术语使用 `visible` / `callable` 和统一生命周期词汇。1.1 中新增的 DTO / API 命名应提前与这些术语一致。[Source: _bmad-output/planning-artifacts/ux-agent-platform.md]
- 本次 story 创建未发现 `project-context.md` 持久事实文件，因此实现约束主要以 `AGENTS.md`、Agent Platform PRD/addendum、architecture-agent-platform.md、ux-agent-platform.md 以及当前仓库真实结构为准。

### Project Structure Notes

- 预计新增文件：
  - `model/agentplatform/resource.go`
  - `model/agentplatform/migration.go`
  - `model/agentplatform/resource_test.go`
  - `service/agentplatform/resource.go`
  - `service/agentplatform/resource_test.go`
  - `controller/agentplatform/resource.go`
  - `controller/agentplatform/resource_test.go`
  - `dto/agentplatform/resource.go`
  - `router/agentplatform-router.go`
- 预计修改文件：
  - `model/main.go`
  - `router/api-router.go`
  - `docs/openapi/api.json`（若本故事引入实际管理 API）
- 保持不改或只读依赖：
  - `relay/**`
  - `docs/openapi/relay.json`
  - `web/default/**`
  - `oauth/**`
  - `controller/log.go`
  - `model/log.go`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.1: 建立共享资源注册表与稳定资源身份]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#4-1-统一资源控制面]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#UJ-1-林涛将-Cherry-Studio-接入平台并跑通首个-Skill-闭环]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#2-关键边界]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#4-建议的后台信息架构]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Core tables]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Implementation Rules]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Requirements -> Components Mapping]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Complete Project Directory Structure]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1507]
- [Source: _bmad-output/planning-artifacts/ux-agent-platform.md]
- [Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go]
- [Source: AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]
- [Source: model/main.go]
- [Source: model/enterprise/migration.go]
- [Source: router/api-router.go]
- [Source: router/enterprise-router.go]
- [Source: middleware/enterprise_admin.go]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 对 Agent Platform 的 `1.1 -> ap-1-1-*` key 映射与 epic source 解析已修正；本 story 文件由 manual create-story takeover 补齐，用于解除 create-story 连续 stall。
- 已加载 `epics-agent-platform.md`、`architecture-agent-platform.md`、`prd-agent-platform-2026-05-31/prd.md`、`addendum.md`、`ux-agent-platform.md`、`model/main.go`、`router/api-router.go`、`router/enterprise-router.go` 作为当前故事的主要上下文源。
- 当前仓库尚未存在 `model/service/controller/dto/agentplatform` 目录树，也未看到 `/api/agent-platform/**` 相关实现，说明 Story 1.1 仍是 Agent Platform bounded context 的真实起点。

### Completion Notes List

- 已手工补齐 Story 1.1 的开发上下文，明确本故事聚焦 shared registry + stable identity，不提前做 typed detail、projection/exposure、OAuth、open-capability 或 UI。
- 已将目录落点、迁移接入模式、control plane 路由边界、跨库/JSON 约束和测试要求写入 story，供后续 `dev-story` 直接执行。
- 已落地 Agent Platform bounded context 最小骨架，新增共享 `agent_platform_resources` registry、`res_` opaque `resource_id` 生成规则、管理面 `/api/agent-platform/resources` 路由与 DTO/service/controller 基线。
- 已在 `model/main.go` 迁移链路接入 Agent Platform migration，并补充模型、service、controller 层的定向测试文件与 `docs/openapi/api.json` 管理面契约条目。
- 已根据 story-automator review 自动修复稳定身份相关缺陷：资源创建阶段的 `resource_id` 生成时序、`resource_id` / `resource_type` 不可变约束、以及列表接口对非法 `resource_type` 过滤值的显式校验。
- 已在本地使用 `GOCACHE=/private/tmp/go-build-cache go test ./model/agentplatform ./service/agentplatform ./controller/agentplatform` 完成定向验证，模型、service、controller 三层测试均通过。

### File List

- `_bmad-output/implementation-artifacts/ap-1-1-establish-shared-resource-registry-and-stable-identity.md`
- `model/agentplatform/resource.go`
- `model/agentplatform/migration.go`
- `model/agentplatform/resource_test.go`
- `service/agentplatform/resource.go`
- `service/agentplatform/resource_test.go`
- `controller/agentplatform/resource.go`
- `controller/agentplatform/resource_test.go`
- `dto/agentplatform/resource.go`
- `router/agentplatform-router.go`
- `model/main.go`
- `router/api-router.go`
- `docs/openapi/api.json`

## Senior Developer Review (AI)

### Review Date

- 2026-06-01 00:36:25 +0800

### Outcome

- Approve

### Scope Reviewed

- 直接审查了 Story 1.1 对应的实现文件：`model/agentplatform/resource.go`、`model/agentplatform/migration.go`、`service/agentplatform/resource.go`、`controller/agentplatform/resource.go`、`router/agentplatform-router.go`、`router/api-router.go`、`model/main.go`、相关测试，以及 `docs/openapi/api.json` 的 Agent Platform 管理面契约。
- 交叉核对了故事 Acceptance Criteria、Tasks/Subtasks、Dev Agent Record -> File List 与当前源代码变更；本次 review 未发现需要阻塞交付的 git/story 源码清单不一致问题。
- 未执行外部 MCP / web 文档检索；本次审查以仓库内 `AGENTS.md`、Agent Platform PRD/addendum、architecture、epic 与源码为准，已足够覆盖 Story 1.1 的实现边界。

### Findings Fixed Automatically

- HIGH: `Resource` 模型在创建时依赖 `BeforeCreate` 生成 `resource_id`，但 `BeforeSave` 已先执行完整校验，导致新资源创建可能因缺少 `resource_id` 直接失败；已修正创建时序并补充回归测试。
- HIGH: `resource_id` 与 `resource_type` 只有文档约束，没有模型级不可变保护，后续更新可能破坏稳定身份；已在模型更新钩子中补充不可变约束，并为整实体保存与 map 局部更新分别处理。
- MEDIUM: 列表接口对非法 `resource_type` 查询值会静默返回空结果，而不是显式拒绝非法治理词汇；已在 service 层增加枚举校验，并补充 controller/service 测试。
- MEDIUM: `GenerateResourceID` 静默吞掉随机串生成错误，理论上会把熵源失败伪装成业务成功；已改为向上返回错误。
- MEDIUM: 模型层对 `display_name` 与 `owner_user_id` 的完整性校验不足，允许绕过 service 直接写入不完整 registry 记录；已下沉最小治理字段校验，并补充测试。

### Validation Evidence

- `GOCACHE=/private/tmp/go-build-cache go test ./model/agentplatform ./service/agentplatform ./controller/agentplatform`
- 结果：全部通过。

## Change Log

- 2026-06-01: story-automator review 完成，自动修复稳定身份/枚举校验问题，补充回归测试，并将故事状态推进为 `done`。
