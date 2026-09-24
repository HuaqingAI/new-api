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
import { useQuery } from '@tanstack/react-query'
import { MonitorSmartphone, Search, Users } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTablePage, useDataTable } from '@/components/data-table'
import { ErrorState } from '@/components/error-state'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'

import { type AionUiClientPlatform, getClientInstallations } from '../api'
import {
  clientInstallationPlatformLabel,
  useClientInstallationColumns,
} from './client-installation-columns'

type ClientInstallationFilters = {
  keyword: string
  platform: '' | AionUiClientPlatform
  version: string
}

const INITIAL_FILTERS: ClientInstallationFilters = {
  keyword: '',
  platform: '',
  version: '',
}

const PLATFORM_OPTIONS: AionUiClientPlatform[] = [
  'windows_x64',
  'mac_arm64',
  'mac_x64',
]

export function ClientInstallations() {
  const { t } = useTranslation()
  const columns = useClientInstallationColumns()
  const [draftFilters, setDraftFilters] =
    useState<ClientInstallationFilters>(INITIAL_FILTERS)
  const [filters, setFilters] =
    useState<ClientInstallationFilters>(INITIAL_FILTERS)
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: 20 })
  const installationsQuery = useQuery({
    queryKey: [
      'aionui-client-packages',
      'installations',
      filters,
      pagination.pageIndex,
      pagination.pageSize,
    ],
    queryFn: () =>
      getClientInstallations({
        page: pagination.pageIndex + 1,
        pageSize: pagination.pageSize,
        keyword: filters.keyword.trim(),
        platform: filters.platform || undefined,
        version: filters.version.trim(),
      }),
  })
  const result = installationsQuery.data?.data
  const summary = result?.summary
  const { table } = useDataTable({
    data: result?.items ?? [],
    columns,
    getRowId: (row) => String(row.id),
    manualFiltering: true,
    manualPagination: true,
    onPaginationChange: setPagination,
    pagination,
    totalCount: result?.total ?? 0,
    withFacetedRowModel: false,
    withFilteredRowModel: false,
  })
  const metrics = [
    {
      icon: Users,
      label: t('Active users'),
      value: summary?.active_users ?? 0,
      active: true,
    },
    {
      icon: MonitorSmartphone,
      label: t('Active installations'),
      value: summary?.active_installations ?? 0,
      active: true,
    },
    {
      icon: Users,
      label: t('Total users'),
      value: summary?.total_users ?? 0,
      active: false,
    },
    {
      icon: MonitorSmartphone,
      label: t('Total installations'),
      value: summary?.total_installations ?? 0,
      active: false,
    },
  ]

  const submitFilters = () => {
    setFilters(draftFilters)
    setPagination((current) => ({ ...current, pageIndex: 0 }))
  }

  return (
    <section className='mt-6 space-y-4'>
      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
        {metrics.map((metric) => {
          const Icon = metric.icon
          return (
            <Card key={metric.label}>
              <CardContent className='flex items-start justify-between gap-3 py-4'>
                <div>
                  <p
                    className='text-muted-foreground text-sm'
                    title={
                      metric.active
                        ? t('Recent successful heartbeat (90 minutes)')
                        : undefined
                    }
                  >
                    {metric.label}
                  </p>
                  <p className='mt-1 text-2xl font-semibold tabular-nums'>
                    {metric.value}
                  </p>
                </div>
                <Icon
                  className='text-muted-foreground size-5'
                  aria-hidden='true'
                />
              </CardContent>
            </Card>
          )
        })}
      </div>

      <Card>
        <CardHeader className='gap-4 sm:flex-row sm:items-center sm:justify-between'>
          <CardTitle>{t('Client installations')}</CardTitle>
          <form
            className='flex w-full flex-wrap items-center gap-2 sm:w-auto'
            onSubmit={(event) => {
              event.preventDefault()
              submitFilters()
            }}
          >
            <Input
              aria-label={t('Search by user')}
              className='w-full sm:w-48'
              placeholder={t('Search by user')}
              value={draftFilters.keyword}
              onChange={(event) =>
                setDraftFilters({
                  ...draftFilters,
                  keyword: event.target.value,
                })
              }
            />
            <NativeSelect
              aria-label={t('Platform')}
              className='w-full sm:w-40'
              value={draftFilters.platform}
              onChange={(event) =>
                setDraftFilters({
                  ...draftFilters,
                  platform: event.target.value as '' | AionUiClientPlatform,
                })
              }
            >
              <NativeSelectOption value=''>
                {t('All platforms')}
              </NativeSelectOption>
              {PLATFORM_OPTIONS.map((platform) => (
                <NativeSelectOption key={platform} value={platform}>
                  {clientInstallationPlatformLabel(platform, t)}
                </NativeSelectOption>
              ))}
            </NativeSelect>
            <Input
              aria-label={t('Version')}
              className='w-full sm:w-28'
              placeholder={t('Version')}
              value={draftFilters.version}
              onChange={(event) =>
                setDraftFilters({
                  ...draftFilters,
                  version: event.target.value,
                })
              }
            />
            <Button type='submit' size='sm'>
              <Search className='size-4' />
              {t('Search')}
            </Button>
          </form>
        </CardHeader>
        <CardContent>
          {installationsQuery.isError ? (
            <ErrorState onRetry={() => installationsQuery.refetch()} />
          ) : (
            <DataTablePage
              columns={columns}
              emptyDescription={t('No clients have reported a heartbeat yet.')}
              emptyTitle={t('No client installations')}
              fixedHeight={false}
              isFetching={installationsQuery.isFetching}
              isLoading={installationsQuery.isLoading}
              paginationInFooter={false}
              skeletonKeyPrefix='aionui-client-installations-skeleton'
              table={table}
              toolbar={null}
            />
          )}
        </CardContent>
      </Card>
    </section>
  )
}
