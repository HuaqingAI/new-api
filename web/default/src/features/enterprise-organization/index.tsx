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
  ShieldCheck,
  RefreshCw,
  RotateCcw,
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
  denyDepartmentOwner,
  departmentBudgetDetailQueryScopeKey,
  departmentBudgetDetailQueryKey,
  departmentBudgetListQueryScopeKey,
  departmentBudgetListQueryKey,
  departmentBudgetQueryKey,
  departmentMembersQueryKey,
  departmentOwnersQueryKey,
  getDepartmentOwners,
  getDepartmentBudget,
  getDepartmentBudgetDetail,
  getDepartmentBudgets,
  getDepartmentMembers,
  getQuotaAllocations,
  getUserDepartments,
  grantDepartmentOwner,
  quotaAllocationQueryKey,
  quotaAllocationQueryScopeKey,
  renameDepartmentMember,
  restoreDepartmentMember,
  revokeDepartmentOwnerDeny,
  revokeDepartmentOwnerGrant,
  revokeQuotaAllocation,
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
import {
  formatEnterpriseUserPrimary,
  formatEnterpriseUserSecondary,
} from './lib/user-display'
import type {
  ApiResponse,
  DepartmentBudgetDetailResponse,
  DepartmentBudgetItem,
  DepartmentBudgetSortField,
  DepartmentBudgetWalletDetail,
  DepartmentMemberItem,
  DepartmentOwnersResponse,
  EffectiveDepartmentOwnerItem,
  DepartmentOwnerFactItem,
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

export function normalizeMemberSelection(params: {
  members: DepartmentMemberItem[]
  selectedUserId: number | null | undefined
}) {
  if (
    params.selectedUserId != null &&
    params.members.some((item) => item.user_id === params.selectedUserId)
  ) {
    return params.selectedUserId
  }
  return null
}

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

export function resolveDepartmentMemberSelection(
  members: DepartmentMemberItem[],
  selectedMemberUserId: number | null | undefined
) {
  if (
    selectedMemberUserId != null &&
    members.some((item) => item.user_id === selectedMemberUserId)
  ) {
    return selectedMemberUserId
  }
  return null
}

export function createRenameUsernameSchema(t: (key: string) => string) {
  return z.object({
    tenant_id: z.coerce.number().int().nonnegative(),
    new_username: z
      .string()
      .trim()
      .min(3, t('Username must be 3-20 characters'))
      .max(20, t('Username must be 3-20 characters'))
      .regex(
        /^[a-z0-9]+(?:_[a-z0-9]+)*$/,
        t(
          'Username can only use lowercase letters, numbers, and single underscores'
        )
      ),
  })
}

export function syncAllocationFormDraft(params: {
  current: {
    tenant_id: number
    department_id: number
    department_budget_id: number
    target_user_id: number
    committed_quota: number
    reason: string
  }
  departmentId: number
  selectedBudgetId: number | null
  selectedMemberUserId: number | null
  resetMode: 'department-change' | 'budget-change' | 'member-change'
}) {
  if (params.resetMode === 'department-change') {
    return {
      tenant_id: params.current.tenant_id,
      department_id: params.departmentId,
      department_budget_id: 0,
      target_user_id: 0,
      committed_quota: 0,
      reason: '',
    }
  }

  return {
    tenant_id: params.current.tenant_id,
    department_id: params.departmentId,
    department_budget_id: params.selectedBudgetId ?? 0,
    target_user_id: params.selectedMemberUserId ?? 0,
    committed_quota: 0,
    reason: '',
  }
}

function parsePositiveInt(value: string) {
  const parsed = Number(value)
  if (!Number.isInteger(parsed) || parsed <= 0) return null
  return parsed
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

export function __testRenderUsernameMutationMessage(
  result: { message?: string } | null | undefined,
  translator: (key: string) => string
) {
  if (!result?.message) return translator('Request failed')
  return translator(result.message)
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
  if (
    fallbackBudgetId != null &&
    availableBudgetIds.includes(fallbackBudgetId)
  ) {
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

function departmentSourceLabel(sourceType: number, t: (key: string) => string) {
  return sourceType === 2 ? t('DingTalk') : t('Manual')
}

function departmentSyncLabel(syncStatus: number, t: (key: string) => string) {
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
    () =>
      findDepartmentNode(departments, resolvedSelection.selectedDepartmentId),
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
  const { t } = useTranslation()
  const [departmentMembers, setDepartmentMembers] = useState<
    DepartmentMemberItem[]
  >([])
  const [selectedMemberUserId, setSelectedMemberUserId] = useState<
    number | null
  >(null)

  useEffect(() => {
    if (!props.currentDepartment) {
      setDepartmentMembers([])
      setSelectedMemberUserId(null)
    }
  }, [props.currentDepartment])

  useEffect(() => {
    setSelectedMemberUserId((current) =>
      resolveDepartmentMemberSelection(departmentMembers, current)
    )
  }, [departmentMembers])

  const selectedMember = useMemo(
    () =>
      departmentMembers.find((item) => item.user_id === selectedMemberUserId) ??
      null,
    [departmentMembers, selectedMemberUserId]
  )
  const selectedMemberMembershipsQuery = useQuery({
    queryKey: userDepartmentsQueryKey(selectedMember?.user_id ?? null),
    queryFn: async () => {
      if (!selectedMember?.user_id) return []
      const result = await getUserDepartments(selectedMember.user_id)
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data?.items ?? []
    },
    enabled: Boolean(selectedMember?.user_id),
  })

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
        selectedMemberUserId={selectedMemberUserId}
        onSelectedMemberChange={setSelectedMemberUserId}
        onMembersChange={setDepartmentMembers}
      />
      <DepartmentOwnersPanel
        departmentId={props.currentDepartment.id}
        departmentName={props.currentDepartment.name}
        members={departmentMembers}
      />
      <DepartmentMemberContextCard
        currentDepartment={props.currentDepartment}
        selectedMember={selectedMember}
        memberships={selectedMemberMembershipsQuery.data ?? []}
        loading={selectedMemberMembershipsQuery.isLoading}
        onMemberRenamed={setSelectedMemberUserId}
      />
      <DepartmentBudgetPanel
        departmentId={props.currentDepartment.id}
        departmentName={props.currentDepartment.name}
        selectedBudgetId={props.selectedBudgetId}
        onSelectedBudgetIdChange={props.onSelectedBudgetIdChange}
        selectedMember={selectedMember}
      />
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
            value={
              department.created_at
                ? formatTimestamp(department.created_at)
                : '-'
            }
          />
          <DepartmentMetaStat
            label={t('Updated At')}
            value={
              department.updated_at
                ? formatTimestamp(department.updated_at)
                : '-'
            }
          />
        </div>
        <div className='rounded-lg border p-3'>
          <div className='text-muted-foreground text-xs'>
            {t('Historical Names')}
          </div>
          <div className='mt-2 text-sm'>
            {department.name_history.length > 0
              ? department.name_history.map((entry) => entry.name).join(', ')
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
  selectedMemberUserId,
  onSelectedMemberChange,
  onMembersChange,
}: {
  departmentId: number
  departmentName: string
  selectedMemberUserId: number | null
  onSelectedMemberChange: (userId: number | null) => void
  onMembersChange: (items: DepartmentMemberItem[]) => void
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

  useEffect(() => {
    onMembersChange(departmentMembersQuery.data?.items ?? [])
  }, [departmentMembersQuery.data?.items, onMembersChange])

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
          selectedMemberUserId={selectedMemberUserId}
          onSelectMember={onSelectedMemberChange}
        />
      </CardContent>
    </Card>
  )
}

function DepartmentOwnersPanel({
  departmentId,
  departmentName,
  members,
}: {
  departmentId: number
  departmentName: string
  members: DepartmentMemberItem[]
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [ownerUserIdText, setOwnerUserIdText] = useState('')
  const tenantId = 0

  const ownersQuery = useQuery({
    queryKey: departmentOwnersQueryKey(departmentId, tenantId),
    queryFn: async () => {
      const result = await getDepartmentOwners(departmentId, tenantId)
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return (
        result.data ?? {
          facts: [],
          effective_owners: [],
          owner_count: 0,
          fallback: 'admin',
        }
      )
    },
  })

  const invalidateOwners = async () => {
    await queryClient.invalidateQueries({
      queryKey: departmentOwnersQueryKey(departmentId, tenantId),
    })
  }

  const grantMutation = useMutation({
    mutationFn: (userId: number) =>
      grantDepartmentOwner(departmentId, {
        tenant_id: tenantId,
        user_id: userId,
      }),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return
      }
      setOwnerUserIdText('')
      await invalidateOwners()
      toast.success(t('Department owner granted'))
    },
  })
  const denyMutation = useMutation({
    mutationFn: (userId: number) =>
      denyDepartmentOwner(departmentId, {
        tenant_id: tenantId,
        user_id: userId,
      }),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return
      }
      setOwnerUserIdText('')
      await invalidateOwners()
      toast.success(t('Local deny override applied'))
    },
  })
  const revokeGrantMutation = useMutation({
    mutationFn: (userId: number) =>
      revokeDepartmentOwnerGrant(departmentId, userId, {
        tenant_id: tenantId,
        user_id: userId,
      }),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return
      }
      await invalidateOwners()
      toast.success(t('Manual grant revoked'))
    },
  })
  const revokeDenyMutation = useMutation({
    mutationFn: (userId: number) =>
      revokeDepartmentOwnerDeny(departmentId, userId, {
        tenant_id: tenantId,
        user_id: userId,
      }),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return
      }
      await invalidateOwners()
      toast.success(t('Local deny override revoked'))
    },
  })

  const ownerUserId = parsePositiveInt(ownerUserIdText)
  const data = ownersQuery.data

  return (
    <Card>
      <CardHeader>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div>
            <CardTitle>{t('Department Owners')}</CardTitle>
            <CardDescription>
              {t(
                'Review effective owners and source facts for {{department}}.',
                { department: departmentName }
              )}
            </CardDescription>
          </div>
          <OwnerResolutionBadge data={data} />
        </div>
      </CardHeader>
      <CardContent className='space-y-4'>
        <div className='flex flex-col gap-2 sm:flex-row'>
          <Input
            inputMode='numeric'
            value={ownerUserIdText}
            onChange={(event) => setOwnerUserIdText(event.target.value)}
            placeholder={t('User ID')}
          />
          <Button
            variant='outline'
            onClick={() => ownerUserId && grantMutation.mutate(ownerUserId)}
            disabled={!ownerUserId || grantMutation.isPending}
          >
            <ShieldCheck data-icon='inline-start' />
            {t('Manual Grant')}
          </Button>
          <Button
            variant='outline'
            onClick={() => ownerUserId && denyMutation.mutate(ownerUserId)}
            disabled={!ownerUserId || denyMutation.isPending}
          >
            <ShieldX data-icon='inline-start' />
            {t('Local Deny')}
          </Button>
        </div>
        <DepartmentOwnersEffectiveList
          owners={data?.effective_owners ?? []}
          members={members}
        />
        <DepartmentOwnerFactsTable
          facts={data?.facts ?? []}
          members={members}
          onRevokeGrant={(userId) => revokeGrantMutation.mutate(userId)}
          onRevokeDeny={(userId) => revokeDenyMutation.mutate(userId)}
        />
      </CardContent>
    </Card>
  )
}

function OwnerResolutionBadge({
  data,
}: {
  data: DepartmentOwnersResponse | undefined
}) {
  const { t } = useTranslation()
  if (!data) {
    return <Badge variant='outline'>{t('Loading')}</Badge>
  }
  if (data.owner_count === 0) {
    return (
      <StatusBadge
        label={`${t('No effective owner')} · ${t('Admin fallback')}`}
        variant='warning'
        copyable={false}
      />
    )
  }
  return (
    <StatusBadge
      label={t('{{count}} effective owner(s)', { count: data.owner_count })}
      variant='success'
      copyable={false}
    />
  )
}

function DepartmentOwnersEffectiveList({
  owners,
  members,
}: {
  owners: EffectiveDepartmentOwnerItem[]
  members: DepartmentMemberItem[]
}) {
  const { t } = useTranslation()

  if (owners.length === 0) {
    return (
      <Empty className='min-h-[180px]'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <ShieldX />
          </EmptyMedia>
          <EmptyTitle>{t('No effective owner')}</EmptyTitle>
          <EmptyDescription>
            {t('Admin fallback will handle owner-required actions.')}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  return (
    <div className='grid gap-2 md:grid-cols-2'>
      {owners.map((owner) => (
        <div
          key={`${owner.user_id}-${owner.role_fact_id}`}
          className='rounded-lg border p-3'
        >
          <div className='font-medium'>
            {formatOwnerUser(owner.user_id, members, t)}
          </div>
          <div className='mt-2 flex flex-wrap gap-2'>
            <Badge variant='secondary'>
              {ownerSourceLabel(owner.source, t)}
            </Badge>
            <Badge variant='outline'>
              {t('Inherited from department #{{id}}', {
                id: owner.inherited_from_department_id,
              })}
            </Badge>
          </div>
        </div>
      ))}
    </div>
  )
}

function DepartmentOwnerFactsTable({
  facts,
  members,
  onRevokeGrant,
  onRevokeDeny,
}: {
  facts: DepartmentOwnerFactItem[]
  members: DepartmentMemberItem[]
  onRevokeGrant: (userId: number) => void
  onRevokeDeny: (userId: number) => void
}) {
  const { t } = useTranslation()

  if (facts.length === 0) {
    return (
      <div className='text-muted-foreground rounded-lg border p-3 text-sm'>
        {t('No owner source facts recorded')}
      </div>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('User')}</TableHead>
          <TableHead>{t('Source Fact')}</TableHead>
          <TableHead>{t('Effect')}</TableHead>
          <TableHead>{t('Status')}</TableHead>
          <TableHead className='text-right'>{t('Actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {facts.map((fact) => (
          <TableRow key={fact.id}>
            <TableCell>{formatOwnerUser(fact.user_id, members, t)}</TableCell>
            <TableCell>{ownerSourceLabel(fact.source, t)}</TableCell>
            <TableCell>{ownerEffectLabel(fact.effect, t)}</TableCell>
            <TableCell>
              <StatusBadge
                label={
                  fact.status === 1
                    ? t('Active')
                    : t('Fact inactive or revoked')
                }
                variant={fact.status === 1 ? 'success' : 'neutral'}
                copyable={false}
              />
            </TableCell>
            <TableCell className='text-right'>
              {fact.source === 'manual_grant' && fact.status === 1 ? (
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => onRevokeGrant(fact.user_id)}
                >
                  {t('Revoke Grant')}
                </Button>
              ) : null}
              {fact.source === 'manual_deny_override' && fact.status === 1 ? (
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => onRevokeDeny(fact.user_id)}
                >
                  {t('Revoke Local Deny')}
                </Button>
              ) : null}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function formatOwnerUser(
  userId: number,
  members: DepartmentMemberItem[],
  t: (key: string, options?: Record<string, unknown>) => string
) {
  const member = members.find((item) => item.user_id === userId)
  if (!member) return t('User #{{id}}', { id: userId })
  return formatEnterpriseUserPrimary({
    displayName: member.display_name,
    username: member.username,
    userId,
  })
}

function ownerSourceLabel(source: string, t: (key: string) => string) {
  if (source === 'manual_deny_override') return t('Local deny override')
  if (source === 'manual_grant') return t('Manual grant')
  if (source === 'dingtalk_synced_owner') return t('DingTalk synced owner')
  return source
}

function ownerEffectLabel(effect: string, t: (key: string) => string) {
  if (effect === 'deny') return t('Deny')
  if (effect === 'allow') return t('Allow')
  return effect
}

function DepartmentMembersTable({
  items,
  departmentId,
  selectedMemberUserId,
  onSelectMember,
}: {
  items: DepartmentMemberItem[]
  departmentId: number
  selectedMemberUserId: number | null
  onSelectMember: (userId: number | null) => void
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
          <TableHead>{t('Current Member')}</TableHead>
          <TableHead className='text-right'>{t('Actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow
            key={item.id}
            className={
              selectedMemberUserId === item.user_id ? 'bg-muted/50' : undefined
            }
          >
            <TableCell>
              <div className='flex min-w-[160px] flex-col gap-1'>
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
            <TableCell>
              <MembershipStatusBadge status={item.status} />
            </TableCell>
            <TableCell>
              <Badge variant='secondary'>
                {item.external_source || 'manual'}
              </Badge>
            </TableCell>
            <TableCell>
              <Button
                variant={
                  selectedMemberUserId === item.user_id
                    ? 'secondary'
                    : 'outline'
                }
                size='sm'
                onClick={() => onSelectMember(item.user_id)}
              >
                {selectedMemberUserId === item.user_id
                  ? t('Current department member')
                  : t('Use in workspace')}
              </Button>
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

export function DepartmentMemberContextCard({
  currentDepartment,
  selectedMember,
  memberships,
  loading,
  onMemberRenamed,
}: {
  currentDepartment: DepartmentTreeNode
  selectedMember: DepartmentMemberItem | null
  memberships: UserDepartmentItem[]
  loading: boolean
  onMemberRenamed?: (userId: number | null) => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const renderUsernameMessage = (
    result: { message?: string } | null | undefined
  ) => __testRenderUsernameMutationMessage(result, t)
  const renameSchema = createRenameUsernameSchema(t)
  type RenameFormValues = z.infer<typeof renameSchema>
  const form = useForm<RenameFormValues>({
    resolver: zodResolver(
      renameSchema
    ) as unknown as Resolver<RenameFormValues>,
    defaultValues: {
      tenant_id: currentDepartment.tenant_id ?? 0,
      new_username: selectedMember?.username ?? '',
    },
  })

  useEffect(() => {
    form.reset({
      tenant_id: currentDepartment.tenant_id ?? 0,
      new_username: selectedMember?.username ?? '',
    })
  }, [currentDepartment.tenant_id, form, selectedMember?.username])

  const renameMutation = useMutation({
    mutationFn: async (values: RenameFormValues) => {
      if (!selectedMember) throw new Error(t('Select a department member'))
      return renameDepartmentMember(
        currentDepartment.id,
        selectedMember.user_id,
        {
          tenant_id: values.tenant_id || undefined,
          new_username: values.new_username.trim(),
        }
      )
    },
    onSuccess: async (result) => {
      if (!selectedMember) return
      if (!result.success) {
        toast.error(renderUsernameMessage(result))
        return
      }
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: departmentMembersQueryKey(currentDepartment.id),
        }),
        queryClient.invalidateQueries({
          queryKey: userDepartmentsQueryKey(selectedMember.user_id),
        }),
        queryClient.invalidateQueries({
          queryKey: departmentBudgetQueryKey(
            currentDepartment.id,
            currentDepartment.tenant_id ?? 0
          ),
        }),
        queryClient.invalidateQueries({
          queryKey: departmentBudgetListQueryScopeKey(
            currentDepartment.id,
            currentDepartment.tenant_id ?? 0
          ),
        }),
        queryClient.invalidateQueries({
          queryKey: departmentBudgetDetailQueryScopeKey(
            currentDepartment.id,
            currentDepartment.tenant_id ?? 0
          ),
        }),
        queryClient.invalidateQueries({
          queryKey: quotaAllocationQueryScopeKey(
            currentDepartment.id,
            currentDepartment.tenant_id ?? 0
          ),
        }),
      ])
      onMemberRenamed?.(selectedMember.user_id)
      toast.success(t('Username updated'))
    },
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Current Member Governance')}</CardTitle>
        <CardDescription>
          {t(
            'Use the current department member context to inspect memberships and drive wallet allocation without typing a user identifier.'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        {!selectedMember ? (
          <Empty className='min-h-[220px] border'>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <Users className='size-4' />
              </EmptyMedia>
              <EmptyTitle>{t('Select a department member')}</EmptyTitle>
              <EmptyDescription>
                {t(
                  'Choose a member from the current department list to inspect memberships and create wallet allocations without typing a user identifier.'
                )}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <>
            <div className='grid gap-3 md:grid-cols-3'>
              <DepartmentMetaStat
                label={t('Current Member')}
                value={formatEnterpriseUserPrimary({
                  displayName: selectedMember.display_name,
                  username: selectedMember.username,
                  userId: selectedMember.user_id,
                })}
              />
              <DepartmentMetaStat
                label={t('Current department member')}
                value={t('Selected from {{department}}', {
                  department: currentDepartment.name,
                })}
              />
              <DepartmentMetaStat
                label={t('Membership Status')}
                value={statusLabel(selectedMember.status, t)}
              />
            </div>
            <div className='rounded-lg border p-4'>
              <div className='mb-3'>
                <div className='text-sm font-medium'>
                  {t('Rename Username')}
                </div>
                <div className='text-muted-foreground text-xs'>
                  {t(
                    'Keep history logs and risk events on their original username snapshots. Current governance views switch to the updated username after refresh.'
                  )}
                </div>
              </div>
              <Form {...form}>
                <form
                  className='grid gap-3 md:grid-cols-[minmax(0,1fr)_auto]'
                  onSubmit={form.handleSubmit((values) =>
                    renameMutation.mutate(values)
                  )}
                >
                  <FormField
                    control={form.control}
                    name='new_username'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Readable Username')}</FormLabel>
                        <FormControl>
                          <Input
                            {...field}
                            placeholder={t(
                              'Use lowercase letters, numbers, or underscores'
                            )}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <div className='flex items-end'>
                    <Button
                      type='submit'
                      disabled={!selectedMember || renameMutation.isPending}
                    >
                      {t('Update Username')}
                    </Button>
                  </div>
                </form>
              </Form>
            </div>
            {loading ? (
              <Skeleton className='h-32 w-full' />
            ) : (
              <UserDepartmentsTable items={memberships} hasResult />
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}

function DepartmentBudgetPanel({
  departmentId,
  departmentName,
  selectedBudgetId,
  onSelectedBudgetIdChange,
  selectedMember,
}: {
  departmentId: number
  departmentName: string
  selectedBudgetId: number | null
  onSelectedBudgetIdChange: (budgetId: number | null) => void
  selectedMember: DepartmentMemberItem | null
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
  const previousDepartmentIdRef = useRef(departmentId)
  const previousBudgetIdRef = useRef<number | null>(selectedBudgetId)
  const previousSelectedMemberIdRef = useRef<number | null>(
    selectedMember?.user_id ?? null
  )

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
  }, [allocationForm, departmentId, form])

  useEffect(() => {
    if (allocationForm.getValues('tenant_id') !== tenantId) {
      allocationForm.setValue('tenant_id', tenantId)
    }
  }, [allocationForm, tenantId])

  useEffect(() => {
    if (previousDepartmentIdRef.current === departmentId) return
    allocationForm.reset(
      syncAllocationFormDraft({
        current: allocationForm.getValues(),
        departmentId,
        selectedBudgetId: effectiveBudgetId,
        selectedMemberUserId: selectedMember?.user_id ?? null,
        resetMode: 'department-change',
      })
    )
    previousDepartmentIdRef.current = departmentId
    previousBudgetIdRef.current = effectiveBudgetId
    previousSelectedMemberIdRef.current = selectedMember?.user_id ?? null
  }, [allocationForm, departmentId, effectiveBudgetId, selectedMember])

  useEffect(() => {
    const normalizedBudgetId = effectiveBudgetId ?? null
    if (selectedBudgetId !== normalizedBudgetId) {
      onSelectedBudgetIdChange(normalizedBudgetId)
    }
  }, [effectiveBudgetId, onSelectedBudgetIdChange, selectedBudgetId])

  useEffect(() => {
    if (previousBudgetIdRef.current === effectiveBudgetId) return
    allocationForm.reset(
      syncAllocationFormDraft({
        current: allocationForm.getValues(),
        departmentId,
        selectedBudgetId: effectiveBudgetId,
        selectedMemberUserId: selectedMember?.user_id ?? null,
        resetMode: 'budget-change',
      })
    )
    previousBudgetIdRef.current = effectiveBudgetId
  }, [allocationForm, departmentId, effectiveBudgetId, selectedMember])

  useEffect(() => {
    const nextSelectedMemberId = selectedMember?.user_id ?? null
    if (previousSelectedMemberIdRef.current === nextSelectedMemberId) return
    allocationForm.reset(
      syncAllocationFormDraft({
        current: allocationForm.getValues(),
        departmentId,
        selectedBudgetId: effectiveBudgetId,
        selectedMemberUserId: nextSelectedMemberId,
        resetMode: 'member-change',
      })
    )
    previousSelectedMemberIdRef.current = nextSelectedMemberId
  }, [allocationForm, departmentId, effectiveBudgetId, selectedMember])

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
                              <SelectItem value='daily'>
                                {t('Daily')}
                              </SelectItem>
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
          <CardTitle>{t('Current Member Wallet Allocation')}</CardTitle>
          <CardDescription>
            {t(
              'Keep budget detail, derived wallets, and current member allocation in one current-department workflow.'
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
              <div className='rounded-lg border p-3 md:col-span-2'>
                <div className='text-muted-foreground text-xs'>
                  {t('Current Member')}
                </div>
                <div className='mt-1 text-sm font-medium'>
                  {selectedMember
                    ? formatEnterpriseUserPrimary({
                        displayName: selectedMember.display_name,
                        username: selectedMember.username,
                        userId: selectedMember.user_id,
                      })
                    : t('No member selected')}
                </div>
                <div className='text-muted-foreground mt-1 text-xs'>
                  {selectedMember
                    ? t('Selected from {{department}}', {
                        department: departmentName,
                      })
                    : t(
                        'Select a member from the current department list before creating a wallet allocation.'
                      )}
                </div>
              </div>
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
              {!effectiveBudgetId ? (
                <div className='md:col-span-2'>
                  <Empty className='min-h-[140px] border'>
                    <EmptyHeader>
                      <EmptyMedia variant='icon'>
                        <CreditCard className='size-4' />
                      </EmptyMedia>
                      <EmptyTitle>{t('Select a budget pool')}</EmptyTitle>
                      <EmptyDescription>
                        {t(
                          'Choose a budget pool in the current department before creating a member wallet allocation.'
                        )}
                      </EmptyDescription>
                    </EmptyHeader>
                  </Empty>
                </div>
              ) : null}
              <div className='flex justify-end md:col-span-2'>
                <Button
                  type='submit'
                  disabled={
                    !effectiveBudgetId ||
                    !selectedMember ||
                    allocationMutation.isPending
                  }
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
            <TableCell>
              <div className='flex min-w-[180px] flex-col gap-1'>
                <span className='font-medium'>
                  {formatEnterpriseUserPrimary({
                    displayName: item.target_display_name,
                    username: item.target_username,
                    userId: item.target_user_id,
                  })}
                </span>
                <span className='text-muted-foreground text-xs'>
                  {formatEnterpriseUserSecondary(
                    {
                      displayName: item.target_display_name,
                      username: item.target_username,
                      userId: item.target_user_id,
                    },
                    t
                  )}
                </span>
              </div>
            </TableCell>
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
                  {formatEnterpriseUserPrimary({
                    displayName: wallet.target_display_name,
                    username: wallet.target_username,
                    userId: wallet.target_user_id,
                  })}
                </span>
                <span className='text-muted-foreground text-xs'>
                  {formatEnterpriseUserSecondary(
                    {
                      displayName: wallet.target_display_name,
                      username: wallet.target_username,
                      userId: wallet.target_user_id,
                    },
                    t
                  )}
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
