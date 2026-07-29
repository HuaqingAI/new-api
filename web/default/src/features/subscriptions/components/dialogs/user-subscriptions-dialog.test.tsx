import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import i18n from '@/i18n/config'

import {
  renderSubscriptionSourceLabel,
  renderSubscriptionSourceTypeLabel,
} from './user-subscriptions-dialog'

describe('User subscription dialog enterprise allocation labels', () => {
  test('renders enterprise allocation title when no plan title is available', () => {
    const title = renderSubscriptionSourceLabel(
      {
        id: 1,
        user_id: 1001,
        plan_id: 0,
        status: 'active',
        source: 'enterprise_allocation',
        source_type: 'enterprise_allocation',
        source_allocation_id: 10,
        sort_order: -100,
        is_primary: false,
        start_time: 1700000000,
        end_time: 0,
        amount_total: 300,
        amount_used: 0,
        next_reset_time: 0,
      },
      i18n.t.bind(i18n)
    )

    assert.equal(title, 'Enterprise Allocation Wallet')
  })

  test('prefers plan title for ordinary subscriptions', () => {
    const title = renderSubscriptionSourceLabel(
      {
        id: 2,
        user_id: 1001,
        plan_id: 8,
        status: 'active',
        source: 'admin',
        source_type: 'admin',
        source_allocation_id: 0,
        sort_order: 100,
        is_primary: true,
        start_time: 1700000000,
        end_time: 1800000000,
        amount_total: 1000,
        amount_used: 200,
        next_reset_time: 0,
      },
      i18n.t.bind(i18n),
      'VIP Plan'
    )

    assert.equal(title, 'VIP Plan')
  })

  test('treats source-only enterprise subscriptions as enterprise wallets', () => {
    const title = renderSubscriptionSourceLabel(
      {
        id: 21,
        user_id: 1001,
        plan_id: 0,
        status: 'active',
        source: 'enterprise_allocation',
        source_type: '',
        source_allocation_id: 10,
        sort_order: -100,
        is_primary: false,
        start_time: 1700000000,
        end_time: 0,
        amount_total: 300,
        amount_used: 0,
        next_reset_time: 0,
      },
      i18n.t.bind(i18n)
    )

    assert.equal(title, 'Enterprise Allocation Wallet')
  })

  test('treats legacy enterprise source types as enterprise wallets', () => {
    const title = renderSubscriptionSourceLabel(
      {
        id: 22,
        user_id: 1001,
        plan_id: 0,
        status: 'active',
        source: 'enterprise',
        source_type: 'enterprise',
        source_allocation_id: 10,
        sort_order: -100,
        is_primary: false,
        start_time: 1700000000,
        end_time: 0,
        amount_total: 300,
        amount_used: 0,
        next_reset_time: 0,
      },
      i18n.t.bind(i18n)
    )

    assert.equal(title, 'Enterprise Allocation Wallet')
  })

  test('maps source detail labels without leaking raw internal codes', async () => {
    const previousLanguage = i18n.language
    await i18n.changeLanguage('zhCN')
    try {
      assert.equal(
        renderSubscriptionSourceTypeLabel(
          {
            id: 3,
            user_id: 1001,
            plan_id: 0,
            status: 'active',
            source: 'enterprise_allocation',
            source_type: '',
            source_allocation_id: 10,
            sort_order: -100,
            is_primary: false,
            start_time: 1700000000,
            end_time: 0,
            amount_total: 300,
            amount_used: 0,
            next_reset_time: 0,
          },
          i18n.t.bind(i18n)
        ),
        '企业分配'
      )
      assert.equal(
        renderSubscriptionSourceTypeLabel(
          {
            id: 31,
            user_id: 1001,
            plan_id: 0,
            status: 'active',
            source: 'enterprise',
            source_type: 'enterprise',
            source_allocation_id: 10,
            sort_order: -100,
            is_primary: false,
            start_time: 1700000000,
            end_time: 0,
            amount_total: 300,
            amount_used: 0,
            next_reset_time: 0,
          },
          i18n.t.bind(i18n)
        ),
        '企业分配'
      )
      assert.equal(
        renderSubscriptionSourceTypeLabel(
          {
            id: 4,
            user_id: 1001,
            plan_id: 0,
            status: 'active',
            source: 'future_source',
            source_type: '',
            source_allocation_id: 0,
            sort_order: 100,
            is_primary: true,
            start_time: 1700000000,
            end_time: 0,
            amount_total: 100,
            amount_used: 0,
            next_reset_time: 0,
          },
          i18n.t.bind(i18n)
        ),
        '未知来源'
      )
    } finally {
      await i18n.changeLanguage(previousLanguage)
    }
  })
})
