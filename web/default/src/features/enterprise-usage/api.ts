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
  ApiResponse,
  DepartmentUsageDetailResponse,
  DepartmentUsageSummaryResponse,
} from './types'

export const enterpriseUsageQueryKey = ['enterprise', 'usage'] as const

export function departmentSummaryQueryKey(
  from: number,
  to: number,
  tenantId?: number
) {
  return [
    ...enterpriseUsageQueryKey,
    'department-summary',
    tenantId === undefined ? { from, to } : { from, to, tenantId },
  ] as const
}

export function departmentDetailQueryKey(
  deptId: number,
  from: number,
  to: number,
  tenantId?: number
) {
  return [
    ...enterpriseUsageQueryKey,
    'department-detail',
    tenantId === undefined
      ? { deptId, from, to }
      : { deptId, from, to, tenantId },
  ] as const
}

export async function getDepartmentUsageSummary(params: {
  from: number
  to: number
  tenantId?: number
}): Promise<ApiResponse<DepartmentUsageSummaryResponse>> {
  const res = await api.get('/api/enterprise/usage/department-summary', {
    params: {
      from: params.from,
      to: params.to,
      ...(params.tenantId === undefined ? {} : { tenant_id: params.tenantId }),
    },
  })
  return res.data
}

export async function getDepartmentUsageDetail(params: {
  deptId: number
  from: number
  to: number
  tenantId?: number
}): Promise<ApiResponse<DepartmentUsageDetailResponse>> {
  const res = await api.get('/api/enterprise/usage/department-detail', {
    params: {
      dept_id: params.deptId,
      from: params.from,
      to: params.to,
      ...(params.tenantId === undefined ? {} : { tenant_id: params.tenantId }),
    },
  })
  return res.data
}
