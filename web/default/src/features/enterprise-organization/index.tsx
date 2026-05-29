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
import { useMemo, useState } from 'react'
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
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm, type Resolver } from 'react-hook-form'
import { z } from 'zod'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
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
import { formatTimestamp } from '@/lib/format'
import {
  addDepartmentMember,
  createQuotaAllocation,
  deactivateDepartmentMember,
  createDepartmentBudget,
  departmentBudgetQueryKey,
  enterpriseOrganizationQueryKey,
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
  DepartmentBudgetItem,
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
            message: t('Subscription budget cycle quota must be greater than 0'),
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
    committed_quota: z.coerce.number().int().positive({
      message: t('Allocation quota must be greater than 0'),
    }),
    reason: z.string().trim().max(500).default(''),
  })
}

export function __testRenderApiMessage(
  result: ApiResponse<EnterpriseBudgetErrorData | unknown> | null | undefined,
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
  const { data: departments = [], isLoading, isFetching, refetch } =
    useDepartmentTree()

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
    queryKey: [...enterpriseOrganizationQueryKey, 'users', userId, 'departments'],
    queryFn: async () => {
      if (!userId) return null
      const result = await getUserDepartments(userId)
      if (!result.success) throw new Error(result.message || t('Request failed'))
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
      if (!result.success) throw new Error(result.message || t('Request failed'))
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
  const renderApiMessage = (
    result: ApiResponse<EnterpriseBudgetErrorData> | null | undefined
  ) => __testRenderApiMessage(result, t)
  const budgetSchema = createBudgetSchema(t)
  const allocationSchema = createAllocationSchema(t)
  type BudgetFormValues = z.infer<typeof budgetSchema>
  type AllocationFormValues = z.infer<typeof allocationSchema>

  const form = useForm<BudgetFormValues>({
    resolver: zodResolver(budgetSchema) as unknown as Resolver<BudgetFormValues>,
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
    resolver: zodResolver(allocationSchema) as unknown as Resolver<AllocationFormValues>,
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

  const budgetQuery = useQuery({
    queryKey: [...departmentBudgetQueryKey, tenantId, departmentId],
    queryFn: async () => {
      if (!departmentId) return null
      const result = await getDepartmentBudget(departmentId, tenantId)
      if (!result.success) throw new Error(result.message || t('Request failed'))
      return result.data?.item ?? null
    },
    enabled: Boolean(departmentId),
  })

  const currentBudgetId = budgetQuery.data?.id ?? 0

  const allocationListQuery = useQuery({
    queryKey: [
      ...quotaAllocationQueryKey,
      tenantId,
      departmentId,
      currentBudgetId,
    ],
    queryFn: async () => {
      if (!currentBudgetId || !departmentId) return []
      const result = await getQuotaAllocations(currentBudgetId, tenantId, departmentId)
      if (!result.success) throw new Error(result.message || t('Request failed'))
      return result.data?.items ?? []
    },
    enabled: Boolean(currentBudgetId && departmentId),
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
      await queryClient.invalidateQueries({ queryKey: departmentBudgetQueryKey })
      await queryClient.invalidateQueries({
        queryKey: enterpriseOrganizationQueryKey,
      })
      const budgetId = result.data?.item?.id ?? 0
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
      await queryClient.invalidateQueries({ queryKey: departmentBudgetQueryKey })
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
      await queryClient.invalidateQueries({ queryKey: departmentBudgetQueryKey })
      await queryClient.invalidateQueries({ queryKey: quotaAllocationQueryKey })
      toast.success(t('Wallet allocation revoked'))
    },
  })

  if (allocationForm.getValues('tenant_id') !== tenantId) {
    allocationForm.setValue('tenant_id', tenantId)
  }
  if (allocationForm.getValues('department_id') !== departmentId) {
    allocationForm.setValue('department_id', departmentId)
  }
  if (allocationForm.getValues('department_budget_id') !== currentBudgetId) {
    allocationForm.setValue('department_budget_id', currentBudgetId)
  }

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
              onSubmit={form.handleSubmit((values) => createMutation.mutate(values))}
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
                        <SelectItem value='balance'>{t('Balance Budget')}</SelectItem>
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
                        <Select value={field.value} onValueChange={field.onChange}>
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem value='daily'>{t('Daily')}</SelectItem>
                            <SelectItem value='weekly'>{t('Weekly')}</SelectItem>
                            <SelectItem value='monthly'>{t('Monthly')}</SelectItem>
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
        <DepartmentBudgetStatusCard item={budgetQuery.data ?? null} />
        <Card>
          <CardHeader>
            <CardTitle>{t('Allocate Wallet')}</CardTitle>
            <CardDescription>
              {t(
                'Allocate department budget into a member wallet without leaving the budget tab.'
              )}
            </CardDescription>
          </CardHeader>
          <CardContent className='space-y-4'>
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
                <div className='md:col-span-2 flex justify-end'>
                  <Button
                    type='submit'
                    disabled={!currentBudgetId || allocationMutation.isPending}
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

export function DepartmentBudgetStatusCard({
  item,
}: {
  item: DepartmentBudgetItem | null
}) {
  const { t } = useTranslation()

  if (!item) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t('Current Budget Pool')}</CardTitle>
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
                {t('Create a budget pool to view the department budget state here.')}
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
        <CardTitle>{t('Current Budget Pool')}</CardTitle>
        <CardDescription>
          {t('Shows the latest budget pool saved for this department.')}
        </CardDescription>
      </CardHeader>
      <CardContent className='grid gap-3 sm:grid-cols-2'>
        <BudgetStat
          label={t('Budget Type')}
          value={
            item.type === 'balance'
              ? t('Balance Budget')
              : t('Subscription Budget')
          }
        />
        <BudgetStat
          label={t('Budget Status')}
          value={enterpriseBudgetStatusLabel(item.status, t)}
        />
        <BudgetStat label={t('Remaining Quota')} value={String(item.remaining)} />
        <BudgetStat label={t('Total Quota')} value={String(item.total_quota)} />
        <BudgetStat label={t('Cycle Quota')} value={String(item.cycle_quota)} />
        <BudgetStat
          label={t('Cycle Type')}
          value={formatBudgetCycleType(item.cycle_type, t)}
        />
        <BudgetStat
          label={t('Cycle Start Time')}
          value={item.cycle_started_at ? formatTimestamp(item.cycle_started_at) : '-'}
        />
        <BudgetStat
          label={t('Custom Cycle Seconds')}
          value={item.custom_seconds ? String(item.custom_seconds) : '-'}
        />
        <BudgetStat
          label={t('Expires At (optional)')}
          value={item.expires_at ? formatTimestamp(item.expires_at) : '-'}
        />
      </CardContent>
    </Card>
  )
}

function BudgetStat({ label, value }: { label: string; value: string }) {
  return (
    <div className='rounded-lg border p-3'>
      <div className='text-muted-foreground text-xs'>{label}</div>
      <div className='mt-1 flex items-center gap-2 font-medium'>
        <CalendarClock className='size-4 text-muted-foreground' />
        <span>{value}</span>
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
