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
import {
  ArrowRight,
  Cable,
  Clock3,
  Download,
  FolderKanban,
  Sparkles,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import { HeroProductShowcase } from '../hero-product-showcase'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

type ClientPlatform = 'windows' | 'macos' | 'linux' | 'other'

function detectClientPlatform(): ClientPlatform {
  if (typeof navigator === 'undefined') return 'other'

  const userAgentData = (
    navigator as Navigator & {
      userAgentData?: { mobile?: boolean; platform?: string }
    }
  ).userAgentData
  const userAgent = navigator.userAgent
  const platform = userAgentData?.platform || navigator.platform || userAgent
  const isMobile =
    userAgentData?.mobile ||
    /Android|iPhone|iPad|iPod|Mobile/i.test(userAgent) ||
    (/Mac/i.test(platform) && navigator.maxTouchPoints > 1)

  if (isMobile) return 'other'
  if (/Windows|Win32|Win64/i.test(platform)) return 'windows'
  if (/Macintosh|MacIntel|MacPPC|Mac68K|macOS/i.test(platform)) return 'macos'
  if (/Linux|X11/i.test(platform)) return 'linux'
  return 'other'
}

interface AgentCapability {
  title: string
  description: string
  icon: LucideIcon
}

const AGENT_CAPABILITIES: AgentCapability[] = [
  {
    title: 'Switch between multiple models',
    description: 'Unified access and management through NewAPI',
    icon: Sparkles,
  },
  {
    title: 'Workplace connectors',
    description: 'Calendars, tasks, approvals, and knowledge bases',
    icon: Cable,
  },
  {
    title: 'Scheduled automation',
    description: 'Let agents keep recurring work moving',
    icon: Clock3,
  },
  {
    title: 'Local files and projects',
    description: 'Create deliverables within authorized folders',
    icon: FolderKanban,
  },
]

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const clientPlatform = detectClientPlatform()

  let downloadButtonLabel = t('View client downloads')
  let downloadPrimaryDetail = t(
    'Choose the right desktop version for your device'
  )
  let downloadSecondaryDetail: string | null = null

  if (clientPlatform === 'windows') {
    downloadButtonLabel = t('Download for Windows')
    downloadPrimaryDetail = t('Windows 10 / 11')
    downloadSecondaryDetail = t('64-bit installer')
  } else if (clientPlatform === 'macos') {
    downloadButtonLabel = t('Download for macOS')
    downloadPrimaryDetail = t('Supports Apple Silicon and Intel')
  } else if (clientPlatform === 'linux') {
    downloadPrimaryDetail = t('Linux client is not available yet')
  }

  return (
    <section className='relative z-10 overflow-hidden px-6 pt-24 pb-14 md:pt-28 md:pb-20 lg:pt-32 lg:pb-24'>
      <div
        aria-hidden
        className='pointer-events-none absolute inset-0 -z-10 opacity-30 dark:opacity-[0.14]'
        style={{
          background: [
            'radial-gradient(ellipse 60% 50% at 16% 18%, oklch(0.72 0.18 250 / 75%) 0%, transparent 70%)',
            'radial-gradient(ellipse 52% 42% at 84% 18%, oklch(0.66 0.16 295 / 55%) 0%, transparent 72%)',
            'radial-gradient(ellipse 38% 30% at 48% 82%, oklch(0.65 0.14 20 / 24%) 0%, transparent 72%)',
          ].join(', '),
        }}
      />
      <div
        aria-hidden
        className='absolute inset-0 -z-10 bg-[linear-gradient(to_right,var(--border)_1px,transparent_1px),linear-gradient(to_bottom,var(--border)_1px,transparent_1px)] [mask-image:linear-gradient(to_bottom,black_5%,transparent_88%)] bg-[size:4rem_4rem] opacity-[0.07]'
      />

      <div className='mx-auto max-w-6xl'>
        <div className='grid grid-cols-1 items-center gap-12 lg:grid-cols-12 lg:gap-10'>
          <div className='flex flex-col items-start text-left lg:col-span-6'>
            <div
              className='landing-animate-fade-up mb-5 inline-flex items-center gap-2 rounded-full border border-red-500/20 bg-red-500/5 px-2.5 py-1.5 text-[11px] font-medium text-red-700 opacity-0 shadow-xs dark:border-red-400/20 dark:bg-red-400/5 dark:text-red-300'
              style={{ animationDelay: '0ms' }}
            >
              <span className='flex size-5 items-center justify-center rounded-md bg-red-700 text-[10px] font-bold text-white shadow-sm dark:bg-red-600'>
                H
              </span>
              <span className='relative flex size-1.5'>
                <span className='absolute inline-flex size-full animate-ping rounded-full bg-red-400 opacity-70' />
                <span className='relative inline-flex size-1.5 rounded-full bg-red-500' />
              </span>
              <span>
                {t('Official companion agent · Huaqing AI Assistant')}
              </span>
            </div>

            <h1
              className='landing-animate-fade-up text-[clamp(2.45rem,4.6vw,3.25rem)] leading-[1.06] font-bold tracking-[-0.045em] opacity-0'
              style={{ animationDelay: '60ms' }}
            >
              {t('Unified access to AI models,')}
              <br />
              <span className='bg-gradient-to-r from-blue-500 via-violet-500 to-purple-600 bg-clip-text text-transparent lg:whitespace-nowrap dark:from-blue-400 dark:via-violet-400 dark:to-purple-400'>
                {t('let agents finish the work')}
              </span>
            </h1>

            <p
              className='landing-animate-fade-up text-muted-foreground/85 mt-5 max-w-xl text-[15px] leading-7 opacity-0 md:text-base'
              style={{ animationDelay: '120ms' }}
            >
              {t(
                'NewAPI provides stable multi-model access and usage management. Huaqing AI Assistant brings these capabilities to the desktop, using local files, workplace connectors, and automation tools to move from understanding requests to delivering results.'
              )}
            </p>

            <div
              className='landing-animate-fade-up mt-7 flex w-full flex-col gap-3 opacity-0 sm:w-auto sm:flex-row sm:flex-wrap sm:items-center'
              style={{ animationDelay: '180ms' }}
            >
              <Button
                render={<Link to='/downloads' />}
                className='group h-11 justify-center rounded-lg bg-red-700 px-5 text-sm font-medium text-white shadow-[0_12px_28px_-14px_rgba(185,28,28,0.75)] hover:bg-red-800 sm:justify-start dark:bg-red-600 dark:hover:bg-red-500'
              >
                <Download className='mr-1.5 size-4' />
                {downloadButtonLabel}
                <ArrowRight className='ml-1 size-4 transition-transform duration-200 group-hover:translate-x-0.5' />
              </Button>

              <Button
                variant='outline'
                className='border-border/60 hover:border-border hover:bg-muted/50 h-11 justify-center rounded-lg px-5 text-sm font-medium sm:justify-start'
                render={
                  <Link
                    to={props.isAuthenticated ? '/dashboard' : '/sign-up'}
                  />
                }
              >
                {props.isAuthenticated
                  ? t('Go to Dashboard')
                  : t('Get Started')}
                <ArrowRight className='ml-1.5 size-4' />
              </Button>
            </div>

            <div
              className='landing-animate-fade-up text-muted-foreground/60 mt-3 flex flex-wrap items-center gap-x-2.5 gap-y-1.5 text-xs opacity-0'
              style={{ animationDelay: '220ms' }}
            >
              <span>{downloadPrimaryDetail}</span>
              {downloadSecondaryDetail ? (
                <>
                  <span
                    aria-hidden
                    className='size-0.5 rounded-full bg-current'
                  />
                  <span>{downloadSecondaryDetail}</span>
                </>
              ) : null}
            </div>

            <div
              className='landing-animate-fade-up mt-8 hidden w-full max-w-xl grid-cols-1 gap-2.5 opacity-0 sm:grid sm:grid-cols-2'
              style={{ animationDelay: '260ms' }}
            >
              {AGENT_CAPABILITIES.map((capability) => {
                const Icon = capability.icon
                return (
                  <div
                    key={capability.title}
                    className='border-border/50 bg-background/55 flex min-h-16 items-center gap-3 rounded-xl border px-3.5 py-3 backdrop-blur-sm'
                  >
                    <span className='flex size-8 shrink-0 items-center justify-center rounded-lg border border-blue-500/10 bg-gradient-to-br from-blue-500/10 to-violet-500/10 text-violet-600 dark:text-violet-300'>
                      <Icon className='size-4' />
                    </span>
                    <span>
                      <strong className='block text-xs font-semibold'>
                        {t(capability.title)}
                      </strong>
                      <span className='text-muted-foreground mt-0.5 block text-[10px] leading-4'>
                        {t(capability.description)}
                      </span>
                    </span>
                  </div>
                )
              })}
            </div>
          </div>

          <div
            className='landing-animate-fade-up flex w-full justify-center opacity-0 lg:col-span-6'
            style={{ animationDelay: '320ms' }}
          >
            <HeroProductShowcase />
          </div>
        </div>

        <div
          className='landing-animate-fade-up border-border/50 bg-background/50 mt-8 flex flex-col gap-3 rounded-xl border px-4 py-3.5 opacity-0 backdrop-blur-sm md:flex-row md:items-center md:justify-between'
          style={{ animationDelay: '380ms' }}
        >
          <div className='flex items-center gap-2 text-xs font-semibold'>
            <span className='size-2 rounded-full bg-gradient-to-br from-blue-500 to-violet-500 shadow-[0_0_0_5px_rgba(99,102,241,0.08)]' />
            {t('From unified access to finished deliverables')}
          </div>
          <div className='text-muted-foreground flex flex-wrap items-center gap-x-2 gap-y-1.5 text-[10px] md:justify-end'>
            <span className='text-violet-500'>01</span>
            <span>{t('NewAPI model gateway')}</span>
            <ArrowRight className='size-3 opacity-50' />
            <span className='text-violet-500'>02</span>
            <span>{t('Huaqing AI Assistant')}</span>
            <ArrowRight className='size-3 opacity-50' />
            <span className='text-violet-500'>03</span>
            <span>{t('Workplace and local tools')}</span>
            <ArrowRight className='size-3 opacity-50' />
            <span className='text-violet-500'>04</span>
            <span>{t('Documents, spreadsheets, and task results')}</span>
          </div>
        </div>
      </div>
    </section>
  )
}
