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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { I18nextProvider } from 'react-i18next'
import { describe, expect, it, vi } from 'vitest'

import i18n from '@/i18n/config'

import { getClientInstallations } from '../api'
import { ClientInstallations } from '../components/client-installations'

vi.mock('../api', () => ({
  getClientInstallations: vi.fn(),
}))

const clientInstallationsResponse = {
  success: true,
  data: {
    items: [
      {
        id: 1,
        client_version: '1.2.3',
        platform: 'windows_x64' as const,
        lan_ip: '192.168.1.20',
        user_id: 7,
        username: 'alice',
        email: 'alice@example.com',
        display_name: 'Alice Example',
        departments: ['Engineering'],
        first_heartbeat_at: '2026-09-11T01:00:00Z',
        last_heartbeat_at: '2026-09-11T01:30:00Z',
        is_active: true,
      },
    ],
    total: 1,
    page: 1,
    page_size: 20,
    summary: {
      active_users: 1,
      active_installations: 1,
      total_users: 1,
      total_installations: 1,
      active_since: '2026-09-11T00:00:00Z',
    },
  },
}

function renderClientInstallations() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <ClientInstallations />
      </I18nextProvider>
    </QueryClientProvider>
  )
}

describe('ClientInstallations', () => {
  it('shows the four company-wide statistics and current client record', async () => {
    vi.mocked(getClientInstallations).mockResolvedValue(
      clientInstallationsResponse
    )
    renderClientInstallations()

    expect(await screen.findByText('Alice Example')).toBeInTheDocument()
    expect(screen.getByText('Active installations')).toBeInTheDocument()
    expect(screen.getByText('Total installations')).toBeInTheDocument()
    expect(screen.getByText('Client-reported LAN IP')).toBeInTheDocument()
    expect(screen.getByText('192.168.1.20')).toBeInTheDocument()
  })

  it('uses user, platform, and version filters in the installation query', async () => {
    vi.mocked(getClientInstallations).mockResolvedValue(
      clientInstallationsResponse
    )
    const user = userEvent.setup()
    renderClientInstallations()
    await screen.findByText('Alice Example')

    await user.type(
      screen.getByRole('textbox', { name: 'Search by user' }),
      'alice'
    )
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Platform' }),
      'mac_arm64'
    )
    await user.type(screen.getByRole('textbox', { name: 'Version' }), '1.2.3')
    await user.click(screen.getByRole('button', { name: 'Search' }))

    await waitFor(() => {
      expect(getClientInstallations).toHaveBeenLastCalledWith({
        page: 1,
        pageSize: 20,
        keyword: 'alice',
        platform: 'mac_arm64',
        version: '1.2.3',
      })
    })
  })

  it('offers the existing error state when the query fails', async () => {
    vi.mocked(getClientInstallations).mockRejectedValue(new Error('failed'))
    renderClientInstallations()

    expect(
      await screen.findByText('Oops! Something went wrong')
    ).toBeInTheDocument()
  })
})
