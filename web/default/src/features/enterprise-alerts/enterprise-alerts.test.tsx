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
import { isRedirect } from '@tanstack/react-router'
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { Route as EnterpriseAlertsRoute } from '@/routes/_authenticated/enterprise-alerts/index'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import {
  enterpriseAlertsSearchSchema,
  formatDepartmentSnapshot,
  mapAlertFilterFormToSearch,
} from './index'

describe('Enterprise alerts feature', () => {
  test('maps form filters into API search params', () => {
    assert.deepEqual(
      mapAlertFilterFormToSearch({
        tenant_id: '7',
        department_id: '11',
        user_id: '99',
        username: 'alice',
        model_name: 'gpt-4o-mini',
        risk_type: 'sensitive_words',
        from: '1717117200',
        to: '1717203600',
        page_size: '50',
      }),
      {
        tenant_id: 7,
        department_id: 11,
        user_id: 99,
        username: 'alice',
        model_name: 'gpt-4o-mini',
        risk_type: 'sensitive_words',
        from: 1717117200,
        to: 1717203600,
        page: 1,
        page_size: 50,
      }
    )
  })

  test('search schema preserves page and page size defaults', () => {
    const parsed = enterpriseAlertsSearchSchema.parse({
      department_id: '11',
      page: '2',
      page_size: '50',
    })
    assert.equal(parsed.department_id, 11)
    assert.equal(parsed.page, 2)
    assert.equal(parsed.page_size, 50)
  })

  test('formats department snapshot for multi-department events and falls back to unassigned', () => {
    assert.equal(
      formatDepartmentSnapshot(
        {
          department_snapshot: [
            {
              department_id: 11,
              department_name: 'Engineering',
              external_source: 'manual',
              status: 1,
            },
            {
              department_id: 22,
              department_name: 'Security',
              external_source: 'manual',
              status: 1,
            },
          ],
        },
        'Unassigned'
      ),
      'Engineering (#11), Security (#22)'
    )

    assert.equal(
      formatDepartmentSnapshot({ department_snapshot: [] }, 'Unassigned'),
      'Unassigned'
    )
  })

  test('route guard redirects non-admin users and allows admins', () => {
    const { auth } = useAuthStore.getState()
    const previousUser = auth.user

    try {
      auth.setUser({
        id: 1001,
        username: 'member',
        role: ROLE.USER,
      })

      let redirected: unknown = null
      try {
        EnterpriseAlertsRoute.options.beforeLoad?.({} as never)
      } catch (error) {
        redirected = error
      }

      assert.ok(isRedirect(redirected))
      if (isRedirect(redirected)) {
        assert.equal(redirected.options.to, '/403')
      }

      auth.setUser({
        id: 1002,
        username: 'admin',
        role: ROLE.ADMIN,
      })

      assert.doesNotThrow(() =>
        EnterpriseAlertsRoute.options.beforeLoad?.({} as never)
      )
    } finally {
      auth.setUser(previousUser)
    }
  })

  test('mapped filters never include sensitive raw content fields', () => {
    const mapped = mapAlertFilterFormToSearch({
      tenant_id: '',
      department_id: '',
      user_id: '',
      username: '',
      model_name: '',
      risk_type: '',
      from: '',
      to: '',
      page_size: '20',
    })

    assert.ok(!('prompt' in mapped))
    assert.ok(!('messages' in mapped))
  })
})
