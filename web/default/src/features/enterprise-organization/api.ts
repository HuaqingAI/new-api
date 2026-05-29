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
  DepartmentMemberItem,
  DepartmentMembersResponse,
  DepartmentTreeNode,
  ReplaceUserDepartmentsPayload,
  UserDepartmentsResponse,
} from './types'

export const enterpriseOrganizationQueryKey = [
  'enterprise',
  'organization',
] as const

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
  const res = await api.get(`/api/enterprise/departments/${departmentId}/members`)
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
