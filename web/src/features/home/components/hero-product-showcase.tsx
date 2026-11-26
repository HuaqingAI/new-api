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
import { Bot, Check, Circle } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

import { HeroTerminalDemo } from './hero-terminal-demo'

type PreviewMode = 'assistant' | 'api'

interface HeroProductShowcaseProps {
  className?: string
}

export function HeroProductShowcase(props: HeroProductShowcaseProps) {
  const { t } = useTranslation()
  const [previewMode, setPreviewMode] = useState<PreviewMode>('assistant')
  const isAssistantPreview = previewMode === 'assistant'

  return (
    <div className={cn('relative w-full pb-7 sm:pb-8', props.className)}>
      <div className='border-border/60 bg-background/95 relative overflow-hidden rounded-2xl border shadow-[0_24px_70px_-28px_rgba(34,42,76,0.3)] backdrop-blur-sm dark:border-white/[0.07] dark:bg-[#0b0f17]/95 dark:shadow-[0_26px_70px_-28px_rgba(0,0,0,0.8)]'>
        <div className='border-border/50 bg-muted/25 flex h-12 items-center border-b px-3 sm:h-14 sm:px-4 dark:border-white/[0.05] dark:bg-white/[0.025]'>
          <div aria-hidden className='flex items-center gap-1.5'>
            <Circle className='size-2.5 fill-red-400 text-red-400' />
            <Circle className='size-2.5 fill-amber-400 text-amber-400' />
            <Circle className='size-2.5 fill-emerald-400 text-emerald-400' />
          </div>

          <div
            className='border-border/60 bg-muted/50 absolute left-1/2 flex -translate-x-1/2 items-center rounded-lg border p-0.5 shadow-inner dark:border-white/[0.06] dark:bg-white/[0.04]'
            role='tablist'
            aria-label={t('Product preview')}
          >
            <button
              type='button'
              role='tab'
              aria-selected={isAssistantPreview}
              onClick={() => setPreviewMode('assistant')}
              className={cn(
                'rounded-md px-2.5 py-1.5 text-[10px] font-medium transition-colors sm:px-3 sm:text-xs',
                isAssistantPreview
                  ? 'bg-background text-foreground shadow-sm dark:bg-white/[0.09]'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              {t('Assistant')}
            </button>
            <button
              type='button'
              role='tab'
              aria-selected={!isAssistantPreview}
              onClick={() => setPreviewMode('api')}
              className={cn(
                'rounded-md px-2.5 py-1.5 text-[10px] font-medium transition-colors sm:px-3 sm:text-xs',
                !isAssistantPreview
                  ? 'bg-background text-foreground shadow-sm dark:bg-white/[0.09]'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              {t('API Calls')}
            </button>
          </div>

          <span className='text-muted-foreground/55 ml-auto hidden text-[9px] font-semibold tracking-[0.08em] sm:block'>
            {t('HTH · Powered by NewAPI')}
          </span>
        </div>

        {isAssistantPreview ? (
          <div
            role='tabpanel'
            className='relative aspect-[800/493] overflow-hidden bg-[#f7f8fa]'
          >
            <img
              src='/huaqing-assistant/hero-v2.webp'
              alt={t('Huaqing AI Assistant product interface')}
              width={1600}
              height={986}
              fetchPriority='high'
              className='block size-full object-contain'
            />
          </div>
        ) : (
          <div role='tabpanel' className='bg-muted/15 p-3 sm:p-4'>
            <HeroTerminalDemo className='max-w-none' />
          </div>
        )}
      </div>

      {isAssistantPreview ? (
        <>
          <div className='border-border/60 bg-background/90 absolute top-20 -right-4 hidden items-center gap-2.5 rounded-xl border px-3 py-2.5 shadow-[0_14px_35px_-18px_rgba(15,23,42,0.45)] backdrop-blur-md sm:flex dark:border-white/[0.08] dark:bg-[#111722]/90'>
            <span className='flex size-8 items-center justify-center rounded-lg bg-violet-500 text-white shadow-sm'>
              <Check className='size-4' />
            </span>
            <span>
              <strong className='block text-[11px] font-semibold'>
                {t('Task completed')}
              </strong>
              <span className='text-muted-foreground mt-0.5 block text-[9px]'>
                {t('Project report generated just now')}
              </span>
            </span>
          </div>

          <div className='border-border/60 bg-background/90 absolute bottom-0 -left-3 hidden items-center gap-2.5 rounded-xl border px-3 py-2.5 shadow-[0_14px_35px_-18px_rgba(15,23,42,0.45)] backdrop-blur-md sm:flex dark:border-white/[0.08] dark:bg-[#111722]/90'>
            <span className='relative flex size-8 items-center justify-center rounded-lg bg-blue-500/10 text-blue-600 dark:text-blue-400'>
              <Bot className='size-4' />
              <span className='border-background absolute -right-0.5 -bottom-0.5 size-2.5 rounded-full border-2 bg-emerald-500' />
            </span>
            <span>
              <strong className='block text-[11px] font-semibold'>
                {t('Multi-model gateway connected')}
              </strong>
              <span className='text-muted-foreground mt-0.5 block text-[9px]'>
                {t('Routing, quota, and usage are managed together')}
              </span>
            </span>
          </div>
        </>
      ) : null}
    </div>
  )
}
