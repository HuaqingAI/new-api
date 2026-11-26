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
export const DEPARTMENT_STATUS = {
  ENABLED: 1,
  DISABLED: 2,
  DELETED: 3,
} as const

export const DEPARTMENT_SYNC_STATUS = {
  NONE: 0,
  OK: 1,
  WARNING: 2,
  FAILED: 3,
} as const

export const DEPARTMENT_SOURCE_TYPE = {
  MANUAL: 1,
  DINGTALK: 2,
} as const

export const DEPARTMENT_TREE_QUERY_KEY = [
  'enterprise',
  'organization',
  'department-tree',
] as const
