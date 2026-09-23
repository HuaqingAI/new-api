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
import { ArrowLeft } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Markdown } from '@/components/ui/markdown'
import { cn } from '@/lib/utils'

import { DocsTopBar } from './components/docs-top-bar'
import { getTutorialDoc, resolveTutorialMarkdown } from './tutorials'

type DocsTutorialPageProps = {
  toolId: string
  systemId: string
}

type TutorialHeading = {
  depth: 2 | 3
  id: string
  text: string
}

const headingPattern = /^(#{2,3})\s+(.+?)\s*#*\s*$/
const fencePattern = /^\s*(```|~~~)/

function cleanHeadingText(value: string): string {
  return value
    .replaceAll(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replaceAll(/<\/?[^>]+>/g, '')
    .replaceAll(/[*_`]/g, '')
    .trim()
}

function escapeHtml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}

function getTutorialHeadings(markdown: string): TutorialHeading[] {
  const headings: TutorialHeading[] = []
  let inFence = false

  markdown.split('\n').forEach((line) => {
    if (fencePattern.test(line)) {
      inFence = !inFence
      return
    }

    if (inFence) return

    const match = headingPattern.exec(line)
    if (!match) return

    headings.push({
      depth: match[1].length as 2 | 3,
      id: `tutorial-heading-${headings.length + 1}`,
      text: cleanHeadingText(match[2]),
    })
  })

  return headings
}

function addHeadingIds(markdown: string, headings: TutorialHeading[]): string {
  let inFence = false
  let headingIndex = 0

  return markdown
    .split('\n')
    .map((line) => {
      if (fencePattern.test(line)) {
        inFence = !inFence
        return line
      }

      if (inFence || !headingPattern.test(line)) {
        return line
      }

      const heading = headings[headingIndex]
      headingIndex += 1
      if (!heading) return line

      return `<h${heading.depth} id="${heading.id}">${escapeHtml(heading.text)}</h${heading.depth}>`
    })
    .join('\n')
}

function TutorialOutline(props: {
  activeHeadingId: string
  headings: TutorialHeading[]
}) {
  const { t } = useTranslation()

  if (props.headings.length === 0) {
    return null
  }

  return (
    <aside className='relative hidden self-stretch lg:block'>
      <nav
        aria-label={t('本页目录')}
        className='sticky top-24 max-h-[calc(100svh-7rem)] overflow-auto border-l py-1 text-sm'
      >
        <p className='text-foreground mb-4 pl-4 font-semibold'>
          {t('本页目录')}
        </p>
        <ol className='space-y-1'>
          {props.headings.map((heading) => (
            <li key={heading.id}>
              <a
                href={`#${heading.id}`}
                className={cn(
                  'block border-l-2 py-1.5 pr-2 transition-colors',
                  heading.depth === 3 ? 'pl-8 text-xs' : 'pl-4 text-sm',
                  heading.id === props.activeHeadingId
                    ? 'border-primary text-foreground font-medium'
                    : 'text-muted-foreground hover:text-foreground border-transparent'
                )}
              >
                <span className='line-clamp-2'>{heading.text}</span>
              </a>
            </li>
          ))}
        </ol>
      </nav>
    </aside>
  )
}

export function DocsTutorialPage(props: DocsTutorialPageProps) {
  const { t } = useTranslation()
  const tutorial = getTutorialDoc(props.toolId, props.systemId)
  const origin = typeof window === 'undefined' ? '' : window.location.origin
  const tutorialMarkdown = useMemo(() => {
    if (!tutorial) return ''
    return resolveTutorialMarkdown(tutorial.markdown, origin)
  }, [origin, tutorial])
  const headings = useMemo(
    () => getTutorialHeadings(tutorialMarkdown),
    [tutorialMarkdown]
  )
  const markdownWithHeadingIds = useMemo(
    () => addHeadingIds(tutorialMarkdown, headings),
    [headings, tutorialMarkdown]
  )
  const [activeHeadingId, setActiveHeadingId] = useState('')

  useEffect(() => {
    setActiveHeadingId(headings[0]?.id ?? '')
  }, [headings])

  useEffect(() => {
    if (headings.length === 0 || typeof IntersectionObserver === 'undefined') {
      return
    }

    const observer = new IntersectionObserver(
      (entries) => {
        const activeEntry = entries.find((entry) => entry.isIntersecting)
        if (activeEntry?.target.id) {
          setActiveHeadingId(activeEntry.target.id)
        }
      },
      { rootMargin: '-120px 0px -65% 0px', threshold: 0.1 }
    )

    headings.forEach((heading) => {
      const element = document.querySelector(`#${heading.id}`)
      if (element) {
        observer.observe(element)
      }
    })

    return () => observer.disconnect()
  }, [headings])

  return (
    <div className='bg-background text-foreground min-h-svh overflow-x-clip'>
      <DocsTopBar />
      <main className='mx-auto max-w-7xl px-4 py-10 md:px-8 md:py-14'>
        <Button
          variant='ghost'
          className='mb-8 h-9 rounded-lg'
          render={<a href='/docs' />}
        >
          <ArrowLeft data-icon='inline-start' />
          {t('返回配置手册')}
        </Button>

        {tutorial ? (
          <div className='grid gap-10 lg:grid-cols-[minmax(0,1fr)_16rem]'>
            <article className='max-w-4xl min-w-0'>
              <p className='text-muted-foreground text-sm'>{t('教程文档')}</p>
              <h1 className='mt-3 text-4xl leading-tight font-bold tracking-tight md:text-5xl'>
                {t(tutorial.title)}
              </h1>
              <Markdown className='mt-10 text-[15px] leading-7'>
                {markdownWithHeadingIds}
              </Markdown>
            </article>
            <TutorialOutline
              activeHeadingId={activeHeadingId}
              headings={headings}
            />
          </div>
        ) : (
          <section className='bg-muted/50 rounded-xl border p-8 text-center'>
            <h1 className='text-3xl font-bold'>{t('教程不存在')}</h1>
            <p className='text-muted-foreground mt-3'>
              {t('请选择配置手册中的工具和系统。')}
            </p>
          </section>
        )}
      </main>
    </div>
  )
}
