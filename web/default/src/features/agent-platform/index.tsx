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
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import {
  AlertTriangle,
  BookOpen,
  Bot,
  Boxes,
  Compass,
  Layers3,
  Network,
  Puzzle,
  RefreshCw,
  ShieldCheck,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type {
  AgentPlatformItem,
  AgentPlatformListResponse,
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
  errorPrefix: string
  icon: React.ElementType
  isLoading: boolean
  onRetry: () => void
  response?: AgentPlatformListResponse
  title: string
}

type SummaryMetric = {
  key: string
  label: string
  value: string | number
  helper: string
  icon: React.ElementType
}

function formatStatusLabel(status: string, t: ReturnType<typeof useTranslation>['t']) {
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

function ResourceListCard(props: ResourceListCardProps) {
  const { t } = useTranslation()
  const Icon = props.icon
  const items = getItems(props.response)
  const loadFailed =
    props.response != null &&
    props.response.success === false &&
    !!props.response.message

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
          <LoadingState message={t('Loading...')} className='min-h-[240px]' />
        ) : loadFailed ? (
          <ErrorState
            className='min-h-[240px]'
            title={t('Failed to load data')}
            description={`${props.errorPrefix}: ${props.response?.message ?? t('Request failed')}`}
            onRetry={props.onRetry}
          />
        ) : items.length === 0 ? (
          <div className='flex min-h-[240px] items-center justify-center rounded-xl border border-dashed'>
            <div className='space-y-2 px-6 text-center'>
              <AlertTriangle className='text-muted-foreground mx-auto size-5' />
              <p className='font-medium'>{t('No Data')}</p>
              <p className='text-muted-foreground text-sm'>
                {props.emptyDescription}
              </p>
            </div>
          </div>
        ) : (
          <div className='space-y-3'>
            {items.map((item) => (
              <ResourceRow key={item.resource_id} item={item} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function ResourceRow(props: { item: AgentPlatformItem }) {
  const { t } = useTranslation()
  const item = props.item

  return (
    <div className='rounded-xl border bg-muted/20 px-4 py-3'>
      <div className='flex items-start justify-between gap-3'>
        <div className='space-y-1'>
          <p className='font-medium'>{item.display_name}</p>
          <p className='text-muted-foreground text-xs'>{item.resource_id}</p>
        </div>
        <StatusBadge
          label={formatStatusLabel(item.status, t)}
          variant={statusVariantFor(item.status)}
          copyable={false}
        />
      </div>
      <div className='text-muted-foreground mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs'>
        <span>
          {t('Owner')}: {item.owner_user_id}
        </span>
        <span>
          {t('Latest version')}: {item.latest_version || t('Not versioned')}
        </span>
        <span>
          {t('Tenant')}: {item.tenant_id || t('Global')}
        </span>
      </div>
    </div>
  )
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
      key: 'domains',
      label: t('Workspace domains'),
      value: 7,
      helper: t('Overview to diagnostics'),
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
                  <CardTitle>{t('Agent Platform Overview')}</CardTitle>
                  <CardDescription>
                    {t(
                      'The Agent Platform control plane now uses live admin routes, live control-plane data, and the same interaction language as the existing web/default governance modules.'
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
                      className='rounded-xl border bg-muted/20 px-4 py-3'
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

          <Tabs
            defaultValue='overview'
            className='space-y-6'
          >
            <TabsList className='grid w-full grid-cols-4 md:w-[560px]'>
              <TabsTrigger value='overview'>{t('Overview')}</TabsTrigger>
              <TabsTrigger value='skills'>{t('Skills')}</TabsTrigger>
              <TabsTrigger value='knowledge'>{t('Knowledge')}</TabsTrigger>
              <TabsTrigger value='agents'>{t('Agents')}</TabsTrigger>
            </TabsList>

            <TabsContent value='overview' className='space-y-6'>
              <div className='grid gap-4 xl:grid-cols-[1.2fr_1fr]'>
                <Card>
                  <CardHeader className='gap-3 border-b'>
                    <CardTitle>{t('Navigation contract')}</CardTitle>
                    <CardDescription>
                      {t(
                        'The management navigation order remains fixed so that later Clients, Publishing, and Audit slices can extend this page without reworking the information architecture.'
                      )}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className='pt-4'>
                    <div className='grid gap-3 sm:grid-cols-2'>
                      {[
                        {
                          icon: Compass,
                          label: t('Overview'),
                        },
                        {
                          icon: Network,
                          label: t('Clients'),
                        },
                        {
                          icon: Puzzle,
                          label: t('Skills'),
                        },
                        {
                          icon: BookOpen,
                          label: t('Knowledge'),
                        },
                        {
                          icon: Bot,
                          label: t('Agents'),
                        },
                        {
                          icon: Layers3,
                          label: t('Publishing'),
                        },
                        {
                          icon: ShieldCheck,
                          label: t('Audit & Diagnostics'),
                        },
                      ].map((entry, index) => {
                        const Icon = entry.icon
                        return (
                          <div
                            key={entry.label}
                            className='flex items-center gap-3 rounded-xl border bg-muted/20 px-4 py-3'
                          >
                            <span className='text-muted-foreground text-sm font-medium'>
                              {index + 1}.
                            </span>
                            <Icon className='text-muted-foreground size-4' />
                            <span className='font-medium'>{entry.label}</span>
                          </div>
                        )
                      })}
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className='gap-3 border-b'>
                    <CardTitle>{t('Current implementation focus')}</CardTitle>
                    <CardDescription>
                      {t(
                        'This stabilization pass closes the gap between planned control-plane capabilities and the live web/default admin experience.'
                      )}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className='space-y-3 pt-4'>
                    <div className='rounded-xl border bg-muted/20 px-4 py-3'>
                      <p className='font-medium'>{t('Live route surface')}</p>
                      <p className='text-muted-foreground mt-1 text-sm'>
                        {t(
                          'The Agent Platform entry now needs to be validated against the generated router tree, not just local feature code.'
                        )}
                      </p>
                    </div>
                    <div className='rounded-xl border bg-muted/20 px-4 py-3'>
                      <p className='font-medium'>{t('Control-plane API wiring')}</p>
                      <p className='text-muted-foreground mt-1 text-sm'>
                        {t(
                          'Skill, Knowledge, and Agent data should come from live control-plane endpoints, with explicit compatibility fallback only when required.'
                        )}
                      </p>
                    </div>
                    <div className='rounded-xl border bg-muted/20 px-4 py-3'>
                      <p className='font-medium'>{t('UI parity')}</p>
                      <p className='text-muted-foreground mt-1 text-sm'>
                        {t(
                          'Agent Platform pages must use the same loading, empty, error, and status expression patterns as the existing enterprise admin surfaces.'
                        )}
                      </p>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </TabsContent>

            <TabsContent value='skills'>
              <ResourceListCard
                title={t('Skill management')}
                description={t(
                  'Skill definitions exposed by the Agent Platform control plane.'
                )}
                badgeLabel={t('Epic 3 active')}
                icon={Puzzle}
                response={skillsQuery.data}
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
                response={knowledgeQuery.data}
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
                response={agentsQuery.data}
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
