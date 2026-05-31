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
import { z } from 'zod'
import { useForm, type Resolver } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useSearch } from '@tanstack/react-router'
import {
  Building2,
  CalendarClock,
  Coins,
  CreditCard,
  RefreshCw,
  RotateCcw,
  Search,
  Settings,
  ShieldX,
  UserPlus,
  Users,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { formatNumber, formatPercent, formatTimestamp } from '@/lib/format'
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
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import {
  addDepartmentMember,
  createQuotaAllocation,
  deactivateDepartmentMember,
  createDepartmentBudget,
  departmentBudgetDetailQueryScopeKey,
  departmentBudgetDetailQueryKey,
  departmentBudgetListQueryScopeKey,
  departmentBudgetListQueryKey,
  departmentBudgetQueryKey,
  departmentMembersQueryKey,
  enterpriseOrganizationQueryKey,
  getDepartmentBudget,
  getDepartmentBudgetDetail,
  getDepartmentBudgets,
  getDepartmentMembers,
  getQuotaAllocations,
  getUserDepartments,
  quotaAllocationQueryKey,
  quotaAllocationQueryScopeKey,
  restoreDepartmentMember,
  revokeQuotaAllocation,
  replaceUserDepartments,
  userDepartmentsQueryKey,
} from './api'
import { DepartmentTree } from './components/DepartmentTree'
import { useDepartmentTree } from './hooks/use-department-tree'
import {
  findDepartmentNode,
  resolveDepartmentSelection,
  syncExpandedDepartmentIds,
  toggleExpandedDepartmentId,
} from './lib/tree-utils'
import type {
  ApiResponse,
  DepartmentBudgetDetailResponse,
  DepartmentBudgetItem,
  DepartmentBudgetSortField,
  DepartmentBudgetWalletDetail,
  DepartmentMemberItem,
  DepartmentTreeNode,
  EnterpriseBudgetErrorData,
  MembershipStatus,
  QuotaAllocationItem,
  UserDepartmentItem,
} from './types'

const enterpriseOrganizationRoute = '/_authenticated/enterprise-organization/'

export const enterpriseOrganizationSearchSchema = z.object({
  dept_id: z.coerce.number().int().positive().optional().catch(undefined),
  budget_id: z.coerce.number().int().positive().optional().catch(undefined),
})

export type EnterpriseOrganizationSearch = z.infer<
  typeof enterpriseOrganizationSearchSchema
>

export function normalizeEnterpriseOrganizationSearch(params: {
  departments: DepartmentTreeNode[]
  search: EnterpriseOrganizationSearch
}) {
  if (params.departments.length === 0) {
    return {
      ...params.search,
      dept_id: undefined,
      budget_id: undefined,
    }
  }

  const resolved = resolveDepartmentSelection(
    params.departments,
    params.search.dept_id
  )
  const normalizedDepartmentId = resolved.normalizedDepartmentId ?? undefined
  const shouldResetBudget = params.search.dept_id !== normalizedDepartmentId

  return {
    ...params.search,
    dept_id: normalizedDepartmentId,
    budget_id: shouldResetBudget ? undefined : params.search.budget_id,
  }
}

function parsePositiveInt(value: string) {
  const parsed = Number(value)
  if (!Number.isInteger(parsed) || parsed <= 0) return null
  return parsed
}

function parseDepartmentIds(value: string) {
  if (!value.trim()) return []
  const ids = value
    .split(',')
    .map((part) => Number(part.trim()))
    .filter((id) => Number.isInteger(id) && id > 0)
  return Array.from(new Set(ids))
}

export function createBudgetSchema(t: (key: string) => string) {
  return z
    .object({
      tenant_id: z.coerce.number().int().nonnegative(),
      department_id: z.coerce.number().int().positive(),
      type: z.enum(['balance', 'subscription']),
      total_quota: z.coerce.number().int().nonnegative(),
      cycle_quota: z.coerce.number().int().nonnegative(),
      cycle_type: z.enum(['daily', 'weekly', 'monthly', 'custom']),
      cycle_started_at: z.string().trim(),
      custom_seconds: z.coerce.number().int().nonnegative(),
      expires_at: z.string().trim(),
    })
    .superRefine((value, ctx) => {
      if (value.type === 'balance' && value.total_quota <= 0) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['total_quota'],
          message: t('Balance budget quota must be greater than 0'),
        })
      }
      if (value.type === 'subscription') {
        if (value.cycle_quota <= 0) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['cycle_quota'],
            message: t(
              'Subscription budget cycle quota must be greater than 0'
            ),
          })
        }
        if (!value.cycle_started_at) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['cycle_started_at'],
            message: t('Subscription budget cycle start time is required'),
          })
        }
        if (value.cycle_type === 'custom' && value.custom_seconds <= 0) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['custom_seconds'],
            message: t('Custom cycle must be greater than 0 seconds'),
          })
        }
      }
    })
}

export function createAllocationSchema(t: (key: string) => string) {
  return z.object({
    tenant_id: z.coerce.number().int().nonnegative(),
    department_id: z.coerce.number().int().positive(),
    department_budget_id: z.coerce.number().int().positive(),
    target_user_id: z.coerce.number().int().positive(),
    committed_quota: z.coerce
      .number()
      .int()
      .positive({
        message: t('Allocation quota must be greater than 0'),
      }),
    reason: z.string().trim().max(500).default(''),
  })
}

export function __testRenderApiMessage(
  result: { message?: string; data?: unknown } | null | undefined,
  translator: (key: string) => string
) {
  if (!result) return translator('Request failed')
  const maybeReason =
    result.data && typeof result.data === 'object' && 'reason' in result.data
      ? (result.data as EnterpriseBudgetErrorData).reason
      : undefined
  if (maybeReason) return translator(maybeReason)
  if (result.message) return translator(result.message)
  return translator('Request failed')
}

export function resolveEnterpriseOrganizationSelection(
  nodes: DepartmentTreeNode[],
  departmentId: number | null | undefined
) {
  return resolveDepartmentSelection(nodes, departmentId)
}

export function resolveBudgetSelection(
  availableBudgetIds: number[],
  selectedBudgetId: number | null | undefined,
  fallbackBudgetId: number | null | undefined
) {
  if (
    selectedBudgetId != null &&
    availableBudgetIds.includes(selectedBudgetId)
  ) {
    return selectedBudgetId
  }
  if (fallbackBudgetId != null && availableBudgetIds.includes(fallbackBudgetId)) {
    return fallbackBudgetId
  }
  return availableBudgetIds[0] ?? null
}

function statusVariant(status: MembershipStatus) {
  if (status === 1) return 'success'
  if (status === 2 || status === 3) return 'neutral'
  return 'warning'
}

function statusLabel(status: MembershipStatus, t: (key: string) => string) {
  if (status === 1) return t('Active')
  if (status === 2) return t('Inactive')
  if (status === 3) return t('Left')
  return t('Pending')
}

function enterpriseBudgetStatusLabel(
  status: string,
  t: (key: string) => string
) {
  if (status === 'active') return t('Active')
  if (status === 'paused') return t('Paused')
  if (status === 'revoked') return t('Revoked')
  if (status === 'expired') return t('Expired')
  return status || '-'
}

function enterpriseBudgetStatusVariant(status: string) {
  if (status === 'active') return 'success' as const
  if (status === 'paused') return 'warning' as const
  if (status === 'revoked' || status === 'expired') return 'danger' as const
  return 'neutral' as const
}

function thresholdStateLabel(status: string, t: (key: string) => string) {
  if (status === 'critical') return t('Critical')
  if (status === 'warning') return t('Warning')
  return t('Healthy')
}

function thresholdStateVariant(status: string) {
  if (status === 'critical') return 'danger' as const
  if (status === 'warning') return 'warning' as const
  return 'success' as const
}

function formatBudgetType(type: string, t: (key: string) => string) {
  return type === 'balance' ? t('Balance Budget') : t('Subscription Budget')
}

function MembershipStatusBadge({ status }: { status: MembershipStatus }) {
  const { t } = useTranslation()
  return (
    <StatusBadge
      label={statusLabel(status, t)}
      variant={statusVariant(status)}
      copyable={false}
    />
  )
}

function departmentStatusLabel(status: number, t: (key: string) => string) {
  if (status === 1) return t('Enabled')
  if (status === 2) return t('Disabled')
  if (status === 3) return t('Deleted upstream')
  return '-'
}

function departmentStatusVariant(status: number) {
  if (status === 1) return 'success' as const
  if (status === 2) return 'warning' as const
  if (status === 3) return 'danger' as const
  return 'neutral' as const
}

function departmentSourceLabel(
  sourceType: number,
  t: (key: string) => string
) {
  return sourceType === 2 ? t('DingTalk') : t('Manual')
}

function departmentSyncLabel(
  syncStatus: number,
  t: (key: string) => string
) {
  if (syncStatus === 1) return t('Synced')
  if (syncStatus === 2) return t('Sync warning')
  if (syncStatus === 3) return t('Sync failed')
  return t('Not synced')
}

function departmentSyncVariant(syncStatus: number) {
  if (syncStatus === 1) return 'success' as const
  if (syncStatus === 2) return 'warning' as const
  if (syncStatus === 3) return 'danger' as const
  return 'neutral' as const
}

export function EnterpriseOrganization() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const search = useSearch({
    from: enterpriseOrganizationRoute,
  }) as EnterpriseOrganizationSearch
  const {
    data: departments = [],
    isLoading,
    isFetching,
    refetch,
  } = useDepartmentTree()
  const resolvedSelection = useMemo(
    () => resolveDepartmentSelection(departments, search.dept_id),
    [departments, search.dept_id]
  )
  const normalizedSearch = useMemo(
    () => normalizeEnterpriseOrganizationSearch({ departments, search }),
    [departments, search]
  )
  const [expandedIds, setExpandedIds] = useState<number[]>([])

  useEffect(() => {
    setExpandedIds((current) =>
      syncExpandedDepartmentIds(
        current,
        departments,
        resolvedSelection.requiredExpandedIds
      )
    )
  }, [departments, resolvedSelection.requiredExpandedIds])

  useEffect(() => {
    if (
      search.dept_id === normalizedSearch.dept_id &&
      search.budget_id === normalizedSearch.budget_id
    ) {
      return
    }
    navigate({
      to: '/enterprise-organization',
      search: (prev) => ({
        ...prev,
        dept_id: normalizedSearch.dept_id,
        budget_id: normalizedSearch.budget_id,
      }),
      replace: true,
    })
  }, [
    navigate,
    normalizedSearch.budget_id,
    normalizedSearch.dept_id,
    search.budget_id,
    search.dept_id,
  ])

  const currentDepartment = useMemo(
    () => findDepartmentNode(departments, resolvedSelection.selectedDepartmentId),
    [departments, resolvedSelection.selectedDepartmentId]
  )
  const parentDepartment = useMemo(() => {
    if (!currentDepartment?.parent_id) return null
    return findDepartmentNode(departments, currentDepartment.parent_id)
  }, [currentDepartment, departments])

  const handleSelectDepartment = (departmentId: number) => {
    navigate({
      to: '/enterprise-organization',
      search: (prev) => ({
        ...prev,
        dept_id: departmentId,
        budget_id: undefined,
      }),
    })
  }

  const handleSelectedBudgetChange = (budgetId: number | null) => {
    const nextBudgetId = budgetId ?? undefined
    if (search.budget_id === nextBudgetId) return
    navigate({
      to: '/enterprise-organization',
      search: (prev) => ({
        ...prev,
        budget_id: nextBudgetId,
      }),
      replace: true,
    })
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Enterprise Organization')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          size='sm'
          onClick={() => refetch()}
          disabled={isFetching}
        >
          <RefreshCw className='size-4' />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        {isLoading || departments.length === 0 ? (
          <EnterpriseOrganizationContent
            isLoading={isLoading}
            departments={departments}
          />
        ) : (
          <div className='grid gap-4 xl:grid-cols-[minmax(320px,360px)_minmax(0,1fr)]'>
            <Card className='overflow-hidden'>
              <CardHeader>
                <CardTitle>{t('Department Tree')}</CardTitle>
                <CardDescription>
                  {t(
                    'Select a department from the tree to drive the governance workspace on the right.'
                  )}
                </CardDescription>
              </CardHeader>
            <CardContent className='px-0 pb-0'>
              <EnterpriseOrganizationContent
                isLoading={false}
                departments={departments}
                expandedIds={expandedIds}
                  selectedDepartmentId={currentDepartment?.id ?? null}
                  onToggleExpand={(departmentId) =>
                    setExpandedIds((current) =>
                      toggleExpandedDepartmentId(current, departmentId)
                    )
                  }
                  onSelectDepartment={handleSelectDepartment}
                />
              </CardContent>
            </Card>
            <EnterpriseOrganizationWorkspace
              currentDepartment={currentDepartment}
              parentDepartment={parentDepartment}
              selectedBudgetId={search.budget_id ?? null}
              onSelectedBudgetIdChange={handleSelectedBudgetChange}
            />
          </div>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

export function EnterpriseOrganizationWorkspace(props: {
  currentDepartment: DepartmentTreeNode | null
  parentDepartment: DepartmentTreeNode | null
  selectedBudgetId: number | null
  onSelectedBudgetIdChange: (budgetId: number | null) => void
}) {
  if (!props.currentDepartment) {
    return <EnterpriseOrganizationEmptyState />
  }

  return (
    <div className='space-y-4'>
      <DepartmentSummaryCard
        department={props.currentDepartment}
        parentDepartment={props.parentDepartment}
      />
      <DepartmentMembersPanel
        departmentId={props.currentDepartment.id}
        departmentName={props.currentDepartment.name}
      />
      <DepartmentBudgetPanel
        departmentId={props.currentDepartment.id}
        departmentName={props.currentDepartment.name}
        selectedBudgetId={props.selectedBudgetId}
        onSelectedBudgetIdChange={props.onSelectedBudgetIdChange}
      />
      <UserDepartmentsPanel />
    </div>
  )
}

export function EnterpriseOrganizationContent(props: {
  isLoading: boolean
  departments: ReturnType<typeof useDepartmentTree>['data']
  expandedIds?: number[]
  selectedDepartmentId?: number | null
  onToggleExpand?: (departmentId: number) => void
  onSelectDepartment?: (departmentId: number) => void
}) {
  if (props.isLoading) {
    return <DepartmentTreeSkeleton />
  }
  if (!props.departments || props.departments.length === 0) {
    return <EnterpriseOrganizationEmptyState />
  }
  return (
    <DepartmentTree
      nodes={props.departments}
      expandedIds={props.expandedIds}
      selectedDepartmentId={props.selectedDepartmentId}
      onToggleExpand={props.onToggleExpand}
      onSelectDepartment={props.onSelectDepartment}
    />
  )
}

function DepartmentTreeSkeleton() {
  return (
    <div className='space-y-2 rounded-lg border p-4'>
      {Array.from({ length: 6 }).map((_, index) => (
        <Skeleton key={index} className='h-10 w-full' />
      ))}
    </div>
  )
}

function EnterpriseOrganizationEmptyState() {
  const { t } = useTranslation()

  return (
    <Empty className='min-h-[360px] border'>
      <EmptyHeader>
        <EmptyMedia variant='icon'>
          <Building2 className='size-4' />
        </EmptyMedia>
        <EmptyTitle>{t('No departments yet')}</EmptyTitle>
        <EmptyDescription>
          {t(
            'Start by configuring DingTalk synchronization, or create departments manually when manual creation is available.'
          )}
        </EmptyDescription>
      </EmptyHeader>
      <EmptyContent>
        <div className='flex flex-wrap justify-center gap-2'>
          <Button variant='outline' render={<Link to='/enterprise-dingtalk' />}>
            <Settings className='size-4' />
            {t('Configure DingTalk sync')}
          </Button>
          <Button variant='outline' disabled>
            <UserPlus className='size-4' />
            {t('Manual creation coming soon')}
          </Button>
        </div>
      </EmptyContent>
    </Empty>
  )
}

export function DepartmentSummaryCard({
  department,
  parentDepartment,
}: {
  department: DepartmentTreeNode
  parentDepartment: DepartmentTreeNode | null
}) {
  const { t } = useTranslation()

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Current Department Workspace')}</CardTitle>
        <CardDescription>
          {t(
            'Everything below is scoped to the current department so you can manage members, budgets, and wallet allocations without leaving this page.'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div className='space-y-1'>
            <div className='text-xl font-semibold'>{department.name}</div>
            <div className='text-muted-foreground text-sm'>
              {t('Department ID')} #{department.id}
            </div>
          </div>
          <div className='flex flex-wrap gap-2'>
            <StatusBadge
              label={departmentStatusLabel(department.status, t)}
              variant={departmentStatusVariant(department.status)}
              copyable={false}
            />
            <StatusBadge
              label={departmentSyncLabel(department.sync_status, t)}
              variant={departmentSyncVariant(department.sync_status)}
              copyable={false}
            />
            <Badge variant='outline'>
              {departmentSourceLabel(department.source_type, t)}
            </Badge>
          </div>
        </div>
        <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-3'>
          <DepartmentMetaStat
            label={t('Parent Department')}
            value={
              parentDepartment
                ? `${parentDepartment.name} (#${parentDepartment.id})`
                : t('Root department')
            }
          />
          <DepartmentMetaStat
            label={t('External ID')}
            value={department.external_id || t('Local only')}
          />
          <DepartmentMetaStat
            label={t('Source')}
            value={departmentSourceLabel(department.source_type, t)}
          />
          <DepartmentMetaStat
            label={t('Sync State')}
            value={departmentSyncLabel(department.sync_status, t)}
          />
          <DepartmentMetaStat
            label={t('Created At')}
            value={department.created_at ? formatTimestamp(department.created_at) : '-'}
          />
          <DepartmentMetaStat
            label={t('Updated At')}
            value={department.updated_at ? formatTimestamp(department.updated_at) : '-'}
          />
        </div>
        <div className='rounded-lg border p-3'>
          <div className='text-muted-foreground text-xs'>
            {t('Historical Names')}
          </div>
          <div className='mt-2 text-sm'>
            {department.name_history.length > 0
              ? department.name_history
                  .map((entry) => entry.name)
                  .join(', ')
              : t('No historical names recorded')}
          </div>
          {department.sync_error ? (
            <div className='text-muted-foreground mt-2 text-xs'>
              {department.sync_error}
            </div>
          ) : null}
        </div>
      </CardContent>
    </Card>
  )
}

function DepartmentMetaStat({
  label,
  value,
}: {
  label: string
  value: string
}) {
  return (
    <div className='rounded-lg border p-3'>
      <div className='text-muted-foreground text-xs'>{label}</div>
      <div className='mt-1 text-sm font-medium'>{value}</div>
    </div>
  )
}

function UserDepartmentsPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [userIdText, setUserIdText] = useState('')
  const [departmentIdsText, setDepartmentIdsText] = useState('')
  const userId = useMemo(() => parsePositiveInt(userIdText), [userIdText])
  const departmentIds = useMemo(
    () => parseDepartmentIds(departmentIdsText),
    [departmentIdsText]
  )

  const userDepartmentsQuery = useQuery({
    queryKey: userDepartmentsQueryKey(userId),
    queryFn: async () => {
      if (!userId) return null
      const result = await getUserDepartments(userId)
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data ?? { items: [], total: 0, is_unassigned: true }
    },
    enabled: Boolean(userId),
  })

  const replaceMutation = useMutation({
    mutationFn: () => {
      if (!userId) throw new Error('missing user id')
      return replaceUserDepartments(userId, {
        department_ids: departmentIds,
        deactivate_stale: true,
      })
    },
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return
      }
      await queryClient.invalidateQueries({
        queryKey: enterpriseOrganizationQueryKey,
      })
      if (userId) {
        await queryClient.invalidateQueries({
          queryKey: userDepartmentsQueryKey(userId),
        })
      }
      toast.success(t('Department memberships updated'))
    },
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Membership Lookup')}</CardTitle>
        <CardDescription>
          {t(
            'Use this secondary tool when you need to inspect or replace memberships for a known user. The primary governance flow remains department-driven.'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className='flex flex-col gap-4'>
        <div className='flex flex-col gap-2 sm:flex-row'>
          <Input
            inputMode='numeric'
            value={userIdText}
            onChange={(event) => setUserIdText(event.target.value)}
            placeholder={t('User ID')}
          />
          <Input
            value={departmentIdsText}
            onChange={(event) => setDepartmentIdsText(event.target.value)}
            placeholder={t('Department IDs, comma separated')}
          />
          <Button
            onClick={() => replaceMutation.mutate()}
            disabled={!userId || replaceMutation.isPending}
          >
            <Search data-icon='inline-start' />
            {t('Replace Departments')}
          </Button>
        </div>
        <UserDepartmentsTable
          items={userDepartmentsQuery.data?.items ?? []}
          hasResult={Boolean(userDepartmentsQuery.data)}
        />
      </CardContent>
    </Card>
  )
}

function UserDepartmentsTable({
  items,
  hasResult,
}: {
  items: UserDepartmentItem[]
  hasResult: boolean
}) {
  const { t } = useTranslation()

  if (!hasResult) return null

  if (items.length === 0) {
    return (
      <Empty className='min-h-[220px]'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Users />
          </EmptyMedia>
          <EmptyTitle>{t('Unassigned')}</EmptyTitle>
          <EmptyDescription>
            {t('This user has no department memberships.')}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('Department')}</TableHead>
          <TableHead>{t('Membership Status')}</TableHead>
          <TableHead>{t('External Source')}</TableHead>
          <TableHead>{t('Joined At')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.id}>
            <TableCell>
              <div className='flex min-w-[160px] flex-col gap-1'>
                <span className='font-medium'>
                  {item.department_name || `#${item.department_id}`}
                </span>
                <span className='text-muted-foreground text-xs'>
                  {t('Department ID')} #{item.department_id}
                </span>
              </div>
            </TableCell>
            <TableCell>
              <MembershipStatusBadge status={item.status} />
            </TableCell>
            <TableCell>
              <Badge variant='secondary'>
                {item.external_source || 'manual'}
              </Badge>
            </TableCell>
            <TableCell>
              {item.joined_at ? formatTimestamp(item.joined_at) : '-'}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function DepartmentMembersPanel({
  departmentId,
  departmentName,
}: {
  departmentId: number
  departmentName: string
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [memberUserIdText, setMemberUserIdText] = useState('')

  const departmentMembersQuery = useQuery({
    queryKey: departmentMembersQueryKey(departmentId),
    queryFn: async () => {
      const result = await getDepartmentMembers(departmentId)
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data ?? { items: [], total: 0 }
    },
  })

  const addMemberMutation = useMutation({
    mutationFn: () => {
      const memberUserId = parsePositiveInt(memberUserIdText)
      if (!memberUserId) throw new Error('missing ids')
      return addDepartmentMember(departmentId, { user_id: memberUserId })
    },
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return
      }
      setMemberUserIdText('')
      await queryClient.invalidateQueries({
        queryKey: departmentMembersQueryKey(departmentId),
      })
      toast.success(t('Department member added'))
    },
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Department Members')}</CardTitle>
        <CardDescription>
          {t(
            'Manage the member list for {{department}} without re-entering a department identifier.',
            {
              department: departmentName,
            }
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className='flex flex-col gap-4'>
        <div className='flex flex-col gap-2 sm:flex-row'>
          <Input
            inputMode='numeric'
            value={memberUserIdText}
            onChange={(event) => setMemberUserIdText(event.target.value)}
            placeholder={t('User ID')}
          />
          <Button
            onClick={() => addMemberMutation.mutate()}
            disabled={
              !parsePositiveInt(memberUserIdText) || addMemberMutation.isPending
            }
          >
            <UserPlus data-icon='inline-start' />
            {t('Add Member')}
          </Button>
        </div>
        <DepartmentMembersTable
          departmentId={departmentId}
          items={departmentMembersQuery.data?.items ?? []}
        />
      </CardContent>
    </Card>
  )
}

function DepartmentMembersTable({
  items,
  departmentId,
}: {
  items: DepartmentMemberItem[]
  departmentId: number
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const deactivateMutation = useMutation({
    mutationFn: (userId: number) =>
      deactivateDepartmentMember(departmentId, userId),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return
      }
      await queryClient.invalidateQueries({
        queryKey: departmentMembersQueryKey(departmentId),
      })
    },
  })
  const restoreMutation = useMutation({
    mutationFn: (userId: number) =>
      restoreDepartmentMember(departmentId, userId),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return
      }
      await queryClient.invalidateQueries({
        queryKey: departmentMembersQueryKey(departmentId),
      })
    },
  })

  if (items.length === 0) {
    return (
      <Empty className='min-h-[220px]'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Building2 />
          </EmptyMedia>
          <EmptyTitle>{t('No Department Members')}</EmptyTitle>
          <EmptyDescription>
            {t('This department has no members yet.')}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('User')}</TableHead>
          <TableHead>{t('Membership Status')}</TableHead>
          <TableHead>{t('External Source')}</TableHead>
          <TableHead className='text-right'>{t('Actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.id}>
            <TableCell>
              <div className='flex min-w-[160px] flex-col gap-1'>
                <span className='font-medium'>
                  {item.username || `#${item.user_id}`}
                </span>
                <span className='text-muted-foreground text-xs'>
                  {item.display_name || `${t('User ID')} #${item.user_id}`}
                </span>
              </div>
            </TableCell>
            <TableCell>
              <MembershipStatusBadge status={item.status} />
            </TableCell>
            <TableCell>
              <Badge variant='secondary'>
                {item.external_source || 'manual'}
              </Badge>
            </TableCell>
            <TableCell className='text-right'>
              {item.status === 1 ? (
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => deactivateMutation.mutate(item.user_id)}
                  disabled={deactivateMutation.isPending}
                >
                  {t('Deactivate')}
                </Button>
              ) : (
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => restoreMutation.mutate(item.user_id)}
                  disabled={restoreMutation.isPending}
                >
                  <RotateCcw data-icon='inline-start' />
                  {t('Restore')}
                </Button>
              )}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function DepartmentBudgetPanel({
  departmentId,
  departmentName,
  selectedBudgetId,
  onSelectedBudgetIdChange,
}: {
  departmentId: number
  departmentName: string
  selectedBudgetId: number | null
  onSelectedBudgetIdChange: (budgetId: number | null) => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const renderApiMessage = (result: ApiResponse<unknown> | null | undefined) =>
    __testRenderApiMessage(result, t)
  const budgetSchema = createBudgetSchema(t)
  const allocationSchema = createAllocationSchema(t)
  type BudgetFormValues = z.infer<typeof budgetSchema>
  type AllocationFormValues = z.infer<typeof allocationSchema>

  const form = useForm<BudgetFormValues>({
    resolver: zodResolver(
      budgetSchema
    ) as unknown as Resolver<BudgetFormValues>,
    defaultValues: {
      tenant_id: 0,
      department_id: departmentId,
      type: 'balance',
      total_quota: 0,
      cycle_quota: 0,
      cycle_type: 'monthly',
      cycle_started_at: '',
      custom_seconds: 0,
      expires_at: '',
    },
  })
  const allocationForm = useForm<AllocationFormValues>({
    resolver: zodResolver(
      allocationSchema
    ) as unknown as Resolver<AllocationFormValues>,
    defaultValues: {
      tenant_id: 0,
      department_id: departmentId,
      department_budget_id: 0,
      target_user_id: 0,
      committed_quota: 0,
      reason: '',
    },
  })
  const tenantId = form.watch('tenant_id')
  const budgetType = form.watch('type')
  const cycleType = form.watch('cycle_type')
  const [sortBy, setSortBy] = useState<DepartmentBudgetSortField>('usage_ratio')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')
  const normalizedTenantId = tenantId || 0

  const budgetQuery = useQuery({
    queryKey: departmentBudgetQueryKey(departmentId, normalizedTenantId),
    queryFn: async () => {
      const result = await getDepartmentBudget(departmentId, tenantId)
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data?.item ?? null
    },
  })

  const currentBudgetId = budgetQuery.data?.id ?? 0

  const budgetListQuery = useQuery({
    queryKey: departmentBudgetListQueryKey(
      departmentId,
      normalizedTenantId,
      sortBy,
      sortOrder
    ),
    queryFn: async () => {
      const result = await getDepartmentBudgets(departmentId, {
        tenant_id: tenantId || undefined,
        sort_by: sortBy,
        sort_order: sortOrder,
      })
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return (
        result.data ?? {
          items: [],
          thresholds: { warning: 80, critical: 95 },
        }
      )
    },
  })

  const effectiveBudgetId = resolveBudgetSelection(
    budgetListQuery.data?.items.map((item) => item.id) ?? [],
    selectedBudgetId,
    currentBudgetId || null
  )

  const budgetDetailQuery = useQuery({
    queryKey: departmentBudgetDetailQueryKey(
      departmentId,
      normalizedTenantId,
      effectiveBudgetId
    ),
    queryFn: async (): Promise<DepartmentBudgetDetailResponse> => {
      if (!effectiveBudgetId) {
        return {
          budget: null,
          wallets: [],
          thresholds: budgetListQuery.data?.thresholds ?? {
            warning: 80,
            critical: 95,
          },
        }
      }
      const result = await getDepartmentBudgetDetail(
        departmentId,
        effectiveBudgetId,
        tenantId || undefined
      )
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return (
        result.data ?? {
          budget: null,
          wallets: [],
          thresholds: budgetListQuery.data?.thresholds ?? {
            warning: 80,
            critical: 95,
          },
        }
      )
    },
    enabled: Boolean(effectiveBudgetId),
  })

  const allocationListQuery = useQuery({
    queryKey: quotaAllocationQueryKey(
      departmentId,
      normalizedTenantId,
      effectiveBudgetId
    ),
    queryFn: async () => {
      if (!effectiveBudgetId) return []
      const result = await getQuotaAllocations(
        effectiveBudgetId,
        tenantId,
        departmentId
      )
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data?.items ?? []
    },
    enabled: Boolean(effectiveBudgetId),
  })

  const createMutation = useMutation({
    mutationFn: async (values: BudgetFormValues) => {
      const payload =
        values.type === 'balance'
          ? {
              tenant_id: Number(values.tenant_id),
              type: 'balance' as const,
              total_quota: Number(values.total_quota),
              expires_at: parseDateTimeToUnix(values.expires_at),
            }
          : {
              tenant_id: Number(values.tenant_id),
              type: 'subscription' as const,
              cycle_quota: Number(values.cycle_quota),
              cycle_type: values.cycle_type,
              cycle_started_at: parseRequiredDateTimeToUnix(
                values.cycle_started_at
              ),
              custom_seconds:
                values.cycle_type === 'custom'
                  ? Number(values.custom_seconds)
                  : undefined,
              expires_at: parseDateTimeToUnix(values.expires_at),
            }
      return createDepartmentBudget(departmentId, payload)
    },
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(renderApiMessage(result))
        return
      }
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetQueryKey(departmentId, normalizedTenantId),
      })
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetListQueryScopeKey(
          departmentId,
          normalizedTenantId
        ),
      })
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetDetailQueryScopeKey(
          departmentId,
          normalizedTenantId
        ),
      })
      const budgetId = result.data?.item?.id ?? null
      onSelectedBudgetIdChange(budgetId)
      toast.success(t('Department budget saved'))
    },
  })

  const allocationMutation = useMutation({
    mutationFn: async (values: AllocationFormValues) =>
      createQuotaAllocation({
        tenant_id: Number(values.tenant_id),
        department_id: departmentId,
        department_budget_id: Number(values.department_budget_id),
        target_user_id: Number(values.target_user_id),
        committed_quota: Number(values.committed_quota),
        reason: values.reason.trim() || undefined,
      }),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(renderApiMessage(result))
        return
      }
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetQueryKey(departmentId, normalizedTenantId),
      })
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetListQueryScopeKey(
          departmentId,
          normalizedTenantId
        ),
      })
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetDetailQueryScopeKey(
          departmentId,
          normalizedTenantId
        ),
      })
      await queryClient.invalidateQueries({
        queryKey: quotaAllocationQueryScopeKey(
          departmentId,
          normalizedTenantId
        ),
      })
      toast.success(t('Wallet allocation created'))
    },
  })

  const revokeMutation = useMutation({
    mutationFn: async (allocationId: number) =>
      revokeQuotaAllocation(allocationId, {
        tenant_id: tenantId || undefined,
        department_id: departmentId,
      }),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(renderApiMessage(result))
        return
      }
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetQueryKey(departmentId, normalizedTenantId),
      })
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetListQueryScopeKey(
          departmentId,
          normalizedTenantId
        ),
      })
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetDetailQueryScopeKey(
          departmentId,
          normalizedTenantId
        ),
      })
      await queryClient.invalidateQueries({
        queryKey: quotaAllocationQueryScopeKey(
          departmentId,
          normalizedTenantId
        ),
      })
      toast.success(t('Wallet allocation revoked'))
    },
  })

  useEffect(() => {
    if (form.getValues('department_id') !== departmentId) {
      form.setValue('department_id', departmentId)
    }
    if (allocationForm.getValues('department_id') !== departmentId) {
      allocationForm.setValue('department_id', departmentId)
    }
  }, [allocationForm, departmentId, form])

  useEffect(() => {
    if (allocationForm.getValues('tenant_id') !== tenantId) {
      allocationForm.setValue('tenant_id', tenantId)
    }
  }, [allocationForm, tenantId])

  useEffect(() => {
    const nextBudgetId = effectiveBudgetId ?? 0
    if (allocationForm.getValues('department_budget_id') !== nextBudgetId) {
      allocationForm.setValue('department_budget_id', nextBudgetId)
    }
  }, [allocationForm, effectiveBudgetId])

  useEffect(() => {
    const normalizedBudgetId = effectiveBudgetId ?? null
    if (selectedBudgetId !== normalizedBudgetId) {
      onSelectedBudgetIdChange(normalizedBudgetId)
    }
  }, [effectiveBudgetId, onSelectedBudgetIdChange, selectedBudgetId])

  return (
    <div className='space-y-4'>
      <div className='grid gap-4 xl:grid-cols-[minmax(0,360px)_minmax(0,1fr)]'>
        <Card>
          <CardHeader>
            <CardTitle>{t('Department Budget')}</CardTitle>
            <CardDescription>
              {t(
                'Create a balance or subscription budget pool for {{department}} without leaving the current department context.',
                {
                  department: departmentName,
                }
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Form {...form}>
              <form
                className='flex flex-col gap-4'
                onSubmit={form.handleSubmit((values) =>
                  createMutation.mutate(values)
                )}
              >
                <FormField
                  control={form.control}
                  name='tenant_id'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Tenant ID')}</FormLabel>
                      <FormControl>
                        <Input inputMode='numeric' {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='type'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Budget Type')}</FormLabel>
                      <Select
                        value={field.value}
                        onValueChange={(value) =>
                          field.onChange(value as 'balance' | 'subscription')
                        }
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectItem value='balance'>
                            {t('Balance Budget')}
                          </SelectItem>
                          <SelectItem value='subscription'>
                            {t('Subscription Budget')}
                          </SelectItem>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                {budgetType === 'balance' ? (
                  <FormField
                    control={form.control}
                    name='total_quota'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Total Quota')}</FormLabel>
                        <FormControl>
                          <Input inputMode='numeric' {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                ) : (
                  <>
                    <FormField
                      control={form.control}
                      name='cycle_quota'
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('Cycle Quota')}</FormLabel>
                          <FormControl>
                            <Input inputMode='numeric' {...field} />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name='cycle_type'
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('Cycle Type')}</FormLabel>
                          <Select
                            value={field.value}
                            onValueChange={field.onChange}
                          >
                            <FormControl>
                              <SelectTrigger>
                                <SelectValue />
                              </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                              <SelectItem value='daily'>{t('Daily')}</SelectItem>
                              <SelectItem value='weekly'>
                                {t('Weekly')}
                              </SelectItem>
                              <SelectItem value='monthly'>
                                {t('Monthly')}
                              </SelectItem>
                              <SelectItem value='custom'>
                                {t('Custom (seconds)')}
                              </SelectItem>
                            </SelectContent>
                          </Select>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name='cycle_started_at'
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('Cycle Start Time')}</FormLabel>
                          <FormControl>
                            <Input
                              placeholder='2026-05-29T12:00'
                              type='datetime-local'
                              {...field}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    {cycleType === 'custom' ? (
                      <FormField
                        control={form.control}
                        name='custom_seconds'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Custom Cycle Seconds')}</FormLabel>
                            <FormControl>
                              <Input inputMode='numeric' {...field} />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    ) : null}
                  </>
                )}
                <FormField
                  control={form.control}
                  name='expires_at'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Expires At (optional)')}</FormLabel>
                      <FormControl>
                        <Input type='datetime-local' {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <Button type='submit' disabled={createMutation.isPending}>
                  <Coins data-icon='inline-start' />
                  {t('Create Budget Pool')}
                </Button>
              </form>
            </Form>
          </CardContent>
        </Card>
        <DepartmentBudgetOverviewCard
          item={budgetQuery.data ?? null}
          selectedBudget={budgetDetailQuery.data?.budget ?? null}
          thresholds={
            budgetDetailQuery.data?.thresholds ??
            budgetListQuery.data?.thresholds ?? { warning: 80, critical: 95 }
          }
        />
      </div>
      <DepartmentBudgetListCard
        items={budgetListQuery.data?.items ?? []}
        loading={budgetListQuery.isLoading}
        selectedBudgetId={effectiveBudgetId}
        sortBy={sortBy}
        sortOrder={sortOrder}
        onSelectBudget={onSelectedBudgetIdChange}
        onSortByChange={setSortBy}
        onSortOrderChange={setSortOrder}
      />
      <Card>
        <CardHeader>
          <CardTitle>{t('Budget Wallet Detail')}</CardTitle>
          <CardDescription>
            {t(
              'Track the selected budget pool, its derived wallets, and the source allocation chain in one place.'
            )}
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <DepartmentBudgetDetailTable
            budget={budgetDetailQuery.data?.budget ?? null}
            wallets={budgetDetailQuery.data?.wallets ?? []}
            loading={budgetDetailQuery.isLoading}
          />
          <Form {...allocationForm}>
            <form
              className='grid gap-4 md:grid-cols-2'
              onSubmit={allocationForm.handleSubmit((values) =>
                allocationMutation.mutate(values)
              )}
            >
              <FormField
                control={allocationForm.control}
                name='target_user_id'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Target User ID')}</FormLabel>
                    <FormControl>
                      <Input inputMode='numeric' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={allocationForm.control}
                name='committed_quota'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Allocation Quota')}</FormLabel>
                    <FormControl>
                      <Input inputMode='numeric' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={allocationForm.control}
                name='reason'
                render={({ field }) => (
                  <FormItem className='md:col-span-2'>
                    <FormLabel>{t('Reason')}</FormLabel>
                    <FormControl>
                      <Textarea
                        {...field}
                        value={field.value ?? ''}
                        placeholder={t('Optional allocation note')}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className='flex justify-end md:col-span-2'>
                <Button
                  type='submit'
                  disabled={!effectiveBudgetId || allocationMutation.isPending}
                >
                  <CreditCard data-icon='inline-start' />
                  {t('Create wallet allocation')}
                </Button>
              </div>
            </form>
          </Form>
          <QuotaAllocationTable
            items={allocationListQuery.data ?? []}
            loading={allocationListQuery.isLoading}
            onRevoke={(allocationId) => revokeMutation.mutate(allocationId)}
            revokePendingId={
              revokeMutation.isPending ? revokeMutation.variables : null
            }
          />
        </CardContent>
      </Card>
    </div>
  )
}

export function QuotaAllocationTable({
  items,
  loading,
  onRevoke,
  revokePendingId,
}: {
  items: QuotaAllocationItem[]
  loading: boolean
  onRevoke?: (allocationId: number) => void
  revokePendingId?: number | null
}) {
  const { t } = useTranslation()

  if (loading) {
    return <Skeleton className='h-32 w-full' />
  }
  if (items.length === 0) {
    return (
      <Empty className='min-h-[180px] border'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Coins className='size-4' />
          </EmptyMedia>
          <EmptyTitle>{t('No wallet allocations yet')}</EmptyTitle>
          <EmptyDescription>
            {t(
              'Create an allocation to place a department wallet ahead of the member primary wallet.'
            )}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('Target User ID')}</TableHead>
          <TableHead>{t('Allocation Quota')}</TableHead>
          <TableHead>{t('Wallet ID')}</TableHead>
          <TableHead>{t('Status')}</TableHead>
          <TableHead>{t('Processed At')}</TableHead>
          <TableHead>{t('Created At')}</TableHead>
          <TableHead>{t('Actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.id}>
            <TableCell>{item.target_user_id}</TableCell>
            <TableCell>{item.committed_quota}</TableCell>
            <TableCell>{item.wallet_id}</TableCell>
            <TableCell>
              <Badge variant='secondary'>
                {enterpriseBudgetStatusLabel(item.status, t)}
              </Badge>
            </TableCell>
            <TableCell>
              {item.processed_at ? formatTimestamp(item.processed_at) : '-'}
            </TableCell>
            <TableCell>{formatTimestamp(item.created_at)}</TableCell>
            <TableCell>
              <Button
                type='button'
                size='sm'
                variant='outline'
                disabled={
                  !onRevoke ||
                  item.status === 'revoked' ||
                  item.status === 'expired' ||
                  revokePendingId === item.id
                }
                onClick={() => onRevoke?.(item.id)}
              >
                <ShieldX data-icon='inline-start' />
                {item.status === 'revoked' || item.status === 'expired'
                  ? t('Already processed')
                  : t('Revoke allocation')}
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

export function DepartmentBudgetListCard({
  items,
  loading,
  selectedBudgetId,
  sortBy,
  sortOrder,
  onSelectBudget,
  onSortByChange,
  onSortOrderChange,
}: {
  items: DepartmentBudgetItem[]
  loading: boolean
  selectedBudgetId: number | null
  sortBy: DepartmentBudgetSortField
  sortOrder: 'asc' | 'desc'
  onSelectBudget: (budgetId: number | null) => void
  onSortByChange: (field: DepartmentBudgetSortField) => void
  onSortOrderChange: (order: 'asc' | 'desc') => void
}) {
  const { t } = useTranslation()

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Budget Pool List')}</CardTitle>
        <CardDescription>
          {t(
            'Sort budget pools by usage, remaining quota, type, or status and open one detail view at a time.'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        <div className='grid gap-3 md:grid-cols-2'>
          <div className='space-y-2'>
            <Label>{t('Sort By')}</Label>
            <Select
              value={sortBy}
              onValueChange={(value) =>
                onSortByChange(value as DepartmentBudgetSortField)
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='usage_ratio'>{t('Usage Ratio')}</SelectItem>
                <SelectItem value='remaining'>
                  {t('Remaining Quota')}
                </SelectItem>
                <SelectItem value='type'>{t('Budget Type')}</SelectItem>
                <SelectItem value='status'>{t('Budget Status')}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className='space-y-2'>
            <Label>{t('Sort Order')}</Label>
            <Select
              value={sortOrder}
              onValueChange={(value) =>
                onSortOrderChange(value as 'asc' | 'desc')
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='desc'>{t('Descending')}</SelectItem>
                <SelectItem value='asc'>{t('Ascending')}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        {loading ? (
          <Skeleton className='h-40 w-full' />
        ) : items.length === 0 ? (
          <Empty className='min-h-[220px] border'>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <Coins className='size-4' />
              </EmptyMedia>
              <EmptyTitle>{t('No budget pools yet')}</EmptyTitle>
              <EmptyDescription>
                {t(
                  'Create the first pool for this department to unlock health monitoring and wallet tracing.'
                )}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Budget Type')}</TableHead>
                <TableHead>{t('Remaining Quota')}</TableHead>
                <TableHead>{t('Allocated Total')}</TableHead>
                <TableHead>{t('Usage Ratio')}</TableHead>
                <TableHead>{t('Budget Status')}</TableHead>
                <TableHead>{t('Threshold State')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((item) => {
                const selected = selectedBudgetId === item.id
                return (
                  <TableRow
                    key={item.id}
                    className={selected ? 'bg-muted/50' : undefined}
                    onClick={() => onSelectBudget(item.id)}
                  >
                    <TableCell>
                      <div className='flex min-w-[160px] flex-col gap-1'>
                        <span className='font-medium'>
                          {formatBudgetType(item.type, t)}
                        </span>
                        <span className='text-muted-foreground text-xs'>
                          #{item.id}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>{formatNumber(item.remaining)}</TableCell>
                    <TableCell>{formatNumber(item.allocated_total)}</TableCell>
                    <TableCell>{formatPercent(item.usage_ratio)}</TableCell>
                    <TableCell>
                      <StatusBadge
                        label={enterpriseBudgetStatusLabel(item.status, t)}
                        variant={enterpriseBudgetStatusVariant(item.status)}
                        copyable={false}
                      />
                    </TableCell>
                    <TableCell>
                      <StatusBadge
                        label={thresholdStateLabel(item.threshold_state, t)}
                        variant={thresholdStateVariant(item.threshold_state)}
                        copyable={false}
                      />
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}

export function DepartmentBudgetDetailTable({
  budget,
  wallets,
  loading,
}: {
  budget: DepartmentBudgetItem | null
  wallets: DepartmentBudgetWalletDetail[]
  loading: boolean
}) {
  const { t } = useTranslation()

  if (loading) {
    return <Skeleton className='h-36 w-full' />
  }

  if (!budget) {
    return (
      <Empty className='min-h-[180px] border'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <CreditCard className='size-4' />
          </EmptyMedia>
          <EmptyTitle>{t('Select a budget pool')}</EmptyTitle>
          <EmptyDescription>
            {t(
              'Choose a budget pool from the list to inspect derived wallets and allocation lineage.'
            )}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  if (wallets.length === 0) {
    return (
      <Empty className='min-h-[180px] border'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Users className='size-4' />
          </EmptyMedia>
          <EmptyTitle>{t('No derived wallets yet')}</EmptyTitle>
          <EmptyDescription>
            {t(
              'This budget pool has no derived wallets yet. Create an allocation below to start tracking wallet state.'
            )}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('Target User')}</TableHead>
          <TableHead>{t('Wallet')}</TableHead>
          <TableHead>{t('Quota')}</TableHead>
          <TableHead>{t('Remain Quota')}</TableHead>
          <TableHead>{t('Cycle / Expiry')}</TableHead>
          <TableHead>{t('Wallet Status')}</TableHead>
          <TableHead>{t('Allocation Status')}</TableHead>
          <TableHead>{t('Source')}</TableHead>
          <TableHead>{t('Processed At')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {wallets.map((wallet) => (
          <TableRow key={wallet.allocation_id}>
            <TableCell>
              <div className='flex min-w-[180px] flex-col gap-1'>
                <span className='font-medium'>
                  {wallet.target_display_name ||
                    wallet.target_username ||
                    `#${wallet.target_user_id}`}
                </span>
                <span className='text-muted-foreground text-xs'>
                  {t('User ID')} #{wallet.target_user_id}
                </span>
              </div>
            </TableCell>
            <TableCell>
              <div className='flex min-w-[120px] flex-col gap-1'>
                <span className='font-medium'>#{wallet.wallet_id}</span>
                <span className='text-muted-foreground text-xs'>
                  {formatBudgetType(wallet.source_parent_budget_type, t)}
                </span>
              </div>
            </TableCell>
            <TableCell>{formatNumber(wallet.quota)}</TableCell>
            <TableCell>{formatNumber(wallet.remain_quota)}</TableCell>
            <TableCell>
              <div className='flex min-w-[180px] flex-col gap-1 text-sm'>
                <span>{formatBudgetCycleType(wallet.cycle_type, t)}</span>
                <span className='text-muted-foreground text-xs'>
                  {wallet.next_reset_time
                    ? `${t('Next Reset')}: ${formatTimestamp(wallet.next_reset_time)}`
                    : wallet.expires_at
                      ? `${t('Expires At (optional)')}: ${formatTimestamp(wallet.expires_at)}`
                      : '-'}
                </span>
              </div>
            </TableCell>
            <TableCell>
              <StatusBadge
                label={enterpriseBudgetStatusLabel(wallet.wallet_status, t)}
                variant={enterpriseBudgetStatusVariant(wallet.wallet_status)}
                copyable={false}
              />
            </TableCell>
            <TableCell>
              <StatusBadge
                label={enterpriseBudgetStatusLabel(wallet.allocation_status, t)}
                variant={enterpriseBudgetStatusVariant(
                  wallet.allocation_status
                )}
                copyable={false}
              />
            </TableCell>
            <TableCell>
              <div className='flex min-w-[160px] flex-col gap-1 text-sm'>
                <span>
                  {t('Allocation')} #{wallet.source_allocation_id}
                </span>
                <span className='text-muted-foreground text-xs'>
                  {t('Parent Budget')} #{wallet.source_parent_budget_id}
                </span>
              </div>
            </TableCell>
            <TableCell>
              {wallet.processed_at ? formatTimestamp(wallet.processed_at) : '-'}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

export function DepartmentBudgetOverviewCard({
  item,
  selectedBudget,
  thresholds,
}: {
  item: DepartmentBudgetItem | null
  selectedBudget: DepartmentBudgetItem | null
  thresholds: { warning: number; critical: number }
}) {
  const { t } = useTranslation()
  const budget = selectedBudget ?? item

  if (!budget) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t('Budget Pool Overview')}</CardTitle>
          <CardDescription>
            {t('The selected department does not have a budget pool yet.')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Empty className='min-h-[280px]'>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <CreditCard className='size-4' />
              </EmptyMedia>
              <EmptyTitle>{t('No budget pool yet')}</EmptyTitle>
              <EmptyDescription>
                {t(
                  'Create a budget pool to view the department budget state here.'
                )}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Budget Pool Overview')}</CardTitle>
        <CardDescription>
          {t(
            'Shows the selected budget pool health, usage thresholds, and lifecycle status.'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className='grid gap-3 sm:grid-cols-2'>
        <BudgetStat
          label={t('Budget Type')}
          value={formatBudgetType(budget.type, t)}
        />
        <BudgetStat
          label={t('Budget Status')}
          value={enterpriseBudgetStatusLabel(budget.status, t)}
        />
        <BudgetStat
          label={t('Threshold State')}
          value={thresholdStateLabel(budget.threshold_state, t)}
          badgeVariant={thresholdStateVariant(budget.threshold_state)}
        />
        <BudgetStat
          label={t('Usage Ratio')}
          value={formatPercent(budget.usage_ratio)}
        />
        <BudgetStat
          label={t('Remaining Quota')}
          value={formatNumber(budget.remaining)}
        />
        <BudgetStat
          label={t('Allocated Total')}
          value={formatNumber(budget.allocated_total)}
        />
        <BudgetStat
          label={t('Total Quota')}
          value={formatNumber(budget.total_quota)}
        />
        <BudgetStat
          label={t('Cycle Quota')}
          value={formatNumber(budget.cycle_quota)}
        />
        <BudgetStat
          label={t('Cycle Type')}
          value={formatBudgetCycleType(budget.cycle_type, t)}
        />
        <BudgetStat
          label={t('Cycle Start Time')}
          value={
            budget.cycle_started_at
              ? formatTimestamp(budget.cycle_started_at)
              : '-'
          }
        />
        <BudgetStat
          label={t('Custom Cycle Seconds')}
          value={
            budget.custom_seconds ? formatNumber(budget.custom_seconds) : '-'
          }
        />
        <BudgetStat
          label={t('Expires At (optional)')}
          value={budget.expires_at ? formatTimestamp(budget.expires_at) : '-'}
        />
        <BudgetStat
          label={t('Threshold Window')}
          value={`${thresholds.warning}% / ${thresholds.critical}%`}
        />
        <BudgetStat
          label={t('Parent Status')}
          value={enterpriseBudgetStatusLabel(budget.parent_status, t)}
        />
      </CardContent>
    </Card>
  )
}

export function DepartmentBudgetStatusCard({
  item,
}: {
  item: DepartmentBudgetItem | null
}) {
  return (
    <DepartmentBudgetOverviewCard
      item={item}
      selectedBudget={item}
      thresholds={{ warning: 80, critical: 95 }}
    />
  )
}

function BudgetStat({
  label,
  value,
  badgeVariant,
}: {
  label: string
  value: string
  badgeVariant?: 'success' | 'warning' | 'danger' | 'neutral'
}) {
  return (
    <div className='rounded-lg border p-3'>
      <div className='text-muted-foreground text-xs'>{label}</div>
      <div className='mt-1 flex items-center gap-2 font-medium'>
        <CalendarClock className='text-muted-foreground size-4' />
        {badgeVariant ? (
          <StatusBadge label={value} variant={badgeVariant} copyable={false} />
        ) : (
          <span>{value}</span>
        )}
      </div>
    </div>
  )
}

function parseDateTimeToUnix(value: string) {
  if (!value.trim()) return undefined
  const ts = Date.parse(value)
  if (Number.isNaN(ts)) return undefined
  return Math.floor(ts / 1000)
}

function parseRequiredDateTimeToUnix(value: string) {
  const ts = parseDateTimeToUnix(value)
  return ts ?? 0
}

function formatBudgetCycleType(cycleType: string, t: (key: string) => string) {
  if (cycleType === 'daily') return t('Daily')
  if (cycleType === 'weekly') return t('Weekly')
  if (cycleType === 'monthly') return t('Monthly')
  if (cycleType === 'custom') return t('Custom (seconds)')
  if (cycleType === 'never') return t('No Reset')
  return cycleType || '-'
}
