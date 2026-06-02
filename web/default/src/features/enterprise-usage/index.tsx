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
import { useEffect, useMemo, useRef, useState } from 'react'
import z from 'zod'
import { useForm, type UseFormReturn } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useSearch } from '@tanstack/react-router'
import {
  AlertTriangle,
  BarChart3,
  BellRing,
  Coins,
  Download,
  ExternalLink,
  LineChart,
  RefreshCw,
  Rows3,
  Users,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  CartesianGrid,
  Line,
  LineChart as RechartsLineChart,
  XAxis,
  YAxis,
} from 'recharts'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import dayjs from '@/lib/dayjs'
import { formatDateStr, formatNumber, formatQuota } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { SectionPageLayout } from '@/components/layout'
import { PageTransition } from '@/components/page-transition'
import { StatCard } from '@/features/dashboard/components/ui/stat-card'
import { DepartmentTree } from '@/features/enterprise-organization/components/DepartmentTree'
import { useDepartmentTree } from '@/features/enterprise-organization/hooks/use-department-tree'
import {
  findDepartmentNode,
  resolveDepartmentSelection,
  syncExpandedDepartmentIds,
  toggleExpandedDepartmentId,
} from '@/features/enterprise-organization/lib/tree-utils'
import {
  formatEnterpriseUserPrimary,
  formatEnterpriseUserSecondary,
} from '@/features/enterprise-organization/lib/user-display'
import type { DepartmentTreeNode } from '@/features/enterprise-organization/types'
import {
  departmentUsageReportQueryKey,
  exportDepartmentUsageCSV,
  getDepartmentUsageReportConfig,
  saveDepartmentUsageReportConfig,
} from './api'
import { useDepartmentUsageDetail } from './hooks/use-department-usage-detail'
import { useDepartmentUsageSummary } from './hooks/use-department-usage-summary'
import type {
  DepartmentUsageDetailResponse,
  DepartmentUsageLogEntryLink,
  DepartmentUsageLogUserOption,
  DepartmentUsageReportJobItem,
  DepartmentUsageSummaryScope,
  DepartmentUsageSummarySort,
  DepartmentUsageSummaryItem,
  DepartmentUsageUserRankItem,
  DepartmentUsageUserRankSort,
  EnterpriseUsagePreset,
  EnterpriseUsageSearch,
  DepartmentUsageTrendPoint,
  UsageSortOrder,
  UsageModelDistributionItem,
} from './types'

const ENTERPRISE_USAGE_PRESETS = [
  'today',
  'yesterday',
  'last7d',
  'last30d',
  'custom',
] as const satisfies readonly EnterpriseUsagePreset[]

const reportConfigSchema = (t: (key: string) => string) =>
  z.object({
    receivers: z
      .string()
      .trim()
      .min(1, t('At least one email receiver is required'))
      .refine((value) => {
        const receivers = value
          .split(/[\n,;]+/)
          .map((item) => item.trim())
          .filter(Boolean)
        return receivers.length > 0
      }, t('At least one email receiver is required'))
      .refine((value) => {
        const receivers = value
          .split(/[\n,;]+/)
          .map((item) => item.trim())
          .filter(Boolean)
        return receivers.every(
          (item) => z.string().email().safeParse(item).success
        )
      }, t('Each receiver must be a valid email address')),
    frequency: z.enum(['daily', 'weekly', 'monthly']),
    range_type: z.enum(['today', 'last7d', 'last30d']),
    enabled: z.boolean(),
  })

type ReportConfigFormValues = z.infer<ReturnType<typeof reportConfigSchema>>

export function shouldResetReportForm(params: {
  hasReportData: boolean
  isDirty: boolean
  scopeChanged: boolean
}) {
  if (!params.hasReportData) {
    return params.scopeChanged || !params.isDirty
  }
  return params.scopeChanged || !params.isDirty
}

function reportConfigToFormValues(
  item?: DepartmentUsageReportJobItem | null
): ReportConfigFormValues {
  return {
    receivers: item?.receivers?.join('\n') ?? '',
    frequency: item?.frequency ?? 'daily',
    range_type: item?.range_type ?? 'last7d',
    enabled: item?.enabled ?? false,
  }
}

export const enterpriseUsageSearchSchema = z.object({
  preset: z.enum(ENTERPRISE_USAGE_PRESETS).optional().catch('today'),
  from: z.coerce.number().int().optional().catch(undefined),
  to: z.coerce.number().int().optional().catch(undefined),
  tenant_id: z.coerce.number().int().nonnegative().optional().catch(undefined),
  dept_id: z.coerce.number().int().positive().optional().catch(undefined),
  include_descendants: z.coerce.boolean().optional().catch(false),
  sort: z.enum(['quota', 'requests', 'tokens']).optional().catch('quota'),
  summary_sort: z
    .enum(['requests', 'quota', 'users', 'dept_name'])
    .optional()
    .catch('requests'),
  summary_order: z.enum(['asc', 'desc']).optional().catch('desc'),
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

export function resolveDepartmentUsageExportParams(
  search: EnterpriseUsageSearch,
  range: Pick<ResolvedEnterpriseUsageRange, 'from' | 'to'>
) {
  return {
    from: range.from,
    to: range.to,
    tenantId: search.tenant_id,
    departmentId: search.dept_id,
    includeDescendants: search.include_descendants,
    summarySort: search.summary_sort,
    summaryOrder: search.summary_order,
  }
}

export function normalizeEnterpriseUsageSearch(params: {
  departments: DepartmentTreeNode[]
  search: EnterpriseUsageSearch
  treeResolved?: boolean
}) {
  if (params.departments.length === 0) {
    if (params.treeResolved === false) {
      return {
        ...params.search,
        include_descendants: params.search.include_descendants ?? false,
      }
    }

    return {
      ...params.search,
      dept_id: undefined,
      include_descendants: false,
      log_user: undefined,
    }
  }

  const resolved = resolveDepartmentSelection(
    params.departments,
    params.search.dept_id
  )
  const normalizedDepartmentId = resolved.normalizedDepartmentId ?? undefined
  const shouldResetChildState = params.search.dept_id !== normalizedDepartmentId

  return {
    ...params.search,
    dept_id: normalizedDepartmentId,
    include_descendants: params.search.include_descendants ?? false,
    log_user: shouldResetChildState ? undefined : params.search.log_user,
  }
}

export function shouldSyncEnterpriseUsageSearch(
  current: EnterpriseUsageSearch,
  normalized: EnterpriseUsageSearch
) {
  return (
    current.dept_id !== normalized.dept_id ||
    current.include_descendants !== normalized.include_descendants ||
    current.log_user !== normalized.log_user
  )
}

export function resolveEnterpriseUsageSyncedSearch(
  current: EnterpriseUsageSearch,
  normalized: EnterpriseUsageSearch
): EnterpriseUsageSearch {
  return {
    ...current,
    dept_id: normalized.dept_id,
    include_descendants:
      current.include_descendants === undefined &&
      normalized.include_descendants === false
        ? undefined
        : normalized.include_descendants,
    log_user: normalized.log_user,
  }
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
  const queryClient = useQueryClient()
  const search = useSearch({
    from: '/_authenticated/enterprise-usage/',
  }) as EnterpriseUsageSearch
  const navigate = useNavigate()
  const { auth } = useAuthStore()
  const {
    data: departments = [],
    isLoading: treeLoading,
    error: treeError,
    refetch: refetchTree,
  } = useDepartmentTree()
  const resolvedSelection = useMemo(
    () => resolveDepartmentSelection(departments, search.dept_id),
    [departments, search.dept_id]
  )
  const normalizedSearch = useMemo(
    () =>
      normalizeEnterpriseUsageSearch({
        departments,
        search,
        treeResolved: !treeLoading && !treeError,
      }),
    [departments, search, treeError, treeLoading]
  )
  const [expandedDepartmentIds, setExpandedDepartmentIds] = useState<number[]>(
    []
  )
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

  useEffect(() => {
    setExpandedDepartmentIds((current) =>
      syncExpandedDepartmentIds(
        current,
        departments,
        resolvedSelection.requiredExpandedIds
      )
    )
  }, [departments, resolvedSelection.requiredExpandedIds])

  useEffect(() => {
    if (!shouldSyncEnterpriseUsageSearch(search, normalizedSearch)) {
      return
    }
    navigate({
      to: '/enterprise-usage',
      search: (prev) => resolveEnterpriseUsageSyncedSearch(prev, normalizedSearch),
      replace: true,
    })
  }, [
    navigate,
    normalizedSearch.dept_id,
    normalizedSearch.include_descendants,
    normalizedSearch.log_user,
    search.dept_id,
    search.include_descendants,
    search.log_user,
  ])

  const usageQuery = useDepartmentUsageSummary({
    from: resolvedRange.from,
    to: resolvedRange.to,
    tenantId: search.tenant_id,
    departmentId: normalizedSearch.dept_id,
    includeDescendants: normalizedSearch.include_descendants,
    summarySort: search.summary_sort,
    summaryOrder: search.summary_order,
  })
  const detailQuery = useDepartmentUsageDetail({
    deptId: normalizedSearch.dept_id,
    from: resolvedRange.from,
    to: resolvedRange.to,
    tenantId: search.tenant_id,
  })

  const customRangeState = getCustomRangeState(customFromDate, customToDate)
  const [isExporting, setIsExporting] = useState(false)
  const reportQuery = useQuery({
    queryKey:
      search.tenant_id === undefined
        ? departmentUsageReportQueryKey
        : [...departmentUsageReportQueryKey, search.tenant_id],
    queryFn: async () => {
      const response = await getDepartmentUsageReportConfig({
        tenantId: search.tenant_id,
      })
      if (!response.success) {
        throw new Error(response.message || 'Request failed')
      }
      return response.data.item
    },
  })
  const reportForm = useForm<ReportConfigFormValues>({
    resolver: zodResolver(reportConfigSchema(t)),
    defaultValues: reportConfigToFormValues(),
  })
  const lastReportScopeRef = useRef<string>(
    String(search.tenant_id ?? reportQuery.data?.tenant_id ?? 0)
  )

  useEffect(() => {
    const reportScope = String(
      search.tenant_id ?? reportQuery.data?.tenant_id ?? 0
    )
    const scopeChanged = lastReportScopeRef.current !== reportScope
    lastReportScopeRef.current = reportScope
    if (!reportQuery.data) {
      if (
        shouldResetReportForm({
          hasReportData: false,
          isDirty: reportForm.formState.isDirty,
          scopeChanged,
        })
      ) {
        reportForm.reset(reportConfigToFormValues())
      }
      return
    }
    if (
      shouldResetReportForm({
        hasReportData: true,
        isDirty: reportForm.formState.isDirty,
        scopeChanged,
      })
    ) {
      reportForm.reset(reportConfigToFormValues(reportQuery.data))
    }
  }, [
    reportForm,
    reportForm.formState.isDirty,
    reportQuery.data,
    search.tenant_id,
  ])

  const reportMutation = useMutation({
    mutationFn: async (values: ReportConfigFormValues) => {
      const response = await saveDepartmentUsageReportConfig({
        tenantId: search.tenant_id,
        receivers: values.receivers
          .split(/[\n,;]+/)
          .map((item) => item.trim())
          .filter(Boolean),
        frequency: values.frequency,
        rangeType: values.range_type,
        enabled: values.enabled,
      })
      if (!response.success) {
        throw new Error(response.message || 'Request failed')
      }
      return response.data.item
    },
    onSuccess: async (item) => {
      await queryClient.invalidateQueries({
        queryKey:
          search.tenant_id === undefined
            ? departmentUsageReportQueryKey
            : [...departmentUsageReportQueryKey, search.tenant_id],
      })
      reportForm.reset(reportConfigToFormValues(item))
      toast.success(t('Usage report configuration saved'))
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

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

  const handleSortChange = (sort: DepartmentUsageUserRankSort) => {
    navigate({
      to: '/enterprise-usage',
      search: (prev) => ({
        ...prev,
        sort,
      }),
    })
  }

  const handleSummarySortChange = (
    summarySort: DepartmentUsageSummarySort,
    summaryOrder: UsageSortOrder
  ) => {
    navigate({
      to: '/enterprise-usage',
      search: (prev) => ({
        ...prev,
        summary_sort: summarySort,
        summary_order: summaryOrder,
      }),
    })
  }

  const handleExport = async () => {
    setIsExporting(true)
    try {
      const { blob, fileName } = await exportDepartmentUsageCSV(
        resolveDepartmentUsageExportParams(search, resolvedRange)
      )
      const url = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = fileName
      document.body.appendChild(anchor)
      anchor.click()
      anchor.remove()
      window.setTimeout(() => URL.revokeObjectURL(url), 0)
    } catch (error) {
      const message =
        error instanceof Error ? error.message : t('Request failed')
      toast.error(message)
    } finally {
      setIsExporting(false)
    }
  }

  const handleOpenRecentLogs = (entry: DepartmentUsageLogEntryLink) => {
    void navigate({
      to: '/usage-logs/$section',
      params: { section: entry.section as 'common' },
      search: resolveRecentLogsSearch(entry, search.log_user),
    })
  }

  const isAdmin = (auth.user?.role ?? 0) >= ROLE.ADMIN
  const currentDepartment = useMemo(
    () =>
      findDepartmentNode(departments, resolvedSelection.selectedDepartmentId),
    [departments, resolvedSelection.selectedDepartmentId]
  )
  const handleRefresh = () => {
    void refetchTree()
    void usageQuery.refetch()
    void detailQuery.refetch()
    void reportQuery.refetch()
  }

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
            onClick={handleRefresh}
            disabled={
              usageQuery.isFetching ||
              detailQuery.isFetching ||
              reportQuery.isFetching
            }
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
                {t(
                  'Only enterprise administrators can review department usage dashboards.'
                )}
              </AlertDescription>
            </Alert>
          ) : (
            <EnterpriseUsageContent
              departments={departments}
              treeLoading={treeLoading}
              treeErrorMessage={
                treeError instanceof Error ? treeError.message : null
              }
              expandedDepartmentIds={expandedDepartmentIds}
              items={usageQuery.data?.items ?? []}
              summaryScope={usageQuery.data?.scope ?? null}
              isLoading={usageQuery.isLoading}
              errorMessage={
                usageQuery.error instanceof Error
                  ? usageQuery.error.message
                  : null
              }
              rangeLabel={resolvedRange.rangeLabel}
              selectedDepartmentId={normalizedSearch.dept_id}
              currentDepartmentName={currentDepartment?.name ?? null}
              includeDescendants={normalizedSearch.include_descendants ?? false}
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
              onRetryTree={() => refetchTree()}
              onToggleDepartmentExpand={(departmentId) =>
                setExpandedDepartmentIds((current) =>
                  toggleExpandedDepartmentId(current, departmentId)
                )
              }
              onSelectDepartment={handleSelectDepartment}
              onRetryDetail={() => detailQuery.refetch()}
              rankSort={search.sort ?? 'quota'}
              onSortChange={handleSortChange}
              summarySort={search.summary_sort ?? 'requests'}
              summaryOrder={search.summary_order ?? 'desc'}
              onSummarySortChange={handleSummarySortChange}
              onIncludeDescendantsChange={(includeDescendants) =>
                navigate({
                  to: '/enterprise-usage',
                  search: (prev: EnterpriseUsageSearch) => ({
                    ...prev,
                    include_descendants: includeDescendants,
                  }),
                })
              }
              onExport={handleExport}
              exportLoading={isExporting}
              report={reportQuery.data ?? null}
              reportLoading={reportQuery.isLoading}
              reportErrorMessage={
                reportQuery.error instanceof Error
                  ? reportQuery.error.message
                  : null
              }
              reportForm={reportForm}
              onSaveReport={() =>
                void reportForm.handleSubmit((values) =>
                  reportMutation.mutateAsync(values)
                )()
              }
              reportSaving={reportMutation.isPending}
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
  departments?: DepartmentTreeNode[]
  treeLoading?: boolean
  treeErrorMessage?: string | null
  expandedDepartmentIds?: number[]
  items: DepartmentUsageSummaryItem[]
  summaryScope?: DepartmentUsageSummaryScope | null
  selectedDepartmentId?: number
  currentDepartmentName?: string | null
  includeDescendants?: boolean
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
  onRetryTree?: () => void
  onToggleDepartmentExpand?: (departmentId: number) => void
  onSelectDepartment: (deptId: number | null) => void
  onBackToOverview?: () => void
  onRetryDetail: () => void
  rankSort: DepartmentUsageUserRankSort
  onSortChange: (sort: DepartmentUsageUserRankSort) => void
  summarySort: DepartmentUsageSummarySort
  summaryOrder: UsageSortOrder
  onSummarySortChange: (
    summarySort: DepartmentUsageSummarySort,
    summaryOrder: UsageSortOrder
  ) => void
  onIncludeDescendantsChange?: (value: boolean) => void
  onExport: () => void
  exportLoading: boolean
  report?: DepartmentUsageReportJobItem | null
  reportLoading?: boolean
  reportErrorMessage?: string | null
  reportForm?: UseFormReturn<ReportConfigFormValues>
  onSaveReport?: () => void
  reportSaving?: boolean
  selectedLogUser?: string
  onOpenRecentLogs: (entry: DepartmentUsageLogEntryLink) => void
}

function buildUsageDepartmentTreeNodes(items: DepartmentUsageSummaryItem[]) {
  return items
    .filter((item) => item.dept_id != null)
    .map((item) => ({
      id: item.dept_id as number,
      tenant_id: 0,
      name: item.dept_name || String(item.dept_id),
      parent_id: null,
      status: 1,
      source_type: 1,
      external_id: '',
      sync_status: 1,
      sync_error: '',
      name_history: [],
      created_at: 0,
      updated_at: 0,
      deleted_at: 0,
      children: [],
    }))
}

function detailToSummaryItem(
  detail: DepartmentUsageDetailResponse
): DepartmentUsageSummaryItem {
  return {
    dept_id: detail.dept_id,
    dept_name: detail.dept_name,
    window_start: detail.window_start,
    window_end: detail.window_end,
    request_count: detail.request_count,
    prompt_tokens: detail.prompt_tokens,
    completion_tokens: detail.completion_tokens,
    quota: detail.quota,
    user_count: detail.user_count,
    model_distribution: detail.model_distribution,
  }
}

export function EnterpriseUsageContent(props: EnterpriseUsageContentProps) {
  const { t } = useTranslation()
  const treeNodes = useMemo(
    () =>
      props.departments && props.departments.length > 0
        ? props.departments
        : buildUsageDepartmentTreeNodes(props.items),
    [props.departments, props.items]
  )
  const treeSelection = useMemo(
    () => resolveDepartmentSelection(treeNodes, props.selectedDepartmentId),
    [treeNodes, props.selectedDepartmentId]
  )
  const selectedSummaryItem = useMemo(
    () =>
      props.items.find(
        (item) => item.dept_id === treeSelection.normalizedDepartmentId
      ) ?? null,
    [props.items, treeSelection.normalizedDepartmentId]
  )
  const analysisSummaryItems = props.detail
    ? [detailToSummaryItem(props.detail)]
    : selectedSummaryItem
      ? [selectedSummaryItem]
      : props.items
  const currentDepartmentLabel =
    props.currentDepartmentName ||
    props.detail?.dept_name ||
    selectedSummaryItem?.dept_name ||
    t('No department selected')
  const scopeLabel =
    props.summaryScope?.department_name || currentDepartmentLabel

  return (
    <div className='space-y-4'>
      <Alert>
        <BarChart3 className='size-4' />
        <AlertTitle>{t('Department totals are non-additive')}</AlertTitle>
        <AlertDescription>
          {t('enterprise.usage.multi_dept_disclaimer')}
        </AlertDescription>
      </Alert>

      <div className='grid gap-4 xl:grid-cols-[minmax(320px,360px)_minmax(0,1fr)]'>
        <Card className='overflow-hidden'>
          <CardHeader>
            <CardTitle>{t('Department Tree')}</CardTitle>
            <CardDescription>
              {t(
                'Select a department from the tree to drive the analysis workspace on the right.'
              )}
            </CardDescription>
          </CardHeader>
          <CardContent className='px-0 pb-0'>
            {props.treeLoading ? (
              <EnterpriseUsageTreeSkeleton />
            ) : props.treeErrorMessage ? (
              <Alert variant='destructive' className='mx-6 mb-6 gap-2'>
                <AlertTriangle className='size-4' />
                <AlertTitle>{t('Unable to load department tree')}</AlertTitle>
                <AlertDescription>{t(props.treeErrorMessage)}</AlertDescription>
                <div className='pt-2'>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={props.onRetryTree}
                  >
                    {t('Retry')}
                  </Button>
                </div>
              </Alert>
            ) : treeNodes.length === 0 ? (
              <Empty className='px-6 pb-6'>
                <EmptyHeader>
                  <EmptyMedia variant='icon'>
                    <BarChart3 className='size-5' />
                  </EmptyMedia>
                  <EmptyTitle>{t('No departments yet')}</EmptyTitle>
                  <EmptyDescription>
                    {t(
                      'Department analysis becomes available after the organization tree is synced.'
                    )}
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : (
              <DepartmentTree
                nodes={treeNodes}
                expandedIds={props.expandedDepartmentIds}
                selectedDepartmentId={treeSelection.normalizedDepartmentId}
                onToggleExpand={props.onToggleDepartmentExpand}
                onSelectDepartment={
                  props.onSelectDepartment as (departmentId: number) => void
                }
              />
            )}
          </CardContent>
        </Card>

        <div className='space-y-4'>
          <Card>
            <CardHeader className='gap-3'>
              <div className='flex flex-col gap-2 lg:flex-row lg:items-start lg:justify-between'>
                <div className='space-y-1'>
                  <CardTitle>{t('Current Department Analysis')}</CardTitle>
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
                  <Button
                    type='button'
                    size='sm'
                    variant='outline'
                    onClick={props.onExport}
                    disabled={props.exportLoading}
                  >
                    <Download className='size-4' />
                    {t('Export CSV')}
                  </Button>
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
              <div className='flex items-center justify-between rounded-lg border px-3 py-2'>
                <div className='space-y-1'>
                  <div className='text-sm font-medium'>
                    {t('Include descendants')}
                  </div>
                  <div className='text-muted-foreground text-xs'>
                    {props.includeDescendants
                      ? t('Current scope: {{department}} and all descendant departments', {
                          department: scopeLabel,
                        })
                      : t('Current scope: {{department}} only', {
                          department: scopeLabel,
                        })}
                  </div>
                </div>
                <Switch
                  checked={props.includeDescendants ?? false}
                  onCheckedChange={(value) =>
                    props.onIncludeDescendantsChange?.(value)
                  }
                />
              </div>
              {!props.customRange.isValid ? (
                <Alert variant='destructive'>
                  <AlertDescription>
                    {t(
                      'Start time must be earlier than end time for a custom range'
                    )}
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
                  <AlertTitle>
                    {t('Unable to load department usage')}
                  </AlertTitle>
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
                      {t(
                        'Usage snapshots are empty for the selected time window.'
                      )}
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
                  <div className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]'>
                    <Card>
                      <CardHeader>
                        <CardTitle>{t('Current department context')}</CardTitle>
                        <CardDescription>
                          {currentDepartmentLabel}
                        </CardDescription>
                      </CardHeader>
                      <CardContent className='grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
                        <ReportStat
                          label={t('Scope')}
                          value={scopeLabel}
                        />
                        <ReportStat
                          label={t('Requests')}
                          value={formatNumber(props.summaryScope?.request_count ?? 0)}
                        />
                        <ReportStat
                          label={t('Users')}
                          value={formatNumber(props.summaryScope?.user_count ?? 0)}
                        />
                        <ReportStat
                          label={t('Quota')}
                          value={formatQuota(props.summaryScope?.quota ?? 0)}
                        />
                      </CardContent>
                    </Card>
                    <Card>
                      <CardHeader>
                        <CardTitle>{t('Analysis Notes')}</CardTitle>
                        <CardDescription>
                          {t(
                            'Usage analysis stays pinned to the current department while report settings keep their original scope.'
                          )}
                        </CardDescription>
                      </CardHeader>
                      <CardContent className='space-y-3 text-sm'>
                        <p>{t('enterprise.usage.multi_dept_disclaimer')}</p>
                        <p>
                          {t(
                            'Scheduled reports remain tenant-scoped unless explicitly stated otherwise in the configuration card below.'
                          )}
                        </p>
                        <p>
                          {t(
                            'Detail ranking and recent logs stay on the current department and do not auto-expand the full descendant tree.'
                          )}
                        </p>
                      </CardContent>
                    </Card>
                  </div>

                  <EnterpriseUsageSummaryCards items={analysisSummaryItems} />

                  <DepartmentUsageDetailPanel
                    detail={props.detail}
                    isLoading={props.detailLoading}
                    errorMessage={props.detailErrorMessage}
                    rankSort={props.rankSort}
                    onSortChange={props.onSortChange}
                    onRetry={props.onRetryDetail}
                    selectedLogUser={props.selectedLogUser}
                    onOpenRecentLogs={props.onOpenRecentLogs}
                  />

                  <Card>
                    <CardHeader className='gap-3'>
                      <div className='flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between'>
                        <div>
                          <CardTitle>{t('Peer Department Snapshot')}</CardTitle>
                          <CardDescription>
                            {t(
                              'This supporting list keeps existing department overview semantics without replacing tree-driven navigation.'
                            )}
                          </CardDescription>
                        </div>
                        <div className='flex flex-wrap gap-2'>
                          {(
                            [
                              ['requests', 'desc', t('Sort by Requests')],
                              ['quota', 'desc', t('Sort by Quota')],
                              ['users', 'desc', t('Sort by Users')],
                              ['dept_name', 'asc', t('Sort by Department')],
                            ] as const
                          ).map(([field, order, label]) => (
                            <Button
                              key={`${field}-${order}`}
                              type='button'
                              size='sm'
                              variant={
                                props.summarySort === field &&
                                props.summaryOrder === order
                                  ? 'default'
                                  : 'outline'
                              }
                              onClick={() =>
                                props.onSummarySortChange(field, order)
                              }
                            >
                              {label}
                            </Button>
                          ))}
                        </div>
                      </div>
                    </CardHeader>
                    <CardContent className='overflow-x-auto'>
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
                            <TableRow
                              key={`${item.dept_id ?? 'unassigned'}-${index}`}
                              className={
                                item.dept_id ===
                                treeSelection.normalizedDepartmentId
                                  ? 'bg-muted/40'
                                  : undefined
                              }
                            >
                              <TableCell className='font-medium'>
                                {item.dept_name || t('Unassigned')}
                              </TableCell>
                              <TableCell>
                                {formatNumber(item.request_count)}
                              </TableCell>
                              <TableCell>
                                {formatNumber(item.prompt_tokens)}
                              </TableCell>
                              <TableCell>
                                {formatNumber(item.completion_tokens)}
                              </TableCell>
                              <TableCell>{formatQuota(item.quota)}</TableCell>
                              <TableCell>
                                {formatNumber(item.user_count)}
                              </TableCell>
                              <TableCell className='max-w-[340px] text-sm whitespace-normal'>
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
                                    item.dept_id ===
                                    treeSelection.normalizedDepartmentId
                                      ? 'default'
                                      : 'outline'
                                  }
                                  disabled={item.dept_id == null}
                                  onClick={() =>
                                    props.onSelectDepartment(item.dept_id)
                                  }
                                >
                                  {t('Use as Current Context')}
                                </Button>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </CardContent>
                  </Card>

                  <DepartmentUsageReportCard
                    report={props.report ?? null}
                    isLoading={props.reportLoading ?? false}
                    errorMessage={props.reportErrorMessage ?? null}
                    form={props.reportForm}
                    onSave={props.onSaveReport}
                    saving={props.reportSaving ?? false}
                  />
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}

function DepartmentUsageReportCard(props: {
  report: DepartmentUsageReportJobItem | null
  isLoading: boolean
  errorMessage: string | null
  form?: UseFormReturn<ReportConfigFormValues>
  onSave?: () => void
  saving: boolean
}) {
  const { t } = useTranslation()
  const form = props.form

  if (!form) {
    return null
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className='flex items-center gap-2'>
          <BellRing className='size-4' />
          {t('Scheduled Usage Reports')}
        </CardTitle>
        <CardDescription>
          {t(
            'Configure email receivers, cadence, and report range for department usage digests.'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        {props.isLoading ? (
          <Skeleton className='h-40 w-full' />
        ) : props.errorMessage ? (
          <Alert variant='destructive'>
            <AlertTriangle className='size-4' />
            <AlertTitle>
              {t('Unable to load usage report configuration')}
            </AlertTitle>
            <AlertDescription>{t(props.errorMessage)}</AlertDescription>
          </Alert>
        ) : (
          <>
            <Form {...form}>
              <form
                className='space-y-4'
                onSubmit={(event) => {
                  event.preventDefault()
                  props.onSave?.()
                }}
              >
                <FormField
                  control={form.control}
                  name='receivers'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Report Receivers')}</FormLabel>
                      <FormControl>
                        <textarea
                          className='border-input bg-background min-h-[96px] w-full rounded-md border px-3 py-2 text-sm'
                          placeholder={t('Enter one email per line')}
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <div className='grid gap-4 md:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name='frequency'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Report Frequency')}</FormLabel>
                        <FormControl>
                          <select
                            className='border-input bg-background h-10 w-full rounded-md border px-3 text-sm'
                            value={field.value}
                            onChange={(event) =>
                              field.onChange(event.target.value)
                            }
                          >
                            <option value='daily'>{t('Daily')}</option>
                            <option value='weekly'>{t('Weekly')}</option>
                            <option value='monthly'>{t('Monthly')}</option>
                          </select>
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name='range_type'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Report Range')}</FormLabel>
                        <FormControl>
                          <select
                            className='border-input bg-background h-10 w-full rounded-md border px-3 text-sm'
                            value={field.value}
                            onChange={(event) =>
                              field.onChange(event.target.value)
                            }
                          >
                            <option value='today'>{t('Today')}</option>
                            <option value='last7d'>{t('Last 7 Days')}</option>
                            <option value='last30d'>{t('Last 30 Days')}</option>
                          </select>
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
                <FormField
                  control={form.control}
                  name='enabled'
                  render={({ field }) => (
                    <FormItem className='flex items-center justify-between rounded-lg border p-3'>
                      <div>
                        <FormLabel>{t('Enable Scheduled Reports')}</FormLabel>
                        <p className='text-muted-foreground text-sm'>
                          {t(
                            'Send recurring usage summaries to configured receivers.'
                          )}
                        </p>
                      </div>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <Button type='submit' disabled={props.saving}>
                  <RefreshCw className='size-4' />
                  {props.saving ? t('Saving') : t('Save Report Configuration')}
                </Button>
              </form>
            </Form>
            <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
              <ReportStat
                label={t('Status')}
                value={props.report?.status ?? '-'}
              />
              <ReportStat
                label={t('Next Run')}
                value={formatOptionalTimestamp(props.report?.next_run_at)}
              />
              <ReportStat
                label={t('Last Success')}
                value={formatOptionalTimestamp(props.report?.last_success_at)}
              />
              <ReportStat
                label={t('Failures')}
                value={formatNumber(props.report?.failure_count ?? 0)}
              />
            </div>
            {props.report?.error_reason ? (
              <Alert variant='destructive'>
                <AlertTriangle className='size-4' />
                <AlertTitle>{t('Last Delivery Error')}</AlertTitle>
                <AlertDescription>{props.report.error_reason}</AlertDescription>
              </Alert>
            ) : null}
          </>
        )}
      </CardContent>
    </Card>
  )
}

function ReportStat(props: { label: string; value: string }) {
  return (
    <div className='bg-muted/50 rounded-lg border px-3 py-3'>
      <div className='text-muted-foreground text-xs'>{props.label}</div>
      <div className='mt-1 text-sm font-medium'>{props.value}</div>
    </div>
  )
}

function formatOptionalTimestamp(value?: number) {
  if (!value) return '-'
  return dayjs(value * 1000).format('YYYY-MM-DD HH:mm')
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
            <CardTitle>{t('Current Department Analysis')}</CardTitle>
            <CardDescription>
              {props.detail?.dept_name
                ? t(
                    'Inspect top users, model mix, time trend, and recent logs.'
                  )
                : t(
                    'Select a department from the tree to keep analysis pinned to the current context.'
                  )}
            </CardDescription>
          </div>
          {props.detail ? (
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() =>
                props.onOpenRecentLogs(props.detail!.recent_logs_entry)
              }
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
              <EmptyTitle>{t('Current Department Analysis')}</EmptyTitle>
              <EmptyDescription>
                {t(
                  'Select a department from the tree to keep analysis pinned to the current context.'
                )}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className='space-y-4'>
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
                    {t(
                      'Models ranked by quota and request count inside this department.'
                    )}
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
                      {t(
                        'Ranking is calculated only within the current department scope.'
                      )}
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
                        variant={
                          props.rankSort === sort ? 'default' : 'outline'
                        }
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
                <div className='text-muted-foreground flex flex-wrap gap-2 text-xs'>
                  <span>{t('Recent Logs User Filter')}:</span>
                  {resolveRecentLogsUserOptions(
                    props.detail.recent_logs_entry
                  ).map((user) => (
                    <span
                      key={user.username}
                      className={
                        user.username === props.selectedLogUser
                          ? 'bg-primary/10 text-primary rounded-full px-2 py-1'
                          : 'bg-muted rounded-full px-2 py-1'
                      }
                    >
                      {formatEnterpriseUserPrimary({
                        displayName: user.display_name,
                        username: user.username,
                        userId: user.user_id,
                      })}
                    </span>
                  ))}
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
    return (
      <div className='text-muted-foreground text-sm'>
        {t('No trend data for this time range')}
      </div>
    )
  }

  return (
    <ChartContainer config={detailTrendChartConfig} className='h-72 w-full'>
      <RechartsLineChart accessibilityLayer data={data}>
        <CartesianGrid vertical={false} />
        <XAxis
          dataKey='label'
          tickLine={false}
          axisLine={false}
          minTickGap={24}
        />
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
              <TableCell
                colSpan={3}
                className='text-muted-foreground text-center'
              >
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
              <TableCell
                colSpan={4}
                className='text-muted-foreground text-center'
              >
                {t('No department members consumed within this time range')}
              </TableCell>
            </TableRow>
          ) : (
            props.items.map((item) => (
              <TableRow key={item.user_id}>
                <TableCell>
                  <div className='flex min-w-[180px] flex-col gap-1'>
                    <span className='font-medium'>
                      {formatEnterpriseUserPrimary({
                        displayName: item.display_name,
                        username: item.username,
                        userId: item.user_id,
                      })}
                    </span>
                    <span className='text-muted-foreground text-xs'>
                      {formatEnterpriseUserSecondary(
                        {
                          displayName: item.display_name,
                          username: item.username,
                          userId: item.user_id,
                        },
                        t
                      )}
                    </span>
                  </div>
                </TableCell>
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

function EnterpriseUsageTreeSkeleton() {
  return (
    <div className='space-y-2 px-6 pb-6'>
      <Skeleton className='h-10 w-full' />
      <Skeleton className='h-10 w-full' />
      <Skeleton className='h-10 w-full' />
      <Skeleton className='h-10 w-4/5' />
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
      unassignedCount: props.items.filter((item) => item.dept_id == null)
        .length,
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
  items: DepartmentUsageSummaryItem[],
  summarySort: DepartmentUsageSummarySort = 'requests',
  summaryOrder: UsageSortOrder = 'desc'
): DepartmentUsageSummaryItem[] {
  return [...items]
    .map((item) => ({
      ...item,
      dept_name: item.dept_name || '',
      model_distribution: item.model_distribution ?? [],
    }))
    .sort((a, b) => {
      return compareDepartmentUsageItems(a, b, summarySort, summaryOrder)
    })
}

export function compareDepartmentUsageItems(
  a: DepartmentUsageSummaryItem,
  b: DepartmentUsageSummaryItem,
  summarySort: DepartmentUsageSummarySort,
  summaryOrder: UsageSortOrder
) {
  const unassignedName = '未归属'
  const applyOrder = (value: number) =>
    summaryOrder === 'asc' ? value : -value
  const compareNumber = (left: number, right: number) =>
    left < right ? -1 : left > right ? 1 : 0
  const compareName = (left: string, right: string) =>
    left.localeCompare(right, undefined, { sensitivity: 'base' })
  const sortName = (item: DepartmentUsageSummaryItem) =>
    item.dept_name || unassignedName
  const compareOptionalId = (
    left: number | null | undefined,
    right: number | null | undefined
  ) => {
    if (left == null && right == null) return 0
    if (left == null) return 1
    if (right == null) return -1
    return compareNumber(left, right)
  }

  let primary = 0
  if (summarySort === 'quota') {
    primary = compareNumber(a.quota, b.quota)
  } else if (summarySort === 'users') {
    primary = compareNumber(a.user_count, b.user_count)
  } else if (summarySort === 'dept_name') {
    primary = compareName(sortName(a), sortName(b))
  } else {
    primary = compareNumber(a.request_count, b.request_count)
  }
  if (primary !== 0) {
    return applyOrder(primary)
  }
  if (a.request_count !== b.request_count) {
    return b.request_count - a.request_count
  }
  if (a.quota !== b.quota) {
    return b.quota - a.quota
  }
  if (a.user_count !== b.user_count) {
    return b.user_count - a.user_count
  }
  if (compareName(sortName(a), sortName(b)) !== 0) {
    return compareName(sortName(a), sortName(b))
  }
  return compareOptionalId(a.dept_id, b.dept_id)
}

export function normalizeDepartmentUsageDetail(
  detail: DepartmentUsageDetailResponse
): DepartmentUsageDetailResponse {
  return {
    ...detail,
    dept_name: detail.dept_name || '',
    user_ranking: (detail.user_ranking ?? []).map((item) => ({
      ...item,
      display_name: item.display_name ?? '',
    })),
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
        user_options: (detail.recent_logs_entry.filters.user_options ?? []).map(
          (item) => ({
            ...item,
            display_name: item.display_name ?? '',
          })
        ),
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
    if (b.request_count !== a.request_count)
      return b.request_count - a.request_count
    if (b.token_count !== a.token_count) return b.token_count - a.token_count
    return a.username.localeCompare(b.username)
  })
}

export function resolveRecentLogsSearch(
  entry: DepartmentUsageLogEntryLink,
  selectedLogUser?: string
) {
  const selectedUsername =
    selectedLogUser && entry.filters.username_options.includes(selectedLogUser)
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

function resolveRecentLogsUserOptions(entry: DepartmentUsageLogEntryLink) {
  const unique = new Map<string, DepartmentUsageLogUserOption>()
  for (const item of entry.filters.user_options ?? []) {
    if (!item.username) continue
    unique.set(item.username, item)
  }
  if (unique.size > 0) {
    return Array.from(unique.values())
  }
  return (entry.filters.username_options ?? []).map((username, index) => ({
    user_id: index + 1,
    username,
    display_name: '',
  }))
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
  return dayjs(unixSeconds * 1000)
    .subtract(1, 'day')
    .format('YYYY-MM-DD')
}

function buildRangeLabel(from: number, to: number): string {
  return `${formatDateStr(new Date(from * 1000))} ~ ${formatDateStr(
    new Date(
      dayjs(to * 1000)
        .subtract(1, 'second')
        .valueOf()
    )
  )}`
}
