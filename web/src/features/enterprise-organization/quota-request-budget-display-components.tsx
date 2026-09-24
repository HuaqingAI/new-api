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
import { useTranslation } from 'react-i18next'

import {
  getQuotaRequestBudgetDisplayText,
  getQuotaRequestBudgetTriggerLabel,
} from './quota-request-budget-display'
import type { DepartmentBudgetItem } from './types'

export function QuotaRequestBudgetOption({
  item,
}: {
  item: DepartmentBudgetItem
}) {
  const { t } = useTranslation()
  const display = getQuotaRequestBudgetDisplayText(item, t)
  return (
    <div className='min-w-0 space-y-1 text-left whitespace-normal'>
      <div className='flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1'>
        <span className='truncate font-medium'>{display.departmentName}</span>
        <span className='text-muted-foreground shrink-0 text-xs'>
          {display.identity}
        </span>
      </div>
      <div className='text-muted-foreground flex flex-wrap gap-x-3 gap-y-1 text-xs'>
        <span>{display.typeLabel}</span>
        <span>{display.remainingLabel}</span>
        <span>{display.remainingAmountLabel}</span>
        <span>{display.statusLabel}</span>
      </div>
    </div>
  )
}

export function QuotaRequestBudgetSummary({
  item,
}: {
  item: DepartmentBudgetItem
}) {
  const { t } = useTranslation()
  const display = getQuotaRequestBudgetDisplayText(item, t)
  return (
    <div className='mt-1 space-y-1'>
      <div className='text-sm font-medium'>
        {getQuotaRequestBudgetTriggerLabel(item, t)}
      </div>
      <div className='text-muted-foreground flex flex-wrap gap-x-3 gap-y-1 text-xs'>
        <span>{display.typeLabel}</span>
        <span>{display.remainingLabel}</span>
        <span>{display.remainingAmountLabel}</span>
        <span>{display.statusLabel}</span>
      </div>
    </div>
  )
}
