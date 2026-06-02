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
  superseded_by_id: number
  supersedes_allocation_id: number
  revoke_reason: string
  reclaimed_quota: number
  processed_source: string
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

export type QuotaRequestItem = {
  id: number
  tenant_id: number
  department_id: number
  department_name: string
  department_budget_id: number
  budget_mode: string
  requester_user_id: number
  requester_username: string
  requester_display_name: string
  requested_quota: number
  approved_quota: number
  status: string
  approver_user_id: number
  approver_username: string
  approval_reason: string
  request_reason: string
  allocation_id: number
  owner_count_snapshot: number
  fallback: string
  submitted_at: number
  approved_at: number
  rejected_at: number
  fulfilled_at: number
  processed_at: number
  expires_at: number
  created_at: number
  updated_at: number
}

export type QuotaRequestResponse = {
  item: QuotaRequestItem | null
  allocation?: QuotaAllocationItem | null
}

export type QuotaRequestListResponse = {
  items: QuotaRequestItem[]
}

export type QuotaRequestCapabilityBudgetItem = DepartmentBudgetItem

export type QuotaRequestCapabilityResponse = {
  can_submit: boolean
  can_govern: boolean
  budgets: QuotaRequestCapabilityBudgetItem[]
}

export type BudgetDelegationItem = {
  id: number
  tenant_id: number
  source_department_id: number
  source_department_name: string
  source_budget_id: number
  target_department_id: number
  target_department_name: string
  target_budget_id: number
  actor_id: number
  committed_quota: number
  budget_type_snapshot: string
  cycle_type_snapshot: string
  before_source_budget_snapshot: string
  after_source_budget_snapshot: string
  before_target_budget_snapshot: string
  after_target_budget_snapshot: string
  status: string
  superseded_by_id: number
  processed_at: number
  reason: string
  created_at: number
  updated_at: number
}

export type BudgetDelegationResponse = {
  item: BudgetDelegationItem | null
}

export type BudgetDelegationListResponse = {
  items: BudgetDelegationItem[]
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

export type CreateBudgetDelegationPayload = {
  tenant_id?: number
  source_department_id: number
  source_budget_id: number
  target_department_id: number
  target_budget_id: number
  committed_quota?: number
  reason?: string
}

export type SupersedeBudgetDelegationPayload = {
  tenant_id?: number
  source_department_id: number
  new_committed_quota?: number
  reason?: string
}

export type RevokeQuotaAllocationPayload = {
  tenant_id?: number
  department_id: number
  reason?: string
}

export type SupersedeQuotaAllocationPayload = {
  tenant_id?: number
  department_id: number
  new_committed_quota?: number
  reason?: string
}

export type CancelQuotaAllocationPayload = {
  tenant_id?: number
  department_id: number
  reason?: string
}

export type ReclaimQuotaAllocationPayload = {
  tenant_id?: number
  department_id: number
  reason?: string
}

export type SubmitQuotaRequestPayload = {
  tenant_id?: number
  department_id: number
  department_budget_id: number
  budget_mode: string
  requested_quota?: number
  request_reason?: string
  idempotency_key?: string
}

export type DecideQuotaRequestPayload = {
  tenant_id?: number
  action: 'approve' | 'reject'
  approved_quota?: number
  approval_reason?: string
  rejected_reason?: string
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
