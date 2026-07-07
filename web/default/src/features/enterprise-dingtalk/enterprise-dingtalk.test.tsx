import i18n from '@/i18n/config'
import { isRedirect } from '@tanstack/react-router'
import { Route as EnterpriseDingTalkRoute } from '@/routes/_authenticated/enterprise-dingtalk/index'
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import {
  EnterpriseDingTalkConnectivityResult,
  EnterpriseDingTalkSyncPanel,
  DingTalkSyncConflictList,
} from './index'
import type { DingTalkConnectivityCode } from './types'

i18n.changeLanguage('en')

describe('Enterprise DingTalk connectivity guidance', () => {
  test('renders stable guidance for every backend result code', () => {
    const expected: Record<DingTalkConnectivityCode, string> = {
      auth_success: 'DingTalk app is reachable',
      auth_invalid_credentials: 'Check DingTalk app credentials',
      auth_permission_insufficient: 'Grant address book permission',
      network_unreachable: 'DingTalk network is unreachable',
      callback_misconfigured: 'Fix DingTalk callback URL',
    }

    for (const [code, title] of Object.entries(expected)) {
      const html = renderConnectivityResult(code as DingTalkConnectivityCode)
      assert.match(html, new RegExp(escapeRegExp(title)))
      assert.match(html, new RegExp(escapeRegExp(code)))
      assert.doesNotMatch(html, /plain-secret|access-token|raw upstream/i)
    }
  })
})

describe('Enterprise DingTalk sync panel', () => {
  test('renders task counters and sanitized logs', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <EnterpriseDingTalkSyncPanel
          canEdit
          config={{
            id: 1,
            tenant_id: 0,
            corp_id: 'corp',
            app_key: 'app-key',
            callback_url: 'https://example.com/api/oauth/dingtalk',
            sync_scope: '1',
            login_enabled: true,
            sync_enabled: true,
            has_app_secret: true,
            created_at: 0,
            updated_at: 0,
          }}
          task={{
            id: 9,
            tenant_id: 0,
            mode: 'full',
            status: 'succeeded',
            progress: 100,
            departments_created: 2,
            departments_updated: 1,
            departments_disabled: 0,
            users_created: 1,
            users_updated: 0,
            memberships_created: 2,
            memberships_updated: 0,
            memberships_disabled: 0,
            skipped_count: 3,
            failed_count: 0,
            error_summary: '',
            created_by: 1,
            started_at: 1700000000,
            finished_at: 1700000001,
            created_at: 1700000000,
            updated_at: 1700000001,
          }}
          logs={[
            {
              id: 1,
              task_id: 9,
              tenant_id: 0,
              object_type: 'department',
              object_external_id: '10',
              action: 'created',
              status: 'success',
              message: 'department_created',
              created_at: 1700000001,
            },
          ]}
          conflicts={[
            {
              id: 5,
              tenant_id: 0,
              task_id: 9,
              external_user_id: 'staff-conflict',
              union_id: 'union-conflict',
              mobile: '13800000000',
              email: 'taken@example.com',
              name: 'Conflict User',
              conflict_type: 'email_mobile',
              candidate_user_id: 100,
              details: 'email_and_mobile_match_existing_accounts',
              status: 'pending',
              last_task_id: 9,
              resolved_by: 0,
              resolved_at: 0,
              created_at: 1700000001,
              updated_at: 1700000001,
            },
          ]}
          logsLoading={false}
          conflictsLoading={false}
          syncing={false}
          resolvingConflictId={null}
          onStartSync={() => undefined}
          onRefreshLogs={() => undefined}
          onBindCandidate={() => undefined}
        />
      </I18nextProvider>
    )

    assert.match(html, /Address Book Sync/)
    assert.match(html, /Pending Sync Conflicts/)
    assert.match(html, /Conflict User/)
    assert.match(html, /Bind candidate/)
    assert.match(html, /Task/)
    assert.match(html, /department_created/)
    assert.doesNotMatch(html, /plain-secret|access-token|raw upstream/i)
  })

  test('renders bind action only for conflicts with a single candidate', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <DingTalkSyncConflictList
          loading={false}
          resolvingConflictId={null}
          onBindCandidate={() => undefined}
          conflicts={[
            {
              id: 5,
              tenant_id: 0,
              task_id: 9,
              external_user_id: 'staff-single',
              union_id: 'union-single',
              mobile: '13800000000',
              email: 'single@example.com',
              name: 'Single Candidate',
              conflict_type: 'email',
              candidate_user_id: 100,
              details: 'email_matches_existing_local_user',
              status: 'pending',
              last_task_id: 9,
              resolved_by: 0,
              resolved_at: 0,
              created_at: 1700000001,
              updated_at: 1700000001,
            },
            {
              id: 6,
              tenant_id: 0,
              task_id: 9,
              external_user_id: 'staff-many',
              union_id: 'union-many',
              mobile: '',
              email: 'many@example.com',
              name: 'Many Candidates',
              conflict_type: 'email',
              candidate_user_id: 0,
              details: 'email_matches_existing_local_user',
              status: 'pending',
              last_task_id: 9,
              resolved_by: 0,
              resolved_at: 0,
              created_at: 1700000001,
              updated_at: 1700000001,
            },
          ]}
        />
      </I18nextProvider>
    )

    assert.match(html, /Single Candidate/)
    assert.match(html, /Candidate user #100/)
    assert.match(html, /Many Candidates/)
    assert.match(html, /Multiple candidate users/)
    assert.equal((html.match(/Bind candidate/g) ?? []).length, 1)
  })
})

describe('Enterprise DingTalk route guard', () => {
  test('allows only super admins to enter DingTalk integration', () => {
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
        EnterpriseDingTalkRoute.options.beforeLoad?.({} as never)
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
        EnterpriseDingTalkRoute.options.beforeLoad?.({} as never)
      )
    } finally {
      auth.setUser(previousUser)
    }
  })
})

function renderConnectivityResult(code: DingTalkConnectivityCode) {
  return renderToStaticMarkup(
    <I18nextProvider i18n={i18n}>
      <EnterpriseDingTalkConnectivityResult
        result={{
          tenant_id: 0,
          code,
          stage: 'address_book_probe',
          summary: 'stable_summary',
          checked_at: 1700000000,
        }}
      />
    </I18nextProvider>
  )
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
