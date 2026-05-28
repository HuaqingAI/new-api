import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'
import i18n from '@/i18n/config'
import { EnterpriseDingTalkConnectivityResult } from './index'
import type { DingTalkConnectivityCode } from './types'

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
