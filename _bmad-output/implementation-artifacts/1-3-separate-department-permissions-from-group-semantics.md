---
baseline_commit: 0978fc6e749f0a7cbe6dc0a7f4f4e0935a8df138
---

# Story 1.3: 保持部门权限与 Group 语义分离

Status: done

## Story

As a 系统管理员,
I want 部门权限和现有 `Group` 计费体系互不干扰,
so that 企业组织能力不会破坏现有模型、渠道和倍率行为。

## Acceptance Criteria

1. Given 用户属于一个或多个部门, When 用户部门关系发生变化, Then `User.Group`、`Token.Group`、渠道分组和模型倍率行为保持不变, And relay 请求 DTO、计费表达式和模型定价变量不增加部门字段。
2. Given 部门管理员权限被配置, When 部门管理员访问企业组织接口, Then 后端按企业权限服务校验可管理部门范围, And 前端导航/按钮可见性只作为辅助，不作为权限来源。

## Tasks / Subtasks

- [x] 新增部门权限模型与迁移 (AC: 2)
  - [x] 新增 `model/enterprise/department_role.go`，表名 `enterprise_department_roles`，只表达部门管理权限，不扩展 `users.role`。
  - [x] 在 `constant/enterprise.go` 定义部门角色和状态枚举。
  - [x] 在 `model/enterprise/migration.go` 注册迁移，并保持 SQLite/MySQL/PostgreSQL 兼容。
- [x] 实现企业权限服务与中间件 (AC: 2)
  - [x] 新增 `service/enterprise/permission.go`，根据部门角色计算可管理部门范围。
  - [x] 新增 `middleware/enterprise_dept_admin.go`，对部门资源路由做后端权限校验。
  - [x] 后端拒绝部门管理员权限不足时返回 `error.enterprise.permission.dept_admin_required`。
- [x] 接入企业组织 API (AC: 2)
  - [x] `GET /api/enterprise/departments/tree` 对系统 Admin/Root 返回全量，对部门管理员返回可管理范围。
  - [x] 部门成员列表、添加、停用、恢复接口按 `:id` 部门范围校验。
  - [x] 用户维度部门维护仍要求系统 Admin/Root，不用前端可见性替代后端鉴权。
- [x] 保持 Group/relay/billing 边界 (AC: 1)
  - [x] 成员关系变更不修改 `users.group`、`tokens.group`、渠道分组、模型倍率。
  - [x] 不向 relay 请求 DTO、计费表达式、模型定价变量增加部门字段。
  - [x] 保留禁止触碰 `relay/**`、`pkg/billingexpr/**`、`router/relay-router.go`、`docs/openapi/relay.json` 的守护测试。
- [x] 前端辅助可见性调整 (AC: 2)
  - [x] Default 企业组织路由不把系统 Admin 角色当作唯一权限来源，真实拒绝由后端返回。
  - [x] 侧栏/按钮可见性只作为辅助体验，不作为访问控制。
- [x] 测试与验证 (AC: 1, 2)
  - [x] 覆盖部门权限范围、子部门继承、未授权拒绝、系统 Admin 全局通过。
  - [x] 覆盖部门成员接口的后端权限中间件。
  - [x] 覆盖 `User.Group` / `Token.Group` / 渠道分组 / 模型倍率在部门关系变化后不变。
  - [x] 运行 Go 测试、Default i18n sync、Default typecheck 和 diff hygiene。

## Dev Notes

- Story 1.3 依赖 Story 1.1 的 `enterprise_departments` 和 Story 1.2 的 `enterprise_user_departments`。
- 部门管理员权限放入新增企业表，不扩展 `users.role` 枚举。
- `Group` 继续代表模型、渠道和倍率语义，Department 只代表组织治理范围。
- 企业权限校验必须在后端 service/middleware 中完成；前端导航和按钮可见性只能帮助用户发现入口。
- 本 story 严禁触碰 relay、billing expression、log、relay OpenAPI 等核心路径。

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Continued from Story 1.3 branch `codex/story-1-3`, based on Story 1.1 plus Story 1.2 commits.
- Added `enterprise_department_roles`, enterprise permission service, and department-admin middleware without extending `users.role`.
- Validation passed: `go test ./model/enterprise ./service/enterprise ./controller/enterprise ./middleware ./tests/api`.
- Validation passed: `GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn GOCACHE=/private/tmp/sa-go-cache-story-1-3 GOMODCACHE=/Users/hq-it/go/pkg/mod GOPATH=/Users/hq-it/go go test ./model ./router ./controller/... ./service/... ./middleware/... ./tests/api`.
- Validation passed: `cd web/default && /Users/hq-it/.bun/bin/bun run i18n:sync`.
- Validation passed after restoring dependencies with `/Users/hq-it/.bun/bin/bun install --frozen-lockfile`: `cd web/default && /Users/hq-it/.bun/bin/bun run typecheck`.
- Validation passed: `git diff --check`.
- Boundary scan passed: no changed paths under `relay/**`, `pkg/billingexpr/**`, `pkg/cachex/**`, `model/log.go`, `controller/log.go`, `router/relay-router.go`, or `docs/openapi/relay.json`.

### Completion Notes List

- Implemented department-admin roles and backend permission checks for department tree and department-member routes.
- Kept user-level department maintenance behind system Admin/Root authorization.
- Added tests proving department membership changes leave `User.Group`, `Token.Group`, channel groups, group ratio, and model ratio unchanged.
- Kept relay DTOs, billing expression code, relay router, relay OpenAPI, and log core files untouched.
