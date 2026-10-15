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
import { Download, Sparkles } from 'lucide-react'
import { useState } from 'react'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'

type GenerateImageInstallCardProps = {
  prompt: string
  downloadUrl: string
  available: boolean
}

export function GenerateImageInstallCard(props: GenerateImageInstallCardProps) {
  const [copyStatus, setCopyStatus] = useState<'success' | 'error' | null>(null)

  return (
    <aside
      aria-label='安装 generate-image'
      className='bg-muted/50 min-w-0 rounded-xl border p-5 shadow-sm md:p-7 lg:order-2'
    >
      <div className='bg-background text-primary mb-6 hidden size-10 items-center justify-center rounded-lg border md:flex'>
        <Sparkles className='size-5' aria-hidden='true' />
      </div>
      <h2 className='text-2xl font-bold tracking-tight'>把安装交给 Codex。</h2>
      <p className='text-muted-foreground mt-3 text-sm leading-6'>
        复制下面的提示词，粘贴到你电脑上的 Codex。它会下载、安装并检查 Skill。
      </p>

      {props.available ? (
        <>
          <div className='mt-5 flex flex-col gap-3 sm:flex-row'>
            <CopyButton
              value={props.prompt}
              tooltip='复制到 Codex'
              successTooltip='已复制！'
              aria-label='复制到 Codex'
              onCopy={() => setCopyStatus('success')}
              onCopyError={() => setCopyStatus('error')}
              variant='default'
              size='lg'
              className='h-11 flex-1 rounded-lg'
            >
              复制到 Codex
            </CopyButton>
            <Button
              role='link'
              variant='outline'
              size='lg'
              className='bg-background h-11 flex-1 rounded-lg'
              render={
                <a
                  href={props.downloadUrl}
                  target='_blank'
                  rel='noopener noreferrer'
                />
              }
            >
              <Download aria-hidden='true' data-icon='inline-start' />
              下载安装包
            </Button>
          </div>
          <p className='text-muted-foreground mt-3 text-xs leading-5'>
            复制后不会自动打开 Codex，请切换到 Codex 粘贴并发送。
          </p>
          {copyStatus === 'success' && (
            <p className='text-primary mt-2 text-sm leading-6' role='status'>
              已复制。打开 Codex，粘贴后发送即可。
            </p>
          )}
          {copyStatus === 'error' && (
            <p className='text-destructive mt-2 text-sm leading-6' role='alert'>
              复制失败。请选中完整提示词，手动复制到 Codex。
            </p>
          )}
          <div className='bg-background mt-5 min-w-0 rounded-lg border p-4'>
            <p
              id='generate-image-prompt-label'
              className='text-muted-foreground text-xs font-medium'
            >
              将复制的完整提示词，可滚动查看
            </p>
            <pre
              role='region'
              aria-label='安装提示词'
              aria-describedby='generate-image-prompt-label'
              tabIndex={0}
              className='hover-scrollbar focus-visible:ring-ring mt-3 max-h-40 overflow-auto rounded-sm font-sans text-xs leading-6 break-words whitespace-pre-wrap outline-none focus-visible:ring-2 md:max-h-72 md:text-[13px]'
            >
              {props.prompt}
            </pre>
          </div>
        </>
      ) : (
        <p className='text-destructive mt-5 text-sm' role='alert'>
          安装入口暂不可用。
        </p>
      )}
    </aside>
  )
}
