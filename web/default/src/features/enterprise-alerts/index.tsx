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
import { zodResolver } from '@hookform/resolvers/zod'
import {
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import {
  AlertTriangle,
  ArrowRight,
  BellRing,
  Info,
  Plus,
  Save,
  Search,
  Trash2,
} from 'lucide-react'
import dayjs from 'dayjs'
import {
  CartesianGrid,
  Line,
  LineChart as RechartsLineChart,
  XAxis,
  YAxis,
} from 'recharts'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { formatTimestamp } from '@/lib/format'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
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
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from '@/components/ui/chart'
import { SectionPageLayout } from '@/components/layout'
import {
  alertEventsListQueryKey,
  alertDeliveriesListQueryKey,
  alertOverviewQueryKey,
  enterpriseAlertDeliveriesQueryKey,
  getDepartmentRiskSummary,
  alertRulesListQueryKey,
  deleteAlertRule,
  getAlertDeliveries,
  getAlertEvents,
  getAlertRules,
  resendAlertDelivery,
  saveAlertRule,
} from './api'
import type {
  AlertDeliveryItem,
  AlertDeliveryStatus,
  AlertEventItem,
  DepartmentRiskSummaryItem,
  DepartmentRiskTrendPoint,
  AlertRuleItem,
  AlertRuleUpsertRequest,
  EnterpriseAlertDeliveriesSearch,
  EnterpriseAlertsSearch,
} from './types'

export const enterpriseAlertsSearchSchema = z.object({
  tab: z.enum(['overview', 'events', 'deliveries', 'rules']).optional().catch(undefined),
  tenant_id: z.coerce.number().int().nonnegative().optional().catch(undefined),
  department_id: z.coerce.number().int().positive().optional().catch(undefined),
  unassigned_only: z.coerce.boolean().optional().catch(undefined),
  user_id: z.coerce.number().int().positive().optional().catch(undefined),
  username: z.string().optional().catch(undefined),
  model_name: z.string().optional().catch(undefined),
  risk_type: z.string().optional().catch(undefined),
  from: z.coerce.number().int().positive().optional().catch(undefined),
  to: z.coerce.number().int().positive().optional().catch(undefined),
  page: z.coerce.number().int().positive().optional().catch(1),
  page_size: z.coerce.number().int().positive().optional().catch(20),
  summary_sort: z.enum(['requests', 'quota', 'users', 'dept_name']).optional().catch(undefined),
  summary_order: z.enum(['asc', 'desc']).optional().catch(undefined),
})

const filterSchema = z.object({
  tenant_id: z.string(),
  department_id: z.string(),
  user_id: z.string(),
  username: z.string(),
  model_name: z.string(),
  risk_type: z.string(),
  from: z.string(),
  to: z.string(),
  page_size: z.string(),
})

type AlertFilterFormValues = z.infer<typeof filterSchema>

type DeliveryFilterState = {
  status: '' | AlertDeliveryStatus
  channel_type: '' | 'email' | 'webhook' | 'dingtalk_robot'
  trigger_source: '' | 'rule_match' | 'manual_resend'
}

type RuleEditorState = {
  id?: number
  name: string
  enabled: boolean
  riskTypes: string
  departmentIds: string
  dedupeWindowSeconds: string
  emailEnabled: boolean
  emailReceivers: string
  webhookEnabled: boolean
  webhookUrl: string
  webhookSecret: string
  webhookSecretConfigured: boolean
  webhookSecretMasked: string
  dingtalkEnabled: boolean
  dingtalkRobotUrl: string
  dingtalkRobotSecret: string
  dingtalkSecretConfigured: boolean
  dingtalkSecretMasked: string
}

export function mapAlertFilterFormToSearch(values: AlertFilterFormValues) {
  return {
    tab: 'events' as const,
    tenant_id: parseOptionalNumber(values.tenant_id),
    department_id: parseOptionalNumber(values.department_id),
    unassigned_only: undefined,
    user_id: parseOptionalNumber(values.user_id),
    username: emptyToUndefined(values.username),
    model_name: emptyToUndefined(values.model_name),
    risk_type: emptyToUndefined(values.risk_type),
    from: parseOptionalNumber(values.from),
    to: parseOptionalNumber(values.to),
    page: 1,
    page_size: parseOptionalNumber(values.page_size) ?? 20,
  }
}

export function formatDepartmentSnapshot(
  item: Pick<AlertEventItem, 'department_snapshot'>,
  fallbackLabel: string
) {
  if (!item.department_snapshot.length) return fallbackLabel
  return item.department_snapshot
    .map((department) => `${department.department_name} (#${department.department_id})`)
    .join(', ')
}

export function formatDeliveryStatus(
  status: string,
  t: (value: string) => string
) {
  switch (status) {
    case 'pending':
      return t('Pending')
    case 'sent':
      return t('Sent')
    case 'failed':
      return t('Failed')
    case 'final_failed':
      return t('Final failed')
    case 'resent':
      return t('Resent')
    default:
      return status
  }
}

export function formatDeliveryTraceSummary(
  item: Pick<AlertDeliveryItem, 'trace'>,
  fallbackLabel: string
) {
  if (!item.trace) return fallbackLabel
  return [item.trace.department_summary, item.trace.request_id, item.trace.detail_route]
    .filter(Boolean)
    .join(' · ')
}

export function formatTriggerSource(
  triggerSource: string,
  t: (value: string) => string
) {
  switch (triggerSource) {
    case 'manual_resend':
      return t('Manual resend')
    case 'rule_match':
      return t('Rule match')
    default:
      return triggerSource || t('Unknown')
  }
}

export function buildDeliveryFiltersAfterResend(
  filters: DeliveryFilterState
): DeliveryFilterState {
  return {
    ...filters,
    status: '',
    trigger_source: '',
  }
}

export function buildAlertRulePayload(
  draft: RuleEditorState,
  tenantId?: number
): AlertRuleUpsertRequest {
  const payload: AlertRuleUpsertRequest = {
    ...(draft.id === undefined ? {} : { id: draft.id }),
    ...(tenantId === undefined ? {} : { tenant_id: tenantId }),
    name: draft.name.trim(),
    enabled: draft.enabled,
    risk_types: parseLineSeparatedList(draft.riskTypes),
    department_ids: parseNumberList(draft.departmentIds),
    dedupe_window_seconds:
      parseOptionalNumber(draft.dedupeWindowSeconds) ?? 0,
    channel_configs: [
      {
        type: 'email',
        enabled: draft.emailEnabled,
        receivers: parseLineSeparatedList(draft.emailReceivers),
      },
      {
        type: 'webhook',
        enabled: draft.webhookEnabled,
        webhook_url: emptyToUndefined(draft.webhookUrl),
        ...(draft.webhookSecret.trim()
          ? { webhook_secret: draft.webhookSecret.trim() }
          : {}),
      },
      {
        type: 'dingtalk_robot',
        enabled: draft.dingtalkEnabled,
        dingtalk_robot_url: emptyToUndefined(draft.dingtalkRobotUrl),
        ...(draft.dingtalkRobotSecret.trim()
          ? { dingtalk_robot_secret: draft.dingtalkRobotSecret.trim() }
          : {}),
      },
    ],
  }
  return payload
}

export function describeAlertRuleSecretStatus(
  configured: boolean,
  masked: string,
  configuredLabel: string,
  missingLabel: string
) {
  if (!configured) {
    return missingLabel
  }
  return masked ? `${configuredLabel} (${masked})` : configuredLabel
}

export function formatRiskRate(value: number) {
  return `${(value * 100).toFixed(1)}%`
}

const departmentRiskTrendChartConfig = {
  riskRate: {
    label: 'Risk Rate',
    color: 'var(--color-chart-1)',
  },
  riskEvents: {
    label: 'Risk Events',
    color: 'var(--color-chart-2)',
  },
} as const

function searchToFormDefaults(search: EnterpriseAlertsSearch): AlertFilterFormValues {
  return {
    tenant_id: numberToString(search.tenant_id),
    department_id: numberToString(search.department_id),
    user_id: numberToString(search.user_id),
    username: search.username ?? '',
    model_name: search.model_name ?? '',
    risk_type: search.risk_type ?? '',
    from: numberToString(search.from),
    to: numberToString(search.to),
    page_size: numberToString(search.page_size ?? 20),
  }
}

export function overviewSearchFromAlerts(
  search: EnterpriseAlertsSearch
): EnterpriseAlertsSearch {
  const current = dayjs()
  return {
    tenant_id: search.tenant_id,
    from: search.from ?? current.subtract(6, 'day').startOf('day').unix(),
    to: search.to ?? current.add(1, 'day').startOf('day').unix(),
    summary_sort: search.summary_sort ?? 'quota',
    summary_order: search.summary_order ?? 'desc',
  }
}

export function buildAlertEventEntrySearch(
  previous: EnterpriseAlertsSearch,
  entry: DepartmentRiskSummaryItem['event_entry']
): EnterpriseAlertsSearch {
  return {
    tab: 'events',
    tenant_id: previous.tenant_id,
    department_id: entry.department_id,
    unassigned_only: entry.unassigned_only || undefined,
    from: entry.from,
    to: entry.to,
    page: 1,
    page_size: previous.page_size,
  }
}

function deliveriesSearchFromAlerts(
  search: EnterpriseAlertsSearch,
  filters: DeliveryFilterState
): EnterpriseAlertDeliveriesSearch {
  return {
    tenant_id: search.tenant_id,
    ...(filters.status ? { status: filters.status } : {}),
    ...(filters.channel_type ? { channel_type: filters.channel_type } : {}),
    ...(filters.trigger_source ? { trigger_source: filters.trigger_source } : {}),
    page: search.page ?? 1,
    page_size: search.page_size ?? 20,
  }
}

export function DepartmentRiskOverviewTab(props: {
  summary:
    | {
        items: DepartmentRiskSummaryItem[]
        top_departments: DepartmentRiskSummaryItem[]
        trend: DepartmentRiskTrendPoint[]
        unassigned: DepartmentRiskSummaryItem
        formula: {
          expression: string
          numerator_label: string
          denominator_label: string
        }
        disclaimer_key: string
      }
    | undefined
  isLoading: boolean
  errorMessage: string | null
  onOpenEventEntry: (entry: DepartmentRiskSummaryItem['event_entry']) => void
}) {
  const { t } = useTranslation()
  const summary = props.summary

  return (
    <div className='space-y-6'>
      <Alert>
        <Info className='size-4' />
        <AlertTitle>{t('Department risk overview')}</AlertTitle>
        <AlertDescription>
          {t(summary?.disclaimer_key ?? 'enterprise.usage.multi_dept_disclaimer')}
        </AlertDescription>
      </Alert>

      <Card>
        <CardHeader>
          <CardTitle>{t('Risk Rate Formula')}</CardTitle>
          <CardDescription>
            {t('Department totals are non-additive')}
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-2 text-sm'>
          <div className='font-medium'>
            {t(summary?.formula.expression ?? 'Risk rate = risky requests / total requests')}
          </div>
          <div className='text-muted-foreground'>
            {t(summary?.formula.numerator_label ?? 'Requests that triggered risk events')}
          </div>
          <div className='text-muted-foreground'>
            {t(summary?.formula.denominator_label ?? 'Total requests from department members')}
          </div>
        </CardContent>
      </Card>

      {props.isLoading ? (
        <div className='space-y-3'>
          <Skeleton className='h-32 w-full' />
          <Skeleton className='h-64 w-full' />
        </div>
      ) : props.errorMessage ? (
        <Alert variant='destructive'>
          <AlertTriangle className='size-4' />
          <AlertTitle>{t('Request failed')}</AlertTitle>
          <AlertDescription>{t(props.errorMessage)}</AlertDescription>
        </Alert>
      ) : (
        <>
          <div className='grid gap-4 xl:grid-cols-[0.9fr_1.1fr]'>
            <Card>
              <CardHeader>
                <CardTitle>{t('Top Risk Departments')}</CardTitle>
                <CardDescription>
                  {t('Ranked by risk rate with drill-down entry into the event list.')}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className='space-y-3'>
                  {(summary?.top_departments ?? []).length === 0 ? (
                    <div className='text-muted-foreground text-sm'>
                      {t('No risk overview data for this time range')}
                    </div>
                  ) : (
                    summary?.top_departments.map((item) => (
                      <button
                        key={`${item.dept_id ?? 'unassigned'}-${item.window_start}`}
                        type='button'
                        className='hover:bg-muted/60 flex w-full items-center justify-between rounded-lg border px-4 py-3 text-left'
                        onClick={() => props.onOpenEventEntry(item.event_entry)}
                      >
                        <div className='space-y-1'>
                          <div className='font-medium'>{item.dept_name || t('Unassigned')}</div>
                          <div className='text-muted-foreground text-xs'>
                            {t('Risk events')}: {item.risk_event_count} · {t('Requests')}: {item.total_request_count}
                          </div>
                        </div>
                        <div className='flex items-center gap-3'>
                          <Badge variant='secondary'>{formatRiskRate(item.risk_rate)}</Badge>
                          <ArrowRight className='size-4' />
                        </div>
                      </button>
                    ))
                  )}
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>{t('Recent Trend')}</CardTitle>
                <CardDescription>
                  {t('Trend lines reflect the same multi-department duplicate-counting rule as the summary.')}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <DepartmentRiskTrendChart trend={summary?.trend ?? []} />
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>{t('Department Risk Summary')}</CardTitle>
              <CardDescription>
                {t('Review each department, the unassigned bucket, and jump straight to matching risk events.')}
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              <div className='overflow-x-auto'>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('Department')}</TableHead>
                      <TableHead>{t('Risk Events')}</TableHead>
                      <TableHead>{t('Requests')}</TableHead>
                      <TableHead>{t('Risk Rate')}</TableHead>
                      <TableHead>{t('Actions')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(summary?.items ?? []).length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={5} className='text-muted-foreground text-center'>
                          {t('No risk overview data for this time range')}
                        </TableCell>
                      </TableRow>
                    ) : (
                      (summary?.items ?? []).map((item) => (
                        <TableRow key={`${item.dept_id ?? 'unassigned'}-${item.window_start}`}>
                          <TableCell className='font-medium'>
                            {item.dept_name || t('Unassigned')}
                          </TableCell>
                          <TableCell>{item.risk_event_count}</TableCell>
                          <TableCell>{item.total_request_count}</TableCell>
                          <TableCell>{formatRiskRate(item.risk_rate)}</TableCell>
                          <TableCell>
                            <Button
                              type='button'
                              size='sm'
                              variant='outline'
                              onClick={() => props.onOpenEventEntry(item.event_entry)}
                            >
                              {t('View Risk Events')}
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </div>
              <div className='text-muted-foreground text-sm'>
                {t('Unassigned')}: {summary?.unassigned.risk_event_count ?? 0} / {summary?.unassigned.total_request_count ?? 0}
              </div>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}

function DepartmentRiskTrendChart(props: {
  trend: DepartmentRiskTrendPoint[]
}) {
  const { t } = useTranslation()
  const data = props.trend.map((point) => ({
    label: dayjs(point.window_start * 1000).format('MM-DD HH:mm'),
    riskRate: Number((point.risk_rate * 100).toFixed(2)),
    riskEvents: point.risk_event_count,
  }))

  if (data.length === 0) {
    return (
      <div className='text-muted-foreground text-sm'>
        {t('No trend data for this time range')}
      </div>
    )
  }

  return (
    <ChartContainer config={departmentRiskTrendChartConfig} className='h-72 w-full'>
      <RechartsLineChart accessibilityLayer data={data}>
        <CartesianGrid vertical={false} />
        <XAxis dataKey='label' tickLine={false} axisLine={false} minTickGap={24} />
        <YAxis yAxisId='left' tickLine={false} axisLine={false} width={48} />
        <YAxis yAxisId='right' orientation='right' tickLine={false} axisLine={false} width={48} />
        <ChartTooltip content={<ChartTooltipContent />} />
        <Line yAxisId='left' type='monotone' dataKey='riskRate' stroke='var(--color-riskRate)' strokeWidth={2} dot={false} />
        <Line yAxisId='right' type='monotone' dataKey='riskEvents' stroke='var(--color-riskEvents)' strokeWidth={2} dot={false} />
      </RechartsLineChart>
    </ChartContainer>
  )
}

function createEmptyRuleDraft(): RuleEditorState {
  return {
    name: '',
    enabled: true,
    riskTypes: 'abuse\nsensitive_words',
    departmentIds: '',
    dedupeWindowSeconds: '300',
    emailEnabled: true,
    emailReceivers: '',
    webhookEnabled: false,
    webhookUrl: '',
    webhookSecret: '',
    webhookSecretConfigured: false,
    webhookSecretMasked: '',
    dingtalkEnabled: false,
    dingtalkRobotUrl: '',
    dingtalkRobotSecret: '',
    dingtalkSecretConfigured: false,
    dingtalkSecretMasked: '',
  }
}

function mapRuleToDraft(rule: AlertRuleItem): RuleEditorState {
  const email = rule.channel_configs.find((item) => item.type === 'email')
  const webhook = rule.channel_configs.find((item) => item.type === 'webhook')
  const dingtalk = rule.channel_configs.find(
    (item) => item.type === 'dingtalk_robot'
  )

  return {
    id: rule.id,
    name: rule.name,
    enabled: rule.enabled,
    riskTypes: rule.risk_types.join('\n'),
    departmentIds: rule.department_ids.join(', '),
    dedupeWindowSeconds: String(rule.dedupe_window_seconds ?? 0),
    emailEnabled: email?.enabled ?? true,
    emailReceivers: (email?.receivers ?? []).join('\n'),
    webhookEnabled: webhook?.enabled ?? false,
    webhookUrl: webhook?.webhook_url ?? '',
    webhookSecret: '',
    webhookSecretConfigured: webhook?.webhook_secret_configured ?? false,
    webhookSecretMasked: webhook?.webhook_secret_masked ?? '',
    dingtalkEnabled: dingtalk?.enabled ?? false,
    dingtalkRobotUrl: dingtalk?.dingtalk_robot_url ?? '',
    dingtalkRobotSecret: '',
    dingtalkSecretConfigured:
      dingtalk?.dingtalk_robot_secret_configured ?? false,
    dingtalkSecretMasked: dingtalk?.dingtalk_robot_secret_masked ?? '',
  }
}

export function EnterpriseAlertsPage() {
  const { t } = useTranslation()
  const search = useSearch({
    from: '/_authenticated/enterprise-alerts/',
  }) as EnterpriseAlertsSearch
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<'overview' | 'events' | 'deliveries' | 'rules'>(
    (search.tab as 'overview' | 'events' | 'deliveries' | 'rules') ?? 'overview'
  )
  const [draft, setDraft] = useState<RuleEditorState>(createEmptyRuleDraft())
  const [deliveryFilters, setDeliveryFilters] = useState<DeliveryFilterState>({
    status: 'final_failed',
    channel_type: '',
    trigger_source: '',
  })

  const form = useForm<AlertFilterFormValues>({
    resolver: zodResolver(filterSchema),
    values: searchToFormDefaults(search),
  })

  const normalizedSearch = useMemo(
    () => ({
      ...search,
      page: search.page ?? 1,
      page_size: search.page_size ?? 20,
    }),
    [search]
  )
  const normalizedDeliveriesSearch = useMemo(
    () => deliveriesSearchFromAlerts(normalizedSearch, deliveryFilters),
    [normalizedSearch, deliveryFilters]
  )
  const overviewSearch = useMemo(
    () => overviewSearchFromAlerts(normalizedSearch),
    [normalizedSearch]
  )

  useEffect(() => {
    setActiveTab(
      (search.tab as 'overview' | 'events' | 'deliveries' | 'rules') ?? 'overview'
    )
  }, [search.tab])

  const overviewQuery = useQuery({
    queryKey: alertOverviewQueryKey(overviewSearch),
    queryFn: async () => {
      const response = await getDepartmentRiskSummary(overviewSearch)
      if (!response.success) {
        throw new Error(response.message || 'Request failed')
      }
      return response.data
    },
  })

  const alertsQuery = useQuery({
    queryKey: alertEventsListQueryKey(normalizedSearch),
    queryFn: async () => {
      const response = await getAlertEvents(normalizedSearch)
      if (!response.success) {
        throw new Error(response.message || 'Request failed')
      }
      return response.data
    },
  })

  const alertRulesQuery = useQuery({
    queryKey: alertRulesListQueryKey(normalizedSearch.tenant_id),
    queryFn: async () => {
      const response = await getAlertRules(normalizedSearch.tenant_id)
      if (!response.success) {
        throw new Error(response.message || 'Request failed')
      }
      return response.data
    },
  })

  const deliveriesQuery = useQuery({
    queryKey: alertDeliveriesListQueryKey(normalizedDeliveriesSearch),
    queryFn: async () => {
      const response = await getAlertDeliveries(normalizedDeliveriesSearch)
      if (!response.success) {
        throw new Error(response.message || 'Request failed')
      }
      return response.data
    },
  })

  useEffect(() => {
    if (!alertRulesQuery.data?.items.length) {
      return
    }
    const currentRuleExists =
      draft.id !== undefined &&
      alertRulesQuery.data.items.some((item) => item.id === draft.id)
    if (draft.id === undefined || !currentRuleExists) {
      setDraft(mapRuleToDraft(alertRulesQuery.data.items[0]))
    }
  }, [alertRulesQuery.data, draft.id])

  const saveRuleMutation = useMutation({
    mutationFn: async (payload: AlertRuleUpsertRequest) => {
      const response = await saveAlertRule(payload)
      if (!response.success) {
        throw new Error(response.message || 'Request failed')
      }
      return response.data.item
    },
    onSuccess: async (item) => {
      setDraft(mapRuleToDraft(item))
      await queryClient.invalidateQueries({
        queryKey: alertRulesListQueryKey(normalizedSearch.tenant_id),
      })
      toast.success(t('Alert rule saved'))
    },
  })

  const deleteRuleMutation = useMutation({
    mutationFn: async (id: number) => {
      const response = await deleteAlertRule(id, normalizedSearch.tenant_id)
      if (!response.success) {
        throw new Error(response.message || 'Request failed')
      }
      return response.data.item
    },
    onSuccess: async () => {
      setDraft(createEmptyRuleDraft())
      await queryClient.invalidateQueries({
        queryKey: alertRulesListQueryKey(normalizedSearch.tenant_id),
      })
      toast.success(t('Alert rule deleted'))
    },
  })

  const resendDeliveryMutation = useMutation({
    mutationFn: async (delivery: AlertDeliveryItem) => {
      const response = await resendAlertDelivery(
        delivery.id,
        normalizedSearch.tenant_id
      )
      if (!response.success) {
        throw new Error(response.message || 'Request failed')
      }
      return response.data
    },
    onSuccess: async (data) => {
      setDeliveryFilters((prev) => buildDeliveryFiltersAfterResend(prev))
      await queryClient.invalidateQueries({
        queryKey: enterpriseAlertDeliveriesQueryKey,
      })
      toast.success(
        data.created
          ? t('Resend created')
          : t('Existing manual resend is still pending')
      )
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : t('Request failed')
      )
    },
  })

  const totalPages = Math.max(
    1,
    Math.ceil((alertsQuery.data?.total ?? 0) / (alertsQuery.data?.page_size ?? 20))
  )

  const onSubmit = (values: AlertFilterFormValues) => {
    navigate({
      to: '/enterprise-alerts',
      search: mapAlertFilterFormToSearch(values),
    })
  }

  const changePage = (page: number) => {
    navigate({
      to: '/enterprise-alerts',
      search: (prev) => ({
        ...prev,
        page,
      }),
    })
  }

  const submitRule = async () => {
    await saveRuleMutation.mutateAsync(
      buildAlertRulePayload(draft, normalizedSearch.tenant_id)
    )
  }

  const deleteCurrentRule = async () => {
    if (draft.id === undefined) {
      return
    }
    await deleteRuleMutation.mutateAsync(draft.id)
  }

  const handleTabChange = (value: 'overview' | 'events' | 'deliveries' | 'rules') => {
    setActiveTab(value)
    navigate({
      to: '/enterprise-alerts',
      search: (prev) => ({
        ...prev,
        tab: value,
      }),
    })
  }

  const handleOpenEventEntry = (entry: DepartmentRiskSummaryItem['event_entry']) => {
    navigate({
      to: '/enterprise-alerts',
      search: (prev) => buildAlertEventEntrySearch(prev, entry),
    })
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Enterprise Alerts')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='space-y-6'>
          <Tabs
            value={activeTab}
            onValueChange={(value) =>
              handleTabChange(value as 'overview' | 'events' | 'deliveries' | 'rules')
            }
            className='space-y-6'
          >
            <TabsList className='grid w-full grid-cols-4 md:w-[640px]'>
              <TabsTrigger value='overview'>{t('Risk Overview')}</TabsTrigger>
              <TabsTrigger value='events'>{t('Risk Events')}</TabsTrigger>
              <TabsTrigger value='deliveries'>{t('Deliveries')}</TabsTrigger>
              <TabsTrigger value='rules'>{t('Alert Rules')}</TabsTrigger>
            </TabsList>

            <TabsContent value='overview' className='space-y-6'>
              <DepartmentRiskOverviewTab
                summary={overviewQuery.data}
                isLoading={overviewQuery.isLoading}
                errorMessage={
                  overviewQuery.error instanceof Error
                    ? overviewQuery.error.message
                    : null
                }
                onOpenEventEntry={handleOpenEventEntry}
              />
            </TabsContent>

            <TabsContent value='events' className='space-y-6'>
              <Card>
                <CardHeader>
                  <CardTitle>{t('Search')}</CardTitle>
                  <CardDescription>
                    {t(
                      'Filter enterprise risk events by department snapshot, actor, model, risk type, and time window.'
                    )}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <Form {...form}>
                    <form
                      className='grid gap-4 md:grid-cols-3'
                      onSubmit={form.handleSubmit(onSubmit)}
                    >
                      <FormField
                        control={form.control}
                        name='department_id'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Department')}</FormLabel>
                            <FormControl>
                              <Input {...field} placeholder='11' />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name='user_id'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('User')}</FormLabel>
                            <FormControl>
                              <Input {...field} placeholder='1001' />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name='username'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Username')}</FormLabel>
                            <FormControl>
                              <Input {...field} placeholder='alice' />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name='model_name'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Model')}</FormLabel>
                            <FormControl>
                              <Input {...field} placeholder='gpt-4o-mini' />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name='risk_type'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Risk Type')}</FormLabel>
                            <FormControl>
                              <Input {...field} placeholder='sensitive_words' />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name='page_size'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Page Size')}</FormLabel>
                            <FormControl>
                              <Input {...field} placeholder='20' />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name='from'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('From Timestamp')}</FormLabel>
                            <FormControl>
                              <Input {...field} placeholder='1717117200' />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name='to'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('To Timestamp')}</FormLabel>
                            <FormControl>
                              <Input {...field} placeholder='1717203600' />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <div className='flex items-end'>
                        <Button className='w-full' type='submit'>
                          <Search className='mr-2 size-4' />
                          {t('Search')}
                        </Button>
                      </div>
                    </form>
                  </Form>
                </CardContent>
              </Card>

              {alertsQuery.error ? (
                <Alert variant='destructive'>
                  <AlertTriangle className='size-4' />
                  <AlertTitle>{t('Request failed')}</AlertTitle>
                  <AlertDescription>
                    {alertsQuery.error instanceof Error
                      ? alertsQuery.error.message
                      : t('Request failed')}
                  </AlertDescription>
                </Alert>
              ) : null}

              <Card>
                <CardHeader>
                  <CardTitle>{t('Enterprise Alerts')}</CardTitle>
                  <CardDescription>
                    {t(
                      'Only traceable summary fields are shown. Sensitive raw prompts are never returned here.'
                    )}
                  </CardDescription>
                </CardHeader>
                <CardContent className='space-y-4'>
                  {alertsQuery.isLoading ? (
                    <div className='space-y-3'>
                      <Skeleton className='h-12 w-full' />
                      <Skeleton className='h-12 w-full' />
                      <Skeleton className='h-12 w-full' />
                    </div>
                  ) : alertsQuery.data?.items.length ? (
                    <>
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>{t('Created At')}</TableHead>
                            <TableHead>{t('Department')}</TableHead>
                            <TableHead>{t('User')}</TableHead>
                            <TableHead>{t('Request ID')}</TableHead>
                            <TableHead>{t('Model')}</TableHead>
                            <TableHead>{t('Risk Type')}</TableHead>
                            <TableHead>{t('Action Result')}</TableHead>
                            <TableHead>{t('Summary')}</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {alertsQuery.data.items.map((item) => (
                            <TableRow key={item.id}>
                              <TableCell>
                                {formatTimestamp(item.created_at)}
                              </TableCell>
                              <TableCell>
                                {formatDepartmentSnapshot(item, t('Unassigned'))}
                              </TableCell>
                              <TableCell>{item.username}</TableCell>
                              <TableCell>{item.request_id}</TableCell>
                              <TableCell>{item.model_name}</TableCell>
                              <TableCell>{item.risk_type}</TableCell>
                              <TableCell>{item.action_result}</TableCell>
                              <TableCell>{item.summary}</TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>

                      <Pagination>
                        <PaginationContent>
                          <PaginationItem>
                            <PaginationPrevious
                              href='#'
                              onClick={(event) => {
                                event.preventDefault()
                                if (normalizedSearch.page && normalizedSearch.page > 1) {
                                  changePage(normalizedSearch.page - 1)
                                }
                              }}
                            />
                          </PaginationItem>
                          {Array.from({ length: totalPages }).map((_, index) => {
                            const page = index + 1
                            return (
                              <PaginationItem key={page}>
                                <PaginationLink
                                  href='#'
                                  isActive={page === (normalizedSearch.page ?? 1)}
                                  onClick={(event) => {
                                    event.preventDefault()
                                    changePage(page)
                                  }}
                                >
                                  {page}
                                </PaginationLink>
                              </PaginationItem>
                            )
                          })}
                          <PaginationItem>
                            <PaginationNext
                              href='#'
                              onClick={(event) => {
                                event.preventDefault()
                                if ((normalizedSearch.page ?? 1) < totalPages) {
                                  changePage((normalizedSearch.page ?? 1) + 1)
                                }
                              }}
                            />
                          </PaginationItem>
                        </PaginationContent>
                      </Pagination>
                    </>
                  ) : (
                    <Empty>
                      <EmptyHeader>
                        <EmptyMedia>
                          <AlertTriangle className='size-6' />
                        </EmptyMedia>
                        <EmptyTitle>{t('No risk events found')}</EmptyTitle>
                        <EmptyDescription>
                          {t('Adjust the filters and search again.')}
                        </EmptyDescription>
                      </EmptyHeader>
                      <EmptyContent />
                    </Empty>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value='deliveries' className='space-y-6'>
              {deliveriesQuery.error ? (
                <Alert variant='destructive'>
                  <AlertTriangle className='size-4' />
                  <AlertTitle>{t('Request failed')}</AlertTitle>
                  <AlertDescription>
                    {deliveriesQuery.error instanceof Error
                      ? deliveriesQuery.error.message
                      : t('Request failed')}
                  </AlertDescription>
                </Alert>
              ) : null}

              <Card>
                <CardHeader>
                  <CardTitle>{t('Deliveries')}</CardTitle>
                  <CardDescription>
                    {t(
                      'Review notification delivery results, retry timing, and trace lookup hints without exposing secrets or raw prompts.'
                    )}
                  </CardDescription>
                </CardHeader>
                <CardContent className='space-y-4'>
                  <div className='grid gap-4 md:grid-cols-4'>
                    <div className='space-y-2'>
                      <FormLabel>{t('Status')}</FormLabel>
                      <select
                        className='border-input bg-background rounded-md border px-3 py-2 text-sm'
                        value={deliveryFilters.status}
                        onChange={(event) =>
                          setDeliveryFilters((prev) => ({
                            ...prev,
                            status: event.target.value as DeliveryFilterState['status'],
                          }))
                        }
                      >
                        <option value=''>{t('All statuses')}</option>
                        <option value='final_failed'>{t('Final failed')}</option>
                        <option value='pending'>{t('Pending')}</option>
                        <option value='failed'>{t('Failed')}</option>
                        <option value='resent'>{t('Resent')}</option>
                        <option value='sent'>{t('Sent')}</option>
                      </select>
                    </div>
                    <div className='space-y-2'>
                      <FormLabel>{t('Channel')}</FormLabel>
                      <select
                        className='border-input bg-background rounded-md border px-3 py-2 text-sm'
                        value={deliveryFilters.channel_type}
                        onChange={(event) =>
                          setDeliveryFilters((prev) => ({
                            ...prev,
                            channel_type: event.target.value as DeliveryFilterState['channel_type'],
                          }))
                        }
                      >
                        <option value=''>{t('All channels')}</option>
                        <option value='email'>{t('Email')}</option>
                        <option value='webhook'>{t('Webhook')}</option>
                        <option value='dingtalk_robot'>{t('DingTalk robot')}</option>
                      </select>
                    </div>
                    <div className='space-y-2'>
                      <FormLabel>{t('Retry source')}</FormLabel>
                      <select
                        className='border-input bg-background rounded-md border px-3 py-2 text-sm'
                        value={deliveryFilters.trigger_source}
                        onChange={(event) =>
                          setDeliveryFilters((prev) => ({
                            ...prev,
                            trigger_source: event.target.value as DeliveryFilterState['trigger_source'],
                          }))
                        }
                      >
                        <option value=''>{t('All sources')}</option>
                        <option value='rule_match'>{t('Rule match')}</option>
                        <option value='manual_resend'>{t('Manual resend')}</option>
                      </select>
                    </div>
                    <div className='flex items-end'>
                      <Button
                        className='w-full'
                        variant='outline'
                        onClick={() =>
                          setDeliveryFilters({
                            status: 'final_failed',
                            channel_type: '',
                            trigger_source: '',
                          })
                        }
                      >
                        {t('Show final failed only')}
                      </Button>
                    </div>
                  </div>
                  {deliveriesQuery.isLoading ? (
                    <div className='space-y-3'>
                      <Skeleton className='h-12 w-full' />
                      <Skeleton className='h-12 w-full' />
                    </div>
                  ) : deliveriesQuery.data?.items.length ? (
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Created At')}</TableHead>
                          <TableHead>{t('Status')}</TableHead>
                          <TableHead>{t('Channel')}</TableHead>
                          <TableHead>{t('Attempts')}</TableHead>
                          <TableHead>{t('Retry source')}</TableHead>
                          <TableHead>{t('Parent delivery')}</TableHead>
                          <TableHead>{t('Last Attempt')}</TableHead>
                          <TableHead>{t('Final failed at')}</TableHead>
                          <TableHead>{t('Trace Details')}</TableHead>
                          <TableHead>{t('Error Reason')}</TableHead>
                          <TableHead>{t('Actions')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {deliveriesQuery.data.items.map((item) => (
                          <TableRow key={item.id}>
                            <TableCell>{formatTimestamp(item.created_at)}</TableCell>
                            <TableCell>
                              <Badge
                                variant={
                                  item.status === 'final_failed'
                                    ? 'destructive'
                                    : 'outline'
                                }
                              >
                                {formatDeliveryStatus(item.status, t)}
                              </Badge>
                            </TableCell>
                            <TableCell>{item.channel_type}</TableCell>
                            <TableCell>
                              {item.attempt_count}/{item.max_attempts}
                            </TableCell>
                            <TableCell>
                              {formatTriggerSource(item.trigger_source, t)}
                            </TableCell>
                            <TableCell>
                              {item.manual_parent_id
                                ? `#${item.manual_parent_id}`
                                : t('None')}
                            </TableCell>
                            <TableCell>
                              {item.last_attempt_at > 0
                                ? formatTimestamp(item.last_attempt_at)
                                : t('Never')}
                            </TableCell>
                            <TableCell>
                              {item.final_failed_at > 0
                                ? formatTimestamp(item.final_failed_at)
                                : t('Not final failed')}
                            </TableCell>
                            <TableCell>
                              {formatDeliveryTraceSummary(
                                item,
                                t('No trace details yet')
                              )}
                            </TableCell>
                            <TableCell>{item.error_reason || t('No error')}</TableCell>
                            <TableCell>
                              {item.status === 'final_failed' ? (
                                <Button
                                  size='sm'
                                  variant='outline'
                                  disabled={resendDeliveryMutation.isPending}
                                  onClick={() =>
                                    void resendDeliveryMutation.mutateAsync(item)
                                  }
                                >
                                  {t('Resend')}
                                </Button>
                              ) : item.trigger_source === 'manual_resend' ? (
                                <span className='text-muted-foreground text-sm'>
                                  {t('Manual resend')}
                                </span>
                              ) : (
                                <span className='text-muted-foreground text-sm'>
                                  {t('No action')}
                                </span>
                              )}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  ) : deliveryFilters.status === 'final_failed' ? (
                    <Empty>
                      <EmptyHeader>
                        <EmptyMedia>
                          <BellRing className='size-6' />
                        </EmptyMedia>
                        <EmptyTitle>{t('No final failed deliveries')}</EmptyTitle>
                        <EmptyDescription>
                          {t(
                            'Switch filters to review the full history or wait for the dispatcher to produce new results.'
                          )}
                        </EmptyDescription>
                      </EmptyHeader>
                      <EmptyContent />
                    </Empty>
                  ) : (
                    <Empty>
                      <EmptyHeader>
                        <EmptyMedia>
                          <BellRing className='size-6' />
                        </EmptyMedia>
                        <EmptyTitle>{t('No deliveries yet')}</EmptyTitle>
                        <EmptyDescription>
                          {t(
                            'Alert deliveries will appear here after the background dispatcher runs.'
                          )}
                        </EmptyDescription>
                      </EmptyHeader>
                      <EmptyContent />
                    </Empty>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value='rules' className='space-y-6'>
              {alertRulesQuery.error ? (
                <Alert variant='destructive'>
                  <AlertTriangle className='size-4' />
                  <AlertTitle>{t('Request failed')}</AlertTitle>
                  <AlertDescription>
                    {alertRulesQuery.error instanceof Error
                      ? alertRulesQuery.error.message
                      : t('Request failed')}
                  </AlertDescription>
                </Alert>
              ) : null}

              <div className='grid gap-6 xl:grid-cols-[1.1fr_1.3fr]'>
                <Card>
                  <CardHeader className='flex flex-row items-center justify-between gap-4'>
                    <div>
                      <CardTitle>{t('Alert Rules')}</CardTitle>
                      <CardDescription>
                        {t(
                          'Configure department-aware risk notifications. Email is required for the V1 alert loop.'
                        )}
                      </CardDescription>
                    </div>
                    <Button
                      variant='outline'
                      onClick={() => setDraft(createEmptyRuleDraft())}
                    >
                      <Plus className='mr-2 size-4' />
                      {t('New Rule')}
                    </Button>
                  </CardHeader>
                  <CardContent className='space-y-3'>
                    {alertRulesQuery.isLoading ? (
                      <div className='space-y-3'>
                        <Skeleton className='h-18 w-full' />
                        <Skeleton className='h-18 w-full' />
                      </div>
                    ) : alertRulesQuery.data?.items.length ? (
                      alertRulesQuery.data.items.map((rule) => (
                        <button
                          key={rule.id}
                          type='button'
                          className={`w-full rounded-xl border p-4 text-left transition-colors ${
                            draft.id === rule.id
                              ? 'border-primary bg-primary/5'
                              : 'border-border hover:bg-muted/40'
                          }`}
                          onClick={() => setDraft(mapRuleToDraft(rule))}
                        >
                          <div className='flex items-start justify-between gap-3'>
                            <div className='space-y-2'>
                              <div className='flex items-center gap-2'>
                                <span className='font-medium'>{rule.name}</span>
                                <Badge
                                  variant={rule.enabled ? 'default' : 'outline'}
                                >
                                  {rule.enabled ? t('Enabled') : t('Disabled')}
                                </Badge>
                              </div>
                              <p className='text-muted-foreground text-sm'>
                                {rule.risk_types.join(', ') || t('No risk types')}
                              </p>
                              <p className='text-muted-foreground text-sm'>
                                {rule.department_ids.length
                                  ? t('Departments: {{value}}', {
                                      value: rule.department_ids.join(', '),
                                    })
                                  : t('All departments')}
                              </p>
                            </div>
                            <BellRing className='text-muted-foreground size-4' />
                          </div>
                        </button>
                      ))
                    ) : (
                      <Empty>
                        <EmptyHeader>
                          <EmptyMedia>
                            <BellRing className='size-6' />
                          </EmptyMedia>
                          <EmptyTitle>{t('No alert rules yet')}</EmptyTitle>
                          <EmptyDescription>
                            {t(
                              'Create the first rule to notify owners when risky content is detected.'
                            )}
                          </EmptyDescription>
                        </EmptyHeader>
                        <EmptyContent />
                      </Empty>
                    )}
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle>
                      {draft.id === undefined ? t('Create Rule') : t('Edit Rule')}
                    </CardTitle>
                    <CardDescription>
                      {t(
                        'Optional channels can stay disabled. They must not block email-based alert recording.'
                      )}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className='space-y-5'>
                    <div className='grid gap-4 md:grid-cols-2'>
                      <div className='space-y-2'>
                        <FormLabel>{t('Rule Name')}</FormLabel>
                        <Input
                          value={draft.name}
                          onChange={(event) =>
                            setDraft((prev) => ({
                              ...prev,
                              name: event.target.value,
                            }))
                          }
                          placeholder={t('Critical content abuse')}
                        />
                      </div>
                      <div className='space-y-2'>
                        <FormLabel>{t('Deduplication Window (seconds)')}</FormLabel>
                        <Input
                          value={draft.dedupeWindowSeconds}
                          onChange={(event) =>
                            setDraft((prev) => ({
                              ...prev,
                              dedupeWindowSeconds: event.target.value,
                            }))
                          }
                          placeholder='300'
                        />
                      </div>
                    </div>

                    <div className='flex flex-wrap gap-6 rounded-xl border p-4'>
                      <label className='flex items-center gap-3'>
                        <Switch
                          checked={draft.enabled}
                          onCheckedChange={(checked) =>
                            setDraft((prev) => ({ ...prev, enabled: checked }))
                          }
                        />
                        <span className='text-sm font-medium'>{t('Rule Enabled')}</span>
                      </label>
                      <label className='flex items-center gap-3'>
                        <Switch
                          checked={draft.emailEnabled}
                          onCheckedChange={(checked) =>
                            setDraft((prev) => ({
                              ...prev,
                              emailEnabled: checked,
                            }))
                          }
                        />
                        <span className='text-sm font-medium'>
                          {t('Email Channel Enabled')}
                        </span>
                      </label>
                    </div>

                    <div className='grid gap-4 lg:grid-cols-2'>
                      <div className='space-y-2'>
                        <FormLabel>{t('Risk Types')}</FormLabel>
                        <Textarea
                          value={draft.riskTypes}
                          onChange={(event) =>
                            setDraft((prev) => ({
                              ...prev,
                              riskTypes: event.target.value,
                            }))
                          }
                          placeholder={t('One risk type per line')}
                        />
                      </div>
                      <div className='space-y-2'>
                        <FormLabel>{t('Department IDs')}</FormLabel>
                        <Textarea
                          value={draft.departmentIds}
                          onChange={(event) =>
                            setDraft((prev) => ({
                              ...prev,
                              departmentIds: event.target.value,
                            }))
                          }
                          placeholder={t('Leave empty to target all departments')}
                        />
                      </div>
                    </div>

                    <div className='space-y-2'>
                      <FormLabel>{t('Email Receivers')}</FormLabel>
                      <Textarea
                        value={draft.emailReceivers}
                        onChange={(event) =>
                          setDraft((prev) => ({
                            ...prev,
                            emailReceivers: event.target.value,
                          }))
                        }
                        placeholder={t('One email receiver per line')}
                      />
                    </div>

                    <div className='grid gap-4 lg:grid-cols-2'>
                      <Card className='border-dashed'>
                        <CardHeader className='pb-3'>
                          <div className='flex items-center justify-between gap-3'>
                            <CardTitle className='text-base'>
                              {t('Webhook Channel')}
                            </CardTitle>
                            <Switch
                              checked={draft.webhookEnabled}
                              onCheckedChange={(checked) =>
                                setDraft((prev) => ({
                                  ...prev,
                                  webhookEnabled: checked,
                                }))
                              }
                            />
                          </div>
                          <CardDescription>
                            {t(
                              'Webhook is optional. Secrets are stored but never echoed back in plain text.'
                            )}
                          </CardDescription>
                        </CardHeader>
                        <CardContent className='space-y-3'>
                          <div className='space-y-2'>
                            <FormLabel>{t('Webhook URL')}</FormLabel>
                            <Input
                              value={draft.webhookUrl}
                              onChange={(event) =>
                                setDraft((prev) => ({
                                  ...prev,
                                  webhookUrl: event.target.value,
                                }))
                              }
                              placeholder='https://hooks.example.com/alerts'
                            />
                          </div>
                          <div className='space-y-2'>
                            <FormLabel>{t('Webhook Secret')}</FormLabel>
                            <Input
                              type='password'
                              value={draft.webhookSecret}
                              onChange={(event) =>
                                setDraft((prev) => ({
                                  ...prev,
                                  webhookSecret: event.target.value,
                                }))
                              }
                              placeholder={t('Enter a new secret to replace the stored one')}
                            />
                          </div>
                          <p className='text-muted-foreground text-sm'>
                            {describeAlertRuleSecretStatus(
                              draft.webhookSecretConfigured,
                              draft.webhookSecretMasked,
                              t('Secret already configured'),
                              t('No secret configured')
                            )}
                          </p>
                        </CardContent>
                      </Card>

                      <Card className='border-dashed'>
                        <CardHeader className='pb-3'>
                          <div className='flex items-center justify-between gap-3'>
                            <CardTitle className='text-base'>
                              {t('DingTalk Robot Channel')}
                            </CardTitle>
                            <Switch
                              checked={draft.dingtalkEnabled}
                              onCheckedChange={(checked) =>
                                setDraft((prev) => ({
                                  ...prev,
                                  dingtalkEnabled: checked,
                                }))
                              }
                            />
                          </div>
                          <CardDescription>
                            {t(
                              'DingTalk robot delivery is optional and should not block risk event persistence.'
                            )}
                          </CardDescription>
                        </CardHeader>
                        <CardContent className='space-y-3'>
                          <div className='space-y-2'>
                            <FormLabel>{t('DingTalk Robot URL')}</FormLabel>
                            <Input
                              value={draft.dingtalkRobotUrl}
                              onChange={(event) =>
                                setDraft((prev) => ({
                                  ...prev,
                                  dingtalkRobotUrl: event.target.value,
                                }))
                              }
                              placeholder='https://oapi.dingtalk.com/robot/send'
                            />
                          </div>
                          <div className='space-y-2'>
                            <FormLabel>{t('DingTalk Robot Secret')}</FormLabel>
                            <Input
                              type='password'
                              value={draft.dingtalkRobotSecret}
                              onChange={(event) =>
                                setDraft((prev) => ({
                                  ...prev,
                                  dingtalkRobotSecret: event.target.value,
                                }))
                              }
                              placeholder={t('Enter a new secret to replace the stored one')}
                            />
                          </div>
                          <p className='text-muted-foreground text-sm'>
                            {describeAlertRuleSecretStatus(
                              draft.dingtalkSecretConfigured,
                              draft.dingtalkSecretMasked,
                              t('Secret already configured'),
                              t('No secret configured')
                            )}
                          </p>
                        </CardContent>
                      </Card>
                    </div>

                    <div className='flex flex-wrap items-center gap-3'>
                      <Button
                        onClick={() => void submitRule()}
                        disabled={saveRuleMutation.isPending}
                      >
                        <Save className='mr-2 size-4' />
                        {saveRuleMutation.isPending
                          ? t('Saving...')
                          : t('Save Rule')}
                      </Button>
                      <Button
                        variant='outline'
                        onClick={() => setDraft(createEmptyRuleDraft())}
                      >
                        {t('Reset Draft')}
                      </Button>
                      {draft.id !== undefined ? (
                        <Button
                          variant='destructive'
                          onClick={() => void deleteCurrentRule()}
                          disabled={deleteRuleMutation.isPending}
                        >
                          <Trash2 className='mr-2 size-4' />
                          {deleteRuleMutation.isPending
                            ? t('Deleting...')
                            : t('Delete Rule')}
                        </Button>
                      ) : null}
                    </div>
                  </CardContent>
                </Card>
              </div>
            </TabsContent>
          </Tabs>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function parseLineSeparatedList(value: string) {
  return value
    .split(/[\n,]/g)
    .map((item) => item.trim())
    .filter(Boolean)
}

function parseNumberList(value: string) {
  return value
    .split(/[\n,]/g)
    .map((item) => Number.parseInt(item.trim(), 10))
    .filter((item) => Number.isFinite(item) && item > 0)
}

function parseOptionalNumber(value?: string) {
  if (!value) return undefined
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : undefined
}

function emptyToUndefined(value?: string) {
  const trimmed = value?.trim()
  return trimmed ? trimmed : undefined
}

function numberToString(value?: number) {
  return value === undefined ? '' : String(value)
}
