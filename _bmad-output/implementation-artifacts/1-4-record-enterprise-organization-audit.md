---
baseline_commit: a902d9dc9e15a307278f8049dbd383bf2fa60583
---

# Story 1.4: 记录企业组织管理审计

Status: done

## Story

As a Root 或管理员,
I want 查看企业组织相关管理动作,
so that 部门、成员和权限变更具备可追溯记录。

## Acceptance Criteria

1. Given 管理员执行部门、成员关系或部门权限变更, When 操作成功保存, Then 系统写入 `enterprise_admin_actions` 审计记录, And 记录包含 actor、动作类型、对象、时间、必要字段差异，不写入敏感凭据。
2. Given 管理员查看企业组织操作记录, When 按时间、动作类型或对象筛选, Then 系统返回分页列表, And 每条记录至少包含 action_id、actor_id、action_type、object_type、object_id、created_at 和 diff_summary。
3. Given 普通用户或无企业管理权限用户访问企业组织操作记录, When 请求 `enterprise_admin_actions` 列表或详情, Then 系统拒绝访问并返回 `error.enterprise.permission.admin_required`, And 不暴露管理员专用审计字段。

## Tasks / Subtasks

- [x] 新增企业管理动作审计模型与迁移 (AC: 1, 2)
  - [x] 新增 `model/enterprise/admin_action.go`，表名 `enterprise_admin_actions`。
  - [x] 字段包含 action_id、tenant_id、actor_id、action_type、object_type、object_id、created_at、diff_summary 和 payload。
  - [x] 在 `model/enterprise/migration.go` 注册迁移，保持三库兼容。
- [x] 实现审计 service 与 DTO (AC: 1, 2)
  - [x] 新增 `service/enterprise/admin_action.go`，提供写入、列表筛选、详情查询。
  - [x] 新增 `dto/enterprise/admin_action.go`，请求/响应字段 snake_case，可选标量用指针。
  - [x] 对 payload/diff 做敏感字段净化，避免密钥、token、secret 等写入审计。
- [x] 接入管理动作写入 (AC: 1)
  - [x] 用户部门替换、部门成员新增、停用、恢复成功后写审计。
  - [x] 新增最小部门管理员授权/撤销 API，并在成功后写权限变更审计。
  - [x] 配额分配仍不写 `enterprise_admin_actions`，遵守后续 Epic 的单源审计规则。
- [x] 暴露审计查询 API 与权限 (AC: 2, 3)
  - [x] 新增 `GET /api/enterprise/admin-actions` 分页列表。
  - [x] 新增 `GET /api/enterprise/admin-actions/:id` 详情。
  - [x] 普通用户或无系统管理权限用户返回 `error.enterprise.permission.admin_required`。
- [x] 测试与验证 (AC: 1, 2, 3)
  - [x] 覆盖审计迁移、写入、筛选、详情、敏感字段净化。
  - [x] 覆盖成员关系和部门权限 mutation 成功后写入审计。
  - [x] 覆盖普通用户访问审计 API 被拒绝。
  - [x] 运行 Go 测试、Default i18n sync/typecheck 和 diff hygiene。

## Dev Notes

- `enterprise_quota_allocations` 是后续配额分配的单一审计源，不应双写 `enterprise_admin_actions`。
- Story 1.4 不复用或修改现有 `model/log.go`、`controller/log.go` 或 usage logs 前端模块。
- 审计 payload 仅用于必要追溯，必须净化敏感字段。

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Started from Story 1.4 branch `codex/story-1-4`, based on Story 1.1 + 1.2 + 1.3 commits.
- Validation passed: `go test ./model/enterprise ./service/enterprise ./controller/enterprise ./middleware ./tests/api`.
- Validation passed: `GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn GOCACHE=/private/tmp/sa-go-cache-story-1-4 GOMODCACHE=/Users/hq-it/go/pkg/mod GOPATH=/Users/hq-it/go go test ./model ./router ./controller/... ./service/... ./middleware/... ./tests/api`.
- Validation passed: `cd web/default && /Users/hq-it/.bun/bin/bun run i18n:sync`.
- Validation passed after restoring dependencies with `/Users/hq-it/.bun/bin/bun install --frozen-lockfile`: `cd web/default && /Users/hq-it/.bun/bin/bun run typecheck`.
- Validation passed: `git diff --check`.
- Boundary scan passed: no changed paths under `relay/**`, `pkg/billingexpr/**`, `pkg/cachex/**`, `model/log.go`, `controller/log.go`, `router/relay-router.go`, `docs/openapi/relay.json`, or `web/default/src/features/usage-logs/**`.

### Completion Notes List

- Implemented `enterprise_admin_actions` model, migration, service, DTO, list API, and detail API.
- Added backend-only enterprise admin authorization for audit APIs with `error.enterprise.permission.admin_required`.
- Wrote audit records for membership replacement/add/disable/restore and department-admin grant/revoke.
- Added sensitive payload/diff redaction tests for secret/token/webhook-like fields.
