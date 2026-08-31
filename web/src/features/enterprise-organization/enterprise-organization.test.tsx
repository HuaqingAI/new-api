import { describe, test } from 'bun:test'
import assert from 'node:assert/strict'

import { zodResolver } from '@hookform/resolvers/zod'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  RouterContextProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router'
import { renderToStaticMarkup } from 'react-dom/server'
import { useForm, type Resolver } from 'react-hook-form'
import { I18nextProvider } from 'react-i18next'

import { Card, CardContent } from '@/components/ui/card'
import i18n, { resources } from '@/i18n/config'
import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import {
  createQuotaAllocation,
  departmentMembersQueryKey,
  departmentOwnersQueryKey,
  getDepartmentBudgetDetail,
  getDepartmentBudgets,
  getDepartmentMembers,
  getQuotaAllocations,
  governanceNotificationQueryKey,
  governanceTimelineQueryKey,
  quotaRequestQueryKey,
  renameDepartmentMember,
} from './api'
import { DepartmentTree } from './components/DepartmentTree'
import {
  BudgetDelegationTable,
  __testRenderApiMessage,
  __testRenderUsernameMutationMessage,
  createQuotaRequestDecisionSchema,
  createQuotaRequestSchema,
  DepartmentMemberContextCard,
  DepartmentSummaryCard,
  DepartmentBudgetPanel,
  BudgetDelegationOption,
  enterpriseOrganizationTaskSearchSchema,
  enterpriseOrganizationSearchSchema,
  EnterpriseOrganizationContent,
  EnterpriseOrganizationWorkspace,
  DepartmentTreeBulkActions,
  DepartmentBudgetDetailTable,
  DepartmentBudgetListCard,
  DepartmentBudgetOverviewCard,
  DepartmentBudgetStatusCard,
  GovernanceActivityCard,
  QuotaRequestTable,
  QuotaAllocationTable,
  canManageBudgetLifecycle,
  createBudgetResizeSchema,
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
  getCollapsedDepartmentIds,
  getDefaultExpandedDepartmentIds,
  getExpandableDepartmentIds,
  resolveDepartmentSelection,
  syncExpandedDepartmentIds,
  toggleExpandedDepartmentId,
} from './lib/tree-utils'
import {
  formatEnterpriseUserPrimary,
  formatEnterpriseUserSecondary,
} from './lib/user-display'
import {
  convertEnterpriseQuotaInputMode,
  formatEnterpriseQuotaAmount,
  parseEnterpriseQuotaInput,
  QuotaAmountDisplay,
} from './quota-amount-controls'
import {
  enterpriseBudgetStatusLabel,
  formatBudgetType,
  getQuotaRequestBudgetDisplayText,
  getQuotaRequestBudgetTriggerLabel,
} from './quota-request-budget-display'
import {
  QuotaRequestBudgetOption,
  QuotaRequestBudgetSummary,
} from './quota-request-budget-display-components'
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
  test('formats enterprise user labels with readable name, username, then user id fallback', () => {
    const t = (value: string) => value

    assert.equal(
      formatEnterpriseUserPrimary({
        displayName: 'Alice Zhang',
        username: 'alice_ops',
        userId: 2001,
      }),
      'Alice Zhang'
    )
    assert.equal(
      formatEnterpriseUserSecondary(
        {
          displayName: 'Alice Zhang',
          username: 'alice_ops',
          userId: 2001,
        },
        t
      ),
      'alice_ops · User ID #2001'
    )
    assert.equal(
      formatEnterpriseUserPrimary({
        displayName: '',
        username: 'alice_ops',
        userId: 2001,
      }),
      'alice_ops'
    )
    assert.equal(
      formatEnterpriseUserPrimary({
        displayName: '',
        username: '',
        userId: 2001,
      }),
      '#2001'
    )
    assert.equal(
      formatEnterpriseUserSecondary(
        {
          displayName: '',
          username: '',
          userId: 2001,
        },
        t
      ),
      'User ID #2001'
    )
    assert.equal(
      formatEnterpriseUserSecondary(
        {
          displayName: 'Alice Zhang',
          username: 'alice_ops',
          userId: null,
        },
        t
      ),
      'alice_ops'
    )
    assert.equal(
      formatEnterpriseUserSecondary(
        {
          displayName: 'Alice Zhang',
          username: 'alice_ops',
        },
        t
      ),
      'alice_ops'
    )
    assert.equal(formatEnterpriseUserPrimary({}), '-')
    assert.equal(formatEnterpriseUserSecondary({}, t), '-')
    assert.notEqual(formatEnterpriseUserSecondary({}, t), 'User ID #-')
  })

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

  test('renders collapsed, expanded, and selected tree states', () => {
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
    ]

    const collapsedHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentTree
          nodes={tree}
          expandedIds={[1]}
          selectedDepartmentId={2}
          onToggleExpand={() => undefined}
          onSelectDepartment={() => undefined}
        />
      </I18nextProvider>
    )

    assert.match(collapsedHtml, /Headquarters/)
    assert.match(collapsedHtml, /Engineering/)
    assert.doesNotMatch(collapsedHtml, /Platform/)
    assert.match(collapsedHtml, /aria-label="Collapse department"/)
    assert.match(collapsedHtml, /aria-label="Expand department"/)
    assert.match(collapsedHtml, /bg-muted\/50/)

    const expandedHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentTree
          nodes={tree}
          expandedIds={[1, 2]}
          selectedDepartmentId={3}
          onToggleExpand={() => undefined}
          onSelectDepartment={() => undefined}
        />
      </I18nextProvider>
    )

    assert.match(expandedHtml, /Platform/)
    assert.match(expandedHtml, /Parent ID 2/)
    assert.match(expandedHtml, /bg-muted\/50/)
  })

  test('renders department tree bulk expand and collapse actions', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentTreeBulkActions
          onExpandAll={() => undefined}
          onCollapse={() => undefined}
        />
      </I18nextProvider>
    )

    assert.match(html, /Expand All/)
    assert.match(html, /Collapse All/)
  })

  test('keeps department tree panel height constrained for internal scrolling', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <Card className='h-full min-h-0 overflow-hidden'>
          <CardContent className='min-h-0 flex-1 overflow-y-auto px-0 pb-0'>
            <EnterpriseOrganizationContent
              isLoading={false}
              departments={[
                departmentNode({
                  id: 1,
                  name: 'Headquarters',
                  children: [departmentNode({ id: 2, name: 'Finance' })],
                }),
              ]}
              expandedIds={[1]}
            />
          </CardContent>
        </Card>
      </I18nextProvider>
    )

    assert.match(html, /h-full min-h-0 overflow-hidden/)
    assert.match(html, /min-h-0 flex-1 overflow-y-auto/)
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

  test('task search schema keeps member context for split workspaces', () => {
    const parsed = enterpriseOrganizationTaskSearchSchema.parse({
      dept_id: '8',
      budget_id: '12',
      member_user_id: '2001',
    })

    assert.deepEqual(parsed, {
      dept_id: 8,
      budget_id: 12,
      member_user_id: 2001,
    })

    const invalid = enterpriseOrganizationTaskSearchSchema.parse({
      dept_id: 'x',
      budget_id: '-1',
      member_user_id: '0',
    })

    assert.equal(invalid.dept_id, undefined)
    assert.equal(invalid.budget_id, undefined)
    assert.equal(invalid.member_user_id, undefined)
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

  test('budget overview exposes lifecycle controls and immutable type guidance', () => {
    function LifecycleOverview() {
      const resizeSchema = createBudgetResizeSchema((key) => key)
      const resizeForm = useForm<{ quota: number }>({
        resolver: zodResolver(resizeSchema) as unknown as Resolver<{
          quota: number
        }>,
        defaultValues: { quota: 1000 },
      })
      return (
        <DepartmentBudgetOverviewCard
          item={departmentBudget({
            type: 'balance',
            status: 'active',
            total_quota: 1000,
            remaining: 700,
          })}
          selectedBudget={null}
          thresholds={{ warning: 80, critical: 95 }}
          resizeForm={resizeForm}
          onPause={() => {}}
          onResume={() => {}}
          onResize={() => {}}
        />
      )
    }

    const activeHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <LifecycleOverview />
      </I18nextProvider>
    )

    assert.match(activeHtml, /Budget pool governance/)
    assert.match(activeHtml, /Pause budget pool/)
    assert.match(activeHtml, /Resize budget pool/)
    assert.match(
      activeHtml,
      /Budget type cannot be changed\. Pause this pool and create a new pool for a different type\./
    )
    assert.doesNotMatch(activeHtml, /Subscription Budget/)

    const pausedHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetOverviewCard
          item={departmentBudget({ status: 'paused' })}
          selectedBudget={null}
          thresholds={{ warning: 80, critical: 95 }}
          onPause={() => {}}
          onResume={() => {}}
          onResize={() => {}}
        />
      </I18nextProvider>
    )

    assert.match(pausedHtml, /Resume budget pool/)
    assert.doesNotMatch(pausedHtml, /Pause budget pool/)
  })

  test('budget lifecycle governance is limited to enterprise administrators', () => {
    assert.equal(canManageBudgetLifecycle(undefined), false)
    assert.equal(canManageBudgetLifecycle(1), false)
    assert.equal(canManageBudgetLifecycle(10), true)
    assert.equal(canManageBudgetLifecycle(100), true)
  })

  test('budget pool creation entry is limited to enterprise administrators', () => {
    const { auth } = useAuthStore.getState()
    const previousUser = auth.user
    const department = departmentNode({ id: 1, name: 'Engineering' })

    try {
      auth.setUser({
        id: 1001,
        username: 'dept-admin',
        role: ROLE.USER,
      })
      const departmentAdminHtml = renderWorkspace(department, null)
      assert.doesNotMatch(departmentAdminHtml, /Create Budget Pool/)
      assert.match(departmentAdminHtml, /Budget Pool Overview/)

      auth.setUser({
        id: 1002,
        username: 'enterprise-admin',
        role: ROLE.ADMIN,
      })
      const enterpriseAdminHtml = renderWorkspace(department, null)
      assert.match(enterpriseAdminHtml, /Create Budget Pool/)
      assert.match(enterpriseAdminHtml, /Budget Pool Overview/)
    } finally {
      auth.setUser(previousUser)
    }
  })

  test('budget pool type select displays the translated label instead of raw value', () => {
    const { auth } = useAuthStore.getState()
    const previousUser = auth.user

    try {
      auth.setUser({
        id: 1002,
        username: 'enterprise-admin',
        role: ROLE.ADMIN,
      })
      const html = renderWorkspace(
        departmentNode({ id: 7, name: 'Security' }),
        null
      )

      assert.match(html, /Budget Type/)
      assert.match(html, /Balance Budget/)
      assert.doesNotMatch(html, />balance</)
    } finally {
      auth.setUser(previousUser)
    }
  })

  test('normalizes stale search state for empty trees and invalid department ids', () => {
    assert.deepEqual(
      normalizeEnterpriseOrganizationSearch({
        departments: [],
        search: {
          dept_id: 7,
          budget_id: 12,
        },
        treeLoaded: false,
      }),
      {
        dept_id: 7,
        budget_id: 12,
      }
    )

    assert.deepEqual(
      normalizeEnterpriseOrganizationSearch({
        departments: [],
        search: {
          dept_id: 7,
          budget_id: 12,
        },
        treeLoaded: true,
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
        treeLoaded: true,
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
    assert.deepEqual(getDefaultExpandedDepartmentIds(tree, 3), [1, 2])
    assert.deepEqual(getExpandableDepartmentIds(tree), [1, 2])
    assert.deepEqual(getCollapsedDepartmentIds(tree, 3), [1, 2])
    assert.deepEqual(getCollapsedDepartmentIds(tree, 1), [])

    const resolved = resolveDepartmentSelection(tree, 3)
    assert.equal(resolved.selectedDepartmentId, 3)
    assert.equal(resolved.normalizedDepartmentId, 3)
    assert.deepEqual(resolved.requiredExpandedIds, [1, 2])

    const fallback = resolveDepartmentSelection(tree, 999)
    assert.equal(fallback.selectedDepartmentId, 1)
    assert.equal(fallback.normalizedDepartmentId, 1)
    assert.deepEqual(fallback.requiredExpandedIds, [1])
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

  test('selecting a child department does not expand unrelated departments', () => {
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
      departmentNode({
        id: 4,
        name: 'Operations',
        children: [departmentNode({ id: 5, parent_id: 4, name: 'Finance' })],
      }),
    ]

    const nextExpandedIds = syncExpandedDepartmentIds(
      [],
      tree,
      getDefaultExpandedDepartmentIds(tree, 3)
    )

    assert.deepEqual(nextExpandedIds, [1, 2])
  })

  test('resolves budget selection within current department context only', () => {
    assert.equal(resolveBudgetSelection([11, 12], 12, 11), 12)
    assert.equal(resolveBudgetSelection([11, 12], 99, 12), 12)
    assert.equal(resolveBudgetSelection([11, 12], 99, 98), 11)
    assert.equal(resolveBudgetSelection([], 99, 98), null)
  })

  test('resolves and renders same-department mixed budget pools without type filtering', () => {
    const mixedBudgets = [
      departmentBudget({
        id: 21,
        type: 'balance',
        department_id: 2,
        department_name: 'Engineering',
        total_quota: 1000,
        remaining: 800,
        cycle_quota: 0,
        usage_ratio: 20,
      }),
      departmentBudget({
        id: 22,
        type: 'subscription',
        department_id: 2,
        department_name: 'Engineering',
        cycle_quota: 500,
        remaining: 300,
        allocated_total: 200,
        usage_ratio: 40,
      }),
    ]

    assert.equal(
      resolveBudgetSelection(
        mixedBudgets.map((budget) => budget.id),
        22,
        21
      ),
      22
    )

    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetListCard
          loading={false}
          selectedBudgetId={22}
          sortBy='usage_ratio'
          sortOrder='desc'
          onSelectBudget={() => undefined}
          onSortByChange={() => undefined}
          onSortOrderChange={() => undefined}
          items={mixedBudgets}
        />
      </I18nextProvider>
    )

    assert.match(html, />#21</)
    assert.match(html, />#22</)
    assert.match(html, /Balance Budget/)
    assert.match(html, /Subscription Budget/)
    assert.match(html, /Usage Ratio/)
    assert.match(html, /Descending/)
    assert.match(html, /Engineering/)
  })

  test('formats organization quota request budget options with department, identity, type, remaining, and status', () => {
    const budget = departmentBudget({
      id: 31,
      department_name: 'Engineering',
      type: 'subscription',
      remaining: 300,
      status: 'paused',
    })
    const display = getQuotaRequestBudgetDisplayText(budget, i18n.t)
    assert.equal(
      getQuotaRequestBudgetTriggerLabel(budget, i18n.t),
      'Engineering · Budget #31'
    )

    assert.deepEqual(display, {
      departmentName: 'Engineering',
      identity: 'Budget #31',
      typeLabel: 'Subscription Budget',
      remainingLabel: 'Remaining 300 quota',
      remainingAmountLabel: 'Approx. $0.0006',
      statusLabel: 'Paused',
    })

    const optionHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <QuotaRequestBudgetOption item={budget} />
      </I18nextProvider>
    )
    const summaryHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <QuotaRequestBudgetSummary item={budget} />
      </I18nextProvider>
    )

    for (const html of [optionHtml, summaryHtml]) {
      for (const expected of [
        'Engineering',
        'Budget #31',
        'Subscription Budget',
        'Remaining 300 quota',
        'Approx. $0.0006',
        'Paused',
      ]) {
        assert.match(html, new RegExp(escapeRegExp(expected)))
      }
    }
  })

  test('formats subordinate budget options for both the list and selected value', () => {
    const budget = departmentBudget({
      id: 41,
      department_name: 'AI Native',
    })
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <BudgetDelegationOption
          item={budget}
          sourceDepartmentName='信息系统中心'
        />
      </I18nextProvider>
    )

    assert.match(html, /信息系统中心 -&gt; AI Native -&gt; Budget #41/)
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
      'Add member from user search',
      'Search users by username, display name, or email...',
      'Search for a user, then add the selected user to the current department.',
      'Department Owners',
      'No effective owner',
      'Admin fallback',
      'Budget Pool List',
      'Budget Pool Overview',
      'Current Member Governance',
      'Current Member Wallet Allocation',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }

    assert.ok(
      html.indexOf('Budget Pool List') < html.indexOf('Budget Pool Overview')
    )
    assert.doesNotMatch(html, /Create Budget Pool/)
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

  test('department member query key includes tenant context', () => {
    assert.deepEqual(departmentMembersQueryKey(7, 2), [
      'enterprise',
      'organization',
      'department-members',
      7,
      2,
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

  test('api calls keep members, budgets, and allocations scoped to the selected department', async () => {
    const originalGet = api.get
    const calls: Array<{ url: string; params?: unknown }> = []

    api.get = (async (url: string, config?: Record<string, unknown>) => {
      calls.push({ url, params: config?.params })
      return {
        data: {
          success: true,
          message: '',
          data: {
            items: [],
            total: 0,
            thresholds: { warning: 80, critical: 95 },
          },
        },
      }
    }) as typeof api.get

    try {
      await getDepartmentMembers(7, 2)
      await getDepartmentBudgets(7, {
        tenant_id: 2,
        include_descendants: false,
        sort_by: 'usage_ratio',
        sort_order: 'desc',
      })
      await getDepartmentBudgetDetail(7, 11, 2)
      await getQuotaAllocations(11, 2, 7)

      assert.deepEqual(calls, [
        {
          url: '/api/enterprise/departments/7/members',
          params: { tenant_id: 2 },
        },
        {
          url: '/api/enterprise/departments/7/budgets',
          params: {
            tenant_id: 2,
            include_descendants: false,
            sort_by: 'usage_ratio',
            sort_order: 'desc',
          },
        },
        {
          url: '/api/enterprise/departments/7/budgets/11',
          params: { tenant_id: 2 },
        },
        {
          url: '/api/enterprise/quota-allocations',
          params: {
            department_budget_id: 11,
            department_id: 7,
            tenant_id: 2,
          },
        },
      ])
    } finally {
      api.get = originalGet
    }
  })

  test('quota allocation creation posts selected department and member context without manual ids', async () => {
    const originalPost = api.post
    const calls: Array<{ url: string; payload?: unknown }> = []

    api.post = (async (url: string, payload?: unknown) => {
      calls.push({ url, payload })
      return {
        data: {
          success: true,
          message: '',
          data: { item: quotaAllocation({ department_id: 7 }) },
        },
      }
    }) as typeof api.post

    try {
      await createQuotaAllocation({
        tenant_id: 2,
        department_id: 7,
        department_budget_id: 11,
        target_user_id: 2001,
        committed_quota: 300,
        reason: 'workspace allocation',
      })

      assert.deepEqual(calls, [
        {
          url: '/api/enterprise/quota-allocations',
          payload: {
            tenant_id: 2,
            department_id: 7,
            department_budget_id: 11,
            target_user_id: 2001,
            committed_quota: 300,
            reason: 'workspace allocation',
          },
        },
      ])
    } finally {
      api.post = originalPost
    }
  })

  test('username rename posts the current department member context and tenant payload', async () => {
    const originalPut = api.put
    const calls: Array<{ url: string; payload?: unknown }> = []

    api.put = (async (url: string, payload?: unknown) => {
      calls.push({ url, payload })
      return {
        data: {
          success: true,
          message: '',
          data: departmentMember({
            department_id: 7,
            user_id: 2001,
            username: 'alice_ops',
          }),
        },
      }
    }) as typeof api.put

    try {
      const result = await renameDepartmentMember(7, 2001, {
        tenant_id: 2,
        new_username: 'alice_ops',
      })

      assert.equal(result.success, true)
      assert.equal(result.data?.username, 'alice_ops')
      assert.deepEqual(calls, [
        {
          url: '/api/enterprise/departments/7/members/2001/username',
          payload: {
            tenant_id: 2,
            new_username: 'alice_ops',
          },
        },
      ])
    } finally {
      api.put = originalPut
    }
  })

  test('api functions surface business failures for scoped department member loading', async () => {
    const originalGet = api.get

    api.get = (async () => ({
      data: {
        success: false,
        message: 'enterprise.organization.department_not_found',
      },
    })) as typeof api.get

    try {
      const result = await getDepartmentMembers(404, 2)

      assert.equal(result.success, false)
      assert.equal(
        result.message,
        'enterprise.organization.department_not_found'
      )
    } finally {
      api.get = originalGet
    }
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
    const knownActions = [
      ['enterprise.organization.membership.replace', 'Membership replaced'],
      ['enterprise.organization.membership.add', 'Membership added'],
      ['enterprise.organization.membership.disable', 'Membership disabled'],
      ['enterprise.organization.membership.restore', 'Membership restored'],
      ['enterprise.organization.membership.rename', 'Membership renamed'],
      [
        'enterprise.organization.department_admin.grant',
        'Department admin granted',
      ],
      [
        'enterprise.organization.department_admin.revoke',
        'Department admin revoked',
      ],
      [
        'enterprise.organization.department_owner.manual_grant',
        'Department owner granted',
      ],
      [
        'enterprise.organization.department_owner.manual_grant.revoke',
        'Department owner grant revoked',
      ],
      [
        'enterprise.organization.department_owner.manual_deny',
        'Department owner denied',
      ],
      [
        'enterprise.organization.department_owner.manual_deny.revoke',
        'Department owner denial revoked',
      ],
      [
        'enterprise.organization.department_owner.dingtalk_sync.update',
        'DingTalk owner sync updated',
      ],
      [
        'enterprise.organization.department_owner.resolution.denied',
        'Department owner resolution denied',
      ],
      ['enterprise.dingtalk.config.set', 'DingTalk configuration updated'],
      ['enterprise.dingtalk.connectivity.test', 'DingTalk connectivity tested'],
      ['enterprise.dingtalk.sync.start', 'DingTalk sync started'],
      [
        'enterprise.dingtalk.sync_conflict.bind_candidate',
        'DingTalk sync conflict candidate bound',
      ],
      ['enterprise.usage.report.set', 'Usage report configured'],
      [
        'enterprise.organization.department_budget.create',
        'Department budget created',
      ],
      [
        'enterprise.organization.department_budget.reject',
        'Department budget rejected',
      ],
      [
        'enterprise.organization.budget_delegation.create',
        'Budget delegation created',
      ],
      [
        'enterprise.organization.budget_delegation.supersede',
        'Budget delegation adjusted',
      ],
      [
        'enterprise.organization.budget_delegation.revoke',
        'Budget delegation revoked',
      ],
      [
        'enterprise.organization.budget_delegation.reject',
        'Budget delegation rejected',
      ],
      [
        'enterprise.organization.quota_request.submit',
        'Quota request submitted',
      ],
      [
        'enterprise.organization.quota_request.approve',
        'Quota request approved',
      ],
      [
        'enterprise.organization.quota_request.reject',
        'Quota request rejected',
      ],
      ['enterprise.organization.quota_allocation.create', 'Allocation created'],
      [
        'enterprise.organization.quota_allocation.reclaim',
        'Allocation reclaimed',
      ],
      [
        'enterprise.organization.quota_allocation.cancel',
        'Allocation cancelled',
      ],
      ['enterprise.organization.quota_allocation.revoke', 'Allocation revoked'],
      ['enterprise.alert.rule.save', 'Alert rule saved'],
      ['enterprise.alert.rule.delete', 'Alert rule deleted'],
      ['enterprise.alert.delivery.resend', 'Alert delivery resent'],
    ] as const
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <GovernanceActivityCard
          loading={false}
          timelineItems={knownActions.map(([actionType], index) =>
            governanceTimelineItem({
              source_id: index + 1,
              trace_id: `governance:${index + 1}`,
              action_type: actionType,
              actor_name: index === 1 ? 'Owner Alice' : `Actor ${index + 1}`,
              quota_delta: 80,
              status: 'fulfilled',
            })
          )}
          notificationItems={[
            governanceNotificationItem({
              id: 7,
              trace_id: 'governance:2',
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
      'governance:2',
      'Owner Alice',
      'Final failed',
      'webhook request failed',
      'Resend',
      '80 quota',
      'Approx. $0.0002',
      ...knownActions.map(([, label]) => label),
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
    assert.doesNotMatch(html, /enterprise\.organization\./)
  })

  test('renders governance notification action summary for delivery-only rows', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <GovernanceActivityCard
          loading={false}
          timelineItems={[]}
          notificationItems={[
            governanceNotificationItem({
              id: 9,
              trace_id: 'alert_delivery:9',
              action_type: 'enterprise.alert.delivery.resend',
              status: 'sent',
            }),
          ]}
          resendPendingId={null}
          onResend={() => undefined}
        />
      </I18nextProvider>
    )

    assert.match(html, /Alert delivery resent/)
    assert.match(html, /Sent/)
    assert.doesNotMatch(html, /enterprise\.alert\.delivery\.resend/)
  })

  test('renders fulfilled governance action separately from unconfigured delivery', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <GovernanceActivityCard
          loading={false}
          timelineItems={[
            governanceTimelineItem({
              source_id: 42,
              trace_id: 'quota_request:42',
              action_type: 'enterprise.organization.quota_request.approve',
              actor_name: 'Owner Alice',
              quota_delta: 150,
              status: 'fulfilled',
            }),
          ]}
          notificationItems={[
            governanceNotificationItem({
              id: 7,
              trace_id: 'quota_request:42',
              status: 'unconfigured',
              attempt_count: 0,
              error_reason: 'governance notification channel is not configured',
            }),
          ]}
          resendPendingId={null}
          onResend={() => undefined}
        />
      </I18nextProvider>
    )

    assert.match(html, /Quota request approved/)
    assert.match(html, /Fulfilled/)
    assert.match(html, /Not configured/)
    assert.match(html, /governance notification channel is not configured/)
    assert.match(html, /Attempt 0\/4/)
    assert.doesNotMatch(html, /Final failed/)
    assert.doesNotMatch(html, /Resend/)
  })

  test('renders governance activity with zh locale translations and safe fallbacks', async () => {
    const previousLanguage = i18n.language
    await i18n.changeLanguage('zhCN')
    try {
      const html = renderToStaticMarkup(
        <I18nextProvider i18n={i18n}>
          <GovernanceActivityCard
            loading={false}
            timelineItems={[
              governanceTimelineItem({
                source_id: 1,
                trace_id: 'quota_request:42',
                action_type: 'enterprise.organization.quota_request.approve',
                actor_name: 'Owner Alice',
                quota_delta: 80,
                status: 'fulfilled',
              }),
              governanceTimelineItem({
                source_id: 2,
                trace_id: 'unknown:99',
                action_type:
                  'enterprise.organization.future_action.full.internal.key',
                status: 'mystery_status',
              }),
            ]}
            notificationItems={[
              governanceNotificationItem({
                id: 7,
                trace_id: 'quota_request:42',
                status: 'final_failed',
                error_reason: 'webhook request failed',
              }),
              governanceNotificationItem({
                id: 8,
                trace_id: 'unknown:99',
                status: 'future_status',
              }),
            ]}
            resendPendingId={null}
            onResend={() => undefined}
          />
        </I18nextProvider>
      )

      for (const expected of [
        '治理时间线与通知投递',
        '动作',
        '通知状态',
        '额度申请已批准',
        '最终失败',
        '重新发送',
        '未知治理动作',
        '未知投递状态',
        'quota_request:42',
        'unknown:99',
      ]) {
        assert.match(html, new RegExp(escapeRegExp(expected)))
      }
      assert.doesNotMatch(
        html,
        /enterprise\.organization\.future_action\.full\.internal\.key/
      )
      assert.doesNotMatch(html, />future_status</)
    } finally {
      await i18n.changeLanguage(previousLanguage)
    }
  })

  test('governance locale keys exist for every supported language', () => {
    const requiredKeys = [
      'Governance Timeline and Notification Delivery',
      'Action',
      'Notification Status',
      'Final failed',
      'Not configured',
      'Resend',
      'Attempt {{count}}/{{max}}',
      'Membership replaced',
      'Membership added',
      'Membership disabled',
      'Membership restored',
      'Membership renamed',
      'Department admin granted',
      'Department admin revoked',
      'Department owner granted',
      'Department owner grant revoked',
      'Department owner denied',
      'Department owner denial revoked',
      'DingTalk owner sync updated',
      'Department owner resolution denied',
      'DingTalk configuration updated',
      'DingTalk connectivity tested',
      'DingTalk sync started',
      'DingTalk sync conflict candidate bound',
      'Usage report configured',
      'Alert rule saved',
      'Alert rule deleted',
      'Alert delivery resent',
      'Quota request submitted',
      'Quota request approved',
      'Quota request rejected',
      'Submitted',
      'Approved',
      'Rejected',
      'Fulfilled',
      'Unknown status',
      'Unknown budget type',
      'Allocation created',
      'Allocation reclaimed',
      'Allocation cancelled',
      'Allocation revoked',
      'Budget delegation created',
      'Budget delegation adjusted',
      'Budget delegation revoked',
      'Budget delegation rejected',
      'Department budget created',
      'Department budget rejected',
      'Unknown governance action',
      'Unknown delivery status',
      'Employee Quota Requests',
      'Target Budget Pool',
      'Allocation #{{id}}',
      'Selected request scope',
      'No budget pool selected',
      'Budget #{{budgetId}}',
      '{{department}} · {{budget}}',
      'Remaining {{remaining}}',
      'Trace ID',
    ] as const

    for (const [language, resource] of Object.entries(resources)) {
      for (const key of requiredKeys) {
        assert.notEqual(
          resource.translation[key],
          undefined,
          `${language} missing ${key}`
        )
      }
    }
  })

  test('enterprise wallet expiry locale keys exist for every supported language', () => {
    const requiredKeys = [
      'One-time quota',
      'Never expires',
      'No expiry set',
      'Unknown cycle type',
    ] as const

    for (const [language, resource] of Object.entries(resources)) {
      for (const key of requiredKeys) {
        assert.notEqual(
          resource.translation[key],
          undefined,
          `${language} missing ${key}`
        )
      }
    }
  })

  test('maps quota request, allocation, wallet status, and budget type without raw fallback', () => {
    for (const [status, expected] of [
      ['submitted', 'Submitted'],
      ['approved', 'Approved'],
      ['rejected', 'Rejected'],
      ['fulfilled', 'Fulfilled'],
      ['active', 'Active'],
      ['paused', 'Paused'],
      ['revoked', 'Revoked'],
      ['expired', 'Expired'],
      ['superseded', 'Superseded'],
      ['closed', 'Closed'],
      ['cancelled', 'Cancelled'],
      ['future_status', 'Unknown status'],
      ['', 'Unknown status'],
    ] as const) {
      assert.equal(enterpriseBudgetStatusLabel(status, i18n.t), expected)
    }

    assert.equal(formatBudgetType('balance', i18n.t), 'Balance Budget')
    assert.equal(
      formatBudgetType('subscription', i18n.t),
      'Subscription Budget'
    )
    assert.equal(
      formatBudgetType('future_budget', i18n.t),
      'Unknown budget type'
    )
  })

  test('formats enterprise quota amounts and parses amount view through wallet helpers', () => {
    const display = formatEnterpriseQuotaAmount(1_000_000, i18n.t)

    assert.equal(display.rawQuota, 1_000_000)
    assert.equal(display.quotaLabel, '1,000,000 quota')
    assert.equal(display.amount, 2)
    assert.equal(display.amountLabel, '$2')
    assert.equal(display.auxiliaryLabel, 'Approx. $2')

    assert.equal(parseEnterpriseQuotaInput('2', 'amount'), 1_000_000)
    assert.equal(parseEnterpriseQuotaInput('0.5', 'amount'), 250_000)
    assert.equal(parseEnterpriseQuotaInput('250000', 'quota'), 250_000)
    assert.equal(parseEnterpriseQuotaInput('1.5', 'quota'), null)
    assert.equal(parseEnterpriseQuotaInput('abc', 'amount'), null)

    assert.deepEqual(
      convertEnterpriseQuotaInputMode({
        value: '1000000',
        from: 'quota',
        to: 'amount',
      }),
      { value: '2', quota: 1_000_000 }
    )
    assert.deepEqual(
      convertEnterpriseQuotaInputMode({
        value: '2',
        from: 'amount',
        to: 'quota',
      }),
      { value: '1000000', quota: 1_000_000 }
    )
  })

  test('quota amount display preserves negative governance deltas', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <QuotaAmountDisplay quota={-500_000} />
      </I18nextProvider>
    )

    assert.match(html, /-500,000 quota/)
    assert.match(html, /Approx\. -\$1/)
    assert.doesNotMatch(html, />0 quota</)
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
      '300 quota',
      'Approx. $0.0006',
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

  test('delegation schema rejects missing subordinate target and quota, then accepts valid input', () => {
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
      '640 quota',
      'Approx. $0.0013',
      'Total Quota',
      '800 quota',
      'One-time quota',
      'Never expires',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }

    assert.doesNotMatch(html, /No Reset/)
    assert.doesNotMatch(html, /1970|1969|Invalid Date/)
    assert.doesNotMatch(html, />Weekly</)
    assert.doesNotMatch(html, />Custom \\(seconds\\)</)
  })

  test('renders non-positive budget expiry and unknown cycles with semantic fallback', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetOverviewCard
          item={departmentBudget({
            type: 'balance',
            cycle_type: 'future_cycle',
            expires_at: -1,
          })}
          selectedBudget={departmentBudget({
            type: 'balance',
            cycle_type: 'future_cycle',
            expires_at: -1,
          })}
          thresholds={{ warning: 80, critical: 95 }}
        />
      </I18nextProvider>
    )

    assert.match(html, /Unknown cycle type/)
    assert.match(html, /Never expires/)
    assert.doesNotMatch(html, />future_cycle</)
    assert.doesNotMatch(html, /No Reset/)
    assert.doesNotMatch(html, /1970|1969|Invalid Date/)
  })

  test('renders budget pool list with selectable threshold states and usage metrics', () => {
    const emptyHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetListCard
          loading={false}
          selectedBudgetId={null}
          sortBy='usage_ratio'
          sortOrder='desc'
          onSelectBudget={() => undefined}
          onSortByChange={() => undefined}
          onSortOrderChange={() => undefined}
          items={[]}
        />
      </I18nextProvider>
    )
    assert.match(
      emptyHtml,
      /Budget pools created by enterprise administrators will appear here/
    )
    assert.doesNotMatch(emptyHtml, /Create the first pool for this department/)

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
      '300 quota',
      '180 quota',
      'Approx. $0.0006',
      'Approx. $0.0004',
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

  test('renders one-time derived wallet expiry without raw cycle codes or epoch dates', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetDetailTable
          loading={false}
          budget={departmentBudget({
            id: 17,
            type: 'balance',
            status: 'active',
            cycle_type: 'never',
            expires_at: 0,
          })}
          wallets={[
            walletDetail({
              allocation_id: 32,
              wallet_id: 42,
              wallet_status: 'active',
              target_user_id: 2002,
              target_username: 'bob',
              target_display_name: 'Bob',
              cycle_type: 'never',
              next_reset_time: 0,
              expires_at: -1,
              source_parent_budget_id: 17,
              source_parent_budget_type: 'balance',
              processed_at: 1700000600,
            }),
          ]}
        />
      </I18nextProvider>
    )

    assert.match(html, /One-time quota/)
    assert.match(html, /Never expires/)
    assert.doesNotMatch(html, /No Reset/)
    assert.doesNotMatch(html, />never</)
    assert.doesNotMatch(html, /1970|1969|Invalid Date/)
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

  test('renders budget delegation empty state with subordinate allocation guidance', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <BudgetDelegationTable items={[]} loading={false} />
      </I18nextProvider>
    )

    assert.match(html, /No budget delegations yet/)
    assert.match(
      html,
      /Choose a subordinate department budget pool to create the first allocation in this governance chain\./
    )
    assert.doesNotMatch(html, /Descendant/)
    assert.doesNotMatch(html, /descendant department/)
    assert.doesNotMatch(html, /后代部门/)
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
      '450 quota',
      'Approx. $0.0009',
      'Active',
      'aria-label="Delegation Quota"',
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
      '300 quota',
      'Approx. $0.0006',
      '301',
      'Active',
      'aria-label="Allocation Quota"',
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
      '175 quota',
      'Approx. $0.0004',
      'Close old allocation and create a new one',
      'Historical allocation',
      'Reclaim allocation',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('renders quota request rows with localized budget identity and request statuses', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <QuotaRequestTable
          loading={false}
          items={[
            quotaRequest({
              id: 1,
              status: 'submitted',
              department_budget_id: 11,
            }),
            quotaRequest({
              id: 2,
              approved_quota: 80,
              allocation_id: 21,
              status: 'fulfilled',
              department_budget_id: 12,
            }),
            quotaRequest({
              id: 3,
              status: 'mystery_status',
              department_budget_id: 13,
            }),
          ]}
          decisionDrafts={{}}
          onDecisionDraftChange={() => undefined}
          onApprove={() => undefined}
          onReject={() => undefined}
          pendingRequestId={null}
          canGovern={true}
        />
      </I18nextProvider>
    )

    for (const expected of [
      'Requester',
      'Target Budget Pool',
      'Engineering',
      'Budget #11',
      'Budget #12',
      'Budget #13',
      '120 quota',
      '80 quota',
      'Approx. $0.0002',
      'Allocation #21',
      'Submitted',
      'Fulfilled',
      'Unknown status',
      'aria-label="Approved Quota"',
      'Approve',
      'Reject',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
    assert.doesNotMatch(html, />submitted</)
    assert.doesNotMatch(html, />fulfilled</)
    assert.doesNotMatch(html, />mystery_status</)
    assert.doesNotMatch(html, />#11</)
  })

  test('renders quota request governance surfaces with zh locale without internal status fallback', async () => {
    const previousLanguage = i18n.language
    await i18n.changeLanguage('zhCN')
    try {
      const html = renderToStaticMarkup(
        <I18nextProvider i18n={i18n}>
          <QuotaRequestTable
            loading={false}
            items={[
              quotaRequest({
                id: 1,
                status: 'submitted',
                department_budget_id: 11,
              }),
              quotaRequest({
                id: 2,
                status: 'fulfilled',
                allocation_id: 21,
                department_budget_id: 12,
              }),
              quotaRequest({
                id: 3,
                status: 'mystery_status',
                department_budget_id: 13,
              }),
            ]}
            decisionDrafts={{}}
            onDecisionDraftChange={() => undefined}
            onApprove={() => undefined}
            onReject={() => undefined}
            pendingRequestId={null}
            canGovern={true}
          />
        </I18nextProvider>
      )

      for (const expected of [
        '申请人',
        '目标预算池',
        '预算池 #11',
        '预算池 #12',
        '已提交',
        '已完成',
        '未知状态',
        '分配 #21',
        '批准',
        '拒绝',
      ]) {
        assert.match(html, new RegExp(escapeRegExp(expected)))
      }
      assert.doesNotMatch(html, />submitted</)
      assert.doesNotMatch(html, />fulfilled</)
      assert.doesNotMatch(html, />mystery_status</)
      assert.doesNotMatch(html, />#11</)
    } finally {
      await i18n.changeLanguage(previousLanguage)
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
      'Keep history logs and risk events on their original username snapshots. Current governance views switch to the updated username after refresh.',
      'Readable Username',
      'Update Username',
      'Department',
      'Membership Status',
    ]) {
      assert.match(selectedHtml, new RegExp(escapeRegExp(expected)))
    }
  })

  test('wallet allocation form explains missing current member and budget context instead of exposing id inputs', () => {
    const html = renderDepartmentBudgetPanel({
      departmentId: 7,
      departmentName: 'Security',
      selectedMember: null,
    })

    for (const expected of [
      'Current Member Wallet Allocation',
      'No member selected',
      'Select a member from the current department list before creating a wallet allocation.',
      'Select a budget pool',
      'Choose a budget pool in the current department before creating a member wallet allocation.',
      'Create wallet allocation',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }

    assert.doesNotMatch(html, /Target User ID/)
    assert.doesNotMatch(html, /Membership Lookup/)
  })

  test('allocation workspace places the member selector only in the member wallet tab', () => {
    const html = renderDepartmentBudgetPanel({
      departmentId: 7,
      departmentName: 'Security',
      selectedMember: null,
      mode: 'allocations',
    })

    assert.match(html, /role="tablist"/)
    assert.match(html, /role="tab"/)
    assert.match(html, /Allocate Budget To Subordinate Department/)
    assert.match(html, /Current Member Wallet Allocation/)
    assert.ok(
      html.indexOf('Allocate Budget To Subordinate Department') <
        html.indexOf('Current Member Wallet Allocation')
    )
    assert.ok(
      html.indexOf('Allocate budget to subordinate department') <
        html.indexOf('Current Member Context')
    )
    assert.ok(
      html.indexOf('Current Member Context') <
        html.indexOf('Department Members')
    )
    assert.equal(html.split('Current Member Context').length - 1, 1)
    assert.ok(
      html.indexOf('Department Members') < html.indexOf('Allocation Quota')
    )
  })

  test('wallet allocation form inherits the selected department member context', () => {
    const html = renderDepartmentBudgetPanel({
      departmentId: 7,
      departmentName: 'Security',
      selectedMember: departmentMember({
        department_id: 7,
        user_id: 2001,
        username: 'alice',
        display_name: 'Alice',
      }),
    })

    for (const expected of [
      'Current Member Wallet Allocation',
      'Alice',
      'Selected from Security',
      'Allocation Quota',
      'Optional allocation note',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }

    assert.doesNotMatch(html, /Target User ID/)
    assert.doesNotMatch(html, /Department ID/)
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

function renderDepartmentBudgetPanel({
  departmentId,
  departmentName,
  selectedMember,
  mode = 'all',
}: {
  departmentId: number
  departmentName: string
  selectedMember: DepartmentMemberItem | null
  mode?: 'budgets' | 'allocations' | 'requests' | 'governance' | 'all'
}) {
  return renderToStaticMarkup(
    <QueryClientProvider client={new QueryClient()}>
      <I18nextProvider i18n={i18n}>
        <DepartmentBudgetPanel
          departmentId={departmentId}
          tenantId={0}
          departmentName={departmentName}
          selectedBudgetId={null}
          onSelectedBudgetIdChange={() => undefined}
          selectedMember={selectedMember}
          mode={mode}
          selectedMemberUserId={selectedMember?.user_id ?? null}
          onSelectedMemberChange={() => undefined}
          onMembersChange={() => undefined}
        />
      </I18nextProvider>
    </QueryClientProvider>
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
    target: overrides.target ?? {
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
  return value.replaceAll(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
