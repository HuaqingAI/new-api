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
import { createFileRoute, redirect } from '@tanstack/react-router'

import { AuthenticatedLayout } from '@/components/layout'
import { getSelf } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

// Avoid revalidating the same authenticated user on every route transition.
let verifiedSessionUserId: number | null = null

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: async ({ location }) => {
    const { auth } = useAuthStore.getState()

    if (!auth.user) {
      verifiedSessionUserId = null
      throw redirect({
        to: '/sign-in',
        search: { redirect: location.href },
      })
    }

    if (verifiedSessionUserId !== auth.user.id) {
      const res = await getSelf().catch((err: unknown) =>
        (err as { response?: { status?: number } })?.response?.status === 401
          ? { success: false }
          : null
      )

      if (res?.success && res.data) {
        auth.setUser(res.data)
        verifiedSessionUserId = res.data.id
      } else if (res) {
        verifiedSessionUserId = null
        auth.reset()
        throw redirect({
          to: '/sign-in',
          search: { redirect: location.href },
        })
      }
    }
  },
  component: AuthenticatedLayout,
})
