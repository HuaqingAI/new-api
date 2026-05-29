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
import type {
  ApiResponse,
  DingTalkConnectivityResult,
  DingTalkConfig,
  DingTalkConfigPayload,
  DingTalkSyncConflictsResult,
  DingTalkSyncLogsResult,
  DingTalkSyncTask,
} from './types'

export const enterpriseDingTalkQueryKey = [
  'enterprise',
  'dingtalk',
  'config',
] as const

export const enterpriseDingTalkConnectivityQueryKey = [
  'enterprise',
  'dingtalk',
  'connectivity',
] as const

export const enterpriseDingTalkSyncLogsQueryKey = [
  'enterprise',
  'dingtalk',
  'sync',
  'logs',
] as const

export const enterpriseDingTalkSyncConflictsQueryKey = [
  'enterprise',
  'dingtalk',
  'sync',
  'conflicts',
] as const

export async function getDingTalkConfig(): Promise<
  ApiResponse<DingTalkConfig>
> {
  const res = await api.get('/api/enterprise/dingtalk/config')
  return res.data
}

export async function saveDingTalkConfig(
  payload: DingTalkConfigPayload
): Promise<ApiResponse<DingTalkConfig>> {
  const res = await api.put('/api/enterprise/dingtalk/config', payload)
  return res.data
}

export async function testDingTalkConnectivity(): Promise<
  ApiResponse<DingTalkConnectivityResult>
> {
  const res = await api.post('/api/enterprise/dingtalk/connectivity-test')
  return res.data
}

export async function startDingTalkFullSync(
  inline = false
): Promise<ApiResponse<DingTalkSyncTask>> {
  const res = await api.post('/api/enterprise/dingtalk/sync/full', { inline })
  return res.data
}

export async function getDingTalkSyncTask(
  taskId: number
): Promise<ApiResponse<DingTalkSyncTask>> {
  const res = await api.get(`/api/enterprise/dingtalk/sync/tasks/${taskId}`)
  return res.data
}

export async function listDingTalkSyncLogs(params?: {
  task_id?: number
  status?: string
  object_type?: string
  page?: number
  page_size?: number
}): Promise<ApiResponse<DingTalkSyncLogsResult>> {
  const res = await api.get('/api/enterprise/dingtalk/sync/logs', { params })
  return res.data
}

export async function listDingTalkSyncConflicts(params?: {
  status?: string
  page?: number
  page_size?: number
}): Promise<ApiResponse<DingTalkSyncConflictsResult>> {
  const res = await api.get('/api/enterprise/dingtalk/sync/conflicts', {
    params,
  })
  return res.data
}
