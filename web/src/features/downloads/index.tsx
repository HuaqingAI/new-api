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
import { Apple, Download, MonitorDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import {
  type AionUiClientPackage,
  type AionUiClientPlatform,
  clientPackageDownloadUrl,
  getLatestClientPackages,
} from './api'

const DOWNLOAD_PLATFORMS: Array<{
  platform: AionUiClientPlatform
  titleKey: string
  descriptionKey: string
  icon: typeof MonitorDown
}> = [
  {
    platform: 'windows_x64',
    titleKey: 'Windows',
    descriptionKey: 'Windows 10/11, Intel/AMD 64-bit',
    icon: MonitorDown,
  },
  {
    platform: 'mac_arm64',
    titleKey: 'macOS Apple Silicon',
    descriptionKey: 'Mac with Apple M-series chip',
    icon: Apple,
  },
  {
    platform: 'mac_x64',
    titleKey: 'macOS Intel',
    descriptionKey: 'Mac with Intel processor',
    icon: Apple,
  },
]

export function Downloads() {
  const { t } = useTranslation()
  const latestQuery = useQuery({
    queryKey: ['aionui-client-packages', 'latest'],
    queryFn: getLatestClientPackages,
  })
  const releases = new Map(
    (latestQuery.data?.data?.items ?? []).map((item) => [
      item.platform,
      item.release,
    ])
  )

  return (
    <PublicLayout>
      <div className='mx-auto max-w-6xl space-y-6 py-8'>
        <div className='space-y-2'>
          <h1 className='text-3xl font-bold tracking-tight'>
            {t('Client Downloads')}
          </h1>
          <p className='text-muted-foreground'>
            {t('Download the latest desktop client for your platform.')}
          </p>
        </div>
        <div className='grid gap-4 md:grid-cols-3'>
          {DOWNLOAD_PLATFORMS.map((platform) => (
            <DownloadCard
              key={platform.platform}
              title={t(platform.titleKey)}
              description={t(platform.descriptionKey)}
              icon={platform.icon}
              release={releases.get(platform.platform) ?? null}
              loading={latestQuery.isLoading}
            />
          ))}
        </div>
      </div>
    </PublicLayout>
  )
}

function DownloadCard(props: {
  title: string
  description: string
  icon: typeof MonitorDown
  release: AionUiClientPackage | null
  loading: boolean
}) {
  const { t } = useTranslation()
  const Icon = props.icon
  return (
    <Card className='h-full'>
      <CardHeader>
        <div className='flex items-center justify-between gap-3'>
          <CardTitle className='flex items-center gap-2'>
            <Icon className='size-5' />
            {props.title}
          </CardTitle>
          {props.release ? (
            <Badge variant='secondary'>v{props.release.version}</Badge>
          ) : null}
        </div>
      </CardHeader>
      <CardContent className='space-y-4'>
        <p className='text-muted-foreground text-sm'>{props.description}</p>
        {props.release ? (
          <div className='space-y-3'>
            <div className='text-muted-foreground text-sm'>
              {formatBytes(props.release.file_size)}
            </div>
            <Button
              className='w-full'
              render={
                <a href={clientPackageDownloadUrl(props.release.id)}>
                  <Download className='size-4' />
                  {t('Download')}
                </a>
              }
            />
          </div>
        ) : (
          <Button className='w-full' variant='outline' disabled>
            {props.loading ? t('Loading...') : t('Coming soon')}
          </Button>
        )}
      </CardContent>
    </Card>
  )
}

function formatBytes(value: number) {
  if (!Number.isFinite(value) || value <= 0) {
    return '-'
  }
  const units = ['B', 'KB', 'MB', 'GB']
  let size = value
  let unitIndex = 0
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex += 1
  }
  return `${size.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`
}
