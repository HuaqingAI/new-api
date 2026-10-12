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
import { useState } from 'react'

import { CopyButton } from '@/components/copy-button'
import { Badge } from '@/components/ui/badge'

import {
  GENERATE_IMAGE_EXAMPLES,
  GENERATE_IMAGE_MODELS,
} from '../generate-image-content'

export function GenerateImageUsageGuide() {
  const [failedExample, setFailedExample] = useState<string | null>(null)

  return (
    <div className='mt-12 space-y-12 md:mt-16'>
      <section aria-labelledby='generate-image-examples'>
        <h2 id='generate-image-examples' className='text-2xl font-bold'>
          试一个示例
        </h2>
        <p className='text-muted-foreground mt-3 text-sm leading-7'>
          复制后粘贴到 Codex。发送这些任务会调用 HTH
          生图服务，请先完成凭据配置。
        </p>
        <div className='mt-5 grid gap-4 md:grid-cols-2'>
          {GENERATE_IMAGE_EXAMPLES.map((example) => (
            <article
              key={example.id}
              className='bg-card min-w-0 rounded-xl border p-5'
            >
              <div className='flex items-center justify-between gap-3'>
                <h3 className='font-semibold'>{example.title}</h3>
                <CopyButton
                  value={example.prompt}
                  tooltip={`复制${example.title}示例`}
                  successTooltip='已复制！'
                  aria-label={`复制${example.title}示例`}
                  onCopy={() => setFailedExample(null)}
                  onCopyError={() => setFailedExample(example.id)}
                  variant='outline'
                  size='sm'
                >
                  复制示例
                </CopyButton>
              </div>
              <p className='text-muted-foreground mt-3 text-sm leading-6'>
                {example.description}
              </p>
              <pre className='mt-4 font-sans text-sm leading-7 break-words whitespace-pre-wrap'>
                {example.prompt}
              </pre>
              {failedExample === example.id && (
                <p className='text-destructive mt-3 text-sm' role='alert'>
                  复制失败。请选中这条示例，手动复制到 Codex。
                </p>
              )}
            </article>
          ))}
        </div>
      </section>

      <section aria-labelledby='generate-image-models'>
        <h2 id='generate-image-models' className='text-2xl font-bold'>
          支持的模型
        </h2>
        <p className='text-muted-foreground mt-3 text-sm leading-7'>
          模型标识来自当前发布包，实际可用性以 HTH 服务配置和账户权限为准。
          默认使用 gpt-image-2.5，具体上游映射由 HTH 决定。
        </p>
        <div className='mt-5 grid gap-4 lg:grid-cols-3'>
          {GENERATE_IMAGE_MODELS.map((provider) => (
            <article
              key={provider.name}
              className='bg-card min-w-0 rounded-xl border p-5'
            >
              <h3 className='font-semibold'>{provider.name}</h3>
              <p className='text-muted-foreground mt-3 text-sm leading-6'>
                {provider.description}
              </p>
              <ul className='mt-4 space-y-2'>
                {provider.models.map((model) => (
                  <li key={model}>
                    <Badge
                      variant='secondary'
                      className='h-auto max-w-full rounded-md py-1 font-mono break-all whitespace-normal'
                    >
                      {model}
                    </Badge>
                  </li>
                ))}
              </ul>
            </article>
          ))}
        </div>
      </section>
    </div>
  )
}
