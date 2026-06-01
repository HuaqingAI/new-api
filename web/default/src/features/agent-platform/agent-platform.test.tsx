import i18n from '@/i18n/config'
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  RouterContextProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'
import { AgentPlatformShell } from './index'

i18n.changeLanguage('en')

const rootRoute = createRootRoute()
const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: AgentPlatformShell,
})
const testRouter = createRouter({
  routeTree: rootRoute.addChildren([indexRoute]),
  history: createMemoryHistory({ initialEntries: ['/'] }),
})

describe('Agent Platform shell', () => {
  test('renders route-backed management workspace copy', () => {
    const queryClient = new QueryClient()
    const html = renderToStaticMarkup(
      <RouterContextProvider router={testRouter}>
        <QueryClientProvider client={queryClient}>
          <I18nextProvider i18n={i18n}>
            <AgentPlatformShell />
          </I18nextProvider>
        </QueryClientProvider>
      </RouterContextProvider>
    )

    assert.match(html, /Agent Platform/)
    assert.match(html, /Agent Platform Overview/)
    assert.match(html, /Overview/)
    assert.match(html, /Skills/)
    assert.match(html, /Knowledge/)
    assert.match(html, /Agents/)
    assert.match(html, /Navigation contract/)
    assert.match(html, /Current implementation focus/)
    assert.match(html, /Live route surface/)
    assert.match(html, /Control-plane API wiring/)
    assert.match(html, /UI parity/)
    assert.match(html, /Audit &amp; Diagnostics/)
    assert.match(html, /Workspace domains/)
  })
})
