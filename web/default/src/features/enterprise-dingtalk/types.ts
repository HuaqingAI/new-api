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
export type ApiResponse<T = unknown> = {
  success: boolean
  message?: string
  data?: T
}

export type DingTalkConfig = {
  id: number
  tenant_id: number
  corp_id: string
  app_key: string
  callback_url: string
  sync_scope: string
  login_enabled: boolean
  sync_enabled: boolean
  has_app_secret: boolean
  created_at: number
  updated_at: number
}

export type DingTalkConfigPayload = {
  tenant_id?: number
  corp_id: string
  app_key: string
  app_secret?: string
  callback_url: string
  sync_scope: string
  login_enabled?: boolean
  sync_enabled?: boolean
}

export type DingTalkConnectivityCode =
  | 'auth_success'
  | 'auth_invalid_credentials'
  | 'auth_permission_insufficient'
  | 'network_unreachable'
  | 'callback_misconfigured'

export type DingTalkConnectivityResult = {
  tenant_id: number
  code: DingTalkConnectivityCode
  stage: string
  summary: string
  http_status?: number
  checked_at: number
}

export type DingTalkSyncTask = {
  id: number
  tenant_id: number
  mode: string
  status: 'pending' | 'running' | 'succeeded' | 'failed'
  progress: number
  departments_created: number
  departments_updated: number
  departments_disabled: number
  users_created: number
  users_updated: number
  memberships_created: number
  memberships_updated: number
  memberships_disabled: number
  skipped_count: number
  failed_count: number
  error_summary: string
  created_by: number
  started_at: number
  finished_at: number
  created_at: number
  updated_at: number
}

export type DingTalkSyncLog = {
  id: number
  task_id: number
  tenant_id: number
  object_type: string
  object_external_id: string
  action: string
  status: string
  message: string
  created_at: number
}

export type DingTalkSyncLogsResult = {
  items: DingTalkSyncLog[]
  total: number
  page: number
  page_size: number
}
