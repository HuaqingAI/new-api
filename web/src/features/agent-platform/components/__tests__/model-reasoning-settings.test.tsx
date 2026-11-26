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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { modelReasoningFormSchema } from '../../lib/model-reasoning-form'
import type {
  ModelReasoningPolicy,
  ModelReasoningRule,
} from '../../model-reasoning-api'
import { ModelReasoningSettings } from '../model-reasoning-settings'

const originalAuth = useAuthStore.getState().auth
const endpoint = '/api/agent-platform/model-reasoning-policy'
const config: ModelReasoningPolicy = {
  schema_version: 1,
  scope: 'global',
  revision: 'sha256:test',
  default_efforts: ['low', 'medium', 'high'],
  rules: [],
}
let client: QueryClient

beforeEach(() => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  useAuthStore.setState({
    auth: {
      ...originalAuth,
      user: { id: 1, username: 'root', role: ROLE.SUPER_ADMIN },
    },
  })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: config },
  })
  vi.spyOn(toast, 'success').mockImplementation(() => 'success')
  vi.spyOn(toast, 'error').mockImplementation(() => 'error')
})

afterEach(() => {
  cleanup()
  client.clear()
  useAuthStore.setState({ auth: originalAuth })
})

function renderSettings() {
  return render(
    <QueryClientProvider client={client}>
      <ModelReasoningSettings />
    </QueryClientProvider>
  )
}

async function openSettings() {
  const user = userEvent.setup()
  await user.click(
    screen.getByRole('button', { name: 'Model reasoning settings' })
  )
  await screen.findByRole('button', { name: 'Add reasoning rule' })
  return user
}

describe('model reasoning settings', () => {
  it('keeps the model explanation in one description and omits the runtime timing note', async () => {
    renderSettings()
    await openSettings()
    expect(screen.getByRole('dialog')).toHaveAccessibleDescription(
      'Configure the available model reasoning efforts in Huaqing AI Assistant, with exact model rules taking priority over family rules and unmatched models using low, medium, high by default.'
    )
    expect(
      screen.queryByText(
        'Changes apply when AionUi next prepares or restarts an OpenCode or Codex runtime.'
      )
    ).not.toBeInTheDocument()
  })

  it('hides settings and makes no configuration request for ordinary administrators', () => {
    useAuthStore.setState({
      auth: {
        ...originalAuth,
        user: { id: 2, username: 'admin', role: ROLE.ADMIN },
      },
    })
    renderSettings()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
    expect(api.get).not.toHaveBeenCalled()
  })

  it('saves a family rule and an exact model rule with multiple reasoning choices', async () => {
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true, data: config } })
    renderSettings()
    expect(api.get).not.toHaveBeenCalled()
    const user = await openSettings()
    await user.click(screen.getByRole('button', { name: 'Add reasoning rule' }))
    const family = within(screen.getByRole('group', { name: 'Rule 1' }))
    await user.selectOptions(
      family.getByRole('combobox', { name: 'Model family' }),
      'gpt'
    )
    await user.click(
      family.getByRole('combobox', { name: 'Select reasoning efforts' })
    )
    expect(await screen.findByRole('option', { name: 'xhigh' })).toBeVisible()
    await user.click(await screen.findByRole('option', { name: 'none' }))
    await user.keyboard('{Escape}')
    await user.click(screen.getByRole('button', { name: 'Add reasoning rule' }))
    const exact = within(screen.getByRole('group', { name: 'Rule 2' }))
    await user.selectOptions(
      exact.getByRole('combobox', { name: 'Rule type' }),
      'model'
    )
    await user.type(
      exact.getByRole('textbox', { name: 'Exact model ID' }),
      'gpt-5.6-terra'
    )
    await user.click(
      exact.getByRole('combobox', { name: 'Select reasoning efforts' })
    )
    await user.click(await screen.findByRole('option', { name: 'max' }))
    await user.keyboard('{Escape}')
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(endpoint, {
        rules: [
          {
            match_type: 'family',
            match_value: 'gpt',
            efforts: ['low', 'medium', 'high', 'none'],
          },
          {
            match_type: 'model',
            match_value: 'gpt-5.6-terra',
            efforts: ['low', 'medium', 'high', 'max'],
          },
        ],
      })
    )
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    )
    expect(toast.success).toHaveBeenCalledWith('Model reasoning settings saved')
  })

  it('keeps unsaved input after a business save failure', async () => {
    vi.spyOn(api, 'put').mockResolvedValue({
      data: { success: false, message: 'storage unavailable' },
    })
    renderSettings()
    const user = await openSettings()
    await user.click(screen.getByRole('button', { name: 'Add reasoning rule' }))
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Model family' }),
      'deepseek'
    )
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(toast.error).toHaveBeenCalledTimes(1))
    expect(screen.getByRole('combobox', { name: 'Model family' })).toHaveValue(
      'deepseek'
    )
    expect(screen.getByRole('button', { name: 'Save' })).toBeEnabled()
    expect(toast.success).not.toHaveBeenCalled()
  })

  it('connects a model validation error to its input and prevents saving', async () => {
    const put = vi.spyOn(api, 'put')
    renderSettings()
    const user = await openSettings()
    await user.click(screen.getByRole('button', { name: 'Add reasoning rule' }))
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Rule type' }),
      'model'
    )
    const input = screen.getByRole('textbox', { name: 'Exact model ID' })
    await user.click(input)
    await user.paste('x'.repeat(256))
    await user.click(screen.getByRole('button', { name: 'Save' }))
    const error = await screen.findByText(
      'Model IDs must be at most 255 characters'
    )
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input.getAttribute('aria-describedby')).toContain(error.id)
    await waitFor(() => expect(input).toHaveFocus())
    expect(put).not.toHaveBeenCalled()
  })

  it('requires confirmation to discard changes and restores the saved configuration', async () => {
    renderSettings()
    const user = await openSettings()
    await user.click(screen.getByRole('button', { name: 'Add reasoning rule' }))
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    const confirm = screen.getByRole('alertdialog', {
      name: 'Discard unsaved changes?',
    })
    await user.click(within(confirm).getByRole('button', { name: 'Discard' }))
    const trigger = screen.getByRole('button', {
      name: 'Model reasoning settings',
    })
    await waitFor(() => expect(trigger).toHaveFocus())
    await user.click(trigger)
    expect(await screen.findByText('No reasoning rules')).toBeVisible()
    expect(
      screen.queryByRole('group', { name: 'Rule 1' })
    ).not.toBeInTheDocument()
  })

  it('does not expose an editable empty configuration after a failed read and supports retry', async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { success: false, message: 'settings unavailable' },
    })
    renderSettings()
    const user = userEvent.setup()
    await user.click(
      screen.getByRole('button', { name: 'Model reasoning settings' })
    )
    expect(
      await screen.findByText('Failed to load model reasoning settings')
    ).toBeVisible()
    expect(
      screen.queryByRole('button', { name: 'Save' })
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(
      await screen.findByRole('button', { name: 'Add reasoning rule' })
    ).toBeEnabled()
  })

  it('removes only draft rules and persists an explicit empty rule list', async () => {
    const saved = {
      ...config,
      rules: [{ match_type: 'family', match_value: 'gpt', efforts: ['high'] }],
    }
    vi.mocked(api.get).mockResolvedValue({
      data: { success: true, data: saved },
    })
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true, data: config } })
    renderSettings()
    const user = await openSettings()
    await user.click(screen.getByRole('button', { name: 'Remove rule 1' }))
    expect(put).not.toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(endpoint, { rules: [] })
    )
  })
})

describe('model reasoning form validation', () => {
  const rule: ModelReasoningRule = {
    match_type: 'model',
    match_value: 'gpt-test',
    efforts: ['none', 'max'],
  }
  it('accepts all families, case-sensitive exact IDs, Unicode and explicit none', () => {
    for (const match_value of ['deepseek', 'qwen', 'glm', 'gpt']) {
      expect(
        modelReasoningFormSchema.safeParse({
          rules: [{ ...rule, match_type: 'family', match_value }],
        }).success
      ).toBe(true)
    }
    expect(
      modelReasoningFormSchema.safeParse({
        rules: [
          rule,
          { ...rule, match_value: 'GPT-test' },
          { ...rule, match_value: '🧠'.repeat(255) },
        ],
      }).success
    ).toBe(true)
    expect(modelReasoningFormSchema.safeParse({ rules: [] }).success).toBe(true)
  })

  it.each([
    { match_value: '' },
    { match_value: '🧠'.repeat(256) },
    { match_value: 'gpt\u0000test' },
    { match_value: '\tgpt-test\n' },
    { match_value: 'gpt\ud800' },
    { efforts: [] },
    { efforts: ['unknown'] },
    { match_type: 'family', match_value: 'other' },
  ])(
    'rejects missing or invalid model, family and effort fields: %#',
    (change) => {
      expect(
        modelReasoningFormSchema.safeParse({ rules: [{ ...rule, ...change }] })
          .success
      ).toBe(false)
    }
  )

  it('rejects duplicate normalized targets and oversized rule sets', () => {
    expect(
      modelReasoningFormSchema.safeParse({
        rules: [rule, { ...rule, match_value: ' gpt-test ' }],
      }).success
    ).toBe(false)
    expect(
      modelReasoningFormSchema.safeParse({
        rules: [
          { ...rule, match_type: 'family', match_value: 'GPT' },
          { ...rule, match_type: 'family', match_value: 'gpt' },
        ],
      }).success
    ).toBe(false)
    expect(
      modelReasoningFormSchema.safeParse({
        rules: Array.from({ length: 1001 }, (_, index) => ({
          ...rule,
          match_value: String(index),
        })),
      }).success
    ).toBe(false)
    expect(
      modelReasoningFormSchema.safeParse({
        rules: Array.from({ length: 100 }, (_, index) => ({
          ...rule,
          match_value: '🧠'.repeat(250) + index,
        })),
      }).success
    ).toBe(false)
  })
})
