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
      {
        title: 'DingTalk Integration',
        url: '/enterprise-dingtalk',
        requiredRole: 100,
      },
      {
        title: 'System Info',
        url: '/system-info',
        requiredRole: 20,
      },
      {
        title: 'System Settings',
        url: '/system-settings/site',
        requiredRole: 100,
      },
    ],
  },
]

describe('sidebar role filtering', () => {
  test('exposes only enterprise organization to department governors', () => {
    const groups = filterRootNavGroupsByRole(rootNavGroups, {
      role: 10,
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
      role: 1,
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
      role: 10,
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

  test('hides requiredRole items for admins below the threshold', () => {
    const groups = filterRootNavGroupsByRole(rootNavGroups, {
      role: 10,
      isAdmin: true,
      canAccessEnterpriseOrganization: false,
    })

    const adminItems = groups.find((group) => group.id === 'admin')?.items ?? []
    assert.equal(
      adminItems.some((item) => 'url' in item && item.url === '/system-info'),
      false
    )
    assert.equal(
      adminItems.some(
        (item) => 'url' in item && item.url === '/enterprise-dingtalk'
      ),
      false
    )
    assert.equal(
      adminItems.some(
        (item) => 'url' in item && item.url === '/system-settings/site'
      ),
      false
    )
  })

  test('shows requiredRole items for sufficiently privileged admins', () => {
    const groups = filterRootNavGroupsByRole(rootNavGroups, {
      role: 100,
      isAdmin: true,
      canAccessEnterpriseOrganization: false,
    })

    const adminItems = groups.find((group) => group.id === 'admin')?.items ?? []
    assert.equal(
      adminItems.some((item) => 'url' in item && item.url === '/system-info'),
      true
    )
    assert.equal(
      adminItems.some(
        (item) => 'url' in item && item.url === '/enterprise-dingtalk'
      ),
      true
    )
    assert.equal(
      adminItems.some(
        (item) => 'url' in item && item.url === '/system-settings/site'
      ),
      true
    )
  })
})
