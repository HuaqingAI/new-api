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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { isRedirect } from '@tanstack/react-router'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'
import i18n from '@/i18n/config'
import { ROLE } from '@/lib/roles'
import { Route as EnterpriseUsageRoute } from '@/routes/_authenticated/enterprise-usage/index'
import { useAuthStore } from '@/stores/auth-store'
import { departmentSummaryQueryKey } from './api'
import {
  EnterpriseUsageContent,
  enterpriseUsageSearchSchema,
  formatModelDistributionSummary,
  normalizeDepartmentUsageItems,
  resolveEnterpriseUsageRange,
} from './index'
import type { DepartmentUsageSummaryItem } from './types'

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
    assert.deepEqual(
      departmentSummaryQueryKey(1748390400, 1748476799),
      [
        'enterprise',
        'usage',
        'department-summary',
        { from: 1748390400, to: 1748476799 },
      ]
    )

    assert.deepEqual(
      departmentSummaryQueryKey(1748390400, 1748476800, 2),
      [
        'enterprise',
        'usage',
        'department-summary',
        { from: 1748390400, to: 1748476800, tenantId: 2 },
      ]
    )
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
    ]) {
      assert.match(html, new RegExp(escapeRegExp(expected)))
    }
  })

  test('renders empty and error states with retry affordance', () => {
    const emptyHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <EnterpriseUsageContent
          items={[]}
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
        />
      </I18nextProvider>
    )
    assert.match(emptyHtml, /No department usage data for this time range/)
    assert.match(emptyHtml, /Enterprise Usage Overview/)

    const errorHtml = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <EnterpriseUsageContent
          items={[]}
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

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
