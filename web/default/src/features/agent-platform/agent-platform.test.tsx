import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  RouterContextProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router'
import i18n from '@/i18n/config'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { describe, test } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'
import { api } from '@/lib/api'
import { filterSidebarNavGroupsForConfig } from '@/hooks/use-sidebar-config'
import type { NavGroup } from '@/components/layout/types'
import { getAgentPlatformAgents, getAgentPlatformSkills } from './api'
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
    queryClient.setQueryData(['agent-platform', 'skills', 'summary'], {
      success: true,
      data: {
        items: [
          {
            id: 1,
            resource_id: 'skill_translate',
            resource_type: 'skill',
            display_name: 'Translate Skill',
            owner_user_id: 7,
            status: 'published',
            latest_version: '1.0.0',
            tenant_id: 0,
            created_at: 1717113600,
            updated_at: 1717117200,
          },
        ],
        total: 1,
        page: 1,
        page_size: 12,
      },
    })
    queryClient.setQueryData(['agent-platform', 'knowledge', 'summary'], {
      success: true,
      data: { items: [], total: 0, page: 1, page_size: 12 },
    })
    queryClient.setQueryData(['agent-platform', 'agents', 'summary'], {
      success: true,
      data: { items: [], total: 0, page: 1, page_size: 12 },
    })
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
    assert.match(html, /Resource control plane/)
    assert.match(html, /Skills/)
    assert.match(html, /Knowledge/)
    assert.match(html, /Agents/)
    assert.match(html, /Published resources/)
    assert.match(html, /Translate Skill/)
    assert.match(html, /skill_translate/)
    assert.match(html, /Latest version/)
    assert.match(html, /Details/)
  })

  test('renders control-plane failure responses as error state', () => {
    const queryClient = new QueryClient()
    queryClient.setQueryData(['agent-platform', 'skills', 'summary'], {
      success: false,
      message: 'upstream unavailable',
    })
    queryClient.setQueryData(['agent-platform', 'knowledge', 'summary'], {
      success: true,
      data: { items: [], total: 0, page: 1, page_size: 12 },
    })
    queryClient.setQueryData(['agent-platform', 'agents', 'summary'], {
      success: true,
      data: { items: [], total: 0, page: 1, page_size: 12 },
    })

    const html = renderToStaticMarkup(
      <RouterContextProvider router={testRouter}>
        <QueryClientProvider client={queryClient}>
          <I18nextProvider i18n={i18n}>
            <AgentPlatformShell />
          </I18nextProvider>
        </QueryClientProvider>
      </RouterContextProvider>
    )

    assert.match(html, /Failed to load data/)
    assert.match(html, /Skill API request failed/)
    assert.match(html, /upstream unavailable/)
    assert.doesNotMatch(html, /No Skill resources are currently available/)
  })

  test('generated router contains the authenticated agent-platform route', async () => {
    const source = await readFile('src/routeTree.gen.ts', 'utf8')

    assert.match(source, /AuthenticatedAgentPlatformIndexRouteImport/)
    assert.match(source, /'\/_authenticated\/agent-platform\/'/)
    assert.match(source, /fullPath: '\/agent-platform\/'/)
    assert.match(
      source,
      /'\/agent-platform': typeof AuthenticatedAgentPlatformIndexRoute/
    )
  })

  test('sidebar module configuration controls the Agent Platform admin entry', () => {
    const navGroups: NavGroup[] = [
      {
        id: 'admin',
        title: 'Admin',
        items: [
          {
            title: 'Agent Platform',
            url: '/agent-platform',
          },
          {
            title: 'Users',
            url: '/users',
          },
        ],
      },
    ]

    const visibleByDefault = filterSidebarNavGroupsForConfig(
      navGroups,
      null,
      null
    )
    assert.deepEqual(
      visibleByDefault[0].items.map((item) => item.title),
      ['Agent Platform', 'Users']
    )

    const hiddenByAdmin = filterSidebarNavGroupsForConfig(
      navGroups,
      JSON.stringify({
        admin: {
          enabled: true,
          agent_platform: false,
          user: true,
        },
      }),
      null
    )
    assert.deepEqual(
      hiddenByAdmin[0].items.map((item) => item.title),
      ['Users']
    )
  })

  test('loads skill data from dedicated control-plane endpoint first', async () => {
    const originalGet = api.get
    const calls: Array<{ url: string; params?: unknown }> = []

    api.get = (async (url: string, config?: Record<string, unknown>) => {
      calls.push({
        url,
        params: config?.params,
      })
      return {
        data: {
          success: true,
          data: {
            items: [],
            total: 0,
            page: 1,
            page_size: 12,
          },
        },
      }
    }) as typeof api.get

    try {
      await getAgentPlatformSkills()

      assert.deepEqual(calls, [
        {
          url: '/api/agent-platform/skills',
          params: {
            page: 1,
            page_size: 12,
          },
        },
      ])
    } finally {
      api.get = originalGet
    }
  })

  test('does not fall back for non-404 control-plane errors', async () => {
    const originalGet = api.get
    const calls: Array<{
      url: string
      params?: unknown
      skipErrorHandler?: unknown
    }> = []
    const serverError = {
      isAxiosError: true,
      response: { status: 500 },
      message: 'server unavailable',
    }

    api.get = (async (url: string, config?: Record<string, unknown>) => {
      calls.push({
        url,
        params: config?.params,
        skipErrorHandler: config?.skipErrorHandler,
      })
      throw serverError
    }) as typeof api.get

    try {
      await assert.rejects(() => getAgentPlatformSkills(), serverError)
      assert.deepEqual(calls, [
        {
          url: '/api/agent-platform/skills',
          params: {
            page: 1,
            page_size: 12,
          },
          skipErrorHandler: true,
        },
      ])
    } finally {
      api.get = originalGet
    }
  })

  test('falls back to typed resources endpoint only for explicit 404 compatibility', async () => {
    const originalGet = api.get
    const calls: Array<{
      url: string
      params?: unknown
      skipErrorHandler?: unknown
    }> = []

    api.get = (async (url: string, config?: Record<string, unknown>) => {
      calls.push({
        url,
        params: config?.params,
        skipErrorHandler: config?.skipErrorHandler,
      })
      if (url === '/api/agent-platform/agents') {
        throw {
          isAxiosError: true,
          response: { status: 404 },
        }
      }
      return {
        data: {
          success: true,
          data: {
            items: [],
            total: 0,
            page: 1,
            page_size: 12,
          },
        },
      }
    }) as typeof api.get

    try {
      await getAgentPlatformAgents()

      assert.deepEqual(calls, [
        {
          url: '/api/agent-platform/agents',
          params: {
            page: 1,
            page_size: 12,
          },
          skipErrorHandler: true,
        },
        {
          url: '/api/agent-platform/resources',
          params: {
            page: 1,
            page_size: 12,
            resource_type: 'agent',
          },
          skipErrorHandler: undefined,
        },
      ])
    } finally {
      api.get = originalGet
    }
  })
})
