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
import type {
  DepartmentUsageDetailResponse,
  DepartmentUsageSummaryItem,
  DepartmentUsageSummarySort,
  UsageSortOrder,
} from '../types'

export function normalizeDepartmentUsageItems(
  items: DepartmentUsageSummaryItem[],
  summarySort: DepartmentUsageSummarySort = 'requests',
  summaryOrder: UsageSortOrder = 'desc'
): DepartmentUsageSummaryItem[] {
  return [...items]
    .map((item) => ({
      ...item,
      dept_name: item.dept_name || '',
      model_distribution: item.model_distribution ?? [],
    }))
    .sort((left, right) =>
      compareDepartmentUsageItems(left, right, summarySort, summaryOrder)
    )
}

export function compareDepartmentUsageItems(
  left: DepartmentUsageSummaryItem,
  right: DepartmentUsageSummaryItem,
  summarySort: DepartmentUsageSummarySort,
  summaryOrder: UsageSortOrder
): number {
  const unassignedName = '\u672a\u5f52\u5c5e'
  const applyOrder = (value: number) =>
    summaryOrder === 'asc' ? value : -value
  const compareNumber = (a: number, b: number) => {
    if (a < b) {
      return -1
    }
    if (a > b) {
      return 1
    }
    return 0
  }
  const compareName = (a: string, b: string) =>
    a.localeCompare(b, undefined, { sensitivity: 'base' })
  const sortName = (item: DepartmentUsageSummaryItem) =>
    item.dept_name || unassignedName
  const compareOptionalId = (
    a: number | null | undefined,
    b: number | null | undefined
  ) => {
    if (a == null && b == null) return 0
    if (a == null) return 1
    if (b == null) return -1
    return compareNumber(a, b)
  }

  let primary = 0
  if (summarySort === 'quota') {
    primary = compareNumber(left.quota, right.quota)
  } else if (summarySort === 'users') {
    primary = compareNumber(left.user_count, right.user_count)
  } else if (summarySort === 'dept_name') {
    primary = compareName(sortName(left), sortName(right))
  } else {
    primary = compareNumber(left.request_count, right.request_count)
  }
  if (primary !== 0) return applyOrder(primary)
  if (left.request_count !== right.request_count) {
    return right.request_count - left.request_count
  }
  if (left.quota !== right.quota) {
    return right.quota - left.quota
  }
  if (left.user_count !== right.user_count) {
    return right.user_count - left.user_count
  }
  if (compareName(sortName(left), sortName(right)) !== 0) {
    return compareName(sortName(left), sortName(right))
  }
  return compareOptionalId(left.dept_id, right.dept_id)
}

export function normalizeDepartmentUsageDetail(
  detail: DepartmentUsageDetailResponse
): DepartmentUsageDetailResponse {
  const departmentIds =
    detail.scope?.department_ids ??
    (detail.dept_id == null ? [] : [detail.dept_id])
  return {
    ...detail,
    dept_name: detail.dept_name || '',
    user_ranking: (detail.user_ranking ?? []).map((item) => ({
      ...item,
      display_name: item.display_name ?? '',
    })),
    model_distribution: detail.model_distribution ?? [],
    trend: detail.trend ?? [],
    child_departments: normalizeDepartmentUsageItems(
      detail.child_departments ?? []
    ),
    scope: {
      department_ids: departmentIds,
      include_descendants: detail.scope?.include_descendants ?? false,
      metric_basis: detail.scope?.metric_basis ?? 'direct',
      department_count: detail.scope?.department_count ?? departmentIds.length,
      consuming_department_count: detail.scope?.consuming_department_count ?? 0,
      data_through: detail.scope?.data_through ?? 0,
      is_partial: detail.scope?.is_partial ?? false,
    },
    recent_logs_entry: {
      ...detail.recent_logs_entry,
      filters: {
        ...detail.recent_logs_entry.filters,
        department_id: detail.recent_logs_entry.filters.department_id ?? null,
        username: detail.recent_logs_entry.filters.username ?? '',
        username_options:
          detail.recent_logs_entry.filters.username_options ?? [],
        user_options: (detail.recent_logs_entry.filters.user_options ?? []).map(
          (item) => ({
            ...item,
            display_name: item.display_name ?? '',
          })
        ),
      },
    },
  }
}
