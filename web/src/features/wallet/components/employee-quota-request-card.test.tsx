import { describe, test } from 'bun:test'
import assert from 'node:assert/strict'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'

import {
  getQuotaRequestCapability,
  getUserDepartments,
  governanceNotificationQueryKey,
  governanceTimelineQueryKey,
  quotaRequestQueryScopeKey,
  submitQuotaRequest,
  userDepartmentsQueryKey,
} from '@/features/enterprise-organization/api'
import {
  convertEnterpriseQuotaInputMode,
  parseEnterpriseQuotaInput,
} from '@/features/enterprise-organization/quota-amount-controls'
import {
  getQuotaRequestBudgetDisplayText,
  getQuotaRequestBudgetTriggerLabel,
} from '@/features/enterprise-organization/quota-request-budget-display'
import {
  QuotaRequestBudgetOption,
  QuotaRequestBudgetSummary,
} from '@/features/enterprise-organization/quota-request-budget-display-components'
import type {
  QuotaRequestCapabilityBudgetItem,
  UserDepartmentItem,
} from '@/features/enterprise-organization/types'
import i18n from '@/i18n/config'
import { api } from '@/lib/api'

import type { UserWalletData } from '../types'
import {
  EmployeeQuotaRequestCard,
  getEmployeeQuotaRequestDepartmentOptions,
  invalidateQuotaRequestGovernanceQueries,
  resolveEmployeeQuotaRequestBudgetId,
} from './employee-quota-request-card'

i18n.changeLanguage('en')

describe('Employee quota request wallet entry', () => {
  test('filters quota request departments to active memberships only', () => {
    const departments = getEmployeeQuotaRequestDepartmentOptions([
      userDepartment({
        id: 1,
        department_id: 11,
        department_name: 'Engineering',
        status: 1,
      }),
      userDepartment({
        id: 2,
        department_id: 12,
        department_name: 'Finance',
        status: 2,
      }),
      userDepartment({
        id: 3,
        department_id: 13,
        department_name: 'Platform',
        status: 1,
      }),
    ])

    assert.deepEqual(
      departments.map((item) => item.department_name),
      ['Engineering', 'Platform']
    )
  })

  test('requires an explicit budget selection from the selected department capability', () => {
    assert.equal(resolveEmployeeQuotaRequestBudgetId([31, 32], 32), 32)
    assert.equal(resolveEmployeeQuotaRequestBudgetId([31, 32], 99), null)
    assert.equal(resolveEmployeeQuotaRequestBudgetId([], 31), null)
    assert.equal(resolveEmployeeQuotaRequestBudgetId([31], null), null)
  })

  test('formats wallet budget options and selected summary with identifiable budget details', () => {
    const budget = quotaRequestBudget({
      id: 31,
      department_id: 11,
      department_name: 'Engineering',
      type: 'balance',
      status: 'active',
      remaining: 1250,
    })
    const display = getQuotaRequestBudgetDisplayText(budget, i18n.t)
    assert.equal(
      getQuotaRequestBudgetTriggerLabel(budget, i18n.t),
      'Engineering · Budget #31'
    )

    assert.deepEqual(display, {
      departmentName: 'Engineering',
      identity: 'Budget #31',
      typeLabel: 'Balance Budget',
      remainingLabel: 'Remaining 1,250 quota',
      remainingAmountLabel: 'Approx. $0.0025',
      statusLabel: 'Active',
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
        'Balance Budget',
        'Remaining 1,250 quota',
        'Approx. $0.0025',
        'Active',
      ]) {
        assert.match(html, new RegExp(escapeRegExp(expected)))
      }
    }
  })

  test('formats wallet quota request budget surfaces with zh locale translations', async () => {
    const previousLanguage = i18n.language
    await i18n.changeLanguage('zhCN')
    try {
      const budget = {
        ...quotaRequestBudget({
          id: 31,
          department_id: 11,
          department_name: 'Engineering',
          remaining: 1250,
        }),
        type: 'future_budget',
        status: 'future_status',
      } as unknown as QuotaRequestCapabilityBudgetItem
      const display = getQuotaRequestBudgetDisplayText(budget, i18n.t)

      assert.deepEqual(display, {
        departmentName: 'Engineering',
        identity: '预算池 #31',
        typeLabel: '未知预算类型',
        remainingLabel: '剩余额度 1,250 额度',
        remainingAmountLabel: '约 $0.0025',
        statusLabel: '未知状态',
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
          '预算池 #31',
          '未知预算类型',
          '剩余额度 1,250 额度',
          '约 $0.0025',
          '未知状态',
        ]) {
          assert.match(html, new RegExp(escapeRegExp(expected)))
        }
        assert.doesNotMatch(html, />future_budget</)
        assert.doesNotMatch(html, />future_status</)
      }
    } finally {
      await i18n.changeLanguage(previousLanguage)
    }
  })

  test('renders wallet quota request form labels with zh locale translations', async () => {
    const previousLanguage = i18n.language
    await i18n.changeLanguage('zhCN')
    try {
      const html = renderToStaticMarkup(
        <QueryClientProvider client={new QueryClient()}>
          <I18nextProvider i18n={i18n}>
            <EmployeeQuotaRequestCard user={userWallet()} />
          </I18nextProvider>
        </QueryClientProvider>
      )

      for (const expected of [
        '需要更多额度？',
        '从钱包申请额度',
        '目标部门',
        '目标预算池',
        '已选申请范围',
        '未选择预算池',
        '申请额度',
        '金额视图',
        '按额度单位存储',
        '提交额度申请',
      ]) {
        assert.match(html, new RegExp(escapeRegExp(expected)))
      }
      assert.doesNotMatch(html, /Target Budget Pool/)
      assert.doesNotMatch(html, /Submit quota request/)
    } finally {
      await i18n.changeLanguage(previousLanguage)
    }
  })

  test('renders wallet-side quota request guidance without enterprise organization navigation dependency', () => {
    const html = renderToStaticMarkup(
      <QueryClientProvider client={new QueryClient()}>
        <I18nextProvider i18n={i18n}>
          <EmployeeQuotaRequestCard user={userWallet()} />
        </I18nextProvider>
      </QueryClientProvider>
    )

    for (const expected of [
      'Need more quota?',
      'Apply for quota from your wallet',
      'Choose an active department and one of its available budget pools before submitting.',
      'Target Department',
      'Target Budget Pool',
      'Selected request scope',
      'No budget pool selected',
      'Requested Quota',
      'Amount view',
      'Stored as quota units',
      'Submit quota request',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }

    assert.doesNotMatch(html, /Enterprise Organization/)
  })

  test('renders wallet quota request amounts as a fixed monetary input', () => {
    const html = renderToStaticMarkup(
      <QueryClientProvider client={new QueryClient()}>
        <I18nextProvider i18n={i18n}>
          <EmployeeQuotaRequestCard user={userWallet()} />
        </I18nextProvider>
      </QueryClientProvider>
    )

    assert.match(html, /inputMode="decimal"/)
    assert.doesNotMatch(html, />Quota view</)
    assert.match(html, />Amount view</)
  })

  test('reuses enterprise quota request endpoints and query scope from the wallet entry', async () => {
    const calls: Array<{
      method: string
      url: string
      params?: unknown
      data?: unknown
    }> = []
    const originalGet = api.get
    const originalPost = api.post

    api.get = (async (url: string, config?: { params?: unknown }) => {
      calls.push({ method: 'GET', url, params: config?.params })
      if (url === '/api/enterprise/users/2001/departments') {
        return {
          data: {
            success: true,
            data: {
              items: [
                userDepartment({
                  id: 1,
                  department_id: 11,
                  department_name: 'Engineering',
                }),
              ],
            },
          },
        }
      }
      if (url === '/api/enterprise/quota-requests/capability/11') {
        return {
          data: {
            success: true,
            data: {
              can_submit: true,
              can_govern: false,
              budgets: [
                {
                  id: 31,
                  tenant_id: 0,
                  department_id: 11,
                  department_name: 'Engineering',
                  type: 'balance',
                  status: 'active',
                  remaining: 500,
                },
              ],
            },
          },
        }
      }
      throw new Error(`Unexpected GET ${url}`)
    }) as typeof api.get
    api.post = (async (url: string, data?: unknown) => {
      calls.push({ method: 'POST', url, data })
      return { data: { success: true, data: quotaRequestResponse() } }
    }) as typeof api.post

    try {
      const departments = await getUserDepartments(2001)
      const capability = await getQuotaRequestCapability(11, 0)
      const submit = await submitQuotaRequest({
        tenant_id: 0,
        department_id: 11,
        department_budget_id: 31,
        budget_mode: 'department_budget',
        requested_quota: 200,
        request_reason: 'temporary launch quota',
      })

      assert.equal(departments.success, true)
      assert.equal(capability.success, true)
      assert.equal(submit.success, true)
      assert.deepEqual(userDepartmentsQueryKey(2001), [
        'enterprise',
        'organization',
        'user-departments',
        2001,
      ])
      assert.deepEqual(quotaRequestQueryScopeKey(11, 0), [
        'enterprise',
        'organization',
        'quota-request',
        11,
        0,
      ])
      assert.deepEqual(calls, [
        {
          method: 'GET',
          url: '/api/enterprise/users/2001/departments',
          params: undefined,
        },
        {
          method: 'GET',
          url: '/api/enterprise/quota-requests/capability/11',
          params: { tenant_id: 0 },
        },
        {
          method: 'POST',
          url: '/api/enterprise/quota-requests',
          data: {
            tenant_id: 0,
            department_id: 11,
            department_budget_id: 31,
            budget_mode: 'department_budget',
            requested_quota: 200,
            request_reason: 'temporary launch quota',
          },
        },
      ])
    } finally {
      api.get = originalGet
      api.post = originalPost
    }
  })

  test('wallet quota request amount view still resolves to quota-unit payload values', () => {
    assert.equal(parseEnterpriseQuotaInput('2', 'amount'), 1_000_000)
    assert.equal(parseEnterpriseQuotaInput('0.25', 'amount'), 125_000)
    assert.deepEqual(
      convertEnterpriseQuotaInputMode({
        value: '2',
        from: 'amount',
        to: 'quota',
      }),
      { value: '1000000', quota: 1_000_000 }
    )
  })

  test('invalidates governance timeline and notification caches after wallet submission', async () => {
    const queryClient = new QueryClient()
    const invalidated: unknown[] = []
    const originalInvalidateQueries =
      queryClient.invalidateQueries.bind(queryClient)
    queryClient.invalidateQueries = ((filters: { queryKey?: unknown }) => {
      invalidated.push(filters.queryKey)
      return Promise.resolve()
    }) as typeof queryClient.invalidateQueries

    try {
      await invalidateQuotaRequestGovernanceQueries(queryClient, 11, 0)
    } finally {
      queryClient.invalidateQueries = originalInvalidateQueries
    }

    assert.deepEqual(invalidated, [
      governanceTimelineQueryKey(11, 0),
      governanceNotificationQueryKey(11, 0),
    ])
  })

  test('clears stale wallet budget selection when department capability changes', () => {
    assert.equal(resolveEmployeeQuotaRequestBudgetId([41], 31), null)
    assert.equal(resolveEmployeeQuotaRequestBudgetId([41], 41), 41)
  })

  test('renders empty guidance for employees without requestable department memberships', () => {
    const queryClient = new QueryClient()
    queryClient.setQueryData(userDepartmentsQueryKey(2001), [])
    const html = renderToStaticMarkup(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <EmployeeQuotaRequestCard user={userWallet()} />
        </I18nextProvider>
      </QueryClientProvider>
    )

    assert.match(html, /No requestable departments/)
    assert.match(
      html,
      /You must be an active member of a department before submitting a quota request from wallet\./
    )
  })
})

function userWallet(overrides: Partial<UserWalletData> = {}): UserWalletData {
  return {
    id: overrides.id ?? 2001,
    username: overrides.username ?? 'alice',
    quota: overrides.quota ?? 0,
    used_quota: overrides.used_quota ?? 100,
    request_count: overrides.request_count ?? 10,
    aff_quota: overrides.aff_quota ?? 0,
    aff_history_quota: overrides.aff_history_quota ?? 0,
    aff_count: overrides.aff_count ?? 0,
    group: overrides.group ?? 'default',
  }
}

function quotaRequestBudget(
  overrides: Partial<QuotaRequestCapabilityBudgetItem> = {}
): QuotaRequestCapabilityBudgetItem {
  return {
    id: overrides.id ?? 31,
    tenant_id: overrides.tenant_id ?? 0,
    department_id: overrides.department_id ?? 11,
    department_name: overrides.department_name ?? 'Engineering',
    type: overrides.type ?? 'balance',
    status: overrides.status ?? 'active',
    total_quota: overrides.total_quota ?? 0,
    remaining: overrides.remaining ?? 500,
    allocated_total: overrides.allocated_total ?? 0,
    cycle_quota: overrides.cycle_quota ?? 0,
    cycle_type: overrides.cycle_type ?? '',
    cycle_started_at: overrides.cycle_started_at ?? 0,
    custom_seconds: overrides.custom_seconds ?? 0,
    expires_at: overrides.expires_at ?? 0,
    parent_status: overrides.parent_status ?? '',
    usage_ratio: overrides.usage_ratio ?? 0,
    threshold_state: overrides.threshold_state ?? 'healthy',
    created_at: overrides.created_at ?? 1700000000,
    updated_at: overrides.updated_at ?? 1700000001,
  }
}

function userDepartment(
  overrides: Partial<UserDepartmentItem> &
    Pick<UserDepartmentItem, 'id' | 'department_id' | 'department_name'>
): UserDepartmentItem {
  return {
    id: overrides.id,
    tenant_id: overrides.tenant_id ?? 0,
    user_id: overrides.user_id ?? 2001,
    department_id: overrides.department_id,
    department_name: overrides.department_name,
    external_user_id: overrides.external_user_id ?? '',
    external_source: overrides.external_source ?? 'manual',
    status: overrides.status ?? 1,
    joined_at: overrides.joined_at ?? 1700000000,
    left_at: overrides.left_at ?? 0,
    created_at: overrides.created_at ?? 1700000000,
    updated_at: overrides.updated_at ?? 1700000001,
  }
}

function quotaRequestResponse() {
  return {
    id: 1,
    tenant_id: 0,
    department_id: 11,
    department_name: 'Engineering',
    department_budget_id: 31,
    budget_mode: 'department_budget',
    requester_user_id: 2001,
    requester_username: 'alice',
    requester_display_name: 'Alice',
    requested_quota: 200,
    approved_quota: 0,
    status: 'submitted',
    approver_user_id: 0,
    approver_username: '',
    approval_reason: '',
    request_reason: 'temporary launch quota',
    allocation_id: 0,
    owner_count_snapshot: 1,
    fallback: '',
    submitted_at: 1700000000,
    approved_at: 0,
    rejected_at: 0,
    fulfilled_at: 0,
    processed_at: 0,
    expires_at: 0,
    created_at: 1700000000,
    updated_at: 1700000001,
  }
}

function escapeRegExp(value: string) {
  return value.replaceAll(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
