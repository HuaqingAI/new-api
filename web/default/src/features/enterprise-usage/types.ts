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

export type DepartmentUsageUserRankItem = {
  user_id: number
  username: string
  request_count: number
  prompt_tokens: number
  completion_tokens: number
  token_count: number
  quota: number
}

export type DepartmentUsageTrendPoint = {
  window_start: number
  window_end: number
  request_count: number
  prompt_tokens: number
  completion_tokens: number
  token_count: number
  quota: number
  user_count: number
}

export type DepartmentUsageLogFilters = {
  department_id: number | null
  department_name: string
  start_timestamp: number
  end_timestamp: number
  username: string
  username_options: string[]
}

export type DepartmentUsageLogEntryLink = {
  path: string
  section: string
  filters: DepartmentUsageLogFilters
}

export type DepartmentUsageDetailResponse = {
  dept_id: number | null
  dept_name: string
  window_start: number
  window_end: number
  request_count: number
  prompt_tokens: number
  completion_tokens: number
  token_count: number
  quota: number
  user_count: number
  user_ranking: DepartmentUsageUserRankItem[]
  model_distribution: UsageModelDistributionItem[]
  trend: DepartmentUsageTrendPoint[]
  recent_logs_entry: DepartmentUsageLogEntryLink
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
  dept_id?: number
  sort?: DepartmentUsageUserRankSort
  summary_sort?: DepartmentUsageSummarySort
  summary_order?: UsageSortOrder
  log_user?: string
}

export type DepartmentUsageUserRankSort = 'quota' | 'requests' | 'tokens'
export type DepartmentUsageSummarySort =
  | 'requests'
  | 'quota'
  | 'users'
  | 'dept_name'
export type UsageSortOrder = 'asc' | 'desc'
