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
/* oxlint-disable react/only-export-components */
import { Crown, RefreshCw } from 'lucide-react'
import { useState, useEffect, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  StatusBadge,
  dotColorMap,
  textColorMap,
} from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { TitledCard } from '@/components/ui/titled-card'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  getPublicPlans,
  getSelfSubscriptionFull,
  reorderSelfSubscription,
  updateBillingPreference,
} from '@/features/subscriptions/api'
import type {
  PlanRecord,
  UserSubscription,
  UserSubscriptionRecord,
} from '@/features/subscriptions/types'
import { formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

export function getBillingPreferenceLabel(
  preference: string,
  t: (key: string) => string
): string {
  switch (preference) {
    case 'subscription_first':
      return t('Subscription First')
    case 'wallet_first':
      return t('Wallet First')
    case 'subscription_only':
      return t('Subscription Only')
    case 'wallet_only':
      return t('Wallet Only')
    default:
      return t('Unknown billing preference')
  }
}

export function getSubscriptionSourceLabel(
  source: string | undefined,
  t: (key: string) => string
): string {
  switch (source) {
    case 'enterprise_allocation':
    case 'enterprise':
      return t('Enterprise allocation')
    case 'admin':
      return t('Admin')
    case 'user':
      return t('User')
    case 'system':
      return t('System')
    case 'payment':
      return t('Payment')
    case 'manual':
      return t('Manual')
    case 'subscription':
      return t('Subscription')
    case 'wallet':
      return t('Wallet')
    default:
      return source ? t('Unknown source') : '-'
  }
}

export function getSubscriptionCardTitle(
  sub: UserSubscriptionRecord['subscription'],
  t: (key: string) => string,
  planTitle?: string
): string {
  if (planTitle) {
    return `${planTitle} · ${t('Subscription')} #${sub.id}`
  }
  if (isEnterpriseAllocationSubscription(sub)) {
    return `${t('Enterprise Allocation Wallet')} · ${t('Subscription')} #${sub.id}`
  }
  return `${t('Subscription')} #${sub.id}`
}

export function getManagedSubscriptionNote(
  sub: UserSubscriptionRecord['subscription'],
  t: (key: string) => string
): string | null {
  if (!isEnterpriseAllocationSubscription(sub)) {
    return null
  }
  return `${t('Managed by department')} · ${t('Cannot be deleted by user')}`
}

function isEnterpriseAllocationSubscription(sub: UserSubscription): boolean {
  return (
    sub.source_type === 'enterprise_allocation' ||
    sub.source_type === 'enterprise' ||
    sub.source === 'enterprise_allocation' ||
    sub.source === 'enterprise'
  )
}

function hasPositiveTimestamp(
  value: number | undefined | null
): value is number {
  return typeof value === 'number' && value > 0
}

export function getSubscriptionRemainingDays(
  sub: UserSubscription,
  nowSeconds = Date.now() / 1000
): number | null {
  if (!hasPositiveTimestamp(sub.end_time)) {
    return null
  }
  return Math.max(0, Math.ceil((sub.end_time - nowSeconds) / 86400))
}

export function getSubscriptionStatusDisplay(
  sub: UserSubscription,
  t: (key: string) => string,
  nowSeconds = Date.now() / 1000
): {
  label: string
  variant: 'success' | 'neutral'
  isActive: boolean
  isCancelled: boolean
  isExpired: boolean
} {
  const isCancelled = sub.status === 'cancelled'
  const hasEndTime = hasPositiveTimestamp(sub.end_time)
  const isEnterpriseNoExpiry =
    isEnterpriseAllocationSubscription(sub) && !hasEndTime
  const isExpired =
    !isCancelled &&
    !isEnterpriseNoExpiry &&
    (!hasEndTime || sub.end_time < nowSeconds)
  const isActive = sub.status === 'active' && !isExpired

  if (isActive) {
    return {
      label: t('Active'),
      variant: 'success',
      isActive: true,
      isCancelled: false,
      isExpired: false,
    }
  }
  if (isCancelled) {
    return {
      label: t('Cancelled'),
      variant: 'neutral',
      isActive: false,
      isCancelled: true,
      isExpired: false,
    }
  }
  return {
    label: t('Expired'),
    variant: 'neutral',
    isActive: false,
    isCancelled: false,
    isExpired: true,
  }
}

export function getSubscriptionExpiryDisplay(
  sub: UserSubscription,
  t: (key: string) => string,
  nowSeconds = Date.now() / 1000
): { label: string; value: string } {
  const status = getSubscriptionStatusDisplay(sub, t, nowSeconds)
  let label = t('Until')
  if (status.isCancelled) {
    label = t('Cancelled at')
  } else if (status.isExpired) {
    label = t('Expired at')
  }

  if (!hasPositiveTimestamp(sub.end_time)) {
    const isEnterpriseNoExpiry =
      isEnterpriseAllocationSubscription(sub) && status.isActive

    return {
      label,
      value: isEnterpriseNoExpiry ? t('Never expires') : t('No expiry set'),
    }
  }

  return {
    label,
    value: new Date(sub.end_time * 1000).toLocaleString(),
  }
}

export function isActiveSubscriptionForBilling(
  sub: UserSubscription,
  nowSeconds = Date.now() / 1000
): boolean {
  return (
    sub.status === 'active' && (sub.end_time === 0 || sub.end_time > nowSeconds)
  )
}

export function isUsableSubscriptionForDisplayFilter(
  sub: UserSubscription,
  nowSeconds = Date.now() / 1000
): boolean {
  if (!isActiveSubscriptionForBilling(sub, nowSeconds)) {
    return false
  }

  const total = Number(sub.amount_total || 0)
  if (total <= 0) {
    return true
  }
  return Number(sub.amount_used || 0) < total
}

export type SubscriptionDisplayFilter = 'all' | 'available'

export function getSubscriptionHeaderStatus(
  filter: SubscriptionDisplayFilter,
  activeCount: number,
  availableCount: number
): { count: number; labelKey: 'active' | 'Available'; showCount: boolean } {
  if (filter === 'available') {
    return {
      count: availableCount,
      labelKey: 'Available',
      showCount: true,
    }
  }

  return {
    count: activeCount,
    labelKey: 'active',
    showCount: activeCount > 0,
  }
}

export function SubscriptionPlansCard() {
  const { t } = useTranslation()

  const [plans, setPlans] = useState<PlanRecord[]>([])
  const [activeSubscriptions, setActiveSubscriptions] = useState<
    UserSubscriptionRecord[]
  >([])
  const [allSubscriptions, setAllSubscriptions] = useState<
    UserSubscriptionRecord[]
  >([])
  const [billingPreference, setBillingPreference] =
    useState('subscription_first')
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [subscriptionFilter, setSubscriptionFilter] =
    useState<SubscriptionDisplayFilter>('all')

  const fetchPlans = useCallback(async () => {
    try {
      const res = await getPublicPlans()
      if (res.success) {
        setPlans(res.data || [])
      }
    } catch {
      setPlans([])
    }
  }, [])

  const fetchSelfSubscription = useCallback(async () => {
    try {
      const res = await getSelfSubscriptionFull()
      if (res.success && res.data) {
        setBillingPreference(
          res.data.billing_preference || 'subscription_first'
        )
        setActiveSubscriptions(res.data.subscriptions || [])
        setAllSubscriptions(res.data.all_subscriptions || [])
      }
    } catch {
      // ignore
    }
  }, [])

  useEffect(() => {
    const init = async () => {
      setLoading(true)
      await Promise.all([fetchPlans(), fetchSelfSubscription()])
      setLoading(false)
    }
    init()
  }, [fetchPlans, fetchSelfSubscription])

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      await fetchSelfSubscription()
    } finally {
      setRefreshing(false)
    }
  }

  const handleSelfMove = async (
    subId: number,
    currentSortOrder: number,
    direction: 'up' | 'down'
  ) => {
    const delta = direction === 'up' ? -100 : 100
    try {
      const res = await reorderSelfSubscription({
        user_subscription_id: subId,
        target_sort_order: currentSortOrder + delta,
      })
      if (res.success) {
        toast.success(t('Subscription priority updated'))
        if (res.data) {
          setAllSubscriptions(res.data)
          setActiveSubscriptions(
            res.data.filter((item) =>
              isActiveSubscriptionForBilling(item.subscription)
            )
          )
        } else {
          await fetchSelfSubscription()
        }
      } else {
        toast.error(res.message || t('Request failed'))
      }
    } catch {
      toast.error(t('Request failed'))
    }
  }

  const handlePreferenceChange = async (pref: string) => {
    const previous = billingPreference
    setBillingPreference(pref)
    try {
      const res = await updateBillingPreference(pref)
      if (res.success) {
        toast.success(t('Updated successfully'))
        const normalized = res.data?.billing_preference || pref
        setBillingPreference(normalized)
        await fetchSelfSubscription()
      } else {
        toast.error(res.message || t('Update failed'))
        setBillingPreference(previous)
      }
    } catch {
      toast.error(t('Request failed'))
      setBillingPreference(previous)
    }
  }

  const hasActive = activeSubscriptions.length > 0
  const hasAny = allSubscriptions.length > 0
  const disablePref = !hasActive
  const isSubPref =
    billingPreference === 'subscription_first' ||
    billingPreference === 'subscription_only'
  const displayedSubscriptions = useMemo(() => {
    return allSubscriptions
      .map((sub, index) => ({
        record: sub,
        priorityRank: index + 1,
      }))
      .filter((item) => {
        if (subscriptionFilter === 'all') {
          return true
        }
        return isUsableSubscriptionForDisplayFilter(item.record.subscription)
      })
  }, [allSubscriptions, subscriptionFilter])
  const subscriptionHeaderStatus = getSubscriptionHeaderStatus(
    subscriptionFilter,
    activeSubscriptions.length,
    displayedSubscriptions.length
  )

  const planTitleMap = useMemo(() => {
    const map = new Map<number, string>()
    for (const p of plans) {
      if (p?.plan?.id) {
        map.set(p.plan.id, p.plan.title || '')
      }
    }
    return map
  }, [plans])

  const getUsagePercent = (sub: UserSubscriptionRecord) => {
    const total = Number(sub?.subscription?.amount_total || 0)
    const used = Number(sub?.subscription?.amount_used || 0)
    if (total <= 0) return 0
    return Math.round((used / total) * 100)
  }

  if (loading) {
    return (
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
          <Skeleton className='h-6 w-32' />
        </CardHeader>
        <CardContent className='space-y-4 p-3 sm:p-5'>
          <Skeleton className='h-20 w-full' />
          <div className='grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3'>
            {['first', 'second', 'third'].map((key) => (
              <Skeleton key={key} className='h-48 w-full' />
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!hasAny) {
    return null
  }

  return (
    <TitledCard
      title={t('Subscription Plans')}
      description={t('Subscribe to a plan for model access')}
      icon={<Crown className='h-4 w-4' />}
      iconTone='warning'
      disableHoverEffect
      contentClassName='space-y-4 sm:space-y-5'
    >
      {/* My subscriptions & billing preference */}
      <div className='rounded-xl border p-3 sm:p-4'>
        <div className='flex flex-wrap items-center justify-between gap-2.5 sm:gap-3'>
          <div className='flex min-w-0 flex-wrap items-center gap-2'>
            <span className='text-sm font-medium'>{t('My Subscriptions')}</span>
            <span className='flex items-center gap-1.5 text-xs font-medium'>
              <span
                className={cn(
                  'size-1.5 shrink-0 rounded-full',
                  subscriptionHeaderStatus.count > 0
                    ? dotColorMap.success
                    : dotColorMap.neutral
                )}
                aria-hidden='true'
              />
              {subscriptionHeaderStatus.showCount ? (
                <span
                  className={cn(
                    subscriptionHeaderStatus.count > 0
                      ? textColorMap.success
                      : 'text-muted-foreground'
                  )}
                >
                  {subscriptionHeaderStatus.count}{' '}
                  {t(subscriptionHeaderStatus.labelKey)}
                </span>
              ) : (
                <span className='text-muted-foreground'>{t('No Active')}</span>
              )}
              {subscriptionFilter === 'all' &&
                allSubscriptions.length > activeSubscriptions.length && (
                  <>
                    <span className='text-muted-foreground/30'>·</span>
                    <span className='text-muted-foreground'>
                      {allSubscriptions.length - activeSubscriptions.length}{' '}
                      {t('expired')}
                    </span>
                  </>
                )}
            </span>
          </div>
          <div className='flex w-full items-center gap-2 sm:w-auto'>
            <Select
              items={[
                {
                  value: 'all',
                  label: t('All subscriptions'),
                },
                {
                  value: 'available',
                  label: t('Available subscriptions'),
                },
              ]}
              value={subscriptionFilter}
              onValueChange={(v) => {
                if (v === 'all' || v === 'available') {
                  setSubscriptionFilter(v)
                }
              }}
            >
              <SelectTrigger className='h-8 flex-1 text-xs sm:w-[128px] sm:flex-none'>
                <SelectValue>
                  {subscriptionFilter === 'available'
                    ? t('Available subscriptions')
                    : t('All subscriptions')}
                </SelectValue>
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  <SelectItem value='all'>{t('All subscriptions')}</SelectItem>
                  <SelectItem value='available'>
                    {t('Available subscriptions')}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
            <Select
              items={[
                {
                  value: 'subscription_first',
                  label: (
                    <>
                      {getBillingPreferenceLabel('subscription_first', t)}
                      {disablePref ? ` (${t('No Active')})` : ''}
                    </>
                  ),
                },
                {
                  value: 'wallet_first',
                  label: getBillingPreferenceLabel('wallet_first', t),
                },
                {
                  value: 'subscription_only',
                  label: (
                    <>
                      {getBillingPreferenceLabel('subscription_only', t)}
                      {disablePref ? ` (${t('No Active')})` : ''}
                    </>
                  ),
                },
                {
                  value: 'wallet_only',
                  label: getBillingPreferenceLabel('wallet_only', t),
                },
              ]}
              value={billingPreference}
              onValueChange={(v) => v !== null && handlePreferenceChange(v)}
            >
              <SelectTrigger className='h-8 flex-1 text-xs sm:w-[140px] sm:flex-none'>
                <SelectValue>
                  {getBillingPreferenceLabel(billingPreference, t)}
                </SelectValue>
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  <SelectItem value='subscription_first' disabled={disablePref}>
                    {getBillingPreferenceLabel('subscription_first', t)}
                    {disablePref ? ` (${t('No Active')})` : ''}
                  </SelectItem>
                  <SelectItem value='wallet_first'>
                    {getBillingPreferenceLabel('wallet_first', t)}
                  </SelectItem>
                  <SelectItem value='subscription_only' disabled={disablePref}>
                    {getBillingPreferenceLabel('subscription_only', t)}
                    {disablePref ? ` (${t('No Active')})` : ''}
                  </SelectItem>
                  <SelectItem value='wallet_only'>
                    {getBillingPreferenceLabel('wallet_only', t)}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
            <Button
              variant='ghost'
              size='icon'
              className='h-8 w-8'
              onClick={handleRefresh}
              disabled={refreshing}
            >
              <RefreshCw
                className={`h-3.5 w-3.5 ${refreshing ? 'animate-spin' : ''}`}
              />
            </Button>
          </div>
        </div>

        {disablePref && isSubPref && (
          <p className='text-muted-foreground mt-2 text-xs'>
            {billingPreference === 'subscription_only'
              ? t(
                  'Preference saved as {{pref}}, but no active subscription. Requests will be rejected.',
                  { pref: t('Subscription Only') }
                )
              : t(
                  'Preference saved as {{pref}}, but no active subscription. Wallet will be used automatically.',
                  { pref: t('Subscription First') }
                )}
          </p>
        )}

        {hasAny && (
          <>
            <Separator className='my-3' />
            <div className='max-h-64 space-y-3 overflow-y-auto pr-1'>
              {displayedSubscriptions.length === 0 && (
                <p className='text-muted-foreground py-6 text-center text-xs'>
                  {t('No subscription records')}
                </p>
              )}
              {displayedSubscriptions.map((item) => {
                const sub = item.record
                const subscription = sub.subscription
                const totalAmount = Number(subscription.amount_total || 0)
                const usedAmount = Number(subscription.amount_used || 0)
                const remainAmount =
                  totalAmount > 0 ? Math.max(0, totalAmount - usedAmount) : 0
                const planTitle = planTitleMap.get(subscription.plan_id) || ''
                const remainDays = getSubscriptionRemainingDays(subscription)
                const usagePercent = getUsagePercent(sub)
                const statusDisplay = getSubscriptionStatusDisplay(
                  subscription,
                  t
                )
                const expiryDisplay = getSubscriptionExpiryDisplay(
                  subscription,
                  t
                )
                const nextResetTime = subscription?.next_reset_time ?? 0

                return (
                  <div
                    key={subscription.id}
                    className='bg-background rounded-md border p-3 text-xs'
                  >
                    <div className='flex items-center justify-between'>
                      <div className='flex items-center gap-2'>
                        <span className='font-medium'>
                          {getSubscriptionCardTitle(subscription, t, planTitle)}
                        </span>
                        <StatusBadge
                          label={statusDisplay.label}
                          variant={statusDisplay.variant}
                          copyable={false}
                        />
                      </div>
                      {statusDisplay.isActive && remainDays !== null && (
                        <span className='text-muted-foreground'>
                          {t('{{count}} days remaining', {
                            count: remainDays,
                          })}
                        </span>
                      )}
                    </div>
                    <div className='text-muted-foreground mt-1.5'>
                      {expiryDisplay.label} {expiryDisplay.value}
                    </div>
                    {statusDisplay.isActive && nextResetTime > 0 && (
                      <div className='text-muted-foreground mt-1'>
                        {t('Next reset')}:{' '}
                        {new Date(nextResetTime * 1000).toLocaleString()}
                      </div>
                    )}
                    <div className='text-muted-foreground mt-1'>
                      {t('Source')}:{' '}
                      {getSubscriptionSourceLabel(
                        subscription?.source_type || subscription?.source,
                        t
                      )}{' '}
                      · {t('Subscription Priority')}:{' '}
                      {t('No. {{rank}}', { rank: item.priorityRank })}
                    </div>
                    {getManagedSubscriptionNote(subscription, t) && (
                      <div className='text-muted-foreground mt-1'>
                        {getManagedSubscriptionNote(subscription, t)}
                      </div>
                    )}
                    <div className='mt-2 flex flex-wrap gap-2'>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          handleSelfMove(
                            subscription.id,
                            subscription.sort_order ?? 0,
                            'up'
                          )
                        }
                      >
                        {t('Move Up')}
                      </Button>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          handleSelfMove(
                            subscription.id,
                            subscription.sort_order ?? 0,
                            'down'
                          )
                        }
                      >
                        {t('Move Down')}
                      </Button>
                    </div>
                    <div className='text-muted-foreground mt-1'>
                      {t('Total Quota')}:{' '}
                      {totalAmount > 0 ? (
                        <Tooltip>
                          <TooltipTrigger
                            render={<span className='cursor-help' />}
                          >
                            {formatQuota(usedAmount)}/{formatQuota(totalAmount)}{' '}
                            · {t('Remaining')} {formatQuota(remainAmount)}
                          </TooltipTrigger>
                          <TooltipContent>
                            {t('Raw Quota')}: {usedAmount}/{totalAmount} ·{' '}
                            {t('Remaining')} {remainAmount}
                          </TooltipContent>
                        </Tooltip>
                      ) : (
                        t('Unlimited')
                      )}
                      {totalAmount > 0 && (
                        <span className='ml-2'>
                          {t('Used')} {usagePercent}%
                        </span>
                      )}
                    </div>
                    {totalAmount > 0 && statusDisplay.isActive && (
                      <Progress value={usagePercent} className='mt-2 h-1.5' />
                    )}
                  </div>
                )
              })}
            </div>
          </>
        )}
      </div>
    </TitledCard>
  )
}
