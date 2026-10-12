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
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterContextProvider,
} from '@tanstack/react-router'
import {
  act,
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { I18nextProvider } from 'react-i18next'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ThemeProvider } from '@/context/theme-provider'

import {
  buildGenerateImageInstallPrompt,
  GENERATE_IMAGE_RELEASE,
  type GenerateImageRelease,
} from '../generate-image-content'
import { GenerateImageDocsPage } from '../generate-image-page'

vi.mock('@/hooks/use-system-config', () => ({
  useSystemConfig: () => ({ systemName: 'HTH Gateway', logo: '/logo.png' }),
}))

const clipboardDescriptor = Object.getOwnPropertyDescriptor(
  navigator,
  'clipboard'
)
const execCommandDescriptor = Object.getOwnPropertyDescriptor(
  document,
  'execCommand'
)

beforeEach(async () => {
  localStorage.clear()
  await i18next.changeLanguage('en')
})

afterEach(async () => {
  cleanup()
  if (clipboardDescriptor) {
    Object.defineProperty(navigator, 'clipboard', clipboardDescriptor)
  } else {
    Reflect.deleteProperty(navigator, 'clipboard')
  }
  if (execCommandDescriptor) {
    Object.defineProperty(document, 'execCommand', execCommandDescriptor)
  } else {
    Reflect.deleteProperty(document, 'execCommand')
  }
  localStorage.clear()
  await i18next.changeLanguage('en')
})

function renderGenerateImagePage(release?: GenerateImageRelease) {
  const router = createRouter({
    routeTree: createRootRoute(),
    history: createMemoryHistory({ initialEntries: ['/docs/generate-image'] }),
  })
  return render(<GenerateImageDocsPage release={release} />, {
    wrapper: (props) => (
      <I18nextProvider i18n={i18next}>
        <ThemeProvider defaultTheme='light'>
          <RouterContextProvider router={router}>
            {props.children}
          </RouterContextProvider>
        </ThemeProvider>
      </I18nextProvider>
    ),
  })
}

describe('generate-image installation prompt', () => {
  it('includes release v1.0 and seven installation steps without a checksum or credential setup instructions', () => {
    const prompt = buildGenerateImageInstallPrompt(GENERATE_IMAGE_RELEASE)

    expect(prompt).toContain(
      'https://hqdtc.oss-us-west-1.aliyuncs.com/hth-skills/generate-image/generate-image.zip'
    )
    expect(GENERATE_IMAGE_RELEASE.version).toBe('1.0')
    expect(prompt).toContain('发布版本：1.0')
    expect(prompt).not.toContain('SHA-256')
    expect(prompt).not.toContain('安装期间不读取或修改我的 Codex 登录凭据')
    expect(prompt).not.toContain('HTH_API_KEY')
    expect(prompt).toContain('Windows')
    expect(prompt).toContain('macOS')
    expect(prompt).toContain('%USERPROFILE%\\.agents\\skills')
    expect(prompt).toContain('~/.agents/skills')
    expect(prompt).not.toMatch(/Linux|WSL/i)
    expect(prompt.match(/^\d+\./gm)).toEqual([
      '1.',
      '2.',
      '3.',
      '4.',
      '5.',
      '6.',
      '7.',
    ])
    for (const requirement of [
      '不要把同名 Skill 同时安装到多个目录',
      '备份必须放在 Codex 不扫描的目录中',
      '保留已有 .env 等本地配置',
      '下载检查失败时停止安装',
      '不允许压缩包中的文件写到目标目录之外',
      '--help、--list-models 和一个不带远程输入的 --dry-run',
      '不调用生图接口，不生成图片',
      '不要声称已经完成未执行的操作',
    ]) {
      expect(prompt).toContain(requirement)
    }
  })

  it('preserves complete URL query values without escaping or template placeholders', () => {
    const downloadUrl =
      'https://example.com/generate-image.zip?Expires=1791654485&Signature=b%2FEI3q%3D'
    const prompt = buildGenerateImageInstallPrompt({
      ...GENERATE_IMAGE_RELEASE,
      downloadUrl,
    })

    expect(prompt).toContain(`安装包下载地址：\n${downloadUrl}\n`)
    expect(prompt).not.toContain('&amp;')
    expect(prompt).not.toContain('\\&')
    expect(prompt).not.toMatch(/\$\{|\{\{|\}\}/)
  })
})

describe('GenerateImageDocsPage', () => {
  it('copies exactly the visible complete installation prompt through the shared button', async () => {
    const user = userEvent.setup()
    const writeText = vi
      .spyOn(navigator.clipboard, 'writeText')
      .mockResolvedValue(undefined)
    renderGenerateImagePage()
    const preview = screen.getByRole('region', { name: '安装提示词' })
    const prompt = buildGenerateImageInstallPrompt(GENERATE_IMAGE_RELEASE)

    expect(preview.textContent).toBe(prompt)
    await user.click(screen.getByRole('button', { name: '复制到 Codex' }))

    await waitFor(() => expect(writeText).toHaveBeenCalledWith(prompt))
    expect(await screen.findByRole('status')).toHaveTextContent(/已复制.*Codex/)
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('keeps the permanent package download as a safe direct anchor', () => {
    renderGenerateImagePage()
    const download = screen.getByRole('link', { name: '下载安装包' })

    expect(download.tagName).toBe('A')
    expect(download).toHaveAttribute(
      'href',
      'https://hqdtc.oss-us-west-1.aliyuncs.com/hth-skills/generate-image/generate-image.zip'
    )
    expect(download).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('links to desktop Codex tutorials and the retained image examples', () => {
    renderGenerateImagePage()
    const tutorials = within(
      screen.getByRole('navigation', { name: 'Codex 配置教程' })
    )

    expect(
      tutorials.getByRole('link', { name: '桌面端 · Windows' })
    ).toHaveAttribute('href', '/docs/tutorials/codex-desktop/windows')
    expect(
      tutorials.getByRole('link', { name: '桌面端 · macOS' })
    ).toHaveAttribute('href', '/docs/tutorials/codex-desktop/macos')
    expect(
      tutorials.queryByRole('link', { name: 'CLI · Windows' })
    ).not.toBeInTheDocument()
    expect(
      tutorials.queryByRole('link', { name: 'CLI · macOS' })
    ).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: '查看生图示例' })).toHaveAttribute(
      'href',
      '#generate-image-examples'
    )
    expect(screen.getByRole('heading', { name: '试一个示例' })).toHaveAttribute(
      'id',
      'generate-image-examples'
    )
  })

  it('offers selectable manual recovery when both browser copy methods fail', async () => {
    const user = userEvent.setup()
    vi.spyOn(navigator.clipboard, 'writeText').mockRejectedValue(
      new Error('Clipboard permission denied')
    )
    Object.defineProperty(document, 'execCommand', {
      configurable: true,
      value: vi.fn().mockReturnValue(false),
    })
    renderGenerateImagePage()

    await user.click(screen.getByRole('button', { name: '复制到 Codex' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      '复制失败。请选中完整提示词，手动复制到 Codex。'
    )
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    expect(
      screen.getByRole('region', { name: '安装提示词' })
    ).toHaveTextContent(GENERATE_IMAGE_RELEASE.downloadUrl)
  })

  it('places the installation actions before a keyboard-accessible scrolling preview', async () => {
    const user = userEvent.setup()
    renderGenerateImagePage()
    const copy = screen.getByRole('button', { name: '复制到 Codex' })
    const preview = screen.getByRole('region', { name: '安装提示词' })

    expect(
      copy.compareDocumentPosition(preview) & Node.DOCUMENT_POSITION_FOLLOWING
    ).not.toBe(0)
    expect(preview).toHaveAttribute('tabindex', '0')
    expect(preview).toHaveClass('whitespace-pre-wrap')
    expect(preview.className).toMatch(/\boverflow-(y-)?auto\b/)
    expect(preview.className).toMatch(/break-words|break-all|overflow-wrap/)
    act(() => copy.focus())
    await user.tab()
    expect(screen.getByRole('link', { name: '下载安装包' })).toHaveFocus()
    await user.tab()
    expect(preview).toHaveFocus()
  })

  it('keeps the installation content Chinese when the global interface uses English', () => {
    renderGenerateImagePage()

    expect(i18next.language).toBe('en')
    expect(screen.getByText('generate-image v1.0')).toBeVisible()
    expect(screen.getByRole('main')).not.toHaveTextContent('SHA-256')
    expect(screen.getByRole('main')).not.toHaveTextContent(/Linux|WSL/i)
    expect(screen.getByRole('button', { name: '复制到 Codex' })).toBeVisible()
    expect(
      screen.getByRole('region', { name: '安装提示词' })
    ).toHaveTextContent(
      '请帮我在这台电脑上安装 generate-image（HTH 生图 Skill）'
    )
    expect(screen.getByRole('heading', { name: '文生图' })).toBeVisible()
    expect(
      screen.queryByRole('heading', { name: '安装好了，接下来怎么用？' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('heading', { name: '常见问题' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: '需要自己下载并解压吗？' })
    ).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: '复制精细改图示例' })
    ).toBeVisible()
  })

  it('keeps installation available in browsers without URL.canParse', () => {
    const descriptor = Object.getOwnPropertyDescriptor(URL, 'canParse')
    Object.defineProperty(URL, 'canParse', {
      configurable: true,
      value: undefined,
    })

    try {
      renderGenerateImagePage()

      expect(screen.getByRole('button', { name: '复制到 Codex' })).toBeVisible()
      expect(screen.getByRole('link', { name: '下载安装包' })).toHaveAttribute(
        'href',
        GENERATE_IMAGE_RELEASE.downloadUrl
      )
    } finally {
      if (descriptor) {
        Object.defineProperty(URL, 'canParse', descriptor)
      } else {
        Reflect.deleteProperty(URL, 'canParse')
      }
    }
  })

  it.each([
    {
      label: 'missing download URL',
      release: { ...GENERATE_IMAGE_RELEASE, downloadUrl: '' },
    },
    {
      label: 'malformed download URL',
      release: { ...GENERATE_IMAGE_RELEASE, downloadUrl: 'https://' },
    },
    {
      label: 'HTTP download URL',
      release: {
        ...GENERATE_IMAGE_RELEASE,
        downloadUrl: 'http://example.com/generate-image.zip',
      },
    },
    {
      label: 'JavaScript download URL',
      release: {
        ...GENERATE_IMAGE_RELEASE,
        downloadUrl: 'javascript:alert(1)',
      },
    },
    {
      label: 'blank version',
      release: { ...GENERATE_IMAGE_RELEASE, version: ' ' },
    },
  ])('withholds installation actions for $label', (fixture) => {
    renderGenerateImagePage(fixture.release)

    expect(screen.getByText('安装入口暂不可用。')).toBeVisible()
    expect(
      screen.queryByRole('button', { name: '复制到 Codex' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('link', { name: '下载安装包' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('region', { name: '安装提示词' })
    ).not.toBeInTheDocument()
  })

  it.each([
    [
      '复制文生图示例',
      'gpt-image-2.5-flare',
      '使用 $generate-image 的 gpt-image-2.5-flare 帮我生成一张白色陶瓷杯的产品照片。杯子放在浅色桌面上，柔和自然光，背景简洁，适合电商展示。',
    ],
    [
      '复制精细改图示例',
      'gpt-image-2.5-sunburst',
      '使用 $generate-image 的 gpt-image-2.5-sunburst  编辑我提供的图片：把杯子的颜色改成深蓝色，保留杯子的形状、光照和背景。',
    ],
    [
      '复制多参考图合成示例',
      'doubao-seedream-5.0-pro',
      '使用 $generate-image 的 doubao-seedream-5.0-pro 将我提供的产品图和背景图合成一张宣传图片。保持产品外观和比例，统一光照与透视，让产品自然融入背景。',
    ],
    [
      '复制Gemini 场景图示例',
      'gemini-3.1-flash-image',
      '使用 $generate-image 的 gemini-3.1-flash-image  生成一张清晨山谷的横向风景图，有薄雾和柔和阳光，画面比例 16:9。',
    ],
  ])(
    'copies the complete visible sample when %s is activated',
    async (name, model, expectedPrompt) => {
      const user = userEvent.setup()
      const writeText = vi
        .spyOn(navigator.clipboard, 'writeText')
        .mockResolvedValue(undefined)
      renderGenerateImagePage()
      const sample = screen.getByText(
        (content) =>
          content.startsWith('使用 $generate-image') && content.includes(model)
      )
      expect(sample.textContent).toBe(expectedPrompt)

      await user.click(screen.getByRole('button', { name }))

      await waitFor(() =>
        expect(writeText).toHaveBeenCalledWith(expectedPrompt)
      )
    }
  )

  it('clears previous copy feedback and copies the new prompt after the release changes', async () => {
    const user = userEvent.setup()
    const writeText = vi
      .spyOn(navigator.clipboard, 'writeText')
      .mockResolvedValue(undefined)
    const view = renderGenerateImagePage()
    await user.click(screen.getByRole('button', { name: '复制到 Codex' }))
    expect(await screen.findByRole('status')).toHaveTextContent(/已复制/)
    const updatedRelease = {
      ...GENERATE_IMAGE_RELEASE,
      version: '1.1',
      downloadUrl: 'https://example.com/generate-image-1.1.zip',
    }

    view.rerender(<GenerateImageDocsPage release={updatedRelease} />)

    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: '下载安装包' })).toHaveAttribute(
      'href',
      updatedRelease.downloadUrl
    )
    await user.click(screen.getByRole('button', { name: '复制到 Codex' }))
    await waitFor(() =>
      expect(writeText).toHaveBeenLastCalledWith(
        buildGenerateImageInstallPrompt(updatedRelease)
      )
    )
  })
})
