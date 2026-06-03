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
import axios from 'axios'
import { api } from '@/lib/api'

export type AgentPlatformResourceType = 'skill' | 'knowledge' | 'agent'

export type AgentPlatformItem = {
  id: number
  resource_id: string
  resource_type: AgentPlatformResourceType
  display_name: string
  owner_user_id: number
  status: string
  latest_version: string
  tenant_id: number
  created_at: number
  updated_at: number
}

export type AgentPlatformListPayload = {
  items: AgentPlatformItem[]
  total: number
  page: number
  page_size: number
}

export type AgentPlatformListResponse = {
  success: boolean
  message?: string
  data?: AgentPlatformListPayload
}

type ResourceEndpointConfig = {
  endpoint: string
  resourceType: AgentPlatformResourceType
}

const RESOURCE_ENDPOINTS: Record<
  AgentPlatformResourceType,
  ResourceEndpointConfig
> = {
  skill: {
    endpoint: '/api/agent-platform/skills',
    resourceType: 'skill',
  },
  knowledge: {
    endpoint: '/api/agent-platform/knowledge-bases',
    resourceType: 'knowledge',
  },
  agent: {
    endpoint: '/api/agent-platform/agents',
    resourceType: 'agent',
  },
}

function isNotFoundError(error: unknown): boolean {
  return axios.isAxiosError(error) && error.response?.status === 404
}

async function fetchResourceList(
  resourceType: AgentPlatformResourceType
): Promise<AgentPlatformListResponse> {
  const config = RESOURCE_ENDPOINTS[resourceType]
  const params = { page: 1, page_size: 12 }

  try {
    const res = await api.get<AgentPlatformListResponse>(config.endpoint, {
      params,
      skipErrorHandler: true,
    })
    return res.data
  } catch (error) {
    if (!isNotFoundError(error)) {
      throw error
    }
  }

  const fallback = await api.get<AgentPlatformListResponse>(
    '/api/agent-platform/resources',
    {
      params: {
        ...params,
        resource_type: config.resourceType,
      },
    }
  )
  return fallback.data
}

export function getAgentPlatformSkills() {
  return fetchResourceList('skill')
}

export function getAgentPlatformKnowledge() {
  return fetchResourceList('knowledge')
}

export function getAgentPlatformAgents() {
  return fetchResourceList('agent')
}
