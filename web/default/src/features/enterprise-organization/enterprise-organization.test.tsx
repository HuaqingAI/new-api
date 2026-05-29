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
  __testRenderApiMessage,
  EnterpriseOrganizationContent,
  DepartmentBudgetDetailTable,
  DepartmentBudgetListCard,
  DepartmentBudgetOverviewCard,
  DepartmentBudgetStatusCard,
  QuotaAllocationTable,
  createBudgetSchema,
  createAllocationSchema,
} from './index'
import type {
  ApiResponse,
  DepartmentBudgetDetailResponse,
  DepartmentBudgetItem,
  DepartmentTreeNode,
  EnterpriseBudgetErrorData,
  QuotaAllocationItem,
} from './types'

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
    const html = renderEnterpriseOrganizationContent([
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
    ])

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
      'User ID #2001',
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
      '2001',
      '300',
      '301',
      'Active',
      'Revoke allocation',
    ]) {
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
})

function renderEnterpriseOrganizationContent(
  departments: DepartmentTreeNode[]
) {
  return renderToStaticMarkup(
    <RouterContextProvider router={testRouter}>
      <I18nextProvider i18n={i18n}>
        <EnterpriseOrganizationContent
          isLoading={false}
          departments={departments}
        />
      </I18nextProvider>
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

function quotaAllocation(
  overrides: Partial<QuotaAllocationItem> = {}
): QuotaAllocationItem {
  return {
    id: overrides.id ?? 1,
    tenant_id: overrides.tenant_id ?? 0,
    department_budget_id: overrides.department_budget_id ?? 1,
    department_id: overrides.department_id ?? 2,
    target_user_id: overrides.target_user_id ?? 2001,
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
    processed_at: overrides.processed_at ?? 0,
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
    source_parent_budget_type: overrides.source_parent_budget_type ?? 'subscription',
    source_parent_budget_status: overrides.source_parent_budget_status ?? 'active',
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
