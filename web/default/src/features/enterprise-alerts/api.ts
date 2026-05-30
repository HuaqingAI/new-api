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
  AlertRuleResponse,
  AlertRulesResponse,
  AlertRuleUpsertRequest,
  AlertEventsResponse,
  ApiResponse,
  EnterpriseAlertsSearch,
} from './types'

export const enterpriseAlertsQueryKey = ['enterprise', 'alerts', 'events'] as const
export const enterpriseAlertRulesQueryKey = ['enterprise', 'alerts', 'rules'] as const

export function alertEventsListQueryKey(search: EnterpriseAlertsSearch) {
  return [...enterpriseAlertsQueryKey, search] as const
}

export function alertRulesListQueryKey(tenantId?: number) {
  return [...enterpriseAlertRulesQueryKey, tenantId ?? 'default'] as const
}

export async function getAlertEvents(
  search: EnterpriseAlertsSearch
): Promise<ApiResponse<AlertEventsResponse>> {
  const res = await api.get('/api/enterprise/alerts/events', {
    params: {
      ...(search.tenant_id === undefined ? {} : { tenant_id: search.tenant_id }),
      ...(search.department_id === undefined
        ? {}
        : { department_id: search.department_id }),
      ...(search.user_id === undefined ? {} : { user_id: search.user_id }),
      ...(search.username ? { username: search.username } : {}),
      ...(search.model_name ? { model_name: search.model_name } : {}),
      ...(search.risk_type ? { risk_type: search.risk_type } : {}),
      ...(search.from === undefined ? {} : { from: search.from }),
      ...(search.to === undefined ? {} : { to: search.to }),
      ...(search.page === undefined ? {} : { page: search.page }),
      ...(search.page_size === undefined ? {} : { page_size: search.page_size }),
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
