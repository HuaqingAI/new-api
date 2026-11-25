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
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createInstance } from 'i18next'
import { I18nextProvider } from 'react-i18next'
import { toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import en from '@/i18n/locales/en.json'
import zh from '@/i18n/locales/zh.json'
import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import type { ClientNotification } from '../../notification-api'
import { ClientNotificationSettings } from '../ClientNotificationSettings'

const endpoint = '/api/agent-platform/notifications'
const originalAuth = useAuthStore.getState().auth
const item: ClientNotification = {
  id: 'ntf_notice',
  title: 'Service update',
  summary: 'Please read',
  content: 'First line\nSecond line',
  enabled: false,
  version: 3,
  published_at: 0,
  created_at: 1795525200,
  updated_at: 1795529400,
  created_by: 1,
  updated_by: 1,
}
let client: QueryClient

beforeEach(() => {
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  useAuthStore.setState({
    auth: {
      ...originalAuth,
      user: { id: 1, username: 'root', role: ROLE.SUPER_ADMIN },
    },
  })
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data:
        url === endpoint
          ? { items: [item], total: 1, page: 1, page_size: 20 }
          : item,
    },
  }))
  vi.spyOn(toast, 'success').mockImplementation(() => 'success')
  vi.spyOn(toast, 'error').mockImplementation(() => 'error')
})
afterEach(() => {
  client.clear()
  useAuthStore.setState({ auth: originalAuth })
})

function renderSettings() {
  render(
    <QueryClientProvider client={client}>
      <ClientNotificationSettings />
    </QueryClientProvider>
  )
}
async function openSettings() {
  const user = userEvent.setup()
  await user.click(screen.getByRole('button', { name: 'Client notifications' }))
  await screen.findByText('Service update')
  return user
}

describe('client notification management', () => {
  it('disables editing and dismissal while a save is pending', async () => {
    let complete: (() => void) | undefined
    const post = vi.spyOn(api, 'post').mockImplementation(
      () =>
        new Promise((resolve) => {
          complete = () => resolve({ data: { success: true, data: item } })
        })
    )
    renderSettings()
    const user = await openSettings()
    await user.click(screen.getByRole('button', { name: 'Add notification' }))
    await user.type(
      screen.getByRole('textbox', { name: 'Title' }),
      'Maintenance'
    )
    await user.type(screen.getByRole('textbox', { name: 'Content' }), 'Details')
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
    expect(screen.getByRole('button', { name: 'Saving...' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled()
    expect(screen.getByRole('textbox', { name: 'Title' })).toBeDisabled()
    expect(screen.getByRole('switch', { name: 'Status' })).toHaveAttribute(
      'aria-disabled',
      'true'
    )
    await user.keyboard('{Escape}')
    expect(
      screen.getByRole('dialog', { name: 'Add notification' })
    ).toBeVisible()
    await act(async () => complete?.())
    expect(
      await screen.findByRole('button', { name: 'Add notification' })
    ).toBeVisible()
  })

  it('updates visible labels when the interface language changes', async () => {
    const i18n = createInstance()
    await i18n.init({
      lng: 'en',
      resources: { en, zh },
      keySeparator: false,
      nsSeparator: false,
    })
    render(
      <I18nextProvider i18n={i18n}>
        <QueryClientProvider client={client}>
          <ClientNotificationSettings />
        </QueryClientProvider>
      </I18nextProvider>
    )
    const user = await openSettings()
    await act(async () => {
      await i18n.changeLanguage('zh')
    })
    expect(screen.getByRole('dialog', { name: '消息通知' })).toBeVisible()
    await user.click(screen.getByRole('button', { name: '新增消息' }))
    expect(screen.getByRole('textbox', { name: '标题' })).toBeVisible()
    expect(
      screen.getByRole('switch', { name: zh.translation.Status })
    ).toBeChecked()
  })

  it('returns to the preceding page when deleting the last item on a page', async () => {
    let deleted = false
    vi.mocked(api.get).mockImplementation(async (_url, config) => {
      const params = config?.params as { page: number }
      return {
        data: {
          success: true,
          data: {
            items: deleted && params.page === 2 ? [] : [item],
            total: deleted ? 20 : 21,
            page: params.page,
            page_size: 20,
          },
        },
      }
    })
    vi.spyOn(api, 'delete').mockImplementation(async () => {
      deleted = true
      return { data: { success: true, data: { id: item.id, deleted: true } } }
    })
    renderSettings()
    const user = await openSettings()
    await user.click(screen.getByRole('button', { name: 'Go to next page' }))
    await waitFor(() =>
      expect(api.get).toHaveBeenCalledWith(endpoint, {
        params: { page: 2, page_size: 20, status: 'all' },
      })
    )
    await user.click(await screen.findByRole('button', { name: 'Delete' }))
    await user.click(
      within(await screen.findByRole('alertdialog')).getByRole('button', {
        name: 'Confirm',
      })
    )
    await waitFor(() =>
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    )
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Go to previous page' })
      ).toBeDisabled()
    )
    expect(screen.getByText('Service update')).toBeVisible()
  })

  it('hides global management and makes no requests for an ordinary administrator', () => {
    useAuthStore.setState({
      auth: {
        ...originalAuth,
        user: { id: 2, username: 'admin', role: ROLE.ADMIN },
      },
    })
    renderSettings()
    expect(
      screen.queryByRole('button', { name: 'Client notifications' })
    ).not.toBeInTheDocument()
    expect(api.get).not.toHaveBeenCalled()
  })

  it('loads only after opening, keeps pagination inside the dialog, and filters from the first page', async () => {
    renderSettings()
    expect(api.get).not.toHaveBeenCalled()
    const user = await openSettings()
    const dialog = screen.getByRole('dialog')
    expect(
      within(dialog).getByRole('button', { name: 'Go to next page' })
    ).toBeDisabled()
    expect(within(dialog).getByText('Unpublished')).toBeVisible()
    fireEvent.change(
      screen.getByRole('combobox', { name: 'Notification status' }),
      { target: { value: 'enabled' } }
    )
    await waitFor(() =>
      expect(api.get).toHaveBeenCalledWith(endpoint, {
        params: { page: 1, page_size: 20, status: 'enabled' },
      })
    )
    await user.keyboard('{Escape}')
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    )
    expect(
      screen.getByRole('button', { name: 'Client notifications' })
    ).toHaveFocus()
  })

  it.each([true, false])(
    'creates a plain text notification with enabled=%s from an enabled default',
    async (enabled) => {
      const post = vi.spyOn(api, 'post').mockResolvedValue({
        data: { success: true, data: { ...item, enabled } },
      })
      renderSettings()
      const user = await openSettings()
      await user.click(screen.getByRole('button', { name: 'Add notification' }))
      const status = screen.getByRole('switch', { name: 'Status' })
      expect(status).toBeChecked()
      if (!enabled) await user.click(status)
      await user.type(
        screen.getByRole('textbox', { name: 'Title' }),
        '  Scheduled work  '
      )
      await user.type(
        screen.getByRole('textbox', { name: 'Content' }),
        '<script>text</script>\nSecond line'
      )
      await user.click(screen.getByRole('button', { name: 'Save' }))
      await waitFor(() =>
        expect(post).toHaveBeenCalledWith(endpoint, {
          title: 'Scheduled work',
          summary: '',
          content: '<script>text</script>\nSecond line',
          enabled,
        })
      )
      expect(
        await screen.findByRole('button', { name: 'Add notification' })
      ).toBeVisible()
      expect(toast.success).toHaveBeenCalledWith('Client notification saved')
    }
  )

  it('rejects an empty form, connects field errors, and confirms discarding a dirty draft', async () => {
    const post = vi.spyOn(api, 'post')
    renderSettings()
    const user = await openSettings()
    await user.click(screen.getByRole('button', { name: 'Add notification' }))
    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(
      await screen.findByText('Notification title is required')
    ).toBeVisible()
    expect(screen.getByRole('textbox', { name: 'Title' })).toHaveAttribute(
      'aria-invalid',
      'true'
    )
    expect(post).not.toHaveBeenCalled()
    await user.type(screen.getByRole('textbox', { name: 'Title' }), 'Draft')
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    const confirm = await screen.findByRole('alertdialog')
    await user.click(within(confirm).getByRole('button', { name: 'Cancel' }))
    expect(screen.getByRole('textbox', { name: 'Title' })).toHaveValue('Draft')
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    await user.click(
      within(await screen.findByRole('alertdialog')).getByRole('button', {
        name: 'Discard',
      })
    )
    expect(
      await screen.findByRole('button', { name: 'Add notification' })
    ).toBeVisible()
  })

  it('keeps the edited text after a business failure and sends the current If-Match version', async () => {
    const put = vi.spyOn(api, 'put').mockResolvedValue({
      data: { success: false, message: 'Storage unavailable' },
    })
    renderSettings()
    const user = await openSettings()
    await user.click(screen.getByRole('button', { name: 'Edit' }))
    const title = await screen.findByRole('textbox', { name: 'Title' })
    expect(screen.getByRole('switch', { name: 'Status' })).not.toBeChecked()
    await user.clear(title)
    await user.type(title, 'Changed title')
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(
        `${endpoint}/${item.id}`,
        {
          title: 'Changed title',
          summary: item.summary,
          content: item.content,
          enabled: false,
        },
        { headers: { 'If-Match': '"3"' } }
      )
    )
    await waitFor(() => expect(toast.error).toHaveBeenCalled())
    expect(title).toHaveValue('Changed title')
    expect(screen.getByRole('button', { name: 'Save' })).toBeEnabled()
    expect(toast.success).not.toHaveBeenCalled()
  })

  it('preserves a conflicting edit until explicit reload and then uses the refreshed version', async () => {
    const put = vi
      .spyOn(api, 'put')
      .mockRejectedValueOnce({
        response: {
          status: 412,
          data: { success: false, message: 'Conflict' },
        },
      })
      .mockResolvedValue({
        data: { success: true, data: { ...item, version: 5 } },
      })
    renderSettings()
    const user = await openSettings()
    await user.click(screen.getByRole('button', { name: 'Edit' }))
    const title = await screen.findByRole('textbox', { name: 'Title' })
    await user.type(title, ' local')
    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(
      await screen.findByText('Reloading discards your unsaved changes.')
    ).toBeVisible()
    expect(title).toHaveValue('Service update local')
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
    vi.mocked(api.get).mockResolvedValueOnce({
      data: {
        success: true,
        data: { ...item, title: 'Remote edit', version: 4 },
      },
    })
    await user.click(screen.getByRole('button', { name: 'Reload' }))
    await waitFor(() => expect(title).toHaveValue('Remote edit'))
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(put).toHaveBeenLastCalledWith(
        `${endpoint}/${item.id}`,
        expect.objectContaining({ title: 'Remote edit' }),
        { headers: { 'If-Match': '"4"' } }
      )
    )
  })

  it.each([false, true])(
    'toggles the status switch directly from enabled=%s without confirmation or action buttons',
    async (enabled) => {
      let stored = { ...item, enabled }
      vi.mocked(api.get).mockImplementation(async () => ({
        data: {
          success: true,
          data: { items: [stored], total: 1, page: 1, page_size: 20 },
        },
      }))
      const patch = vi.spyOn(api, 'patch').mockImplementation(async () => {
        stored = { ...stored, enabled: !enabled, version: 4 }
        return { data: { success: true, data: stored } }
      })
      renderSettings()
      const user = await openSettings()
      const toggle = screen.getByRole('switch', { name: 'Notification status' })
      expect(toggle).toHaveAttribute('aria-checked', String(enabled))
      expect(
        screen.queryByRole('button', { name: 'Enable' })
      ).not.toBeInTheDocument()
      expect(
        screen.queryByRole('button', { name: 'Disable' })
      ).not.toBeInTheDocument()
      await user.click(toggle)
      await waitFor(() =>
        expect(patch).toHaveBeenCalledWith(
          `${endpoint}/${item.id}/status`,
          { enabled: !enabled },
          { headers: { 'If-Match': '"3"' } }
        )
      )
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
      await waitFor(() =>
        expect(
          screen.getByRole('switch', { name: 'Notification status' })
        ).toHaveAttribute('aria-checked', String(!enabled))
      )
    }
  )

  it('keeps the server status and reports an error when a switch update fails', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        success: true,
        data: {
          items: [{ ...item, enabled: true }],
          total: 1,
          page: 1,
          page_size: 20,
        },
      },
    })
    const patch = vi.spyOn(api, 'patch').mockResolvedValue({
      data: { success: false, message: 'Status update unavailable' },
    })
    renderSettings()
    const user = await openSettings()
    await user.click(
      screen.getByRole('switch', { name: 'Notification status' })
    )
    await waitFor(() =>
      expect(patch).toHaveBeenCalledWith(
        `${endpoint}/${item.id}/status`,
        { enabled: false },
        { headers: { 'If-Match': '"3"' } }
      )
    )
    await waitFor(() => expect(toast.error).toHaveBeenCalledTimes(1))
    expect(
      screen.getByRole('switch', { name: 'Notification status' })
    ).toBeChecked()
    expect(
      screen.getByRole('switch', { name: 'Notification status' })
    ).not.toHaveAttribute('aria-disabled', 'true')
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(toast.success).not.toHaveBeenCalled()
  })

  it('prevents duplicate switch updates until saving and the refreshed server status finish', async () => {
    let firstRead = true
    let completePatch: (() => void) | undefined
    let completeRead: (() => void) | undefined
    const updated = { ...item, enabled: true, version: 4 }
    vi.mocked(api.get).mockImplementation(async () => {
      if (firstRead) {
        firstRead = false
        return {
          data: {
            success: true,
            data: { items: [item], total: 1, page: 1, page_size: 20 },
          },
        }
      }
      return new Promise((resolve) => {
        completeRead = () =>
          resolve({
            data: {
              success: true,
              data: { items: [updated], total: 1, page: 1, page_size: 20 },
            },
          })
      })
    })
    const patch = vi.spyOn(api, 'patch').mockImplementation(
      () =>
        new Promise((resolve) => {
          completePatch = () =>
            resolve({ data: { success: true, data: updated } })
        })
    )
    renderSettings()
    const user = await openSettings()
    await user.click(
      screen.getByRole('switch', { name: 'Notification status' })
    )
    await waitFor(() => expect(patch).toHaveBeenCalledTimes(1))
    expect(
      screen.getByRole('switch', { name: 'Notification status' })
    ).toHaveAttribute('aria-disabled', 'true')
    expect(
      screen.getByRole('switch', { name: 'Notification status' })
    ).not.toBeChecked()
    await user.click(
      screen.getByRole('switch', { name: 'Notification status' })
    )
    expect(patch).toHaveBeenCalledTimes(1)
    await act(async () => completePatch?.())
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(2))
    expect(
      screen.getByRole('switch', { name: 'Notification status' })
    ).toHaveAttribute('aria-disabled', 'true')
    await user.click(
      screen.getByRole('switch', { name: 'Notification status' })
    )
    expect(patch).toHaveBeenCalledTimes(1)
    await act(async () => completeRead?.())
    await waitFor(() =>
      expect(
        screen.getByRole('switch', { name: 'Notification status' })
      ).toBeChecked()
    )
    await waitFor(() =>
      expect(
        screen.getByRole('switch', { name: 'Notification status' })
      ).not.toHaveAttribute('aria-disabled', 'true')
    )
  })

  it('requires confirmation for deletion and retains the confirmation after failure', async () => {
    const remove = vi
      .spyOn(api, 'delete')
      .mockRejectedValueOnce(new Error('Network unavailable'))
      .mockResolvedValue({
        data: { success: true, data: { id: item.id, deleted: true } },
      })
    renderSettings()
    const user = await openSettings()
    await user.click(screen.getByRole('button', { name: 'Delete' }))
    expect(remove).not.toHaveBeenCalled()
    await user.click(
      within(await screen.findByRole('alertdialog')).getByRole('button', {
        name: 'Confirm',
      })
    )
    await waitFor(() => expect(toast.error).toHaveBeenCalled())
    expect(screen.getByRole('alertdialog')).toBeVisible()
    await user.click(
      within(screen.getByRole('alertdialog')).getByRole('button', {
        name: 'Confirm',
      })
    )
    await waitFor(() =>
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    )
    expect(remove).toHaveBeenLastCalledWith(`${endpoint}/${item.id}`, {
      headers: { 'If-Match': '"3"' },
    })
  })

  it('shows a failed read as an error and retries instead of showing an empty success', async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { success: false, message: 'Feed unavailable' },
    })
    renderSettings()
    const user = userEvent.setup()
    await user.click(
      screen.getByRole('button', { name: 'Client notifications' })
    )
    expect(await screen.findByText('Feed unavailable')).toBeVisible()
    expect(
      screen.queryByText('No client notifications')
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByText('Service update')).toBeVisible()
  })
})
