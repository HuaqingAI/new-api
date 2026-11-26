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
  auto_sync_on_login: boolean
  scheduled_full_sync_enabled?: boolean
  scheduled_full_sync_cron?: string
  scheduled_full_sync_timezone?: string
  scheduled_full_sync_next_run_at?: number
  scheduled_full_sync_last_run_at?: number
  scheduled_full_sync_last_task_id?: number
  scheduled_full_sync_last_status?: string
  scheduled_full_sync_last_error?: string
  scheduled_full_sync_revision?: number
  scheduled_full_sync_updated_at?: number
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
  auto_sync_on_login?: boolean
  scheduled_full_sync_enabled?: boolean
  scheduled_full_sync_cron?: string
  scheduled_full_sync_timezone?: string
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

export type DingTalkSyncConflict = {
  id: number
  tenant_id: number
  task_id: number
  trigger_source: 'full_sync' | 'login'
  external_user_id: string
  union_id: string
  mobile: string
  email: string
  name: string
  conflict_type: string
  candidate_user_id: number
  details: string
  status: 'pending' | 'resolved' | 'ignored'
  last_task_id: number
  resolved_by: number
  resolved_at: number
  created_at: number
  updated_at: number
}

export type DingTalkSyncConflictsResult = {
  items: DingTalkSyncConflict[]
  total: number
  page: number
  page_size: number
}
