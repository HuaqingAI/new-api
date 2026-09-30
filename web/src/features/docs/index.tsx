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
import { ArrowRight, Cable, ExternalLink, Sparkles } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import { DocsTopBar } from './components/docs-top-bar'
import {
  getTutorialDoc,
  getTutorialPath,
  resolveTutorialMarkdown,
  type TutorialSystemId as SystemId,
  type TutorialToolId as ToolId,
} from './tutorials'

type ToolOption = {
  id: ToolId
  name: string
  description: string
  icon?: string
}

type SystemOption = {
  id: SystemId
  name: string
}

const TOOL_OPTIONS: ToolOption[] = [
  {
    id: 'claude-code',
    name: 'Claude Code',
    description: '在终端里使用 Claude 编程',
    icon: 'Claude.Color',
  },
  {
    id: 'codex-desktop',
    name: 'Codex 桌面端',
    description: '使用有聊天窗口的 Codex 应用',
    icon: 'OpenAI.Color',
  },
  {
    id: 'codex-cli',
    name: 'Codex CLI',
    description: '在终端里使用 Codex 命令',
    icon: 'OpenAI.Color',
  },
  {
    id: 'openclaw',
    name: 'OpenClaw',
    description: '连接本地网关、飞书等渠道',
  },
]

const SYSTEM_OPTIONS: SystemOption[] = [
  { id: 'windows', name: 'Windows' },
  { id: 'macos', name: 'macOS' },
]

function ToolIcon(props: { option: ToolOption }) {
  if (props.option.icon) {
    return (
      <span className='flex size-8 items-center justify-center'>
        {getLobeIcon(props.option.icon, 32)}
      </span>
    )
  }

  return (
    <span className='bg-primary/10 text-primary flex size-8 items-center justify-center rounded-lg'>
      <Cable className='size-4' aria-hidden='true' />
    </span>
  )
}

function ToolButton(props: {
  option: ToolOption
  selected: boolean
  onSelect: (id: ToolId) => void
}) {
  const { t } = useTranslation()

  return (
    <Button
      type='button'
      variant='outline'
      role='radio'
      aria-checked={props.selected}
      className={cn(
        'relative h-auto min-h-22 justify-start rounded-lg p-3 text-start shadow-none',
        props.selected && 'border-primary bg-primary/10'
      )}
      onClick={() => props.onSelect(props.option.id)}
    >
      <ToolIcon option={props.option} />
      <span className='min-w-0'>
        <span className='block truncate font-semibold'>
          {props.option.name}
        </span>
        <span className='text-muted-foreground mt-1 block text-xs font-normal whitespace-normal'>
          {t(props.option.description)}
        </span>
      </span>
      <span
        className={cn(
          'absolute top-2 right-2 size-2 rounded-full border',
          props.selected
            ? 'border-primary bg-primary'
            : 'border-border bg-background'
        )}
        aria-hidden='true'
      />
    </Button>
  )
}

function SystemButton(props: {
  option: SystemOption
  selected: boolean
  onSelect: (id: SystemId) => void
}) {
  return (
    <Button
      type='button'
      variant={props.selected ? 'default' : 'outline'}
      role='radio'
      aria-checked={props.selected}
      className='h-11 min-w-0 rounded-lg'
      onClick={() => props.onSelect(props.option.id)}
    >
      {props.option.name}
    </Button>
  )
}

function AssistantPanel(props: {
  selectedTool: ToolOption
  selectedSystem: SystemOption
}) {
  const { t } = useTranslation()
  const [copyStatus, setCopyStatus] = useState<'prompt' | 'full-guide' | null>(
    null
  )
  const tutorialPath = getTutorialPath(
    props.selectedTool.id,
    props.selectedSystem.id
  )
  const origin = typeof window === 'undefined' ? '' : window.location.origin
  const tutorialUrl = origin
    ? new URL(tutorialPath, origin).toString()
    : tutorialPath
  const tutorial = getTutorialDoc(
    props.selectedTool.id,
    props.selectedSystem.id
  )

  useEffect(() => {
    setCopyStatus(null)
  }, [props.selectedTool.id, props.selectedSystem.id])

  const prompt = [
    t(
      '我是新手，正在 {{system}} 上配置 {{tool}}，使用HTH API。请先阅读这份教程：',
      {
        tool: props.selectedTool.name,
        system: props.selectedSystem.name,
      }
    ),
    tutorialUrl,
    '',
    t(
      '请先确认我已经安装了什么、进行到哪一步。每次只指导一个步骤，写清楚在哪里操作、填什么、成功后会看到什么，等我反馈再继续。遇到错误先根据教程排查，不要猜测不存在的模型、分组或界面选项。已有配置先备份并只合并必要字段。API Key 用 sk-xxx 占位，由我在本机填写。'
    ),
    '',
    t(
      '如果无法读取链接，请明确告诉我，让我粘贴完整教程。你没有本机操作能力时，请指导我执行，不要声称已经替我修改成功。'
    ),
  ].join('\n')
  const fullGuide = tutorial
    ? [
        prompt,
        '',
        t('{{tool}} · {{system}} 配置教程', {
          tool: props.selectedTool.name,
          system: props.selectedSystem.name,
        }),
        '',
        resolveTutorialMarkdown(tutorial.markdown, origin),
      ].join('\n')
    : prompt

  return (
    <aside
      id='assistant-prompt'
      aria-label={t('AI 助手教程')}
      className='bg-muted/50 h-full rounded-xl border p-5 shadow-sm md:p-7'
    >
      <div className='bg-background text-primary flex size-10 items-center justify-center rounded-lg border'>
        <Sparkles className='size-5' aria-hidden='true' />
      </div>

      <p className='text-muted-foreground mt-8 text-xs'>
        {t('一步一步，有人帮你做')}
      </p>
      <h2 className='mt-3 text-2xl font-bold tracking-tight'>
        {t('把教程交给豆包。')}
      </h2>
      <p className='text-muted-foreground mt-4 text-sm leading-7'>
        {t(
          '告诉它你用什么工具、做到哪一步。它会根据教程，帮你看下一步该点哪里。'
        )}
      </p>

      <div className='bg-background mt-6 rounded-lg border p-4'>
        <p className='text-muted-foreground text-xs font-medium'>
          {t('「复制给豆包」的原文，可滚动查看')}
        </p>
        <pre className='hover-scrollbar mt-4 max-h-48 overflow-auto font-sans text-xs leading-6 break-words whitespace-pre-wrap md:text-[13px]'>
          {prompt}
        </pre>
      </div>

      <div className='mt-5 flex flex-col gap-3 sm:flex-row'>
        <CopyButton
          value={prompt}
          tooltip={t('复制给豆包')}
          successTooltip={t('已复制！')}
          aria-label={t('复制给豆包')}
          onCopy={() => setCopyStatus('prompt')}
          variant='default'
          size='lg'
          className='h-10 flex-1 rounded-lg'
        >
          {t('复制给豆包')}
          <ExternalLink data-icon='inline-end' />
        </CopyButton>
        <CopyButton
          value={fullGuide}
          tooltip={t('复制完整教程')}
          successTooltip={t('已复制！')}
          aria-label={t('复制完整教程')}
          onCopy={() => setCopyStatus('full-guide')}
          variant='outline'
          size='lg'
          className='bg-background h-10 flex-1 rounded-lg'
        >
          {t('复制完整教程')}
        </CopyButton>
      </div>

      <p className='text-muted-foreground mt-4 text-xs leading-5'>
        {t('豆包打不开链接时，使用「复制完整教程」。')}
      </p>
      {copyStatus === 'prompt' && (
        <p className='text-primary mt-2 text-xs leading-5' role='status'>
          {t('已复制。打开豆包，粘贴后发送即可。')}
        </p>
      )}
      {copyStatus === 'full-guide' && (
        <p className='text-primary mt-2 text-xs leading-5' role='status'>
          {t('完整教程已复制。直接粘贴给豆包即可，无需它读取链接。')}
        </p>
      )}

      <Separator className='my-6' />

      <p className='text-muted-foreground text-sm'>
        {t('也可以交给其他能阅读文字或网页的 AI 助手。')}
      </p>
    </aside>
  )
}

export function DocsGuide() {
  const { t } = useTranslation()
  const [selectedToolId, setSelectedToolId] = useState<ToolId>('claude-code')
  const [selectedSystemId, setSelectedSystemId] = useState<SystemId>('macos')
  const selectedTool = useMemo(
    () =>
      TOOL_OPTIONS.find((option) => option.id === selectedToolId) ??
      TOOL_OPTIONS[0],
    [selectedToolId]
  )
  const selectedSystem = useMemo(
    () =>
      SYSTEM_OPTIONS.find((option) => option.id === selectedSystemId) ??
      SYSTEM_OPTIONS[0],
    [selectedSystemId]
  )

  return (
    <div className='bg-background text-foreground min-h-svh overflow-x-clip'>
      <DocsTopBar />
      <main className='mx-auto max-w-6xl px-4 py-10 md:px-8 md:py-14'>
        <section>
          <div>
            <div className='text-muted-foreground flex items-center gap-2 text-sm'>
              <span className='bg-primary size-1.5 rounded-full' />
              <span>{t('HTH API 配置手册 · 简易模式')}</span>
            </div>
            <h1 className='mt-7 text-5xl leading-tight font-bold tracking-tight text-balance md:text-6xl'>
              {t('先选工具，')}
              <br />
              <span className='text-primary'>{t('再跟着做。')}</span>
            </h1>
            <p className='text-muted-foreground mt-6 max-w-lg text-base leading-8'>
              {t(
                '不用先看懂所有参数。找到你的工具和系统，从第一步开始，也可以把教程交给豆包。'
              )}
            </p>
          </div>
        </section>

        <section className='mt-10 grid items-stretch gap-6 lg:grid-cols-[minmax(0,1.55fr)_minmax(22rem,1fr)]'>
          <div className='h-full'>
            <section
              aria-label={t('配置选择')}
              className='bg-card flex h-full flex-col rounded-xl border p-5 shadow-sm md:p-6'
            >
              <div className='flex items-center gap-2'>
                <Badge variant='secondary' className='text-primary'>
                  01
                </Badge>
                <h2 className='text-base font-semibold'>
                  {t('选择你使用的工具')}
                </h2>
              </div>

              <div
                className='mt-4 grid gap-3 md:grid-cols-2'
                role='radiogroup'
                aria-label={t('选择你使用的工具')}
              >
                {TOOL_OPTIONS.map((option) => (
                  <ToolButton
                    key={option.id}
                    option={option}
                    selected={option.id === selectedTool.id}
                    onSelect={setSelectedToolId}
                  />
                ))}
              </div>

              <div className='mt-7 flex items-center gap-2'>
                <Badge variant='secondary' className='text-primary'>
                  02
                </Badge>
                <h2 className='text-base font-semibold'>
                  {t('选择你的电脑系统')}
                </h2>
              </div>

              <div
                className='mt-4 grid gap-3 sm:grid-cols-2'
                role='radiogroup'
                aria-label={t('选择你的电脑系统')}
              >
                {SYSTEM_OPTIONS.map((option) => (
                  <SystemButton
                    key={option.id}
                    option={option}
                    selected={option.id === selectedSystem.id}
                    onSelect={setSelectedSystemId}
                  />
                ))}
              </div>

              <p className='text-muted-foreground mt-3 text-sm'>
                {t(
                  '不知道选哪个？Windows 是常见 PC 系统，苹果电脑通常选 macOS。'
                )}
              </p>

              <div className='mt-auto'>
                <Separator className='my-7' />

                <div className='flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between'>
                  <div className='min-w-0'>
                    <p className='text-muted-foreground text-xs'>
                      {t('你的教程')}
                    </p>
                    <p className='mt-1 truncate text-sm font-semibold'>
                      {t('{{tool}} · {{system}} 配置教程', {
                        tool: selectedTool.name,
                        system: selectedSystem.name,
                      })}
                    </p>
                  </div>
                  <Button
                    className='h-10 rounded-lg sm:min-w-32'
                    render={
                      <a
                        href={getTutorialPath(
                          selectedTool.id,
                          selectedSystem.id
                        )}
                      />
                    }
                  >
                    {t('打开教程')}
                    <ArrowRight data-icon='inline-end' />
                  </Button>
                </div>
              </div>
            </section>
          </div>

          <AssistantPanel
            selectedTool={selectedTool}
            selectedSystem={selectedSystem}
          />
        </section>
      </main>
    </div>
  )
}
