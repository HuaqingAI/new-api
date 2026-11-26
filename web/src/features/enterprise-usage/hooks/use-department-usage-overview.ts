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
import { useQuery } from '@tanstack/react-query'

import { departmentOverviewQueryKey, getDepartmentUsageOverview } from '../api'
import { normalizeDepartmentUsageItems } from '../lib/usage-normalizers'
import type { DepartmentUsageSummarySort, UsageSortOrder } from '../types'

export function useDepartmentUsageOverview(params: {
  from: number
  to: number
  tenantId?: number
  summarySort?: DepartmentUsageSummarySort
  summaryOrder?: UsageSortOrder
}) {
  return useQuery({
    queryKey: departmentOverviewQueryKey(
      params.from,
      params.to,
      params.tenantId,
      params.summarySort,
      params.summaryOrder
    ),
    queryFn: async () => {
      const response = await getDepartmentUsageOverview(params)
      if (!response.success) {
        throw new Error(response.message || 'Request failed')
      }
      return {
        metrics: response.data?.metrics,
        items: normalizeDepartmentUsageItems(
          response.data?.items ?? [],
          params.summarySort,
          params.summaryOrder
        ),
        secondLevelItems: normalizeDepartmentUsageItems(
          response.data?.second_level_items ?? [],
          params.summarySort,
          params.summaryOrder
        ),
        trend: response.data?.trend ?? [],
        isPartial: response.data?.is_partial ?? false,
      }
    },
  })
}
