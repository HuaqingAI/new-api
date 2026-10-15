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
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { EnterpriseDingTalk } from '../index'
import type { DingTalkConfig } from '../types'

let client: QueryClient

async function renderSettings() {
  const config: DingTalkConfig = {
    id: 1,
    tenant_id: 0,
    corp_id: 'test-corp',
    app_key: 'test-app',
    callback_url: 'https://gateway.example.com/oauth/dingtalk',
    sync_scope: '',
    login_enabled: false,
    sync_enabled: false,
    auto_sync_on_login: false,
    has_app_secret: true,
    created_at: 0,
    updated_at: 0,
  }
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/enterprise/dingtalk/config') {
      return { data: { success: true, data: config } }
    }
    if (
      url === '/api/enterprise/dingtalk/sync/logs' ||
      url === '/api/enterprise/dingtalk/sync/conflicts'
    ) {
      return { data: { success: true, data: { items: [], total: 0 } } }
    }
    throw new Error(`Unexpected GET ${url}`)
  })
  const put = vi.spyOn(api, 'put').mockResolvedValue({
    data: { success: true, data: config },
  })
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'root',
    role: ROLE.SUPER_ADMIN,
  })
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const route = createRootRoute({ component: EnterpriseDingTalk })
  const router = createRouter({
    routeTree: route,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  const callback = await screen.findByRole('textbox', { name: 'Callback URL' })
  await waitFor(() => expect(callback).toHaveValue(config.callback_url))
  return { callback, put }
}

afterEach(() => {
  client?.clear()
  useAuthStore.getState().auth.reset()
})

describe('DingTalk callback URL validation', () => {
  it.each([
    'http://127.0.0.1:3001/oauth/dingtalk',
    'http://gateway.example.com/oauth/dingtalk',
    'https://gateway.example.com/oauth/dingtalk',
  ])('saves callback %s when enabling DingTalk login', async (callbackUrl) => {
    const { callback, put } = await renderSettings()
    const user = userEvent.setup()
    await user.clear(callback)
    await user.type(callback, callbackUrl)
    await user.click(
      screen.getByRole('switch', { name: 'Enable DingTalk login' })
    )
    await user.click(screen.getByRole('button', { name: /^Save$/ }))

    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(
        '/api/enterprise/dingtalk/config',
        expect.objectContaining({
          callback_url: callbackUrl,
          login_enabled: true,
        })
      )
    )
    expect(callback).not.toHaveAttribute('aria-invalid', 'true')
  })

  it.each([
    ['invalid URL', 'not a URL'],
    ['unsupported protocol', 'ftp://gateway.example.com/oauth/dingtalk'],
    ['relative URL', '/oauth/dingtalk'],
    ['empty URL', ''],
  ])('rejects an %s when enabling DingTalk login', async (_, callbackUrl) => {
    const { callback, put } = await renderSettings()
    const user = userEvent.setup()
    await user.clear(callback)
    if (callbackUrl) await user.type(callback, callbackUrl)
    await user.click(
      screen.getByRole('switch', { name: 'Enable DingTalk login' })
    )
    await user.click(screen.getByRole('button', { name: /^Save$/ }))

    const message = callbackUrl
      ? 'Enter a valid callback URL'
      : 'Callback URL is required when enabling DingTalk'
    expect(await screen.findByText(message)).toBeVisible()
    expect(callback).toHaveAttribute('aria-invalid', 'true')
    expect(put).not.toHaveBeenCalled()
  })
})
