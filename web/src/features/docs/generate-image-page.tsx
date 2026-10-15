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
import { ArrowLeft, ArrowRight } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'

import { DocsTopBar } from './components/docs-top-bar'
import { GenerateImageInstallCard } from './components/generate-image-install-card'
import { GenerateImageUsageGuide } from './components/generate-image-usage-guide'
import {
  buildGenerateImageInstallPrompt,
  GENERATE_IMAGE_RELEASE,
  type GenerateImageRelease,
} from './generate-image-content'

type GenerateImageDocsPageProps = {
  release?: GenerateImageRelease
}

export function GenerateImageDocsPage(props: GenerateImageDocsPageProps) {
  const release = props.release ?? GENERATE_IMAGE_RELEASE
  let available = false
  try {
    available =
      new URL(release.downloadUrl).protocol === 'https:' &&
      release.version.trim().length > 0
  } catch {
    available = false
  }
  const prompt = available ? buildGenerateImageInstallPrompt(release) : ''

  return (
    <div className='bg-background text-foreground min-h-svh'>
      <DocsTopBar />
      <main
        lang='zh-CN'
        className='mx-auto max-w-6xl px-4 py-5 md:px-8 md:py-12'
      >
        <Button
          role='link'
          variant='ghost'
          className='mb-4 h-9'
          render={<Link to='/docs' />}
        >
          <ArrowLeft aria-hidden='true' data-icon='inline-start' />
          返回配置手册
        </Button>
        <div className='flex flex-wrap items-center gap-2'>
          <Badge variant='secondary' className='text-primary'>
            HTH 生图 Skill
          </Badge>
          {available && (
            <span className='text-muted-foreground text-xs'>
              generate-image v{release.version}
            </span>
          )}
        </div>
        <h1 className='mt-4 text-3xl leading-tight font-bold tracking-tight text-balance md:text-5xl'>
          让 Codex 帮你<span className='text-primary'>生成和编辑图片。</span>
        </h1>
        <p className='text-muted-foreground mt-4 max-w-2xl text-sm leading-7 md:text-base'>
          安装 generate-image，把你的想法变成图片，也能编辑已有图片。
        </p>

        <div className='mt-6 grid items-start gap-6 lg:grid-cols-[minmax(0,1.55fr)_minmax(22rem,1fr)]'>
          <GenerateImageInstallCard
            key={prompt}
            prompt={prompt}
            downloadUrl={release.downloadUrl}
            available={available}
          />
          <section
            aria-labelledby='generate-image-prepare'
            className='bg-card min-w-0 rounded-xl border p-5 shadow-sm md:p-6 lg:order-1'
          >
            <h2 id='generate-image-prepare' className='text-xl font-bold'>
              安装前准备
            </h2>
            <dl className='mt-5 space-y-4 text-sm'>
              <div>
                <dt className='font-semibold'>本机 Codex</dt>
                <dd className='text-muted-foreground mt-1 leading-6'>
                  在 Windows 或 macOS 上使用能够执行本机文件与命令操作的 Codex
                  会话。
                </dd>
              </div>
              <div>
                <dt className='font-semibold'>{release.pythonRequirement}</dt>
                <dd className='text-muted-foreground mt-1 leading-6'>
                  Codex 会检查可用版本；当前包仅使用 Python 标准库。
                </dd>
              </div>
              <div>
                <dt className='font-semibold'>网络与 HTH 凭据</dt>
                <dd className='text-muted-foreground mt-1 leading-6'>
                  安装包可直接下载。实际生图需要能访问 HTH 服务的网络、有效的
                  API Key 和模型权限。
                </dd>
              </div>
            </dl>
            <Separator className='my-6' />
            <h3 className='font-semibold'>三步开始</h3>
            <ol className='mt-4 space-y-4 text-sm'>
              <li className='flex gap-3'>
                <Badge variant='secondary'>01</Badge>
                <div>
                  <p className='font-medium'>复制安装提示词</p>
                  <p className='text-muted-foreground mt-1 leading-6'>
                    点击“复制到 Codex”，一次复制完整安装要求。
                  </p>
                </div>
              </li>
              <li className='flex gap-3'>
                <Badge variant='secondary'>02</Badge>
                <div>
                  <p className='font-medium'>粘贴到本机 Codex</p>
                  <p className='text-muted-foreground mt-1 leading-6'>
                    粘贴并发送，让 Codex 下载、安装并执行离线检查。
                  </p>
                </div>
              </li>
              <li className='flex gap-3'>
                <Badge variant='secondary'>03</Badge>
                <div>
                  <p className='font-medium'>开始使用</p>
                  <p className='text-muted-foreground mt-1 leading-6'>
                    查看检查结果，配置 HTH 凭据，再发送生图或改图任务。
                  </p>
                </div>
              </li>
            </ol>
            <Separator className='my-6' />
            <p className='text-sm font-medium'>还没有配置 Codex？</p>
            <nav
              aria-label='Codex 配置教程'
              className='mt-3 flex flex-wrap gap-2'
            >
              <Button
                role='link'
                variant='outline'
                size='sm'
                render={
                  <Link
                    to='/docs/tutorials/$toolId/$systemId'
                    params={{ toolId: 'codex-desktop', systemId: 'windows' }}
                  />
                }
              >
                桌面端 · Windows
              </Button>
              <Button
                role='link'
                variant='outline'
                size='sm'
                render={
                  <Link
                    to='/docs/tutorials/$toolId/$systemId'
                    params={{ toolId: 'codex-desktop', systemId: 'macos' }}
                  />
                }
              >
                桌面端 · macOS
              </Button>
            </nav>
            <Button
              role='link'
              variant='link'
              className='mt-3 px-0'
              render={<a href='#generate-image-examples' />}
            >
              查看生图示例
              <ArrowRight aria-hidden='true' data-icon='inline-end' />
            </Button>
          </section>
        </div>
        <GenerateImageUsageGuide />
      </main>
    </div>
  )
}
