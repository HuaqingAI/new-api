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
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import {
  formatNumber,
  formatQuota,
  parseQuotaFromDollars,
  quotaUnitsToDollars,
} from '@/lib/format'
import { cn } from '@/lib/utils'

export type QuotaAmountInputMode = 'quota' | 'amount'

export type EnterpriseQuotaAmountDisplay = {
  rawQuota: number
  quotaLabel: string
  amount: number
  amountLabel: string
  auxiliaryLabel: string
}

function normalizeQuotaNumber(value: unknown): number {
  const number = Number(value)
  if (!Number.isFinite(number)) return 0
  return Math.max(0, Math.round(number))
}

function normalizeQuotaDisplayNumber(value: unknown): number {
  const number = Number(value)
  if (!Number.isFinite(number)) return 0
  return Math.round(number)
}

export function formatEnterpriseQuotaAmount(
  quota: number | string | null | undefined,
  t: (key: string, options?: Record<string, unknown>) => string
): EnterpriseQuotaAmountDisplay {
  const rawQuota = normalizeQuotaDisplayNumber(quota)
  const amountLabel = formatQuota(rawQuota)
  return {
    rawQuota,
    quotaLabel: t('{{quota}} quota', { quota: formatNumber(rawQuota) }),
    amount: quotaUnitsToDollars(rawQuota),
    amountLabel,
    auxiliaryLabel: t('Approx. {{amount}}', { amount: amountLabel }),
  }
}

export function parseEnterpriseQuotaInput(
  value: string,
  mode: QuotaAmountInputMode
): number | null {
  const trimmed = value.trim()
  if (!trimmed) return 0
  const parsed = Number(trimmed)
  if (!Number.isFinite(parsed) || parsed < 0) return null
  if (mode === 'amount') return parseQuotaFromDollars(parsed)
  if (!Number.isInteger(parsed)) return null
  return parsed
}

export function convertEnterpriseQuotaInputMode(params: {
  value: string
  from: QuotaAmountInputMode
  to: QuotaAmountInputMode
}): { value: string; quota: number | null } {
  const quota = parseEnterpriseQuotaInput(params.value, params.from)
  if (quota == null) return { value: params.value, quota: null }
  if (params.to === 'quota') return { value: String(quota), quota }
  return {
    value: formatAmountInputValue(quotaUnitsToDollars(quota)),
    quota,
  }
}

function formatAmountInputValue(value: number) {
  if (!Number.isFinite(value)) return ''
  return Number(value.toFixed(6)).toString()
}

export function QuotaAmountDisplay({
  quota,
  className,
  mutedClassName,
}: {
  quota: number | string | null | undefined
  className?: string
  mutedClassName?: string
}) {
  const { t } = useTranslation()
  const display = formatEnterpriseQuotaAmount(quota, t)

  return (
    <div className={cn('flex min-w-[120px] flex-col gap-0.5', className)}>
      <span>{display.quotaLabel}</span>
      <span className={cn('text-muted-foreground text-xs', mutedClassName)}>
        {display.auxiliaryLabel}
      </span>
    </div>
  )
}

export function QuotaAmountInput({
  value,
  onChange,
  disabled,
  ariaLabel,
  className,
}: {
  value: number | string | null | undefined
  onChange: (value: string) => void
  disabled?: boolean
  ariaLabel?: string
  className?: string
}) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<QuotaAmountInputMode>('quota')
  const [draftValue, setDraftValue] = useState(String(value ?? ''))
  const quotaValue = useMemo(() => normalizeQuotaNumber(value), [value])
  const display = formatEnterpriseQuotaAmount(quotaValue, t)

  useEffect(() => {
    const draftQuota = parseEnterpriseQuotaInput(draftValue, mode)
    if (draftQuota === quotaValue) return
    setDraftValue(
      draftQuota == null || mode === 'quota'
        ? String(value ?? '')
        : formatAmountInputValue(display.amount)
    )
  }, [display.amount, draftValue, mode, quotaValue, value])

  const handleModeChange = (nextMode: QuotaAmountInputMode) => {
    if (nextMode === mode) return
    const converted = convertEnterpriseQuotaInputMode({
      value: draftValue,
      from: mode,
      to: nextMode,
    })
    setMode(nextMode)
    setDraftValue(converted.value)
    if (converted.quota != null) {
      onChange(String(converted.quota))
    }
  }

  return (
    <div className={cn('space-y-2', className)}>
      <div className='grid grid-cols-2 rounded-md border p-0.5'>
        <button
          type='button'
          className={cn(
            'h-8 rounded-sm px-2 text-xs font-medium transition-colors',
            mode === 'quota'
              ? 'bg-primary text-primary-foreground'
              : 'text-muted-foreground hover:bg-muted'
          )}
          disabled={disabled}
          aria-pressed={mode === 'quota'}
          onClick={() => handleModeChange('quota')}
        >
          {t('Quota view')}
        </button>
        <button
          type='button'
          className={cn(
            'h-8 rounded-sm px-2 text-xs font-medium transition-colors',
            mode === 'amount'
              ? 'bg-primary text-primary-foreground'
              : 'text-muted-foreground hover:bg-muted'
          )}
          disabled={disabled}
          aria-pressed={mode === 'amount'}
          onClick={() => handleModeChange('amount')}
        >
          {t('Amount view')}
        </button>
      </div>
      <Input
        inputMode={mode === 'amount' ? 'decimal' : 'numeric'}
        value={draftValue}
        onChange={(event) => {
          const nextValue = event.target.value
          setDraftValue(nextValue)
          const nextQuota = parseEnterpriseQuotaInput(nextValue, mode)
          if (nextQuota == null) {
            onChange(event.target.value)
            return
          }
          onChange(String(nextQuota))
        }}
        disabled={disabled}
        aria-label={ariaLabel}
      />
      <div className='text-muted-foreground flex flex-wrap gap-x-3 gap-y-1 text-xs'>
        <span>{t('Stored as quota units')}</span>
        <span>{display.quotaLabel}</span>
        <span>{display.auxiliaryLabel}</span>
      </div>
    </div>
  )
}
