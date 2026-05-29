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
import { EnterpriseOrganizationContent } from './index'
import type { DepartmentTreeNode } from './types'

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

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
