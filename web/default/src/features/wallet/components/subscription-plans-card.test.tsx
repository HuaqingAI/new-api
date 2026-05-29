import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import i18n from '@/i18n/config'
import {
  getManagedSubscriptionNote,
  getSubscriptionCardTitle,
} from './subscription-plans-card'

describe('Subscription plans card enterprise wallet helpers', () => {
  test('renders enterprise wallet title when plan title is unavailable', () => {
    const title = getSubscriptionCardTitle(
      {
        id: 12,
        user_id: 9001,
        plan_id: 0,
        status: 'active',
        source: 'enterprise_allocation',
        source_type: 'enterprise_allocation',
        source_allocation_id: 88,
        sort_order: -100,
        is_primary: false,
        start_time: 1700000000,
        end_time: 1800000000,
        amount_total: 300,
        amount_used: 10,
        next_reset_time: 0,
      },
      i18n.t.bind(i18n)
    )

    assert.equal(title, 'Enterprise Allocation Wallet · Subscription #12')
  })

  test('returns managed note only for department wallets', () => {
    const managed = getManagedSubscriptionNote(
      {
        id: 13,
        user_id: 9001,
        plan_id: 0,
        status: 'active',
        source: 'enterprise_allocation',
        source_type: 'enterprise_allocation',
        source_allocation_id: 89,
        sort_order: -100,
        is_primary: false,
        start_time: 1700000000,
        end_time: 1800000000,
        amount_total: 500,
        amount_used: 0,
        next_reset_time: 0,
      },
      i18n.t.bind(i18n)
    )
    const ordinary = getManagedSubscriptionNote(
      {
        id: 14,
        user_id: 9001,
        plan_id: 7,
        status: 'active',
        source: 'admin',
        source_type: 'admin',
        source_allocation_id: 0,
        sort_order: 100,
        is_primary: true,
        start_time: 1700000000,
        end_time: 1800000000,
        amount_total: 500,
        amount_used: 0,
        next_reset_time: 0,
      },
      i18n.t.bind(i18n)
    )

    assert.equal(managed, 'Managed by department · Cannot be deleted by user')
    assert.equal(ordinary, null)
  })
})
