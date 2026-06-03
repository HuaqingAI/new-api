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
import { formatNumber } from '@/lib/format'
import type { QuotaRequestCapabilityBudgetItem } from './types'

export type QuotaRequestBudgetDisplayText = {
  departmentName: string
  identity: string
  typeLabel: string
  remainingLabel: string
  statusLabel: string
}

export function enterpriseBudgetStatusLabel(
  status: string,
  t: (key: string) => string
) {
  if (status === 'active') return t('Active')
  if (status === 'paused') return t('Paused')
  if (status === 'revoked') return t('Revoked')
  if (status === 'expired') return t('Expired')
  if (status === 'superseded') return t('Superseded')
  if (status === 'closed') return t('Closed')
  if (status === 'cancelled') return t('Cancelled')
  return status || '-'
}

export function formatBudgetType(type: string, t: (key: string) => string) {
  return type === 'balance' ? t('Balance Budget') : t('Subscription Budget')
}

export function getQuotaRequestBudgetDisplayText(
  item: QuotaRequestCapabilityBudgetItem,
  t: (key: string, options?: Record<string, unknown>) => string
): QuotaRequestBudgetDisplayText {
  return {
    departmentName: item.department_name || `#${item.department_id}`,
    identity: t('Budget #{{budgetId}}', { budgetId: item.id }),
    typeLabel: formatBudgetType(item.type, t),
    remainingLabel: t('Remaining {{remaining}}', {
      remaining: formatNumber(item.remaining),
    }),
    statusLabel: enterpriseBudgetStatusLabel(item.status, t),
  }
}

