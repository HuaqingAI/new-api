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

export type AgentPlatformResourceType = 'mcp' | 'skill' | 'knowledge' | 'agent'
export type AgentPlatformAgentCliType = 'opencode' | 'codex'

export type AgentPlatformJsonValue =
  | null
  | boolean
  | number
  | string
  | AgentPlatformJsonValue[]
  | { [key: string]: AgentPlatformJsonValue }

export type AgentPlatformItem = {
  id: number
  resource_id: string
  resource_type: AgentPlatformResourceType
  display_name: string
  description: string
  avatar: string
  owner_user_id: number
  owner_name?: string
  status: string
  latest_version: string
  tenant_id: number
  created_at: number
  updated_at: number
}

export type AgentPlatformMcpItem = AgentPlatformItem & {
  config?: AgentPlatformJsonValue
}

export type AgentPlatformSkillItem = AgentPlatformItem & {
  file_name?: string
  sha256?: string
  size_bytes?: number
}

export type AgentPlatformKnowledgeItem = AgentPlatformItem & {
  external_knowledge_id?: string
}

export type AgentPlatformAgentItem = AgentPlatformItem & {
  cli_type: AgentPlatformAgentCliType
  instructions?: string
  model_token_id?: number
  default_model?: string
  model_token_user_id?: number
  model_token_user_name?: string
  model_token_name?: string
  model_token_masked_key?: string
  mcp_ids?: string[]
  skill_ids?: string[]
  knowledge_ids?: string[]
}

export type AgentPlatformListPayload<TItem extends AgentPlatformItem> = {
  items: TItem[]
  total: number
  page: number
  page_size: number
}

export type AgentPlatformListResponse<
  TItem extends AgentPlatformItem = AgentPlatformItem,
> = {
  success: boolean
  message?: string
  data?: AgentPlatformListPayload<TItem>
}

export type AgentPlatformItemResponse<
  TItem extends AgentPlatformItem = AgentPlatformItem,
> = {
  success: boolean
  message?: string
  data?: TItem
}

export type CreateAgentPlatformMcpRequest = {
  display_name: string
  description?: string
  config: AgentPlatformJsonValue
  owner_user_id?: number
  tenant_id?: number
}

export type UpdateAgentPlatformMcpRequest = {
  display_name: string
  description?: string
  config: AgentPlatformJsonValue
}

export type CreateAgentPlatformSkillRequest = {
  display_name: string
  description?: string
  owner_user_id?: number
  tenant_id?: number
  file?: File | null
}

export type UpdateAgentPlatformSkillRequest = {
  display_name: string
  description?: string
}

export type CreateAgentPlatformKnowledgeRequest = {
  display_name: string
  description?: string
  external_knowledge_id?: string
  owner_user_id?: number
  tenant_id?: number
}

export type UpdateAgentPlatformKnowledgeRequest = {
  display_name: string
  description?: string
  external_knowledge_id?: string
}

export type CreateAgentPlatformAgentRequest = {
  cli_type: AgentPlatformAgentCliType
  display_name: string
  description?: string
  avatar?: string
  instructions?: string
  model_token_id?: number
  default_model?: string
  mcp_ids?: string[]
  skill_ids?: string[]
  knowledge_ids?: string[]
  owner_user_id?: number
  tenant_id?: number
}

export type UpdateAgentPlatformAgentRequest = Omit<
  CreateAgentPlatformAgentRequest,
  'owner_user_id' | 'tenant_id'
>

export type AgentPlatformGrantSubjectType = 'user' | 'department'

export type AgentPlatformGrantRequest = {
  subject_type: AgentPlatformGrantSubjectType
  subject_id: string
  subject_name?: string
}

export type PublishAgentPlatformAgentRequest = {
  summary: string
  grants: {
    users: string[]
    departments: string[]
  }
}

export type AgentPlatformAgentVersion = {
  resource_id: string
  version: string
  summary: string
  status: string
  package_path: string
  package_sha256: string
  package_size: number
  published_at?: string
  created_at: string
}

export type AgentPlatformAgentVersionsResponse = {
  success: boolean
  message?: string
  data?: {
    items: AgentPlatformAgentVersion[]
  }
}

export type AgentPlatformAgentGrantsResponse = {
  success: boolean
  message?: string
  data?: {
    items: AgentPlatformGrantRequest[]
  }
}

export type PublishAgentPlatformDefaultsResponse = {
  success: boolean
  message?: string
  data?: {
    resource_id: string
    latest_version: string
    next_version: string
    grants: AgentPlatformGrantRequest[]
  }
}

export type PublishAgentPlatformAgentResponse = {
  success: boolean
  message?: string
  data?: {
    resource_id: string
    version: string
    status: string
    artifact: {
      cli_type: string
      url: string
      sha256: string
      size: number
    }
    grants: Array<{
      grant_id: string
      subject_type: AgentPlatformGrantSubjectType
      subject_id: string
      subject_name?: string
    }>
  }
}

export type AgentPlatformModelKey = {
  id: number
  user_id: number
  user_name: string
  name: string
  masked_key: string
  status: number
  expired_time: number
  remain_quota: number
  unlimited_quota: boolean
  group: string
  model_limits_enabled: boolean
  model_count: number
  available: boolean
  disabled_reason?: string
}

export type AgentPlatformModelItem = {
  model: string
  display_name: string
  status: string
  capabilities?: AgentPlatformJsonValue
}

export type AgentPlatformModelKeysResponse = {
  success: boolean
  message?: string
  data?: {
    items: AgentPlatformModelKey[]
  }
}

export type AgentPlatformModelKeyModelsResponse = {
  success: boolean
  message?: string
  data?: {
    token_id: number
    items: AgentPlatformModelItem[]
  }
}

export type UploadAgentPlatformAvatarResponse = {
  success: boolean
  message?: string
  data?: {
    url: string
  }
}

type ResourceEndpointConfig = {
  endpoint: string
  resourceType: AgentPlatformResourceType
}

const RESOURCE_ENDPOINTS: Record<
  AgentPlatformResourceType,
  ResourceEndpointConfig
> = {
  mcp: {
    endpoint: '/api/agent-platform/mcps',
    resourceType: 'mcp',
  },
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

async function fetchResourceList<TItem extends AgentPlatformItem>(
  resourceType: AgentPlatformResourceType
): Promise<AgentPlatformListResponse<TItem>> {
  const config = RESOURCE_ENDPOINTS[resourceType]
  const params = { page: 1, page_size: 100 }

  try {
    const res = await api.get<AgentPlatformListResponse<TItem>>(
      config.endpoint,
      {
        params,
        skipErrorHandler: true,
      }
    )
    return res.data
  } catch (error) {
    if (!isNotFoundError(error)) {
      throw error
    }
  }

  const fallback = await api.get<AgentPlatformListResponse<TItem>>(
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

function buildSkillFormData(
  payload: CreateAgentPlatformSkillRequest,
  file?: File | null
) {
  const formData = new FormData()
  formData.set('display_name', payload.display_name)
  if (payload.description) {
    formData.set('description', payload.description)
  }
  if (payload.owner_user_id != null) {
    formData.set('owner_user_id', String(payload.owner_user_id))
  }
  if (payload.tenant_id != null) {
    formData.set('tenant_id', String(payload.tenant_id))
  }
  if (file) {
    formData.set('file', file)
  }
  return formData
}

export async function getAgentPlatformResource<
  TItem extends AgentPlatformItem = AgentPlatformItem,
>(resourceType: AgentPlatformResourceType, resourceId: string) {
  const res = await api.get<AgentPlatformItemResponse<TItem>>(
    `${resourceEndpoint(resourceType)}/${encodeURIComponent(resourceId)}`
  )
  return res.data
}

export async function getAgentPlatformMcps() {
  return fetchResourceList<AgentPlatformMcpItem>('mcp')
}

export async function getAgentPlatformSkills() {
  return fetchResourceList<AgentPlatformSkillItem>('skill')
}

export async function getAgentPlatformKnowledge() {
  return fetchResourceList<AgentPlatformKnowledgeItem>('knowledge')
}

export async function getAgentPlatformAgents() {
  return fetchResourceList<AgentPlatformAgentItem>('agent')
}

export async function getAgentPlatformModelKeys(keyword?: string) {
  const normalizedKeyword = keyword?.trim()
  const res = await api.get<AgentPlatformModelKeysResponse>(
    '/api/agent-platform/model-keys',
    {
      params: normalizedKeyword ? { keyword: normalizedKeyword } : undefined,
    }
  )
  return res.data
}

export async function getAgentPlatformModelKeyModels(tokenId: number) {
  const res = await api.get<AgentPlatformModelKeyModelsResponse>(
    `/api/agent-platform/model-keys/${encodeURIComponent(String(tokenId))}/models`
  )
  return res.data
}

export async function createAgentPlatformMcp(
  payload: CreateAgentPlatformMcpRequest
) {
  const res = await api.post<AgentPlatformItemResponse<AgentPlatformMcpItem>>(
    RESOURCE_ENDPOINTS.mcp.endpoint,
    payload
  )
  return res.data
}

export async function updateAgentPlatformMcp(
  resourceId: string,
  payload: UpdateAgentPlatformMcpRequest
) {
  const res = await api.put<AgentPlatformItemResponse<AgentPlatformMcpItem>>(
    `${RESOURCE_ENDPOINTS.mcp.endpoint}/${encodeURIComponent(resourceId)}`,
    payload
  )
  return res.data
}

export async function createAgentPlatformSkill(
  payload: CreateAgentPlatformSkillRequest
) {
  const res = await api.post<AgentPlatformItemResponse<AgentPlatformSkillItem>>(
    RESOURCE_ENDPOINTS.skill.endpoint,
    buildSkillFormData(payload, payload.file)
  )
  return res.data
}

export async function updateAgentPlatformSkill(
  resourceId: string,
  payload: UpdateAgentPlatformSkillRequest
) {
  const res = await api.put<AgentPlatformItemResponse<AgentPlatformSkillItem>>(
    `${RESOURCE_ENDPOINTS.skill.endpoint}/${encodeURIComponent(resourceId)}`,
    payload
  )
  return res.data
}

export async function uploadAgentPlatformSkillPackage(
  resourceId: string,
  file: File
) {
  const formData = new FormData()
  formData.set('file', file)
  const res = await api.post<AgentPlatformItemResponse<AgentPlatformSkillItem>>(
    `${RESOURCE_ENDPOINTS.skill.endpoint}/${encodeURIComponent(resourceId)}/package`,
    formData
  )
  return res.data
}

export async function createAgentPlatformKnowledge(
  payload: CreateAgentPlatformKnowledgeRequest
) {
  const res = await api.post<
    AgentPlatformItemResponse<AgentPlatformKnowledgeItem>
  >(RESOURCE_ENDPOINTS.knowledge.endpoint, payload)
  return res.data
}

export async function updateAgentPlatformKnowledge(
  resourceId: string,
  payload: UpdateAgentPlatformKnowledgeRequest
) {
  const res = await api.put<
    AgentPlatformItemResponse<AgentPlatformKnowledgeItem>
  >(
    `${RESOURCE_ENDPOINTS.knowledge.endpoint}/${encodeURIComponent(resourceId)}`,
    payload
  )
  return res.data
}

export async function createAgentPlatformAgent(
  payload: CreateAgentPlatformAgentRequest
) {
  const res = await api.post<AgentPlatformItemResponse<AgentPlatformAgentItem>>(
    RESOURCE_ENDPOINTS.agent.endpoint,
    payload
  )
  return res.data
}

export async function updateAgentPlatformAgent(
  resourceId: string,
  payload: UpdateAgentPlatformAgentRequest
) {
  const res = await api.put<AgentPlatformItemResponse<AgentPlatformAgentItem>>(
    `${RESOURCE_ENDPOINTS.agent.endpoint}/${encodeURIComponent(resourceId)}`,
    payload
  )
  return res.data
}

export async function setAgentPlatformResourceEnabled(
  resourceType: Exclude<AgentPlatformResourceType, 'agent'>,
  resourceId: string,
  enabled: boolean
) {
  const res = await api.post<AgentPlatformItemResponse>(
    `${resourceEndpoint(resourceType)}/${encodeURIComponent(resourceId)}/${enabled ? 'enable' : 'disable'}`
  )
  return res.data
}

export async function deleteAgentPlatformResource(
  resourceType: AgentPlatformResourceType,
  resourceId: string
) {
  const res = await api.delete<{ success: boolean; message?: string }>(
    `${resourceEndpoint(resourceType)}/${encodeURIComponent(resourceId)}`
  )
  return res.data
}

export async function publishAgentPlatformAgent(
  resourceId: string,
  payload: PublishAgentPlatformAgentRequest
) {
  const res = await api.post<PublishAgentPlatformAgentResponse>(
    `${RESOURCE_ENDPOINTS.agent.endpoint}/${encodeURIComponent(resourceId)}/publish`,
    payload
  )
  return res.data
}

export async function uploadAgentPlatformAvatar(file: File) {
  const formData = new FormData()
  formData.set('file', file)
  const res = await api.post<UploadAgentPlatformAvatarResponse>(
    '/api/agent-platform/assets/avatars',
    formData
  )
  return res.data
}

export async function getAgentPlatformAgentVersions(resourceId: string) {
  const res = await api.get<AgentPlatformAgentVersionsResponse>(
    `${RESOURCE_ENDPOINTS.agent.endpoint}/${encodeURIComponent(resourceId)}/versions`
  )
  return res.data
}

export async function getAgentPlatformAgentVersionGrants(
  resourceId: string,
  version: string
) {
  const res = await api.get<AgentPlatformAgentGrantsResponse>(
    `${RESOURCE_ENDPOINTS.agent.endpoint}/${encodeURIComponent(resourceId)}/versions/${encodeURIComponent(version)}/grants`
  )
  return res.data
}

export async function getAgentPlatformPublishDefaults(resourceId: string) {
  const res = await api.get<PublishAgentPlatformDefaultsResponse>(
    `${RESOURCE_ENDPOINTS.agent.endpoint}/${encodeURIComponent(resourceId)}/publish-defaults`
  )
  return res.data
}

export function getAgentPlatformSkillPackageDownloadUrl(resourceId: string) {
  return `${RESOURCE_ENDPOINTS.skill.endpoint}/${encodeURIComponent(resourceId)}/package/download`
}
