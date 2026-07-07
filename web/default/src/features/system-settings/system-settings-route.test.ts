import { isRedirect } from '@tanstack/react-router'
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import { Route as SystemSettingsRoute } from '@/routes/_authenticated/system-settings/route'

describe('System settings route guard', () => {
  test('allows only super admins to enter system settings', () => {
    const { auth } = useAuthStore.getState()
    const previousUser = auth.user

    try {
      auth.setUser({
        id: 1001,
        username: 'admin',
        role: ROLE.ADMIN,
      })

      let redirected: unknown = null
      try {
        SystemSettingsRoute.options.beforeLoad?.({} as never)
      } catch (error) {
        redirected = error
      }

      assert.ok(isRedirect(redirected))
      if (isRedirect(redirected)) {
        assert.equal(redirected.options.to, '/403')
      }

      auth.setUser({
        id: 1002,
        username: 'root',
        role: ROLE.SUPER_ADMIN,
      })

      assert.doesNotThrow(() =>
        SystemSettingsRoute.options.beforeLoad?.({} as never)
      )
    } finally {
      auth.setUser(previousUser)
    }
  })
})
