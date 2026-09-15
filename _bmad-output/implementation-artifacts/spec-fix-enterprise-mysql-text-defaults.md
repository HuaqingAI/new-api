---
title: 'Fix Enterprise MySQL Text Defaults'
type: 'bugfix'
created: '2026-05-29'
status: 'done'
route: 'one-shot'
---

## Intent

**Problem:** MySQL rejects enterprise table creation during upgrades because several `TEXT` columns declare database defaults, producing `Error 1101 (42000)`.

**Approach:** Remove database-level defaults from enterprise `TEXT` fields and keep semantic defaults in Go creation paths/tests.

## Suggested Review Order

**Schema Compatibility**

- Remove the failing `TEXT DEFAULT '[]'` DDL source.
  [`department.go:26`](../../model/enterprise/department.go#L26)

- Remove the audit payload `TEXT DEFAULT '{}'` DDL source.
  [`admin_action.go:17`](../../model/enterprise/admin_action.go#L17)

- Preserve empty audit payload semantics in application code.
  [`admin_action.go:25`](../../model/enterprise/admin_action.go#L25)

- Remove the DingTalk sync scope `TEXT DEFAULT ''` DDL source.
  [`dingtalk_config.go:10`](../../model/enterprise/dingtalk_config.go#L10)

**Regression Coverage**

- Guard the whole enterprise migration batch against future `TEXT` defaults.
  [`text_default_test.go:14`](../../model/enterprise/text_default_test.go#L14)

- Verify application defaults survive after DB defaults are removed.
  [`text_default_test.go:39`](../../model/enterprise/text_default_test.go#L39)

- Keep focused audit payload default coverage near admin action migration tests.
  [`admin_action_test.go:35`](../../model/enterprise/admin_action_test.go#L35)
