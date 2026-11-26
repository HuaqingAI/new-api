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
import { api } from '@/lib/api'
import {
  createServerError,
  requireServerSuccess,
} from '@/lib/server-error-message'

export const MODEL_REASONING_QUERY_KEY = [
  'agent-platform',
  'model-reasoning-policy',
] as const
export const REASONING_EFFORTS = [
  'none',
  'low',
  'medium',
  'high',
  'xhigh',
  'max',
] as const
export const MODEL_FAMILIES = ['deepseek', 'qwen', 'glm', 'gpt'] as const
export type ReasoningEffort = (typeof REASONING_EFFORTS)[number]
export type ModelReasoningRule = {
  match_type: 'family' | 'model'
  match_value: string
  efforts: ReasoningEffort[]
}
export type ModelReasoningPolicy = {
  schema_version: 1
  scope: 'global'
  revision: string
  default_efforts: ReasoningEffort[]
  rules: ModelReasoningRule[]
}

type PolicyResponse = {
  success: boolean
  message?: string
  data?: ModelReasoningPolicy
}
const endpoint = '/api/agent-platform/model-reasoning-policy'

export async function getModelReasoningPolicy(): Promise<ModelReasoningPolicy> {
  const response = await api.get<PolicyResponse>(endpoint)
  const result = requireServerSuccess(response.data)
  if (!result.data) {
    throw createServerError(result, 'Failed to load model reasoning settings')
  }
  return result.data
}

export async function updateModelReasoningPolicy(payload: {
  rules: ModelReasoningRule[]
}): Promise<ModelReasoningPolicy> {
  const response = await api.put<PolicyResponse>(endpoint, payload)
  const result = requireServerSuccess(response.data)
  if (!result.data) {
    throw createServerError(result, 'Failed to save model reasoning settings')
  }
  return result.data
}
