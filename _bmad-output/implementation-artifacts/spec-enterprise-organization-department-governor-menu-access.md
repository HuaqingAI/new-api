---
title: 'Enterprise Organization Department Governor Menu Access'
type: 'bugfix'
created: '2026-06-04'
status: 'done'
route: 'one-shot'
context: []
---

# Enterprise Organization Department Governor Menu Access

## Intent

**Problem:** Department owners and department admins could still be allowed by the enterprise organization page and backend department-governance APIs, but the sidebar placed the page inside an admin-only group so they had no practical menu entry.

**Approach:** Expose a backend `enterprise_organization` user permission based on the existing enterprise permission service, then let the frontend sidebar surface only the Enterprise Organization item for department governors while keeping all other admin navigation and budget lifecycle controls administrator-only.

## Suggested Review Order

1. `../../controller/user.go` -- backend permission source for the new enterprise organization menu capability.
2. `../../web/default/src/hooks/use-sidebar-view.ts` -- role-level sidebar filtering that allows only the enterprise organization item for non-admin department governors.
3. `../../web/default/src/hooks/use-sidebar-config.ts` and `../../web/default/src/features/system-settings/maintenance/config.ts` -- sidebar module mapping/defaults for enterprise organization and usage modules.
4. `../../controller/user_self_test.go` and `../../web/default/src/hooks/use-sidebar-view.test.ts` -- regression coverage for permission exposure and menu filtering.
5. `../../web/default/src/features/enterprise-organization/index.tsx` -- unchanged budget lifecycle gating remains tied to `ROLE.ADMIN`.
