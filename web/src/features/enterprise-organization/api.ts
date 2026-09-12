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
  AddDepartmentMemberPayload,
  ApiResponse,
  BudgetDelegationListResponse,
  BudgetDelegationResponse,
  CreateBudgetDelegationPayload,
  CreateDepartmentBudgetPayload,
  CreatePublicBudgetPoolPayload,
  DepartmentBudgetLifecyclePayload,
  CreateQuotaAllocationPayload,
  DecideQuotaRequestPayload,
  DepartmentBudgetDetailResponse,
  DepartmentBudgetListResponse,
  DepartmentBudgetSortField,
  DepartmentBudgetResponse,
  GovernanceNotificationResponse,
  GovernanceNotificationResendResponse,
  GovernanceTimelineResponse,
  DepartmentMemberItem,
  DepartmentMembersResponse,
  DepartmentOwnerMutationPayload,
  DepartmentOwnersResponse,
  DepartmentTreeNode,
  QuotaAllocationListResponse,
  QuotaAllocationResponse,
  QuotaRequestCapabilityResponse,
  QuotaRequestListResponse,
  QuotaRequestResponse,
  RenameDepartmentMemberPayload,
  ResizeDepartmentBudgetPayload,
  ReclaimQuotaAllocationPayload,
  RevokeQuotaAllocationPayload,
  ReplaceUserDepartmentsPayload,
  CancelQuotaAllocationPayload,
  SupersedeQuotaAllocationPayload,
  SupersedeBudgetDelegationPayload,
  SubmitQuotaRequestPayload,
  UserDepartmentsResponse,
} from './types'

export const enterpriseOrganizationQueryKey = [
  'enterprise',
  'organization',
] as const

export function publicBudgetPoolQueryKey(tenantId: number) {
  return [
    ...enterpriseOrganizationQueryKey,
    'public-budget-pools',
    tenantId,
  ] as const
}

export function departmentBudgetQueryKey(
  departmentId: number,
  tenantId: number
) {
  return [
    ...enterpriseOrganizationQueryKey,
    'department-budget',
    departmentId,
    tenantId,
  ] as const
}

export function quotaAllocationQueryScopeKey(
  departmentId: number,
  tenantId: number
) {
  return [
    ...enterpriseOrganizationQueryKey,
    'quota-allocation',
    departmentId,
    tenantId,
  ] as const
}

export function quotaAllocationQueryKey(
  departmentId: number,
  tenantId: number,
  budgetId: number | null
) {
  return [
    ...quotaAllocationQueryScopeKey(departmentId, tenantId),
    budgetId,
  ] as const
}

export function budgetDelegationQueryKey(
  departmentId: number,
  tenantId: number
) {
  return [
    ...enterpriseOrganizationQueryKey,
    'budget-delegation',
    departmentId,
    tenantId,
  ] as const
}

export function quotaRequestQueryScopeKey(
  departmentId: number,
  tenantId: number
) {
  return [
    ...enterpriseOrganizationQueryKey,
    'quota-request',
    departmentId,
    tenantId,
  ] as const
}

export function quotaRequestQueryKey(
  departmentId: number,
  tenantId: number,
  requesterUserId: number | null | undefined
) {
  return [
    ...quotaRequestQueryScopeKey(departmentId, tenantId),
    requesterUserId ?? 'all',
  ] as const
}

export function governanceTimelineQueryKey(
  departmentId: number,
  tenantId: number
) {
  return [
    ...enterpriseOrganizationQueryKey,
    'governance-timeline',
    departmentId,
    tenantId,
  ] as const
}

export function governanceNotificationQueryKey(
  departmentId: number,
  tenantId: number
) {
  return [
    ...enterpriseOrganizationQueryKey,
    'governance-notification',
    departmentId,
    tenantId,
  ] as const
}

export function departmentBudgetListQueryScopeKey(
  departmentId: number,
  tenantId: number
) {
  return [
    ...enterpriseOrganizationQueryKey,
    'department-budget-list',
    departmentId,
    tenantId,
  ] as const
}

export function departmentBudgetListQueryKey(
  departmentId: number,
  tenantId: number,
  includeDescendants: boolean,
  sortBy?: DepartmentBudgetSortField,
  sortOrder?: 'asc' | 'desc'
) {
  return [
    ...departmentBudgetListQueryScopeKey(departmentId, tenantId),
    includeDescendants,
    sortBy,
    sortOrder,
  ] as const
}

export function departmentBudgetDetailQueryScopeKey(
  departmentId: number,
  tenantId: number
) {
  return [
    ...enterpriseOrganizationQueryKey,
    'department-budget-detail',
    departmentId,
    tenantId,
  ] as const
}

export function departmentBudgetDetailQueryKey(
  departmentId: number,
  tenantId: number,
  budgetId: number | null
) {
  return [
    ...departmentBudgetDetailQueryScopeKey(departmentId, tenantId),
    budgetId,
  ] as const
}

export function departmentMembersQueryKey(departmentId: number, tenantId = 0) {
  return [
    ...enterpriseOrganizationQueryKey,
    'department-members',
    departmentId,
    tenantId,
  ] as const
}

export function departmentOwnersQueryKey(departmentId: number, tenantId = 0) {
  return [
    ...enterpriseOrganizationQueryKey,
    'department-owners',
    departmentId,
    tenantId,
  ] as const
}

export function userDepartmentsQueryKey(userId: number | null) {
  return [
    ...enterpriseOrganizationQueryKey,
    'user-departments',
    userId,
  ] as const
}

export async function getDepartmentTree(): Promise<
  ApiResponse<DepartmentTreeNode[]>
> {
  const res = await api.get('/api/enterprise/departments/tree')
  return res.data
}

export async function getUserDepartments(
  userId: number
): Promise<ApiResponse<UserDepartmentsResponse>> {
  const res = await api.get(`/api/enterprise/users/${userId}/departments`)
  return res.data
}

export async function replaceUserDepartments(
  userId: number,
  payload: ReplaceUserDepartmentsPayload
): Promise<ApiResponse<UserDepartmentsResponse>> {
  const res = await api.put(
    `/api/enterprise/users/${userId}/departments`,
    payload
  )
  return res.data
}

export async function getDepartmentMembers(
  departmentId: number,
  tenantId?: number
): Promise<ApiResponse<DepartmentMembersResponse>> {
  const res = await api.get(
    `/api/enterprise/departments/${departmentId}/members`,
    {
      params: tenantId === undefined ? undefined : { tenant_id: tenantId },
    }
  )
  return res.data
}

export async function getDepartmentOwners(
  departmentId: number,
  tenantId?: number
): Promise<ApiResponse<DepartmentOwnersResponse>> {
  const res = await api.get(
    `/api/enterprise/departments/${departmentId}/owners`,
    {
      params: tenantId === undefined ? undefined : { tenant_id: tenantId },
    }
  )
  return res.data
}

export async function grantDepartmentOwner(
  departmentId: number,
  payload: DepartmentOwnerMutationPayload
): Promise<ApiResponse> {
  const res = await api.post(
    `/api/enterprise/departments/${departmentId}/owners/grants`,
    payload
  )
  return res.data
}

export async function denyDepartmentOwner(
  departmentId: number,
  payload: DepartmentOwnerMutationPayload
): Promise<ApiResponse> {
  const res = await api.post(
    `/api/enterprise/departments/${departmentId}/owners/denies`,
    payload
  )
  return res.data
}

export async function revokeDepartmentOwnerGrant(
  departmentId: number,
  userId: number,
  payload?: DepartmentOwnerMutationPayload
): Promise<ApiResponse> {
  const res = await api.delete(
    `/api/enterprise/departments/${departmentId}/owners/grants/${userId}`,
    { data: payload }
  )
  return res.data
}

export async function revokeDepartmentOwnerDeny(
  departmentId: number,
  userId: number,
  payload?: DepartmentOwnerMutationPayload
): Promise<ApiResponse> {
  const res = await api.delete(
    `/api/enterprise/departments/${departmentId}/owners/denies/${userId}`,
    { data: payload }
  )
  return res.data
}

export async function getDepartmentBudget(
  departmentId: number,
  tenantId?: number
): Promise<ApiResponse<DepartmentBudgetResponse>> {
  const res = await api.get(
    `/api/enterprise/departments/${departmentId}/budget`,
    {
      params: tenantId === undefined ? undefined : { tenant_id: tenantId },
    }
  )
  return res.data
}

export async function getDepartmentBudgets(
  departmentId: number,
  params?: {
    tenant_id?: number
    include_descendants?: boolean
    sort_by?: DepartmentBudgetSortField
    sort_order?: 'asc' | 'desc'
  }
): Promise<ApiResponse<DepartmentBudgetListResponse>> {
  const res = await api.get(
    `/api/enterprise/departments/${departmentId}/budgets`,
    {
      params,
    }
  )
  return res.data
}

export async function getDepartmentBudgetDetail(
  departmentId: number,
  budgetId: number,
  tenantId?: number
): Promise<ApiResponse<DepartmentBudgetDetailResponse>> {
  const res = await api.get(
    `/api/enterprise/departments/${departmentId}/budgets/${budgetId}`,
    {
      params: tenantId === undefined ? undefined : { tenant_id: tenantId },
    }
  )
  return res.data
}

export async function createDepartmentBudget(
  departmentId: number,
  payload: CreateDepartmentBudgetPayload
): Promise<ApiResponse<DepartmentBudgetResponse>> {
  const res = await api.post(
    `/api/enterprise/departments/${departmentId}/budget`,
    payload
  )
  return res.data
}

export async function pauseDepartmentBudget(
  departmentId: number,
  budgetId: number,
  payload: DepartmentBudgetLifecyclePayload
): Promise<ApiResponse<DepartmentBudgetResponse>> {
  const res = await api.post(
    `/api/enterprise/departments/${departmentId}/budgets/${budgetId}/pause`,
    payload
  )
  return res.data
}

export async function resumeDepartmentBudget(
  departmentId: number,
  budgetId: number,
  payload: DepartmentBudgetLifecyclePayload
): Promise<ApiResponse<DepartmentBudgetResponse>> {
  const res = await api.post(
    `/api/enterprise/departments/${departmentId}/budgets/${budgetId}/resume`,
    payload
  )
  return res.data
}

export async function resizeDepartmentBudget(
  departmentId: number,
  budgetId: number,
  payload: ResizeDepartmentBudgetPayload
): Promise<ApiResponse<DepartmentBudgetResponse>> {
  const res = await api.post(
    `/api/enterprise/departments/${departmentId}/budgets/${budgetId}/resize`,
    payload
  )
  return res.data
}

export async function getPublicBudgetPools(
  tenantId?: number,
  includeInactive = true
): Promise<ApiResponse<DepartmentBudgetListResponse>> {
  const res = await api.get('/api/enterprise/public-budget-pools', {
    params: {
      ...(tenantId === undefined ? {} : { tenant_id: tenantId }),
      include_inactive: includeInactive,
    },
  })
  return res.data
}

export async function createPublicBudgetPool(
  payload: CreatePublicBudgetPoolPayload
): Promise<ApiResponse<DepartmentBudgetResponse>> {
  const res = await api.post('/api/enterprise/public-budget-pools', payload)
  return res.data
}

export async function pausePublicBudgetPool(
  budgetId: number,
  payload: DepartmentBudgetLifecyclePayload
): Promise<ApiResponse<DepartmentBudgetResponse>> {
  const res = await api.post(
    `/api/enterprise/public-budget-pools/${budgetId}/pause`,
    payload
  )
  return res.data
}

export async function resumePublicBudgetPool(
  budgetId: number,
  payload: DepartmentBudgetLifecyclePayload
): Promise<ApiResponse<DepartmentBudgetResponse>> {
  const res = await api.post(
    `/api/enterprise/public-budget-pools/${budgetId}/resume`,
    payload
  )
  return res.data
}

export async function resizePublicBudgetPool(
  budgetId: number,
  payload: ResizeDepartmentBudgetPayload
): Promise<ApiResponse<DepartmentBudgetResponse>> {
  const res = await api.post(
    `/api/enterprise/public-budget-pools/${budgetId}/resize`,
    payload
  )
  return res.data
}

export async function getQuotaAllocations(
  departmentBudgetId: number,
  tenantId: number | undefined,
  departmentId: number
): Promise<ApiResponse<QuotaAllocationListResponse>> {
  const res = await api.get('/api/enterprise/quota-allocations', {
    params: {
      department_budget_id: departmentBudgetId,
      department_id: departmentId,
      ...(tenantId === undefined ? {} : { tenant_id: tenantId }),
    },
  })
  return res.data
}

export async function createQuotaAllocation(
  payload: CreateQuotaAllocationPayload
): Promise<ApiResponse<QuotaAllocationResponse>> {
  const res = await api.post('/api/enterprise/quota-allocations', payload)
  return res.data
}

export async function revokeQuotaAllocation(
  allocationId: number,
  payload: RevokeQuotaAllocationPayload
): Promise<ApiResponse<QuotaAllocationResponse>> {
  const res = await api.post(
    `/api/enterprise/quota-allocations/${allocationId}/revoke`,
    payload
  )
  return res.data
}

export async function supersedeQuotaAllocation(
  allocationId: number,
  payload: SupersedeQuotaAllocationPayload
): Promise<ApiResponse<QuotaAllocationResponse>> {
  const res = await api.post(
    `/api/enterprise/quota-allocations/${allocationId}/supersede`,
    payload
  )
  return res.data
}

export async function cancelQuotaAllocation(
  allocationId: number,
  payload: CancelQuotaAllocationPayload
): Promise<ApiResponse<QuotaAllocationResponse>> {
  const res = await api.post(
    `/api/enterprise/quota-allocations/${allocationId}/cancel`,
    payload
  )
  return res.data
}

export async function reclaimQuotaAllocation(
  allocationId: number,
  payload: ReclaimQuotaAllocationPayload
): Promise<ApiResponse<QuotaAllocationResponse>> {
  const res = await api.post(
    `/api/enterprise/quota-allocations/${allocationId}/reclaim`,
    payload
  )
  return res.data
}

export async function getQuotaRequests(params: {
  department_id?: number
  tenant_id?: number
  requester_user_id?: number
  include_pending?: boolean
  limit?: number
  view?: 'history' | 'approval'
  status?: string
  page?: number
  page_size?: number
}): Promise<ApiResponse<QuotaRequestListResponse>> {
  const res = await api.get('/api/enterprise/quota-requests', {
    params,
  })
  return res.data
}

export async function submitQuotaRequest(
  payload: SubmitQuotaRequestPayload
): Promise<ApiResponse<QuotaRequestResponse>> {
  const res = await api.post('/api/enterprise/quota-requests', payload)
  return res.data
}

export async function decideQuotaRequest(
  requestId: number,
  payload: DecideQuotaRequestPayload
): Promise<ApiResponse<QuotaRequestResponse>> {
  const res = await api.post(
    `/api/enterprise/quota-requests/${requestId}/decision`,
    payload,
    {
      skipBusinessError: true,
    }
  )
  return res.data
}

export async function getQuotaRequestCapability(
  departmentId: number,
  tenantId?: number
): Promise<ApiResponse<QuotaRequestCapabilityResponse>> {
  const res = await api.get(
    `/api/enterprise/quota-requests/capability/${departmentId}`,
    {
      params: tenantId === undefined ? undefined : { tenant_id: tenantId },
    }
  )
  return res.data
}

export async function getGovernanceTimeline(params: {
  tenant_id?: number
  department_id?: number
  source_type?: string
  action_type?: string
  status?: string
  page?: number
  page_size?: number
}): Promise<ApiResponse<GovernanceTimelineResponse>> {
  const res = await api.get('/api/enterprise/governance/timeline', {
    params,
  })
  return res.data
}

export async function getGovernanceNotifications(params: {
  tenant_id?: number
  department_id?: number
  source_type?: string
  action_type?: string
  status?: string
  page?: number
  page_size?: number
}): Promise<ApiResponse<GovernanceNotificationResponse>> {
  const res = await api.get('/api/enterprise/governance/notifications', {
    params,
  })
  return res.data
}

export async function resendGovernanceNotification(
  deliveryId: number,
  tenantId?: number
): Promise<ApiResponse<GovernanceNotificationResendResponse>> {
  const res = await api.post(
    `/api/enterprise/governance/notifications/${deliveryId}/resend`,
    null,
    {
      params: tenantId === undefined ? undefined : { tenant_id: tenantId },
    }
  )
  return res.data
}

export async function getBudgetDelegations(
  departmentId: number,
  tenantId?: number
): Promise<ApiResponse<BudgetDelegationListResponse>> {
  const res = await api.get('/api/enterprise/budget-delegations', {
    params: {
      department_id: departmentId,
      ...(tenantId === undefined ? {} : { tenant_id: tenantId }),
    },
  })
  return res.data
}

export async function createBudgetDelegation(
  payload: CreateBudgetDelegationPayload
): Promise<ApiResponse<BudgetDelegationResponse>> {
  const res = await api.post('/api/enterprise/budget-delegations', payload)
  return res.data
}

export async function supersedeBudgetDelegation(
  delegationId: number,
  payload: SupersedeBudgetDelegationPayload
): Promise<ApiResponse<BudgetDelegationResponse>> {
  const res = await api.post(
    `/api/enterprise/budget-delegations/${delegationId}/supersede`,
    payload
  )
  return res.data
}

export async function addDepartmentMember(
  departmentId: number,
  payload: AddDepartmentMemberPayload
): Promise<ApiResponse<DepartmentMemberItem>> {
  const res = await api.post(
    `/api/enterprise/departments/${departmentId}/members`,
    payload
  )
  return res.data
}

export async function deactivateDepartmentMember(
  departmentId: number,
  userId: number,
  tenantId?: number
): Promise<ApiResponse> {
  const res = await api.delete(
    `/api/enterprise/departments/${departmentId}/members/${userId}`,
    {
      params: tenantId === undefined ? undefined : { tenant_id: tenantId },
    }
  )
  return res.data
}

export async function restoreDepartmentMember(
  departmentId: number,
  userId: number,
  tenantId?: number
): Promise<ApiResponse<DepartmentMemberItem>> {
  const res = await api.post(
    `/api/enterprise/departments/${departmentId}/members/${userId}/restore`,
    undefined,
    {
      params: tenantId === undefined ? undefined : { tenant_id: tenantId },
    }
  )
  return res.data
}

export async function renameDepartmentMember(
  departmentId: number,
  userId: number,
  payload: RenameDepartmentMemberPayload
): Promise<ApiResponse<DepartmentMemberItem>> {
  const res = await api.put(
    `/api/enterprise/departments/${departmentId}/members/${userId}/username`,
    payload
  )
  return res.data
}
