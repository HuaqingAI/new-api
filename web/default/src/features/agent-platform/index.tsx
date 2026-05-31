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
  Compass,
  Layers3,
  Network,
  Puzzle,
  ShieldCheck,
  Sparkles,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SectionPageLayout } from '@/components/layout'
import { getAgentPlatformSkills } from './api'
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
import { cn } from '@/lib/utils'

type DomainCard = {
  key: string
  icon: React.ElementType
  title: string
  description: string
  status: string
}

const STATUS_TONES: Record<string, string> = {
  foundation: 'border-emerald-300 bg-emerald-50 text-emerald-700',
  pending: 'border-amber-300 bg-amber-50 text-amber-700',
  diagnostics: 'border-slate-300 bg-slate-50 text-slate-700',
}

export function AgentPlatformShell() {
  const { t } = useTranslation()
  const skillsQuery = useQuery({
    queryKey: ['agent-platform', 'skills', 'summary'],
    queryFn: getAgentPlatformSkills,
  })

  const domainCards: DomainCard[] = [
    {
      key: 'overview',
      icon: Compass,
      title: t('Agent Platform Overview'),
      description: t(
        'Track registry foundations, projection rollout, and the next implementation slices in one place.'
      ),
      status: t('Foundation ready'),
    },
    {
      key: 'clients',
      icon: Network,
      title: t('Clients'),
      description: t(
        'Client registration lands in the next epic, but this shell reserves the contract and capability declaration workspace now.'
      ),
      status: t('Pending Epic 2'),
    },
    {
      key: 'skills',
      icon: Puzzle,
      title: t('Skills'),
      description: t(
        'Skill management now reads from the control plane and anchors the next invoke-contract stories.'
      ),
      status: t('Skill control plane live'),
    },
    {
      key: 'knowledge',
      icon: BookOpen,
      title: t('Knowledge'),
      description: t(
        'Knowledge retrieval contracts and provider-backed detail views will use this section as their management shell.'
      ),
      status: t('Pending Epic 4'),
    },
    {
      key: 'agents',
      icon: Bot,
      title: t('Agents'),
      description: t(
        'Agent definition metadata and dependency boundaries will appear here without turning the platform into a runtime orchestrator.'
      ),
      status: t('Pending Epic 5'),
    },
    {
      key: 'publishing',
      icon: Layers3,
      title: t('Publishing'),
      description: t(
        'Projection state, visibility, callable readiness, and client-targeted rollout actions converge in this publishing lane.'
      ),
      status: t('Projection baseline ready'),
    },
    {
      key: 'audit',
      icon: ShieldCheck,
      title: t('Audit & Diagnostics'),
      description: t(
        'Lifecycle governance already records action trails; this shell reserves the drill-down surface for future diagnostics work.'
      ),
      status: t('Diagnostics baseline ready'),
    },
  ]

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Agent Platform')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <div className='flex items-center gap-3'>
          <Badge
            variant='outline'
            className='border-emerald-300 bg-emerald-50 text-emerald-700'
          >
            {t('Web Default MVP')}
          </Badge>
          <Button
            variant='default'
            render={
              <Link to='/system-settings/site'>
                {t('Review system settings')}
              </Link>
            }
          />
        </div>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='space-y-8'>
          <div className='space-y-2'>
            <p className='text-muted-foreground max-w-3xl text-sm leading-6'>
              {t(
                'A dedicated control-plane shell for clients, resources, publishing, and diagnostics in web/default.'
              )}
            </p>
          </div>
        <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-3'>
          {domainCards.map((card) => {
            const Icon = card.icon
            const tone =
              card.key === 'overview' || card.key === 'publishing'
                ? STATUS_TONES.foundation
                : card.key === 'audit'
                  ? STATUS_TONES.diagnostics
                  : STATUS_TONES.pending

            return (
              <Card
                key={card.key}
                className='border-border/70 bg-card/80 shadow-sm backdrop-blur'
              >
                <CardHeader className='space-y-4'>
                  <div className='flex items-start justify-between gap-3'>
                    <div className='flex items-center gap-3'>
                      <div className='bg-primary/10 text-primary flex h-10 w-10 items-center justify-center rounded-2xl'>
                        <Icon className='h-5 w-5' />
                      </div>
                      <div>
                        <CardTitle className='text-lg'>{card.title}</CardTitle>
                        <CardDescription>{card.description}</CardDescription>
                      </div>
                    </div>
                    <Badge
                      variant='outline'
                      className={cn('whitespace-nowrap', tone)}
                    >
                      {card.status}
                    </Badge>
                  </div>
                </CardHeader>
              </Card>
            )
          })}
        </div>

        <div className='grid gap-6 xl:grid-cols-[1.6fr_1fr]'>
          <Card className='border-border/70 bg-card/80 shadow-sm'>
            <CardHeader>
              <CardTitle>{t('Navigation contract')}</CardTitle>
              <CardDescription>
                {t(
                  'The MVP shell freezes the information architecture before the deeper client, publishing, and diagnostics screens arrive.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              <div className='grid gap-3 sm:grid-cols-2'>
                {[
                  t('Overview'),
                  t('Clients'),
                  t('Skills'),
                  t('Knowledge'),
                  t('Agents'),
                  t('Publishing'),
                  t('Audit & Diagnostics'),
                ].map((item, index) => (
                  <div
                    key={item}
                    className='border-border/60 bg-muted/40 flex items-center gap-3 rounded-2xl border px-4 py-3'
                  >
                    <span className='text-muted-foreground text-sm font-medium'>
                      {index + 1}.
                    </span>
                    <span className='font-medium'>{item}</span>
                  </div>
                ))}
              </div>
              <div className='border-border/60 bg-muted/40 rounded-2xl border p-4'>
                <div className='mb-2 flex items-center gap-2 text-sm font-medium'>
                  <Sparkles className='text-primary h-4 w-4' />
                  {t('Implementation note')}
                </div>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Classic theme parity stays out of MVP scope, so this shell intentionally anchors future Agent Platform work in web/default only.'
                  )}
                </p>
              </div>
            </CardContent>
          </Card>

          <Card className='border-border/70 bg-card/80 shadow-sm'>
            <CardHeader>
              <CardTitle>{t('Current focus')}</CardTitle>
              <CardDescription>
                {t(
                  'The shell now exposes the first live Skill management slice while deeper client, publishing, and diagnostics flows continue to land story by story.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className='space-y-4'>
                <div className='flex items-center justify-between gap-3'>
                  <div>
                    <p className='text-sm font-medium'>{t('Skill management')}</p>
                    <p className='text-muted-foreground text-sm'>
                      {t(
                        'Published and draft Skills now surface here from the live Agent Platform control plane.'
                      )}
                    </p>
                  </div>
                  <Badge
                    variant='outline'
                    className='border-emerald-300 bg-emerald-50 text-emerald-700'
                  >
                    {t('Epic 3 active')}
                  </Badge>
                </div>

                {skillsQuery.data?.success && Array.isArray(skillsQuery.data?.data?.items) && skillsQuery.data.data.items.length > 0 ? (
                  <div className='space-y-3'>
                    {skillsQuery.data.data.items.map((skill: any) => (
                      <div
                        key={skill.resource_id}
                        className='border-border/60 bg-muted/30 rounded-2xl border px-4 py-3'
                      >
                        <div className='flex items-start justify-between gap-3'>
                          <div className='space-y-1'>
                            <p className='font-medium'>{skill.display_name}</p>
                            <p className='text-muted-foreground text-xs'>
                              {skill.resource_id}
                            </p>
                          </div>
                          <Badge variant='outline'>
                            {skill.status || t('draft')}
                          </Badge>
                        </div>
                        <div className='text-muted-foreground mt-3 flex flex-wrap gap-4 text-xs'>
                          <span>{t('Owner')}: {skill.owner_user_id}</span>
                          <span>{t('Latest version')}: {skill.latest_version || t('Not versioned')}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <Empty className='min-h-[280px] rounded-2xl border border-dashed border-amber-300 bg-amber-50/50'>
                    <EmptyHeader>
                      <EmptyMedia
                        variant='icon'
                        className='bg-amber-100 text-amber-700'
                      >
                        <AlertTriangle className='h-4 w-4' />
                      </EmptyMedia>
                      <EmptyTitle>
                        {skillsQuery.isLoading
                          ? t('Loading Skills')
                          : t('No Skills yet')}
                      </EmptyTitle>
                      <EmptyDescription>
                        {skillsQuery.isLoading
                          ? t('The Skill control-plane slice is fetching the current registry view.')
                          : t(
                              'Create the first Skill in the Agent Platform control plane to turn this shell into a live management surface.'
                            )}
                      </EmptyDescription>
                    </EmptyHeader>
                    <EmptyContent>
                      <Button
                        variant='outline'
                        render={
                          <Link to='/enterprise-alerts'>
                            {t('Review existing diagnostics patterns')}
                          </Link>
                        }
                      />
                    </EmptyContent>
                  </Empty>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
