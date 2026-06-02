---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
workflowType: 'architecture'
lastStep: 8
status: 'complete'
completedAt: '2026-05-27'
revisedAt: '2026-05-27'
revision: 'V1.1'
revisionNotes: |
  V1.1 修订（基于用户 OQ-6/OQ-7 决策，2026-05-27 同日完成）：
  - OQ-7 关闭：取消"主部门"概念，用户↔部门为 N:N 平等关系。
  - OQ-6 关闭：部门预算 = balance/subscription 双类型，子分配落地为用户名下的 wallet
    （复用 subscriptions 表 + 2 列扩展 source_type / source_allocation_id）。
  - 新增 CP-14：部门用量归因 = 成员消费视角，多部门用户重复计入，UI 标注不可加和。
  - 移除 department_resolver.go 与历史快照机制，归因改为运行时 join enterprise_user_departments。
  - 新增后台任务：wallet_state_sync_task、balance_expiry_task。
  - 修订 CP-2、CP-5；新增 CP-14；Step 7 OQ-6/7 状态置为"已关闭"。
  V1.3 修订（基于用户分层预算治理与额度审批 follow-up，2026-06-01）：
  - 新增分层治理约束：部门负责人来源同时支持钉钉同步与手动指定，优先级为
    manual_deny_override > manual_grant > dingtalk_synced_owner。
  - 新增 descendant scope：预算/用量/告警 summary 支持 include_descendants，继续沿用邻接表，
    如有性能需要可增路径缓存字段，不引入闭包表。
  - 新增 budget delegation：支持祖先部门预算池向任意后代部门预算池分配；成员 wallet 分配仍由
    enterprise_quota_allocations 承接，relay/billing 主链路保持不变。
  - 新增 quota request workflow：员工申请 -> 部门负责人单步审批 -> 复用现有 allocation service 自动分配。
  - 新增治理通知约束：通知失败不得阻塞核心治理事务提交，必须记录投递状态与错误原因。
inputDocuments:
  - "_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md"
  - "_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md"
  - "_bmad-output/planning-artifacts/briefs/brief-new-api-2026-05-27/brief.md"
  - "_bmad-output/planning-artifacts/briefs/brief-new-api-2026-05-27/addendum.md"
  - "docs/api-contracts-server.md"
  - "docs/architecture-electron.md"
  - "docs/architecture-server.md"
  - "docs/architecture-web-classic.md"
  - "docs/architecture-web-default.md"
  - "docs/component-inventory-electron.md"
  - "docs/component-inventory-web-classic.md"
  - "docs/component-inventory-web-default.md"
  - "docs/contribution-guide.md"
  - "docs/data-models-server.md"
  - "docs/deployment-guide.md"
  - "docs/development-guide-electron.md"
  - "docs/development-guide-server.md"
  - "docs/development-guide-web-classic.md"
  - "docs/development-guide-web-default.md"
  - "docs/index.md"
  - "docs/integration-architecture.md"
  - "docs/ionet-client.md"
  - "docs/project-overview.md"
  - "docs/source-tree-analysis.md"
  - "docs/translation-glossary.fr.md"
  - "docs/translation-glossary.md"
  - "docs/translation-glossary.ru.md"
  - "docs/channel/other_setting.md"
  - "docs/installation/BT.md"
workflowType: 'architecture'
project_name: 'new-api'
user_name: 'hth'
date: '2026-05-27'
---

# Architecture Decision Document

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

## Project Context Analysis

### Requirements Overview

**Functional Requirements:**

V1 是对现有 new-api 的企业管控增强，不是新建产品或重写网关。需求由 26 个 FR 组成，集中在 6 个能力域：

1. 组织与部门模型：新增企业/部门/用户部门关系，但必须保持部门与现有 `Group` 语义分离。部门用于治理、统计和告警，`Group` 继续用于模型、渠道和计费倍率。**用户↔部门为 N:N 平等关系，不存在"主部门"概念**（V1.1 修订，OQ-7 已关闭）。
2. 钉钉登录与通讯录同步：钉钉是 V1 唯一交付的企业身份入口。登录依赖最近一次成功同步的本地部门快照建立 N:N 关系，不因同步进行中而阻塞登录。
3. 部门配额与额度控制：部门预算池采用 **balance / subscription 双类型**（与现有 user token/subscription 同构）。部门管理员把预算分配给成员时，**在被分配用户名下创建一个独立 wallet（subscription 行）**，relay 路径完全不动；wallet 重置/过期/撤销与父预算池状态联动（V1.1 修订，OQ-6 已关闭）。
4. 组织用量看板与导出：提供部门维度聚合、详情、CSV 导出和定期邮件报告。**多部门用户的消费在所有归属部门重复计入，UI 需明确标注"部门间不可加和"**（V1.1 新增 CP-14）。归因运行时按 `enterprise_user_departments` join，无需历史快照表。
5. 内容监控告警：基于现有敏感词/过滤能力补充风险事件、告警规则、告警投递和部门风险概览，不引入复杂语义审核模型。
6. 分层预算治理与额度审批：在既有企业组织、预算、用量和告警基础上，补齐部门负责人权限模型、包含子部门的查询作用域、上级预算池向后代预算池分配、成员 allocation 调整/取消/回收、员工额度申请单步审批，以及治理动作审计与钉钉通知。

**Non-Functional Requirements:**

关键 NFR 会直接决定架构：

- 数据库必须同时兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+。
- 敏感凭据不得明文返回前端、写入日志或导出。
- 组织同步失败不得影响现有登录、API 调用、计费和日志链路。
- 部门配额扣减必须与用户 quota 最终一致。
- 同步任务、配额拦截、告警发送必须有后台可查状态和错误原因。
- 部门用量看板在 100 万条 logs、部门数不超过 100 的环境下，近 30 天首屏 P95 目标为小于 3 秒。
- Default 前端新增文案必须进入 i18n，英文 key 为源字符串；Classic 主题如同步实现需遵守其现有 i18n 风格。

**Scale & Complexity:**

- Primary domain: full-stack enterprise admin + API gateway billing extension
- Complexity level: high / enterprise-leaning
- Estimated architectural components: 12

主要组件包括：组织/部门模型、用户部门关系、钉钉 OAuth provider、钉钉同步任务、同步结果与冲突处理、部门配额策略、部门扣减/补偿、部门用量聚合、CSV/报告、内容风险事件、告警投递、两套前端管理界面。

### Technical Constraints & Dependencies

- 后端沿用 Go + Gin + GORM 的 Router -> Controller -> Service -> Model 分层。
- 所有 JSON marshal/unmarshal 必须使用 `common/json.go` wrapper。
- 新增数据库对象和查询必须三库兼容，避免 DB 专属 SQL；不要新增名为 `group` 的组织字段。
- `new-api` 与 `QuantumNous` 是受保护标识，不得删除、替换或改名。
- 部门配额不得改变 `pkg/billingexpr` 的表达式语义，不新增模型定价变量，只消费最终 quota。
- `tenant_id` 默认只在新增组织相关表预留；是否扩展到 `users`、`logs`、`quota_data` 是 OQ-3。
- 钉钉登录是 V1 交付项，飞书/企微仅做结构预留。
- OQ-6 / OQ-7 在 V1.1 修订中已关闭：OQ-7 取消"主部门"，N:N 平等关系；OQ-6 部门预算 = balance/subscription 双类型，子分配落地为用户名下 wallet。详见本文档 CP-2 / CP-5 / CP-14。

### Cross-Cutting Concerns Identified

- 配额一致性：用户 quota、部门 quota、消费日志、聚合数据必须最终一致。
- 历史快照：部门改名、移动、停用、用户转部门不能污染历史用量口径。
- 并发控制：同部门多用户并发请求不得造成部门额度负数或漏扣。
- 可观测性：同步、配额、告警都需要状态表、错误原因和可重试路径。
- 权限隔离：普通用户不能看到其他部门风险事件或管理员字段。
- 性能：部门看板不能默认扫全量日志，应优先考虑快照/聚合表。
- 棕地兼容：不能破坏现有 OAuth、用户、Token、Group、渠道选择、订阅升级、计费表达式和日志链路。

## Starter Template Evaluation

### Primary Technology Domain

Brownfield full-stack AI API gateway / enterprise admin platform.

现有基础已经明确：

- Backend: Go + Gin + GORM + Redis + SQLite/MySQL/PostgreSQL
- Default Web: React 19 + TypeScript + Rsbuild + TanStack Router/Query + Base UI + Tailwind
- Classic Web: React 18 + Vite + Semi Design
- Desktop: Electron shell spawning the embedded Go server
- Deployment: Go single binary with embedded frontends, Docker, systemd, Electron

### Starter Options Considered

**Option 1: Existing new-api monorepo**

Use the current repository as the architectural starter.

Rationale:

- It already contains the real production integration points: relay pipeline, billing, quota, logs, settings, OAuth, two frontend themes, Electron, Docker, and DB migration patterns.
- V1 is a brownfield extension, not a greenfield rewrite.
- This option preserves protected identifiers, existing module paths, go:embed constraints, and cross-DB migration patterns.
- It avoids re-solving framework setup and keeps architecture decisions focused on organization, DingTalk, department quota, department usage, and alerting.

**Option 2: create-rsbuild React starter**

Official Rsbuild supports `create-rsbuild` and current React templates, including optional tools such as Tailwind, Storybook, ESLint, Prettier, and testing support. This is useful for greenfield frontend apps, but not suitable here because the default frontend already has a mature feature-sliced structure and is embedded by the Go backend.

**Option 3: create-vite React starter**

Official Vite supports `create-vite` with React TypeScript templates. This is relevant only to greenfield frontend work or the existing Classic theme's tooling lineage. It does not help the Default theme, backend, billing, or deployment architecture.

**Option 4: Gin quickstart / examples**

Gin's official quickstart is useful for creating a new minimal Go API service. It is not a starter for this project because new-api already has a layered server architecture, router groups, middleware, relay adapters, model layer, settings, and deployment constraints.

### Selected Starter: Existing new-api Monorepo

**Rationale for Selection:**

The selected starter is the current repository. New work should be added inside existing architectural boundaries instead of scaffolding a new app.

**Initialization Command:**

```bash
# No external starter command.
# Continue from the existing repository and add modules within current boundaries.
```

**Architectural Decisions Provided by Starter:**

**Language & Runtime:**

Go backend, React/TypeScript default frontend, React/JavaScript classic frontend, Electron desktop shell.

**Styling Solution:**

Default theme uses Tailwind/Base UI/shadcn-style primitives. Classic theme uses Semi Design.

**Build Tooling:**

Default theme uses Rsbuild. Classic theme uses Vite. Backend embeds both frontend builds through Go `go:embed`.

**Testing Framework:**

Go package tests exist across controller/service/model/common/pkg/relay. Default frontend relies mainly on TypeScript/build checks with limited component tests. Classic and Electron rely mostly on build/manual validation.

**Code Organization:**

Backend follows Router -> Controller -> Service -> Model. Frontend Default follows feature-sliced modules with file routing. Classic follows centralized React Router pages/components. Electron is a thin shell.

**Development Experience:**

Use existing commands:

- `go test ./...`
- `go build ./...`
- `cd web/default && bun run build`
- `cd web/default && bun run typecheck`
- `cd web/classic && bun run build`

**Note:**

The first implementation story should not initialize a new project. It should scaffold the V1 enterprise modules inside the existing repository: organization/department model, DingTalk integration, department quota, department usage aggregation, and alerting.

## Core Architectural Decisions

### Decision Priority Analysis

**Critical Decisions (Block Implementation):**

- V1 基于现有 new-api brownfield monorepo 扩展，不引入外部 starter。
- 企业模块必须保持上游兼容，优先新增独立表、service、controller、route、frontend feature，避免破坏 relay、billing、user、token、log 核心链路。
- 部门额度是预算分配池（balance/subscription 双类型），不是运行时计费账户；分配 = 在用户名下创建独立 wallet（subscriptions 表新行），API 调用按 wallet 优先级扣减（零 relay 改动）。
- 钉钉能力使用专用企业集成模块，不塞进通用 Custom OAuth Provider。
- 企业 API 属于管理后台接口，进入 `docs/openapi/api.json`，不进入 AI 模型接口 `docs/openapi/relay.json`。
- 后台同步、聚合、告警必须有状态表、错误原因和可重试路径。

**Important Decisions (Shape Architecture):**

- 部门层级 V1 使用邻接表；仅在看板性能需要时增加 path/cache 字段。
- `tenant_id` V1 只在新增组织相关表预留，不扩展 `users`、`logs`、`quota_data`。
- Default frontend 是完整企业管理体验；Classic 提供必要可用管理入口。
- Redis 可增强锁、缓存、限流和去重，但 DB 状态表是正确性来源。
- 新增迁移必须三库兼容：SQLite、MySQL 5.7.8+、PostgreSQL 9.6+。

**Deferred Decisions (Post-MVP):**

- SaaS 多租户全链路隔离。
- 部门 quota 参与 relay 预扣费/结算。
- 闭包表或复杂组织图模型。
- 外部消息队列。
- 配额自动回收/撤销。
- 飞书、企微等第二企业身份源。

### Data Architecture

**Upstream Compatibility Principle:**

本项目作为 new-api 分支，需要持续合并官方功能。V1 企业模块应避免 breaking update，除非该改动明确要回提交主线。默认使用新增表、新 service、新路由、wrapper/adapter 集成点，而不是侵入 relay、billing、user、token、log 内部实现。

**Department Hierarchy:**

部门使用 adjacency-list 建模。字段包括本地部门 ID、父部门 ID、钉钉部门 ID、名称、状态、排序、同步批次、快照时间。V1 不引入 closure table。

**Tenant Reservation:**

`tenant_id` 只在新增组织/企业相关表中预留。V1 不扩展 `users`、`logs`、`quota_data`，避免过早改造主表和历史数据链路。

**Department Quota Semantics:**

部门预算池采用 **双类型**：
- `balance`：一次性余额型，可设 `expires_at`（NULL 表示永久），不重置；分配 = 从池里划账给成员，撤销 = 回收剩余部分。
- `subscription`：订阅周期型，每周期 reset，与现有 user subscription 周期机制（`daily/weekly/monthly/custom_seconds`）同构；父池暂停/过期 → 所有子分配同步状态。

部门管理员把预算分配给成员时，**在被分配用户名下创建一个独立 wallet**，落地为 `subscriptions` 表新增一行（扩展 `source_type='enterprise_allocation'` + `source_allocation_id` 两列）。父子继承规则：

| 父池 type | 子 wallet 配置 | reset 行为 | 撤销行为 |
|---|---|---|---|
| balance | `is_subscription=false`，`remain_quota=committed_quota`，`expires_at` snapshot | 不重置；到期作废 | `parent.remaining += wallet.remain_quota`；wallet 失效 |
| subscription | `is_subscription=true`，`quota=committed_quota`，cycle/expires 跟随父（soft-ref） | **REPLACE**：每周期 `wallet.remain_quota = wallet.quota`（复用现有订阅 reset 任务，零 relay 改动） | `parent.allocated_total -= wallet.quota`；wallet 失效；当周期未用部分自然消失 |

**铁律**：`Σ parent.children.committed_quota ≤ parent.total_quota`（balance）或 `cycle_quota`（subscription），分配/扩容前校验。

**公共额度**（如部门下 N8N 共享 wallet）：通过 service account user（`users.type='service'`，不可登录、不绑钉钉、由部门 admin 创建）+ 给它分配 wallet 的方式实现，零特殊路径。

**默认优先级**：部门分配 wallet 在用户优先级列表中默认排在主钱包之前；用户可调整顺序，但不能删除（删除只能由 admin 撤销）。

**No Relay-Path Department Deduction:**

V1 不在 `PreConsumeQuota`、`PostConsumeQuota`、billing expression settlement、relay handler、token quota 中加入部门扣减逻辑。

**Concurrency Strategy:**

沿用现有 wallet/subscription 更新方式：Redis/cache-aware helper、batch update、GORM `gorm.Expr` delta update。部门预算池剩余额度变更使用 GORM `gorm.Expr("remaining + ?", delta)` 原子 update + WHERE `remaining + ? >= 0` 防超扣；wallet 重置/扣减完全复用现有 subscription 表的并发安全机制，不引入新行锁子系统。

**Allocation Ledger:**

新增部门配额分配流水（`enterprise_quota_allocations`），记录父预算池 ID、目标用户 ID、生成的 wallet/subscription ID、操作者、金额、类型快照（balance/subscription）、cycle 配置快照（subscription 模式时）、expires_at 快照、原因、分配前后预算快照、状态（active/paused/revoked/expired）。该流水**自身即审计源**，不再向 `enterprise_admin_actions` 双写。

**Usage Attribution:**

部门用量看板按 **成员消费视角** 归因：运行时 join `enterprise_user_departments`，把用户的每笔消费**同时计入**该用户当前所有归属部门的统计中。多部门用户的消费在 N 个部门看板都出现（重复计入），UI 必须明确标注 _"该统计含跨部门用户的消费，部门间不可直接相加"_。归因不依赖历史快照表，**取消 `department_resolver.go`**。

**Revocation Policy:**

撤销 = 把 wallet 标记为 inactive 并在父预算池上释放承诺额：
- balance 撤销：`parent.remaining += wallet.remain_quota`（仅释放未消费部分；已消费部分由用户保留）；wallet 状态 `revoked`。
- subscription 撤销：`parent.allocated_total -= wallet.quota`；wallet 状态 `revoked`，下个周期不再 reset；当周期已发出但未用完的额度自然消失。
- 父池过期/暂停时所有子 wallet 自动联动状态变更，由 `wallet_state_sync_task` 处理。

**PRD Alignment Needed:**

PRD FR-10 需要从”部门参与运行时预扣费/结算”修正为”部门预算分配给成员，**在用户名下创建独立 wallet**，relay 路径继续按 wallet 优先级扣减”。FR-2 措辞需从”主部门关系”改为”用户↔部门 N:N 关系”。FR-13/14/20 需补充”多部门用户消费在所有归属部门重复计入，部门间不可加和”的口径说明。

### Authentication & Security

**DingTalk Integration Boundary:**

钉钉登录、corp/app access token、通讯录同步、部门成员关系、冲突处理、同步状态属于专用 DingTalk enterprise integration module，不复用 `custom_oauth_providers` 作为主模型。

**Login Session:**

钉钉登录成功后复用现有 session、`setupLogin`、`UserAuth/AdminAuth/RootAuth`，不新建 JWT/session 体系。

**Identity Binding:**

新增钉钉身份绑定表，记录 `corp_id`、`union_id/open_id/user_id` 与本地 `user_id`。优先使用钉钉稳定身份；邮箱/手机号只作为无冲突时的辅助匹配。冲突进入待处理表，由管理员人工确认。

**Authorization Model:**

Root 管理钉钉密钥和系统级企业配置；Admin 管理全局企业能力；部门管理员权限放在新增部门角色/成员关系表中，不扩展 `users.role` 枚举。

**Department Owner Precedence (V1.3):**

部门负责人来源同时支持钉钉同步与手动指定。最终有效权限按以下顺序解析：

`manual_deny_override > manual_grant > dingtalk_synced_owner`

其中：
- 钉钉同步负责人保留为组织来源事实；
- 手动授权/禁用属于本地权限层覆盖，不回写外部事实；
- 若某部门无任何有效负责人，治理审批动作可回退给 Admin，但必须显式可见且可审计。

**Secrets:**

钉钉 app secret、client secret、access token、suite ticket 不返回前端，不进入日志、导出和错误响应。

**Sync Safety:**

通讯录同步失败不得阻断现有登录、API 调用、计费和日志链路。登录使用最近一次成功同步的本地部门快照建立用户↔部门 N:N 关系；当无任何成功同步快照时，用户登录仍可完成，部门归属为空（看板上显示为"未归属"）。

### API & Communication Patterns

**API Style:**

继续使用 Gin REST。企业管理 API 使用 `/api/enterprise/**` 这类代码路由命名空间，例如 departments、dingtalk、quota-allocations、usage、alerts。

**OpenAPI Placement:**

企业 API 归入管理后台接口 `docs/openapi/api.json`，不进入 AI 模型接口 `docs/openapi/relay.json`。可新增 OpenAPI tags：企业组织、钉钉集成、部门配额、组织用量、风险告警。

**Relay Boundary:**

不改 `/v1/**`、`/v1beta/**`、`/mj/**`、`/suno/**` 的协议表面。不向 relay 请求 DTO 添加部门字段，不改变 OpenAI 兼容错误格式和 `TokenAuth` 行为。

**Response Format:**

控制台 API 沿用 `{ success, message, data }` 与 `common.ApiSuccess/ApiError/ApiSuccessI18n/ApiErrorI18n`。CSV/附件导出可以使用文件响应，但不能泄露敏感凭据。

**Error Handling:**

企业模块内部定义稳定业务错误原因，用于同步任务、配额分配、冲突处理和告警重试；对前端仍映射为现有响应格式。

**Async Work:**

通讯录同步、报表聚合、定期邮件、告警投递采用后台 job + 状态表。V1 不引入消息队列作为必需依赖。

### Frontend Architecture

**Default Theme:**

Default frontend 是 V1 企业能力完整交付面。新增企业 feature，或按子域拆分 organization、dingtalk、enterprise-usage、enterprise-alerts。路由放在 `_authenticated` 下，并纳入现有 sidebar/nav 模块配置。

**Classic Theme:**

Classic 不做同等深度重构，但提供必要管理入口：钉钉配置/同步状态、部门树与成员、部门配额分配、组织用量看板、告警列表。实现遵循 Classic 的 `pages/`、`components/`、Semi Design。

**State & Data Fetching:**

Default 使用 feature `api.ts` + TanStack Query hooks，queryKey 使用 `['enterprise', scope, params]`。Classic 继续使用 axios helpers 和页面级 loading/error 状态。

**Permission UI:**

前端只做导航和按钮可见性控制，真实权限由后端鉴权和企业权限 service 决定。部门管理员范围以后端返回的可管理部门集合为准。

**Information Architecture:**

不要把企业能力塞进现有 Group、User、Log 页面。新增“企业组织/组织管理”模块，内部用 tabs/sections 承载组织架构、钉钉同步、配额分配、用量报表、风险告警。负责人审批、治理时间线和通知投递仍留在该工作台；但普通员工的额度申请入口应贴近余额 / 钱包场景，不要求普通用户通过企业组织管理菜单进入。

**Descendant Scope (V1.3):**

“包含子部门”优先作为查询/汇总作用域，而不是全量明细展开能力。第一版仅要求覆盖：
- 预算 summary
- 用量 summary
- 告警 summary

API 层建议增加 `include_descendants` 显式参数。树解析继续基于邻接表；若性能成为瓶颈，可增加跨库兼容的路径缓存字段，但 V1.3 不引入 closure table。

**i18n:**

Default 新文案使用英文 key；Classic 遵守当前域已有 key 风格。术语固定：Department 不等于 Group，Quota Allocation 不等于 runtime billing。

### Infrastructure & Deployment

**Deployment Shape:**

不新增 V1 必需服务。继续支持单二进制、Docker、systemd、Electron。企业模块随 Go 二进制和两套前端一起构建/发布。

**Migration:**

新增表进入现有 GORM migration，只在主节点执行。所有迁移必须兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+。

**Configuration:**

钉钉配置优先走管理后台 Option 或专用配置表。`.env` 只保留部署级开关和可选默认值。

**Background Tasks:**

同步、聚合、定期邮件、告警投递使用现有 goroutine/ticker/task service 模式，并受 `common.IsMasterNode` 控制，避免多节点重复执行。

**Redis:**

Redis 可用于同步锁、缓存、限流、任务去重增强；无 Redis 时必须通过 DB 状态表和幂等逻辑保持可用。

**Observability:**

新增同步任务、聚合任务、告警投递状态记录。日志只写摘要，不写 secret、token 或个人敏感信息。

**Notifications:**

复用现有邮件/webhook/Bark/Gotify 通知能力和 notify limit；企业告警只新增部门维度收件人解析与投递记录。

### Department Usage Attribution (V1.1 新增)

**Attribution Model:**

V1.1 取消"主部门"概念后，部门用量归因采用 **成员消费视角（方案 B）**：用户每笔消费同时计入该用户当前所有归属部门的统计中。多部门用户的同一笔消费在 N 个部门看板上同时可见，**部门间不可直接相加**。

**Why This Choice:**

| 候选方案 | V1.1 选择 | 拒绝原因 |
|---|---|---|
| A. 仅预算池视角（看板只看分配/撤销/剩余） | ❌ | FR-14 "模型分布/用户排行"、FR-20 "风险率"在部门维度无法落地 |
| **B. 成员消费视角，多部门重复计入** | ✅ | 实现简单（运行时 join `enterprise_user_departments`）；FR-13/14/20 全部覆盖；与 N:N 模型自洽；与"用户多部门可叠加享受预算"的产品意图一致 |
| C. 分配来源归因（按 wallet 来源分摊每笔消费） | ❌ | 需在 relay/logs 层加追溯字段；与"不动 relay"承诺冲突；并发安全难；撤销时回退已消费部分需要新规则 |

**Implementation:**

- 用量聚合任务 `usage_aggregation_task` 按时间窗扫描 `logs`，对每条日志 `INNER JOIN enterprise_user_departments` 展开为 `(log_row, dept_id)` 多行，分别累加到 `enterprise_usage_snapshots` 的对应部门桶。
- 单用户单时段在多部门的累加是**业务上正确的重复**，不是数据缺陷。
- 未归属用户（无任何 `enterprise_user_departments` 行）的消费单独归入 `dept_id = NULL`，看板上显示为"未归属"分组，**不参与任何部门的风险率分母**。

**UI Annotation Requirements:**

- 部门用量看板（FR-13）、部门详情（FR-14）、部门风险概览（FR-20）页面顶部必须有固定文案 _"本统计含跨部门用户的消费，多部门用户在每个部门均计入一次，部门间数值不可直接相加。"_
- 文案进入 i18n key `enterprise.usage.multi_dept_disclaimer`。
- CSV 导出（FR-15）首行说明区追加同样的免责说明。
- 部门内"用户排行"展示同一用户跨部门重复出现的，按部门维度独立计算（属于哪个部门的用户排行就只看该部门口径下的累计）。

**Storage Tradeoff:**

`enterprise_usage_snapshots` 多部门重复计入会让聚合表行数膨胀（每个 log 行最多被复制 N 次，N = 该用户当前部门数）。V1.1 评估：
- 中型企业（员工 ≤ 2000）平均部门归属数 ≈ 1.2~1.5，膨胀系数可控。
- 100 万 logs + ≤100 部门 + 平均 1.5 重复 → 聚合表 ~150 万行，按 `(period, dept_id, date)` 聚合后实际写入 ≤ 部门数 × 时间桶数 × 模型数，远小于 logs 表本身。
- 性能仍满足 NFR "近 30 天 P95 < 3s"。

**Risk Hedge:**

- 若未来发现重复计入引发管理困惑，可在 V2 追加"分配来源归因（方案 C）"作为可选模式，schema 兼容（`enterprise_quota_allocations` 已有 `wallet_id` 可追溯）。
- 当前 V1.1 不预留 schema 字段以保持表结构简洁。

### Decision Impact Analysis

**Implementation Sequence:**

1. 新增企业基础表：部门、用户部门关系（N:N，无 is_primary）、部门角色、钉钉绑定、同步任务、冲突表。
2. 新增钉钉 integration service：token 获取/缓存、OAuth 登录、通讯录同步。
3. 扩展现有 `subscriptions` 表：追加 `source_type` + `source_allocation_id` 两列（三库兼容迁移）。
4. 新增部门预算池 + wallet 分配 service：父池 CRUD、分配/撤销/暂停、子 wallet 联动状态。
5. 新增组织用量归因与聚合表（按 `enterprise_user_departments` 多部门重复计入）。
6. 新增风险事件、告警规则、告警投递状态。
7. 新增 `/api/enterprise/**` 管理后台 API，并更新 `docs/openapi/api.json`。
8. Default 完整企业管理 UI（含"多部门用户重复计入"说明文案）。
9. Classic 必要企业管理入口。
10. 部署文档补充钉钉配置、多节点与后台任务说明。

**Cross-Component Dependencies:**

- 钉钉登录依赖最近一次成功同步快照（建立用户↔部门 N:N 关系）。
- 部门预算池与子 wallet 复用现有 `subscriptions` 表（扩展 `source_type` + `source_allocation_id` 两列），不依赖 relay。
- 用量看板依赖运行时 join `enterprise_user_departments` + 聚合任务，不依赖前端扫全量 logs；多部门用户重复计入由聚合阶段处理。
- 告警依赖风险事件与部门 N:N 归因（同样多部门重复计入）。
- 前端权限展示依赖后端权限摘要。
- OpenAPI 管理后台文档是企业前端和外部管理客户端的契约来源。

**Version/Docs Verification Notes:**

- React docs 当前 latest major 为 19.2，符合 Default React 19 方向。
- Go 当前稳定线已到 1.26.x，但本仓库应保持现有 Go baseline 与构建链，不在企业模块中强制升级。
- Gin/GORM 当前上游版本高于仓库 pin 版本，V1 企业模块不以依赖升级为前提。
- OpenAPI operation tags 用于文档组织，因此企业 API 应作为管理后台接口 tags，而不是 AI 模型接口。

## Implementation Patterns & Consistency Rules

### Pattern Categories Defined

**Critical Conflict Points Identified:**

13 个 AI Agent 在 V1 企业模块开发中可能做出不同选择的领域，已在本节逐项锁定。所有企业模块代码生成必须严格遵循以下模式，以避免跨模块、跨 Agent、跨 PR 的实现漂移。

### Naming Patterns

**Database Naming Conventions:**

- 表名统一加 `enterprise_` 前缀，复数形式：`enterprise_departments`、`enterprise_user_departments`、`enterprise_dingtalk_bindings`、`enterprise_dingtalk_sync_jobs`、`enterprise_quota_allocations`、`enterprise_admin_actions`、`enterprise_alert_events`、`enterprise_alert_rules`、`enterprise_alert_deliveries`、`enterprise_usage_snapshots`。
- 列名 snake_case：`parent_id`、`dingtalk_dept_id`、`synced_at`、`tenant_id`、`is_active`。
- 主键统一 `id`（GORM 默认），不使用 `AUTO_INCREMENT`/`SERIAL` 字面量；让 GORM 处理跨库主键生成。
- 外键列名格式 `<entity>_id`：`department_id`、`user_id`、`allocation_id`，不使用 `fk_xxx` 形式。
- 索引命名 `idx_<table_short>_<col1>[_<col2>]`，`<table_short>` 去除 `enterprise_` 前缀以保证 ≤ 64 字符。例：`enterprise_user_departments` 上索引为 `idx_user_departments_user_id`、`idx_user_departments_department_id`。
- 唯一索引命名 `uq_<table_short>_<col1>[_<col2>]`：`uq_dingtalk_bindings_corp_user`。
- `tenant_id` V1 仅在新增企业相关表预留（默认 `0`），**不**扩展到 `users`、`logs`、`quota_data`。
- 保留字列（`group`、`key`）若必须使用，沿用 `commonGroupCol`/`commonKeyCol` 变量；企业模块尽量避免引入新的保留字列名。
- 时间戳字段统一 `int64` Unix 秒，列名 `created_at`、`updated_at`、`deleted_at`、`synced_at` 等，与现有 `model/log.go`、`model/user.go` 风格对齐。**不**使用 `time.Time` 字段类型（除 GORM 内建 `gorm.DeletedAt` 外）。
- 状态枚举字段使用 `int`（`status int`，1=enabled, 2=disabled, 3=pending），不使用 bool/字符串；常量集中在 `constant/enterprise.go`。

**API Naming Conventions:**

- 企业管理 API 一律落在 `/api/enterprise/<resource>` 路径下，资源名使用 **kebab-case 复数**：`/api/enterprise/departments`、`/api/enterprise/dingtalk/configs`、`/api/enterprise/dingtalk/sync-jobs`、`/api/enterprise/quota-allocations`、`/api/enterprise/admin-actions`、`/api/enterprise/usage`、`/api/enterprise/alerts/events`、`/api/enterprise/alerts/rules`。
- 路径参数使用 `:id` 风格（与 Gin 一致）：`/api/enterprise/departments/:id`、`/api/enterprise/quota-allocations/:id`。
- query 参数使用 **snake_case**（与现有 `/api/log/**`、`/api/usage/**` 对齐）：`?department_id=1&from=...&to=...&page=1&page_size=20`。
- 路由分组在 `router/enterprise-router.go` 内统一定义，挂载到 `/api/enterprise`，受 `AdminAuth/RootAuth/UserAuth` 中间件控制。
- 不向 `/v1/**`、`/v1beta/**`、`/mj/**`、`/suno/**`、`/pg/**` 添加任何企业字段或路由。
- 响应包装一律 `common.ApiSuccess/ApiError/ApiSuccessI18n/ApiErrorI18n`，HTTP 永远 200。

**Code Naming Conventions:**

- Go 包路径：`model/enterprise`、`service/enterprise`、`controller/enterprise`、`dto/enterprise`、`router/enterprise-router.go`、`constant/enterprise.go`、`middleware/enterprise_*.go`（如 `middleware/enterprise_dept_admin.go`）。
- Go 类型/函数：导出类型 `UpperCamelCase`，包内 `lowerCamelCase`，与现有 `controller/user.go` 一致（如 `Login`、`setupLogin`）。
- Go JSON tag 一律 **snake_case**：`json:"department_id"`、`json:"display_name"`，与 `model/user.go`/`dto/openai_request.go` 对齐。
- Go 文件名 snake_case：`department_service.go`、`dingtalk_sync_task.go`、`quota_allocation_controller.go`。
- Default 前端组件 PascalCase 文件 + PascalCase 默认导出：`DepartmentTree.tsx`、`AllocationDialog.tsx`；hooks 用 camelCase：`useDepartmentTree.ts`。
- Default 前端 feature 目录 kebab-case：`enterprise-organization/`、`enterprise-dingtalk/`、`enterprise-usage/`、`enterprise-alerts/`。
- Default 前端非组件文件全 kebab-case：`api.ts`、`types.ts`、`constants.ts`、`use-department-tree.ts`（与现有 dashboard feature 风格一致）。
- Classic 主题文件遵循 Classic 当前风格（`pages/Enterprise/Department.js` 等 PascalCase 页面），不强行向 Default 对齐。
- Go 业务错误使用 sentinel 形式：`ErrDepartmentNotFound`、`ErrDingTalkSyncInProgress`、`ErrAllocationExceedsBudget`，集中在 `service/enterprise/errors.go`。

### Structure Patterns

**Project Organization:**

企业模块在后端形成完整的"垂直切片"，目录归属如下：

```
constant/enterprise.go               # 企业模块常量
dto/enterprise/                      # DTO 子包
  department.go
  dingtalk.go
  allocation.go
  admin_action.go
  usage.go
  alert.go
model/enterprise/                    # 数据模型子包
  department.go
  user_department.go
  department_role.go
  dingtalk_binding.go
  dingtalk_sync_job.go
  dingtalk_conflict.go
  quota_allocation.go
  admin_action.go
  alert_event.go
  alert_rule.go
  alert_delivery.go
  usage_snapshot.go
  errors.go                          # model 层 sentinel
  migration.go                       # 企业表 AutoMigrate 入口
service/enterprise/                  # 业务服务子包
  department.go
  department_role.go
  dingtalk_token.go                  # corp/app access token 缓存
  dingtalk_oauth.go                  # 钉钉登录
  dingtalk_sync.go                   # 通讯录同步业务逻辑
  dingtalk_sync_task.go              # 后台 ticker 入口
  dingtalk_conflict.go
  quota_allocation.go
  admin_action.go
  usage_aggregation.go
  usage_aggregation_task.go
  usage_export.go                    # CSV / 邮件报告
  alert.go
  alert_dispatch.go
  alert_dispatch_task.go
  scheduler.go                       # 注册并按 IsMasterNode 启动后台任务
  errors.go                          # service 层 sentinel
controller/enterprise/               # HTTP 控制器子包
  department.go
  dingtalk.go
  quota_allocation.go
  admin_action.go
  usage.go
  alert.go
middleware/enterprise_dept_admin.go  # 部门管理员权限中间件
router/enterprise-router.go          # 路由注册（在 router/main.go 中调用）
```

前端 Default 模块按以下 4 个 feature 拆分，每个 feature 内部一律使用七件套：

```
web/default/src/features/enterprise-organization/
  api.ts
  components/                        # 含部门树、配额分配、操作记录等子组件
  hooks/
  lib/
  constants.ts
  types.ts
  index.tsx
web/default/src/features/enterprise-dingtalk/
web/default/src/features/enterprise-usage/
web/default/src/features/enterprise-alerts/
```

Classic 主题不引入 feature-sliced 结构，新增的企业管理入口落在：

```
web/classic/src/pages/Enterprise/
  Department.js
  DingTalk.js
  Allocation.js
  Usage.js
  Alerts.js
web/classic/src/services/enterprise.js
web/classic/src/components/Enterprise/
```

企业 i18n 文案：

- 后端 `i18n/locales/en.yaml`、`zh-CN.yaml`、`zh-TW.yaml` 内的企业 key 统一加前缀 `enterprise.<feature>.<message>`（与现有 yaml 平铺 key 兼容；既有非企业 key 不动）。
- Default 前端 i18n 沿用英文源字符串作为 key，不引入命名空间前缀，与上游兼容。
- Classic 前端 i18n 沿用 Classic 现有中文 key 风格。

**File Structure Patterns:**

- 测试文件采用 Go 标准 `*_test.go` 与源文件**同目录**（与现有 `controller/channel_test_internal_test.go`、`service/error_test.go` 一致），不使用集中 `__tests__/` 目录。
- 配置项默认走管理后台 Option 表 + 专用配置表（`enterprise_dingtalk_configs`），`.env` 仅保留部署级开关与可选默认值。
- OpenAPI：企业 API 进入 `docs/openapi/api.json`（管理后台契约），新增 tags：`Enterprise / Organization`、`Enterprise / DingTalk`、`Enterprise / Quota Allocation`、`Enterprise / Usage`、`Enterprise / Alerts`、`Enterprise / Admin Action`。**不**写入 `docs/openapi/relay.json`。
- 静态资源（钉钉图标等）放置于 `web/default/src/assets/enterprise/`。
- 文档：模块设计文档放在 `docs/enterprise/`（如 `docs/enterprise/department-quota.md`、`docs/enterprise/dingtalk-sync.md`），不污染现有 `docs/` 顶层文件。

### Format Patterns

**API Response Formats:**

控制台/管理后台 API 一律使用现有包装：

```json
{ "success": true,  "message": "",       "data": { } }
{ "success": false, "message": "...",    "data": null     }
```

列表接口的 `data` 形态固定为：

```json
{
  "items": [],
  "total": 123,
  "page": 1,
  "page_size": 20
}
```

- 单资源接口 `data` 直接是资源对象，不再包一层。
- CSV/附件导出使用文件响应（`c.Data(...)` / `c.FileAttachment(...)`），文件命名 `<feature>-<scope>-<yyyymmdd>[-<yyyymmdd>].csv`，例：`usage-department-20260401-20260430.csv`。
- 不引入新的全局错误码字段；业务错误统一通过 `ApiErrorI18n("error.enterprise.<feature>.<reason>")` 返回，Controller 层负责将 service sentinel 映射为 i18n key。
- HTTP 状态码继续维持现有惯例（业务错误也返回 200）；仅在中间件层（鉴权、限流）使用 4xx/5xx，与现有中间件保持一致。

**Data Exchange Formats:**

- JSON 字段命名一律 **snake_case**（前后端对齐 `model/user.go`/`dto/openai_request.go` 风格）。
- 布尔字段直接使用 JSON `true`/`false`；不使用 `1`/`0`。数据库存储侧通过 `commonTrueVal`/`commonFalseVal` 处理跨库差异。
- 时间字段在 JSON 中一律 `int64` Unix 秒，字段名 `created_at`、`updated_at`、`synced_at` 等；前端展示时再用 `dayjs`/`date-fns` 本地化。
- Null 处理：可选标量字段在 DTO 与响应中**一律使用指针 `*int / *uint / *float64 / *bool` + `omitempty`**（与项目 Rule 6 一致）。这适用于所有企业管理 API 的请求/响应 DTO，避免 `0`/`false` 被误吞导致配额分配、同步参数等语义异常。
- 数组字段缺省时返回 `[]` 而非 `null`，前端可以无条件 `.map`。
- 钉钉密钥、access token、suite ticket、refresh token：**永不**出现在前端响应、CSV 导出、日志或错误信息中；写入数据库的密钥字段在 model 层用 `json:"-"` 屏蔽。

### Communication Patterns

**Event System Patterns:**

- V1 不引入正式事件总线/外部消息队列。后台异步通过 `service/enterprise/scheduler.go` 注册的 goroutine + ticker 实现，受 `common.IsMasterNode` 控制。
- 任务"事件"以 DB 状态表为正确性来源，命名风格 `<feature>_<lifecycle>`（仅日志/状态字段层面）：`dingtalk_sync_started`、`dingtalk_sync_completed`、`dingtalk_sync_failed`、`alert_dispatched`、`alert_delivery_failed`。
- 状态字段使用 `int` 枚举常量集中在 `constant/enterprise.go`，例：

```go
const (
    SyncJobStatusPending  = 1
    SyncJobStatusRunning  = 2
    SyncJobStatusSuccess  = 3
    SyncJobStatusFailed   = 4
)
```

- 任务表必须包含：`id`、`type`、`status`、`started_at`、`finished_at`、`error_reason`、`retry_count`、`payload`（JSON 字符串，使用 `common.Marshal/Unmarshal`）。
- 重试策略：失败任务由调度器在固定退避（如 1m → 5m → 30m → 停止）下自动重试，最大重试次数到达后写 `error_reason` 并停止；管理员可通过 `/api/enterprise/dingtalk/sync-jobs/:id/retry` 手动触发。

**State Management Patterns:**

- Default 前端使用 **TanStack Query** 作为服务端状态主战场；本地 UI 状态使用 `useState`/`useReducer`，跨页面状态用现有 Zustand store（如已存在），**不**新增独立 Redux/Jotai。
- TanStack `queryKey` 统一格式 `['enterprise', '<feature>', '<scope>', ...params]`：

```ts
['enterprise', 'organization', 'department-tree']
['enterprise', 'organization', 'department', deptId]
['enterprise', 'organization', 'admin-actions', { from, to, page }]
['enterprise', 'organization', 'quota-allocations', { departmentId, page }]
['enterprise', 'dingtalk', 'config']
['enterprise', 'dingtalk', 'sync-jobs', { page }]
['enterprise', 'usage', 'department-summary', { from, to, tenantId, summarySort, summaryOrder }]
['enterprise', 'usage', 'department-detail', { departmentId, from, to }]
['enterprise', 'usage', 'report-config']
['enterprise', 'alerts', 'events', { departmentId, status, page }]
['enterprise', 'alerts', 'rules']
```

- mutation 成功后通过 `invalidateQueries({ queryKey: ['enterprise', '<feature>'] })` 精确失效，**禁止** `invalidateQueries({ queryKey: ['enterprise'] })` 这种粗粒度全量失效。
- 全局通知（toast）走现有 Default 主题 toast helper（保持与其他 feature 一致）；Classic 走 Semi `Toast`。
- 表单状态使用现有项目已选用的表单方案（如 React Hook Form），不在企业模块内引入新的表单库。
- 不可变更新：所有 `setState` 路径不得直接 mutate 现有对象/数组；接 spread 或 `produce`（如已存在 immer）。

### Process Patterns

**Error Handling Patterns:**

- Go 后端三层错误分工：
  - **Model 层**：返回 `gorm.ErrRecordNotFound` 或本地 sentinel（`model/enterprise/errors.go`）。
  - **Service 层**：返回业务 sentinel（`service/enterprise/errors.go`），并在内部用 `errors.Join`/`fmt.Errorf("...: %w", err)` 携带原因。
  - **Controller 层**：使用 `errors.Is` 匹配 sentinel，调用 `common.ApiErrorI18n("error.enterprise.<feature>.<reason>", ...)`；未识别错误统一调用 `common.ApiError(c, err)` 并写 `common.SysError`。
- 后台任务错误：写入对应任务表 `error_reason` 字段（≤ 1KB，超长截断），不写敏感凭据；同时通过 `common.SysLog/SysError` 输出标准日志。
- 重试是**任务级**而非函数级：service 内部不做指数退避循环，由调度器统一控制。
- 钉钉 API 错误：将钉钉返回的 `errcode` 映射到企业 sentinel 后再上抛；**禁止**将钉钉原始 `errmsg` 直接透传到前端响应。
- 部门管理员权限拒绝：`middleware/enterprise_dept_admin.go` 直接返回 `common.ApiErrorI18n("error.enterprise.permission.dept_admin_required")`，HTTP 仍 200。

**Loading State Patterns:**

- Default：列表/表格 loading 使用 Base UI Skeleton 占位（与 dashboard feature 一致）；详情/表单使用居中 spinner；错误使用 Alert 组件展示 `error.message` 并提供 "重试" 按钮调用 `refetch()`。
- Default：mutation 进行中：触发按钮使用 `disabled + loading icon`；不阻塞其他页面交互。
- Classic：沿用 Semi `Spin` + 局部 `loading` state，错误用 `Toast.error` + 内联 `Banner`。
- 全局 loading（页面切换骨架）已由路由层统一处理，企业模块不再单独引入。
- 长耗时操作默认分两类处理：同步触发类（如钉钉全量同步）返回任务 ID 供前端轮询；部门用量 CSV 导出在 V1 采用同步文件流响应，不单独创建导出任务表。

### Enforcement Guidelines

**All AI Agents MUST:**

- 严格遵循 `CLAUDE.md` 七条规则，特别是：
  - JSON 操作只走 `common.Marshal/Unmarshal/UnmarshalJsonStr/DecodeJson`。
  - 数据库代码三库兼容（SQLite、MySQL ≥ 5.7.8、PostgreSQL ≥ 9.6）；遇 PG 专属操作必须分支 fallback。
  - 不修改/删除任何 `new-api` / `QuantumNous` 标识。
  - 上游 relay 请求 DTO 的可选标量字段一律 `*T + omitempty`。
- 新增企业表使用 GORM `AutoMigrate`，并在 `model/enterprise/migration.go` 集中注册，由 `model/main.go` 在企业模块初始化阶段调用。
- 所有企业管理动作（部门 CRUD、钉钉配置变更、告警规则变更等低频操作）必须通过 `service/enterprise/admin_action.go` 统一写入 `enterprise_admin_actions` 表；配额分配/撤销动作直接落入 `enterprise_quota_allocations` 表（其本身即为审计来源），**不**重复写入 `admin_actions`。
- 所有企业新增表均预留 `tenant_id`（默认 `0`）；查询时不强制按 `tenant_id` 过滤，但写入必须填值，为后续 SaaS 多租户演进做准备。
- 永不修改 `model/log.go`、`controller/log.go`、`router/api-router.go` 中 `/api/log` 路由的现有逻辑；新审计/操作记录页位于企业 feature 内部新页面，不复用 `web/default/src/features/usage-logs/`。
- 永不向 relay 路径（`/v1/**`、`/v1beta/**`、`/mj/**`、`/suno/**`、`/pg/**`、`relay/`）添加部门/企业字段。
- 钉钉相关密钥字段在 model 层使用 `json:"-"`；在 service 层避免落入日志；在 controller 层避免回传响应。

**Pattern Enforcement:**

- Code Review 检查清单（`docs/enterprise/code-review-checklist.md`，由 dev story 创建）：
  - 是否新增表均带 `enterprise_` 前缀？
  - 是否所有 JSON 操作走 `common/json.go` wrapper？
  - 是否含 PG/MySQL 专属 SQL 而无 fallback？
  - 是否触碰 `model/log.go` 等核心文件？
  - 是否在 relay DTO 上添加部门字段？
  - 是否将密钥写入响应/日志/CSV？
  - DTO 可选标量是否使用指针 + `omitempty`？
  - i18n 新文案是否进入 `i18n/locales/*.yaml` 与 `web/default/src/i18n/locales/*.json`？
  - 后台任务是否受 `common.IsMasterNode` 控制？
- 模式违反需在 PR 描述中明确说明并取得 Architect 批准；通过的例外记入 `docs/enterprise/pattern-exceptions.md`。
- 模式更新走标准 architecture amendment：先在本架构文档追加变更块（保留历史），再同步到代码与文档。

### Pattern Examples

**Good Examples:**

路由定义：

```go
// router/enterprise-router.go
func SetEnterpriseRouter(router *gin.Engine) {
    api := router.Group("/api/enterprise")
    api.Use(middleware.UserAuth())
    {
        deptRoute := api.Group("/departments")
        deptRoute.Use(middleware.AdminAuth())
        {
            deptRoute.GET("",       controllerEnt.ListDepartments)
            deptRoute.POST("",      controllerEnt.CreateDepartment)
            deptRoute.GET("/:id",   controllerEnt.GetDepartment)
            deptRoute.PUT("/:id",   controllerEnt.UpdateDepartment)
            deptRoute.DELETE("/:id",controllerEnt.DeleteDepartment)
        }
    }
}
```

DTO 可选字段：

```go
// dto/enterprise/allocation.go
type AllocateQuotaRequest struct {
    DepartmentBudgetId int     `json:"department_budget_id"     validate:"required"`
    UserId             int     `json:"user_id"                  validate:"required"`
    Quota              *int64  `json:"quota,omitempty"          validate:"required"`   // committed_quota for the child wallet
    ExpiresAt          *int64  `json:"expires_at,omitempty"`                           // balance mode only; NULL = follow parent
    Reason             *string `json:"reason,omitempty"`
}
```

TanStack Query：

```ts
// web/default/src/features/enterprise-organization/api.ts
export const useDepartmentTree = () =>
  useQuery({
    queryKey: ['enterprise', 'organization', 'department-tree'],
    queryFn: () => api.get('/api/enterprise/departments?tree=1').then(r => r.data.data),
  });
```

Service sentinel + Controller 映射：

```go
// service/enterprise/errors.go
var (
    ErrDepartmentNotFound      = errors.New("department not found")
    ErrAllocationExceedsBudget = errors.New("allocation exceeds department budget")
)

// controller/enterprise/quota_allocation.go
if errors.Is(err, serviceEnt.ErrAllocationExceedsBudget) {
    common.ApiErrorI18n(c, "error.enterprise.allocation.exceeds_budget")
    return
}
```

**Anti-Patterns:**

- ❌ 直接 `import "encoding/json"` 并调用 `json.Marshal/Unmarshal`（必须走 `common.Marshal/Unmarshal`）。
- ❌ 表名 `departments`（缺前缀）或 `Departments`（大小写）。
- ❌ 在 `model/log.go` 上添加 `DepartmentId` 字段。
- ❌ 在 `dto/openai_request.go`、`relay/channel/**` 中加入 `department_id` 或 `tenant_id`。
- ❌ 将钉钉 `app_secret` 通过 `/api/enterprise/dingtalk/configs` 响应回前端用于"展示当前配置"。
- ❌ Controller 层直接返回 GORM 错误：`common.ApiError(c, gorm.ErrRecordNotFound)`，应映射为 i18n key。
- ❌ TanStack `invalidateQueries({ queryKey: ['enterprise'] })` 全量刷新。
- ❌ 在 service 内部写 `for { time.Sleep(...); retry() }` 重试循环；重试归调度器统一控制。
- ❌ 使用 `gorm.Expr` 直接拼 PG 专属 SQL（如 `JSONB` 操作符）而无 SQLite/MySQL fallback。
- ❌ 后端时间字段使用 `time.Time` 与 `created_at` 列（应统一 `int64`）。
- ❌ 后台任务在所有节点启动而不判断 `common.IsMasterNode`。
- ❌ 使用 `AUTO_INCREMENT` / `SERIAL` 字面量而非 GORM 默认主键。
- ❌ 配额分配动作既写 `enterprise_quota_allocations` 又写 `enterprise_admin_actions`（重复审计源）。

## Project Structure & Boundaries

### Requirements → Components Mapping

V1 的 26 个 FR 按 PRD 的 6 个能力域归入企业模块的不同子组件。下表为权威映射（dev 创建 epic/story 时按此分配文件归属）：

| 能力域 | FR | 后端模型 | 后端服务 | 后端控制器 | 路由 | 前端 Default Feature | 前端 Classic 页面 |
|---|---|---|---|---|---|---|---|
| 组织与部门模型 | FR-1, FR-2, FR-3, FR-4 | `model/enterprise/department.go`、`user_department.go`、`department_role.go` | `service/enterprise/department.go`、`department_role.go` | `controller/enterprise/department.go` | `/api/enterprise/departments`、`/api/enterprise/departments/:id/roles` | `enterprise-organization/` | `pages/Enterprise/Department.js` |
| 钉钉登录与通讯录同步 | FR-5, FR-6, FR-7, FR-8 | `model/enterprise/dingtalk_binding.go`、`dingtalk_sync_job.go`、`dingtalk_conflict.go`、`dingtalk_config.go` | `service/enterprise/dingtalk_token.go`、`dingtalk_oauth.go`、`dingtalk_sync.go`、`dingtalk_sync_task.go`、`dingtalk_conflict.go` | `controller/enterprise/dingtalk.go` | `/api/enterprise/dingtalk/configs`、`/api/enterprise/dingtalk/sync-jobs`、`/api/enterprise/dingtalk/conflicts`、`/api/oauth/dingtalk` | `enterprise-dingtalk/` | `pages/Enterprise/DingTalk.js` |
| 部门配额与额度控制 | FR-9, FR-10, FR-11, FR-12 | `model/enterprise/quota_allocation.go`、`department_budget.go` + 扩展现有 `model/subscription.go`（V1.1 加 `source_type` / `source_allocation_id`） | `service/enterprise/quota_allocation.go`、`department_budget.go`、`wallet_subscription.go`、`wallet_state_sync_task.go`、`balance_expiry_task.go` | `controller/enterprise/quota_allocation.go`、`department_budget.go` | `/api/enterprise/quota-allocations`、`/api/enterprise/departments/:id/budget` | `enterprise-organization/`（配额分配 tab） | `pages/Enterprise/Allocation.js` |
| 组织用量看板与导出 | FR-13, FR-14, FR-15, FR-16 | `model/enterprise/usage_snapshot.go`、`usage_report_job.go` | `service/enterprise/usage_aggregation.go`、`usage_aggregation_task.go`、`usage_export.go`、`usage_report_task.go` | `controller/enterprise/usage.go` | `/api/enterprise/usage/department-summary`、`/api/enterprise/usage/department-detail`、`/api/enterprise/usage/export`、`/api/enterprise/usage/reports` | `enterprise-usage/` | `pages/Enterprise/Usage.js` |
| 内容监控告警 | FR-17, FR-18, FR-19, FR-20 | `model/enterprise/alert_event.go`、`alert_rule.go`、`alert_delivery.go` | `service/enterprise/alert.go`、`alert_dispatch.go`、`alert_dispatch_task.go` | `controller/enterprise/alert.go` | `/api/enterprise/alerts/events`、`/api/enterprise/alerts/rules`、`/api/enterprise/alerts/deliveries`、`/api/enterprise/alerts/department-summary` | `enterprise-alerts/` | `pages/Enterprise/Alerts.js` |
| 分层预算治理与额度审批 | FR-21, FR-22, FR-23, FR-24, FR-25, FR-26 | `model/enterprise/department_role.go`、`budget_delegation.go`、`quota_request.go` | `service/enterprise/department_role.go`、`budget_delegation.go`、`quota_request.go`、`quota_allocation.go` | `controller/enterprise/department_role.go`、`budget_delegation.go`、`quota_request.go` | `/api/enterprise/departments/:id/owners`、`/api/enterprise/budget-delegations`、`/api/enterprise/quota-requests` | `enterprise-organization/`、`enterprise-usage/`、`enterprise-alerts/` | `pages/Enterprise/Allocation.js`、`pages/Enterprise/Usage.js` |

**横切关注点（不属于任何单一域）：**

| 关注点 | 位置 |
|---|---|
| 部门管理员鉴权 | `middleware/enterprise_dept_admin.go` |
| 企业模块常量 | `constant/enterprise.go` |
| 企业模块 i18n 文案 | `i18n/locales/{en,zh-CN,zh-TW}.yaml`（key 前缀 `enterprise.*`），`web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` |
| 后台任务调度入口 | `service/enterprise/scheduler.go`，由 `main.go` 在 `IsMasterNode` 下启动；调度任务清单含 `dingtalk_sync_task`、`usage_aggregation_task`、`usage_report_task`、`alert_dispatch_task`、`wallet_state_sync_task`（V1.1）、`balance_expiry_task`（V1.1） |
| 企业表 AutoMigrate | `model/enterprise/migration.go`，由 `model/main.go` 在 `InitDB` 后调用；同时触发 `subscriptions` 表 V1.1 列扩展（`source_type`、`source_allocation_id`） |
| 管理动作审计（部门/钉钉/告警等低频管理） | `service/enterprise/admin_action.go` → `enterprise_admin_actions` 表 |
| 配额分配审计（即业务事实，自身即审计源） | `service/enterprise/quota_allocation.go` → `enterprise_quota_allocations` 表 |
| OpenAPI 契约 | `docs/openapi/api.json`（管理后台），新增 tags `Enterprise / *` |

### Complete Project Directory Structure

只展示企业模块**新增/触碰**的路径；现有非企业代码维持现状不显示。

```
new-api/
├── main.go                                          # ⚠️ 一行新增：启动企业 scheduler（IsMasterNode 下）
├── CLAUDE.md                                        # 0 改动
├── go.mod / go.sum                                  # 0 改动（V1 不引入新 Go 依赖；钉钉 SDK 用标准 net/http）
├── constant/
│   └── enterprise.go                                # 🆕 状态枚举、角色枚举、上下文 key、cache key
├── dto/
│   └── enterprise/                                  # 🆕 子包
│       ├── department.go                            # CreateDeptRequest / UpdateDeptRequest / DeptListItem / DeptTreeNode
│       ├── dingtalk.go                              # DingTalkConfigRequest / SyncJobItem / ConflictItem
│       ├── allocation.go                            # AllocateQuotaRequest / AllocationListItem
│       ├── admin_action.go                          # AdminActionListItem
│       ├── usage.go                                 # DeptUsageQuery / DeptUsageSummary / DeptUsageDetail
│       └── alert.go                                 # AlertEventQuery / AlertRuleRequest / AlertDeliveryItem
├── model/
│   ├── main.go                                      # ⚠️ 两处改动：(1) AutoMigrate 列表追加：调用 enterprise.Migrate(DB)；(2) subscriptions 表追加 source_type/source_allocation_id 两列迁移（V1.1）
│   └── enterprise/                                  # 🆕 子包
│       ├── migration.go                             # 集中调用所有企业表 AutoMigrate，并触发 subscriptions 表的 V1.1 列扩展迁移
│       ├── errors.go                                # model 层 sentinel
│       ├── department.go                            # enterprise_departments
│       ├── user_department.go                       # enterprise_user_departments（用户↔部门 N:N，无 is_primary 字段）
│       ├── department_role.go                       # enterprise_department_roles（部门管理员）
│       ├── department_budget.go                     # enterprise_department_budgets（balance/subscription 双类型父预算池）
│       ├── dingtalk_config.go                       # enterprise_dingtalk_configs（corp_id, app_key 等）
│       ├── dingtalk_binding.go                      # enterprise_dingtalk_bindings（用户↔钉钉身份）
│       ├── dingtalk_sync_job.go                     # enterprise_dingtalk_sync_jobs（同步任务状态）
│       ├── dingtalk_conflict.go                     # enterprise_dingtalk_conflicts（待人工确认冲突）
│       ├── quota_allocation.go                      # enterprise_quota_allocations（分配流水＝审计源；含 wallet_id 反查）
│       ├── admin_action.go                          # enterprise_admin_actions（低频管理动作审计）
│       ├── usage_snapshot.go                        # enterprise_usage_snapshots（聚合表）
│       ├── usage_report_job.go                      # enterprise_usage_report_jobs（定期邮件状态）
│       ├── alert_event.go                           # enterprise_alert_events
│       ├── alert_rule.go                            # enterprise_alert_rules
│       └── alert_delivery.go                        # enterprise_alert_deliveries
├── service/
│   └── enterprise/                                  # 🆕 子包
│       ├── errors.go                                # service 层 sentinel
│       ├── scheduler.go                             # 启动并管理所有企业后台任务（受 IsMasterNode 控制）
│       ├── department.go                            # 部门 CRUD、移动、停用、树查询、缓存
│       ├── department_role.go                       # 部门管理员任命/撤销
│       ├── department_membership.go                 # 用户↔部门 N:N 关系维护（V1.1：取代旧 department_resolver.go）
│       ├── dingtalk_token.go                        # corp/app access token 获取与 cachex 缓存
│       ├── dingtalk_client.go                       # 钉钉 OpenAPI HTTP 客户端封装（含错误码映射）
│       ├── dingtalk_oauth.go                        # 钉钉登录回调与本地账号绑定/创建
│       ├── dingtalk_sync.go                         # 通讯录全量/增量同步业务流程
│       ├── dingtalk_sync_task.go                    # 调度入口（ticker、IsMasterNode 守卫）
│       ├── dingtalk_conflict.go                     # 冲突解析与人工确认动作
│       ├── quota_allocation.go                      # 部门预算分配：在用户名下创建 wallet/subscription 行 + 写流水
│       ├── department_budget.go                     # 父预算池 CRUD、暂停/恢复、cycle_quota 调整、状态联动
│       ├── wallet_subscription.go                   # 子 wallet 创建、撤销、状态同步父池（balance/subscription 双类型）
│       ├── wallet_state_sync_task.go                # 父池暂停/过期 → 所有子 wallet 联动（IsMasterNode 守卫）
│       ├── balance_expiry_task.go                   # balance 子 wallet expires_at 到期回收（best-effort，IsMasterNode 守卫）
│       ├── admin_action.go                          # 写入 enterprise_admin_actions 的统一入口
│       ├── usage_aggregation.go                     # 按 enterprise_user_departments 多部门重复计入聚合消费日志
│       ├── usage_aggregation_task.go                # 聚合后台任务
│       ├── usage_export.go                          # CSV 生成（流式、避免内存峰值）
│       ├── usage_report_task.go                     # 定期邮件/通知报告任务
│       ├── alert.go                                 # 告警事件、规则匹配
│       ├── alert_dispatch.go                        # 告警投递（邮件/webhook/Bark/Gotify 等）
│       └── alert_dispatch_task.go                   # 告警投递后台任务
├── controller/
│   └── enterprise/                                  # 🆕 子包
│       ├── department.go                            # 部门 CRUD、树、移动、角色
│       ├── dingtalk.go                              # 配置、登录回调、同步触发/重试、冲突
│       ├── quota_allocation.go                      # 分配/撤销/查询
│       ├── admin_action.go                          # 操作记录列表
│       ├── usage.go                                 # 部门用量 summary/detail/export/reports
│       └── alert.go                                 # 事件/规则/投递/部门概览
├── middleware/
│   ├── enterprise_dept_admin.go                     # 🆕 部门管理员权限中间件
│   └── auth.go                                      # 0 改动
├── router/
│   ├── main.go                                      # ⚠️ 一行新增：SetEnterpriseRouter(router)
│   ├── api-router.go                                # 0 改动
│   ├── enterprise-router.go                         # 🆕 /api/enterprise/** 全部路由
│   └── relay-router.go                              # 0 改动（明令禁止触碰）
├── pkg/
│   ├── billingexpr/                                 # 0 改动（明令禁止触碰）
│   ├── cachex/                                      # 0 改动（直接使用现有 API）
│   └── ionet/                                       # 0 改动
├── i18n/
│   └── locales/
│       ├── en.yaml                                  # ⚠️ 追加 enterprise.* key
│       ├── zh-CN.yaml                               # ⚠️ 追加 enterprise.* key
│       └── zh-TW.yaml                               # ⚠️ 追加 enterprise.* key
├── docs/
│   ├── openapi/
│   │   ├── api.json                                 # ⚠️ 追加 Enterprise / * tags 与所有 /api/enterprise/** path
│   │   └── relay.json                               # 0 改动（明令禁止触碰）
│   └── enterprise/                                  # 🆕 模块设计文档
│       ├── README.md                                # 模块导览
│       ├── department-quota.md                      # 部门预算与配额分配设计
│       ├── dingtalk-integration.md                  # 钉钉 Token、OAuth、同步设计
│       ├── usage-aggregation.md                     # 用量聚合与归因
│       ├── alerting.md                              # 告警事件与投递
│       ├── data-model.md                            # 表清单与索引说明
│       ├── code-review-checklist.md                 # 评审清单（Step 5 提到）
│       └── pattern-exceptions.md                    # 通过的模式例外记录
├── web/
│   ├── default/
│   │   └── src/
│   │       ├── features/
│   │       │   ├── enterprise-organization/        # 🆕
│   │       │   │   ├── api.ts
│   │       │   │   ├── components/
│   │       │   │   │   ├── DepartmentTree.tsx
│   │       │   │   │   ├── DepartmentDetailDrawer.tsx
│   │       │   │   │   ├── DepartmentRoleAssignDialog.tsx
│   │       │   │   │   ├── AllocationDialog.tsx
│   │       │   │   │   ├── AllocationLedgerTable.tsx
│   │       │   │   │   ├── AdminActionLogTable.tsx
│   │       │   │   │   └── DepartmentBudgetCard.tsx
│   │       │   │   ├── hooks/
│   │       │   │   │   ├── use-department-tree.ts
│   │       │   │   │   ├── use-allocation-ledger.ts
│   │       │   │   │   └── use-admin-action-log.ts
│   │       │   │   ├── lib/
│   │       │   │   │   └── tree-utils.ts
│   │       │   │   ├── constants.ts
│   │       │   │   ├── types.ts
│   │       │   │   └── index.tsx                    # 路由入口（tabs：组织 / 配额 / 操作记录）
│   │       │   ├── enterprise-dingtalk/             # 🆕
│   │       │   │   ├── api.ts
│   │       │   │   ├── components/
│   │       │   │   │   ├── DingTalkConfigForm.tsx
│   │       │   │   │   ├── SyncJobsTable.tsx
│   │       │   │   │   ├── SyncJobDetailDrawer.tsx
│   │       │   │   │   └── ConflictResolutionDialog.tsx
│   │       │   │   ├── hooks/
│   │       │   │   ├── lib/
│   │       │   │   ├── constants.ts
│   │       │   │   ├── types.ts
│   │       │   │   └── index.tsx
│   │       │   ├── enterprise-usage/                # 🆕
│   │       │   │   ├── api.ts
│   │       │   │   ├── components/
│   │       │   │   │   ├── DepartmentSummaryTable.tsx
│   │       │   │   │   ├── DepartmentDetailPanel.tsx
│   │       │   │   │   ├── ExportDialog.tsx
│   │       │   │   │   └── ReportSettingsForm.tsx
│   │       │   │   ├── hooks/
│   │       │   │   ├── lib/
│   │       │   │   ├── constants.ts
│   │       │   │   ├── types.ts
│   │       │   │   └── index.tsx
│   │       │   └── enterprise-alerts/               # 🆕
│   │       │       ├── api.ts
│   │       │       ├── components/
│   │       │       │   ├── AlertEventsTable.tsx
│   │       │       │   ├── AlertRuleEditor.tsx
│   │       │       │   ├── DeliveryLogTable.tsx
│   │       │       │   └── DepartmentRiskSummary.tsx
│   │       │       ├── hooks/
│   │       │       ├── lib/
│   │       │       ├── constants.ts
│   │       │       ├── types.ts
│   │       │       └── index.tsx
│   │       ├── i18n/
│   │       │   └── locales/
│   │       │       ├── en.json                      # ⚠️ 追加企业 key（英文源字符串）
│   │       │       ├── zh.json                      # ⚠️
│   │       │       ├── fr.json                      # ⚠️
│   │       │       ├── ru.json                      # ⚠️
│   │       │       ├── ja.json                      # ⚠️
│   │       │       └── vi.json                      # ⚠️
│   │       ├── routes/                              # ⚠️ 新增 _authenticated/enterprise.* 路由文件
│   │       │   └── _authenticated/
│   │       │       ├── enterprise.organization.tsx
│   │       │       ├── enterprise.dingtalk.tsx
│   │       │       ├── enterprise.usage.tsx
│   │       │       └── enterprise.alerts.tsx
│   │       ├── modules/                             # ⚠️ sidebar/nav 配置追加企业入口
│   │       └── assets/
│   │           └── enterprise/                      # 🆕 钉钉/部门相关图标
│   └── classic/
│       └── src/
│           ├── pages/
│           │   └── Enterprise/                      # 🆕
│           │       ├── Department.js
│           │       ├── DingTalk.js
│           │       ├── Allocation.js
│           │       ├── Usage.js
│           │       └── Alerts.js
│           ├── components/
│           │   └── Enterprise/                      # 🆕
│           │       ├── DepartmentTree.js
│           │       ├── AllocationDialog.js
│           │       └── ...
│           ├── services/
│           │   └── enterprise.js                    # 🆕 axios helpers
│           └── App.jsx                              # ⚠️ 路由注册 + 侧边栏新增企业入口
└── deploy/                                          # 0 改动结构；文档侧补钉钉环境变量与多节点说明
```

**改动统计：**

- 🆕 新增目录：12 个（含子包与 feature）
- 🆕 新增文件：~85 个
- ⚠️ 现有文件触碰：6 个（`main.go`、`router/main.go`、`model/main.go`、3 个 yaml + 6 个 json i18n、`docs/openapi/api.json`、Default 路由配置 + sidebar、Classic `App.jsx`）
- ❌ 明令零改动：`relay/**`、`pkg/billingexpr/**`、`pkg/cachex/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`、所有 `relay/channel/**`

### Architectural Boundaries

**API Boundaries:**

- **Relay API（不可触碰）**：`/v1/**`、`/v1beta/**`、`/mj/**`、`/suno/**`、`/pg/**`、`/kling/v1/**`、`/jimeng/**`。OpenAI 兼容协议、错误格式、`TokenAuth` 行为完全冻结。企业模块以**只读旁路**消费 logs/quota，不写入 relay 路径。
- **管理后台 API（既有）**：`/api/**`（除 `/api/enterprise/**`）。现有 user、token、log、channel、option、subscription 等接口冻结，企业模块**不**修改其逻辑、签名或响应。
- **企业 API（新增）**：`/api/enterprise/**`，受 `UserAuth` 兜底鉴权 + 资源级 `AdminAuth/RootAuth/enterprise_dept_admin` 中间件细化。
- **OAuth 回调（新增）**：`/api/oauth/dingtalk` 复用 `controller/oauth.go` 的回调注册风格但实现独立在 `controller/enterprise/dingtalk.go`，复用 `setupLogin` 完成 session。

**Component Boundaries:**

- **Backend 内部分层**：Router → Controller → Service → Model 严格单向。Controller 不直接访问 GORM；Service 不直接 `c *gin.Context` 取参（参数从 controller 解出后传入）；Model 仅 GORM 调用与表定义。
- **Service 子模块互访**：企业 service 之间通过包内函数调用，不引入接口抽象层除非有 ≥2 个实现（避免过度工程）。例外：`alert_dispatch.go` 的投递通道（邮件/webhook/Bark/Gotify）可定义 `Dispatcher` 接口便于扩展。
- **Backend → Frontend**：仅通过 `/api/enterprise/**` REST 通信；不引入 SSE/WebSocket（V1 用轮询 `sync-jobs/:id` 替代）。
- **Frontend 内部**：4 个 enterprise feature 互相**不直接 import**；共享类型放 `types.ts`，共享 UI 原子放在已有 `components/ui/`，企业 feature 之间如需要联动通过路由参数与 TanStack queryKey invalidation 协调。

**Service Boundaries:**

- **企业 ↔ 现有 user**：通过 `model.User` 公共方法（如 `IncreaseUserQuota` / `DecreaseUserQuota` / `GetUserById`）；**不**直接修改 `model/user.go`。
- **企业 ↔ 现有 quota**：仅在 `quota_allocation.go` 内部通过现有 quota helper 增减个人 quota；不在 relay 路径插入部门扣减。
- **企业 ↔ 现有 log**：用量聚合**只读**消费 `logs` 表（`SELECT` 已有列），不新增 `logs` 字段，不修改 `LogConsume`。归因来自消费时刻写入 `logs.username` 的用户，再通过 `enterprise_usage_snapshots` 与历史快照交叉解析。
- **企业 ↔ 现有 notify**：告警投递复用 `service/notify-limit.go`、`service/notify_*` 等通知能力与限流；企业模块只生成"投递任务"，不重新实现 SMTP/HTTP 客户端。
- **企业 ↔ 现有 OAuth**：钉钉登录复用 `setupLogin`，不复用 `custom_oauth_providers`；与 GitHub/Discord/OIDC 并列但独立。

**Data Boundaries:**

- **企业表**：全部 `enterprise_*` 前缀，独立 schema 边界，不与既有表外键约束（GORM 关系仅在企业表内部）。与 `users.id` 的关联通过应用层 join，不建跨表外键。
- **缓存边界**：钉钉 access token 用 `pkg/cachex` 缓存（按 corp_id 命名），TTL 取钉钉返回值减 5 分钟；部门树与配额状态使用应用内缓存 + Redis 失效广播（如有 Redis）。无 Redis 时单节点本地缓存退化为短 TTL。
- **Redis 角色**：可选增强（同步互斥锁、聚合任务去重、告警限流去重）。**正确性来源**始终是 DB 状态表；Redis 故障不应导致数据不一致，仅可能导致重复任务或重复告警（由幂等键防御）。
- **Logs 表**：只读源，企业模块禁止写入；用量聚合通过 `usage_aggregation_task` 按时间窗扫描后写入 `enterprise_usage_snapshots`。

### Integration Points

**Internal Communication:**

```
                    ┌─────────────────────────────┐
                    │  Browser (Default / Classic)│
                    └──────────────┬──────────────┘
                                   │ HTTP /api/enterprise/**
                    ┌──────────────▼──────────────┐
                    │       Gin Router            │
                    │   /api/enterprise/**        │
                    └──────────────┬──────────────┘
                                   │
              ┌────────────────────┼────────────────────┐
              │                    │                    │
       UserAuth (existing)   AdminAuth        enterprise_dept_admin (new)
              │                    │                    │
              └────────────────────┼────────────────────┘
                                   │
                    ┌──────────────▼──────────────┐
                    │ controller/enterprise/*.go  │
                    │   解参 → 调 service → 包响应 │
                    └──────────────┬──────────────┘
                                   │
                    ┌──────────────▼──────────────┐
                    │  service/enterprise/*.go    │
                    │   sentinel error / 业务逻辑 │
                    └──────────────┬──────────────┘
                  ┌────────────────┼────────────────────────────┐
                  │                │                            │
       model/enterprise/    现有 model.User /         pkg/cachex / Redis /
       新增表 GORM           quota helper               common.IsMasterNode
                  │                │                            │
                  └────────────────┴────────────────────────────┘

后台任务 (受 common.IsMasterNode 守卫)：
  scheduler.go ──┬──► dingtalk_sync_task        (ticker)
                 ├──► usage_aggregation_task    (ticker)
                 ├──► usage_report_task         (cron 风格 ticker)
                 ├──► alert_dispatch_task       (ticker)
                 ├──► wallet_state_sync_task    (ticker, V1.1)
                 └──► balance_expiry_task       (ticker, V1.1)
```

**External Integrations:**

| 外部系统 | 协议 | 用途 | 入口文件 |
|---|---|---|---|
| 钉钉 OpenAPI | HTTPS REST | corp/app token、OAuth、通讯录、消息 | `service/enterprise/dingtalk_client.go` |
| 邮件 SMTP | 现有 `service/email.go` | 定期用量报告、告警 | `service/enterprise/usage_report_task.go`、`alert_dispatch.go` |
| Webhook / Bark / Gotify / 钉钉机器人 | 现有 `service/notify_*` | 告警通道 | `alert_dispatch.go` |
| Redis（可选） | 现有 `common/redis.go` | 锁、缓存、限流 | 各 service 按需调用 |

**Data Flow:**

1. **钉钉登录流**：浏览器 → `/api/oauth/dingtalk` → 校验 state → 钉钉换 user_info → 命中 `enterprise_dingtalk_bindings` 找 `user_id` → 用 `setupLogin` 写 session。绑定缺失时按邮箱/手机号匹配；冲突进 `enterprise_dingtalk_conflicts`，登录走"建本地账号 + 等待管理员合并"路径。
2. **通讯录同步流**：scheduler ticker → `dingtalk_sync_task` → 写 `enterprise_dingtalk_sync_jobs(status=running)` → `dingtalk_sync.go` 全量/增量 → 写入/更新 `enterprise_departments` & `enterprise_user_departments`（N:N 平等关系，无 is_primary）→ 冲突进 `enterprise_dingtalk_conflicts` → 写 job(status=success/failed, error_reason)。
3. **配额分配流（V1.1）**：管理员 UI → `POST /api/enterprise/quota-allocations` → `quota_allocation.go` 校验父预算池剩余（balance：`remaining >= committed_quota`；subscription：`Σ children.committed_quota + new ≤ cycle_quota`）→ DB 事务：(a) 减 `enterprise_department_budgets.remaining`（balance）或 `allocated_total += committed_quota`（subscription）；(b) **在目标用户名下创建 subscriptions 行**（balance：`is_subscription=false, remain_quota=committed_quota, expires_at=snapshot`；subscription：`is_subscription=true, quota=committed_quota, cycle_period 跟随父`，扩展列 `source_type='enterprise_allocation'` + `source_allocation_id` 指回 allocation 行）；(c) 写 `enterprise_quota_allocations(wallet_id=新 subscription.id, status=active)` → 返回成功。失败回滚事务，全程**不**触及 relay 路径。
4. **预算委派流（V1.3）**：部门负责人 UI → `POST /api/enterprise/budget-delegations` 或等价接口 → 校验来源部门是目标部门的祖先、来源预算池存在足够可委派额度、预算池类型仍为 `balance / subscription` → DB 事务：(a) 关闭或保留旧 delegation 事实；(b) 在目标部门预算池侧创建或更新新的委派事实；(c) 记录独立 delegation ledger 与审计快照。该流仅发生在部门预算池之间，**不**直接创建成员 wallet，后续成员分配仍走既有 `quota_allocation.go`。
4. **wallet 状态联动流（V1.1）**：scheduler ticker → `wallet_state_sync_task`（受 `IsMasterNode` 守卫）→ 扫 `enterprise_department_budgets` 中 status 变更的父池 → 批量 update 该父池下所有 `enterprise_quota_allocations` 子记录与对应 subscriptions 行（暂停/恢复/过期联动）→ 写审计入口。subscription 类型的周期 reset 复用**现有 subscription reset 任务**（`token.remain_quota = token.quota`），零新代码。
5. **balance 过期回收流（V1.1）**：scheduler ticker → `balance_expiry_task`（受 `IsMasterNode` 守卫）→ 扫 `subscriptions` 中 `source_type='enterprise_allocation' AND is_subscription=false AND expires_at <= now()` → 对每条 wallet 执行 best-effort 回收：`parent.remaining += min(wallet.remain_quota, max(0, ...))`，wallet 标记 `revoked/expired`，写流水。
6. **用量聚合流（V1.1）**：scheduler ticker → `usage_aggregation_task` → 按时间窗扫 `logs` → 对每条 log `INNER JOIN enterprise_user_departments` 展开为 `(log_row, dept_id)` 多行（同一用户多部门 → 多行重复计入）→ 归并写入 `enterprise_usage_snapshots`；未归属用户单独归入 `dept_id IS NULL` 桶。前端看板 `GET /api/enterprise/usage/department-summary` 命中聚合表，不直接查 `logs`。
5. **告警流**：现有敏感词/过滤产生事件 → 写 `enterprise_alert_events`（旁路写入，不阻塞 relay）→ scheduler ticker `alert_dispatch_task` → 匹配 `enterprise_alert_rules` → 解析部门收件人 → 调 `notify_*` 投递 → 写 `enterprise_alert_deliveries`。
6. **管理动作审计流**：`controller/enterprise/*.go` 在所有低频管理 mutation 成功后调 `service.WriteAdminAction(ctx, action)` → 写 `enterprise_admin_actions`。配额分配**不**走此路径（自身在 `enterprise_quota_allocations` 已是审计源）。
7. **额度申请审批流（V1.3）**：员工从余额 / 钱包相关 UI 发起额度申请 → `POST /api/enterprise/quota-requests` → 记录目标部门、目标预算池模式/池子、申请额度、申请原因与幂等键 → 系统按目标部门的有效负责人集合路由单步审批 → 审批通过后调用既有 `quota_allocation.go` 完成自动分配 → 写 request / approval / fulfillment 审计与通知状态。重复审批或重复回调不得产生重复分配。企业组织工作台负责审批、治理时间线与通知状态展示。

8. **预算池类型约束边界（V1.3A）**：`DepartmentBudget.Type` 继续保持单值，因此单个预算池实例仍然只能是 `balance` 或 `subscription` 之一；但同一部门下允许并存多个不同类型预算池。类型兼容性校验应作用于当前操作选中的预算池，不应扩展成“部门级单类型”限制。

### File Organization Patterns

**Configuration Files:**

- 钉钉应用配置（corp_id/app_key/app_secret）：存 `enterprise_dingtalk_configs` 表（密钥列 `json:"-"`），管理后台 `/api/enterprise/dingtalk/configs` 读写。
- 部署级开关（如 `ENTERPRISE_FEATURES_ENABLED`、定时任务间隔覆盖）：`.env` + `common/env.go`。
- 业务参数（如配额分配最大金额、聚合任务窗口）：管理后台 Option 表（`model/option.go`），key 命名 `enterprise.<feature>.<param>`。

**Source Organization:**

- 后端：层 + 域，先按层（model/service/controller）再按域（enterprise/）。同层企业代码全部集中在 `*/enterprise/` 子包，便于上游合并冲突边界清晰。
- 前端：feature-sliced，企业 4 个 feature 平级独立；共享原子继续位于 `components/ui/`、`lib/`。
- 所有 i18n 文案文件位置不变，仅追加 key。

**Test Organization:**

- Go 测试与源文件**同目录**（`*_test.go`），与现有项目一致。
- 关键测试目标：
  - `service/enterprise/quota_allocation_test.go`：并发分配、wallet 创建/撤销事务回滚、`Σ children ≤ parent` 铁律校验、跨库 boolean/列名兼容。
  - `service/enterprise/wallet_state_sync_test.go`：父池暂停/过期 → 所有子 wallet 联动；REPLACE 语义验证（复用现有 subscription reset）；balance 过期回收 best-effort。
  - `service/enterprise/usage_aggregation_test.go`：跨日窗口、多部门用户重复计入断言（同一用户多部门归属 → 各部门聚合行均含其消费）、未归属用户单独归桶。
  - `service/enterprise/dingtalk_sync_test.go`：全量/增量、冲突生成、错误重试。
  - `service/enterprise/usage_aggregation_test.go`：跨日窗口、多部门用户重复计入断言（同一用户多部门归属 → 各部门聚合行均含其消费）、未归属用户单独归桶。
  - `service/enterprise/scheduler_test.go`：`IsMasterNode` 守卫、不重复启动。
- 前端 Default 优先编译/类型检查覆盖；关键交互（配额分配 dialog、同步触发）可加 React Testing Library 组件测试，但 V1 不强制 E2E。

**Asset Organization:**

- 钉钉品牌 logo 放在 `web/default/src/assets/enterprise/dingtalk.svg`；Classic 复用同一资源（通过 webpack/vite 别名或重复一份）。
- CSV 导出无静态模板，由 `usage_export.go` 流式生成。
- 邮件模板（如有）放 `service/enterprise/templates/`（Go embed），不放 `web/`。

### Development Workflow Integration

**Development Server Structure:**

- 后端：`go run main.go` 启动单进程，企业 scheduler 在 master 节点自动启动；本地开发默认是 master 节点。
- Default 前端：`cd web/default && bun run dev` 起独立 Rsbuild dev server，通过 `FRONTEND_BASE_URL` 反向代理后端 `/api/enterprise/**`。
- Classic 前端：`cd web/classic && bun run dev`（Vite），同样代理。
- 钉钉本地联调：使用钉钉开发者后台的"测试企业"或 ngrok 暴露 `/api/oauth/dingtalk`；密钥通过 `/api/enterprise/dingtalk/configs` 配置而非 `.env`。

**Build Process Structure:**

- 后端 `go build ./...`：企业模块进入同一二进制，无独立产物。
- Default 前端 `bun run build` → `web/default/dist/` → 由 Go `go:embed` 嵌入主二进制（`web/default/embed.go` 既有机制，不修改）。
- Classic 前端 `bun run build` → `web/classic/dist/` → 同上。
- 文档：`docs/openapi/api.json` 通过 swag 生成或手工维护（沿用现有项目流程）；企业 tags 进入既有 `api.json`。

**Deployment Structure:**

- 单二进制 / Docker / systemd / Electron：均无新组件，企业能力随核心一起发布。
- 多节点部署：仅 master 节点运行 scheduler；其他节点提供 HTTP 服务。`common.IsMasterNode` 已是项目既有机制，无需改造。
- 数据库迁移：`InitDB` → `enterprise.Migrate(DB)`，仅 master 执行。新增表通过 GORM `AutoMigrate`，所有列变更必须遵循 `CLAUDE.md` Rule 2（SQLite ALTER 限制等）。
- 回滚：企业模块新增表/列通过下线开关（`ENTERPRISE_FEATURES_ENABLED=false`）禁用前端入口与后台任务；数据保留不删除。

## Architecture Validation Results

### Coherence Validation ✅

**Decision Compatibility:**

- 技术栈与决策一致：选择沿用现有 monorepo（Step 3）→ 所有决策（Step 4）以"在既有边界内增量扩展"为前提 → 实现模式（Step 5）锁定与现有 `model/user.go`、`controller/*.go`、`common.ApiSuccess` 等惯例对齐 → 项目结构（Step 6）所有新文件落入 `*/enterprise/` 子包，未与既有目录冲突。
- Go/Gin/GORM/三库兼容/Bun/React 19 + Rsbuild/React 18 + Vite 等版本约束在 Step 3 Verified，Step 5/6 未引入与之冲突的新依赖（V1 不引入新 Go 依赖；钉钉用 `net/http`）。
- 部门配额"非 relay 路径扣费"的核心决策（Step 4）与"配额分配流走独立事务、用量聚合只读 logs"（Step 6 Data Flow #3、#4）形成闭环，无矛盾。
- 审计模式（Step 5 Enforcement：配额分配单源 `enterprise_quota_allocations`，其他低频管理写 `enterprise_admin_actions`）与项目结构（Step 6 横切关注点）完全一致。

**Pattern Consistency:**

- 命名：表前缀 `enterprise_*`、列 snake_case、JSON tag snake_case、Go 文件名 snake_case、Default feature kebab-case、Classic PascalCase 页面 —— Step 5 与 Step 6 全部一致引用，无漂移。
- 错误处理：Step 5 定义"sentinel + i18n key 映射"，Step 6 在多个文件（`service/enterprise/errors.go`、`controller/enterprise/*.go`）落地为具体责任，链路完整。
- TanStack queryKey 规范 `['enterprise', '<feature>', '<scope>', ...]`：Step 5 给出 11 个 key 实例，Step 6 在 4 个 feature 的 `api.ts/hooks/` 中均有归属。
- 后台任务：Step 5 要求 `IsMasterNode` 守卫 + 状态表为正确性源 + 任务级重试，Step 6 `service/enterprise/scheduler.go` 与 4 个 `*_task.go` 文件位置完全对齐。

**Structure Alignment:**

- 路由分层：`router/enterprise-router.go`（新文件，挂 `/api/enterprise/**`）+ `router/main.go` 一行新增调用 —— 与 Step 4 "企业 API 属于管理后台接口"、Step 5 "kebab-case 复数 + UserAuth + 资源级中间件" 全程一致。
- 数据流：6 个核心数据流（钉钉登录 / 通讯录同步 / 配额分配 / 用量聚合 / 告警 / 管理动作审计）均在 Step 6 Integration Points 显式落到具体文件 + 表名 + 中间件 + 触发器。
- 边界：Relay/管理后台/企业 API/数据/缓存/Logs 表 6 类边界均有"可触碰" vs "明令禁止"对应表述，避免后续 Agent 误改。

### Requirements Coverage Validation ✅

**Functional Requirements Coverage:**

| FR | 需求摘要 | 架构落地 |
|---|---|---|
| FR-1 | 建立企业组织与部门层级 | `model/enterprise/department.go`（adjacency-list） + `service/enterprise/department.go` |
| FR-2 | 维护用户↔部门 N:N 关系（V1.1：取消主部门） | `model/enterprise/user_department.go` + `service/enterprise/department_membership.go` |
| FR-3 | 部门 vs Group 语义分离 | Step 4 数据架构明确不动 `User.Group/Token.Group`；Step 5 术语固定；Step 6 现有 `users` 表零改动 |
| FR-4 | `tenant_id` SaaS 演进字段 | Step 5 全部企业新增表预留 `tenant_id`（默认 0）；Step 4 V1 不扩展 `users/logs/quota_data` |
| FR-5 | 配置钉钉企业应用 | `model/enterprise/dingtalk_config.go` + `controller/enterprise/dingtalk.go`（密钥 `json:"-"`） |
| FR-6 | 钉钉 OAuth 登录 | `service/enterprise/dingtalk_oauth.go` + 复用 `setupLogin` + `dingtalk_bindings` |
| FR-7 | 同步钉钉部门与成员 | `dingtalk_sync.go` + `dingtalk_sync_task.go` + `dingtalk_sync_jobs` 状态表 |
| FR-8 | 处理钉钉组织变更 | 增量同步逻辑 + `dingtalk_conflicts` 表 + 管理员人工确认动作 |
| FR-9 | 设置部门预算池（balance / subscription 双类型） | `enterprise_department_budgets` 表 + `service/enterprise/department_budget.go` |
| FR-10 | API 消费链路执行部门配额校验 | **V1.1 重新解释**：分配 = 在用户名下创建 wallet（subscriptions 表新行 + `source_type='enterprise_allocation'`），relay 路径继续按 wallet 优先级扣减；零 relay 改动 |
| FR-11 | 保留计费表达式与倍率语义 | `pkg/billingexpr/**` 0 改动（Step 6 明令禁止触碰） |
| FR-12 | 展示部门配额状态 | `enterprise-organization` feature 的 `DepartmentBudgetCard` + `AllocationLedgerTable` |
| FR-13 | 按部门聚合用量（V1.1：多部门重复计入） | `usage_aggregation_task.go` + `enterprise_usage_snapshots` 聚合表（按 `enterprise_user_departments` join 展开） |
| FR-14 | 查看部门用量详情（V1.1：UI 标注不可加和） | `enterprise-usage` feature 的 `DepartmentDetailPanel` + `/api/enterprise/usage/department-detail` + i18n key `enterprise.usage.multi_dept_disclaimer` |
| FR-15 | 导出部门用量 CSV | `service/enterprise/usage_export.go`（流式生成）+ `ExportDialog.tsx` |
| FR-16 | 发送定期用量报告 | `usage_report_task.go` + `enterprise_usage_report_jobs` + 复用 `service/email.go` |
| FR-17 | 记录内容风险事件 | `enterprise_alert_events` 表 + 现有敏感词链路旁路写入 |
| FR-18 | 配置告警规则与通道 | `enterprise_alert_rules` + `AlertRuleEditor.tsx` |
| FR-19 | 发送可追溯告警通知 | `alert_dispatch.go` + `alert_deliveries` 表 + 复用 `notify_*` 通道 |
| FR-20 | 展示部门风险概览（V1.1：多部门重复计入） | `DepartmentRiskSummary.tsx` + `/api/enterprise/alerts/department-summary`；风险率分母按当前部门口径，未归属用户单独显示 |
| FR-21 | 建立部门负责人权限模型 | `model/enterprise/department_role.go` 扩展来源/覆盖语义 + `service/enterprise/department_role.go` 生效优先级解析 |
| FR-22 | 支持包含子部门的查询作用域 | `department_scope_resolver.go` 或等价 service + summary APIs 的 `include_descendants` 参数 |
| FR-23 | 上级预算池向下级部门预算池分配 | 新增 delegation ledger / service；成员 wallet 分配仍由 `quota_allocation.go` 承接 |
| FR-24 | 成员 allocation 调整 / 取消 / 回收 | 扩展 `quota_allocation.go` 与 wallet lifecycle 审计状态语义 |
| FR-25 | 员工额度申请与单步审批 | 新增 request / approval workflow service，审批通过后复用 `quota_allocation.go` |
| FR-26 | 治理动作审计与钉钉通知 | 复用 `enterprise_admin_actions` + `notify_*` / 钉钉机器人，通知失败不阻塞核心事务 |

**所有 26 个 FR 均有架构落地**（FR-10 经由 Step 4 PRD Alignment 重新定义为预算分配模式，需 PRD 同步修正）。

**Non-Functional Requirements Coverage:**

| NFR | 架构对应 |
|---|---|
| 兼容性（SQLite/MySQL 5.7.8+/PG 9.6+） | Step 5 数据库命名规则 + GORM `AutoMigrate` + `commonTrueVal/commonFalseVal/commonGroupCol` 沿用；Step 6 测试目标包含跨库验证 |
| 安全（密钥不明文） | Step 5 "钉钉密钥/token 永不返回前端、不进日志、不进 CSV"；Step 6 `model/enterprise/dingtalk_config.go` 列 `json:"-"` |
| 隐私（风险事件最小可见） | Step 4 权限隔离 + Step 5 部门管理员中间件 + Step 6 `enterprise-alerts` UI 按部门权限渲染 |
| 可靠性（同步失败不阻塞登录/API） | Step 4 Sync Safety + Step 6 数据流 #1（最近成功快照） |
| 一致性（用户 quota 与部门统计最终一致） | Step 4 No Relay-Path Department Deduction + 预算分配事务（Step 6 数据流 #3） |
| 可观测性 | 所有任务/告警/分配均有状态表 + `error_reason` 字段（Step 5 Communication Patterns） |
| 性能（100 万 logs / 30 天 P95 < 3s） | Step 4 + Step 6 用量看板查 `enterprise_usage_snapshots` 聚合表，**不**直接扫 `logs` |
| 国际化 | Step 5 + Step 6 i18n 文件清单完整（后端 yaml 加 `enterprise.*` 前缀，Default JSON 用英文源字符串） |

**约束与护栏（PRD §9）覆盖：**

- ✅ JSON 必须走 `common/json.go`（Step 5 Enforcement Guidelines、Pattern Examples、Anti-Patterns）
- ✅ 不新增名为 `group` 的字段（Step 5 数据库命名规则、Step 4 数据架构）
- ✅ 不破坏 `User.Group/Token.Group/渠道分组/计费倍率`（Step 6 明令零改动 + 部门 vs Group 术语分离）
- ✅ `pkg/billingexpr` 不改语义（Step 4/6 明令零改动）
- ✅ `new-api` / `QuantumNous` 受保护标识（Step 5 + 项目 Rule 5）
- ✅ V1 不为 SaaS 牺牲交付（Step 4 Deferred Decisions）

**开放问题（PRD §13）状态：**

| OQ | 状态 | 说明 |
|---|---|---|
| OQ-1 产品名 | 不影响架构 | 占位符替换不改变结构 |
| OQ-2 无钉钉时本地组织 UI | **Deferred**（架构已具备，未交付）| `enterprise-organization` feature 的部门 CRUD 已支持手工管理；V1 默认不暴露入口，关闭钉钉时可启用 |
| OQ-3 `tenant_id` 范围 | **架构已锁定**：V1 仅企业表预留，不扩展 `users/logs/quota_data`（Step 4 Important Decisions） |
| OQ-4 钉钉机器人/Webhook 告警必选性 | **架构已具备**：通道由 `alert_dispatch.go` 适配器扩展；V1 必选邮件，其他通道作为可选；交付时机由 PRD 定 |
| OQ-5 钉钉 OAuth 复用方式 | **架构已锁定**：钉钉专用 integration module，不复用 `custom_oauth_providers`（Step 4 Auth & Security） |
| OQ-6 部门配额周期 | ✅ **V1.1 已关闭** | 部门预算池采用 balance（一次性余额，可设 expires_at）/ subscription（订阅周期，与 user subscription 同构 daily/weekly/monthly/custom_seconds）双类型；子分配落地为用户名下 wallet。详见 CP-5。 |
| OQ-7 员工多部门时的归属规则 | ✅ **V1.1 已关闭** | 取消"主部门"概念，用户↔部门 N:N 平等关系；多部门用户的预算可被多个部门管理员独立分配（叠加享受）；用量归因采用方案 B（多部门重复计入，UI 标注不可加和）。详见 CP-2 / CP-14。 |

### Implementation Readiness Validation ✅

**Decision Completeness:**

- 12 个核心架构决策（Step 4）+ 13 个一致性模式 CP（Step 5）+ 6 个能力域 × 完整目录（Step 6）覆盖了 PRD 所有 FR/NFR/约束。
- 关键示例（路由、DTO、TanStack Query、Service sentinel）在 Step 5 Pattern Examples 给出了可拷贝代码块。

**Structure Completeness:**

- 后端：`model/enterprise/`、`service/enterprise/`、`controller/enterprise/`、`dto/enterprise/` 4 个子包共 ~50 个文件均已命名落位。
- 前端：4 个 enterprise feature × 7 件套 + 4 个路由文件 + sidebar 配置触碰点已确认。
- 集成：6 个数据流图 + 4 类外部系统 + 边界规则均在 Step 6。

**Pattern Completeness:**

- 13 个潜在 Agent 冲突点（CP-1~CP-13）全部在 Step 5 锁定，含正反例。
- 错误、loading、并发、审计、跨库、密钥等过程模式均覆盖。

### Gap Analysis Results

**Critical Gaps（阻塞实现，需在 epic/story 前关闭）:**

1. **PRD FR-2 措辞同步**：从"主部门关系"改为"用户↔部门 N:N 关系"。
2. **PRD FR-10 措辞同步**：从"部门参与运行时预扣费/结算"改为"部门预算分配给成员，**在用户名下创建独立 wallet**（subscriptions 表新行），relay 路径继续按 wallet 优先级扣减"。
3. **PRD FR-13/14/20 口径补充**：增加"多部门用户消费在所有归属部门重复计入，部门间不可加和"的口径说明，UI 文案进入 i18n key `enterprise.usage.multi_dept_disclaimer`。

> OQ-6 / OQ-7 在 V1.1 中已通过 CP-2 / CP-5 / CP-14 关闭，不再阻塞 epic/story。

**Important Gaps（不阻塞，但补完更顺利）:**

1. **钉钉 access token 限流策略未明示**：钉钉 OpenAPI 有 QPS 限制，若大企业全量同步并发触发 `dingtalk_client.go` 需 token bucket 限流。建议在 dev story 阶段补充 `pkg/cachex` 或 Redis token bucket 设计；架构本身不阻塞。
2. **聚合任务窗口与回填策略未细化**：`usage_aggregation_task` 默认按多长窗口（5 min / 1 h）扫 logs、是否支持手动回填特定日期范围 —— 由 dev story 阶段在 `docs/enterprise/usage-aggregation.md` 中细化。
3. **subscriptions 表 V1.1 列扩展迁移策略**：三库 ALTER 兼容路径需在迁移文件中明确（SQLite `ADD COLUMN` 限制、MySQL/PG 区别）。建议 dev story 第一篇配额分配卡片中给出具体迁移代码模板。
4. **告警规则 DSL 形式未定**：`alert_rules` 表的 `condition` 字段格式（JSON / 简易表达式 / 单条件）未规范。V1 建议先支持单条件白名单（事件类型 + 部门 + 阈值），由 dev story 关闭。

**Nice-to-Have Gaps（V1 可选，不影响验收）:**

1. **审计动作的 diff 字段格式**：`enterprise_admin_actions.payload` 是否存"前/后"完整对象 vs 字段级 diff —— 不影响 V1 功能，可后续优化。
2. **CSV 大文件导出的异步化**：当前设计为同步流式响应；若部门日量级大可改为生成任务 + 下载链接。
3. **Default 路由懒加载**：4 个 enterprise feature 是否懒加载分包 —— 取决于 Rsbuild 现有 chunk 策略，可后续优化。

### Validation Issues Addressed

- **审计割裂担忧（用户提出）**：通过 Step 5 CP-7 方案 3 解决：配额分配单源 + 低频管理动作单源 + 前端在 `enterprise-organization` feature 内部新页面统一展示，**不**触碰现有 `model/log.go`、`controller/log.go`、`web/default/src/features/usage-logs/`。文件级影响清单已在该轮对话中完整证明。
- **API 路由分隔符不一致（架构师扫描发现）**：现状混用 kebab/snake；通过 Step 5 CP-1 锁定企业模块统一 kebab-case，避免新代码加重既有不一致。
- **DTO 可选字段零值丢失（项目 Rule 6 + 架构师强化）**：Step 5 CP-13 把 Rule 6 从 relay DTO 扩展到所有企业管理 API DTO，明确 `*int/*bool/*float64 + omitempty`，避免分配额度 `0` / 同步 `false` 被误吞。

### Architecture Completeness Checklist

**Requirements Analysis**

- [x] Project context thoroughly analyzed
- [x] Scale and complexity assessed
- [x] Technical constraints identified
- [x] Cross-cutting concerns mapped

**Architectural Decisions**

- [x] Critical decisions documented with versions
- [x] Technology stack fully specified
- [x] Integration patterns defined
- [x] Performance considerations addressed

**Implementation Patterns**

- [x] Naming conventions established
- [x] Structure patterns defined
- [x] Communication patterns specified
- [x] Process patterns documented

**Project Structure**

- [x] Complete directory structure defined
- [x] Component boundaries established
- [x] Integration points mapped
- [x] Requirements to structure mapping complete

### Architecture Readiness Assessment

**Overall Status:** **READY**（V1.1 修订后）

> 选定原因：16 项 checklist 全部 `[x]`；V1.1 修订关闭了 OQ-6 / OQ-7（通过 CP-2 / CP-5 / CP-14），剩余 Critical Gaps 仅为 PRD 文本同步项（FR-2 / FR-10 / FR-13/14/20 措辞），这些**同步责任在 PRD，但架构已锁定唯一可行落地路径**。可直接进入 epic/story 创建（PRD 文本可与 Epic 1 并行修订，不阻塞 Epic 3 之前的 story 落地）。

**Confidence Level:** **High**

依据：

- 现有 monorepo 代码风格、命名、JSON tag、API 包装、跨库工具均已实地核验。
- 14 个 CP（V1.1 新增 CP-14）全部回答了 "AI Agent 可能做出不同选择" 的具体场景。
- 6 个数据流均落到文件 + 表 + 中间件粒度，无 hand-wave；V1.1 新增 wallet 状态联动流（#4）与 balance 过期回收流（#5）。
- 所有 26 个 FR + 8 个 NFR + 6 个约束护栏均在表格中显式映射。
- V1.1 wallet 模型直接复用现有 subscriptions 表 + 现有 reset 任务，零 relay 改动。

**Key Strengths:**

- **棕地兼容优先**：明令禁止触碰 `relay/**`、`pkg/billingexpr/**`、`model/log.go` 等核心文件，最大化保留与上游 new-api 主线合并能力。
- **审计单源 + UI 统一**：避免传统"双写审计"的一致性陷阱，同时通过前端页面消除运维割裂感。
- **配额分配脱离 relay 路径**：把 PRD 中"运行时部门扣费"重新定义为"管理后台预算分配 → 在用户名下创建独立 wallet（subscriptions 表新行）"，复用现有 wallet 优先级与 reset 任务，绕开 relay 并发与计费表达式的复杂性，仍满足部门治理目标。
- **N:N 平等关系（V1.1）**：取消"主部门"概念后，多部门用户的预算可被多个部门管理员独立分配（叠加享受），符合"组织里 1 个人服务多个部门"的真实场景。
- **三库兼容贯穿**：所有迁移、布尔、保留字、JSON 列处理均沿用项目既有 `commonTrueVal/commonGroupCol/UsingPostgreSQL` 等抽象，无 PG 专属 SQL 风险。
- **多节点安全**：所有后台任务统一受 `common.IsMasterNode` 守卫，调度器单入口，无重复执行风险。

**Areas for Future Enhancement:**

- 部门 wallet 参与 relay 预扣费/结算的更细颗粒度归因（V2 可选，目前已通过现有 wallet 优先级机制承载）。
- 闭包表替代 adjacency-list（仅当看板性能不达标时）。
- SaaS 多租户全链路隔离（V2，需扩展 `tenant_id` 至 users/logs/quota_data）。
- 飞书/企微作为第二企业身份源（V2，沿用 `enterprise-dingtalk` 模式新增 feature）。
- 外部消息队列替代 ticker（V2，规模上来后）。
- 告警规则 DSL 升级（V2，复杂条件表达式）。
- CSV 异步导出 + 下载链接（V2，大数据量场景）。
- Default 路由懒加载分包（V1 后期优化）。
- 用量归因方案 C（按 wallet 来源分摊每笔消费，部门间可加和）作为可选模式（V2，schema 兼容当前 `enterprise_quota_allocations.wallet_id` 反查路径）。

### Implementation Handoff

**AI Agent Guidelines:**

- 严格遵守 `CLAUDE.md` 七条规则（JSON wrapper、跨库兼容、保护标识、relay DTO 指针、计费表达式不改、文件依赖说明）。
- 所有企业相关代码必须使用 `enterprise_*` 表前缀、`*/enterprise/` 子包路径、`/api/enterprise/**` 路由前缀。
- **唯一例外**：现有 `subscriptions` 表加 2 列（`source_type`、`source_allocation_id`），不加 `enterprise_` 前缀（属于现有表扩展）；所有企业模块代码读写该表时通过 `service/enterprise/wallet_subscription.go` 包装，不直接散落到其他位置。
- 严禁触碰 `relay/**`、`pkg/billingexpr/**`、`pkg/cachex/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`、所有 `relay/channel/**`。
- DTO 可选标量一律指针 + `omitempty`；后台任务一律受 `common.IsMasterNode` 守卫；密钥一律 `json:"-"` + 不落日志。
- TanStack queryKey 一律 `['enterprise', '<feature>', '<scope>', ...]`；mutation invalidate 必须精确到 feature 级。
- 配额分配仅写 `enterprise_quota_allocations`（自身即审计源）；其他低频管理动作走 `service.WriteAdminAction` → `enterprise_admin_actions`。
- **用量归因（V1.1）**：聚合任务必须按 `enterprise_user_departments` join 展开，多部门用户重复计入，不要"取一个部门"或"按 user_id 唯一归因"。
- **wallet 联动（V1.1）**：父预算池状态变更通过 `wallet_state_sync_task` 批量同步到子 wallet；不要在 controller 层手工逐个 update 子 wallet。
- 所有架构疑问回到本文档查证；本文档无答案时升级到架构师，不要自行决策。

**First Implementation Priority:**

进入 epic/story 创建之前，**必须先在 PRD 阶段完成**（V1.1 修订后，OQ-6/7 已关闭，剩余以下三项）：

1. **FR-2 措辞同步**：从"主部门关系"改为"用户↔部门 N:N 关系"（无主部门字段）。
2. **FR-10 措辞同步**：从"在 relay 消费链路扣减部门配额"改为"部门管理员把预算分配给成员，**在用户名下创建独立 wallet（subscriptions 表新行 + `source_type='enterprise_allocation'`）**；relay 路径继续按现有 wallet 优先级扣减"。
3. **FR-13/14/20 口径补充**：明确"多部门用户消费在所有归属部门重复计入，部门间不可加和"，UI 文案进入 i18n key `enterprise.usage.multi_dept_disclaimer`。

> ✅ **OQ-6 / OQ-7 在架构 V1.1 修订中已关闭**（详见 CP-2 / CP-5 / CP-14），不再阻塞 epic/story 创建。

PRD 关闭以上三项后，建议的 Epic 顺序（与 Step 4 Implementation Sequence 对齐）：

1. **Epic 1: 企业组织基础**：部门、用户↔部门 N:N 关系、部门角色、`enterprise_admin_actions` 审计入口、`middleware/enterprise_dept_admin.go`、Default `enterprise-organization` feature 部门管理 tab。
2. **Epic 2: 钉钉集成**：`dingtalk_config/binding/sync_jobs/conflicts` 表 + token 缓存 + OAuth 登录 + 全量/增量同步（建立 N:N 部门关系）+ Default `enterprise-dingtalk` feature。
3. **Epic 3: 部门配额分配（V1.1 wallet 模型）**：`enterprise_department_budgets`（balance/subscription 双类型）+ `enterprise_quota_allocations` + `subscriptions` 表 V1.1 列扩展迁移 + `wallet_subscription.go` + `wallet_state_sync_task` + `balance_expiry_task` + 分配/撤销 controller + Default `enterprise-organization` 配额 tab + Classic 必要入口。
4. **Epic 4: 组织用量看板（V1.1 多部门重复计入）**：`enterprise_usage_snapshots` + 聚合任务（按 `enterprise_user_departments` join 展开）+ 看板/详情/CSV/定期报告 + i18n disclaimer + Default `enterprise-usage` feature。
5. **Epic 5: 内容监控告警**：`enterprise_alert_events/rules/deliveries` + 投递任务 + Default `enterprise-alerts` feature + 部门风险概览（V1.1 多部门重复计入）。
6. **Epic 6: 企业治理视图重构与账号标识修正**：树驱动治理视图、部门上下文操作、username 受控修改与统一展示规则。
7. **Epic 7: 分层预算治理与额度审批**：部门负责人优先级、包含子部门 summary scope、预算委派、成员 allocation 治理、员工额度申请单步审批与治理动作通知。

每个 Epic 内部可拆分为 3–6 个 story，story 级别再细化 `enterprise_admin_actions` 的具体类型枚举、`alert_rules` 的 DSL 形式、聚合窗口、`subscriptions` 表跨库迁移代码等 Important Gaps 项。

---

## Workflow Completion

### 工作流状态

- **状态：** complete（V1.1）
- **完成日期：** 2026-05-27（V1.0 完成 + V1.1 修订同日完成）
- **完成步骤：** 1, 2, 3, 4, 5, 6, 7, 8
- **修订内容（V1.1）：** OQ-6 / OQ-7 关闭，新增 CP-14（用量归因方案 B），重写 CP-2 / CP-5，新增 wallet 状态联动与 balance 过期回收任务。
- **输出文档：** `_bmad-output/planning-artifacts/architecture.md`
- **下游消费者：** Epic 拆分（`bmad-create-epic`）、Story 创建（`bmad-create-story`）、实现 Agent

### 本次架构工作交付清单

✅ **Project Context Analysis（Step 2）** —— 把 26 个 FR、8 个 NFR、6 条约束护栏映射到棕地代码骨架，明确不可触碰区与允许扩展区。

✅ **Brownfield Continuation Strategy（Step 3）** —— 锁定"沿用 monorepo + 子包扩展 + 表前缀 + 路由前缀"路径，避免新建独立服务或第二个 monorepo 带来的合并成本。

✅ **13 个 Critical Decision Points（Step 4）** —— 涵盖部门层级、`Group` 边界、`tenant_id` 预留、钉钉模块边界、配额分配模型（重定义 FR-10）、看板聚合策略、内容告警生命周期、审计单源、跨节点调度、缓存层、前端 feature 切分、i18n 流程、Classic/Default 同步范围。每个 CP 都给出"选定方案 + 拒绝方案 + 影响半径"。

✅ **Implementation Patterns & Consistency Rules（Step 5）** —— 表前缀 `enterprise_`、JSON 列存 + `application_in_memory` 索引、`*T + omitempty` DTO 模式、Sentinel error + i18n key 映射、TanStack queryKey `['enterprise', '<feature>', '<scope>', ...]` 约定、`common.IsMasterNode` 调度守卫、`pkg/cachex` 双层缓存、敏感字段 `json:"-"` + 不落日志的统一模式。

✅ **Project Structure & Boundaries（Step 6）** —— 后端新增 12 个文件目录、前端 Default 新增 4 个 feature、Classic 仅补必要入口、文档 `docs/enterprise/**` 落位、6 条数据流（钉钉登录、组织同步、配额分配、用量聚合、CSV 导出、内容告警）端到端落到文件粒度。

✅ **Architecture Validation（Step 7）** —— Coherence + Coverage + Readiness 三视角验证全部通过，所有 20 FR / 8 NFR / 6 约束在架构层有显式承接，差距分析按 Critical / Important / Nice-to-have 分级，OQ-6/7 schema 双向兼容。

### 进入实现前的必做事项（顺序敏感，V1.1 修订）

**[必做 1] PRD 阶段同步以下措辞修正**（OQ-6/7 已在架构 V1.1 中关闭，不再列入）：
1. **FR-2**：从"主部门关系"改为"用户↔部门 N:N 关系"。
2. **FR-10**：从"在 relay 消费链路扣减部门配额"改为"部门管理员把预算分配给成员，**在用户名下创建独立 wallet（subscriptions 表新行）**；relay 路径继续按现有 wallet 优先级扣减"。
3. **FR-13/14/20**：补充"多部门用户消费在所有归属部门重复计入，部门间不可加和"的口径说明，UI 文案进入 i18n key `enterprise.usage.multi_dept_disclaimer`。
4. **FR-9**：补充部门预算池支持 balance / subscription 双类型，与现有 user subscription 同构（daily/weekly/monthly/custom_seconds）。

**[必做 2] 用 `bmad-create-epic` 按 Step 7 提议的 6 个 Epic 顺序生成 Epic 文档**。Epic 1 → 6 之间存在依赖（如 Epic 2 依赖 Epic 1 的部门表与角色，Epic 3 依赖 subscriptions 表 V1.1 列扩展迁移），建议串行启动而非并行。

**[必做 3] Epic 拆分后，用 `bmad-create-story` 把每个 Epic 拆为 Story**。每条 Story 都引用本架构文档对应 CP 编号、文件路径与 i18n key 范围，避免 AI Agent 重新发明轮子。

### 推荐进入下一阶段的命令

```
# 1. 查看 bmad 所有可用 skill 与建议路径
/bmad-help

# 2. 进入 Epic 拆分（必须先 PRD 关闭 OQ-6/7 + FR-10）
/bmad-create-epic

# 3. 单条 Epic 完成后进入 Story 创建
/bmad-create-story
```

### 后续访问与维护

- **唯一真源：** 本文档（`_bmad-output/planning-artifacts/architecture.md`）。后续所有 Epic / Story / 实现 PR 都必须引用本文档的 CP 编号或文件路径。
- **变更治理：** 若 Epic 阶段发现架构与实际代码冲突，回到本文档修订对应 CP，更新 `lastStep` / `updated` 时间戳，并在 PR 描述中标注"架构变更：CP-N 修订"。
- **PRD 与架构耦合：** OQ-6 / OQ-7 在 PRD 关闭后，对应 FR 验收口径变化需同步回本架构 Step 4 的 CP-2、CP-5。
- **棕地兼容承诺：** 本架构所有设计均不修改 `relay/**`、`pkg/billingexpr/**`、`pkg/cachex/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`、`relay/channel/**`。后续如需触碰，必须升级到架构层评审，不在 Story 层自由决策。

---

_Architecture workflow complete. Document ready for downstream Epic / Story / Implementation phases._
