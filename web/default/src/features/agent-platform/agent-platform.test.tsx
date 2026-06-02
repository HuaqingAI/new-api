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
  test('renders navigation contract and live skill management copy', () => {
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
    assert.match(html, /Navigation contract/)
    assert.match(html, /Overview/)
    assert.match(html, /Clients/)
    assert.match(html, /Skills/)
    assert.match(html, /Publishing/)
    assert.match(html, /Audit &amp; Diagnostics/)
    assert.match(html, /Web Default MVP/)
    assert.match(html, /Skill management/)
    assert.match(html, /Epic 3 active/)
  })
})
