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

export type AgentPlatformItemResponse = {
  success: boolean
  message?: string
  data?: AgentPlatformItem
}

export type CreateAgentPlatformResourceRequest = {
  display_name: string
  owner_user_id: number
  tenant_id?: number
}

export type UpdateAgentPlatformResourceRequest = {
  display_name: string
}

export type AgentPlatformJsonValue =
  | null
  | boolean
  | number
  | string
  | AgentPlatformJsonValue[]
  | { [key: string]: AgentPlatformJsonValue }

export type AgentPlatformSkillDetailRequest = {
  invoke_schema?: AgentPlatformJsonValue
  output_schema?: AgentPlatformJsonValue
  invoke_mode?: string
  timeout_seconds?: number
  binding_config?: AgentPlatformJsonValue
}

export type AgentPlatformKnowledgeDetailRequest = {
  knowledge_mode?: string
  provider_type?: string
  provider_adapter_key?: string
  provider_config?: AgentPlatformJsonValue
  query_schema?: AgentPlatformJsonValue
  citation_schema?: AgentPlatformJsonValue
  freshness_rules?: AgentPlatformJsonValue
  provider_capabilities?: AgentPlatformJsonValue
}

export type AgentPlatformAgentDetailRequest = {
  manifest?: AgentPlatformJsonValue
  dependencies?: AgentPlatformJsonValue
  prompt_metadata?: AgentPlatformJsonValue
  compatibility_metadata?: AgentPlatformJsonValue
}

export type CreateAgentPlatformVersionRequest = {
  version: string
  contract_version: string
  summary?: string
  schema?: AgentPlatformJsonValue
  skill?: AgentPlatformSkillDetailRequest
  knowledge?: AgentPlatformKnowledgeDetailRequest
  agent?: AgentPlatformAgentDetailRequest
}

export type AgentPlatformSkillDetail = {
  invoke_schema?: AgentPlatformJsonValue
  output_schema?: AgentPlatformJsonValue
  invoke_mode: string
  timeout_seconds: number
  binding_config?: AgentPlatformJsonValue
}

export type AgentPlatformKnowledgeDetail = {
  knowledge_mode: string
  provider_type: string
  provider_adapter_key: string
  provider_config?: AgentPlatformJsonValue
  query_schema?: AgentPlatformJsonValue
  citation_schema?: AgentPlatformJsonValue
  freshness_rules?: AgentPlatformJsonValue
  provider_capabilities?: AgentPlatformJsonValue
}

export type AgentPlatformAgentDetail = {
  manifest?: AgentPlatformJsonValue
  dependencies?: AgentPlatformJsonValue
  prompt_metadata?: AgentPlatformJsonValue
  compatibility_metadata?: AgentPlatformJsonValue
}

export type AgentPlatformVersionItem = {
  resource_id: string
  resource_type: AgentPlatformResourceType
  version: string
  contract_version: string
  summary: string
  schema?: AgentPlatformJsonValue
  status: string
  created_by: number
  published_at?: number
  created_at: number
  updated_at: number
  skill?: AgentPlatformSkillDetail
  knowledge?: AgentPlatformKnowledgeDetail
  agent?: AgentPlatformAgentDetail
}

export type AgentPlatformVersionResponse = {
  success: boolean
  message?: string
  data?: AgentPlatformVersionItem
}

export type AgentPlatformLifecycleAction =
  | 'publish'
  | 'disable'
  | 'revoke'
  | 'offline'

export type AgentPlatformLifecycleRequest = {
  version?: string
  request_id?: string
}

export type AgentPlatformLifecycleResult = {
  resource_id: string
  action: string
  previous_status: string
  current_status: string
  previous_version: string
  current_version: string
  target_version: string
  request_id: string
  audit_action_id: number
}

export type AgentPlatformLifecycleResponse = {
  success: boolean
  message?: string
  data?: AgentPlatformLifecycleResult
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

function resourceEndpoint(resourceType: AgentPlatformResourceType) {
  return RESOURCE_ENDPOINTS[resourceType].endpoint
}

export async function getAgentPlatformResource(
  resourceType: AgentPlatformResourceType,
  resourceId: string
) {
  const res = await api.get<AgentPlatformItemResponse>(
    `${resourceEndpoint(resourceType)}/${encodeURIComponent(resourceId)}`
  )
  return res.data
}

export async function createAgentPlatformResource(
  resourceType: AgentPlatformResourceType,
  payload: CreateAgentPlatformResourceRequest
) {
  const res = await api.post<AgentPlatformItemResponse>(
    resourceEndpoint(resourceType),
    payload
  )
  return res.data
}

export async function updateAgentPlatformResource(
  resourceType: AgentPlatformResourceType,
  resourceId: string,
  payload: UpdateAgentPlatformResourceRequest
) {
  const res = await api.put<AgentPlatformItemResponse>(
    `${resourceEndpoint(resourceType)}/${encodeURIComponent(resourceId)}`,
    payload
  )
  return res.data
}

export async function createAgentPlatformResourceVersion(
  resourceId: string,
  payload: CreateAgentPlatformVersionRequest
) {
  const res = await api.post<AgentPlatformVersionResponse>(
    `/api/agent-platform/resources/${encodeURIComponent(resourceId)}/versions`,
    payload
  )
  return res.data
}

export async function getAgentPlatformResourceVersion(
  resourceId: string,
  version: string
) {
  const res = await api.get<AgentPlatformVersionResponse>(
    `/api/agent-platform/resources/${encodeURIComponent(resourceId)}/versions/${encodeURIComponent(version)}`
  )
  return res.data
}

export async function runAgentPlatformLifecycleAction(
  resourceId: string,
  action: AgentPlatformLifecycleAction,
  payload: AgentPlatformLifecycleRequest = {}
) {
  const res = await api.post<AgentPlatformLifecycleResponse>(
    `/api/agent-platform/resources/${encodeURIComponent(resourceId)}/${action}`,
    payload
  )
  return res.data
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
