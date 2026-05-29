import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'
import {
  RouterContextProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router'
import i18n from '@/i18n/config'
import {
  EnterpriseOrganizationContent,
  DepartmentBudgetStatusCard,
  createBudgetSchema,
} from './index'
import type { DepartmentBudgetItem, DepartmentTreeNode } from './types'

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

    assert.match(emptyHtml, /Current Budget Pool/)
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
    assert.match(issues, /Subscription budget cycle quota must be greater than 0/)
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
})

function renderEnterpriseOrganizationContent(departments: DepartmentTreeNode[]) {
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
  overrides: Partial<DepartmentTreeNode> & Pick<DepartmentTreeNode, 'id' | 'name'>
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
    cycle_quota: overrides.cycle_quota ?? 300,
    cycle_type: overrides.cycle_type ?? 'weekly',
    cycle_started_at: overrides.cycle_started_at ?? 1700000000,
    custom_seconds: overrides.custom_seconds ?? 0,
    expires_at: overrides.expires_at ?? 0,
    parent_status: overrides.parent_status ?? '',
    created_at: overrides.created_at ?? 1700000000,
    updated_at: overrides.updated_at ?? 1700000001,
  }
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
