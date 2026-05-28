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
  RefreshCw,
  RotateCcw,
  Search,
  Settings,
  UserPlus,
  Users,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
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
import { Input } from '@/components/ui/input'
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
  deactivateDepartmentMember,
  enterpriseOrganizationQueryKey,
  getDepartmentMembers,
  getUserDepartments,
  replaceUserDepartments,
  restoreDepartmentMember,
} from './api'
import { DepartmentTree } from './components/DepartmentTree'
import { useDepartmentTree } from './hooks/use-department-tree'
import type {
  DepartmentMemberItem,
  MembershipStatus,
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
