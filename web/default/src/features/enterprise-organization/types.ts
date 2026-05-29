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

export type UserDepartmentsResponse = {
  items: UserDepartmentItem[]
  total: number
  is_unassigned: boolean
}

export type DepartmentMembersResponse = {
  items: DepartmentMemberItem[]
  total: number
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
