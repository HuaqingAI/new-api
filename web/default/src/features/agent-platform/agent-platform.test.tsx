import i18n from '@/i18n/config'
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'
import { AgentPlatformShell } from './index'

i18n.changeLanguage('en')

describe('Agent Platform shell', () => {
  test('renders navigation contract and MVP scope copy', () => {
    const html = renderToStaticMarkup(
      <I18nextProvider i18n={i18n}>
        <AgentPlatformShell />
      </I18nextProvider>
    )

    assert.match(html, /Agent Platform/)
    assert.match(html, /Navigation contract/)
    assert.match(html, /Overview/)
    assert.match(html, /Clients/)
    assert.match(html, /Skills/)
    assert.match(html, /Publishing/)
    assert.match(html, /Audit &amp; Diagnostics/)
    assert.match(html, /Web Default MVP/)
    assert.match(html, /Client and publishing actions come next/)
  })
})
