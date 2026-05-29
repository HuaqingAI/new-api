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
import { Link } from '@tanstack/react-router'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import {
  addDepartmentMember,
  createQuotaAllocation,
  deactivateDepartmentMember,
  createDepartmentBudget,
  departmentBudgetDetailQueryKey,
  departmentBudgetListQueryKey,
  departmentBudgetQueryKey,
  enterpriseOrganizationQueryKey,
  getDepartmentBudgetDetail,
  getDepartmentBudgets,
  getDepartmentBudget,
  getDepartmentMembers,
  getQuotaAllocations,
  getUserDepartments,
  quotaAllocationQueryKey,
  revokeQuotaAllocation,
  replaceUserDepartments,
  restoreDepartmentMember,
} from './api'
import { DepartmentTree } from './components/DepartmentTree'
import { useDepartmentTree } from './hooks/use-department-tree'
import type {
  ApiResponse,
  DepartmentBudgetDetailResponse,
  DepartmentBudgetItem,
  DepartmentBudgetSortField,
  DepartmentBudgetWalletDetail,
  DepartmentMemberItem,
  EnterpriseBudgetErrorData,
  MembershipStatus,
  QuotaAllocationItem,
  UserDepartmentItem,
} from './types'

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

export function EnterpriseOrganization() {
  const { t } = useTranslation()
  const {
    data: departments = [],
    isLoading,
    isFetching,
    refetch,
  } = useDepartmentTree()

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
        <Tabs defaultValue='tree' className='flex flex-col gap-4'>
          <TabsList>
            <TabsTrigger value='tree'>{t('Department Tree')}</TabsTrigger>
            <TabsTrigger value='users'>{t('User Departments')}</TabsTrigger>
            <TabsTrigger value='members'>{t('Department Members')}</TabsTrigger>
            <TabsTrigger value='budgets'>{t('Department Budget')}</TabsTrigger>
          </TabsList>
          <TabsContent value='tree' className='m-0'>
            <EnterpriseOrganizationContent
              isLoading={isLoading}
              departments={departments}
            />
          </TabsContent>
          <TabsContent value='users' className='m-0'>
            <UserDepartmentsPanel />
          </TabsContent>
          <TabsContent value='members' className='m-0'>
            <DepartmentMembersPanel />
          </TabsContent>
          <TabsContent value='budgets' className='m-0'>
            <DepartmentBudgetPanel />
          </TabsContent>
        </Tabs>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

export function EnterpriseOrganizationContent(props: {
  isLoading: boolean
  departments: ReturnType<typeof useDepartmentTree>['data']
}) {
  if (props.isLoading) {
    return <DepartmentTreeSkeleton />
  }
  if (!props.departments || props.departments.length === 0) {
    return <EnterpriseOrganizationEmptyState />
  }
  return <DepartmentTree nodes={props.departments} />
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

  const invalidateOrganization = () =>
    queryClient.invalidateQueries({ queryKey: enterpriseOrganizationQueryKey })

  const userDepartmentsQuery = useQuery({
    queryKey: [
      ...enterpriseOrganizationQueryKey,
      'users',
      userId,
      'departments',
    ],
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
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return
      }
      invalidateOrganization()
    },
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('User Departments')}</CardTitle>
        <CardDescription>
          {t('Shows every department membership for a user.')}
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

function DepartmentMembersPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [departmentIdText, setDepartmentIdText] = useState('')
  const [memberUserIdText, setMemberUserIdText] = useState('')
  const departmentId = useMemo(
    () => parsePositiveInt(departmentIdText),
    [departmentIdText]
  )

  const invalidateOrganization = () =>
    queryClient.invalidateQueries({ queryKey: enterpriseOrganizationQueryKey })

  const departmentMembersQuery = useQuery({
    queryKey: [
      ...enterpriseOrganizationQueryKey,
      'departments',
      departmentId,
      'members',
    ],
    queryFn: async () => {
      if (!departmentId) return null
      const result = await getDepartmentMembers(departmentId)
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data ?? { items: [], total: 0 }
    },
    enabled: Boolean(departmentId),
  })

  const addMemberMutation = useMutation({
    mutationFn: () => {
      const memberUserId = parsePositiveInt(memberUserIdText)
      if (!departmentId || !memberUserId) throw new Error('missing ids')
      return addDepartmentMember(departmentId, { user_id: memberUserId })
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return
      }
      setMemberUserIdText('')
      invalidateOrganization()
    },
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Department Members')}</CardTitle>
        <CardDescription>
          {t('Shows all users attached to the selected department.')}
        </CardDescription>
      </CardHeader>
      <CardContent className='flex flex-col gap-4'>
        <div className='flex flex-col gap-2 sm:flex-row'>
          <Input
            inputMode='numeric'
            value={departmentIdText}
            onChange={(event) => setDepartmentIdText(event.target.value)}
            placeholder={t('Department ID')}
          />
          <Input
            inputMode='numeric'
            value={memberUserIdText}
            onChange={(event) => setMemberUserIdText(event.target.value)}
            placeholder={t('User ID')}
          />
          <Button
            onClick={() => addMemberMutation.mutate()}
            disabled={
              !departmentId ||
              !parsePositiveInt(memberUserIdText) ||
              addMemberMutation.isPending
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
  departmentId: number | null
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const invalidateOrganization = () =>
    queryClient.invalidateQueries({ queryKey: enterpriseOrganizationQueryKey })

  const deactivateMutation = useMutation({
    mutationFn: (userId: number) => {
      if (!departmentId) throw new Error('missing department id')
      return deactivateDepartmentMember(departmentId, userId)
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return
      }
      invalidateOrganization()
    },
  })
  const restoreMutation = useMutation({
    mutationFn: (userId: number) => {
      if (!departmentId) throw new Error('missing department id')
      return restoreDepartmentMember(departmentId, userId)
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return
      }
      invalidateOrganization()
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

function DepartmentBudgetPanel() {
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
      department_id: 1,
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
      department_id: 1,
      department_budget_id: 0,
      target_user_id: 0,
      committed_quota: 0,
      reason: '',
    },
  })
  const tenantId = form.watch('tenant_id')
  const departmentId = form.watch('department_id')
  const budgetType = form.watch('type')
  const cycleType = form.watch('cycle_type')
  const [sortBy, setSortBy] = useState<DepartmentBudgetSortField>('usage_ratio')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')
  const [selectedBudgetId, setSelectedBudgetId] = useState<number | null>(null)

  const budgetQuery = useQuery({
    queryKey: [...departmentBudgetQueryKey, tenantId, departmentId],
    queryFn: async () => {
      if (!departmentId) return null
      const result = await getDepartmentBudget(departmentId, tenantId)
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data?.item ?? null
    },
    enabled: Boolean(departmentId),
  })

  const currentBudgetId = budgetQuery.data?.id ?? 0

  const budgetListQuery = useQuery({
    queryKey: [
      ...departmentBudgetListQueryKey,
      tenantId,
      departmentId,
      sortBy,
      sortOrder,
    ],
    queryFn: async () => {
      if (!departmentId) {
        return {
          items: [],
          thresholds: { warning: 80, critical: 95 },
        }
      }
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
    enabled: Boolean(departmentId),
  })

  const effectiveBudgetId =
    selectedBudgetId ??
    budgetListQuery.data?.items[0]?.id ??
    currentBudgetId ??
    null

  const budgetDetailQuery = useQuery({
    queryKey: [
      ...departmentBudgetDetailQueryKey,
      tenantId,
      departmentId,
      effectiveBudgetId,
    ],
    queryFn: async (): Promise<DepartmentBudgetDetailResponse> => {
      if (!departmentId || !effectiveBudgetId) {
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
    enabled: Boolean(departmentId && effectiveBudgetId),
  })

  const allocationListQuery = useQuery({
    queryKey: [
      ...quotaAllocationQueryKey,
      tenantId,
      departmentId,
      effectiveBudgetId,
    ],
    queryFn: async () => {
      if (!effectiveBudgetId || !departmentId) return []
      const result = await getQuotaAllocations(
        effectiveBudgetId,
        tenantId,
        departmentId
      )
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data?.items ?? []
    },
    enabled: Boolean(effectiveBudgetId && departmentId),
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
      return createDepartmentBudget(values.department_id, payload)
    },
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(renderApiMessage(result))
        return
      }
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetQueryKey,
      })
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetListQueryKey,
      })
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetDetailQueryKey,
      })
      await queryClient.invalidateQueries({
        queryKey: enterpriseOrganizationQueryKey,
      })
      const budgetId = result.data?.item?.id ?? 0
      setSelectedBudgetId(budgetId || null)
      allocationForm.setValue('department_budget_id', budgetId)
      toast.success(t('Department budget saved'))
    },
  })

  const allocationMutation = useMutation({
    mutationFn: async (values: AllocationFormValues) =>
      createQuotaAllocation({
        tenant_id: Number(values.tenant_id),
        department_id: Number(values.department_id),
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
        queryKey: departmentBudgetQueryKey,
      })
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetListQueryKey,
      })
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetDetailQueryKey,
      })
      await queryClient.invalidateQueries({ queryKey: quotaAllocationQueryKey })
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
        queryKey: departmentBudgetQueryKey,
      })
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetListQueryKey,
      })
      await queryClient.invalidateQueries({
        queryKey: departmentBudgetDetailQueryKey,
      })
      await queryClient.invalidateQueries({ queryKey: quotaAllocationQueryKey })
      toast.success(t('Wallet allocation revoked'))
    },
  })

  useEffect(() => {
    if (allocationForm.getValues('tenant_id') !== tenantId) {
      allocationForm.setValue('tenant_id', tenantId)
    }
  }, [allocationForm, tenantId])

  useEffect(() => {
    if (allocationForm.getValues('department_id') !== departmentId) {
      allocationForm.setValue('department_id', departmentId)
    }
  }, [allocationForm, departmentId])

  useEffect(() => {
    const nextBudgetId = effectiveBudgetId ?? 0
    if (allocationForm.getValues('department_budget_id') !== nextBudgetId) {
      allocationForm.setValue('department_budget_id', nextBudgetId)
    }
  }, [allocationForm, effectiveBudgetId])

  useEffect(() => {
    if (
      selectedBudgetId !== null &&
      budgetListQuery.data?.items?.some(
        (item) => item.id === selectedBudgetId
      ) === false
    ) {
      setSelectedBudgetId(null)
    }
  }, [budgetListQuery.data?.items, selectedBudgetId])

  return (
    <div className='grid gap-4 lg:grid-cols-[minmax(0,360px)_minmax(0,1fr)]'>
      <Card>
        <CardHeader>
          <CardTitle>{t('Department Budget')}</CardTitle>
          <CardDescription>
            {t(
              'Create a balance or subscription budget pool for the selected department.'
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
                name='department_id'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Department ID')}</FormLabel>
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
      <div className='space-y-4'>
        <DepartmentBudgetOverviewCard
          item={budgetQuery.data ?? null}
          selectedBudget={budgetDetailQuery.data?.budget ?? null}
          thresholds={
            budgetDetailQuery.data?.thresholds ??
            budgetListQuery.data?.thresholds ?? { warning: 80, critical: 95 }
          }
        />
        <DepartmentBudgetListCard
          items={budgetListQuery.data?.items ?? []}
          loading={budgetListQuery.isLoading}
          selectedBudgetId={effectiveBudgetId}
          sortBy={sortBy}
          sortOrder={sortOrder}
          onSelectBudget={(budgetId) => setSelectedBudgetId(budgetId)}
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
                        <Input
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
                    disabled={
                      !effectiveBudgetId || allocationMutation.isPending
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
  onSelectBudget: (budgetId: number) => void
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
