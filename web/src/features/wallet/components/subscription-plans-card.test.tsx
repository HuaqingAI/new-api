import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import i18n, { resources } from '@/i18n/config'

import {
  getBillingPreferenceLabel,
  getManagedSubscriptionNote,
  getSubscriptionExpiryDisplay,
  getSubscriptionRemainingDays,
  getSubscriptionStatusDisplay,
  getSubscriptionSourceLabel,
  getSubscriptionCardTitle,
  isActiveSubscriptionForBilling,
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

  test('treats source-only enterprise wallets as managed by department', () => {
    const managed = getManagedSubscriptionNote(
      {
        id: 130,
        user_id: 9001,
        plan_id: 0,
        status: 'active',
        source: 'enterprise_allocation',
        source_type: '',
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

    const title = getSubscriptionCardTitle(
      {
        id: 130,
        user_id: 9001,
        plan_id: 0,
        status: 'active',
        source: 'enterprise_allocation',
        source_type: '',
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

    assert.equal(managed, 'Managed by department · Cannot be deleted by user')
    assert.equal(title, 'Enterprise Allocation Wallet · Subscription #130')
  })

  test('treats legacy enterprise wallets as managed by department', () => {
    const subscription = {
      id: 131,
      user_id: 9001,
      plan_id: 0,
      status: 'active',
      source: 'enterprise',
      source_type: 'enterprise',
      source_allocation_id: 89,
      sort_order: -100,
      is_primary: false,
      start_time: 1700000000,
      end_time: 1800000000,
      amount_total: 500,
      amount_used: 0,
      next_reset_time: 0,
    } as const

    assert.equal(
      getManagedSubscriptionNote(subscription, i18n.t.bind(i18n)),
      'Managed by department · Cannot be deleted by user'
    )
    assert.equal(
      getSubscriptionCardTitle(subscription, i18n.t.bind(i18n)),
      'Enterprise Allocation Wallet · Subscription #131'
    )
  })

  test('renders enterprise wallet title and managed note with zh locale translations', async () => {
    const previousLanguage = i18n.language
    await i18n.changeLanguage('zhCN')
    try {
      const subscription = {
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
      } as const

      const title = getSubscriptionCardTitle(subscription, i18n.t.bind(i18n))
      const note = getManagedSubscriptionNote(subscription, i18n.t.bind(i18n))

      assert.equal(title, '企业分配钱包 · 订阅 #12')
      assert.equal(note, '由部门管理 · 用户不可删除')
      assert.doesNotMatch(title, /Enterprise Allocation Wallet/)
      assert.doesNotMatch(note ?? '', /Managed by department/)
      assert.doesNotMatch(note ?? '', /Cannot be deleted by user/)
    } finally {
      await i18n.changeLanguage(previousLanguage)
    }
  })

  test('maps billing preference and subscription source without raw internal fallback', async () => {
    const previousLanguage = i18n.language
    await i18n.changeLanguage('zhCN')
    try {
      assert.equal(
        getBillingPreferenceLabel('subscription_first', i18n.t.bind(i18n)),
        '优先订阅'
      )
      assert.equal(
        getBillingPreferenceLabel('future_preference', i18n.t.bind(i18n)),
        '未知扣费偏好'
      )
      assert.equal(
        getSubscriptionSourceLabel('enterprise_allocation', i18n.t.bind(i18n)),
        '企业分配'
      )
      assert.equal(
        getSubscriptionSourceLabel('enterprise', i18n.t.bind(i18n)),
        '企业分配'
      )
      assert.equal(
        getSubscriptionSourceLabel('future_source', i18n.t.bind(i18n)),
        '未知来源'
      )
      assert.equal(
        getSubscriptionSourceLabel(undefined, i18n.t.bind(i18n)),
        '-'
      )
    } finally {
      await i18n.changeLanguage(previousLanguage)
    }
  })

  test('keeps enterprise allocation wallets with non-positive end time active and semantic', () => {
    const t = i18n.t.bind(i18n)
    const subscription = {
      id: 15,
      user_id: 9001,
      plan_id: 0,
      status: 'active',
      source: 'enterprise_allocation',
      source_type: 'enterprise_allocation',
      source_allocation_id: 90,
      sort_order: -100,
      is_primary: false,
      start_time: 1700000000,
      end_time: 0,
      amount_total: 500,
      amount_used: 20,
      next_reset_time: 0,
    } as const

    const status = getSubscriptionStatusDisplay(subscription, t)
    const expiry = getSubscriptionExpiryDisplay(subscription, t)

    assert.deepEqual(status, {
      label: 'Active',
      variant: 'success',
      isActive: true,
      isCancelled: false,
      isExpired: false,
    })
    assert.equal(expiry.label, 'Until')
    assert.equal(expiry.value, 'Never expires')
    assert.equal(getSubscriptionRemainingDays(subscription), null)
    assert.doesNotMatch(expiry.value, /1970|1969|Invalid Date/)
  })

  test('treats enterprise wallets identified only by source as active and semantic', () => {
    const t = i18n.t.bind(i18n)
    const subscription = {
      id: 151,
      user_id: 9001,
      plan_id: 0,
      status: 'active',
      source: 'enterprise_allocation',
      source_type: '',
      source_allocation_id: 90,
      sort_order: -100,
      is_primary: false,
      start_time: 1700000000,
      end_time: 0,
      amount_total: 500,
      amount_used: 20,
      next_reset_time: 0,
    } as const

    const status = getSubscriptionStatusDisplay(subscription, t)
    const expiry = getSubscriptionExpiryDisplay(subscription, t)

    assert.equal(status.label, 'Active')
    assert.equal(expiry.value, 'Never expires')
    assert.equal(
      getSubscriptionSourceLabel(subscription.source, t),
      'Enterprise allocation'
    )
  })

  test('treats legacy enterprise wallets as active and semantic', () => {
    const t = i18n.t.bind(i18n)
    const subscription = {
      id: 152,
      user_id: 9001,
      plan_id: 0,
      status: 'active',
      source: 'enterprise',
      source_type: 'enterprise',
      source_allocation_id: 90,
      sort_order: -100,
      is_primary: false,
      start_time: 1700000000,
      end_time: 0,
      amount_total: 500,
      amount_used: 20,
      next_reset_time: 0,
    } as const

    const status = getSubscriptionStatusDisplay(subscription, t)
    const expiry = getSubscriptionExpiryDisplay(subscription, t)

    assert.equal(status.label, 'Active')
    assert.equal(expiry.value, 'Never expires')
    assert.equal(
      getSubscriptionSourceLabel(subscription.source, t),
      'Enterprise allocation'
    )
  })

  test('keeps ordinary plans with non-positive end time expired while preserving No Reset plan wording', () => {
    const t = i18n.t.bind(i18n)
    const subscription = {
      id: 16,
      user_id: 9001,
      plan_id: 7,
      status: 'active',
      source: 'subscription',
      source_type: 'subscription',
      source_allocation_id: 0,
      sort_order: 100,
      is_primary: true,
      start_time: 1700000000,
      end_time: 0,
      amount_total: 500,
      amount_used: 20,
      next_reset_time: 0,
    } as const

    const status = getSubscriptionStatusDisplay(subscription, t)
    const expiry = getSubscriptionExpiryDisplay(subscription, t)

    assert.equal(status.label, 'Expired')
    assert.equal(status.isExpired, true)
    assert.equal(expiry.label, 'Expired at')
    assert.equal(expiry.value, 'No expiry set')
    assert.equal(t('No Reset'), 'No Reset')
    assert.doesNotMatch(expiry.value, /1970|1969|Invalid Date/)
    assert.equal(isActiveSubscriptionForBilling(subscription), true)
  })

  test('maps active subscriptions for billing using backend active-subscription semantics', () => {
    const now = 1800000000
    assert.equal(
      isActiveSubscriptionForBilling(
        {
          id: 18,
          user_id: 9001,
          plan_id: 7,
          status: 'active',
          source: 'admin',
          source_type: 'admin',
          source_allocation_id: 0,
          sort_order: 100,
          is_primary: true,
          start_time: 1700000000,
          end_time: 0,
          amount_total: 500,
          amount_used: 20,
          next_reset_time: 0,
        },
        now
      ),
      true
    )
    assert.equal(
      isActiveSubscriptionForBilling(
        {
          id: 19,
          user_id: 9001,
          plan_id: 7,
          status: 'active',
          source: 'admin',
          source_type: 'admin',
          source_allocation_id: 0,
          sort_order: 100,
          is_primary: true,
          start_time: 1700000000,
          end_time: now - 1,
          amount_total: 500,
          amount_used: 20,
          next_reset_time: 0,
        },
        now
      ),
      false
    )
  })

  test('enterprise wallet non-positive expiry semantics are translated in zh locale', async () => {
    const previousLanguage = i18n.language
    await i18n.changeLanguage('zhCN')
    try {
      const subscription = {
        id: 17,
        user_id: 9001,
        plan_id: 0,
        status: 'active',
        source: 'enterprise_allocation',
        source_type: 'enterprise_allocation',
        source_allocation_id: 91,
        sort_order: -100,
        is_primary: false,
        start_time: 1700000000,
        end_time: -1,
        amount_total: 500,
        amount_used: 20,
        next_reset_time: 0,
      } as const

      const expiry = getSubscriptionExpiryDisplay(
        subscription,
        i18n.t.bind(i18n)
      )

      assert.equal(expiry.value, '永不过期')
      assert.doesNotMatch(expiry.value, /Never expires/)
    } finally {
      await i18n.changeLanguage(previousLanguage)
    }
  })

  test('enterprise wallet visible fields have locale coverage for every supported language', () => {
    const requiredKeys = [
      'Enterprise Allocation Wallet',
      'Managed by department',
      'Cannot be deleted by user',
      'Subscription',
      'Active',
      'Cancelled',
      'Expired',
      'Source',
      'Subscription Priority',
      'Total Quota',
      'Remaining',
      'Used',
      'Subscription First',
      'Wallet First',
      'Subscription Only',
      'Wallet Only',
      'Enterprise allocation',
      'Admin',
      'User',
      'System',
      'Payment',
      'Manual',
      'Unknown billing preference',
      'Unknown source',
      'Preference saved as {{pref}}, but no active subscription. Wallet will be used automatically.',
      'Never expires',
      'No expiry set',
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
})
