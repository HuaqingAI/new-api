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
  DepartmentUsageOverviewResponse,
  DepartmentUsagePeersResponse,
  DepartmentUsageReportConfigResponse,
  DepartmentUsageSummaryResponse,
  DepartmentUsageSummarySort,
  UsageSortOrder,
} from './types'

export const enterpriseUsageQueryKey = ['enterprise', 'usage'] as const

export function departmentSummaryQueryKey(
  from: number,
  to: number,
  tenantId?: number,
  departmentId?: number,
  includeDescendants?: boolean,
  summarySort?: DepartmentUsageSummarySort,
  summaryOrder?: UsageSortOrder
) {
  return [
    ...enterpriseUsageQueryKey,
    'department-summary',
    {
      from,
      to,
      tenantId,
      departmentId,
      includeDescendants,
      summarySort,
      summaryOrder,
    },
  ] as const
}

export function departmentDetailQueryKey(
  deptId: number,
  from: number,
  to: number,
  tenantId?: number,
  includeDescendants?: boolean
) {
  return [
    ...enterpriseUsageQueryKey,
    'department-detail',
    tenantId === undefined
      ? { deptId, from, to, includeDescendants }
      : { deptId, from, to, tenantId, includeDescendants },
  ] as const
}

export function departmentOverviewQueryKey(
  from: number,
  to: number,
  tenantId?: number,
  summarySort?: DepartmentUsageSummarySort,
  summaryOrder?: UsageSortOrder
) {
  return [
    ...enterpriseUsageQueryKey,
    'department-overview',
    { from, to, tenantId, summarySort, summaryOrder },
  ] as const
}

export function departmentPeersQueryKey(
  departmentId: number,
  from: number,
  to: number,
  tenantId?: number,
  includeDescendants?: boolean,
  summarySort?: DepartmentUsageSummarySort,
  summaryOrder?: UsageSortOrder
) {
  return [
    ...enterpriseUsageQueryKey,
    'department-peers',
    {
      departmentId,
      from,
      to,
      tenantId,
      includeDescendants,
      summarySort,
      summaryOrder,
    },
  ] as const
}

export async function getDepartmentUsageSummary(params: {
  from: number
  to: number
  tenantId?: number
  departmentId?: number
  includeDescendants?: boolean
  summarySort?: DepartmentUsageSummarySort
  summaryOrder?: UsageSortOrder
}): Promise<ApiResponse<DepartmentUsageSummaryResponse>> {
  const res = await api.get('/api/enterprise/usage/department-summary', {
    params: {
      from: params.from,
      to: params.to,
      ...(params.tenantId === undefined ? {} : { tenant_id: params.tenantId }),
      ...(params.departmentId === undefined
        ? {}
        : { department_id: params.departmentId }),
      ...(params.includeDescendants === undefined
        ? {}
        : { include_descendants: params.includeDescendants }),
      ...(params.summarySort === undefined
        ? {}
        : { summary_sort: params.summarySort }),
      ...(params.summaryOrder === undefined
        ? {}
        : { summary_order: params.summaryOrder }),
    },
  })
  return res.data
}

export async function getDepartmentUsageOverview(params: {
  from: number
  to: number
  tenantId?: number
  summarySort?: DepartmentUsageSummarySort
  summaryOrder?: UsageSortOrder
}): Promise<ApiResponse<DepartmentUsageOverviewResponse>> {
  const res = await api.get('/api/enterprise/usage/department-overview', {
    params: {
      from: params.from,
      to: params.to,
      ...(params.tenantId === undefined ? {} : { tenant_id: params.tenantId }),
      ...(params.summarySort === undefined
        ? {}
        : { summary_sort: params.summarySort }),
      ...(params.summaryOrder === undefined
        ? {}
        : { summary_order: params.summaryOrder }),
    },
  })
  return res.data
}

export async function getDepartmentUsagePeers(params: {
  departmentId: number
  from: number
  to: number
  tenantId?: number
  includeDescendants?: boolean
  summarySort?: DepartmentUsageSummarySort
  summaryOrder?: UsageSortOrder
}): Promise<ApiResponse<DepartmentUsagePeersResponse>> {
  const res = await api.get('/api/enterprise/usage/department-peers', {
    params: {
      department_id: params.departmentId,
      from: params.from,
      to: params.to,
      ...(params.tenantId === undefined ? {} : { tenant_id: params.tenantId }),
      ...(params.includeDescendants === undefined
        ? {}
        : { include_descendants: params.includeDescendants }),
      ...(params.summarySort === undefined
        ? {}
        : { summary_sort: params.summarySort }),
      ...(params.summaryOrder === undefined
        ? {}
        : { summary_order: params.summaryOrder }),
    },
  })
  return res.data
}

export async function exportDepartmentUsageCSV(params: {
  from: number
  to: number
  tenantId?: number
  departmentId?: number
  includeDescendants?: boolean
  summarySort?: DepartmentUsageSummarySort
  summaryOrder?: UsageSortOrder
}) {
  const res = await api.get('/api/enterprise/usage/export', {
    params: {
      from: params.from,
      to: params.to,
      ...(params.tenantId === undefined ? {} : { tenant_id: params.tenantId }),
      ...(params.departmentId === undefined
        ? {}
        : { department_id: params.departmentId }),
      ...(params.includeDescendants === undefined
        ? {}
        : { include_descendants: params.includeDescendants }),
      ...(params.summarySort === undefined
        ? {}
        : { summary_sort: params.summarySort }),
      ...(params.summaryOrder === undefined
        ? {}
        : { summary_order: params.summaryOrder }),
    },
    responseType: 'blob',
    skipBusinessError: true,
  })

  const blob = res.data as Blob
  const contentType = String(
    res.headers['content-type'] ?? blob.type ?? ''
  ).toLowerCase()
  if (contentType.includes('application/json')) {
    const text = await blob.text()
    try {
      const payload = JSON.parse(text) as {
        success?: boolean
        message?: string
      }
      throw new Error(payload.message || 'Request failed')
    } catch (error) {
      if (error instanceof Error) {
        throw error
      }
      throw new Error('Request failed')
    }
  }

  const disposition = String(res.headers['content-disposition'] ?? '')
  const match = disposition.match(/filename="?([^"]+)"?/)
  return {
    blob,
    fileName: match?.[1] ?? 'usage-department.csv',
  }
}

export async function getDepartmentUsageDetail(params: {
  deptId: number
  from: number
  to: number
  tenantId?: number
  includeDescendants?: boolean
}): Promise<ApiResponse<DepartmentUsageDetailResponse>> {
  const res = await api.get('/api/enterprise/usage/department-detail', {
    params: {
      dept_id: params.deptId,
      from: params.from,
      to: params.to,
      ...(params.tenantId === undefined ? {} : { tenant_id: params.tenantId }),
      ...(params.includeDescendants === undefined
        ? {}
        : { include_descendants: params.includeDescendants }),
    },
  })
  return res.data
}

export const departmentUsageReportQueryKey = [
  ...enterpriseUsageQueryKey,
  'report-config',
] as const

export async function getDepartmentUsageReportConfig(params?: {
  tenantId?: number
}): Promise<ApiResponse<DepartmentUsageReportConfigResponse>> {
  const res = await api.get('/api/enterprise/usage/reports', {
    params:
      params?.tenantId === undefined ? {} : { tenant_id: params.tenantId },
  })
  return res.data
}

export async function saveDepartmentUsageReportConfig(params: {
  tenantId?: number
  receivers: string[]
  frequency: 'daily' | 'weekly' | 'monthly'
  rangeType: 'today' | 'last7d' | 'last30d'
  enabled: boolean
}): Promise<ApiResponse<DepartmentUsageReportConfigResponse>> {
  const res = await api.put('/api/enterprise/usage/reports', {
    ...(params.tenantId === undefined ? {} : { tenant_id: params.tenantId }),
    receivers: params.receivers,
    frequency: params.frequency,
    range_type: params.rangeType,
    enabled: params.enabled,
  })
  return res.data
}
