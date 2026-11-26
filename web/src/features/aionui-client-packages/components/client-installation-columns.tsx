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
import type { ColumnDef } from '@tanstack/react-table'
import type { TFunction } from 'i18next'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { BadgeListCell, TruncatedCell } from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'

import type { AionUiClientInstallation, AionUiClientPlatform } from '../api'

export function useClientInstallationColumns(): ColumnDef<AionUiClientInstallation>[] {
  const { t } = useTranslation()

  return useMemo(
    (): ColumnDef<AionUiClientInstallation>[] => [
      {
        id: 'status',
        accessorFn: (row) => row.is_active,
        header: t('Status'),
        meta: { mobileBadge: true },
        cell: ({ row }) => (
          <StatusBadge
            copyable={false}
            label={row.original.is_active ? t('Active') : t('Offline')}
            variant={row.original.is_active ? 'success' : 'neutral'}
          />
        ),
        size: 96,
      },
      {
        id: 'user',
        accessorFn: (row) => row.display_name || row.username,
        header: t('User'),
        meta: { mobileTitle: true },
        cell: ({ row }) => (
          <div className='min-w-0'>
            <TruncatedCell className='font-medium'>
              {row.original.display_name || row.original.username}
            </TruncatedCell>
            <TruncatedCell className='text-muted-foreground text-xs'>
              {row.original.email}
            </TruncatedCell>
          </div>
        ),
        size: 224,
      },
      {
        accessorKey: 'departments',
        header: t('Departments'),
        cell: ({ row }) => (
          <BadgeListCell
            items={row.original.departments.map((department) => (
              <StatusBadge
                key={department}
                copyable={false}
                label={department}
                variant='neutral'
              />
            ))}
          />
        ),
        size: 180,
      },
      {
        accessorKey: 'platform',
        header: t('Platform'),
        cell: ({ row }) => (
          <span>
            {clientInstallationPlatformLabel(row.original.platform, t)}
          </span>
        ),
        size: 168,
      },
      {
        accessorKey: 'client_version',
        header: t('Version'),
        cell: ({ row }) => <span>{row.original.client_version}</span>,
        size: 104,
      },
      {
        accessorKey: 'lan_ip',
        header: t('Client-reported LAN IP'),
        cell: ({ row }) => <span>{row.original.lan_ip || '-'}</span>,
        size: 160,
      },
      {
        accessorKey: 'first_heartbeat_at',
        header: t('First reported'),
        cell: ({ row }) => (
          <span>
            {formatClientInstallationDateTime(row.original.first_heartbeat_at)}
          </span>
        ),
        meta: { mobileHidden: true },
        size: 176,
      },
      {
        accessorKey: 'last_heartbeat_at',
        header: t('Last heartbeat'),
        cell: ({ row }) => (
          <span>
            {formatClientInstallationDateTime(row.original.last_heartbeat_at)}
          </span>
        ),
        size: 176,
      },
    ],
    [t]
  )
}

export function clientInstallationPlatformLabel(
  platform: AionUiClientPlatform,
  t: TFunction
): string {
  switch (platform) {
    case 'windows_x64':
      return t('Windows x64')
    case 'mac_arm64':
      return t('macOS Apple Silicon')
    case 'mac_x64':
      return t('macOS Intel')
  }
}

function formatClientInstallationDateTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '-'
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  }).format(date)
}
