import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { NavGroup } from '@/components/layout/types'
import { filterRootNavGroupsByRole } from './use-sidebar-view'

const rootNavGroups: NavGroup[] = [
  {
    id: 'general',
    title: 'General',
    items: [
      {
        title: 'Dashboard',
        url: '/dashboard/models',
      },
    ],
  },
  {
    id: 'admin',
    title: 'Admin',
    items: [
      {
        title: 'Channels',
        url: '/channels',
      },
      {
        title: 'Enterprise Organization',
        url: '/enterprise-organization',
      },
      {
        title: 'Department Usage Overview',
        url: '/enterprise-usage',
      },
      {
        title: 'Enterprise Alerts',
        url: '/enterprise-alerts',
      },
    ],
  },
]

describe('sidebar role filtering', () => {
  test('exposes only enterprise organization to department governors', () => {
    const groups = filterRootNavGroupsByRole(rootNavGroups, {
      isAdmin: false,
      canAccessEnterpriseOrganization: true,
    })

    const adminItems = groups.find((group) => group.id === 'admin')?.items ?? []
    assert.deepEqual(
      adminItems.map((item) => ('url' in item ? item.url : '')),
      ['/enterprise-organization']
    )
  })

  test('keeps admin navigation hidden for ordinary users', () => {
    const groups = filterRootNavGroupsByRole(rootNavGroups, {
      isAdmin: false,
      canAccessEnterpriseOrganization: false,
    })

    assert.equal(
      groups.some((group) => group.id === 'admin'),
      false
    )
  })

  test('keeps full admin navigation for admins', () => {
    const groups = filterRootNavGroupsByRole(rootNavGroups, {
      isAdmin: true,
      canAccessEnterpriseOrganization: false,
    })

    const adminItems = groups.find((group) => group.id === 'admin')?.items ?? []
    assert.deepEqual(
      adminItems.map((item) => ('url' in item ? item.url : '')),
      [
        '/channels',
        '/enterprise-organization',
        '/enterprise-usage',
        '/enterprise-alerts',
      ]
    )
  })
})
