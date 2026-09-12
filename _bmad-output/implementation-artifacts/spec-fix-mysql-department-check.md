---
title: 'Fix MySQL Department CHECK Constraint'
type: 'bugfix'
created: '2026-05-29'
status: 'done'
baseline_commit: 'efaddbe7f4fea77e66b6a12f816e7b9f65bcaa0c'
context:
  - '{project-root}/_bmad-output/implementation-artifacts/investigations/mysql-check-auto-increment-investigation.md'
---

<frozen-after-approval reason="human-owned intent - do not modify unless human renegotiates">

## Intent

**Problem:** MySQL deployments fail during startup migration because `enterprise_departments` declares a CHECK constraint that references the auto-increment `id` column, producing Error 3818.

**Approach:** Remove the MySQL-incompatible database CHECK from the department model and keep the migration portable across SQLite, MySQL, and PostgreSQL. Preserve existing tree rendering behavior that tolerates cycles/self-parent records without infinite recursion.

## Boundaries & Constraints

**Always:** Keep database migrations compatible with SQLite, MySQL, and PostgreSQL. Keep the enterprise department table, columns, and indexes intact. Use the existing GORM model/migration patterns.

**Ask First:** Any change that adds raw SQL migrations, changes department IDs/parent semantics, or introduces provider-specific schema branching.

**Never:** Do not rename protected project/organization identifiers. Do not alter unrelated enterprise tables or DingTalk sync behavior.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| MySQL upgrade creates department table | MySQL database without `enterprise_departments` | GORM-generated DDL no longer includes `chk_departments_parent_not_self`; startup migration can create the table | Migration errors should reflect other real DB issues, not Error 3818 |
| SQLite migration test | In-memory SQLite enterprise migration | Department columns and indexes still exist; no assertion requires the removed CHECK | Test fails if required table shape/indexes regress |
| Self-parent data read path | A department row has `parent_id = id` | Tree builder still returns without recursion loops | Existing cycle guard handles the malformed row |

</frozen-after-approval>

## Code Map

- `model/enterprise/department.go` -- Department schema source that currently emits the failing CHECK constraint.
- `model/enterprise/department_test.go` -- SQLite migration regression test that currently expects the CHECK.
- `service/enterprise/department.go` -- Existing tree-building guard for cyclic/self-parent data.
- `service/enterprise/department_test.go` -- Existing tests proving cyclic/self-parent data does not recurse forever.

## Tasks & Acceptance

**Execution:**
- [x] `model/enterprise/department.go` -- remove `check:chk_departments_parent_not_self,parent_id IS NULL OR parent_id <> id` from `ParentId` -- prevents MySQL Error 3818.
- [x] `model/enterprise/department_test.go` -- remove the expectation that the CHECK constraint exists while keeping table/index assertions -- aligns tests with portable schema.
- [x] `model/enterprise/department_test.go` or existing service tests -- ensure self-parent read behavior remains covered -- preserves malformed-data tolerance after DB CHECK removal.

**Acceptance Criteria:**
- Given a MySQL database needs to create `enterprise_departments`, when enterprise migration runs, then generated schema must not include a CHECK constraint referencing auto-increment `id`.
- Given SQLite migration tests run, when `Migrate(db)` creates enterprise tables, then department table columns and indexes still pass.
- Given self-parent department data is passed to the tree builder, when the tree is built, then it returns without infinite recursion.

## Spec Change Log

## Verification

**Commands:**
- `go test ./model/enterprise ./service/enterprise` -- expected: enterprise model migration and tree behavior tests pass.
- `rg -n "chk_departments_parent_not_self|check:.*parent_id.*id" model/enterprise` -- expected: no remaining model tag or test assertion for the removed CHECK.

## Suggested Review Order

**Schema Compatibility**

- Remove failing CHECK source
  [`department.go:20`](../../model/enterprise/department.go#L20)

**Regression Coverage**

- Keep migration shape checks
  [`department_test.go:29`](../../model/enterprise/department_test.go#L29)

- Guard against CHECK regression
  [`department_test.go:60`](../../model/enterprise/department_test.go#L60)
