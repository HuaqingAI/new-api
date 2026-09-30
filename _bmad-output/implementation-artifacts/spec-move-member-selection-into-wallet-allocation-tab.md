---
title: 'Move member selection into the wallet allocation tab'
type: 'bugfix'
created: '2026-08-31'
status: 'done'
baseline_commit: '9def92014d04cddd53828bce95f901ed4fccb33d'
context:
  - '{project-root}/AGENTS.md'
  - '{project-root}/web/AGENTS.md'
---

<frozen-after-approval reason="human-owned intent - do not modify unless human renegotiates">

## Intent

**Problem:** On the enterprise organization allocation page, the current-department member selector is rendered above the allocation sub-tabs. Selecting a member is required only for the current member wallet allocation flow, so its separate placement makes the subordinate-department budget flow carry unrelated UI.

**Approach:** Move the existing member-context selector into the `Current Member Wallet Allocation` sub-tab. Keep the selected-member state, department scoping, allocation payload, and reset behavior unchanged; the selector must not appear independently above either allocation sub-tab.

## Boundaries & Constraints

**Always:** Preserve department-scoped member loading and selection; retain the selected member as the allocation form target; keep the subordinate-department allocation tab free of member-selection UI; maintain i18n for all user-visible text; avoid nested cards in the new layout.

**Ask First:** Adding a new member-selection interaction or changing member-management permissions.

**Never:** Change allocation APIs, quota accounting, department membership semantics, URL search-state semantics, or unrelated organization pages.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|---------------|----------------------------|----------------|
| Wallet allocation | A department is selected and the member-wallet tab is open | The current-department member selector is shown inside that tab and changes the allocation target | No selected member leaves allocation disabled with the existing contextual explanation |
| Subordinate allocation | A department is selected and the subordinate-department tab is open | No member selector or member-management UI is rendered in this flow | Existing budget-pool validation remains unchanged |
| Department change | The selected department changes | The selector lists the new department's members and invalid selected-member state is cleared | Existing allocation draft reset continues to prevent stale targets |

</frozen-after-approval>

## Code Map

- `web/src/features/enterprise-organization/index.tsx` -- task-page composition, department member context selector, and allocation sub-tabs.
- `web/src/features/enterprise-organization/enterprise-organization.test.tsx` -- server-rendered enterprise organization layout regression tests.

## Tasks & Acceptance

**Execution:**
- [x] `web/src/features/enterprise-organization/index.tsx` -- pass the existing current-department member-selection bindings into the allocation panel, remove its standalone render from the allocation task page, and render it as a peer section within the member-wallet tab -- scopes required input to the workflow that consumes it while preserving the existing state source.
- [x] `web/src/features/enterprise-organization/index.tsx` -- restructure the member-wallet tab so its selector and wallet form are peer surfaces rather than cards nested inside a card -- follows the frontend layout convention without changing member management behavior.
- [x] `web/src/features/enterprise-organization/enterprise-organization.test.tsx` -- add/update a regression assertion that proves the member context is emitted inside the wallet allocation tab and absent from the subordinate-department tab -- protects the visible placement contract.

**Acceptance Criteria:**
- Given an administrator opens the enterprise organization allocation page for a department, when viewing the allocation sub-tabs, then the member selector is not rendered above the tab control.
- Given the administrator opens `Current Member Wallet Allocation`, when the tab content renders, then the department member selector is available before the wallet allocation form and selecting a member continues to populate the allocation target.
- Given the administrator opens `Allocate Budget To Subordinate Department`, when the tab content renders, then it contains no current-member selector or member-management surface.
- Given the selected department or member changes, when the existing state synchronization runs, then stale allocation drafts and invalid member selections are reset as before.

## Design Notes

The current selector owns no separate allocation state: it delegates to `DepartmentMembersPanel`, which publishes the selected department member to the task-page state. The allocation form already receives that selected member and resets from it. The change should therefore relocate composition only, preserving the single selected-member source and avoiding duplicated member queries or a second selection model.

## Verification

**Commands:**
- `cd web && bun test src/features/enterprise-organization/enterprise-organization.test.tsx` -- expected: affected layout regression tests pass.
- `cd web && bun run typecheck` -- expected: TypeScript completes without errors.
- `cd web && bun run lint src/features/enterprise-organization/index.tsx src/features/enterprise-organization/enterprise-organization.test.tsx` -- expected: no lint errors in modified files.

## Suggested Review Order

**Allocation workflow placement**

- Carry the shared member context into allocation composition.
  [`index.tsx:1334`](../../web/src/features/enterprise-organization/index.tsx#L1334)

- Render member selection only with its consuming wallet workflow.
  [`index.tsx:4066`](../../web/src/features/enterprise-organization/index.tsx#L4066)

- Preserve the context heading without nesting cards.
  [`index.tsx:1354`](../../web/src/features/enterprise-organization/index.tsx#L1354)

**Regression coverage**

- Assert a single selector between the two allocation flows.
  [`enterprise-organization.test.tsx:2649`](../../web/src/features/enterprise-organization/enterprise-organization.test.tsx#L2649)
