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
export interface DepartmentNameHistoryEntry {
  name: string
  changed_at: number
}

export interface DepartmentTreeNode {
  id: number
  tenant_id: number
  name: string
  parent_id: number | null
  status: number
  source_type: number
  external_id: string
  sync_status: number
  sync_error: string
  name_history: DepartmentNameHistoryEntry[]
  created_at: number
  updated_at: number
  deleted_at: number
  children: DepartmentTreeNode[]
}

export type ApiResponse<T = unknown> = {
  success: boolean
  message?: string
  data?: T
}

export type EnterpriseBudgetErrorData = {
  reason?: string
}

export type MembershipStatus = 1 | 2 | 3 | 4

export type UserDepartmentItem = {
  id: number
  tenant_id: number
  user_id: number
  department_id: number
  department_name: string
  external_user_id: string
  external_source: string
  status: MembershipStatus
  joined_at: number
  left_at: number
  created_at: number
  updated_at: number
}

export type DepartmentMemberItem = {
  id: number
  tenant_id: number
  user_id: number
  username: string
  display_name: string
  department_id: number
  external_user_id: string
  external_source: string
  status: MembershipStatus
  joined_at: number
  left_at: number
  created_at: number
  updated_at: number
}

export type RenameDepartmentMemberPayload = {
  tenant_id?: number
  new_username: string
}

export type UserDepartmentsResponse = {
  items: UserDepartmentItem[]
  total: number
  is_unassigned: boolean
}

export type DepartmentMembersResponse = {
  items: DepartmentMemberItem[]
  total: number
}

export type DepartmentOwnerFactItem = {
  id: number
  tenant_id: number
  user_id: number
  department_id: number
  role: number
  source: string
  effect: string
  external_source: string
  status: number
  inherited_from_department_id: number
  created_at: number
  updated_at: number
}

export type EffectiveDepartmentOwnerItem = {
  user_id: number
  department_id: number
  source: string
  effect: string
  inherited_from_department_id: number
  role_fact_id: number
}

export type DepartmentOwnersResponse = {
  facts: DepartmentOwnerFactItem[]
  effective_owners: EffectiveDepartmentOwnerItem[]
  owner_count: number
  fallback: string
}

export type DepartmentOwnerMutationPayload = {
  tenant_id?: number
  user_id: number
}

export type DepartmentBudgetType = 'balance' | 'subscription'
export type DepartmentBudgetStatus = 'active' | 'paused' | 'revoked' | 'expired'

export type DepartmentBudgetThresholdState = 'healthy' | 'warning' | 'critical'

export type DepartmentBudgetSortField =
  | 'usage_ratio'
  | 'remaining'
  | 'type'
  | 'status'

export type DepartmentBudgetThresholds = {
  warning: number
  critical: number
}

export type DepartmentBudgetItem = {
  id: number
  tenant_id: number
  department_id: number
  department_name: string
  type: DepartmentBudgetType
  status: DepartmentBudgetStatus | string
  total_quota: number
  remaining: number
  allocated_total: number
  cycle_quota: number
  cycle_type: string
  cycle_started_at: number
  custom_seconds: number
  expires_at: number
  parent_status: string
  usage_ratio: number
  threshold_state: DepartmentBudgetThresholdState | string
  created_at: number
  updated_at: number
}

export type DepartmentBudgetResponse = {
  item: DepartmentBudgetItem | null
}

export type DepartmentBudgetListResponse = {
  items: DepartmentBudgetItem[]
  thresholds: DepartmentBudgetThresholds
  scope_department_id?: number | null
  scope_department_name: string
  include_descendants: boolean
  scope_department_ids: number[]
}

export type DepartmentBudgetWalletDetail = {
  allocation_id: number
  allocation_status: string
  target_user_id: number
  target_username: string
  target_display_name: string
  wallet_id: number
  wallet_status: string
  quota: number
  remain_quota: number
  cycle_type: string
  cycle_started_at: number
  next_reset_time: number
  expires_at: number
  source_allocation_id: number
  source_parent_budget_id: number
  source_parent_budget_type: string
  source_parent_budget_status: string
  committed_quota: number
  processed_at: number
  created_at: number
  updated_at: number
  reason: string
}

export type DepartmentBudgetDetailResponse = {
  budget: DepartmentBudgetItem | null
  wallets: DepartmentBudgetWalletDetail[]
  thresholds: DepartmentBudgetThresholds
}

export type QuotaAllocationItem = {
  id: number
  tenant_id: number
  department_budget_id: number
  department_id: number
  target_user_id: number
  target_username: string
  target_display_name: string
  wallet_id: number
  actor_id: number
  committed_quota: number
  budget_type_snapshot: string
  cycle_type_snapshot: string
  cycle_started_at_snapshot: number
  custom_seconds_snapshot: number
  expires_at_snapshot: number
  reason: string
  status: string
  processed_at: number
  created_at: number
  updated_at: number
}

export type QuotaAllocationResponse = {
  item: QuotaAllocationItem | null
}

export type QuotaAllocationListResponse = {
  items: QuotaAllocationItem[]
}

export type CreateDepartmentBudgetPayload = {
  tenant_id?: number
  type: DepartmentBudgetType
  total_quota?: number
  cycle_quota?: number
  cycle_type?: string
  cycle_started_at?: number
  custom_seconds?: number
  expires_at?: number
}

export type CreateQuotaAllocationPayload = {
  tenant_id?: number
  department_budget_id: number
  department_id: number
  target_user_id: number
  committed_quota?: number
  reason?: string
}

export type RevokeQuotaAllocationPayload = {
  tenant_id?: number
  department_id: number
  reason?: string
}

export type ReplaceUserDepartmentsPayload = {
  tenant_id?: number
  department_ids: number[]
  external_user_id?: string
  external_source?: string
  deactivate_stale?: boolean
}

export type AddDepartmentMemberPayload = {
  tenant_id?: number
  user_id: number
  external_user_id?: string
  external_source?: string
}
