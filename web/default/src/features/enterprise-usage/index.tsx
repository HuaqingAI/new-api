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
  ArrowLeft,
  BarChart3,
  Coins,
  ExternalLink,
  LineChart,
  RefreshCw,
  Rows3,
  Users,
} from 'lucide-react'
import {
  CartesianGrid,
  Line,
  LineChart as RechartsLineChart,
  XAxis,
  YAxis,
} from 'recharts'
import { useTranslation } from 'react-i18next'
import z from 'zod'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
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
import { useDepartmentUsageDetail } from './hooks/use-department-usage-detail'
import { useDepartmentUsageSummary } from './hooks/use-department-usage-summary'
import type {
  DepartmentUsageDetailResponse,
  DepartmentUsageLogEntryLink,
  DepartmentUsageSummaryItem,
  DepartmentUsageUserRankItem,
  DepartmentUsageUserRankSort,
  EnterpriseUsagePreset,
  EnterpriseUsageSearch,
  DepartmentUsageTrendPoint,
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
  dept_id: z.coerce.number().int().positive().optional().catch(undefined),
  sort: z.enum(['quota', 'requests', 'tokens']).optional().catch('quota'),
  log_user: z.string().optional().catch(undefined),
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
  const detailQuery = useDepartmentUsageDetail({
    deptId: search.dept_id,
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
      search: (prev) => ({
        ...prev,
        preset,
        from: undefined,
        to: undefined,
      }),
    })
  }

  const handleApplyCustomRange = () => {
    if (!customRangeState.isValid) return
    navigate({
      to: '/enterprise-usage',
      search: (prev) => ({
        ...prev,
        preset: 'custom',
        from: customRangeState.from,
        to: customRangeState.to,
      }),
    })
  }

  const handleSelectDepartment = (deptId: number | null) => {
    if (deptId == null) return
    const nextSort: DepartmentUsageUserRankSort =
      search.sort === 'requests' || search.sort === 'tokens'
        ? search.sort
        : 'quota'
    navigate({
      to: '/enterprise-usage',
      search: (prev) => ({
        ...prev,
        dept_id: deptId,
        sort: nextSort,
        log_user: undefined,
      }),
    })
  }

  const handleBackToOverview = () => {
    navigate({
      to: '/enterprise-usage',
      search: (prev) => ({
        ...prev,
        dept_id: undefined,
        log_user: undefined,
      }),
    })
  }

  const handleSortChange = (sort: DepartmentUsageUserRankSort) => {
    navigate({
      to: '/enterprise-usage',
      search: (prev) => ({
        ...prev,
        sort,
      }),
    })
  }

  const handleOpenRecentLogs = (entry: DepartmentUsageLogEntryLink) => {
    void navigate({
      to: '/usage-logs/$section',
      params: { section: entry.section as 'common' },
      search: resolveRecentLogsSearch(entry, search.log_user),
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
              selectedDepartmentId={search.dept_id}
              detail={detailQuery.data ?? null}
              detailLoading={detailQuery.isLoading}
              detailErrorMessage={
                detailQuery.error instanceof Error
                  ? detailQuery.error.message
                  : null
              }
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
              onSelectDepartment={handleSelectDepartment}
              onBackToOverview={handleBackToOverview}
              onRetryDetail={() => detailQuery.refetch()}
              rankSort={search.sort ?? 'quota'}
              onSortChange={handleSortChange}
              selectedLogUser={search.log_user}
              onOpenRecentLogs={handleOpenRecentLogs}
            />
          )}
        </SectionPageLayout.Content>
      </SectionPageLayout>
    </PageTransition>
  )
}

type EnterpriseUsageContentProps = {
  items: DepartmentUsageSummaryItem[]
  selectedDepartmentId?: number
  detail: DepartmentUsageDetailResponse | null
  detailLoading: boolean
  detailErrorMessage: string | null
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
  onSelectDepartment: (deptId: number | null) => void
  onBackToOverview: () => void
  onRetryDetail: () => void
  rankSort: DepartmentUsageUserRankSort
  onSortChange: (sort: DepartmentUsageUserRankSort) => void
  selectedLogUser?: string
  onOpenRecentLogs: (entry: DepartmentUsageLogEntryLink) => void
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
                      <TableHead>{t('Actions')}</TableHead>
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
                        <TableCell>
                          <Button
                            type='button'
                            size='sm'
                            variant={
                              props.selectedDepartmentId === item.dept_id
                                ? 'default'
                                : 'outline'
                            }
                            disabled={item.dept_id == null}
                            onClick={() => props.onSelectDepartment(item.dept_id)}
                          >
                            {t('View Details')}
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
              {props.selectedDepartmentId ? (
                <DepartmentUsageDetailPanel
                  detail={props.detail}
                  isLoading={props.detailLoading}
                  errorMessage={props.detailErrorMessage}
                  rankSort={props.rankSort}
                  onSortChange={props.onSortChange}
                  onBack={props.onBackToOverview}
                  onRetry={props.onRetryDetail}
                  selectedLogUser={props.selectedLogUser}
                  onOpenRecentLogs={props.onOpenRecentLogs}
                />
              ) : null}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

const detailTrendChartConfig = {
  quota: {
    label: 'Quota',
    color: 'var(--color-chart-1)',
  },
  requests: {
    label: 'Requests',
    color: 'var(--color-chart-2)',
  },
} satisfies ChartConfig

function DepartmentUsageDetailPanel(props: {
  detail: DepartmentUsageDetailResponse | null
  isLoading: boolean
  errorMessage: string | null
  rankSort: DepartmentUsageUserRankSort
  onSortChange: (sort: DepartmentUsageUserRankSort) => void
  onBack: () => void
  onRetry: () => void
  selectedLogUser?: string
  onOpenRecentLogs: (entry: DepartmentUsageLogEntryLink) => void
}) {
  const { t } = useTranslation()

  return (
    <Card className='border-dashed'>
      <CardHeader className='gap-3'>
        <div className='flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between'>
          <div className='space-y-1'>
            <Button
              type='button'
              variant='ghost'
              size='sm'
              className='-ml-3 w-fit'
              onClick={props.onBack}
            >
              <ArrowLeft className='size-4' />
              {t('Back to overview')}
            </Button>
            <CardTitle>
              {props.detail?.dept_name || t('Department Usage Details')}
            </CardTitle>
            <CardDescription>
              {props.detail?.dept_name
                ? t('Inspect top users, model mix, time trend, and recent logs.')
                : t('Select a department to inspect drill-down usage details.')}
            </CardDescription>
          </div>
          {props.detail ? (
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() => props.onOpenRecentLogs(props.detail!.recent_logs_entry)}
            >
              <ExternalLink className='size-4' />
              {t('Open Recent Logs')}
            </Button>
          ) : null}
        </div>
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
        ) : props.detail == null ? (
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <LineChart className='size-5' />
              </EmptyMedia>
              <EmptyTitle>{t('Department Usage Details')}</EmptyTitle>
              <EmptyDescription>
                {t('Select a department to inspect drill-down usage details.')}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className='space-y-4'>
            <EnterpriseUsageSummaryCards
              items={[
                {
                  dept_id: props.detail.dept_id,
                  dept_name: props.detail.dept_name,
                  window_start: props.detail.window_start,
                  window_end: props.detail.window_end,
                  request_count: props.detail.request_count,
                  prompt_tokens: props.detail.prompt_tokens,
                  completion_tokens: props.detail.completion_tokens,
                  quota: props.detail.quota,
                  user_count: props.detail.user_count,
                  model_distribution: props.detail.model_distribution,
                },
              ]}
            />
            <div className='grid gap-4 xl:grid-cols-[1.1fr_0.9fr]'>
              <Card>
                <CardHeader>
                  <CardTitle>{t('Usage Trend')}</CardTitle>
                  <CardDescription>
                    {t('Hourly snapshot trend for the selected department.')}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <DepartmentUsageTrendChart trend={props.detail.trend} />
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle>{t('Model Distribution')}</CardTitle>
                  <CardDescription>
                    {t('Models ranked by quota and request count inside this department.')}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <DepartmentModelDistributionTable
                    items={props.detail.model_distribution}
                  />
                </CardContent>
              </Card>
            </div>
            <Card>
              <CardHeader className='gap-3'>
                <div className='flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between'>
                  <div>
                    <CardTitle>{t('User Ranking')}</CardTitle>
                    <CardDescription>
                      {t('Ranking is calculated only within the current department scope.')}
                    </CardDescription>
                  </div>
                  <div className='flex flex-wrap gap-2'>
                    {(
                      [
                        ['quota', t('Sort by Quota')],
                        ['requests', t('Sort by Requests')],
                        ['tokens', t('Sort by Tokens')],
                      ] as const
                    ).map(([sort, label]) => (
                      <Button
                        key={sort}
                        type='button'
                        size='sm'
                        variant={props.rankSort === sort ? 'default' : 'outline'}
                        onClick={() => props.onSortChange(sort)}
                      >
                        {label}
                      </Button>
                    ))}
                  </div>
                </div>
              </CardHeader>
              <CardContent className='space-y-3'>
                <DepartmentUserRankingTable
                  items={sortDepartmentUserRanking(
                    props.detail.user_ranking,
                    props.rankSort
                  )}
                />
                <div className='flex flex-wrap gap-2 text-xs text-muted-foreground'>
                  <span>{t('Recent Logs User Filter')}:</span>
                  {props.detail.recent_logs_entry.filters.username_options.map(
                    (username) => (
                      <span
                        key={username}
                        className={
                          username === props.selectedLogUser
                            ? 'rounded-full bg-primary/10 px-2 py-1 text-primary'
                            : 'rounded-full bg-muted px-2 py-1'
                        }
                      >
                        {username}
                      </span>
                    )
                  )}
                </div>
              </CardContent>
            </Card>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function DepartmentUsageTrendChart(props: {
  trend: DepartmentUsageTrendPoint[]
}) {
  const { t } = useTranslation()
  const data = props.trend.map((point) => ({
    label: dayjs(point.window_start * 1000).format('MM-DD HH:mm'),
    quota: point.quota,
    requests: point.request_count,
  }))

  if (data.length === 0) {
    return <div className='text-sm text-muted-foreground'>{t('No trend data for this time range')}</div>
  }

  return (
    <ChartContainer config={detailTrendChartConfig} className='h-72 w-full'>
      <RechartsLineChart accessibilityLayer data={data}>
        <CartesianGrid vertical={false} />
        <XAxis dataKey='label' tickLine={false} axisLine={false} minTickGap={24} />
        <YAxis yAxisId='left' tickLine={false} axisLine={false} width={48} />
        <YAxis
          yAxisId='right'
          orientation='right'
          tickLine={false}
          axisLine={false}
          width={48}
        />
        <ChartTooltip content={<ChartTooltipContent />} />
        <Line
          yAxisId='left'
          type='monotone'
          dataKey='quota'
          stroke='var(--color-quota)'
          strokeWidth={2}
          dot={false}
        />
        <Line
          yAxisId='right'
          type='monotone'
          dataKey='requests'
          stroke='var(--color-requests)'
          strokeWidth={2}
          dot={false}
        />
      </RechartsLineChart>
    </ChartContainer>
  )
}

function DepartmentModelDistributionTable(props: {
  items: UsageModelDistributionItem[]
}) {
  const { t } = useTranslation()

  return (
    <div className='overflow-x-auto'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Model')}</TableHead>
            <TableHead>{t('Requests')}</TableHead>
            <TableHead>{t('Quota')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {props.items.length === 0 ? (
            <TableRow>
              <TableCell colSpan={3} className='text-center text-muted-foreground'>
                {t('No model distribution for this time range')}
              </TableCell>
            </TableRow>
          ) : (
            props.items.map((item) => (
              <TableRow key={item.model_name}>
                <TableCell className='font-medium'>{item.model_name}</TableCell>
                <TableCell>{formatNumber(item.request_count)}</TableCell>
                <TableCell>{formatQuota(item.quota)}</TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  )
}

function DepartmentUserRankingTable(props: {
  items: DepartmentUsageUserRankItem[]
}) {
  const { t } = useTranslation()

  return (
    <div className='overflow-x-auto'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('User')}</TableHead>
            <TableHead>{t('Requests')}</TableHead>
            <TableHead>{t('Tokens')}</TableHead>
            <TableHead>{t('Quota')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {props.items.length === 0 ? (
            <TableRow>
              <TableCell colSpan={4} className='text-center text-muted-foreground'>
                {t('No department members consumed within this time range')}
              </TableCell>
            </TableRow>
          ) : (
            props.items.map((item) => (
              <TableRow key={item.user_id}>
                <TableCell className='font-medium'>{item.username || item.user_id}</TableCell>
                <TableCell>{formatNumber(item.request_count)}</TableCell>
                <TableCell>{formatNumber(item.token_count)}</TableCell>
                <TableCell>{formatQuota(item.quota)}</TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
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

export function normalizeDepartmentUsageDetail(
  detail: DepartmentUsageDetailResponse
): DepartmentUsageDetailResponse {
  return {
    ...detail,
    dept_name: detail.dept_name || '',
    user_ranking: detail.user_ranking ?? [],
    model_distribution: detail.model_distribution ?? [],
    trend: detail.trend ?? [],
    recent_logs_entry: {
      ...detail.recent_logs_entry,
      filters: {
        ...detail.recent_logs_entry.filters,
        department_id: detail.recent_logs_entry.filters.department_id ?? null,
        username: detail.recent_logs_entry.filters.username ?? '',
        username_options:
          detail.recent_logs_entry.filters.username_options ?? [],
      },
    },
  }
}

export function sortDepartmentUserRanking(
  items: DepartmentUsageUserRankItem[],
  sort: DepartmentUsageUserRankSort
): DepartmentUsageUserRankItem[] {
  return [...items].sort((a, b) => {
    const primary =
      sort === 'requests'
        ? b.request_count - a.request_count
        : sort === 'tokens'
          ? b.token_count - a.token_count
          : b.quota - a.quota
    if (primary !== 0) return primary
    if (b.quota !== a.quota) return b.quota - a.quota
    if (b.request_count !== a.request_count) return b.request_count - a.request_count
    if (b.token_count !== a.token_count) return b.token_count - a.token_count
    return a.username.localeCompare(b.username)
  })
}

export function resolveRecentLogsSearch(
  entry: DepartmentUsageLogEntryLink,
  selectedLogUser?: string
) {
  const selectedUsername =
    selectedLogUser &&
    entry.filters.username_options.includes(selectedLogUser)
      ? selectedLogUser
      : entry.filters.username_options[0]

  return {
    departmentId: entry.filters.department_id ?? undefined,
    departmentName: entry.filters.department_name || undefined,
    startTime: entry.filters.start_timestamp * 1000,
    endTime: (entry.filters.end_timestamp + 1) * 1000,
    username: selectedUsername || undefined,
  }
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
