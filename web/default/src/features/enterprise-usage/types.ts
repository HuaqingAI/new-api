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

export type UsageModelDistributionItem = {
  model_name: string
  request_count: number
  prompt_tokens: number
  completion_tokens: number
  quota: number
}

export type DepartmentUsageSummaryItem = {
  dept_id: number | null
  dept_name: string
  window_start: number
  window_end: number
  request_count: number
  prompt_tokens: number
  completion_tokens: number
  quota: number
  user_count: number
  model_distribution: UsageModelDistributionItem[]
}

export type DepartmentUsageSummaryResponse = {
  items: DepartmentUsageSummaryItem[]
}

export type EnterpriseUsagePreset =
  | 'today'
  | 'yesterday'
  | 'last7d'
  | 'last30d'
  | 'custom'

export type EnterpriseUsageSearch = {
  preset?: EnterpriseUsagePreset
  from?: number
  to?: number
  tenant_id?: number
}
