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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { CopyButton } from '@/components/copy-button'

const clipboardDescriptor = Object.getOwnPropertyDescriptor(
  navigator,
  'clipboard'
)
const execCommandDescriptor = Object.getOwnPropertyDescriptor(
  document,
  'execCommand'
)

afterEach(() => {
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
})

describe('CopyButton', () => {
  it('copies the complete value with keyboard activation and only reports success', async () => {
    const user = userEvent.setup()
    const writeText = vi
      .spyOn(navigator.clipboard, 'writeText')
      .mockResolvedValue(undefined)
    const onCopy = vi.fn()
    const onCopyError = vi.fn()
    const value = 'Install the skill\nhttps://example.com/skill.zip?a=1&b=%2F'
    render(
      <CopyButton
        value={value}
        aria-label='Copy installation prompt'
        onCopy={onCopy}
        onCopyError={onCopyError}
      />
    )

    await user.tab()
    expect(
      screen.getByRole('button', { name: 'Copy installation prompt' })
    ).toHaveFocus()
    await user.keyboard('{Enter}')

    expect(await screen.findByRole('button', { name: 'Copied' })).toHaveFocus()
    expect(writeText).toHaveBeenCalledWith(value)
    expect(onCopy).toHaveBeenCalledOnce()
    expect(onCopyError).not.toHaveBeenCalled()

    await user.keyboard('{Enter}')
    await waitFor(() => expect(onCopy).toHaveBeenCalledTimes(2))
    expect(onCopyError).not.toHaveBeenCalled()
  })

  it('reports success and restores button focus when browser fallback succeeds', async () => {
    const user = userEvent.setup()
    vi.spyOn(navigator.clipboard, 'writeText').mockRejectedValue(
      new Error('Clipboard permission denied')
    )
    const execCommand = vi.fn().mockReturnValue(true)
    Object.defineProperty(document, 'execCommand', {
      configurable: true,
      value: execCommand,
    })
    const onCopy = vi.fn()
    const onCopyError = vi.fn()
    render(
      <CopyButton
        value='Installation prompt'
        aria-label='Copy installation prompt'
        onCopy={onCopy}
        onCopyError={onCopyError}
      />
    )

    await user.click(
      screen.getByRole('button', { name: 'Copy installation prompt' })
    )

    expect(await screen.findByRole('button', { name: 'Copied' })).toHaveFocus()
    expect(execCommand).toHaveBeenCalledWith('copy')
    expect(onCopy).toHaveBeenCalledOnce()
    expect(onCopyError).not.toHaveBeenCalled()
    expect(document.querySelector('textarea')).not.toBeInTheDocument()
  })

  it.each(['returns false', 'throws'])(
    'restores button focus when fallback %s and allows a successful retry',
    async (fallbackBehavior) => {
      const user = userEvent.setup()
      const writeText = vi
        .spyOn(navigator.clipboard, 'writeText')
        .mockRejectedValueOnce(new Error('Clipboard permission denied'))
        .mockResolvedValue(undefined)
      Object.defineProperty(document, 'execCommand', {
        configurable: true,
        value: vi.fn(() => {
          if (fallbackBehavior === 'throws') {
            throw new Error('Browser copy command failed')
          }
          return false
        }),
      })
      const onCopy = vi.fn()
      const onCopyError = vi.fn()
      render(
        <CopyButton
          value='Installation prompt'
          aria-label='Copy installation prompt'
          onCopy={onCopy}
          onCopyError={onCopyError}
        />
      )

      await user.click(
        screen.getByRole('button', { name: 'Copy installation prompt' })
      )

      await waitFor(() => expect(onCopyError).toHaveBeenCalledOnce())
      expect(onCopy).not.toHaveBeenCalled()
      expect(
        screen.getByRole('button', { name: 'Copy installation prompt' })
      ).toHaveFocus()
      expect(document.querySelector('textarea')).not.toBeInTheDocument()
      expect(
        screen.queryByRole('button', { name: 'Copied' })
      ).not.toBeInTheDocument()

      await user.click(
        screen.getByRole('button', { name: 'Copy installation prompt' })
      )

      expect(
        await screen.findByRole('button', { name: 'Copied' })
      ).toBeVisible()
      expect(writeText).toHaveBeenLastCalledWith('Installation prompt')
      expect(onCopy).toHaveBeenCalledOnce()
      expect(onCopyError).toHaveBeenCalledOnce()
    }
  )

  it('preserves the existing retry behavior when no failure callback is supplied', async () => {
    const user = userEvent.setup()
    vi.spyOn(navigator.clipboard, 'writeText')
      .mockRejectedValueOnce(new Error('Clipboard permission denied'))
      .mockResolvedValue(undefined)
    Object.defineProperty(document, 'execCommand', {
      configurable: true,
      value: vi.fn().mockReturnValue(false),
    })
    const onCopy = vi.fn()
    render(
      <CopyButton
        value='Installation prompt'
        aria-label='Copy installation prompt'
        onCopy={onCopy}
      />
    )

    await user.click(
      screen.getByRole('button', { name: 'Copy installation prompt' })
    )
    expect(onCopy).not.toHaveBeenCalled()
    await user.click(
      screen.getByRole('button', { name: 'Copy installation prompt' })
    )

    expect(await screen.findByRole('button', { name: 'Copied' })).toBeVisible()
    expect(onCopy).toHaveBeenCalledOnce()
  })
})
