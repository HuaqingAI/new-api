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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ColumnDef, PaginationState } from '@tanstack/react-table'
import { useCallback, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { DataTablePage, useDataTable } from '@/components/data-table'
import { Dialog } from '@/components/dialog'
import { ErrorState } from '@/components/error-state'
import { Button } from '@/components/ui/button'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import { formatTimestamp } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { getServerErrorMessage } from '@/lib/server-error-message'

import {
  changeClientNotification,
  getClientNotifications,
  NOTIFICATION_QUERY_KEY,
  type NotificationList,
  type NotificationStatus,
} from '../notification-api'

type NotificationItem = NotificationList['items'][number]

export function ClientNotificationList(props: {
  onClose: () => void
  onCreate: () => void
  onEdit: (id: string) => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [status, setStatus] = useState<NotificationStatus>('all')
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [target, setTarget] = useState<NotificationItem | null>(null)
  const query = useQuery({
    queryKey: [...NOTIFICATION_QUERY_KEY, { ...pagination, status }],
    queryFn: () =>
      getClientNotifications({
        page: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        status,
      }),
    meta: { errorToast: false },
    refetchOnMount: 'always',
  })
  const mutation = useMutation({
    mutationFn: (operation: {
      item: NotificationItem
      action: 'status' | 'delete'
    }) => changeClientNotification(operation.item, operation.action),
    onSuccess: (_, operation) => {
      if (operation.action === 'delete') setTarget(null)
      toast.success(t('Client notification updated'))
      return queryClient.invalidateQueries({ queryKey: NOTIFICATION_QUERY_KEY })
    },
    onError: (error) => handleServerError(error),
  })
  const ensurePageInRange = useCallback(
    (pageCount: number) => {
      if (
        query.data &&
        !query.isFetching &&
        pagination.pageIndex >= Math.max(1, pageCount)
      ) {
        setPagination((previous) => ({
          ...previous,
          pageIndex: Math.max(0, pageCount - 1),
        }))
      }
    },
    [query.data, query.isFetching, pagination.pageIndex]
  )
  const columns: ColumnDef<NotificationItem, unknown>[] = [
    {
      accessorKey: 'title',
      header: t('Title'),
      cell: ({ row }) => (
        <div className='max-w-sm break-words whitespace-normal'>
          <p className='line-clamp-2 font-medium'>{row.original.title}</p>
          <p className='text-muted-foreground line-clamp-2 text-xs'>
            {row.original.summary}
          </p>
        </div>
      ),
    },
    {
      accessorKey: 'enabled',
      header: t('Status'),
      cell: ({ row }) => (
        <Switch
          aria-label={t('Notification status')}
          checked={row.original.enabled}
          disabled={mutation.isPending || query.isFetching}
          onCheckedChange={() =>
            mutation.mutate({ item: row.original, action: 'status' })
          }
        />
      ),
    },
    {
      accessorKey: 'published_at',
      header: t('Published at'),
      cell: ({ row }) =>
        row.original.published_at
          ? formatTimestamp(row.original.published_at)
          : t('Unpublished'),
    },
    {
      accessorKey: 'updated_at',
      header: t('Updated at'),
      cell: ({ row }) => formatTimestamp(row.original.updated_at),
    },
    {
      id: 'actions',
      header: t('Actions'),
      cell: ({ row }) => (
        <div className='flex flex-wrap gap-1'>
          <Button
            variant='ghost'
            size='sm'
            disabled={mutation.isPending}
            onClick={() => props.onEdit(row.original.id)}
          >
            {t('Edit')}
          </Button>
          <Button
            variant='ghost'
            size='sm'
            className='text-destructive'
            disabled={mutation.isPending}
            onClick={() => setTarget(row.original)}
          >
            {t('Delete')}
          </Button>
        </div>
      ),
    },
  ]
  const { table } = useDataTable({
    data: query.data?.items ?? [],
    columns,
    totalCount: query.data?.total ?? 0,
    manualPagination: true,
    enableRowSelection: false,
    pagination,
    ensurePageInRange,
    getRowId: (item) => item.id,
    onPaginationChange: (updater) =>
      setPagination((previous) => {
        const next = typeof updater === 'function' ? updater(previous) : updater
        return next.pageSize === previous.pageSize
          ? next
          : { ...next, pageIndex: 0 }
      }),
  })

  return (
    <>
      <Dialog
        open
        title={t('Client notifications')}
        contentClassName='sm:max-w-5xl'
        onOpenChange={(open) => {
          if (!open && !mutation.isPending && !target) props.onClose()
        }}
      >
        <div className='mb-4 flex flex-wrap items-center justify-between gap-2'>
          <NativeSelect
            aria-label={t('Notification status')}
            value={status}
            onChange={(event) => {
              setStatus(event.target.value as NotificationStatus)
              setPagination((previous) => ({ ...previous, pageIndex: 0 }))
            }}
          >
            <NativeSelectOption value='all'>{t('All')}</NativeSelectOption>
            <NativeSelectOption value='enabled'>
              {t('Enabled')}
            </NativeSelectOption>
            <NativeSelectOption value='disabled'>
              {t('Disabled')}
            </NativeSelectOption>
          </NativeSelect>
          <div className='flex gap-2'>
            <Button
              variant='outline'
              disabled={query.isFetching || mutation.isPending}
              onClick={() => void query.refetch()}
            >
              {t('Refresh')}
            </Button>
            <Button onClick={props.onCreate} disabled={mutation.isPending}>
              {t('Add notification')}
            </Button>
          </div>
        </div>
        {query.isError ? (
          <ErrorState
            title={t('Failed to load client notifications')}
            description={getServerErrorMessage(query.error)}
            onRetry={() => void query.refetch()}
          />
        ) : (
          <DataTablePage
            table={table}
            columns={columns}
            isLoading={query.isPending}
            isFetching={query.isFetching}
            emptyTitle={t('No client notifications')}
            toolbarProps={null}
            fixedHeight={false}
            paginationInFooter={false}
          />
        )}
      </Dialog>
      <ConfirmDialog
        open={target !== null}
        onOpenChange={(open) => {
          if (!open && !mutation.isPending) setTarget(null)
        }}
        title={t('Delete notification')}
        desc={t('Delete "{{title}}"? This cannot be undone.', {
          title: target?.title,
        })}
        destructive
        confirmText={t('Confirm')}
        isLoading={mutation.isPending}
        handleConfirm={() => {
          if (target) mutation.mutate({ item: target, action: 'delete' })
        }}
      />
    </>
  )
}
