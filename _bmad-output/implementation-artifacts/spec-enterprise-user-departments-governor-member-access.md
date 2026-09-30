---
title: 'Enterprise User Departments Governor Member Access'
type: 'bugfix'
created: '2026-06-05'
status: 'done'
baseline_commit: 'e6d1c2acec9126692b9a60f1e7d3163734072239'
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** 非管理员部门负责人给部门其它成员分配额度时，前端切换到其它员工作为工作区会调用 `GET /api/enterprise/users/:id/departments`，但当前 controller 把非 Admin 查询他人部门一律拒绝，导致合法治理场景报无权限。

**Approach:** 保留普通员工“只能查自己”的默认边界，同时允许部门负责人读取其可治理部门范围内成员的部门信息。授权判断必须基于现有企业权限模型和成员关系，不开放任意查询其它用户部门。

## Boundaries & Constraints

**Always:** 使用现有 `PermissionService` / membership 数据判断企业治理权限；Root/Admin 仍全局放行；非管理员查他人时必须证明目标用户至少有一个 active membership 落在操作者可治理部门范围内；错误响应保持现有 i18n key，不新增前端文案；数据库查询保持 SQLite/MySQL/PostgreSQL 兼容并优先使用 GORM。

**Ask First:** 如果发现页面实际需要查询“范围外成员”或“无 active membership 的用户”的部门信息，先停下来确认，因为这会扩大当前权限模型。

**Never:** 不把 `GET /api/enterprise/users/:id/departments` 改成所有登录用户可查询任意 user；不降低 quota allocation、budget delegation、department members 等写操作权限；不修改受保护项目标识；不引入第二套部门负责人权限表。

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| 负责人查可治理成员 | actor=部门 1 负责人，target user=部门 1 active member，请求 `/api/enterprise/users/{target}/departments` | 返回 target 的部门列表，供额度分配工作区选择使用 | N/A |
| 普通员工查自己 | actor=普通员工 100，请求 `/api/enterprise/users/100/departments` | 继续返回自己的部门列表 | N/A |
| 普通员工查别人 | actor=普通员工 100，target user=101 且 actor 无治理权限 | 继续返回无权限 | `auth.insufficient_privilege` |
| 负责人查范围外成员 | actor=部门 1 负责人，target user 只属于部门 2，请求 target departments | 返回无权限，不能泄露范围外部门关系 | `auth.insufficient_privilege` |

</frozen-after-approval>

## Code Map

- `controller/enterprise/department_membership.go` -- `ListUserDepartments` 当前有非 Admin 查他人的硬编码拒绝，是本 bug 的直接触发点。
- `service/enterprise/department_membership.go` -- `ListUserDepartments` 返回目标用户 membership；可新增或复用 membership-scope 查询来证明目标用户在操作者可治理范围内。
- `service/enterprise/permission.go` -- `ListManageableDepartmentIds` / `CanGovernDepartment` 已表达部门负责人、继承和本地 deny 规则，应作为授权依据。
- `controller/enterprise/department_membership_test.go` -- 已覆盖“普通用户只能查自己”，需要扩展为“负责人可查可治理成员但不能查范围外成员”。
- `router/enterprise-router.go` -- GET 路由已只挂 `UserAuth`，通常无需改路由；权限应留在 controller/service 组合判断。
- `web/default/src/features/enterprise-organization/api.ts` -- 前端调用 `/api/enterprise/users/:id/departments` 的 API helper，当前无需改动。

## Tasks & Acceptance

**Execution:**
- [x] `controller/enterprise/department_membership.go` -- 将非 Admin 查他人的判断改为调用企业治理授权 helper，只有目标用户属于操作者可治理部门范围时放行。
- [x] `service/enterprise/department_membership.go` 或 `service/enterprise/permission.go` -- 增加小而明确的 helper，用 active membership 和 `PermissionService.ListManageableDepartmentIds` 判断 actor 是否可读取 target user 的部门列表。
- [x] `controller/enterprise/department_membership_test.go` -- 扩展权限回归测试，覆盖负责人可查可治理成员、负责人不可查范围外成员、普通员工仍不可查别人。

**Acceptance Criteria:**
- Given 非管理员部门负责人管理部门 1 且用户 101 是部门 1 active member, when 请求 `GET /api/enterprise/users/101/departments`, then API 成功返回用户 101 的部门列表。
- Given 普通员工 100 不是管理员也不是目标用户可治理范围的负责人, when 请求 `GET /api/enterprise/users/101/departments`, then API 返回 `auth.insufficient_privilege`。
- Given 部门 1 负责人请求只属于部门 2 的用户部门列表, when 调用同一接口, then API 返回 `auth.insufficient_privilege`。
- Given Admin 或用户本人调用同一接口, when 查询部门列表, then 原有成功行为不变。

## Spec Change Log

- Review patch: adversarial/edge review found non-admin governor reads needed explicit tenant scoping, active-only status alignment, and safer large `IN` handling. Amended implementation to set `query.TenantId`, force active status for governor-on-other-user reads, add cross-tenant/status regression assertions, batch readable department membership lookups, and remove an unused bool helper. KEEP: result filtering by readable department ids remains in place so multi-department targets do not leak outside-scope departments.

## Design Notes

授权判断应证明“actor 能治理 target 的至少一个 active department membership”，而不是只证明 actor 是某个部门负责人。一个安全的实现形态是：先取 actor 的 manageable department ids，再在 `enterprise_user_departments` 中查 target user 是否存在 active membership 且 `department_id IN manageableIds`。这样继承、deny、DingTalk/manual source 优先级仍由 `PermissionService` 统一处理。

## Verification

**Commands:**
- `go test ./controller/enterprise -run 'TestEnterpriseMembershipAPI'` -- expected: controller enterprise membership 权限回归全部通过。

## Suggested Review Order

**Read Authorization**

- Entry point: non-admin reads now flow through scoped authorization.
  [`department_membership.go:27`](../../controller/enterprise/department_membership.go#L27)

- Governor reads are tenant-scoped, active-only, and department-filtered.
  [`department_membership.go:44`](../../controller/enterprise/department_membership.go#L44)

**Scope Resolution**

- Batch readable departments from the actor's manageable scope.
  [`permission.go:369`](../../service/enterprise/permission.go#L369)

- Membership queries can now apply server-side department filters.
  [`department_membership.go:18`](../../service/enterprise/department_membership.go#L18)

- The filter composes with tenant, status, and source constraints.
  [`department_membership.go:510`](../../service/enterprise/department_membership.go#L510)

**Regression Tests**

- Existing self-only behavior remains covered for ordinary users.
  [`department_membership_test.go:181`](../../controller/enterprise/department_membership_test.go#L181)

- Governor access covers scoped success, cross-tenant filtering, inactive denial, and outside-scope denial.
  [`department_membership_test.go:214`](../../controller/enterprise/department_membership_test.go#L214)
