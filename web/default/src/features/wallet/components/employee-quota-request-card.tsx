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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { useForm, type Resolver } from 'react-hook-form'
import { Coins } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
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
import { Textarea } from '@/components/ui/textarea'
import {
  getQuotaRequestCapability,
  getUserDepartments,
  quotaRequestQueryScopeKey,
  submitQuotaRequest,
  userDepartmentsQueryKey,
  governanceNotificationQueryKey,
  governanceTimelineQueryKey,
} from '@/features/enterprise-organization/api'
import type {
  UserDepartmentItem,
} from '@/features/enterprise-organization/types'
import {
  QuotaRequestBudgetOption,
  QuotaRequestBudgetSummary,
} from '@/features/enterprise-organization/quota-request-budget-display-components'
import type { UserWalletData } from '../types'

const activeMembershipStatus = 1

function createWalletQuotaRequestSchema(t: (key: string) => string) {
  return z.object({
    tenant_id: z.coerce.number().int().nonnegative(),
    department_id: z.coerce.number().int().positive({
      message: t('Choose a target department'),
    }),
    department_budget_id: z.coerce.number().int().positive({
      message: t('Choose a target budget pool'),
    }),
    budget_mode: z.literal('department_budget'),
    requested_quota: z.coerce.number().int().positive({
      message: t('Requested quota must be greater than 0'),
    }),
    request_reason: z.string().trim().max(500).default(''),
  })
}

export function getEmployeeQuotaRequestDepartmentOptions(
  items: UserDepartmentItem[]
) {
  return items.filter((item) => item.status === activeMembershipStatus)
}

export function resolveEmployeeQuotaRequestBudgetId(
  availableBudgetIds: number[],
  selectedBudgetId: number | null | undefined
) {
  if (
    selectedBudgetId != null &&
    availableBudgetIds.includes(selectedBudgetId)
  ) {
    return selectedBudgetId
  }
  return null
}

export function EmployeeQuotaRequestCard({
  user,
}: {
  user: UserWalletData | null
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const schema = createWalletQuotaRequestSchema(t)
  type QuotaRequestFormValues = z.infer<typeof schema>
  const [selectedDepartmentId, setSelectedDepartmentId] = useState<number | null>(
    null
  )
  const form = useForm<QuotaRequestFormValues>({
    resolver: zodResolver(schema) as unknown as Resolver<QuotaRequestFormValues>,
    defaultValues: {
      tenant_id: 0,
      department_id: 0,
      department_budget_id: 0,
      budget_mode: 'department_budget',
      requested_quota: 0,
      request_reason: '',
    },
  })
  const tenantId = form.watch('tenant_id')
  const selectedBudgetId = form.watch('department_budget_id')
  const normalizedTenantId = tenantId || 0

  const departmentsQuery = useQuery({
    queryKey: userDepartmentsQueryKey(user?.id ?? null),
    queryFn: async () => {
      if (!user?.id) return []
      const result = await getUserDepartments(user.id)
      if (!result.success) throw new Error(result.message || t('Request failed'))
      return getEmployeeQuotaRequestDepartmentOptions(result.data?.items ?? [])
    },
    enabled: Boolean(user?.id),
  })
  const departments = departmentsQuery.data ?? []
  const currentDepartment = useMemo(
    () => departments.find((item) => item.department_id === selectedDepartmentId) ?? null,
    [departments, selectedDepartmentId]
  )

  useEffect(() => {
    if (!selectedDepartmentId && departments.length > 0) {
      const first = departments[0]
      setSelectedDepartmentId(first.department_id)
      form.setValue('department_id', first.department_id)
      form.setValue('tenant_id', first.tenant_id ?? 0)
    }
  }, [departments, form, selectedDepartmentId])

  const capabilityQuery = useQuery({
    queryKey: [
      ...quotaRequestQueryScopeKey(
        selectedDepartmentId ?? 0,
        normalizedTenantId
      ),
      'capability',
      'wallet-entry',
    ],
    queryFn: async () => {
      if (!selectedDepartmentId) {
        return { can_submit: false, can_govern: false, budgets: [] }
      }
      const result = await getQuotaRequestCapability(
        selectedDepartmentId,
        tenantId || undefined
      )
      if (!result.success) throw new Error(result.message || t('Request failed'))
      return result.data ?? { can_submit: false, can_govern: false, budgets: [] }
    },
    enabled: Boolean(selectedDepartmentId),
  })
  const budgets = capabilityQuery.data?.budgets ?? []
  const selectedBudget = useMemo(
    () => budgets.find((item) => item.id === selectedBudgetId) ?? null,
    [budgets, selectedBudgetId]
  )

  useEffect(() => {
    const resolvedBudgetId = resolveEmployeeQuotaRequestBudgetId(
      budgets.map((item) => item.id),
      selectedBudgetId
    )
    if (selectedBudgetId && !resolvedBudgetId) {
      form.setValue('department_budget_id', 0)
    }
  }, [budgets, form, selectedBudgetId])

  const submitMutation = useMutation({
    mutationFn: async (values: QuotaRequestFormValues) =>
      submitQuotaRequest({
        tenant_id: Number(values.tenant_id) || undefined,
        department_id: Number(values.department_id),
        department_budget_id: Number(values.department_budget_id),
        budget_mode: values.budget_mode,
        requested_quota: Number(values.requested_quota),
        request_reason: values.request_reason.trim() || undefined,
      }),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return
      }
      if (selectedDepartmentId) {
        await queryClient.invalidateQueries({
          queryKey: quotaRequestQueryScopeKey(
            selectedDepartmentId,
            normalizedTenantId
          ),
        })
        await invalidateQuotaRequestGovernanceQueries(
          queryClient,
          selectedDepartmentId,
          normalizedTenantId
        )
      }
      form.setValue('requested_quota', 0)
      form.setValue('request_reason', '')
      toast.success(t('Quota request submitted'))
    },
  })

  const handleDepartmentChange = (value: string | null) => {
    if (!value) return
    const departmentId = Number(value)
    const department = departments.find((item) => item.department_id === departmentId)
    setSelectedDepartmentId(departmentId)
    form.setValue('department_id', departmentId)
    form.setValue('tenant_id', department?.tenant_id ?? 0)
    form.setValue('department_budget_id', 0)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Need more quota?')}</CardTitle>
        <CardDescription>
          {t(
            'Apply for quota from your wallet without opening the enterprise governance workspace.'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        <div className='rounded-lg border p-3'>
          <div className='text-sm font-medium'>
            {t('Apply for quota from your wallet')}
          </div>
          <div className='text-muted-foreground mt-1 text-xs'>
            {t(
              'Choose an active department and one of its available budget pools before submitting.'
            )}
          </div>
        </div>
        {!user ? (
          <Empty className='min-h-[140px] border'>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <Coins className='size-4' />
              </EmptyMedia>
              <EmptyTitle>{t('No current user')}</EmptyTitle>
              <EmptyDescription>
                {t('Sign in before submitting a quota request.')}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : null}
        <Form {...form}>
          <form
            className='grid gap-4 md:grid-cols-2'
            onSubmit={form.handleSubmit((values) => submitMutation.mutate(values))}
          >
            <FormField
              control={form.control}
              name='department_id'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Target Department')}</FormLabel>
                  <Select
                    value={field.value ? String(field.value) : undefined}
                    onValueChange={handleDepartmentChange}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder={t('Choose a target department')} />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {departments.map((item) => (
                        <SelectItem
                          key={item.id}
                          value={String(item.department_id)}
                        >
                          {item.department_name || `#${item.department_id}`}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='department_budget_id'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Target Budget Pool')}</FormLabel>
                  <Select
                    value={field.value ? String(field.value) : undefined}
                    onValueChange={(value) => field.onChange(Number(value))}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder={t('Choose a target budget pool')} />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {budgets.map((item) => (
                        <SelectItem
                          key={item.id}
                          value={String(item.id)}
                          className='items-start py-2'
                        >
                          <QuotaRequestBudgetOption item={item} />
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='requested_quota'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Requested Quota')}</FormLabel>
                  <FormControl>
                    <Input inputMode='numeric' {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className='rounded-lg border p-3'>
              <div className='text-muted-foreground text-xs'>
                {t('Selected request scope')}
              </div>
              {selectedBudget ? (
                <QuotaRequestBudgetSummary item={selectedBudget} />
              ) : (
                <>
                  <div className='mt-1 text-sm font-medium'>
                    {currentDepartment?.department_name ??
                      t('No department selected')}
                  </div>
                  <div className='text-muted-foreground mt-1 text-xs'>
                    {t('No budget pool selected')}
                  </div>
                </>
              )}
            </div>
            <FormField
              control={form.control}
              name='request_reason'
              render={({ field }) => (
                <FormItem className='md:col-span-2'>
                  <FormLabel>{t('Request Reason')}</FormLabel>
                  <FormControl>
                    <Textarea
                      {...field}
                      value={field.value ?? ''}
                      placeholder={t('Optional request note')}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {!departmentsQuery.isLoading && departments.length === 0 ? (
              <div className='md:col-span-2'>
                <Empty className='min-h-[140px] border'>
                  <EmptyHeader>
                    <EmptyMedia variant='icon'>
                      <Coins className='size-4' />
                    </EmptyMedia>
                    <EmptyTitle>{t('No requestable departments')}</EmptyTitle>
                    <EmptyDescription>
                      {t(
                        'You must be an active member of a department before submitting a quota request from wallet.'
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
                  !user ||
                  !selectedDepartmentId ||
                  !capabilityQuery.data?.can_submit ||
                  submitMutation.isPending
                }
              >
                <Coins data-icon='inline-start' />
                {t('Submit quota request')}
              </Button>
            </div>
          </form>
        </Form>
      </CardContent>
    </Card>
  )
}

export async function invalidateQuotaRequestGovernanceQueries(
  queryClient: QueryClient,
  departmentId: number,
  tenantId: number
) {
  await Promise.all([
    queryClient.invalidateQueries({
      queryKey: governanceTimelineQueryKey(departmentId, tenantId),
    }),
    queryClient.invalidateQueries({
      queryKey: governanceNotificationQueryKey(departmentId, tenantId),
    }),
  ])
}
