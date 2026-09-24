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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { I18nextProvider } from 'react-i18next'
import { describe, expect, it, vi } from 'vitest'

import i18n from '@/i18n/config'

import { DocsGuide } from '..'
import { DocsTutorialPage } from '../tutorial-page'
import { TUTORIAL_DOCS } from '../tutorials'

vi.mock('@/components/theme-switch', () => ({
  ThemeSwitch: () => <button type='button'>切换主题</button>,
}))

vi.mock('@/hooks/use-system-config', () => ({
  useSystemConfig: () => ({
    systemName: 'Aion Gateway',
    logo: '/logo.png',
  }),
}))

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (name: string) => <span aria-hidden='true'>{name}</span>,
}))

function renderDocsGuide() {
  return render(
    <I18nextProvider i18n={i18n}>
      <DocsGuide />
    </I18nextProvider>
  )
}

function renderTutorialPage(toolId: string, systemId: string) {
  return render(
    <I18nextProvider i18n={i18n}>
      <DocsTutorialPage toolId={toolId} systemId={systemId} />
    </I18nextProvider>
  )
}

function mockClipboardWrite() {
  const writeText = vi.fn().mockResolvedValue(undefined)
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: { writeText },
  })
  return writeText
}

describe('DocsGuide', () => {
  it('creates one tutorial document for each tool and system pair', () => {
    const docKeys = TUTORIAL_DOCS.map(
      (doc) => `${doc.toolId}-${doc.systemId}`
    ).sort()

    expect(docKeys).toEqual([
      'claude-code-macos',
      'claude-code-windows',
      'codex-cli-macos',
      'codex-cli-windows',
      'codex-desktop-macos',
      'codex-desktop-windows',
      'openclaw-macos',
      'openclaw-windows',
    ])
    expect(TUTORIAL_DOCS.every((doc) => doc.markdown.includes('HTH'))).toBe(
      true
    )
    expect(
      TUTORIAL_DOCS.every((doc) => doc.markdown.includes('hthbuddy'))
    ).toBe(true)
    expect(
      TUTORIAL_DOCS.every(
        (doc) => !doc.markdown.includes(['米', '醋'].join(''))
      )
    ).toBe(true)
    expect(
      TUTORIAL_DOCS.every(
        (doc) =>
          !doc.markdown.includes('配置资料核对日期') &&
          !doc.markdown.includes('看不到完整模型列表') &&
          !doc.markdown.includes('模型限制') &&
          !doc.markdown.includes('优选域名通常无需开启代理') &&
          !doc.markdown.includes('CC Switch 使用前，先确认这两项') &&
          !doc.markdown.includes('关闭路由模式') &&
          !doc.markdown.includes('401 Unauthorized') &&
          !doc.markdown.includes('CC Switch 不可用时：手动配置') &&
          !doc.markdown.includes('这是备用方式') &&
          !doc.markdown.includes('配好了，接下来')
      )
    ).toBe(true)
    expect(
      TUTORIAL_DOCS.filter((doc) => doc.toolId !== 'openclaw').every(
        (doc) =>
          doc.markdown.includes(
            '[CC Switch 官网](https://ccswitch.io/zh/download)'
          ) &&
          !doc.markdown.includes('CC Switch 发布页') &&
          !doc.markdown.includes(
            'https://github.com/farion1231/cc-switch/releases'
          )
      )
    ).toBe(true)
  })

  it('renders the Chinese simple-mode guide with the configured brand', () => {
    renderDocsGuide()

    expect(screen.getByText('Aion Gateway')).toBeInTheDocument()
    expect(screen.getByText('先选工具，')).toBeInTheDocument()
    expect(screen.getByText('再跟着做。')).toBeInTheDocument()
    expect(screen.getByText('HTH API 配置手册 · 简易模式')).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: '简易模式' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: '详细模式' })
    ).not.toBeInTheDocument()
    expect(screen.queryByLabelText('搜索文档')).not.toBeInTheDocument()
    expect(screen.queryByText('习惯看视频？')).not.toBeInTheDocument()
    expect(screen.queryByText('Linux')).not.toBeInTheDocument()
  })

  it('updates the visible guide title and assistant prompt after selection changes', async () => {
    const user = userEvent.setup()
    renderDocsGuide()

    expect(screen.getByRole('button', { name: /打开教程/ })).toHaveAttribute(
      'href',
      '/docs/tutorials/claude-code/macos'
    )

    await user.click(screen.getByRole('radio', { name: /Codex CLI/ }))
    await user.click(screen.getByRole('radio', { name: 'Windows' }))

    expect(screen.getByText('Codex CLI · Windows 配置教程')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /打开教程/ })).toHaveAttribute(
      'href',
      '/docs/tutorials/codex-cli/windows'
    )
    expect(
      screen.getByText(/正在 Windows 上配置 Codex CLI，使用HTH API/)
    ).toBeInTheDocument()
    const tutorialUrl = `${window.location.origin}/docs/tutorials/codex-cli/windows`
    const promptPreview = screen.getByText((content, element) => {
      return (
        element?.tagName.toLowerCase() === 'pre' &&
        content.includes(tutorialUrl)
      )
    })
    expect(promptPreview.textContent).toContain(`教程：\n${tutorialUrl}`)
    expect(promptPreview.textContent).not.toContain(`[${tutorialUrl}]`)
    expect(promptPreview.textContent).not.toContain(
      ['hth', 'huaqing', 'run'].join('.')
    )
    expect(screen.queryByText('纯文字教程')).not.toBeInTheDocument()
  })

  it('shows a different copy confirmation for each guide action', async () => {
    const user = userEvent.setup()
    const writeText = mockClipboardWrite()
    renderDocsGuide()

    await user.click(screen.getByRole('button', { name: '复制给豆包' }))
    expect(
      await screen.findByText('已复制。打开豆包，粘贴后发送即可。')
    ).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '复制完整教程' }))
    expect(
      await screen.findByText(
        '完整教程已复制。直接粘贴给豆包即可，无需它读取链接。'
      )
    ).toBeInTheDocument()
    expect(writeText).toHaveBeenLastCalledWith(
      expect.stringContaining(`${window.location.origin}`)
    )
    expect(writeText).toHaveBeenLastCalledWith(
      expect.stringContaining('hthbuddy')
    )
    const promptPreview = screen.getByText((content, element) => {
      return (
        element?.tagName.toLowerCase() === 'pre' &&
        content.includes('/docs/tutorials/claude-code/macos')
      )
    })
    const prompt = promptPreview.textContent ?? ''
    const copiedFullGuide = writeText.mock.calls.at(-1)?.[0] ?? ''
    expect(prompt).not.toBe('')
    expect(copiedFullGuide.startsWith(`${prompt}\n\n`)).toBe(true)
    expect(
      screen.queryByText('已复制。打开豆包，粘贴后发送即可。')
    ).not.toBeInTheDocument()
  })

  it('renders the Codex desktop official app entry as a link', () => {
    renderTutorialPage('codex-desktop', 'windows')

    expect(
      screen.getByRole('link', { name: 'Codex 官方应用入口' })
    ).toHaveAttribute('href', 'https://developers.openai.com/codex/app/')
  })

  it('renders a filled tutorial document page with the current origin', () => {
    renderTutorialPage('openclaw', 'macos')

    expect(
      screen.getByRole('heading', { name: 'OpenClaw · macOS 配置教程' })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: /填入 HTH 连接配置/ })
    ).toBeInTheDocument()
    const outline = screen.getByRole('navigation', { name: '本页目录' })
    expect(outline).toHaveClass('sticky')
    expect(outline.parentElement).toHaveClass('self-stretch')
    expect(outline).toHaveTextContent('这份教程会帮你完成什么')
    expect(outline).toHaveTextContent('4. 填入 HTH 连接配置')
    expect(
      screen.getByRole('link', { name: '4. 填入 HTH 连接配置' })
    ).toHaveAttribute('href', '#tutorial-heading-5')
    expect(
      screen.getByRole('heading', { name: /填入 HTH 连接配置/ })
    ).toHaveAttribute('id', 'tutorial-heading-5')
    expect(document.body).toHaveTextContent('hthbuddy')
    expect(document.body).toHaveTextContent(`${window.location.origin}/v1`)
  })
})
