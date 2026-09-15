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

import { departmentDetailQueryKey, getDepartmentUsageDetail } from '../api'
import { normalizeDepartmentUsageDetail } from '../lib/usage-normalizers'

export function useDepartmentUsageDetail(params: {
  deptId?: number
  from: number
  to: number
  tenantId?: number
  includeDescendants?: boolean
}) {
  return useQuery({
    enabled: typeof params.deptId === 'number',
    queryKey:
      typeof params.deptId === 'number'
        ? departmentDetailQueryKey(
            params.deptId,
            params.from,
            params.to,
            params.tenantId,
            params.includeDescendants
          )
        : ['enterprise', 'usage', 'department-detail', 'disabled'],
    queryFn: async () => {
      const response = await getDepartmentUsageDetail({
        deptId: params.deptId as number,
        from: params.from,
        to: params.to,
        tenantId: params.tenantId,
        includeDescendants: params.includeDescendants,
      })
      if (!response.success) {
        throw new Error(response.message || 'Request failed')
      }
      return normalizeDepartmentUsageDetail(response.data)
    },
  })
}
