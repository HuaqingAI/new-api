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
import { Building2, RefreshCw, Settings, UserPlus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { DepartmentTree } from './components/DepartmentTree'
import { useDepartmentTree } from './hooks/use-department-tree'

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
        <EnterpriseOrganizationContent
          isLoading={isLoading}
          departments={departments}
        />
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
          <Button variant='outline' disabled>
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
