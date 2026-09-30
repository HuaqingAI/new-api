---
baseline_commit: d22f394f7efce531c8a38a6c2dad9218ec38486d
---

# Story 1.1: 查看企业部门树

Status: done

## Story

As a 管理员,
I want 查看企业部门树和部门状态,
so that 我可以确认组织结构已被系统正确承载。

## Acceptance Criteria

1. **部门树展示**：Given 企业模块已启用，When 管理员打开企业组织页面，Then 系统显示部门树、部门名称、父子关系、状态、外部来源标识和同步异常状态；And 支持至少 3 层部门结构，部门停用或删除后不物理删除，历史名称可追溯。
2. **空状态**：Given 企业模块已启用但尚未完成任何部门同步或手工创建，When 管理员打开企业组织页面，Then API 返回空数组而非 `null`；And Default 前端显示空状态和可执行入口，引导有权限的管理员进入钉钉配置或同步流程。
3. **迁移与 schema**：Given 新增企业部门表，When 执行迁移，Then 表结构兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+；And 表名使用 `enterprise_` 前缀，预留 `tenant_id`，不新增 `group` 字段。

## Tasks / Subtasks

- [x] 新增企业部门后端基础切片 (AC: 1, 3)
  - [x] 创建 `constant/enterprise.go`，集中定义部门状态、来源类型、同步状态等 `int` 枚举；不要扩展 `users.role` 或复用 `Group`。
  - [x] 创建 `model/enterprise/department.go`，表名固定为 `enterprise_departments`，采用邻接表：`id`、`tenant_id`、`name`、`parent_id`、`status`、`source_type`、`external_id`、`sync_status`、`sync_error`、`name_history`、`created_at`、`updated_at`、`deleted_at`。
  - [x] `parent_id` 使用可空指针或零值约定，但 API 输出必须能表达根节点；`name_history` 存 JSON 字符串，marshal/unmarshal 只能用 `common.Marshal` / `common.Unmarshal`。
  - [x] 创建 `model/enterprise/migration.go`，集中 `AutoMigrate(&Department{})`；在 `model/main.go` 的普通迁移与 fast 迁移路径中接入企业迁移。
  - [x] 为 `tenant_id`、`parent_id`、`source_type + external_id`、`status` 建立 GORM 索引；索引名遵循 `idx_departments_*` / `uq_departments_*` 且长度控制在 64 字符以内。

- [x] 实现部门树查询 service 与 DTO (AC: 1, 2)
  - [x] 创建 `dto/enterprise/department.go`，定义 `DepartmentTreeNode`，JSON tag 全部 snake_case；子节点字段缺省返回 `[]`，不得返回 `null`。
  - [x] 创建 `service/enterprise/errors.go`，加入 `ErrDepartmentNotFound` 等 sentinel；创建 `service/enterprise/department.go`，实现只读树查询。
  - [x] 查询必须包含启用、停用、同步异常节点；不得物理过滤已停用部门。软删除/历史追溯策略需保证“钉钉删除或停用后本地记录不物理删除”。
  - [x] 构树逻辑在内存中按 `parent_id` 组装，支持至少 3 层；对孤儿节点使用稳定降级策略（作为根节点返回并携带异常状态，或返回明确业务错误，二选一并测试覆盖）。
  - [x] 空表时 service 返回空 slice，controller/API 最终响应 `data: []`。

- [x] 新增企业部门 API 与 OpenAPI (AC: 1, 2)
  - [x] 创建 `controller/enterprise/department.go`，`GET /api/enterprise/departments/tree` 调 service 后使用 `common.ApiSuccess(c, items)` 返回。
  - [x] 创建 `router/enterprise-router.go`，企业路由统一挂 `/api/enterprise`，本故事只注册 `departments/tree`；在 `router/api-router.go` 的 `/api` 分组内调用企业路由注册，沿用 gzip、rate limit、body cleanup 等既有中间件。
  - [x] 后端权限至少使用 `middleware.AdminAuth()`；前端按钮可见性不能替代后端鉴权。
  - [x] 更新 `docs/openapi/api.json`，只加入管理后台 API tag/path；不得修改 `docs/openapi/relay.json`。
  - [x] 控制器不得直接访问 GORM；错误由 service sentinel 映射为 `common.ApiErrorI18n("error.enterprise.organization.<reason>")` 或现有通用错误。

- [x] 实现 Default 前端企业组织页面 (AC: 1, 2)
  - [x] 新增 `web/default/src/features/enterprise-organization/`：`api.ts`、`types.ts`、`constants.ts`、`hooks/use-department-tree.ts`、`components/DepartmentTree.tsx`、`index.tsx`。
  - [x] 使用现有 `api` helper 调 `/api/enterprise/departments/tree`；服务端状态使用 TanStack Query，queryKey 固定为 `['enterprise', 'organization', 'department-tree']`。
  - [x] 新增路由 `web/default/src/routes/_authenticated/enterprise/organization.tsx` 或符合当前 file-router 约定的等价路径，进入前用 `ROLE.ADMIN` 校验，无权限 redirect `/403`。
  - [x] 在 `web/default/src/hooks/use-sidebar-data.ts` 的 Admin 分组增加“Enterprise Organization”入口；如采用嵌套侧栏，还需新增配置并注册到 `web/default/src/components/layout/lib/sidebar-view-registry.ts`。
  - [x] UI 显示部门名称、父子层级、状态、外部来源标识、同步异常状态；空数组时显示空状态和两个可执行入口：进入钉钉配置/同步页面（可先路由到后续占位或禁用态）与后续手工创建入口（若本故事不交付创建，则必须明确禁用）。
  - [x] 不把企业组织能力塞进现有 Group、Users、Usage Logs 页面。

- [x] 完成 i18n 与受保护标识约束 (AC: 1, 2)
  - [x] Default 前端新增英文源字符串 key，并同步到 `web/default/src/i18n/locales/{en,zh,fr,ja,ru,vi}.json`；优先运行 `cd web/default && bun run i18n:sync`。
  - [x] 后端新增错误文案进入 `i18n/locales/{en,zh-CN,zh-TW}.yaml`，key 前缀使用 `enterprise.organization.*` 或 `error.enterprise.organization.*`。
  - [x] 不删除、替换、改名任何 `new-api` / `QuantumNous` 品牌、版权、导入路径、包名或元数据。

- [x] 测试与验证 (AC: 1, 2, 3)
  - [x] 为 `model/enterprise` / `service/enterprise` 添加同目录 `*_test.go`，覆盖三层树、空表返回 `[]`、停用/异常节点保留、历史名称 JSON 解析、孤儿节点策略。
  - [x] 迁移测试至少覆盖 SQLite；MySQL/PostgreSQL 兼容性通过 GORM tag、无 DB 专属 SQL、无 `ALTER COLUMN`、无 `AUTO_INCREMENT`/`SERIAL` 字面量证明。
  - [x] 前端至少通过 `cd web/default && bun run typecheck`；如改动 i18n，运行 `bun run i18n:sync` 并检查六种 locale 都有新增 key。
  - [x] 后端运行 `go test ./model/enterprise ./service/enterprise ./controller/enterprise`；若 router 或 model/main 改动导致包级影响，补跑 `go test ./model ./router ./controller/... ./service/...`。

## Dev Notes

### Context Loaded

- 已加载 `{epics_content}`：`_bmad-output/planning-artifacts/epics.md`。
- 已加载 `{architecture_content}`：`_bmad-output/planning-artifacts/architecture.md`。
- 已加载 `{prd_content}`：`_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md` 与 `addendum.md`。
- 未发现独立 UX 文档；UX 要求来自 Epic/PRD/Architecture。
- 未发现 `project-context.md`，persistent fact glob 无可加载文件。
- 本故事是 Epic 1 第一条，无 previous story intelligence。

### Architecture Guardrails

- 后端遵循 Router -> Controller -> Service -> Model；企业代码放在 `model/enterprise`、`service/enterprise`、`controller/enterprise`、`dto/enterprise`，不要新建第二套应用骨架。[Source: `_bmad-output/planning-artifacts/architecture.md#Architecture Decision Document`]
- 企业 API 统一在 `/api/enterprise/**`，属于管理后台 API，OpenAPI 只更新 `docs/openapi/api.json`。[Source: `_bmad-output/planning-artifacts/architecture.md#API & Communication Patterns`]
- 部门不是 `Group`。不得修改 `User.Group`、`Token.Group`、渠道分组、模型倍率、relay 请求 DTO 或计费表达式变量。[Source: `_bmad-output/planning-artifacts/epics.md#Additional Requirements`]
- 数据库必须同时兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+；优先 GORM，避免 raw SQL。必须用 `enterprise_` 表前缀、snake_case 列名、`int64` Unix 秒时间字段。[Source: `_bmad-output/planning-artifacts/architecture.md#Naming Patterns`]
- 所有 JSON marshal/unmarshal 只能调用 `common/json.go` wrapper；允许引用 `encoding/json` 的类型但不得直接 marshal/unmarshal。[Source: `AGENTS.md#Rule 1: JSON Package — Use common/json.go`]
- DTO 可选标量使用 `*T + omitempty`，数组响应返回 `[]`。这条在企业管理 API 中也适用。[Source: `_bmad-output/planning-artifacts/architecture.md#Data Exchange Formats`]
- V1 企业模块不引入新 Go 依赖，不升级 Gin/GORM/Go；使用仓库 `go.mod` 当前版本。Default 前端使用仓库 `package.json` 当前 React/Rsbuild/TanStack Query 版本，包管理器用 Bun。[Source: `_bmad-output/planning-artifacts/architecture.md#Important Decisions`]

### Existing Code Touchpoints

- `model/main.go`：现有迁移入口是 `migrateDB()` 和 `migrateDBFast()`；企业迁移应通过 `model/enterprise/migration.go` 被调用，不要把企业表散落进多个旧模型文件。
- `router/api-router.go`：`/api` 分组已包含 gzip、BodyStorageCleanup、GlobalAPIRateLimit；企业路由应在该分组内注册，避免绕过管理后台中间件。
- `common/gin.go`：管理后台响应包装为 `{ success, message, data }`，使用 `common.ApiSuccess` / `ApiErrorI18n`。
- `web/default/src/hooks/use-sidebar-data.ts`：Default 侧边栏 Admin 分组在这里定义；新增企业入口从这里接入。
- `web/default/src/components/layout/lib/sidebar-view-registry.ts`：若企业组织采用嵌套侧栏，需要在此注册新的 sidebar view；当前仅注册 `SYSTEM_SETTINGS_VIEW`。
- `web/default/src/routes/_authenticated/channels/index.tsx`：可参考现有 Admin 路由鉴权模式：读取 `useAuthStore`，角色不足 redirect `/403`。
- `web/default/src/features/channels/api.ts` 与 `web/default/src/features/users/api.ts`：可参考 `api` helper、类型导出和业务响应处理风格。

### Suggested Data Model

`enterprise_departments` 最小字段建议：

- `id int`：GORM 默认主键。
- `tenant_id int`：V1 固定 `0`，为后续 SaaS 预留。
- `name string`：当前展示名。
- `parent_id *int`：根节点为空；不使用 `group` 等保留字。
- `status int`：建议 `1=enabled, 2=disabled, 3=deleted`；不物理删除停用/删除部门。
- `source_type string`：如 `manual`、`dingtalk`；本故事只需展示。
- `external_id string`：外部部门稳定 ID；本地手工节点可为空。
- `sync_status int`：如 `0=none, 1=ok, 2=warning, 3=failed`。
- `sync_error string`：最长限制并避免敏感信息。
- `name_history string`：JSON 字符串，记录历史名称快照；用 `common.Marshal/Unmarshal`。
- `created_at int64`、`updated_at int64`、`deleted_at int64`：Unix 秒；不使用 `time.Time` 字段类型。

### Frontend UX Notes

- 第一屏应是实际企业组织管理体验，不做营销/介绍页。
- 部门树是工作台式管理界面：信息密度适中，清晰展示状态、来源、异常，不使用夸张 hero 或装饰卡片。
- 空状态必须可操作，但本故事不需要实现钉钉配置/同步本体；链接或按钮应与后续 Epic 2 路由计划对齐，未交付能力不得假装已可用。
- 所有可见文案必须走 `t('English source')`，并进入六个 Default locale JSON。

### Scope Boundaries

- 本故事只交付“查看企业部门树”的基础闭环：部门表、迁移、树查询 API、Default 企业组织页面、OpenAPI、i18n、测试。
- 不实现部门创建/编辑、用户部门 N:N 维护、部门角色、钉钉配置、钉钉同步、配额分配、用量聚合、风险告警。
- 不触碰 `relay/**`、`pkg/billingexpr/**`、`pkg/cachex/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`、所有 `relay/channel/**`。
- 不修改 `users`、`logs`、`quota_data` 的 schema；`tenant_id` 只在新增企业表中预留。

### Latest Technical Notes

- 本故事不需要外部最新 API 行为：架构明确 V1 不以依赖升级为前提，必须使用仓库锁定的 Go/Gin/GORM/React/Rsbuild/TanStack Query 版本。
- 如果实现时需要查框架文档，只允许按仓库版本查官方文档；不得因为“最新版最佳实践”引入新依赖或升级工具链。

### References

- Story source: `_bmad-output/planning-artifacts/epics.md#Story 1.1: 查看企业部门树`
- Epic context: `_bmad-output/planning-artifacts/epics.md#Epic 1: 企业组织与部门治理基线`
- PRD FR-1 / FR-4: `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#4.1 组织与部门模型`
- Architecture naming and structure: `_bmad-output/planning-artifacts/architecture.md#Naming Patterns`
- Architecture project structure: `_bmad-output/planning-artifacts/architecture.md#Complete Project Directory Structure`
- API and response patterns: `_bmad-output/planning-artifacts/architecture.md#Format Patterns`
- Boundaries: `_bmad-output/planning-artifacts/architecture.md#Architectural Boundaries`
- Project conventions: `AGENTS.md`

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Create-story input discovery: loaded epics, architecture, PRD shards; no standalone UX document found.
- Existing code scan: no existing enterprise module directories found; this story creates first enterprise vertical slice.
- Recent git context: latest commits are documentation/planning oriented (`docs: add enterprise test design plan`, sprint planning artifacts); no prior implementation story to reuse.
- Dev workflow activation: resolved workflow customization manually because local Python lacks `tomllib`; no team/user overrides found.
- Implementation review: existing worktree already contained Story 1.1 implementation files; continued from that state without reverting unrelated files.
- Validation rerun: `go`, `gofmt`, and `bun` are available in this worktree; `gofmt -w` completed for modified Go files.
- Validation blocker: `go test ./model/enterprise ./service/enterprise ./controller/enterprise` cannot resolve dependencies because network access to `proxy.golang.org` is unavailable; writable `GOCACHE`/`GOPATH`/`GOMODCACHE` were configured under `/private/tmp` before retrying.
- Validation blocker: `bun run typecheck` fails because `tsc` is absent; `bun install --frozen-lockfile` cannot download dependencies because package tarball fetches are refused by the restricted network.
- Validation passed: `cd web/default && bun run i18n:sync`; `_sync-report.json` shows zero missing, extra, and untranslated keys for en/zh/fr/ja/ru/vi.
- Validation rerun on 2026-05-28: `cd web/default && bun run i18n:sync` passed again; `_sync-report.json` still shows zero missing, extra, and untranslated keys for en/zh/fr/ja/ru/vi.
- Validation blocker confirmed on 2026-05-28: `cd web/default && bun run typecheck` fails with `/bin/bash: tsc: command not found`; `web/default/node_modules` contains only `.cache`, with no `.bin/tsc` or `typescript` package.
- Validation blocker confirmed on 2026-05-28: `cd web/default && bun install --frozen-lockfile --offline` cannot complete because required package tarballs, including `typescript@6.0.3`, are not present locally and network fetches are refused.
- Validation blocker confirmed on 2026-05-28: `GOPATH=/private/tmp/sa-go-path-story-1-1 GOMODCACHE=/private/tmp/sa-go-modcache-story-1-1 GOCACHE=/private/tmp/sa-go-cache-story-1-1 go test ./model/enterprise ./service/enterprise ./controller/enterprise` reaches dependency resolution but fails on DNS/network access to `goproxy.cn`.
- Validation check on 2026-05-28: `gofmt -w` completed for enterprise Go files and `git diff --check` passed.
- Validation passed on 2026-05-28: `cd web/default && bun run i18n:sync`; `_sync-report.json` shows zero missing, extra, and untranslated keys for en/zh/fr/ja/ru/vi.
- Validation initially failed on 2026-05-28: `cd web/default && bun run typecheck` reported a pre-existing generic row typing issue in `web/default/src/features/usage-logs/components/usage-logs-mobile-card.tsx`; fixed by reading row metadata through `Record<string, unknown>`.
- Validation passed on 2026-05-28: `cd web/default && bun run typecheck`.
- Validation passed on 2026-05-28: `GOCACHE=/private/tmp/sa-go-cache-story-1-1 GOMODCACHE=/Users/hq-it/go/pkg/mod GOPATH=/Users/hq-it/go go test ./model/enterprise ./service/enterprise ./controller/enterprise`.
- Validation blocker remains on 2026-05-28: `GOCACHE=/private/tmp/sa-go-cache-story-1-1 GOMODCACHE=/private/tmp/sa-go-modcache-story-1-1 GOPATH=/private/tmp/sa-go-path-story-1-1 go test ./model ./router ./controller/... ./service/...` cannot download uncached dependencies because sandboxed network/DNS access to `goproxy.cn` is unavailable.
- Applied `/tmp/epic1-regression-gate.patch` on 2026-05-28, covering baseline fixes in `controller/model.go`, `relay/channel/claude/relay-claude.go`, and `relay/helper/stream_scanner.go`.
- Validation passed on 2026-05-28: `gofmt -w` completed for Story 1.1 Go files and regression gate patch files.
- Validation passed on 2026-05-28: `GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn GOCACHE=/private/tmp/sa-go-cache-story-1-1-r6 GOMODCACHE=/Users/hq-it/go/pkg/mod GOPATH=/Users/hq-it/go go test ./model/enterprise ./service/enterprise ./controller/enterprise`.
- Validation passed on 2026-05-28: `GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn GOCACHE=/private/tmp/sa-go-cache-story-1-1-r6 GOMODCACHE=/Users/hq-it/go/pkg/mod GOPATH=/Users/hq-it/go go test ./model ./router ./controller/... ./service/...`.
- Validation passed on 2026-05-28: `cd web/default && bun run typecheck`.
- Validation passed on 2026-05-28: `cd web/default && bun run i18n:sync`; `_sync-report.json` shows zero missing, extra, and untranslated keys for en/zh/fr/ja/ru/vi.
- Validation passed on 2026-05-28: `git diff --check`.
- Review auto-fix on 2026-05-28: added defensive department-tree cycle handling, SQLite self-parent migration constraint coverage, and horizontal overflow handling for the Default department tree.
- Review validation passed on 2026-05-28: `GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn GOCACHE=/private/tmp/sa-go-cache-review-1-1 GOMODCACHE=/Users/hq-it/go/pkg/mod GOPATH=/Users/hq-it/go go test ./model/enterprise ./service/enterprise ./controller/enterprise ./tests/api`.
- Review validation passed on 2026-05-28: `GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn GOCACHE=/private/tmp/sa-go-cache-review-1-1 GOMODCACHE=/Users/hq-it/go/pkg/mod GOPATH=/Users/hq-it/go go test ./model ./router ./controller/... ./service/...`.
- Review validation passed on 2026-05-28: `/Users/hq-it/.bun/bin/bun run typecheck`.
- Review validation passed on 2026-05-28: `/Users/hq-it/.bun/bin/bun run i18n:sync`.
- Review validation passed on 2026-05-28: `/Users/hq-it/.bun/bin/bun run test:e2e`; build emitted a non-blocking optional-dependency warning for `debug` -> `supports-color`.
- Review validation passed on 2026-05-28: `git diff --check`.
- Second review scope audit on 2026-05-28: detected forbidden `relay/**` branch changes from the earlier regression gate patch and removed them from the worktree.
- Second review validation passed on 2026-05-28: `git diff --name-only | rg '^relay/' || true` and `git status --porcelain | rg '(^.. relay/|^\?\? relay/)' || true` returned no relay changes.
- Second review validation passed on 2026-05-28: `GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn GOCACHE=/private/tmp/sa-go-cache-review-1-1-scope2 GOMODCACHE=/Users/hq-it/go/pkg/mod GOPATH=/Users/hq-it/go go test ./model/enterprise ./service/enterprise ./controller/enterprise ./tests/api`.
- Second review validation passed on 2026-05-28: `GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn GOCACHE=/private/tmp/sa-go-cache-review-1-1-scope2 GOMODCACHE=/Users/hq-it/go/pkg/mod GOPATH=/Users/hq-it/go go test ./model ./router ./controller/... ./service/...`.
- Second review validation passed on 2026-05-28: `cd web/default && /Users/hq-it/.bun/bin/bun run typecheck`.
- Second review validation passed on 2026-05-28: `cd web/default && /Users/hq-it/.bun/bin/bun run i18n:sync`.
- Second review validation passed on 2026-05-28: `cd web/default && /Users/hq-it/.bun/bin/bun run test:e2e`; build emitted a non-blocking optional-dependency warning for `debug` -> `supports-color`.
- Second review validation passed on 2026-05-28: `git diff --check`.

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- Checklist validation applied in YOLO mode: added concrete file locations, existing-code touchpoints, API shape, schema fields, i18n requirements, testing requirements, and explicit no-touch boundaries.
- Implemented enterprise department constants/model/migration/service/controller/router/OpenAPI/Default frontend/i18n/tests for the Story 1.1 vertical slice.
- Adjusted `source_type` to use an `int` enum end-to-end to match the task contract.
- Confirmed `npm run i18n:sync` succeeds and all Default locale report counts are zero after adding `DingTalk` to the i18n brand/literal allowlist.
- Marked implementation and test-authoring tasks complete after code inspection, formatting, i18n sync, locale report validation, and static compatibility checks.
- Story remains in-progress because required Go test execution and Default frontend typecheck are blocked by unavailable dependency downloads in the restricted network environment.
- Rechecked implementation against Story 1.1 acceptance criteria and checklist; no additional code changes were needed, but validation checkboxes remain incomplete because required dependency-backed commands cannot run to completion in this environment.
- Completed frontend validation after dependencies became available locally; fixed the existing usage logs mobile card type issue so `bun run typecheck` passes.
- Enterprise package Go tests now pass; story remains in-progress only because the broader affected-package Go regression command is blocked by uncached dependencies and restricted network access.
- Applied the regression gate patch requested for Story 1.1 completion, then reran all required backend, frontend, i18n, formatting, and diff hygiene gates successfully.
- Story 1.1 is complete and ready for review.
- Senior developer review completed in auto-fix mode; all confirmed HIGH/MEDIUM review issues were fixed and no CRITICAL issues remain.
- Second senior developer scope-boundary review removed forbidden `relay/**` changes from the branch and File List; retained Story 1.1 scoped changes plus the non-forbidden `controller/model.go` regression gate fix needed for validation.

### Senior Developer Review (AI)

Reviewer: hth on 2026-05-28

Outcome: Approved after automatic fixes. Story 1.1 acceptance criteria are implemented: API returns `data: []` for empty state, department tree includes 3+ levels and inactive/failed nodes, enterprise migration uses the `enterprise_` table prefix and GORM-based cross-database patterns, and Default frontend exposes the admin-only enterprise organization page.

Findings fixed:

- [MEDIUM] Cyclic or self-referential `parent_id` data could make tree cloning recurse forever. Fixed by tracking traversal path/visited nodes and adding cycle/self-parent tests in `service/enterprise/department_test.go`.
- [MEDIUM] The department table did not prevent a department from referencing itself. Fixed with `chk_departments_parent_not_self` and SQLite migration test coverage in `model/enterprise/department_test.go`.
- [MEDIUM] The Default department tree used `overflow-hidden`, which clipped required columns on narrow viewports. Fixed by switching the tree container to horizontal scrolling.
- [MEDIUM] Story File List omitted actual source/test files introduced by the implementation (`tests/api/enterprise_departments_tree_test.go`, frontend e2e config/test, package script, `.gitignore`). Fixed below.

Validation checklist:

- Story file loaded and status verified as reviewable before review.
- Acceptance Criteria cross-checked against implementation.
- File List cross-checked against git status, including untracked implementation files.
- Backend, frontend, i18n, e2e, and diff hygiene validations rerun after fixes.
- Sprint status synced to `done`.

### Senior Developer Review (AI) - Scope Boundary Cycle 2

Reviewer: hth on 2026-05-28

Outcome: Approved after automatic scope cleanup. Story 1.1 remains within the allowed enterprise department tree scope, with no remaining `relay/**` changes in `git diff` or `git status`.

Findings fixed:

- [HIGH] Forbidden `relay/**` changes were present in `relay/channel/claude/relay-claude.go` and `relay/helper/stream_scanner.go`, violating the Story 1.1 scope boundary. Removed those branch changes.
- [MEDIUM] Story File List still included forbidden relay files after implementation review. Removed them from the File List below.
- [MEDIUM] Dev Agent Record did not explicitly document the second scope-boundary audit. Added this review section and debug/completion notes.

Validation checklist:

- Story scope boundaries re-audited against git changes.
- `relay/**` confirmed clean after auto-fix.
- Backend, frontend, i18n, e2e, and diff hygiene validations rerun after scope cleanup.
- Sprint status remains `done` because no CRITICAL issues remain after cleanup.

### File List

- `_bmad-output/implementation-artifacts/1-1-view-enterprise-department-tree.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `constant/enterprise.go`
- `controller/model.go`
- `controller/enterprise/department.go`
- `docs/openapi/api.json`
- `dto/enterprise/department.go`
- `i18n/keys.go`
- `i18n/locales/en.yaml`
- `i18n/locales/zh-CN.yaml`
- `i18n/locales/zh-TW.yaml`
- `model/enterprise/department.go`
- `model/enterprise/department_test.go`
- `model/enterprise/migration.go`
- `model/main.go`
- `router/api-router.go`
- `router/enterprise-router.go`
- `service/enterprise/department.go`
- `service/enterprise/department_test.go`
- `service/enterprise/errors.go`
- `tests/api/enterprise_departments_tree_test.go`
- `web/default/.gitignore`
- `web/default/package.json`
- `web/default/rsbuild.test.config.ts`
- `web/default/scripts/sync-i18n.mjs`
- `web/default/src/features/enterprise-organization/api.ts`
- `web/default/src/features/enterprise-organization/components/DepartmentTree.tsx`
- `web/default/src/features/enterprise-organization/constants.ts`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/features/enterprise-organization/hooks/use-department-tree.ts`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/types.ts`
- `web/default/src/features/usage-logs/components/usage-logs-mobile-card.tsx`
- `web/default/src/hooks/use-sidebar-data.ts`
- `web/default/src/i18n/locales/_reports/_sync-report.json`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/routeTree.gen.ts`
- `web/default/src/routes/_authenticated/enterprise/organization.tsx`

### Change Log

- 2026-05-28: Added Story 1.1 enterprise department tree vertical slice and tests.
- 2026-05-28: Completed frontend typecheck/i18n validation and enterprise Go package tests; story remains in-progress pending broader affected-package Go regression, which is blocked by restricted network access to uncached dependencies.
- 2026-05-28: Applied regression gate patch, completed all required Story 1.1 validation gates, and marked story ready for review.
- 2026-05-28: Senior developer review auto-fixed cycle/self-parent safeguards, mobile tree overflow, File List gaps, reran validation, and marked story done.
- 2026-05-28: Second review cycle removed forbidden `relay/**` branch changes, updated review notes/File List, reran validation, and kept sprint status done because no CRITICAL issues remain.
