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
  message: string
  data: T
}

export type AlertEventDepartmentSnapshot = {
  department_id: number
  department_name: string
  external_source: string
  status: number
}

export type AlertEventItem = {
  id: number
  tenant_id: number
  user_id: number
  username: string
  request_id: string
  model_name: string
  risk_type: string
  action_result: string
  created_at: number
  department_snapshot: AlertEventDepartmentSnapshot[]
  summary: string
}

export type AlertEventsResponse = {
  items: AlertEventItem[]
  total: number
  page: number
  page_size: number
}

export type AlertRuleChannelType = 'email' | 'webhook' | 'dingtalk_robot'

export type AlertRuleChannelConfigInput = {
  type: AlertRuleChannelType
  enabled?: boolean
  receivers?: string[]
  webhook_url?: string
  webhook_secret?: string
  dingtalk_robot_url?: string
  dingtalk_robot_secret?: string
}

export type AlertRuleChannelConfigItem = {
  type: AlertRuleChannelType
  enabled: boolean
  receivers: string[]
  webhook_url?: string
  webhook_secret_configured: boolean
  webhook_secret_masked?: string
  dingtalk_robot_secret_configured: boolean
  dingtalk_robot_secret_masked?: string
  dingtalk_robot_url?: string
}

export type AlertRuleItem = {
  id: number
  tenant_id: number
  name: string
  enabled: boolean
  risk_types: string[]
  department_ids: number[]
  channel_configs: AlertRuleChannelConfigItem[]
  dedupe_window_seconds: number
  created_by: number
  updated_by: number
  created_at: number
  updated_at: number
}

export type AlertRulesResponse = {
  items: AlertRuleItem[]
  total: number
}

export type AlertRuleResponse = {
  item: AlertRuleItem
}

export type AlertRuleUpsertRequest = {
  id?: number
  tenant_id?: number
  name?: string
  enabled?: boolean
  risk_types: string[]
  department_ids: number[]
  channel_configs: AlertRuleChannelConfigInput[]
  dedupe_window_seconds?: number
}

export type EnterpriseAlertsSearch = {
  tenant_id?: number
  department_id?: number
  user_id?: number
  username?: string
  model_name?: string
  risk_type?: string
  from?: number
  to?: number
  page?: number
  page_size?: number
}
