/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { isRedirect } from '@tanstack/react-router'
import i18n from '@/i18n/config'
import { Route as EnterpriseUsageRoute } from '@/routes/_authenticated/enterprise-usage/index'
import assert from 'node:assert/strict'
import { Buffer } from 'node:buffer'
import { describe, test } from 'node:test'
import { useForm } from 'react-hook-form'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { buildSearchParams } from '@/features/usage-logs/lib/filter'
import {
  departmentDetailQueryKey,
  departmentUsageReportQueryKey,
  departmentSummaryQueryKey,
  exportDepartmentUsageCSV,
  getDepartmentUsageReportConfig,
  saveDepartmentUsageReportConfig,
} from './api'
import {
  EnterpriseUsageContent,
  compareDepartmentUsageItems,
  enterpriseUsageSearchSchema,
  formatModelDistributionSummary,
  normalizeDepartmentUsageDetail,
  normalizeDepartmentUsageItems,
  resolveDepartmentUsageExportParams,
  resolveEnterpriseUsageRange,
  resolveRecentLogsSearch,
  shouldResetReportForm,
  sortDepartmentUserRanking,
} from './index'
import type {
  DepartmentUsageDetailResponse,
  DepartmentUsageReportJobItem,
  DepartmentUsageSummaryItem,
} from './types'

describe('Enterprise usage overview dashboard', () => {
  test('resolves default and preset-backed time windows into exact query params', () => {
    const now = new Date('2026-05-29T12:00:00.000Z')

    const today = resolveEnterpriseUsageRange({}, now)
    assert.equal(today.preset, 'today')
    assert.equal(today.from, 1779984000)
    assert.equal(today.to, 1780070400)
    assert.equal(today.rangeLabel, '2026-05-29 ~ 2026-05-29')

    const yesterday = resolveEnterpriseUsageRange({ preset: 'yesterday' }, now)
    assert.equal(yesterday.from, 1779897600)
    assert.equal(yesterday.to, 1779984000)
    assert.equal(yesterday.rangeLabel, '2026-05-28 ~ 2026-05-28')

    const last7 = resolveEnterpriseUsageRange({ preset: 'last7d' }, now)
    assert.equal(last7.from, 1779465600)
    assert.equal(last7.to, 1780070400)
    assert.equal(last7.rangeLabel, '2026-05-23 ~ 2026-05-29')

    const last30 = resolveEnterpriseUsageRange({ preset: 'last30d' }, now)
    assert.equal(last30.from, 1777478400)
    assert.equal(last30.to, 1780070400)
    assert.equal(last30.rangeLabel, '2026-04-30 ~ 2026-05-29')
  })

  test('accepts valid custom ranges and falls back invalid custom ranges to today', () => {
    const now = new Date('2026-05-29T12:00:00.000Z')

    const custom = resolveEnterpriseUsageRange(
      {
        preset: 'custom',
        from: 1748390400,
        to: 1748476800,
      },
      now
    )
    assert.equal(custom.isCustom, true)
    assert.equal(custom.isRangeValid, true)
    assert.equal(custom.from, 1748390400)
    assert.equal(custom.to, 1748476800)
    assert.equal(custom.customToDate, '2025-05-28')
    assert.equal(custom.rangeLabel, '2025-05-28 ~ 2025-05-29')

    const invalid = resolveEnterpriseUsageRange(
      {
        preset: 'custom',
        from: 1748476800,
        to: 1748476800,
      },
      now
    )
    assert.equal(invalid.preset, 'today')
    assert.equal(invalid.isCustom, false)
    assert.equal(invalid.isRangeValid, true)
  })

  test('search schema preserves custom from/to values', () => {
    const parsed = enterpriseUsageSearchSchema.parse({
      preset: 'custom',
      from: '1748390400',
      to: '1748476800',
      tenant_id: '3',
    })
    assert.equal(parsed.preset, 'custom')
    assert.equal(parsed.from, 1748390400)
    assert.equal(parsed.to, 1748476800)
    assert.equal(parsed.tenant_id, 3)
  })

  test('builds a serializable feature-scoped query key for department summary requests', () => {
    assert.deepEqual(departmentSummaryQueryKey(1748390400, 1748476799), [
      'enterprise',
      'usage',
      'department-summary',
      {
        from: 1748390400,
        to: 1748476799,
        summarySort: undefined,
        summaryOrder: undefined,
      },
    ])

    assert.deepEqual(
      departmentSummaryQueryKey(1748390400, 1748476800, 2, 'quota', 'asc'),
      [
        'enterprise',
        'usage',
        'department-summary',
        {
          from: 1748390400,
          to: 1748476800,
          tenantId: 2,
          summarySort: 'quota',
          summaryOrder: 'asc',
        },
      ]
    )

    assert.deepEqual(departmentDetailQueryKey(5, 1748390400, 1748476800, 2), [
      'enterprise',
      'usage',
      'department-detail',
      { deptId: 5, from: 1748390400, to: 1748476800, tenantId: 2 },
    ])

    assert.deepEqual(departmentUsageReportQueryKey, [
      'enterprise',
      'usage',
      'report-config',
    ])
  })

  test('route guard redirects non-admin users to 403 and allows admins', () => {
    const { auth } = useAuthStore.getState()
    const previousUser = auth.user

    try {
      auth.setUser({
        id: 1001,
        username: 'member',
        role: ROLE.USER,
      })

      let redirected: unknown = null
      try {
        EnterpriseUsageRoute.options.beforeLoad?.({} as never)
      } catch (error) {
        redirected = error
      }

      assert.ok(isRedirect(redirected))
      if (isRedirect(redirected)) {
        assert.equal(redirected.options.to, '/403')
      }

      auth.setUser({
        id: 1002,
        username: 'admin',
        role: ROLE.ADMIN,
      })

      assert.doesNotThrow(() =>
        EnterpriseUsageRoute.options.beforeLoad?.({} as never)
      )
    } finally {
      auth.setUser(previousUser)
    }
  })

  test('renders disclaimer, unassigned row, and compact model distribution summary', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <EnterpriseUsageContent
          items={[
            departmentUsageItem({
              dept_id: null,
              dept_name: '',
              request_count: 12,
              prompt_tokens: 1200,
              completion_tokens: 600,
              quota: 320000,
              user_count: 3,
              model_distribution: [
                {
                  model_name: 'gpt-4o',
                  request_count: 8,
                  prompt_tokens: 800,
                  completion_tokens: 400,
                  quota: 240000,
                },
                {
                  model_name: 'claude-sonnet-4',
                  request_count: 3,
                  prompt_tokens: 300,
                  completion_tokens: 150,
                  quota: 60000,
                },
                {
                  model_name: 'gemini-2.5-pro',
                  request_count: 1,
                  prompt_tokens: 100,
                  completion_tokens: 50,
                  quota: 20000,
                },
                {
                  model_name: 'qwen-max',
                  request_count: 1,
                  prompt_tokens: 80,
                  completion_tokens: 40,
                  quota: 10000,
                },
              ],
            }),
          ]}
          selectedDepartmentId={undefined}
          detail={null}
          detailLoading={false}
          detailErrorMessage={null}
          isLoading={false}
          errorMessage={null}
          rangeLabel='2026-05-28 ~ 2026-05-28'
          customRange={{
            from: '2026-05-28',
            to: '2026-05-28',
            isValid: true,
          }}
          onCustomRangeChange={() => undefined}
          onApplyCustomRange={() => undefined}
          onPresetChange={() => undefined}
          selectedPreset='today'
          onRetry={() => undefined}
          onSelectDepartment={() => undefined}
          onBackToOverview={() => undefined}
          onRetryDetail={() => undefined}
          rankSort='quota'
          onSortChange={() => undefined}
          summarySort='requests'
          summaryOrder='desc'
          onSummarySortChange={() => undefined}
          onExport={() => undefined}
          exportLoading={false}
          selectedLogUser={undefined}
          onOpenRecentLogs={() => undefined}
        />
      </I18nextProvider>
    )

    for (const expected of [
      'Department totals cannot be added together across departments.',
      'Unassigned',
      'gpt-4o',
      'claude-sonnet-4',
      '+1 more models',
      'Enterprise Usage Overview',
      'Top Model',
      'Models',
      'Department totals are non-additive',
      '2026-05-28 ~ 2026-05-28',
      'Today',
      'Yesterday',
      'Last 7 Days',
      'Last 30 Days',
      'Custom',
      'Apply',
      'View Details',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('renders empty and error states with retry affordance', () => {
    const emptyHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <EnterpriseUsageContent
          items={[]}
          selectedDepartmentId={undefined}
          detail={null}
          detailLoading={false}
          detailErrorMessage={null}
          isLoading={false}
          errorMessage={null}
          rangeLabel='2026-05-29 ~ 2026-05-29'
          customRange={{
            from: '2026-05-29',
            to: '2026-05-29',
            isValid: true,
          }}
          onCustomRangeChange={() => undefined}
          onApplyCustomRange={() => undefined}
          onPresetChange={() => undefined}
          selectedPreset='today'
          onRetry={() => undefined}
          onSelectDepartment={() => undefined}
          onBackToOverview={() => undefined}
          onRetryDetail={() => undefined}
          rankSort='quota'
          onSortChange={() => undefined}
          summarySort='requests'
          summaryOrder='desc'
          onSummarySortChange={() => undefined}
          onExport={() => undefined}
          exportLoading={false}
          selectedLogUser={undefined}
          onOpenRecentLogs={() => undefined}
        />
      </I18nextProvider>
    )
    assert.match(emptyHtml, /No department usage data for this time range/)
    assert.match(emptyHtml, /Enterprise Usage Overview/)

    const errorHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <EnterpriseUsageContent
          items={[]}
          selectedDepartmentId={undefined}
          detail={null}
          detailLoading={false}
          detailErrorMessage={null}
          isLoading={false}
          errorMessage='common.invalid_params'
          rangeLabel='2026-05-29 ~ 2026-05-29'
          customRange={{
            from: '2026-05-29',
            to: '2026-05-28',
            isValid: false,
          }}
          onCustomRangeChange={() => undefined}
          onApplyCustomRange={() => undefined}
          onPresetChange={() => undefined}
          selectedPreset='custom'
          onRetry={() => undefined}
          onSelectDepartment={() => undefined}
          onBackToOverview={() => undefined}
          onRetryDetail={() => undefined}
          rankSort='quota'
          onSortChange={() => undefined}
          summarySort='requests'
          summaryOrder='desc'
          onSummarySortChange={() => undefined}
          onExport={() => undefined}
          exportLoading={false}
          selectedLogUser={undefined}
          onOpenRecentLogs={() => undefined}
        />
      </I18nextProvider>
    )
    assert.match(errorHtml, /common\.invalid_params|Invalid parameters/)
    assert.match(errorHtml, />Retry</)
    assert.match(
      errorHtml,
      /Start time must be earlier than end time for a custom range/
    )
  })

  test('renders loading skeletons while summary data is pending', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <EnterpriseUsageContent
          items={[]}
          selectedDepartmentId={undefined}
          detail={null}
          detailLoading={false}
          detailErrorMessage={null}
          isLoading
          errorMessage={null}
          rangeLabel='2026-05-29 ~ 2026-05-29'
          customRange={{
            from: '2026-05-29',
            to: '2026-05-29',
            isValid: true,
          }}
          onCustomRangeChange={() => undefined}
          onApplyCustomRange={() => undefined}
          onPresetChange={() => undefined}
          selectedPreset='today'
          onRetry={() => undefined}
          onSelectDepartment={() => undefined}
          onBackToOverview={() => undefined}
          onRetryDetail={() => undefined}
          rankSort='quota'
          onSortChange={() => undefined}
          summarySort='requests'
          summaryOrder='desc'
          onSummarySortChange={() => undefined}
          onExport={() => undefined}
          exportLoading={false}
          selectedLogUser={undefined}
          onOpenRecentLogs={() => undefined}
        />
      </I18nextProvider>
    )

    assert.match(html, /animate-pulse/)
    assert.doesNotMatch(html, /No department usage data for this time range/)
    assert.doesNotMatch(html, /Unable to load department usage/)
    assert.doesNotMatch(html, /<table/i)
  })

  test('formats model distribution summaries without rendering a heavy detail table', () => {
    const summary = formatModelDistributionSummary(
      [
        {
          model_name: 'gpt-4o',
          request_count: 20,
          prompt_tokens: 0,
          completion_tokens: 0,
          quota: 300000,
        },
        {
          model_name: 'claude-sonnet-4',
          request_count: 10,
          prompt_tokens: 0,
          completion_tokens: 0,
          quota: 120000,
        },
        {
          model_name: 'gemini-2.5-pro',
          request_count: 6,
          prompt_tokens: 0,
          completion_tokens: 0,
          quota: 80000,
        },
        {
          model_name: 'qwen-max',
          request_count: 2,
          prompt_tokens: 0,
          completion_tokens: 0,
          quota: 10000,
        },
      ],
      (key, options) =>
        key === 'enterprise.usage.more_models'
          ? `+${options?.count as number} more models`
          : key
    )

    assert.match(summary, /gpt-4o/)
    assert.match(summary, /claude-sonnet-4/)
    assert.match(summary, /\+1 more models/)
    assert.doesNotMatch(summary, /<table/i)
  })

  test('normalizes nullish values and sorts departments by request count then quota', () => {
    const normalized = normalizeDepartmentUsageItems([
      departmentUsageItem({
        dept_id: 2,
        dept_name: 'Ops',
        request_count: 4,
        quota: 100,
        model_distribution: undefined as never,
      }),
      departmentUsageItem({
        dept_id: null,
        dept_name: '',
        request_count: 9,
        quota: 50,
      }),
      departmentUsageItem({
        dept_id: 3,
        dept_name: 'Engineering',
        request_count: 4,
        quota: 300,
      }),
    ])

    assert.equal(normalized[0]?.dept_id, null)
    assert.equal(normalized[1]?.dept_name, 'Engineering')
    assert.equal(normalized[2]?.dept_name, 'Ops')
    assert.deepEqual(normalized[2]?.model_distribution, [])

    const byUsersAsc = normalizeDepartmentUsageItems(
      [
        departmentUsageItem({ dept_id: 9, dept_name: 'Gamma', user_count: 8 }),
        departmentUsageItem({ dept_id: 8, dept_name: 'Alpha', user_count: 2 }),
      ],
      'users',
      'asc'
    )
    assert.equal(byUsersAsc[0]?.dept_name, 'Alpha')
  })

  test('uses the same summary sort comparator semantics for page ordering and export state', () => {
    const alpha = departmentUsageItem({
      dept_id: 1,
      dept_name: 'Alpha',
      request_count: 3,
      quota: 100,
      user_count: 4,
    })
    const beta = departmentUsageItem({
      dept_id: 2,
      dept_name: 'Beta',
      request_count: 9,
      quota: 200,
      user_count: 2,
    })

    assert.equal(
      compareDepartmentUsageItems(alpha, beta, 'requests', 'desc') > 0,
      true
    )
    assert.equal(
      compareDepartmentUsageItems(alpha, beta, 'dept_name', 'asc') < 0,
      true
    )
    assert.equal(
      compareDepartmentUsageItems(alpha, beta, 'users', 'desc') < 0,
      true
    )

    const duplicateNameHigherId = departmentUsageItem({
      dept_id: 5,
      dept_name: 'Alpha',
      request_count: 10,
      quota: 100,
      user_count: 2,
    })
    const duplicateNameLowerId = departmentUsageItem({
      dept_id: 3,
      dept_name: 'Alpha',
      request_count: 10,
      quota: 100,
      user_count: 2,
    })
    assert.equal(
      compareDepartmentUsageItems(
        duplicateNameHigherId,
        duplicateNameLowerId,
        'dept_name',
        'asc'
      ) > 0,
      true
    )
  })

  test('exports csv with summary filters only and parses filename from headers', async () => {
    const originalGet = api.get
    const calls: Array<{ url: string; params?: unknown }> = []

    api.get = (async (url: string, config?: Record<string, unknown>) => {
      calls.push({ url, params: config?.params })
      return {
        data: new Blob([Buffer.from('csv')], { type: 'text/csv' }),
        headers: {
          'content-disposition':
            'attachment; filename="usage-department-20240501-20240531.csv"',
        },
      }
    }) as typeof api.get

    try {
      const result = await exportDepartmentUsageCSV({
        from: 1714521600,
        to: 1717113600,
        tenantId: 7,
        summarySort: 'users',
        summaryOrder: 'asc',
      })

      assert.equal(result.fileName, 'usage-department-20240501-20240531.csv')
      assert.equal(calls.length, 1)
      assert.deepEqual(calls[0], {
        url: '/api/enterprise/usage/export',
        params: {
          from: 1714521600,
          to: 1717113600,
          tenant_id: 7,
          summary_sort: 'users',
          summary_order: 'asc',
        },
      })
      assert.ok(result.blob instanceof Blob)
    } finally {
      api.get = originalGet
    }
  })

  test('rejects export responses that return business-error json blobs', async () => {
    const originalGet = api.get

    api.get = (async () => {
      return {
        data: new Blob(
          [
            JSON.stringify({
              success: false,
              message: 'common.invalid_params',
            }),
          ],
          { type: 'application/json' }
        ),
        headers: {
          'content-type': 'application/json; charset=utf-8',
        },
      }
    }) as typeof api.get

    try {
      await assert.rejects(
        () =>
          exportDepartmentUsageCSV({
            from: 1714521600,
            to: 1717113600,
          }),
        /common\.invalid_params/
      )
    } finally {
      api.get = originalGet
    }
  })

  test('loads and saves usage report configuration with expected payloads', async () => {
    const originalGet = api.get
    const originalPut = api.put
    const calls: Array<{ method: string; payload: unknown }> = []

    api.get = (async (_url: string, config?: Record<string, unknown>) => {
      calls.push({ method: 'get', payload: config?.params })
      return {
        data: {
          success: true,
          message: '',
          data: {
            item: {
              id: 1,
              tenant_id: 7,
              receivers: ['ops@example.com'],
              frequency: 'weekly',
              range_type: 'last7d',
              enabled: true,
              status: 'success',
              last_run_at: 1717113600,
              next_run_at: 1717718400,
              last_success_at: 1717113600,
              last_window_start: 1716508800,
              last_window_end: 1717113600,
              run_count: 2,
              failure_count: 0,
              error_reason: '',
              last_snapshot: null,
              created_at: 1716500000,
              updated_at: 1717113600,
            },
          },
        },
      }
    }) as typeof api.get

    api.put = (async (_url: string, body?: unknown) => {
      calls.push({ method: 'put', payload: body })
      return {
        data: {
          success: true,
          message: '',
          data: {
            item: {
              id: 1,
              tenant_id: 7,
              receivers: ['ops@example.com', 'cto@example.com'],
              frequency: 'monthly',
              range_type: 'last30d',
              enabled: true,
              status: 'pending',
              last_run_at: 1717113600,
              next_run_at: 1719792000,
              last_success_at: 1717113600,
              last_window_start: 1714521600,
              last_window_end: 1717113600,
              run_count: 3,
              failure_count: 0,
              error_reason: '',
              last_snapshot: null,
              created_at: 1716500000,
              updated_at: 1717113600,
            },
          },
        },
      }
    }) as typeof api.put

    try {
      const loaded = await getDepartmentUsageReportConfig({ tenantId: 7 })
      assert.equal(loaded.data.item.frequency, 'weekly')

      const saved = await saveDepartmentUsageReportConfig({
        tenantId: 7,
        receivers: ['ops@example.com', 'cto@example.com'],
        frequency: 'monthly',
        rangeType: 'last30d',
        enabled: true,
      })
      assert.equal(saved.data.item.range_type, 'last30d')

      assert.deepEqual(calls, [
        { method: 'get', payload: { tenant_id: 7 } },
        {
          method: 'put',
          payload: {
            tenant_id: 7,
            receivers: ['ops@example.com', 'cto@example.com'],
            frequency: 'monthly',
            range_type: 'last30d',
            enabled: true,
          },
        },
      ])
    } finally {
      api.get = originalGet
      api.put = originalPut
    }
  })

  test('only resets the report form when scope changes or the form is still pristine', () => {
    assert.equal(
      shouldResetReportForm({
        hasReportData: true,
        isDirty: true,
        scopeChanged: false,
      }),
      false
    )
    assert.equal(
      shouldResetReportForm({
        hasReportData: true,
        isDirty: false,
        scopeChanged: false,
      }),
      true
    )
    assert.equal(
      shouldResetReportForm({
        hasReportData: true,
        isDirty: true,
        scopeChanged: true,
      }),
      true
    )
    assert.equal(
      shouldResetReportForm({
        hasReportData: false,
        isDirty: true,
        scopeChanged: false,
      }),
      false
    )
    assert.equal(
      shouldResetReportForm({
        hasReportData: false,
        isDirty: true,
        scopeChanged: true,
      }),
      true
    )
  })

  test('derives export params from summary filters without leaking detail-only search state', () => {
    assert.deepEqual(
      resolveDepartmentUsageExportParams(
        {
          preset: 'custom',
          from: 1714521600,
          to: 1717113600,
          tenant_id: 7,
          dept_id: 99,
          sort: 'tokens',
          summary_sort: 'users',
          summary_order: 'asc',
          log_user: 'alice',
        },
        {
          from: 1714521600,
          to: 1717113600,
        }
      ),
      {
        from: 1714521600,
        to: 1717113600,
        tenantId: 7,
        summarySort: 'users',
        summaryOrder: 'asc',
      }
    )
  })

  test('normalizes detail arrays and sorts rankings by selected metric', () => {
    const normalized = normalizeDepartmentUsageDetail(
      detailUsageItem({
        user_ranking: undefined as never,
        model_distribution: undefined as never,
        trend: undefined as never,
        recent_logs_entry: {
          path: '/usage-logs/common',
          section: 'common',
          filters: {
            department_id: 1,
            department_name: 'Engineering',
            start_timestamp: 1748476800,
            end_timestamp: 1748563199,
            username: '',
            username_options: undefined as never,
          },
        },
      })
    )

    assert.deepEqual(normalized.user_ranking, [])
    assert.deepEqual(normalized.model_distribution, [])
    assert.deepEqual(normalized.trend, [])
    assert.deepEqual(normalized.recent_logs_entry.filters.username_options, [])

    const byRequests = sortDepartmentUserRanking(
      detailUsageItem().user_ranking,
      'requests'
    )
    assert.equal(byRequests[0]?.username, 'alice')

    const byTokens = sortDepartmentUserRanking(
      [
        ...detailUsageItem().user_ranking,
        {
          user_id: 3,
          username: 'carol',
          request_count: 1,
          prompt_tokens: 100,
          completion_tokens: 200,
          token_count: 300,
          quota: 50000,
        },
      ],
      'tokens'
    )
    assert.equal(byTokens[0]?.username, 'alice')

    const quotaOrder = sortDepartmentUserRanking(
      [
        {
          user_id: 1,
          username: 'alice',
          request_count: 4,
          prompt_tokens: 400,
          completion_tokens: 160,
          token_count: 560,
          quota: 120000,
        },
        {
          user_id: 2,
          username: 'bob',
          request_count: 6,
          prompt_tokens: 260,
          completion_tokens: 120,
          token_count: 380,
          quota: 90000,
        },
      ],
      'quota'
    )
    assert.deepEqual(
      quotaOrder.map((item) => item.username),
      ['alice', 'bob']
    )

    const requestOrder = sortDepartmentUserRanking(quotaOrder, 'requests')
    assert.deepEqual(
      requestOrder.map((item) => item.username),
      ['bob', 'alice']
    )
  })

  test('renders detail drill-down with disclaimer, sorting controls and recent logs entry', () => {
    const Wrapper = () => {
      const form = useForm<{
        receivers: string
        frequency: 'daily' | 'weekly' | 'monthly'
        range_type: 'today' | 'last7d' | 'last30d'
        enabled: boolean
      }>({
        defaultValues: {
          receivers: 'ops@example.com',
          frequency: 'daily' as const,
          range_type: 'last7d' as const,
          enabled: true,
        },
      })
      return (
        <EnterpriseUsageContent
          items={[
            departmentUsageItem({ dept_id: 1, dept_name: 'Engineering' }),
          ]}
          selectedDepartmentId={1}
          detail={detailUsageItem()}
          detailLoading={false}
          detailErrorMessage={null}
          isLoading={false}
          errorMessage={null}
          rangeLabel='2026-05-28 ~ 2026-05-29'
          customRange={{
            from: '2026-05-28',
            to: '2026-05-29',
            isValid: true,
          }}
          onCustomRangeChange={() => undefined}
          onApplyCustomRange={() => undefined}
          onPresetChange={() => undefined}
          selectedPreset='today'
          onRetry={() => undefined}
          onSelectDepartment={() => undefined}
          onBackToOverview={() => undefined}
          onRetryDetail={() => undefined}
          rankSort='quota'
          onSortChange={() => undefined}
          summarySort='requests'
          summaryOrder='desc'
          onSummarySortChange={() => undefined}
          onExport={() => undefined}
          exportLoading={false}
          report={reportUsageItem()}
          reportLoading={false}
          reportErrorMessage={null}
          reportForm={form}
          onSaveReport={() => undefined}
          reportSaving={false}
          selectedLogUser='alice'
          onOpenRecentLogs={() => undefined}
        />
      )
    }

    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <Wrapper />
      </I18nextProvider>
    )

    for (const expected of [
      'Engineering',
      'Usage Trend',
      'User Ranking',
      'Sort by Quota',
      'Sort by Requests',
      'Sort by Tokens',
      'Open Recent Logs',
      'Back to overview',
      'Recent Logs User Filter',
      'Scheduled Usage Reports',
      'Save Report Configuration',
      'Inspect top users, model mix, time trend, and recent logs.',
      'alice',
      'bob',
      'claude-sonnet-4',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('renders report delivery failure state in the scheduled reports card', () => {
    const Wrapper = () => {
      const form = useForm<{
        receivers: string
        frequency: 'daily' | 'weekly' | 'monthly'
        range_type: 'today' | 'last7d' | 'last30d'
        enabled: boolean
      }>({
        defaultValues: {
          receivers: 'ops@example.com',
          frequency: 'weekly' as const,
          range_type: 'last30d' as const,
          enabled: true,
        },
      })
      return (
        <EnterpriseUsageContent
          items={[departmentUsageItem({ dept_id: 1, dept_name: 'Engineering' })]}
          selectedDepartmentId={1}
          detail={detailUsageItem()}
          detailLoading={false}
          detailErrorMessage={null}
          isLoading={false}
          errorMessage={null}
          rangeLabel='2026-05-28 ~ 2026-05-29'
          customRange={{
            from: '2026-05-28',
            to: '2026-05-29',
            isValid: true,
          }}
          onCustomRangeChange={() => undefined}
          onApplyCustomRange={() => undefined}
          onPresetChange={() => undefined}
          selectedPreset='today'
          onRetry={() => undefined}
          onSelectDepartment={() => undefined}
          onBackToOverview={() => undefined}
          onRetryDetail={() => undefined}
          rankSort='quota'
          onSortChange={() => undefined}
          summarySort='requests'
          summaryOrder='desc'
          onSummarySortChange={() => undefined}
          onExport={() => undefined}
          exportLoading={false}
          report={reportUsageItem({
            status: 'failed',
            failure_count: 2,
            error_reason: 'smtp down',
          })}
          reportLoading={false}
          reportErrorMessage={null}
          reportForm={form}
          onSaveReport={() => undefined}
          reportSaving={false}
          selectedLogUser='alice'
          onOpenRecentLogs={() => undefined}
        />
      )
    }

    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <Wrapper />
      </I18nextProvider>
    )

    for (const expected of [
      'Scheduled Usage Reports',
      'Last Delivery Error',
      'smtp down',
      'failed',
      '2',
      'Save Report Configuration',
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('preserves department context when opening recent logs filters', () => {
    const params = buildSearchParams(
      {
        startTime: new Date(1748476800000),
        endTime: new Date(1748563200000),
        username: 'alice',
        departmentContext: {
          departmentId: 1,
          departmentName: 'Engineering',
        },
      },
      'common'
    )

    assert.equal(params.departmentId, 1)
    assert.equal(params.departmentName, 'Engineering')
    assert.equal(params.username, 'alice')
  })

  test('prefers the selected log user when present and falls back to the first available username', () => {
    const entry = detailUsageItem().recent_logs_entry

    assert.deepEqual(resolveRecentLogsSearch(entry, 'bob'), {
      departmentId: 1,
      departmentName: 'Engineering',
      startTime: 1748476800000,
      endTime: 1748563200000,
      username: 'bob',
    })

    assert.deepEqual(resolveRecentLogsSearch(entry, 'carol'), {
      departmentId: 1,
      departmentName: 'Engineering',
      startTime: 1748476800000,
      endTime: 1748563200000,
      username: 'alice',
    })

    assert.deepEqual(resolveRecentLogsSearch(entry), {
      departmentId: 1,
      departmentName: 'Engineering',
      startTime: 1748476800000,
      endTime: 1748563200000,
      username: 'alice',
    })
  })
})

function departmentUsageItem(
  overrides: Partial<DepartmentUsageSummaryItem>
): DepartmentUsageSummaryItem {
  return {
    dept_id: 1,
    dept_name: 'Engineering',
    window_start: 1748476800,
    window_end: 1748563199,
    request_count: 10,
    prompt_tokens: 1000,
    completion_tokens: 500,
    quota: 250000,
    user_count: 2,
    model_distribution: [],
    ...overrides,
  }
}

function detailUsageItem(
  overrides: Partial<DepartmentUsageDetailResponse> = {}
): DepartmentUsageDetailResponse {
  return {
    dept_id: 1,
    dept_name: 'Engineering',
    window_start: 1748476800,
    window_end: 1748563200,
    request_count: 6,
    prompt_tokens: 600,
    completion_tokens: 240,
    token_count: 840,
    quota: 180000,
    user_count: 2,
    user_ranking: [
      {
        user_id: 1,
        username: 'alice',
        request_count: 4,
        prompt_tokens: 400,
        completion_tokens: 160,
        token_count: 560,
        quota: 120000,
      },
      {
        user_id: 2,
        username: 'bob',
        request_count: 2,
        prompt_tokens: 200,
        completion_tokens: 80,
        token_count: 280,
        quota: 60000,
      },
    ],
    model_distribution: [
      {
        model_name: 'gpt-4o',
        request_count: 4,
        prompt_tokens: 400,
        completion_tokens: 160,
        quota: 120000,
      },
      {
        model_name: 'claude-sonnet-4',
        request_count: 2,
        prompt_tokens: 200,
        completion_tokens: 80,
        quota: 60000,
      },
    ],
    trend: [
      {
        window_start: 1748476800,
        window_end: 1748480400,
        request_count: 3,
        prompt_tokens: 300,
        completion_tokens: 120,
        token_count: 420,
        quota: 90000,
        user_count: 2,
      },
      {
        window_start: 1748480400,
        window_end: 1748484000,
        request_count: 3,
        prompt_tokens: 300,
        completion_tokens: 120,
        token_count: 420,
        quota: 90000,
        user_count: 1,
      },
    ],
    recent_logs_entry: {
      path: '/usage-logs/common',
      section: 'common',
      filters: {
        department_id: 1,
        department_name: 'Engineering',
        start_timestamp: 1748476800,
        end_timestamp: 1748563199,
        username: '',
        username_options: ['alice', 'bob'],
      },
    },
    ...overrides,
  }
}

function reportUsageItem(
  overrides: Partial<DepartmentUsageReportJobItem> = {}
): DepartmentUsageReportJobItem {
  return {
    id: 1,
    tenant_id: 0,
    receivers: ['ops@example.com'],
    frequency: 'daily',
    range_type: 'last7d',
    enabled: true,
    status: 'success',
    last_run_at: 1748563200,
    next_run_at: 1748649600,
    last_success_at: 1748563200,
    last_window_start: 1747964800,
    last_window_end: 1748563200,
    run_count: 2,
    failure_count: 0,
    error_reason: '',
    last_snapshot: null,
    created_at: 1747964800,
    updated_at: 1748563200,
    ...overrides,
  }
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
