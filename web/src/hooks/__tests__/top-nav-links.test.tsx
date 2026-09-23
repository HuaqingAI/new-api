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
import { cleanup, renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { I18nextProvider } from 'react-i18next'
import { afterEach, describe, expect, it } from 'vitest'

import i18n from '@/i18n/config'
import { useAuthStore } from '@/stores/auth-store'

import { useTopNavLinks } from '../use-top-nav-links'

afterEach(() => {
  cleanup()
  useAuthStore.getState().auth.reset()
})

function renderTopNavLinks(status: Record<string, unknown>) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  client.setQueryData(['status'], status)

  function Wrapper(props: { children: ReactNode }) {
    return (
      <QueryClientProvider client={client}>
        <I18nextProvider i18n={i18n}>{props.children}</I18nextProvider>
      </QueryClientProvider>
    )
  }

  return renderHook(() => useTopNavLinks(), { wrapper: Wrapper })
}

describe('useTopNavLinks', () => {
  it('uses the local docs route even when status carries an external docs link', () => {
    const { result } = renderTopNavLinks({
      docs_link: 'https://docs.newapi.pro/',
    })

    const docsLink = result.current.find((link) => link.title === 'Docs')

    expect(docsLink).toMatchObject({
      href: '/docs',
    })
    expect(docsLink?.external).toBeUndefined()
  })
})
