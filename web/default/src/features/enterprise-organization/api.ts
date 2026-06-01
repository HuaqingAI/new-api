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
  CreateQuotaAllocationPayload,
  DepartmentBudgetDetailResponse,
  DepartmentBudgetListResponse,
  DepartmentBudgetSortField,
  DepartmentBudgetResponse,
  DepartmentMemberItem,
  DepartmentMembersResponse,
  DepartmentOwnerMutationPayload,
  DepartmentOwnersResponse,
  DepartmentTreeNode,
  QuotaAllocationListResponse,
  QuotaAllocationResponse,
  RenameDepartmentMemberPayload,
  RevokeQuotaAllocationPayload,
  ReplaceUserDepartmentsPayload,
  SupersedeBudgetDelegationPayload,
  UserDepartmentsResponse,
} from './types'

export const enterpriseOrganizationQueryKey = [
  'enterprise',
  'organization',
] as const

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

export function departmentMembersQueryKey(departmentId: number) {
  return [
    ...enterpriseOrganizationQueryKey,
    'department-members',
    departmentId,
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
  departmentId: number
): Promise<ApiResponse<DepartmentMembersResponse>> {
  const res = await api.get(
    `/api/enterprise/departments/${departmentId}/members`
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
  userId: number
): Promise<ApiResponse> {
  const res = await api.delete(
    `/api/enterprise/departments/${departmentId}/members/${userId}`
  )
  return res.data
}

export async function restoreDepartmentMember(
  departmentId: number,
  userId: number
): Promise<ApiResponse<DepartmentMemberItem>> {
  const res = await api.post(
    `/api/enterprise/departments/${departmentId}/members/${userId}/restore`
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
