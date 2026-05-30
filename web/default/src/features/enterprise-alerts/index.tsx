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
import { useMemo } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { AlertTriangle, Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatTimestamp } from '@/lib/format'
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { SectionPageLayout } from '@/components/layout'
import { alertEventsListQueryKey, getAlertEvents } from './api'
import type { AlertEventItem, EnterpriseAlertsSearch } from './types'

export const enterpriseAlertsSearchSchema = z.object({
  tenant_id: z.coerce.number().int().nonnegative().optional().catch(undefined),
  department_id: z.coerce.number().int().positive().optional().catch(undefined),
  user_id: z.coerce.number().int().positive().optional().catch(undefined),
  username: z.string().optional().catch(undefined),
  model_name: z.string().optional().catch(undefined),
  risk_type: z.string().optional().catch(undefined),
  from: z.coerce.number().int().positive().optional().catch(undefined),
  to: z.coerce.number().int().positive().optional().catch(undefined),
  page: z.coerce.number().int().positive().optional().catch(1),
  page_size: z.coerce.number().int().positive().optional().catch(20),
})

export function mapAlertFilterFormToSearch(values: AlertFilterFormValues) {
  return {
    tenant_id: parseOptionalNumber(values.tenant_id),
    department_id: parseOptionalNumber(values.department_id),
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

export function EnterpriseAlertsPage() {
  const { t } = useTranslation()
  const search = useSearch({
    from: '/_authenticated/enterprise-alerts/',
  }) as EnterpriseAlertsSearch
  const navigate = useNavigate()

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

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Enterprise Alerts')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='space-y-6'>
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
                {t('Only traceable summary fields are shown. Sensitive raw prompts are never returned here.')}
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
                          <TableCell>{formatTimestamp(item.created_at)}</TableCell>
                          <TableCell>
                            {formatDepartmentSnapshot(item, t('Unassigned'))}
                          </TableCell>
                          <TableCell>
                            {item.username} (#{item.user_id})
                          </TableCell>
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
                            if ((alertsQuery.data?.page ?? 1) > 1) {
                              changePage((alertsQuery.data?.page ?? 1) - 1)
                            }
                          }}
                          text={t('Previous')}
                        />
                      </PaginationItem>
                      <PaginationItem>
                        <PaginationLink href='#' isActive>
                          {alertsQuery.data?.page ?? 1}
                        </PaginationLink>
                      </PaginationItem>
                      <PaginationItem>
                        <PaginationNext
                          href='#'
                          onClick={(event) => {
                            event.preventDefault()
                            if ((alertsQuery.data?.page ?? 1) < totalPages) {
                              changePage((alertsQuery.data?.page ?? 1) + 1)
                            }
                          }}
                          text={t('Next')}
                        />
                      </PaginationItem>
                    </PaginationContent>
                  </Pagination>
                </>
              ) : (
                <Empty>
                  <EmptyHeader>
                    <EmptyMedia variant='icon'>
                      <AlertTriangle className='size-4' />
                    </EmptyMedia>
                    <EmptyTitle>{t('No alert events found')}</EmptyTitle>
                    <EmptyDescription>
                      {t(
                        'Adjust the filter set to widen the query window or inspect another department snapshot.'
                      )}
                    </EmptyDescription>
                  </EmptyHeader>
                  <EmptyContent />
                </Empty>
              )}
            </CardContent>
          </Card>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function parseOptionalNumber(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  const parsed = Number(trimmed)
  if (!Number.isInteger(parsed) || parsed <= 0) return undefined
  return parsed
}

function numberToString(value?: number) {
  if (value === undefined) return ''
  return String(value)
}

function emptyToUndefined(value?: string) {
  const trimmed = value?.trim()
  return trimmed ? trimmed : undefined
}
