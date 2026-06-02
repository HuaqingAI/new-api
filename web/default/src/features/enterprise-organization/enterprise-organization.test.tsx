import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  RouterContextProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router'
import i18n from '@/i18n/config'
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'
import {
  departmentOwnersQueryKey,
  governanceNotificationQueryKey,
  governanceTimelineQueryKey,
  quotaRequestQueryKey,
} from './api'
import {
  BudgetDelegationTable,
  __testRenderApiMessage,
  __testRenderUsernameMutationMessage,
  createQuotaRequestDecisionSchema,
  createQuotaRequestSchema,
  DepartmentMemberContextCard,
  DepartmentSummaryCard,
  enterpriseOrganizationSearchSchema,
  EnterpriseOrganizationContent,
  EnterpriseOrganizationWorkspace,
  DepartmentBudgetDetailTable,
  DepartmentBudgetListCard,
  DepartmentBudgetOverviewCard,
  DepartmentBudgetStatusCard,
  GovernanceActivityCard,
  QuotaAllocationTable,
  createBudgetSchema,
  createAllocationSchema,
  createDelegationSchema,
  createRenameUsernameSchema,
  normalizeEnterpriseOrganizationSearch,
  resolveDepartmentMemberSelection,
  resolveBudgetSelection,
  syncAllocationFormDraft,
} from './index'
import {
  getAncestorDepartmentIds,
  getDefaultExpandedDepartmentIds,
  resolveDepartmentSelection,
  syncExpandedDepartmentIds,
  toggleExpandedDepartmentId,
} from './lib/tree-utils'
import type {
  ApiResponse,
  BudgetDelegationItem,
  DepartmentBudgetDetailResponse,
  DepartmentBudgetItem,
  DepartmentMemberItem,
  DepartmentTreeNode,
  EnterpriseBudgetErrorData,
  QuotaAllocationItem,
  QuotaRequestItem,
  GovernanceNotificationItem,
  GovernanceTimelineItem,
} from './types'

i18n.changeLanguage('en')

const rootRoute = createRootRoute()
const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
})
const testRouter = createRouter({
  routeTree: rootRoute.addChildren([indexRoute]),
  history: createMemoryHistory({ initialEntries: ['/'] }),
})

describe('Enterprise organization department tree workflow', () => {
  test('renders the empty state with actionable disabled next-step entries', () => {
    const html = renderEnterpriseOrganizationContent([])

    assert.match(html, /No departments yet/)
    assert.match(html, /Start by configuring DingTalk synchronization/)
    assert.match(html, /Configure DingTalk sync/)
    assert.match(html, /Manual creation coming soon/)
    assert.match(html, /disabled/)
  })

  test('renders a three-level department tree with status, source, external id, and sync state', () => {
    const html = renderEnterpriseOrganizationContent(
      [
        departmentNode({
          id: 1,
          name: 'Headquarters',
          status: 1,
          source_type: 1,
          external_id: '',
          sync_status: 1,
          children: [
            departmentNode({
              id: 2,
              parent_id: 1,
              name: 'Engineering',
              status: 2,
              source_type: 2,
              external_id: 'dingtalk-engineering',
              sync_status: 2,
              name_history: [{ name: 'R&D', changed_at: 1700000000 }],
              children: [
                departmentNode({
                  id: 3,
                  parent_id: 2,
                  name: 'Platform',
                  status: 3,
                  source_type: 2,
                  external_id: 'dingtalk-platform',
                  sync_status: 3,
                  sync_error: 'Deleted upstream during sync',
                }),
              ],
            }),
          ],
        }),
      ],
      [1, 2]
    )

    for (const expected of [
      'Department',
      'Status',
      'Source',
      'External ID',
      'Sync',
      'Headquarters',
      'Engineering',
      'Platform',
      'Root department',
      'Parent ID 1',
      'Parent ID 2',
      'Enabled',
      'Disabled',
      'Deleted upstream',
      'Manual',
      'DingTalk',
      'Local only',
      'dingtalk-engineering',
      'dingtalk-platform',
      'Synced',
      'Sync warning',
      'Sync failed',
      '1 historical name(s)',
      'Deleted upstream during sync',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('search schema accepts valid department and budget identifiers and drops invalid values', () => {
    const parsed = enterpriseOrganizationSearchSchema.parse({
      dept_id: '12',
      budget_id: '34',
    })
    assert.equal(parsed.dept_id, 12)
    assert.equal(parsed.budget_id, 34)

    const invalid = enterpriseOrganizationSearchSchema.parse({
      dept_id: '0',
      budget_id: '-2',
    })
    assert.equal(invalid.dept_id, undefined)
    assert.equal(invalid.budget_id, undefined)
  })

  test('budget list card exposes descendant scope toggle and department labels', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetListCard
          items={[
            departmentBudget({ id: 1, department_name: 'Engineering' }),
            departmentBudget({ id: 2, department_name: 'Platform' }),
          ]}
          loading={false}
          selectedBudgetId={1}
          includeDescendants={true}
          scopeDepartmentName='Engineering'
          sortBy='usage_ratio'
          sortOrder='desc'
          onIncludeDescendantsChange={() => {}}
          onSelectBudget={() => {}}
          onSortByChange={() => {}}
          onSortOrderChange={() => {}}
        />
      </I18nextProvider>
    )

    assert.match(html, /Include descendants/)
    assert.match(html, /Engineering and all descendant departments/)
    assert.match(html, /Platform/)
  })

  test('normalizes stale search state for empty trees and invalid department ids', () => {
    assert.deepEqual(
      normalizeEnterpriseOrganizationSearch({
        departments: [],
        search: {
          dept_id: 7,
          budget_id: 12,
        },
      }),
      {
        dept_id: undefined,
        budget_id: undefined,
      }
    )

    const departments = [
      departmentNode({
        id: 1,
        name: 'Headquarters',
        children: [
          departmentNode({
            id: 2,
            parent_id: 1,
            name: 'Engineering',
          }),
        ],
      }),
    ]

    assert.deepEqual(
      normalizeEnterpriseOrganizationSearch({
        departments,
        search: {
          dept_id: 999,
          budget_id: 12,
        },
      }),
      {
        dept_id: 1,
        budget_id: undefined,
      }
    )
  })

  test('resolves department selection, ancestor expansion, and invalid fallback from current tree', () => {
    const tree = [
      departmentNode({
        id: 1,
        name: 'Headquarters',
        children: [
          departmentNode({
            id: 2,
            parent_id: 1,
            name: 'Engineering',
            children: [
              departmentNode({
                id: 3,
                parent_id: 2,
                name: 'Platform',
              }),
            ],
          }),
        ],
      }),
      departmentNode({
        id: 4,
        name: 'Operations',
      }),
    ]

    assert.deepEqual(getAncestorDepartmentIds(tree, 3), [1, 2])
    assert.deepEqual(getDefaultExpandedDepartmentIds(tree, 3), [1, 2, 4])

    const resolved = resolveDepartmentSelection(tree, 3)
    assert.equal(resolved.selectedDepartmentId, 3)
    assert.equal(resolved.normalizedDepartmentId, 3)
    assert.deepEqual(resolved.requiredExpandedIds, [1, 2, 4])

    const fallback = resolveDepartmentSelection(tree, 999)
    assert.equal(fallback.selectedDepartmentId, 1)
    assert.equal(fallback.normalizedDepartmentId, 1)
    assert.deepEqual(fallback.requiredExpandedIds, [1, 4])
  })

  test('syncs and toggles expanded department state without dropping required ancestors', () => {
    const tree = [
      departmentNode({
        id: 1,
        name: 'Headquarters',
        children: [
          departmentNode({
            id: 2,
            parent_id: 1,
            name: 'Engineering',
            children: [
              departmentNode({ id: 3, parent_id: 2, name: 'Platform' }),
            ],
          }),
        ],
      }),
    ]

    const synced = syncExpandedDepartmentIds([2, 999], tree, [1])
    assert.deepEqual(synced, [1, 2])
    assert.deepEqual(toggleExpandedDepartmentId(synced, 2), [1])
    assert.deepEqual(toggleExpandedDepartmentId([1], 2), [1, 2])
  })

  test('resolves budget selection within current department context only', () => {
    assert.equal(resolveBudgetSelection([11, 12], 12, 11), 12)
    assert.equal(resolveBudgetSelection([11, 12], 99, 12), 12)
    assert.equal(resolveBudgetSelection([11, 12], 99, 98), 11)
    assert.equal(resolveBudgetSelection([], 99, 98), null)
  })

  test('clears stale selected members when the current department member list changes', () => {
    const engineeringMembers = [
      departmentMember({
        id: 1,
        department_id: 2,
        user_id: 2001,
        username: 'alice',
      }),
      departmentMember({
        id: 2,
        department_id: 2,
        user_id: 2002,
        username: 'bob',
      }),
    ]
    const financeMembers = [
      departmentMember({
        id: 3,
        department_id: 8,
        user_id: 3001,
        username: 'carol',
      }),
    ]

    assert.equal(
      resolveDepartmentMemberSelection(engineeringMembers, 2002),
      2002
    )
    assert.equal(resolveDepartmentMemberSelection(financeMembers, 2002), null)
    assert.equal(resolveDepartmentMemberSelection([], 2002), null)
  })

  test('resets allocation draft when department, budget, or selected member context changes', () => {
    const current = {
      tenant_id: 9,
      department_id: 2,
      department_budget_id: 12,
      target_user_id: 2001,
      committed_quota: 300,
      reason: 'carry over',
    }

    assert.deepEqual(
      syncAllocationFormDraft({
        current,
        departmentId: 8,
        selectedBudgetId: null,
        selectedMemberUserId: null,
        resetMode: 'department-change',
      }),
      {
        tenant_id: 9,
        department_id: 8,
        department_budget_id: 0,
        target_user_id: 0,
        committed_quota: 0,
        reason: '',
      }
    )

    assert.deepEqual(
      syncAllocationFormDraft({
        current,
        departmentId: 2,
        selectedBudgetId: 19,
        selectedMemberUserId: 2001,
        resetMode: 'budget-change',
      }),
      {
        tenant_id: 9,
        department_id: 2,
        department_budget_id: 19,
        target_user_id: 2001,
        committed_quota: 0,
        reason: '',
      }
    )

    assert.deepEqual(
      syncAllocationFormDraft({
        current,
        departmentId: 2,
        selectedBudgetId: 12,
        selectedMemberUserId: 2002,
        resetMode: 'member-change',
      }),
      {
        tenant_id: 9,
        department_id: 2,
        department_budget_id: 12,
        target_user_id: 2002,
        committed_quota: 0,
        reason: '',
      }
    )
  })

  test('workspace empty state and scoped panels follow current department context', () => {
    const emptyHtml = renderWorkspace(null, null)

    assert.match(emptyHtml, /No departments yet/)

    const department = departmentNode({
      id: 7,
      name: 'Security',
      parent_id: 1,
      source_type: 2,
      external_id: 'dept-security',
      sync_status: 1,
      name_history: [{ name: 'InfoSec', changed_at: 1700000000 }],
    })
    const parent = departmentNode({
      id: 1,
      name: 'Headquarters',
    })
    const summaryHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentSummaryCard
          department={department}
          parentDepartment={parent}
        />
      </I18nextProvider>
    )

    for (const expected of [
      'Current Department Workspace',
      'Security',
      'Department ID #7',
      'Headquarters (#1)',
      'DingTalk',
      'InfoSec',
    ]) {
      assert.match(summaryHtml, new RegExp(escapeRegExp(expected)))
    }

    const html = renderWorkspace(department, parent)

    for (const expected of [
      'Department Members',
      'Department Owners',
      'No effective owner',
      'Admin fallback',
      'Department Budget',
      'Current Member Governance',
      'Current Member Wallet Allocation',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }

    assert.doesNotMatch(html, /Membership Lookup/)
  })

  test('department owner query key stays scoped to enterprise organization namespace', () => {
    assert.deepEqual(departmentOwnersQueryKey(7, 0), [
      'enterprise',
      'organization',
      'department-owners',
      7,
      0,
    ])
  })

  test('quota request query key stays scoped to enterprise organization namespace', () => {
    assert.deepEqual(quotaRequestQueryKey(7, 0, 100), [
      'enterprise',
      'organization',
      'quota-request',
      7,
      0,
      100,
    ])
  })

  test('governance query keys stay scoped to enterprise organization namespace', () => {
    assert.deepEqual(governanceTimelineQueryKey(7, 0), [
      'enterprise',
      'organization',
      'governance-timeline',
      7,
      0,
    ])
    assert.deepEqual(governanceNotificationQueryKey(7, 0), [
      'enterprise',
      'organization',
      'governance-notification',
      7,
      0,
    ])
  })

  test('renders governance timeline trace, delivery failure reason, and resend action', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <GovernanceActivityCard
          loading={false}
          timelineItems={[
            governanceTimelineItem({
              trace_id: 'quota_request:42',
              action_type: 'enterprise.organization.quota_request.approve',
              actor_name: 'Owner Alice',
              quota_delta: 80,
              status: 'fulfilled',
            }),
          ]}
          notificationItems={[
            governanceNotificationItem({
              id: 7,
              trace_id: 'quota_request:42',
              status: 'final_failed',
              error_reason: 'webhook request failed',
            }),
          ]}
          resendPendingId={null}
          onResend={() => undefined}
        />
      </I18nextProvider>
    )

    for (const expected of [
      'Governance Timeline and Notification Delivery',
      'quota_request:42',
      'Quota request approved',
      'Owner Alice',
      'Final failed',
      'webhook request failed',
      'Resend',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('renders department budget empty state and latest budget details', () => {
    const emptyHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetStatusCard item={null} />
      </I18nextProvider>
    )

    assert.match(emptyHtml, /Budget Pool Overview/)
    assert.match(emptyHtml, /No budget pool yet/)

    const filledHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetStatusCard item={departmentBudget()} />
      </I18nextProvider>
    )

    for (const expected of [
      'Subscription Budget',
      'Remaining Quota',
      'Cycle Quota',
      'Weekly',
      'Custom Cycle Seconds',
    ]) {
      assert.match(filledHtml, new RegExp(escapeRegExp(expected)))
    }
  })

  test('budget schema reports invalid balance and subscription inputs', () => {
    const schema = createBudgetSchema((key) => key)

    const balance = schema.safeParse({
      tenant_id: 0,
      department_id: 1,
      type: 'balance',
      total_quota: 0,
      cycle_quota: 0,
      cycle_type: 'monthly',
      cycle_started_at: '',
      custom_seconds: 0,
      expires_at: '',
    })
    assert.equal(balance.success, false)
    if (balance.success) return
    assert.match(
      JSON.stringify(balance.error.flatten().fieldErrors),
      /Balance budget quota must be greater than 0/
    )

    const subscription = schema.safeParse({
      tenant_id: 0,
      department_id: 1,
      type: 'subscription',
      total_quota: 0,
      cycle_quota: 0,
      cycle_type: 'custom',
      cycle_started_at: '',
      custom_seconds: 0,
      expires_at: '',
    })
    assert.equal(subscription.success, false)
    if (subscription.success) return
    const issues = JSON.stringify(subscription.error.flatten().fieldErrors)
    assert.match(
      issues,
      /Subscription budget cycle quota must be greater than 0/
    )
    assert.match(issues, /Subscription budget cycle start time is required/)
    assert.match(issues, /Custom cycle must be greater than 0 seconds/)
  })

  test('budget schema accepts valid balance and subscription inputs', () => {
    const schema = createBudgetSchema((key) => key)

    const balance = schema.safeParse({
      tenant_id: 0,
      department_id: 7,
      type: 'balance',
      total_quota: 1200,
      cycle_quota: 0,
      cycle_type: 'monthly',
      cycle_started_at: '',
      custom_seconds: 0,
      expires_at: '2026-05-29T12:00',
    })
    assert.equal(balance.success, true)

    const subscription = schema.safeParse({
      tenant_id: 2,
      department_id: 9,
      type: 'subscription',
      total_quota: 0,
      cycle_quota: 300,
      cycle_type: 'custom',
      cycle_started_at: '2026-05-29T08:30',
      custom_seconds: 7200,
      expires_at: '',
    })
    assert.equal(subscription.success, true)
  })

  test('quota request schema requires explicit budget selection', () => {
    const schema = createQuotaRequestSchema((key) => key)

    const invalid = schema.safeParse({
      tenant_id: 0,
      department_id: 1,
      department_budget_id: 0,
      budget_mode: 'department_budget',
      requested_quota: 0,
      request_reason: '',
    })
    assert.equal(invalid.success, false)
    if (invalid.success) return
    const issues = JSON.stringify(invalid.error.flatten().fieldErrors)
    assert.match(issues, /Choose a target budget pool/)
    assert.match(issues, /Requested quota must be greater than 0/)
  })

  test('quota request decision schema supports smaller approval and requires reject reason', () => {
    const schema = createQuotaRequestDecisionSchema((key) => key)

    const approve = schema.safeParse({
      action: 'approve',
      approved_quota: 80,
      approval_reason: 'approve smaller amount',
      rejected_reason: '',
    })
    assert.equal(approve.success, true)

    const reject = schema.safeParse({
      action: 'reject',
      approved_quota: 0,
      approval_reason: '',
      rejected_reason: '',
    })
    assert.equal(reject.success, false)
    if (reject.success) return
    assert.match(
      JSON.stringify(reject.error.flatten().fieldErrors),
      /Rejected requests must include a reject reason/
    )
  })

  test('allocation schema rejects empty target and quota, then accepts valid input', () => {
    const schema = createAllocationSchema((key) => key)

    const invalid = schema.safeParse({
      tenant_id: 0,
      department_id: 9,
      department_budget_id: 0,
      target_user_id: 0,
      committed_quota: 0,
      reason: '',
    })
    assert.equal(invalid.success, false)

    const valid = schema.safeParse({
      tenant_id: 0,
      department_id: 9,
      department_budget_id: 11,
      target_user_id: 2001,
      committed_quota: 300,
      reason: 'department allocation',
    })
    assert.equal(valid.success, true)
  })

  test('delegation schema rejects missing descendant target and quota, then accepts valid input', () => {
    const schema = createDelegationSchema((key) => key)

    const invalid = schema.safeParse({
      tenant_id: 0,
      source_department_id: 1,
      source_budget_id: 11,
      target_department_id: 0,
      target_budget_id: 0,
      committed_quota: 0,
      reason: '',
    })
    assert.equal(invalid.success, false)

    const valid = schema.safeParse({
      tenant_id: 0,
      source_department_id: 1,
      source_budget_id: 11,
      target_department_id: 3,
      target_budget_id: 21,
      committed_quota: 400,
      reason: 'delegate',
    })
    assert.equal(valid.success, true)
  })

  test('rename username schema rejects invalid values and accepts readable usernames', () => {
    const schema = createRenameUsernameSchema((key) => key)

    const invalid = schema.safeParse({
      tenant_id: 0,
      new_username: 'A',
    })
    assert.equal(invalid.success, false)

    const valid = schema.safeParse({
      tenant_id: 0,
      new_username: 'alice_ops',
    })
    assert.equal(valid.success, true)
  })

  test('budget error messaging prefers the specific reason key', () => {
    const payload: ApiResponse<EnterpriseBudgetErrorData> = {
      success: false,
      message: 'enterprise.organization.enterprise_budget_insufficient',
      data: {
        reason: 'enterprise.organization.balance_remaining_insufficient',
      },
    }

    assert.equal(
      __testRenderApiMessage(payload, (key) => key),
      'enterprise.organization.balance_remaining_insufficient'
    )
  })

  test('budget error messaging supports subscription allocation reason key', () => {
    const payload: ApiResponse<EnterpriseBudgetErrorData> = {
      success: false,
      message: 'enterprise.organization.enterprise_budget_insufficient',
      data: {
        reason: 'enterprise.organization.subscription_cycle_allocated_exceeded',
      },
    }

    assert.equal(
      __testRenderApiMessage(payload, (key) => key),
      'enterprise.organization.subscription_cycle_allocated_exceeded'
    )
  })

  test('budget error messaging falls back to message and generic failure text', () => {
    const withoutReason: ApiResponse<EnterpriseBudgetErrorData> = {
      success: false,
      message: 'enterprise.organization.enterprise_budget_insufficient',
      data: {},
    }
    assert.equal(
      __testRenderApiMessage(withoutReason, (key) => key),
      'enterprise.organization.enterprise_budget_insufficient'
    )

    assert.equal(
      __testRenderApiMessage(null, (key) => key),
      'Request failed'
    )
  })

  test('username mutation messaging prefers translated backend reason key', () => {
    assert.equal(
      __testRenderUsernameMutationMessage(
        { message: 'enterprise.organization.username_exists' },
        (key) => key
      ),
      'enterprise.organization.username_exists'
    )
    assert.equal(
      __testRenderUsernameMutationMessage(null, (key) => key),
      'Request failed'
    )
  })

  test('renders balance budget details without subscription-only fields', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetStatusCard
          item={departmentBudget({
            type: 'balance',
            total_quota: 800,
            remaining: 640,
            cycle_quota: 0,
            cycle_type: 'never',
            cycle_started_at: 0,
            custom_seconds: 0,
            expires_at: 0,
          })}
        />
      </I18nextProvider>
    )

    for (const expected of [
      'Balance Budget',
      'Remaining Quota',
      '640',
      'Total Quota',
      '800',
      'No Reset',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }

    assert.doesNotMatch(html, />Weekly</)
    assert.doesNotMatch(html, />Custom \\(seconds\\)</)
  })

  test('renders budget pool list with selectable threshold states and usage metrics', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetListCard
          loading={false}
          selectedBudgetId={12}
          sortBy='usage_ratio'
          sortOrder='desc'
          onSelectBudget={() => undefined}
          onSortByChange={() => undefined}
          onSortOrderChange={() => undefined}
          items={[
            departmentBudget({
              id: 12,
              type: 'subscription',
              status: 'active',
              remaining: 100,
              allocated_total: 400,
              cycle_quota: 500,
              usage_ratio: 80,
              threshold_state: 'warning',
            }),
            departmentBudget({
              id: 13,
              type: 'balance',
              status: 'paused',
              total_quota: 1000,
              remaining: 50,
              allocated_total: 950,
              cycle_quota: 0,
              usage_ratio: 95,
              threshold_state: 'critical',
            }),
          ]}
        />
      </I18nextProvider>
    )

    for (const expected of [
      'Budget Pool List',
      'Usage Ratio',
      'Threshold State',
      'Subscription Budget',
      'Balance Budget',
      '80%',
      '95%',
      'Warning',
      'Critical',
      'Paused',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('renders budget overview threshold window and parent lifecycle state', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetOverviewCard
          item={departmentBudget({
            id: 14,
            type: 'subscription',
            status: 'active',
            cycle_quota: 500,
            remaining: 100,
            allocated_total: 400,
            usage_ratio: 80,
            threshold_state: 'warning',
            parent_status: 'paused',
          })}
          selectedBudget={departmentBudget({
            id: 14,
            type: 'subscription',
            status: 'active',
            cycle_quota: 500,
            remaining: 100,
            allocated_total: 400,
            usage_ratio: 80,
            threshold_state: 'warning',
            parent_status: 'paused',
          })}
          thresholds={{ warning: 70, critical: 90 }}
        />
      </I18nextProvider>
    )

    for (const expected of [
      'Budget Pool Overview',
      'Threshold Window',
      '70% / 90%',
      'Parent Status',
      'Paused',
      'Threshold State',
      'Warning',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('renders budget detail wallet lineage with revoked and expired states', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetDetailTable
          loading={false}
          budget={departmentBudget({
            id: 15,
            type: 'subscription',
            status: 'active',
            cycle_quota: 500,
            remaining: 100,
            allocated_total: 400,
          })}
          wallets={[
            walletDetail({
              allocation_id: 31,
              allocation_status: 'revoked',
              wallet_id: 41,
              wallet_status: 'expired',
              target_user_id: 2001,
              target_username: 'alice',
              target_display_name: 'Alice',
              quota: 300,
              remain_quota: 180,
              cycle_type: 'monthly',
              next_reset_time: 1700000500,
              source_allocation_id: 31,
              source_parent_budget_id: 15,
              source_parent_budget_type: 'subscription',
              source_parent_budget_status: 'paused',
              processed_at: 1700000600,
            }),
          ]}
        />
      </I18nextProvider>
    )

    for (const expected of [
      'Target User',
      'Wallet',
      'Cycle / Expiry',
      'Wallet Status',
      'Allocation Status',
      'Alice',
      'alice · User ID #2001',
      '#41',
      '300',
      '180',
      'Monthly',
      'Next Reset',
      'Expired',
      'Revoked',
      'Allocation #31',
      'Parent Budget #15',
      'Subscription Budget',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('renders budget detail empty states for unselected and wallet-free pools', () => {
    const unselectedHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetDetailTable
          loading={false}
          budget={null}
          wallets={[]}
        />
      </I18nextProvider>
    )
    assert.match(unselectedHtml, /Select a budget pool/)
    assert.match(
      unselectedHtml,
      /Choose a budget pool from the list to inspect derived wallets and allocation lineage\./
    )

    const noWalletHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetDetailTable
          loading={false}
          budget={departmentBudget({ id: 16 })}
          wallets={[]}
        />
      </I18nextProvider>
    )
    assert.match(noWalletHtml, /No derived wallets yet/)
    assert.match(
      noWalletHtml,
      /This budget pool has no derived wallets yet\. Create an allocation below to start tracking wallet state\./
    )
  })

  test('renders quota allocation empty state with enterprise wallet guidance', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <QuotaAllocationTable items={[]} loading={false} />
      </I18nextProvider>
    )

    assert.match(html, /No wallet allocations yet/)
    assert.match(
      html,
      /Create an allocation to place a department wallet ahead of the member primary wallet\./
    )
  })

  test('renders budget delegation empty state with descendant guidance', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <BudgetDelegationTable items={[]} loading={false} />
      </I18nextProvider>
    )

    assert.match(html, /No budget delegations yet/)
    assert.match(
      html,
      /Choose a descendant department budget pool to create the first delegation in this governance chain\./
    )
  })

  test('renders budget delegation rows with route, quota, status, and supersede guidance', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <BudgetDelegationTable
          loading={false}
          items={[
            budgetDelegation({
              id: 41,
              source_department_name: 'Engineering',
              target_department_name: 'Platform',
              target_budget_id: 23,
              committed_quota: 450,
              status: 'active',
            }),
            budgetDelegation({
              id: 42,
              source_department_name: 'Engineering',
              target_department_name: 'Platform',
              target_budget_id: 24,
              committed_quota: 300,
              status: 'superseded',
            }),
          ]}
        />
      </I18nextProvider>
    )

    for (const expected of [
      'Delegation Route',
      'Delegation Quota',
      'Engineering -&gt; Platform -&gt; Budget #23',
      '450',
      'Active',
      'Close old delegation and create a new one',
      'Superseded',
      'Historical delegation',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('renders quota allocation rows with committed quota, wallet id, and status', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <QuotaAllocationTable
          loading={false}
          items={[
            quotaAllocation({
              id: 7,
              department_budget_id: 11,
              department_id: 9,
              target_user_id: 2001,
              wallet_id: 301,
              actor_id: 1001,
              committed_quota: 300,
              status: 'active',
              processed_at: 1700000300,
            }),
          ]}
        />
      </I18nextProvider>
    )

    for (const expected of [
      'Target User ID',
      'Allocation Quota',
      'Wallet ID',
      'Status',
      'Processed At',
      'Actions',
      'Alice',
      'alice · User ID #2001',
      '300',
      '301',
      'Active',
      'Close old allocation and create a new one',
      'Cancel allocation',
      'Reclaim allocation',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('renders quota allocation governance states with lineage and action guidance', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <QuotaAllocationTable
          loading={false}
          items={[
            quotaAllocation({
              id: 10,
              status: 'active',
              supersedes_allocation_id: 7,
              reclaimed_quota: 0,
            }),
            quotaAllocation({
              id: 11,
              status: 'superseded',
              superseded_by_id: 10,
              processed_at: 1700000450,
            }),
            quotaAllocation({
              id: 12,
              status: 'closed',
              reclaimed_quota: 175,
              processed_at: 1700000500,
            }),
          ]}
        />
      </I18nextProvider>
    )

    for (const expected of [
      'Superseded',
      'Closed',
      'Supersedes Allocation #7',
      'Superseded By Allocation #10',
      'Reclaimed Quota',
      '175',
      'Close old allocation and create a new one',
      'Historical allocation',
      'Reclaim allocation',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('renders quota request rows with explicit target department, approved quota, and fulfillment result', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <table>
          <tbody>
            <tr>
              <td>{quotaRequest().department_name}</td>
              <td>{quotaRequest().requested_quota}</td>
              <td>{quotaRequest({ approved_quota: 80 }).approved_quota}</td>
              <td>{quotaRequest({ allocation_id: 21 }).allocation_id}</td>
              <td>{quotaRequest({ status: 'fulfilled' }).status}</td>
            </tr>
          </tbody>
        </table>
      </I18nextProvider>
    )

    for (const expected of ['Engineering', '120', '80', '21', 'fulfilled']) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('renders revoked allocation as already processed', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <QuotaAllocationTable
          loading={false}
          items={[
            quotaAllocation({
              id: 8,
              status: 'revoked',
              processed_at: 1700000400,
            }),
          ]}
        />
      </I18nextProvider>
    )

    assert.match(html, /Revoked/)
    assert.match(html, /Already processed/)
  })

  test('renders expired allocation as already processed', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <QuotaAllocationTable
          loading={false}
          items={[
            quotaAllocation({
              id: 9,
              status: 'expired',
              processed_at: 1700000500,
            }),
          ]}
        />
      </I18nextProvider>
    )

    assert.match(html, /Expired/)
    assert.match(html, /Already processed/)
  })

  test('renders current member governance states inside the current department context', () => {
    const department = departmentNode({
      id: 7,
      name: 'Security',
    })
    const selectedMember = departmentMember({
      department_id: 7,
      user_id: 2001,
      username: 'alice',
      display_name: 'Alice',
      status: 1,
    })

    const emptyHtml = renderToStaticMarkup(
      <QueryClientProvider client={new QueryClient()}>
        <I18nextProvider i18n={i18n}>
          <DepartmentMemberContextCard
            currentDepartment={department}
            selectedMember={null}
            memberships={[]}
            loading={false}
          />
        </I18nextProvider>
      </QueryClientProvider>
    )
    assert.match(emptyHtml, /Select a department member/)
    assert.match(
      emptyHtml,
      /Choose a member from the current department list to inspect memberships and create wallet allocations without typing a user identifier\./
    )

    const selectedHtml = renderToStaticMarkup(
      <QueryClientProvider client={new QueryClient()}>
        <I18nextProvider i18n={i18n}>
          <DepartmentMemberContextCard
            currentDepartment={department}
            selectedMember={selectedMember}
            memberships={[
              {
                id: 91,
                tenant_id: 0,
                user_id: 2001,
                department_id: 7,
                department_name: 'Security',
                external_user_id: '',
                external_source: 'dingtalk',
                status: 1,
                joined_at: 1700000000,
                left_at: 0,
                created_at: 1700000000,
                updated_at: 1700000001,
              },
            ]}
            loading={false}
          />
        </I18nextProvider>
      </QueryClientProvider>
    )

    for (const expected of [
      'Current Member Governance',
      'Alice',
      'Current department member',
      'Selected from Security',
      'Rename Username',
      'Readable Username',
      'Update Username',
      'Department',
      'Membership Status',
    ]) {
      assert.match(selectedHtml, new RegExp(escapeRegExp(expected)))
    }
  })
})

function renderEnterpriseOrganizationContent(
  departments: DepartmentTreeNode[],
  expandedIds?: number[]
) {
  return renderToStaticMarkup(
    <RouterContextProvider router={testRouter}>
      <I18nextProvider i18n={i18n}>
        <EnterpriseOrganizationContent
          isLoading={false}
          departments={departments}
          expandedIds={expandedIds}
        />
      </I18nextProvider>
    </RouterContextProvider>
  )
}

function renderWorkspace(
  currentDepartment: DepartmentTreeNode | null,
  parentDepartment: DepartmentTreeNode | null
) {
  const queryClient = new QueryClient()
  return renderToStaticMarkup(
    <RouterContextProvider router={testRouter}>
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <EnterpriseOrganizationWorkspace
            currentDepartment={currentDepartment}
            parentDepartment={parentDepartment}
            selectedBudgetId={null}
            onSelectedBudgetIdChange={() => undefined}
          />
        </I18nextProvider>
      </QueryClientProvider>
    </RouterContextProvider>
  )
}

function departmentNode(
  overrides: Partial<DepartmentTreeNode> &
    Pick<DepartmentTreeNode, 'id' | 'name'>
): DepartmentTreeNode {
  return {
    id: overrides.id,
    tenant_id: overrides.tenant_id ?? 0,
    name: overrides.name,
    parent_id: overrides.parent_id ?? null,
    status: overrides.status ?? 1,
    source_type: overrides.source_type ?? 1,
    external_id: overrides.external_id ?? '',
    sync_status: overrides.sync_status ?? 0,
    sync_error: overrides.sync_error ?? '',
    name_history: overrides.name_history ?? [],
    created_at: overrides.created_at ?? 1700000000,
    updated_at: overrides.updated_at ?? 1700000000,
    deleted_at: overrides.deleted_at ?? 0,
    children: overrides.children ?? [],
  }
}

function departmentBudget(
  overrides: Partial<DepartmentBudgetItem> = {}
): DepartmentBudgetItem {
  return {
    id: overrides.id ?? 1,
    tenant_id: overrides.tenant_id ?? 0,
    department_id: overrides.department_id ?? 2,
    department_name: overrides.department_name ?? 'Engineering',
    type: overrides.type ?? 'subscription',
    status: overrides.status ?? 'active',
    total_quota: overrides.total_quota ?? 0,
    remaining: overrides.remaining ?? 300,
    allocated_total: overrides.allocated_total ?? 0,
    cycle_quota: overrides.cycle_quota ?? 300,
    cycle_type: overrides.cycle_type ?? 'weekly',
    cycle_started_at: overrides.cycle_started_at ?? 1700000000,
    custom_seconds: overrides.custom_seconds ?? 0,
    expires_at: overrides.expires_at ?? 0,
    parent_status: overrides.parent_status ?? '',
    usage_ratio: overrides.usage_ratio ?? 0,
    threshold_state: overrides.threshold_state ?? 'healthy',
    created_at: overrides.created_at ?? 1700000000,
    updated_at: overrides.updated_at ?? 1700000001,
  }
}

function departmentMember(
  overrides: Partial<DepartmentMemberItem> &
    Pick<DepartmentMemberItem, 'department_id' | 'user_id'>
): DepartmentMemberItem {
  return {
    id: overrides.id ?? overrides.user_id,
    tenant_id: overrides.tenant_id ?? 0,
    user_id: overrides.user_id,
    username: overrides.username ?? `user-${overrides.user_id}`,
    display_name: overrides.display_name ?? '',
    department_id: overrides.department_id,
    external_user_id: overrides.external_user_id ?? '',
    external_source: overrides.external_source ?? 'manual',
    status: overrides.status ?? 1,
    joined_at: overrides.joined_at ?? 1700000000,
    left_at: overrides.left_at ?? 0,
    created_at: overrides.created_at ?? 1700000000,
    updated_at: overrides.updated_at ?? 1700000001,
  }
}

function quotaAllocation(
  overrides: Partial<QuotaAllocationItem> = {}
): QuotaAllocationItem {
  return {
    id: overrides.id ?? 1,
    tenant_id: overrides.tenant_id ?? 0,
    department_budget_id: overrides.department_budget_id ?? 1,
    department_id: overrides.department_id ?? 2,
    target_user_id: overrides.target_user_id ?? 2001,
    target_username: overrides.target_username ?? 'alice',
    target_display_name: overrides.target_display_name ?? 'Alice',
    wallet_id: overrides.wallet_id ?? 101,
    actor_id: overrides.actor_id ?? 1001,
    committed_quota: overrides.committed_quota ?? 300,
    budget_type_snapshot: overrides.budget_type_snapshot ?? 'balance',
    cycle_type_snapshot: overrides.cycle_type_snapshot ?? 'never',
    cycle_started_at_snapshot: overrides.cycle_started_at_snapshot ?? 0,
    custom_seconds_snapshot: overrides.custom_seconds_snapshot ?? 0,
    expires_at_snapshot: overrides.expires_at_snapshot ?? 0,
    reason: overrides.reason ?? '',
    status: overrides.status ?? 'active',
    superseded_by_id: overrides.superseded_by_id ?? 0,
    supersedes_allocation_id: overrides.supersedes_allocation_id ?? 0,
    revoke_reason: overrides.revoke_reason ?? '',
    reclaimed_quota: overrides.reclaimed_quota ?? 0,
    processed_source: overrides.processed_source ?? '',
    processed_at: overrides.processed_at ?? 0,
    created_at: overrides.created_at ?? 1700000000,
    updated_at: overrides.updated_at ?? 1700000001,
  }
}

function budgetDelegation(
  overrides: Partial<BudgetDelegationItem> = {}
): BudgetDelegationItem {
  return {
    id: overrides.id ?? 1,
    tenant_id: overrides.tenant_id ?? 0,
    source_department_id: overrides.source_department_id ?? 1,
    source_department_name: overrides.source_department_name ?? 'HQ',
    source_budget_id: overrides.source_budget_id ?? 11,
    target_department_id: overrides.target_department_id ?? 3,
    target_department_name: overrides.target_department_name ?? 'Platform',
    target_budget_id: overrides.target_budget_id ?? 21,
    actor_id: overrides.actor_id ?? 1001,
    committed_quota: overrides.committed_quota ?? 300,
    budget_type_snapshot: overrides.budget_type_snapshot ?? 'balance',
    cycle_type_snapshot: overrides.cycle_type_snapshot ?? 'never',
    before_source_budget_snapshot:
      overrides.before_source_budget_snapshot ?? '{}',
    after_source_budget_snapshot:
      overrides.after_source_budget_snapshot ?? '{}',
    before_target_budget_snapshot:
      overrides.before_target_budget_snapshot ?? '{}',
    after_target_budget_snapshot:
      overrides.after_target_budget_snapshot ?? '{}',
    status: overrides.status ?? 'active',
    superseded_by_id: overrides.superseded_by_id ?? 0,
    processed_at: overrides.processed_at ?? 0,
    reason: overrides.reason ?? '',
    created_at: overrides.created_at ?? 1700000000,
    updated_at: overrides.updated_at ?? 1700000001,
  }
}

function quotaRequest(
  overrides: Partial<QuotaRequestItem> = {}
): QuotaRequestItem {
  return {
    id: overrides.id ?? 1,
    tenant_id: overrides.tenant_id ?? 0,
    department_id: overrides.department_id ?? 2,
    department_name: overrides.department_name ?? 'Engineering',
    department_budget_id: overrides.department_budget_id ?? 11,
    budget_mode: overrides.budget_mode ?? 'department_budget',
    requester_user_id: overrides.requester_user_id ?? 2001,
    requester_username: overrides.requester_username ?? 'alice',
    requester_display_name: overrides.requester_display_name ?? 'Alice',
    requested_quota: overrides.requested_quota ?? 120,
    approved_quota: overrides.approved_quota ?? 0,
    status: overrides.status ?? 'submitted',
    approver_user_id: overrides.approver_user_id ?? 0,
    approver_username: overrides.approver_username ?? '',
    approval_reason: overrides.approval_reason ?? '',
    request_reason: overrides.request_reason ?? '',
    allocation_id: overrides.allocation_id ?? 0,
    owner_count_snapshot: overrides.owner_count_snapshot ?? 1,
    fallback: overrides.fallback ?? '',
    submitted_at: overrides.submitted_at ?? 1700000000,
    approved_at: overrides.approved_at ?? 0,
    rejected_at: overrides.rejected_at ?? 0,
    fulfilled_at: overrides.fulfilled_at ?? 0,
    processed_at: overrides.processed_at ?? 0,
    expires_at: overrides.expires_at ?? 0,
    created_at: overrides.created_at ?? 1700000000,
    updated_at: overrides.updated_at ?? 1700000001,
  }
}

function governanceTimelineItem(
  overrides: Partial<GovernanceTimelineItem> = {}
): GovernanceTimelineItem {
  return {
    trace_id: overrides.trace_id ?? 'quota_request:1',
    source_type: overrides.source_type ?? 'quota_request',
    source_id: overrides.source_id ?? 1,
    action_type:
      overrides.action_type ?? 'enterprise.organization.quota_request.submit',
    tenant_id: overrides.tenant_id ?? 0,
    actor_id: overrides.actor_id ?? 1001,
    actor_name: overrides.actor_name ?? 'Alice',
    target:
      overrides.target ??
      {
        department_id: 2,
        department_name: 'Engineering',
        user_id: 2001,
        username: 'alice',
        display_name: 'Alice',
        object_type: 'enterprise_quota_request',
        object_id: '1',
      },
    quota_delta: overrides.quota_delta ?? 100,
    before_quota: overrides.before_quota ?? 0,
    after_quota: overrides.after_quota ?? 0,
    status: overrides.status ?? 'submitted',
    occurred_at: overrides.occurred_at ?? 1700000000,
    detail_route: overrides.detail_route ?? '/enterprise-organization',
    detail_api_path:
      overrides.detail_api_path ?? '/api/enterprise/governance/timeline',
    summary: overrides.summary ?? '',
  }
}

function governanceNotificationItem(
  overrides: Partial<GovernanceNotificationItem> = {}
): GovernanceNotificationItem {
  return {
    id: overrides.id ?? 1,
    tenant_id: overrides.tenant_id ?? 0,
    source_type: overrides.source_type ?? 'quota_request',
    source_id: overrides.source_id ?? 1,
    trace_id: overrides.trace_id ?? 'quota_request:1',
    action_type:
      overrides.action_type ?? 'enterprise.organization.quota_request.submit',
    recipient_user_id: overrides.recipient_user_id ?? 1001,
    recipient_kind: overrides.recipient_kind ?? 'owner',
    channel_type: overrides.channel_type ?? 'dingtalk_robot',
    status: overrides.status ?? 'pending',
    attempt_count: overrides.attempt_count ?? 0,
    max_attempts: overrides.max_attempts ?? 4,
    next_retry_at: overrides.next_retry_at ?? 0,
    last_attempt_at: overrides.last_attempt_at ?? 0,
    sent_at: overrides.sent_at ?? 0,
    final_failed_at: overrides.final_failed_at ?? 0,
    error_reason: overrides.error_reason ?? '',
    dedupe_key: overrides.dedupe_key ?? 'dedupe',
    trigger_source: overrides.trigger_source ?? 'governance_action',
    manual_parent_id: overrides.manual_parent_id,
    trace_summary: overrides.trace_summary ?? '',
    trace: overrides.trace,
    created_at: overrides.created_at ?? 1700000000,
    updated_at: overrides.updated_at ?? 1700000001,
  }
}

function walletDetail(
  overrides: Partial<DepartmentBudgetDetailResponse['wallets'][number]> = {}
): DepartmentBudgetDetailResponse['wallets'][number] {
  return {
    allocation_id: overrides.allocation_id ?? 1,
    allocation_status: overrides.allocation_status ?? 'active',
    target_user_id: overrides.target_user_id ?? 2001,
    target_username: overrides.target_username ?? 'alice',
    target_display_name: overrides.target_display_name ?? 'Alice',
    wallet_id: overrides.wallet_id ?? 101,
    wallet_status: overrides.wallet_status ?? 'active',
    quota: overrides.quota ?? 300,
    remain_quota: overrides.remain_quota ?? 250,
    cycle_type: overrides.cycle_type ?? 'monthly',
    cycle_started_at: overrides.cycle_started_at ?? 1700000000,
    next_reset_time: overrides.next_reset_time ?? 1700000200,
    expires_at: overrides.expires_at ?? 0,
    source_allocation_id: overrides.source_allocation_id ?? 1,
    source_parent_budget_id: overrides.source_parent_budget_id ?? 1,
    source_parent_budget_type:
      overrides.source_parent_budget_type ?? 'subscription',
    source_parent_budget_status:
      overrides.source_parent_budget_status ?? 'active',
    committed_quota: overrides.committed_quota ?? 300,
    processed_at: overrides.processed_at ?? 0,
    created_at: overrides.created_at ?? 1700000000,
    updated_at: overrides.updated_at ?? 1700000001,
    reason: overrides.reason ?? '',
  }
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
