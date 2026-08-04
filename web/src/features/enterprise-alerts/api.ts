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
  AlertDeliveriesResponse,
  AlertDeliveryResendResponse,
  AlertRuleResponse,
  AlertRulesResponse,
  AlertRuleUpsertRequest,
  AlertEventsResponse,
  DepartmentRiskSummaryResponse,
  ApiResponse,
  EnterpriseAlertDeliveriesSearch,
  EnterpriseAlertsSearch,
} from './types'

export const enterpriseAlertsQueryKey = [
  'enterprise',
  'alerts',
  'events',
] as const
export const enterpriseAlertOverviewQueryKey = [
  'enterprise',
  'alerts',
  'department-summary',
] as const
export const enterpriseAlertDeliveriesQueryKey = [
  'enterprise',
  'alerts',
  'deliveries',
] as const
export const enterpriseAlertRulesQueryKey = [
  'enterprise',
  'alerts',
  'rules',
] as const

export function alertEventsListQueryKey(search: EnterpriseAlertsSearch) {
  return [...enterpriseAlertsQueryKey, search] as const
}

export function alertOverviewQueryKey(search: EnterpriseAlertsSearch) {
  return [...enterpriseAlertOverviewQueryKey, search] as const
}

export function alertRulesListQueryKey(tenantId?: number) {
  return [...enterpriseAlertRulesQueryKey, tenantId ?? 'default'] as const
}

export function alertDeliveriesListQueryKey(
  search: EnterpriseAlertDeliveriesSearch
) {
  return [...enterpriseAlertDeliveriesQueryKey, search] as const
}

export async function getAlertEvents(
  search: EnterpriseAlertsSearch
): Promise<ApiResponse<AlertEventsResponse>> {
  const res = await api.get('/api/enterprise/alerts/events', {
    params: {
      ...(search.tab ? { tab: search.tab } : {}),
      ...(search.tenant_id === undefined
        ? {}
        : { tenant_id: search.tenant_id }),
      ...(search.event_id === undefined ? {} : { event_id: search.event_id }),
      ...(search.department_id === undefined
        ? {}
        : { department_id: search.department_id }),
      ...(search.unassigned_only === undefined
        ? {}
        : { unassigned_only: search.unassigned_only }),
      ...(search.user_id === undefined ? {} : { user_id: search.user_id }),
      ...(search.username ? { username: search.username } : {}),
      ...(search.model_name ? { model_name: search.model_name } : {}),
      ...(search.risk_type ? { risk_type: search.risk_type } : {}),
      ...(search.from === undefined ? {} : { from: search.from }),
      ...(search.to === undefined ? {} : { to: search.to }),
      ...(search.page === undefined ? {} : { page: search.page }),
      ...(search.page_size === undefined
        ? {}
        : { page_size: search.page_size }),
    },
  })
  return res.data
}

export async function getDepartmentRiskSummary(
  search: EnterpriseAlertsSearch
): Promise<ApiResponse<DepartmentRiskSummaryResponse>> {
  const res = await api.get('/api/enterprise/alerts/department-summary', {
    params: {
      ...(search.tenant_id === undefined
        ? {}
        : { tenant_id: search.tenant_id }),
      ...(search.department_id === undefined
        ? {}
        : { department_id: search.department_id }),
      ...(search.include_descendants === undefined
        ? {}
        : { include_descendants: search.include_descendants }),
      ...(search.from === undefined ? {} : { from: search.from }),
      ...(search.to === undefined ? {} : { to: search.to }),
      ...(search.summary_sort === undefined
        ? {}
        : { summary_sort: search.summary_sort }),
      ...(search.summary_order === undefined
        ? {}
        : { summary_order: search.summary_order }),
    },
  })
  return res.data
}

export async function getAlertRules(
  tenantId?: number
): Promise<ApiResponse<AlertRulesResponse>> {
  const res = await api.get('/api/enterprise/alerts/rules', {
    params: {
      ...(tenantId === undefined ? {} : { tenant_id: tenantId }),
    },
  })
  return res.data
}

export async function getAlertDeliveries(
  search: EnterpriseAlertDeliveriesSearch
): Promise<ApiResponse<AlertDeliveriesResponse>> {
  const res = await api.get('/api/enterprise/alerts/deliveries', {
    params: {
      ...(search.tenant_id === undefined
        ? {}
        : { tenant_id: search.tenant_id }),
      ...(search.rule_id === undefined ? {} : { rule_id: search.rule_id }),
      ...(search.event_id === undefined ? {} : { event_id: search.event_id }),
      ...(search.manual_parent_id === undefined
        ? {}
        : { manual_parent_id: search.manual_parent_id }),
      ...(search.channel_type ? { channel_type: search.channel_type } : {}),
      ...(search.status ? { status: search.status } : {}),
      ...(search.trigger_source
        ? { trigger_source: search.trigger_source }
        : {}),
      ...(search.page === undefined ? {} : { page: search.page }),
      ...(search.page_size === undefined
        ? {}
        : { page_size: search.page_size }),
    },
  })
  return res.data
}

export async function resendAlertDelivery(
  id: number,
  tenantId?: number
): Promise<ApiResponse<AlertDeliveryResendResponse>> {
  const res = await api.post(
    `/api/enterprise/alerts/deliveries/${id}/resend`,
    null,
    {
      params: {
        ...(tenantId === undefined ? {} : { tenant_id: tenantId }),
      },
    }
  )
  return res.data
}

export async function saveAlertRule(
  payload: AlertRuleUpsertRequest
): Promise<ApiResponse<AlertRuleResponse>> {
  const res = await api.put('/api/enterprise/alerts/rules', payload)
  return res.data
}

export async function deleteAlertRule(
  id: number,
  tenantId?: number
): Promise<ApiResponse<AlertRuleResponse>> {
  const res = await api.delete(`/api/enterprise/alerts/rules/${id}`, {
    params: {
      ...(tenantId === undefined ? {} : { tenant_id: tenantId }),
    },
  })
  return res.data
}
