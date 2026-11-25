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
import { t } from 'i18next'

import { api } from '@/lib/api'
import {
  createServerError,
  requireServerSuccess,
} from '@/lib/server-error-message'

export type ClientNotification = {
  id: string
  title: string
  summary: string
  content: string
  enabled: boolean
  version: number
  published_at: number
  created_at: number
  updated_at: number
  created_by: number
  updated_by: number
}

export type NotificationValues = Pick<
  ClientNotification,
  'title' | 'summary' | 'content' | 'enabled'
>
export type NotificationStatus = 'all' | 'enabled' | 'disabled'
export type NotificationList = {
  items: Omit<ClientNotification, 'content'>[]
  total: number
  page: number
  page_size: number
}
type Response<T> = { success: boolean; data?: T; message?: string }

export const NOTIFICATION_QUERY_KEY = [
  'agent-platform',
  'notifications',
  'admin',
] as const
const endpoint = '/api/agent-platform/notifications'

export async function getClientNotifications(params: {
  page: number
  page_size: number
  status: NotificationStatus
}): Promise<NotificationList> {
  const response = requireServerSuccess(
    (await api.get<Response<NotificationList>>(endpoint, { params })).data
  )
  if (!response.data) {
    throw createServerError(response, t('Failed to load client notifications'))
  }
  return response.data
}

export async function getClientNotification(
  id: string
): Promise<ClientNotification> {
  const response = requireServerSuccess(
    (
      await api.get<Response<ClientNotification>>(
        `${endpoint}/${encodeURIComponent(id)}`
      )
    ).data
  )
  if (!response.data) {
    throw createServerError(response, t('Failed to load client notifications'))
  }
  return response.data
}

export async function saveClientNotification(
  values: NotificationValues,
  item?: ClientNotification
): Promise<ClientNotification> {
  const result = item
    ? await api.put<Response<ClientNotification>>(
        `${endpoint}/${encodeURIComponent(item.id)}`,
        values,
        { headers: { 'If-Match': `"${item.version}"` } }
      )
    : await api.post<Response<ClientNotification>>(endpoint, values)
  const response = requireServerSuccess(result.data)
  if (!response.data) {
    throw createServerError(response, t('Failed to save client notification'))
  }
  return response.data
}

export async function changeClientNotification(
  item: Omit<ClientNotification, 'content'>,
  action: 'status' | 'delete'
): Promise<void> {
  const url = `${endpoint}/${encodeURIComponent(item.id)}`
  const config = { headers: { 'If-Match': `"${item.version}"` } }
  const response =
    action === 'delete'
      ? await api.delete<Response<unknown>>(url, config)
      : await api.patch<Response<ClientNotification>>(
          `${url}/status`,
          { enabled: !item.enabled },
          config
        )
  const result = requireServerSuccess(response.data)
  if (!result.data) {
    throw createServerError(result, t('Failed to save client notification'))
  }
}
