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
