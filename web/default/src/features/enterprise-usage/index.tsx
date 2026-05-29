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
import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearch } from '@tanstack/react-router'
import {
  AlertTriangle,
  BarChart3,
  Coins,
  RefreshCw,
  Rows3,
  Users,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import z from 'zod'
import { formatDateStr, formatNumber, formatQuota } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import dayjs from '@/lib/dayjs'
import { useAuthStore } from '@/stores/auth-store'
import { SectionPageLayout } from '@/components/layout'
import { PageTransition } from '@/components/page-transition'
import { StatCard } from '@/features/dashboard/components/ui/stat-card'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useDepartmentUsageSummary } from './hooks/use-department-usage-summary'
import type {
  DepartmentUsageSummaryItem,
  EnterpriseUsagePreset,
  EnterpriseUsageSearch,
  UsageModelDistributionItem,
} from './types'

const ENTERPRISE_USAGE_PRESETS = [
  'today',
  'yesterday',
  'last7d',
  'last30d',
  'custom',
] as const satisfies readonly EnterpriseUsagePreset[]

export const enterpriseUsageSearchSchema = z.object({
  preset: z.enum(ENTERPRISE_USAGE_PRESETS).optional().catch('today'),
  from: z.coerce.number().int().optional().catch(undefined),
  to: z.coerce.number().int().optional().catch(undefined),
  tenant_id: z.coerce.number().int().nonnegative().optional().catch(undefined),
})

type ResolvedEnterpriseUsageRange = {
  preset: EnterpriseUsagePreset
  from: number
  to: number
  rangeLabel: string
  isCustom: boolean
  isRangeValid: boolean
  customFromDate: string
  customToDate: string
}

export function resolveEnterpriseUsageRange(
  search: EnterpriseUsageSearch,
  now: Date = new Date()
): ResolvedEnterpriseUsageRange {
  const current = dayjs(now)

  const today = {
    preset: 'today' as const,
    from: current.startOf('day').unix(),
    to: current.add(1, 'day').startOf('day').unix(),
  }
  const presets: Record<
    Exclude<EnterpriseUsagePreset, 'custom'>,
    { from: number; to: number }
  > = {
    today,
    yesterday: {
      from: current.subtract(1, 'day').startOf('day').unix(),
      to: current.startOf('day').unix(),
    },
    last7d: {
      from: current.subtract(6, 'day').startOf('day').unix(),
      to: current.add(1, 'day').startOf('day').unix(),
    },
    last30d: {
      from: current.subtract(29, 'day').startOf('day').unix(),
      to: current.add(1, 'day').startOf('day').unix(),
    },
  }

  const preset = search.preset ?? 'today'
  if (preset === 'custom') {
    const from = search.from
    const to = search.to
    if (
      typeof from === 'number' &&
      Number.isFinite(from) &&
      typeof to === 'number' &&
      Number.isFinite(to) &&
      from < to
    ) {
      return {
        preset,
        from,
        to,
        rangeLabel: buildRangeLabel(from, to),
        isCustom: true,
        isRangeValid: true,
        customFromDate: formatDateInputValue(from),
        customToDate: formatRangeEndInputValue(to),
      }
    }
  }

  const effectivePreset = preset === 'custom' ? 'today' : preset
  const selected = presets[effectivePreset]
  return {
    preset: effectivePreset,
    from: selected.from,
    to: selected.to,
    rangeLabel: buildRangeLabel(selected.from, selected.to),
    isCustom: false,
    isRangeValid: true,
    customFromDate:
      preset === 'custom' && search.from
        ? formatDateInputValue(search.from)
        : formatDateInputValue(selected.from),
    customToDate:
      preset === 'custom' && search.to
        ? formatRangeEndInputValue(search.to)
        : formatRangeEndInputValue(selected.to),
  }
}

export function formatModelDistributionSummary(
  items: UsageModelDistributionItem[],
  t: (key: string, options?: Record<string, unknown>) => string
): string {
  if (items.length === 0) return '-'

  const topItems = [...items]
    .sort((a, b) => {
      if (b.request_count !== a.request_count) {
        return b.request_count - a.request_count
      }
      return b.quota - a.quota
    })
    .slice(0, 3)
    .map(
      (item) =>
        `${item.model_name} (${formatNumber(item.request_count)} / ${formatQuota(
          item.quota
        )})`
    )

  const remaining = items.length - topItems.length
  if (remaining > 0) {
    topItems.push(t('enterprise.usage.more_models', { count: remaining }))
  }
  return topItems.join(', ')
}

export function EnterpriseUsageOverview() {
  const { t } = useTranslation()
  const search = useSearch({
    from: '/_authenticated/enterprise-usage/',
  }) as EnterpriseUsageSearch
  const navigate = useNavigate()
  const { auth } = useAuthStore()
  const resolvedRange = useMemo(
    () => resolveEnterpriseUsageRange(search),
    [search]
  )
  const [customFromDate, setCustomFromDate] = useState(
    resolvedRange.customFromDate
  )
  const [customToDate, setCustomToDate] = useState(resolvedRange.customToDate)

  useEffect(() => {
    setCustomFromDate(resolvedRange.customFromDate)
    setCustomToDate(resolvedRange.customToDate)
  }, [resolvedRange.customFromDate, resolvedRange.customToDate])

  const usageQuery = useDepartmentUsageSummary({
    from: resolvedRange.from,
    to: resolvedRange.to,
    tenantId: search.tenant_id,
  })

  const customRangeState = getCustomRangeState(customFromDate, customToDate)

  const handlePresetChange = (preset: EnterpriseUsagePreset) => {
    if (preset === 'custom') {
      navigate({
        to: '/enterprise-usage',
        search: (prev) => ({
          ...prev,
          preset: 'custom',
          from: resolvedRange.from,
          to: resolvedRange.to,
        }),
      })
      return
    }

    const nextRange = resolveEnterpriseUsageRange({ preset })
    setCustomFromDate(formatDateInputValue(nextRange.from))
    setCustomToDate(formatDateInputValue(nextRange.to))
    navigate({
      to: '/enterprise-usage',
      search: { preset },
    })
  }

  const handleApplyCustomRange = () => {
    if (!customRangeState.isValid) return
    navigate({
      to: '/enterprise-usage',
      search: {
        preset: 'custom',
        from: customRangeState.from,
        to: customRangeState.to,
      },
    })
  }

  const isAdmin = (auth.user?.role ?? 0) >= ROLE.ADMIN

  return (
    <PageTransition className='flex min-h-0 flex-1 flex-col'>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Department Usage Overview')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <Button
            variant='outline'
            size='sm'
            onClick={() => usageQuery.refetch()}
            disabled={usageQuery.isFetching}
          >
            <RefreshCw className='size-4' />
            {t('Refresh')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          {!isAdmin ? (
            <Alert variant='destructive'>
              <AlertTriangle className='size-4' />
              <AlertTitle>{t('Admin access required')}</AlertTitle>
              <AlertDescription>
                {t('Only enterprise administrators can review department usage dashboards.')}
              </AlertDescription>
            </Alert>
          ) : (
            <EnterpriseUsageContent
              items={usageQuery.data ?? []}
              isLoading={usageQuery.isLoading}
              errorMessage={
                usageQuery.error instanceof Error
                  ? usageQuery.error.message
                  : null
              }
              rangeLabel={resolvedRange.rangeLabel}
              customRange={{
                from: customFromDate,
                to: customToDate,
                isValid: customRangeState.isValid,
              }}
              onCustomRangeChange={(next) => {
                setCustomFromDate(next.from)
                setCustomToDate(next.to)
              }}
              onApplyCustomRange={handleApplyCustomRange}
              onPresetChange={handlePresetChange}
              selectedPreset={
                resolvedRange.isCustom ? 'custom' : resolvedRange.preset
              }
              onRetry={() => usageQuery.refetch()}
            />
          )}
        </SectionPageLayout.Content>
      </SectionPageLayout>
    </PageTransition>
  )
}

type EnterpriseUsageContentProps = {
  items: DepartmentUsageSummaryItem[]
  isLoading: boolean
  errorMessage: string | null
  rangeLabel: string
  customRange: {
    from: string
    to: string
    isValid: boolean
  }
  onCustomRangeChange: (next: { from: string; to: string }) => void
  onApplyCustomRange: () => void
  onPresetChange: (preset: EnterpriseUsagePreset) => void
  selectedPreset: EnterpriseUsagePreset
  onRetry: () => void
}

export function EnterpriseUsageContent(props: EnterpriseUsageContentProps) {
  const { t } = useTranslation()

  return (
    <div className='space-y-4'>
      <Alert>
        <BarChart3 className='size-4' />
        <AlertTitle>{t('Department totals are non-additive')}</AlertTitle>
        <AlertDescription>
          {t('enterprise.usage.multi_dept_disclaimer')}
        </AlertDescription>
      </Alert>

      <Card>
        <CardHeader className='gap-3'>
          <div className='flex flex-col gap-2 lg:flex-row lg:items-start lg:justify-between'>
            <div className='space-y-1'>
              <CardTitle>{t('Enterprise Usage Overview')}</CardTitle>
              <CardDescription>{props.rangeLabel}</CardDescription>
            </div>
            <div className='flex flex-wrap gap-2'>
              {(
                [
                  ['today', t('Today')],
                  ['yesterday', t('Yesterday')],
                  ['last7d', t('Last 7 Days')],
                  ['last30d', t('Last 30 Days')],
                  ['custom', t('Custom')],
                ] as const
              ).map(([preset, label]) => (
                <Button
                  key={preset}
                  type='button'
                  size='sm'
                  variant={
                    props.selectedPreset === preset ? 'default' : 'outline'
                  }
                  onClick={() => props.onPresetChange(preset)}
                >
                  {label}
                </Button>
              ))}
            </div>
          </div>

          <div className='grid gap-2 lg:grid-cols-[1fr_1fr_auto]'>
            <Input
              type='date'
              value={props.customRange.from}
              onChange={(e) =>
                props.onCustomRangeChange({
                  from: e.target.value,
                  to: props.customRange.to,
                })
              }
            />
            <Input
              type='date'
              value={props.customRange.to}
              onChange={(e) =>
                props.onCustomRangeChange({
                  from: props.customRange.from,
                  to: e.target.value,
                })
              }
            />
            <Button
              type='button'
              onClick={props.onApplyCustomRange}
              disabled={!props.customRange.isValid}
            >
              {t('Apply')}
            </Button>
          </div>
          {!props.customRange.isValid ? (
            <Alert variant='destructive'>
              <AlertDescription>
                {t('Start time must be earlier than end time for a custom range')}
              </AlertDescription>
            </Alert>
          ) : null}
        </CardHeader>
        <CardContent>
          {props.isLoading ? (
            <EnterpriseUsageSkeleton />
          ) : props.errorMessage ? (
            <Alert variant='destructive' className='gap-2'>
              <AlertTriangle className='size-4' />
              <AlertTitle>{t('Unable to load department usage')}</AlertTitle>
                <AlertDescription>{t(props.errorMessage)}</AlertDescription>
              <div className='pt-2'>
                <Button variant='outline' size='sm' onClick={props.onRetry}>
                  {t('Retry')}
                </Button>
              </div>
            </Alert>
          ) : props.items.length === 0 ? (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant='icon'>
                  <BarChart3 className='size-5' />
                </EmptyMedia>
                <EmptyTitle>
                  {t('No department usage data for this time range')}
                </EmptyTitle>
                <EmptyDescription>
                  {t('Usage snapshots are empty for the selected time window.')}
                </EmptyDescription>
              </EmptyHeader>
              <EmptyContent>
                <Button variant='outline' size='sm' onClick={props.onRetry}>
                  {t('Retry')}
                </Button>
              </EmptyContent>
              </Empty>
          ) : (
            <div className='space-y-4'>
              <EnterpriseUsageSummaryCards items={props.items} />
              <div className='overflow-x-auto'>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('Department')}</TableHead>
                      <TableHead>{t('Requests')}</TableHead>
                      <TableHead>{t('Prompt Tokens')}</TableHead>
                      <TableHead>{t('Completion Tokens')}</TableHead>
                      <TableHead>{t('Quota')}</TableHead>
                      <TableHead>{t('Users')}</TableHead>
                      <TableHead>{t('Model Distribution')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {props.items.map((item, index) => (
                      <TableRow key={`${item.dept_id ?? 'unassigned'}-${index}`}>
                        <TableCell className='font-medium'>
                          {item.dept_name || t('Unassigned')}
                        </TableCell>
                        <TableCell>{formatNumber(item.request_count)}</TableCell>
                        <TableCell>{formatNumber(item.prompt_tokens)}</TableCell>
                        <TableCell>
                          {formatNumber(item.completion_tokens)}
                        </TableCell>
                        <TableCell>{formatQuota(item.quota)}</TableCell>
                        <TableCell>{formatNumber(item.user_count)}</TableCell>
                        <TableCell className='max-w-[340px] whitespace-normal text-sm'>
                          {formatModelDistributionSummary(
                            item.model_distribution,
                            t
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function EnterpriseUsageSkeleton() {
  return (
    <div className='space-y-3'>
      <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
        <Skeleton className='h-32 w-full' />
        <Skeleton className='h-32 w-full' />
        <Skeleton className='h-32 w-full' />
        <Skeleton className='h-32 w-full' />
      </div>
      <Skeleton className='h-10 w-full' />
      <Skeleton className='h-10 w-full' />
      <Skeleton className='h-10 w-full' />
      <Skeleton className='h-10 w-full' />
    </div>
  )
}

function EnterpriseUsageSummaryCards(props: {
  items: DepartmentUsageSummaryItem[]
}) {
  const { t } = useTranslation()
  const summary = useMemo(() => {
    const modelCounts = new Map<string, number>()
    let requestCount = 0
    let promptTokens = 0
    let completionTokens = 0
    let quota = 0
    let userCount = 0

    for (const item of props.items) {
      requestCount += item.request_count
      promptTokens += item.prompt_tokens
      completionTokens += item.completion_tokens
      quota += item.quota
      userCount += item.user_count

      for (const model of item.model_distribution) {
        modelCounts.set(
          model.model_name,
          (modelCounts.get(model.model_name) ?? 0) + model.request_count
        )
      }
    }

    const topModel = [...modelCounts.entries()].sort((a, b) => b[1] - a[1])[0]

    return {
      requestCount,
      promptTokens,
      completionTokens,
      quota,
      userCount,
      departmentCount: props.items.length,
      modelCount: modelCounts.size,
      topModelName: topModel?.[0] ?? '-',
      topModelRequests: topModel?.[1] ?? 0,
      unassignedCount: props.items.filter((item) => item.dept_id == null).length,
    }
  }, [props.items])

  return (
    <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
      <Card>
        <CardContent className='pt-6'>
          <StatCard
            title={t('Requests')}
            value={formatNumber(summary.requestCount)}
            description={t('Department Usage Overview')}
            details={[
              {
                label: t('Departments'),
                value: formatNumber(summary.departmentCount),
              },
              {
                label: t('Users'),
                value: formatNumber(summary.userCount),
              },
            ]}
            icon={Rows3}
            tone='teal'
          />
        </CardContent>
      </Card>
      <Card>
        <CardContent className='pt-6'>
          <StatCard
            title={t('Prompt Tokens')}
            value={formatNumber(summary.promptTokens)}
            description={t('Completion Tokens')}
            details={[
              {
                label: t('Completion Tokens'),
                value: formatNumber(summary.completionTokens),
              },
              {
                label: t('Quota'),
                value: formatQuota(summary.quota),
              },
            ]}
            icon={Coins}
            tone='rose'
          />
        </CardContent>
      </Card>
      <Card>
        <CardContent className='pt-6'>
          <StatCard
            title={t('Models')}
            value={formatNumber(summary.modelCount)}
            description={t('Model Distribution')}
            details={[
              {
                label: t('Top Model'),
                value:
                  summary.topModelName === '-'
                    ? '-'
                    : `${summary.topModelName} (${formatNumber(
                        summary.topModelRequests
                      )})`,
              },
              {
                label: t('Unassigned'),
                value: formatNumber(summary.unassignedCount),
              },
            ]}
            icon={BarChart3}
            tone='gray'
          />
        </CardContent>
      </Card>
      <Card>
        <CardContent className='pt-6'>
          <StatCard
            title={t('Users')}
            value={formatNumber(summary.userCount)}
            description={t('Department totals are non-additive')}
            details={[
              {
                label: t('Requests'),
                value: formatNumber(summary.requestCount),
              },
              {
                label: t('Quota'),
                value: formatQuota(summary.quota),
              },
            ]}
            icon={Users}
            tone='teal'
          />
        </CardContent>
      </Card>
    </div>
  )
}

export function normalizeDepartmentUsageItems(
  items: DepartmentUsageSummaryItem[]
): DepartmentUsageSummaryItem[] {
  return [...items]
    .map((item) => ({
      ...item,
      dept_name: item.dept_name || '',
      model_distribution: item.model_distribution ?? [],
    }))
    .sort((a, b) => {
      if (b.request_count !== a.request_count) {
        return b.request_count - a.request_count
      }
      if (b.quota !== a.quota) {
        return b.quota - a.quota
      }
      return (a.dept_name || 'Unassigned').localeCompare(
        b.dept_name || 'Unassigned'
      )
    })
}

function getCustomRangeState(fromDate: string, toDate: string) {
  const from = parseDateInputToUnix(fromDate, false)
  const to = parseDateInputToUnix(toDate, false, 1)

  return {
    from,
    to,
    isValid:
      typeof from === 'number' &&
      typeof to === 'number' &&
      Number.isFinite(from) &&
      Number.isFinite(to) &&
      from < to,
  }
}

function parseDateInputToUnix(
  value: string,
  endOfDay: boolean,
  dayOffset = 0
): number | null {
  if (!value) return null
  const parsed = dayjs(value)
  if (!parsed.isValid()) return null
  const boundary = endOfDay ? parsed.endOf('day') : parsed.startOf('day')
  return boundary.add(dayOffset, 'day').unix()
}

function formatDateInputValue(unixSeconds: number): string {
  return dayjs(unixSeconds * 1000).format('YYYY-MM-DD')
}

function formatRangeEndInputValue(unixSeconds: number): string {
  return dayjs(unixSeconds * 1000).subtract(1, 'day').format('YYYY-MM-DD')
}

function buildRangeLabel(from: number, to: number): string {
  return `${formatDateStr(new Date(from * 1000))} ~ ${formatDateStr(
    new Date(dayjs(to * 1000).subtract(1, 'second').valueOf())
  )}`
}
