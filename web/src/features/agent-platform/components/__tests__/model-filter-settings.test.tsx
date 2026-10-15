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
import { toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import {
  CLIENT_MODEL_FILTER_QUERY_KEY,
  type ClientModelFilter,
} from '../../api'
import { AgentPlatformShell } from '../../index'
import { clientModelFilterFormSchema } from '../../lib/model-filter-form'
import { ClientModelFilterSettings } from '../ClientModelFilterSettings'

const originalAuth = useAuthStore.getState().auth
const endpoint = '/api/agent-platform/model-filter'
let client: QueryClient
const config: ClientModelFilter = {
  target_token_name: 'hth-default-apikey',
  default_model: 'gpt-5.6-terra',
  excluded_models: ['saved-model'],
}

beforeEach(() => {
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  useAuthStore.setState({
    auth: {
      ...originalAuth,
      user: { id: 1, username: 'root', role: ROLE.SUPER_ADMIN },
    },
  })
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data:
      url === endpoint
        ? { success: true, data: config }
        : { success: true, data: ['gpt-5.6-terra', 'enabled-model'] },
  }))
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
      <ClientModelFilterSettings />
    </QueryClientProvider>
  )
}

async function openSettings() {
  const user = userEvent.setup()
  const trigger = await screen.findByRole('button', {
    name: /^Client model filter/,
  })
  await waitFor(() => expect(trigger).toBeEnabled())
  await user.click(trigger)
  return user
}

describe('client model filter', () => {
  it.each([ROLE.SUPER_ADMIN, ROLE.ADMIN])(
    'shows the resource-header action and requests global settings only for root, role %s',
    async (role) => {
      useAuthStore.setState({
        auth: { ...originalAuth, user: { id: 1, username: 'operator', role } },
      })
      for (const resource of ['mcps', 'skills', 'knowledge', 'agents']) {
        client.setQueryData(['agent-platform', resource, 'summary'], {
          success: true,
          data: { items: [], total: 0, page: 1, page_size: 100 },
        })
      }
      const router = createRouter({
        routeTree: createRootRoute(),
        history: createMemoryHistory(),
      })
      render(
        <RouterContextProvider router={router}>
          <QueryClientProvider client={client}>
            <AgentPlatformShell />
          </QueryClientProvider>
        </RouterContextProvider>
      )
      if (role === ROLE.SUPER_ADMIN) {
        const trigger = await screen.findByRole('button', {
          name: /^Client model filter/,
        })
        await waitFor(() =>
          expect(within(trigger).getByText('1')).toBeVisible()
        )
        const header = trigger.closest('[data-slot="card-header"]')
        expect(header).toHaveTextContent('Resource control plane')
        expect(header).toHaveClass('flex-col', 'sm:flex-row')
        expect(api.get).toHaveBeenCalledWith(endpoint)
      } else {
        expect(
          screen.queryByRole('button', { name: /Client model filter/ })
        ).not.toBeInTheDocument()
        expect(api.get).not.toHaveBeenCalledWith(endpoint)
      }
    }
  )

  it('preserves unknown IDs, adds custom and suggested IDs, and uses the normalized save response for the count', async () => {
    const put = vi.spyOn(api, 'put').mockResolvedValue({
      data: {
        success: true,
        data: {
          ...config,
          excluded_models: ['future-model', 'gpt-5.6-terra', 'saved-model'],
        },
      },
    })
    renderSettings()
    const user = await openSettings()
    const input = screen.getByRole('combobox')
    await user.type(input, 'future-model{Enter}')
    expect(put).not.toHaveBeenCalled()
    await user.click(input)
    await user.click(
      await screen.findByRole('option', { name: 'gpt-5.6-terra' })
    )
    expect(
      screen.getByText(
        'The default model gpt-5.6-terra is filtered. Creating a client session may fail.'
      )
    ).toBeVisible()
    await user.keyboard('{Escape}')
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(endpoint, {
        excluded_models: ['saved-model', 'future-model', 'gpt-5.6-terra'],
      })
    )
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    )
    expect(
      within(
        screen.getByRole('button', { name: /^Client model filter/ })
      ).getByText('3')
    ).toBeVisible()
    expect(client.getQueryData(['enabled-models'])).toEqual({
      success: true,
      data: ['gpt-5.6-terra', 'enabled-model'],
    })
  })

  it('clears only the draft until saving, while cancellation restores the stored exclusions', async () => {
    const put = vi.spyOn(api, 'put').mockResolvedValue({
      data: { success: true, data: { ...config, excluded_models: [] } },
    })
    renderSettings()
    const user = await openSettings()
    await user.click(screen.getByRole('button', { name: 'Clear all' }))
    expect(screen.getByText('0 excluded models')).toBeVisible()
    expect(put).not.toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    await user.click(
      screen.getByRole('button', { name: /Client model filter/ })
    )
    expect(
      within(screen.getByRole('dialog')).getByText('saved-model')
    ).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Clear all' }))
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(endpoint, { excluded_models: [] })
    )
    await waitFor(() =>
      expect(
        within(
          screen.getByRole('button', { name: /^Client model filter/ })
        ).getByText('0')
      ).toBeVisible()
    )
  })

  it('keeps the edited exclusions after a business save failure and reports no success', async () => {
    const put = vi.spyOn(api, 'put').mockResolvedValue({
      data: { success: false, message: 'storage unavailable' },
    })
    renderSettings()
    const user = await openSettings()
    await user.type(screen.getByRole('combobox'), 'future-model{Enter}')
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(toast.error).toHaveBeenCalledTimes(1))
    expect(screen.getByRole('dialog')).toBeVisible()
    expect(screen.getByText('future-model')).toBeVisible()
    expect(toast.success).not.toHaveBeenCalled()
    expect(put).toHaveBeenCalledWith(endpoint, {
      excluded_models: ['saved-model', 'future-model'],
    })
    expect(screen.getByRole('button', { name: 'Save' })).toBeEnabled()
  })

  it('blocks editing unknown configuration after a failed read and allows an explicit retry', async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { success: false, message: 'settings unavailable' },
    })
    const put = vi.spyOn(api, 'put')
    renderSettings()
    const user = await openSettings()
    expect(screen.getByText('Failed to load model filter')).toBeVisible()
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Save' })
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByRole('combobox')).toBeVisible()
    expect(put).not.toHaveBeenCalled()
  })

  it('allows manual IDs when suggestions fail and keeps the draft during a background configuration update', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => ({
      data:
        url === endpoint
          ? { success: true, data: config }
          : { success: false, message: 'suggestions unavailable' },
    }))
    const put = vi.spyOn(api, 'put').mockResolvedValue({
      data: {
        success: true,
        data: { ...config, excluded_models: ['saved-model', 'future-model'] },
      },
    })
    renderSettings()
    const user = await openSettings()
    expect(await screen.findByText('suggestions unavailable')).toBeVisible()
    await user.type(screen.getByRole('combobox'), 'future-model{Enter}')
    await act(async () =>
      client.setQueryData(CLIENT_MODEL_FILTER_QUERY_KEY, {
        ...config,
        excluded_models: ['other-saved-model'],
      })
    )
    expect(screen.getByText('saved-model')).toBeVisible()
    expect(screen.getByText('future-model')).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(endpoint, {
        excluded_models: ['saved-model', 'future-model'],
      })
    )
  })

  it('focuses the invalid model input and connects its field error without saving', async () => {
    const put = vi.spyOn(api, 'put')
    renderSettings()
    const user = await openSettings()
    const input = screen.getByRole('combobox')
    await user.click(input)
    await user.paste('x'.repeat(256))
    await user.keyboard('{Enter}')
    await user.click(screen.getByRole('button', { name: 'Save' }))
    const error = await screen.findByText(
      'Model IDs must be at most 255 characters'
    )
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input.getAttribute('aria-describedby')).toContain(error.id)
    await waitFor(() => expect(input).toHaveFocus())
    expect(put).not.toHaveBeenCalled()
  })
})

describe('client model filter validation', () => {
  it.each([
    { models: [], valid: true },
    { models: ['future-model'], valid: true },
    { models: ['🧠'.repeat(255)], valid: true },
    { models: ['🧠'.repeat(256)], valid: false },
    { models: ['model\u0000name'], valid: false },
    { models: ['\tmodel\n'], valid: false },
    { models: ['model\ud800name'], valid: false },
    { models: Array.from({ length: 1000 }, (_, i) => String(i)), valid: true },
    { models: Array.from({ length: 1001 }, () => 'duplicate'), valid: false },
    {
      models: Array.from(
        { length: 60 },
        (_, i) => '🧠'.repeat(253) + String(i).padStart(2, '0')
      ),
      valid: true,
    },
    {
      models: Array.from(
        { length: 61 },
        (_, i) => '🧠'.repeat(253) + String(i).padStart(2, '0')
      ),
      valid: false,
    },
  ])(
    'accepts valid Unicode IDs and rejects item/count/byte violations, case %#',
    ({ models, valid }) => {
      expect(
        clientModelFilterFormSchema.safeParse({ excluded_models: models })
          .success
      ).toBe(valid)
    }
  )
})
