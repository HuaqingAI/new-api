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

export interface EnterpriseApiResponse<T> {
  success: boolean
  message?: string
  data: T
}
