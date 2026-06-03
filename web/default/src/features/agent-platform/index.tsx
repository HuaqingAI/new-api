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
import { Link } from '@tanstack/react-router'
import {
  BookOpen,
  Bot,
  Boxes,
  Layers3,
  PanelRightOpen,
  Puzzle,
  RefreshCw,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatTimestamp } from '@/lib/format'
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
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import type {
  AgentPlatformItem,
  AgentPlatformListResponse,
  AgentPlatformResourceType,
} from './api'
import {
  getAgentPlatformAgents,
  getAgentPlatformKnowledge,
  getAgentPlatformSkills,
} from './api'

type ResourceListCardProps = {
  badgeLabel: string
  description: string
  emptyDescription: string
  error?: unknown
  errorPrefix: string
  icon: React.ElementType
  isError: boolean
  isLoading: boolean
  onRetry: () => void
  response?: AgentPlatformListResponse
  title: string
  typeLabel: string
}

type SummaryMetric = {
  key: string
  label: string
  value: string | number
  helper: string
  icon: React.ElementType
}

function formatStatusLabel(
  status: string,
  t: ReturnType<typeof useTranslation>['t']
) {
  const value = status.trim().toLowerCase()
  if (!value) {
    return t('Unknown')
  }
  return t(value)
}

function statusVariantFor(value: string) {
  switch (value.trim().toLowerCase()) {
    case 'published':
      return 'success' as const
    case 'draft':
      return 'warning' as const
    case 'disabled':
    case 'offline':
      return 'neutral' as const
    case 'revoked':
      return 'danger' as const
    default:
      return 'info' as const
  }
}

function getItems(response?: AgentPlatformListResponse) {
  return response?.success && Array.isArray(response.data?.items)
    ? response.data.items
    : []
}

function getErrorMessage(error: unknown, fallback: string) {
  if (error instanceof Error && error.message) {
    return error.message
  }
  if (
    error != null &&
    typeof error === 'object' &&
    'message' in error &&
    typeof error.message === 'string' &&
    error.message.length > 0
  ) {
    return error.message
  }
  return fallback
}

function ResourceListCard(props: ResourceListCardProps) {
  const { t } = useTranslation()
  const Icon = props.icon
  const items = getItems(props.response)
  const loadFailed =
    props.isError ||
    (props.response != null && props.response.success === false)
  const errorMessage =
    props.response?.message ?? getErrorMessage(props.error, t('Request failed'))

  return (
    <Card>
      <CardHeader className='gap-3 border-b'>
        <div className='flex items-start justify-between gap-3'>
          <div className='space-y-1'>
            <CardTitle className='flex items-center gap-2'>
              <span className='bg-primary/10 text-primary inline-flex size-8 items-center justify-center rounded-lg'>
                <Icon className='size-4' />
              </span>
              {props.title}
            </CardTitle>
            <CardDescription>{props.description}</CardDescription>
          </div>
          <Badge variant='outline'>{props.badgeLabel}</Badge>
        </div>
      </CardHeader>
      <CardContent className='pt-4'>
        {props.isLoading ? (
          <ResourceTableSkeleton />
        ) : loadFailed ? (
          <ErrorState
            className='min-h-[240px]'
            title={t('Failed to load data')}
            description={`${props.errorPrefix}: ${errorMessage}`}
            onRetry={props.onRetry}
          />
        ) : items.length === 0 ? (
          <Empty className='min-h-[240px] border'>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <Icon className='text-muted-foreground size-5' />
              </EmptyMedia>
              <EmptyTitle>{t('No resources found')}</EmptyTitle>
              <EmptyDescription>{props.emptyDescription}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent />
          </Empty>
        ) : (
          <ResourceTable items={items} typeLabel={props.typeLabel} />
        )}
      </CardContent>
    </Card>
  )
}

function ResourceTable(props: {
  items: AgentPlatformItem[]
  typeLabel: string
}) {
  const { t } = useTranslation()

  return (
    <div className='overflow-x-auto rounded-md border'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Resource')}</TableHead>
            <TableHead>{t('Type')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Owner')}</TableHead>
            <TableHead>{t('Latest version')}</TableHead>
            <TableHead>{t('Tenant')}</TableHead>
            <TableHead>{t('Updated At')}</TableHead>
            <TableHead className='text-right'>{t('Actions')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {props.items.map((item) => (
            <ResourceRow
              key={item.resource_id}
              item={item}
              typeLabel={props.typeLabel}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function ResourceTableSkeleton() {
  return (
    <div className='space-y-3'>
      <Skeleton className='h-10 w-full' />
      <Skeleton className='h-12 w-full' />
      <Skeleton className='h-12 w-full' />
      <Skeleton className='h-12 w-4/5' />
    </div>
  )
}

function ResourceRow(props: { item: AgentPlatformItem; typeLabel: string }) {
  const { t } = useTranslation()
  const item = props.item

  return (
    <TableRow>
      <TableCell>
        <div className='min-w-[220px] space-y-1'>
          <div className='font-medium'>{item.display_name}</div>
          <div className='text-muted-foreground text-xs'>
            {item.resource_id}
          </div>
        </div>
      </TableCell>
      <TableCell>
        <Badge variant='outline'>{props.typeLabel}</Badge>
      </TableCell>
      <TableCell>
        <StatusBadge
          label={formatStatusLabel(item.status, t)}
          variant={statusVariantFor(item.status)}
          copyable={false}
        />
      </TableCell>
      <TableCell>#{item.owner_user_id}</TableCell>
      <TableCell>{item.latest_version || t('Not versioned')}</TableCell>
      <TableCell>{item.tenant_id || t('Global')}</TableCell>
      <TableCell>{formatTimestamp(item.updated_at)}</TableCell>
      <TableCell className='text-right'>
        <Button variant='outline' size='sm' disabled>
          <PanelRightOpen className='size-4' />
          {t('Details')}
        </Button>
      </TableCell>
    </TableRow>
  )
}

function countResourcesByStatus(
  resourceSets: AgentPlatformItem[][],
  status: string
) {
  return resourceSets.reduce(
    (total, items) =>
      total +
      items.filter((item) => item.status.trim().toLowerCase() === status)
        .length,
    0
  )
}

function resourceTypeLabel(
  type: AgentPlatformResourceType,
  t: (key: string) => string
) {
  switch (type) {
    case 'skill':
      return t('Skill')
    case 'knowledge':
      return t('Knowledge')
    case 'agent':
      return t('Agent')
  }
}

export function AgentPlatformShell() {
  const { t } = useTranslation()

  const skillsQuery = useQuery({
    queryKey: ['agent-platform', 'skills', 'summary'],
    queryFn: getAgentPlatformSkills,
  })
  const knowledgeQuery = useQuery({
    queryKey: ['agent-platform', 'knowledge', 'summary'],
    queryFn: getAgentPlatformKnowledge,
  })
  const agentsQuery = useQuery({
    queryKey: ['agent-platform', 'agents', 'summary'],
    queryFn: getAgentPlatformAgents,
  })

  const skillItems = getItems(skillsQuery.data)
  const knowledgeItems = getItems(knowledgeQuery.data)
  const agentItems = getItems(agentsQuery.data)
  const resourceSets = [skillItems, knowledgeItems, agentItems]

  const metrics: SummaryMetric[] = [
    {
      key: 'skills',
      label: t('Skills'),
      value: skillItems.length,
      helper: t('Live control-plane entries'),
      icon: Puzzle,
    },
    {
      key: 'knowledge',
      label: t('Knowledge'),
      value: knowledgeItems.length,
      helper: t('Provider-backed resources'),
      icon: BookOpen,
    },
    {
      key: 'agents',
      label: t('Agents'),
      value: agentItems.length,
      helper: t('Definition templates'),
      icon: Bot,
    },
    {
      key: 'published',
      label: t('Published resources'),
      value: countResourcesByStatus(resourceSets, 'published'),
      helper: t('Ready for open capability exposure'),
      icon: Layers3,
    },
  ]

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Agent Platform')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <div className='flex items-center gap-2'>
          <Button
            variant='outline'
            size='sm'
            onClick={() => {
              void skillsQuery.refetch()
              void knowledgeQuery.refetch()
              void agentsQuery.refetch()
            }}
            disabled={
              skillsQuery.isFetching ||
              knowledgeQuery.isFetching ||
              agentsQuery.isFetching
            }
          >
            <RefreshCw className='size-4' />
            {t('Refresh')}
          </Button>
          <Button
            variant='outline'
            size='sm'
            render={
              <Link to='/system-settings/site'>{t('System Settings')}</Link>
            }
          />
        </div>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='space-y-6'>
          <Card>
            <CardHeader className='gap-3 border-b'>
              <div className='flex items-start gap-3'>
                <span className='bg-primary/10 text-primary inline-flex size-10 items-center justify-center rounded-xl'>
                  <Boxes className='size-5' />
                </span>
                <div className='space-y-1'>
                  <CardTitle>{t('Resource control plane')}</CardTitle>
                  <CardDescription>
                    {t(
                      'Manage Skill, Knowledge, and Agent resources from the live Agent Platform control-plane endpoints.'
                    )}
                  </CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent className='pt-4'>
              <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
                {metrics.map((metric) => {
                  const Icon = metric.icon
                  return (
                    <div
                      key={metric.key}
                      className='bg-muted/20 rounded-xl border px-4 py-3'
                    >
                      <div className='flex items-center justify-between gap-3'>
                        <span className='text-muted-foreground text-sm font-medium'>
                          {metric.label}
                        </span>
                        <Icon className='text-muted-foreground size-4' />
                      </div>
                      <div className='mt-2 text-2xl font-semibold'>
                        {metric.value}
                      </div>
                      <p className='text-muted-foreground mt-1 text-xs'>
                        {metric.helper}
                      </p>
                    </div>
                  )
                })}
              </div>
            </CardContent>
          </Card>

          <Tabs defaultValue='skills' className='space-y-6'>
            <TabsList className='grid w-full grid-cols-3 md:w-[420px]'>
              <TabsTrigger value='skills'>{t('Skills')}</TabsTrigger>
              <TabsTrigger value='knowledge'>{t('Knowledge')}</TabsTrigger>
              <TabsTrigger value='agents'>{t('Agents')}</TabsTrigger>
            </TabsList>

            <TabsContent value='skills'>
              <ResourceListCard
                title={t('Skill management')}
                description={t(
                  'Skill definitions exposed by the Agent Platform control plane.'
                )}
                badgeLabel={t('Epic 3 active')}
                icon={Puzzle}
                typeLabel={resourceTypeLabel('skill', t)}
                response={skillsQuery.data}
                isError={skillsQuery.isError}
                error={skillsQuery.error}
                isLoading={skillsQuery.isLoading}
                onRetry={() => void skillsQuery.refetch()}
                errorPrefix={t('Skill API request failed')}
                emptyDescription={t(
                  'No Skill resources are currently available in the control plane.'
                )}
              />
            </TabsContent>

            <TabsContent value='knowledge'>
              <ResourceListCard
                title={t('Knowledge management')}
                description={t(
                  'Provider-backed retrieval resources managed by the Agent Platform control plane.'
                )}
                badgeLabel={t('Epic 4 active')}
                icon={BookOpen}
                typeLabel={resourceTypeLabel('knowledge', t)}
                response={knowledgeQuery.data}
                isError={knowledgeQuery.isError}
                error={knowledgeQuery.error}
                isLoading={knowledgeQuery.isLoading}
                onRetry={() => void knowledgeQuery.refetch()}
                errorPrefix={t('Knowledge API request failed')}
                emptyDescription={t(
                  'No Knowledge resources are currently available in the control plane.'
                )}
              />
            </TabsContent>

            <TabsContent value='agents'>
              <ResourceListCard
                title={t('Agent definitions')}
                description={t(
                  'Agent definition templates managed by the control plane without implying server-side runtime ownership.'
                )}
                badgeLabel={t('Epic 5 active')}
                icon={Bot}
                typeLabel={resourceTypeLabel('agent', t)}
                response={agentsQuery.data}
                isError={agentsQuery.isError}
                error={agentsQuery.error}
                isLoading={agentsQuery.isLoading}
                onRetry={() => void agentsQuery.refetch()}
                errorPrefix={t('Agent API request failed')}
                emptyDescription={t(
                  'No Agent definition resources are currently available in the control plane.'
                )}
              />
            </TabsContent>
          </Tabs>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
