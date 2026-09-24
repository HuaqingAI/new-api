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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  AlertCircle,
  Building2,
  CheckCircle2,
  Play,
  RefreshCw,
  Save,
  ShieldCheck,
  TriangleAlert,
  Wifi,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { ROLE } from '@/lib/roles'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import {
  enterpriseDingTalkQueryKey,
  enterpriseDingTalkSyncConflictsQueryKey,
  enterpriseDingTalkSyncLogsQueryKey,
  bindDingTalkSyncConflictCandidate,
  getDingTalkConfig,
  getDingTalkSyncTask,
  listDingTalkSyncConflicts,
  listDingTalkSyncLogs,
  saveDingTalkConfig,
  startDingTalkFullSync,
  testDingTalkConnectivity,
} from './api'
import type {
  DingTalkConfig,
  DingTalkConnectivityCode,
  DingTalkConnectivityResult,
  DingTalkSyncConflict,
  DingTalkSyncLog,
  DingTalkSyncTask,
} from './types'

const dingTalkConfigSchema = (t: (key: string) => string) =>
  z
    .object({
      corp_id: z.string(),
      app_key: z.string(),
      app_secret: z.string(),
      callback_url: z.string(),
      sync_scope: z.string(),
      login_enabled: z.boolean(),
      sync_enabled: z.boolean(),
      auto_sync_on_login: z.boolean(),
      scheduled_full_sync_enabled: z.boolean(),
      scheduled_full_sync_cron: z.string().max(128),
      scheduled_full_sync_timezone: z.string().max(64),
    })
    .superRefine((values, ctx) => {
      const enabling =
        values.login_enabled ||
        values.sync_enabled ||
        values.auto_sync_on_login ||
        values.scheduled_full_sync_enabled
      const callbackUrl = values.callback_url.trim()
      if (callbackUrl) {
        try {
          const parsed = new URL(callbackUrl)
          if (parsed.protocol !== 'https:') {
            ctx.addIssue({
              code: 'custom',
              path: ['callback_url'],
              message: t('DingTalk callback URL must use HTTPS'),
            })
          }
        } catch {
          ctx.addIssue({
            code: 'custom',
            path: ['callback_url'],
            message: t('Enter a valid callback URL'),
          })
        }
      }
      if (!enabling) return
      if (!values.corp_id.trim()) {
        ctx.addIssue({
          code: 'custom',
          path: ['corp_id'],
          message: t('Corp ID is required when enabling DingTalk'),
        })
      }
      if (!values.app_key.trim()) {
        ctx.addIssue({
          code: 'custom',
          path: ['app_key'],
          message: t('App key is required when enabling DingTalk'),
        })
      }
      if (!callbackUrl) {
        ctx.addIssue({
          code: 'custom',
          path: ['callback_url'],
          message: t('Callback URL is required when enabling DingTalk'),
        })
      }
    })

type DingTalkConfigFormValues = z.infer<ReturnType<typeof dingTalkConfigSchema>>

const emptyConfig: DingTalkConfig = {
  id: 0,
  tenant_id: 0,
  corp_id: '',
  app_key: '',
  callback_url: '',
  sync_scope: '',
  login_enabled: false,
  sync_enabled: false,
  auto_sync_on_login: false,
  scheduled_full_sync_enabled: false,
  scheduled_full_sync_cron: '0 0 * * *',
  scheduled_full_sync_timezone: 'Asia/Shanghai',
  scheduled_full_sync_next_run_at: 0,
  scheduled_full_sync_last_run_at: 0,
  scheduled_full_sync_last_task_id: 0,
  scheduled_full_sync_last_status: '',
  scheduled_full_sync_last_error: '',
  scheduled_full_sync_revision: 1,
  scheduled_full_sync_updated_at: 0,
  has_app_secret: false,
  created_at: 0,
  updated_at: 0,
}

function configToFormValues(config: DingTalkConfig): DingTalkConfigFormValues {
  return {
    corp_id: config.corp_id ?? '',
    app_key: config.app_key ?? '',
    app_secret: '',
    callback_url: config.callback_url ?? '',
    sync_scope: config.sync_scope ?? '',
    login_enabled: Boolean(config.login_enabled),
    sync_enabled: Boolean(config.sync_enabled),
    auto_sync_on_login: Boolean(config.auto_sync_on_login),
    scheduled_full_sync_enabled: Boolean(config.scheduled_full_sync_enabled),
    scheduled_full_sync_cron: config.scheduled_full_sync_cron || '0 0 * * *',
    scheduled_full_sync_timezone: config.scheduled_full_sync_timezone || 'Asia/Shanghai',
  }
}

export function EnterpriseDingTalk() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const userRole = useAuthStore((state) => state.auth.user?.role ?? ROLE.GUEST)
  const canEdit = userRole >= ROLE.SUPER_ADMIN

  const query = useQuery({
    queryKey: enterpriseDingTalkQueryKey,
    queryFn: async () => {
      const result = await getDingTalkConfig()
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data ?? emptyConfig
    },
  })

  const config = query.data ?? emptyConfig
  const form = useForm<DingTalkConfigFormValues>({
    resolver: zodResolver(dingTalkConfigSchema(t)),
    defaultValues: configToFormValues(config),
  })
  const formValues = useWatch({ control: form.control })

  useEffect(() => {
    if (!query.data) return
    form.reset(configToFormValues(query.data))
  }, [form, query.data])

  const mutation = useMutation({
    mutationFn: async (values: DingTalkConfigFormValues) => {
      if (
        (values.login_enabled ||
          values.sync_enabled ||
          values.scheduled_full_sync_enabled) &&
        !config.has_app_secret
      ) {
        if (!values.app_secret.trim()) {
          throw new Error(t('App secret is required when enabling DingTalk'))
        }
      }
      const payload = {
        corp_id: values.corp_id.trim(),
        app_key: values.app_key.trim(),
        callback_url: values.callback_url.trim(),
        sync_scope: values.sync_scope.trim(),
        login_enabled: values.login_enabled,
        sync_enabled: values.sync_enabled,
        auto_sync_on_login: values.auto_sync_on_login,
        scheduled_full_sync_enabled: values.scheduled_full_sync_enabled,
        scheduled_full_sync_cron: values.scheduled_full_sync_cron.trim(),
        scheduled_full_sync_timezone: values.scheduled_full_sync_timezone.trim(),
      }
      const appSecret = values.app_secret.trim()
      const result = await saveDingTalkConfig(
        appSecret ? { ...payload, app_secret: appSecret } : payload
      )
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data ?? emptyConfig
    },
    onSuccess: async (saved) => {
      queryClient.setQueryData(enterpriseDingTalkQueryKey, saved)
      form.reset(configToFormValues(saved))
      await queryClient.invalidateQueries({
        queryKey: enterpriseDingTalkQueryKey,
      })
      toast.success(t('DingTalk configuration saved'))
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const connectivityMutation = useMutation({
    mutationFn: async () => {
      const result = await testDingTalkConnectivity()
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data
    },
    onSuccess: (result) => {
      if (!result) return
      if (result.code === 'auth_success') {
        toast.success(t('DingTalk connectivity test passed'))
      } else {
        toast.warning(t('DingTalk connectivity test needs attention'))
      }
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const syncLogsQuery = useQuery({
    queryKey: enterpriseDingTalkSyncLogsQueryKey,
    enabled: canEdit,
    queryFn: async () => {
      const result = await listDingTalkSyncLogs({ page: 1, page_size: 8 })
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data
    },
  })

  const syncConflictsQuery = useQuery({
    queryKey: enterpriseDingTalkSyncConflictsQueryKey,
    enabled: canEdit,
    queryFn: async () => {
      const result = await listDingTalkSyncConflicts({
        status: 'pending',
        page: 1,
        page_size: 5,
      })
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data
    },
  })

  const syncMutation = useMutation({
    mutationFn: async () => {
      const result = await startDingTalkFullSync(true)
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      if (!result.data) throw new Error(t('Missing sync task status'))
      return result.data
    },
    onSuccess: async (task) => {
      if (task.status === 'succeeded') {
        toast.success(t('DingTalk sync completed'))
      } else if (task.status === 'failed') {
        toast.warning(t('DingTalk sync completed with failures'))
      } else {
        toast.success(t('DingTalk sync started'))
      }
      await queryClient.invalidateQueries({
        queryKey: enterpriseDingTalkSyncLogsQueryKey,
      })
      await queryClient.invalidateQueries({
        queryKey: enterpriseDingTalkSyncConflictsQueryKey,
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const bindConflictMutation = useMutation({
    mutationFn: async (conflict: DingTalkSyncConflict) => {
      const result = await bindDingTalkSyncConflictCandidate(
        conflict.id,
        conflict.candidate_user_id
      )
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data
    },
    onSuccess: async () => {
      toast.success(t('DingTalk sync conflict resolved'))
      await queryClient.invalidateQueries({
        queryKey: enterpriseDingTalkSyncConflictsQueryKey,
      })
      await queryClient.invalidateQueries({
        queryKey: enterpriseDingTalkSyncLogsQueryKey,
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const taskQuery = useQuery({
    queryKey: ['enterprise', 'dingtalk', 'sync', 'task', syncMutation.data?.id],
    enabled: Boolean(syncMutation.data?.id),
    queryFn: async () => {
      const taskId = syncMutation.data?.id
      if (!taskId) return null
      const result = await getDingTalkSyncTask(taskId)
      if (!result.success)
        throw new Error(result.message || t('Request failed'))
      return result.data ?? null
    },
  })

  const onSubmit = (values: DingTalkConfigFormValues) => {
    mutation.mutate(values)
  }

  const canTestConnectivity =
    canEdit &&
    Boolean(config.has_app_secret) &&
    Boolean((formValues.corp_id ?? '').trim()) &&
    Boolean((formValues.app_key ?? '').trim()) &&
    Boolean((formValues.callback_url ?? '').trim()) &&
    !form.formState.isDirty

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('DingTalk Integration')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          size='sm'
          onClick={() => query.refetch()}
          disabled={query.isFetching}
        >
          <RefreshCw className='size-4' />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='mx-auto flex max-w-5xl flex-col gap-4'>
          {!canEdit ? (
            <Alert variant='destructive'>
              <AlertCircle className='size-4' />
              <AlertTitle>{t('Root access required')}</AlertTitle>
              <AlertDescription>
                {t(
                  'Only root administrators can edit DingTalk app credentials.'
                )}
              </AlertDescription>
            </Alert>
          ) : null}

          {query.isLoading ? (
            <DingTalkConfigSkeleton />
          ) : (
            <Form {...form}>
              <form
                className='flex flex-col gap-4'
                autoComplete='off'
                onSubmit={form.handleSubmit(onSubmit)}
              >
                <Card>
                  <CardHeader className='gap-2'>
                    <div className='flex flex-wrap items-center justify-between gap-2'>
                      <div>
                        <CardTitle>{t('Enterprise App')}</CardTitle>
                        <CardDescription>
                          {t(
                            'Configure the DingTalk app used for login and address book sync.'
                          )}
                        </CardDescription>
                      </div>
                      <Badge
                        variant={
                          config.has_app_secret ? 'secondary' : 'outline'
                        }
                      >
                        {config.has_app_secret
                          ? t('Secret saved')
                          : t('Secret not set')}
                      </Badge>
                    </div>
                  </CardHeader>
                  <CardContent className='grid gap-6'>
                    <div className='grid gap-6 md:grid-cols-2'>
                      <FormField
                        control={form.control}
                        name='corp_id'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Corp ID')}</FormLabel>
                            <FormControl>
                              <Input
                                autoComplete='off'
                                placeholder='dingxxxxxxxx'
                                disabled={!canEdit}
                                {...field}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name='app_key'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('App Key')}</FormLabel>
                            <FormControl>
                              <Input
                                autoComplete='off'
                                placeholder='dingxxxxxxxx'
                                disabled={!canEdit}
                                {...field}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </div>

                    <FormField
                      control={form.control}
                      name='app_secret'
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('App Secret')}</FormLabel>
                          <FormControl>
                            <Input
                              autoComplete='new-password'
                              type='password'
                              placeholder={
                                config.has_app_secret
                                  ? t('Leave blank to keep the saved secret')
                                  : t('Enter DingTalk app secret')
                              }
                              disabled={!canEdit}
                              {...field}
                            />
                          </FormControl>
                          <FormDescription>
                            {t(
                              'Saved secrets are never returned by the API or stored in page state.'
                            )}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name='callback_url'
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('Callback URL')}</FormLabel>
                          <FormControl>
                            <Input
                              autoComplete='off'
                              placeholder='https://example.com/api/oauth/dingtalk'
                              disabled={!canEdit}
                              {...field}
                            />
                          </FormControl>
                          <FormDescription>
                            {t(
                              'Use the same HTTPS callback URL in the DingTalk developer console.'
                            )}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name='sync_scope'
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('Sync Scope')}</FormLabel>
                          <FormControl>
                            <Textarea
                              className='min-h-24'
                              placeholder={t(
                                'One department ID or rule per line'
                              )}
                              disabled={!canEdit}
                              {...field}
                            />
                          </FormControl>
                          <FormDescription>
                            {t(
                              'Limit address book sync to selected DingTalk departments or leave blank for all.'
                            )}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <div className='flex flex-wrap items-start justify-between gap-3'>
                      <div>
                        <CardTitle>{t('Enablement')}</CardTitle>
                        <CardDescription>
                          {t(
                            'Credentials and a valid callback URL are required before enabling either switch.'
                          )}
                        </CardDescription>
                      </div>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        disabled={
                          !canTestConnectivity || connectivityMutation.isPending
                        }
                        onClick={() => connectivityMutation.mutate()}
                      >
                        <Wifi className='size-4' />
                        {connectivityMutation.isPending
                          ? t('Testing...')
                          : t('Test connection')}
                      </Button>
                    </div>
                  </CardHeader>
                  <CardContent className='grid gap-4'>
                    {connectivityMutation.data ? (
                      <EnterpriseDingTalkConnectivityResult
                        result={connectivityMutation.data}
                      />
                    ) : null}
                    {!canTestConnectivity && canEdit ? (
                      <p className='text-muted-foreground text-sm'>
                        {form.formState.isDirty
                          ? t(
                              'Save the latest DingTalk settings before testing connectivity.'
                            )
                          : t(
                              'Save valid DingTalk credentials before testing connectivity.'
                            )}
                      </p>
                    ) : null}
                    <div className='grid gap-4 md:grid-cols-3'>
                      <FormField
                        control={form.control}
                        name='login_enabled'
                        render={({ field }) => (
                          <ToggleField
                            label={t('Enable DingTalk login')}
                            description={t(
                              'Allow employees to sign in with DingTalk after OAuth is configured.'
                            )}
                            checked={field.value}
                            disabled={!canEdit}
                            onCheckedChange={field.onChange}
                          />
                        )}
                      />
                      <FormField
                        control={form.control}
                        name='sync_enabled'
                        render={({ field }) => (
                          <ToggleField
                            label={t('Enable address book sync')}
                            description={t(
                              'Allow manual and scheduled DingTalk department synchronization.'
                            )}
                            checked={field.value}
                            disabled={!canEdit}
                            onCheckedChange={field.onChange}
                          />
                        )}
                      />
                      <FormField
                        control={form.control}
                        name='auto_sync_on_login'
                        render={({ field }) => (
                          <ToggleField
                            label={t('Auto-sync new employees on login')}
                            description={t(
                              'Verify and add new in-scope employees during DingTalk login. Existing bindings keep working when this is off.'
                            )}
                            checked={field.value}
                            disabled={
                              !canEdit ||
                              !formValues.login_enabled ||
                              !formValues.sync_enabled
                            }
                            onCheckedChange={field.onChange}
                          />
                        )}
                      />
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle>{t('Scheduled full sync')}</CardTitle>
                    <CardDescription>
                      {t('Run a complete DingTalk address book sync on a cron schedule.')}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className='grid gap-4'>
                    <FormField
                      control={form.control}
                      name='scheduled_full_sync_enabled'
                      render={({ field }) => (
                        <ToggleField
                          label={t('Enable scheduled full sync')}
                          description={t('The schedule requires address book sync to be enabled.')}
                          checked={field.value}
                          disabled={!canEdit || !formValues.sync_enabled}
                          onCheckedChange={field.onChange}
                        />
                      )}
                    />
                    <div className='grid gap-4 md:grid-cols-2'>
                      <FormField
                        control={form.control}
                        name='scheduled_full_sync_cron'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Cron expression')}</FormLabel>
                            <FormControl>
                              <Input placeholder='0 0 * * *' disabled={!canEdit} {...field} />
                            </FormControl>
                            <FormDescription>{t('Five fields: minute hour day month weekday.')}</FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name='scheduled_full_sync_timezone'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Timezone')}</FormLabel>
                            <FormControl>
                              <Input placeholder='Asia/Shanghai' disabled={!canEdit} {...field} />
                            </FormControl>
                            <FormDescription>{t('Use an IANA timezone such as Asia/Shanghai or UTC.')}</FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </div>
                    {config.scheduled_full_sync_enabled ? (
                      <div className='text-muted-foreground text-sm'>
                        {(config.scheduled_full_sync_next_run_at ?? 0) > 0
                          ? `${t('Next run')}: ${new Date((config.scheduled_full_sync_next_run_at ?? 0) * 1000).toLocaleString()}`
                          : t('No next run scheduled')}
                        {config.scheduled_full_sync_last_status
                          ? ` · ${t('Last status')}: ${config.scheduled_full_sync_last_status}`
                          : ''}
                      </div>
                    ) : null}
                  </CardContent>
                </Card>

                <div className='flex flex-wrap items-center justify-between gap-3'>
                  <Button
                    variant='outline'
                    render={<Link to='/enterprise-organization' />}
                  >
                    <Building2 className='size-4' />
                    {t('Enterprise Organization')}
                  </Button>
                  <Button
                    type='submit'
                    disabled={!canEdit || mutation.isPending}
                  >
                    <Save className='size-4' />
                    {mutation.isPending ? t('Saving...') : t('Save')}
                  </Button>
                </div>

                <EnterpriseDingTalkSyncPanel
                  canEdit={canEdit}
                  config={config}
                  task={taskQuery.data ?? syncMutation.data ?? null}
                  logs={syncLogsQuery.data?.items ?? []}
                  conflicts={syncConflictsQuery.data?.items ?? []}
                  logsLoading={
                    syncLogsQuery.isLoading || syncLogsQuery.isFetching
                  }
                  conflictsLoading={
                    syncConflictsQuery.isLoading ||
                    syncConflictsQuery.isFetching
                  }
                  syncing={syncMutation.isPending}
                  resolvingConflictId={
                    bindConflictMutation.isPending
                      ? (bindConflictMutation.variables?.id ?? null)
                      : null
                  }
                  onStartSync={() => syncMutation.mutate()}
                  onRefreshLogs={() => {
                    syncLogsQuery.refetch()
                    syncConflictsQuery.refetch()
                  }}
                  onBindCandidate={(conflict) =>
                    bindConflictMutation.mutate(conflict)
                  }
                />
              </form>
            </Form>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

export function EnterpriseDingTalkSyncPanel({
  canEdit,
  config,
  task,
  logs,
  conflicts,
  logsLoading,
  conflictsLoading,
  syncing,
  resolvingConflictId,
  onStartSync,
  onRefreshLogs,
  onBindCandidate,
}: {
  canEdit: boolean
  config: DingTalkConfig
  task: DingTalkSyncTask | null
  logs: DingTalkSyncLog[]
  conflicts: DingTalkSyncConflict[]
  logsLoading: boolean
  conflictsLoading: boolean
  syncing: boolean
  resolvingConflictId?: number | null
  onStartSync: () => void
  onRefreshLogs: () => void
  onBindCandidate?: (conflict: DingTalkSyncConflict) => void
}) {
  const { t } = useTranslation()
  const canSync = canEdit && config.sync_enabled && config.has_app_secret

  return (
    <Card>
      <CardHeader>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div>
            <CardTitle>{t('Address Book Sync')}</CardTitle>
            <CardDescription>
              {t(
                'Start a DingTalk full sync and review the latest task result.'
              )}
            </CardDescription>
          </div>
          <div className='flex flex-wrap gap-2'>
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={logsLoading}
              onClick={onRefreshLogs}
            >
              <RefreshCw className='size-4' />
              {t('Refresh')}
            </Button>
            <Button
              type='button'
              size='sm'
              disabled={!canSync || syncing}
              onClick={onStartSync}
            >
              <Play className='size-4' />
              {syncing ? t('Syncing...') : t('Start full sync')}
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className='grid gap-4'>
        {!canSync ? (
          <p className='text-muted-foreground text-sm'>
            {t(
              'Enable address book sync and save valid DingTalk credentials before starting a sync.'
            )}
          </p>
        ) : null}
        {task ? <DingTalkSyncTaskSummary task={task} /> : null}
        <DingTalkSyncConflictList
          conflicts={conflicts}
          loading={conflictsLoading}
          resolvingConflictId={resolvingConflictId}
          onBindCandidate={onBindCandidate}
        />
        <DingTalkSyncLogTable logs={logs} loading={logsLoading} />
      </CardContent>
    </Card>
  )
}

export function DingTalkSyncConflictList({
  conflicts,
  loading,
  resolvingConflictId,
  onBindCandidate,
}: {
  conflicts: DingTalkSyncConflict[]
  loading: boolean
  resolvingConflictId?: number | null
  onBindCandidate?: (conflict: DingTalkSyncConflict) => void
}) {
  const { t } = useTranslation()
  const [bindingConflict, setBindingConflict] =
    useState<DingTalkSyncConflict | null>(null)

  if (loading) {
    return <Skeleton className='h-28 w-full' />
  }

  if (conflicts.length === 0) {
    return (
      <p className='text-muted-foreground rounded-md border p-4 text-sm'>
        {t('No pending DingTalk sync conflicts.')}
      </p>
    )
  }

  return (
    <>
      <div className='rounded-md border'>
        <div className='flex items-center gap-2 border-b px-4 py-3'>
          <TriangleAlert className='size-4 text-amber-600' />
          <div className='min-w-0'>
            <h3 className='text-sm font-medium'>
              {t('Pending Sync Conflicts')}
            </h3>
            <p className='text-muted-foreground text-xs'>
              {t('Conflicting DingTalk members are not bound automatically.')}
            </p>
          </div>
        </div>
        <div className='divide-y'>
          {conflicts.map((conflict) => (
            <div
              key={conflict.id}
              className='grid gap-2 px-4 py-3 text-sm md:grid-cols-[minmax(0,1fr)_auto]'
            >
              <div className='min-w-0'>
                <div className='flex flex-wrap items-center gap-2'>
                  <span className='font-medium'>
                    {conflict.name || conflict.external_user_id}
                  </span>
                  <Badge variant='outline'>
                    {t(syncConflictTypeLabel(conflict.conflict_type))}
                  </Badge>
                </div>
                <div className='text-muted-foreground mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs'>
                  {conflict.email ? <span>{conflict.email}</span> : null}
                  {conflict.mobile ? <span>{conflict.mobile}</span> : null}
                  <span>{conflict.external_user_id}</span>
                </div>
              </div>
              <div className='text-muted-foreground text-xs md:text-right'>
                <div>
                  {conflict.candidate_user_id > 0
                    ? t('Candidate user #{{id}}', {
                        id: conflict.candidate_user_id,
                      })
                    : t('Multiple candidate users')}
                </div>
                {conflict.candidate_user_id > 0 && onBindCandidate ? (
                  <Button
                    type='button'
                    size='sm'
                    variant='outline'
                    className='mt-2'
                    disabled={resolvingConflictId === conflict.id}
                    onClick={() => setBindingConflict(conflict)}
                  >
                    <CheckCircle2 className='size-4' />
                    {resolvingConflictId === conflict.id
                      ? t('Binding...')
                      : t('Bind candidate')}
                  </Button>
                ) : null}
              </div>
            </div>
          ))}
        </div>
      </div>
      <ConfirmDialog
        title={t('Bind DingTalk identity')}
        desc={
          bindingConflict
            ? t(
                'Bind this DingTalk identity to candidate user #{{id}} and mark the conflict resolved.',
                { id: bindingConflict.candidate_user_id }
              )
            : ''
        }
        confirmText={
          bindingConflict && resolvingConflictId === bindingConflict.id
            ? t('Binding...')
            : t('Bind candidate')
        }
        open={Boolean(bindingConflict)}
        onOpenChange={(open) => {
          if (!open) setBindingConflict(null)
        }}
        isLoading={
          Boolean(bindingConflict) &&
          resolvingConflictId === bindingConflict?.id
        }
        handleConfirm={() => {
          if (!bindingConflict || !onBindCandidate) return
          onBindCandidate(bindingConflict)
          setBindingConflict(null)
        }}
      />
    </>
  )
}

function DingTalkSyncTaskSummary({ task }: { task: DingTalkSyncTask }) {
  const { t } = useTranslation()
  const counters = [
    [
      t('Departments'),
      task.departments_created +
        task.departments_updated +
        task.departments_disabled,
    ],
    [t('Users'), task.users_created + task.users_updated],
    [
      t('Memberships'),
      task.memberships_created +
        task.memberships_updated +
        task.memberships_disabled,
    ],
    [t('Skipped'), task.skipped_count],
    [t('Failed'), task.failed_count],
  ] as const

  return (
    <div className='grid gap-3 rounded-md border p-4 md:grid-cols-[minmax(0,1fr)_auto]'>
      <div className='min-w-0'>
        <div className='flex flex-wrap items-center gap-2'>
          <Badge
            variant={task.status === 'failed' ? 'destructive' : 'secondary'}
          >
            {t(syncStatusLabel(task.status))}
          </Badge>
          <span className='text-muted-foreground text-sm'>
            {t('Task')} #{task.id}
          </span>
        </div>
        {task.error_summary ? (
          <p className='text-destructive mt-2 text-sm'>{task.error_summary}</p>
        ) : null}
      </div>
      <div className='grid grid-cols-2 gap-2 text-sm sm:grid-cols-5'>
        {counters.map(([label, value]) => (
          <div key={label} className='bg-muted/50 rounded-md px-3 py-2'>
            <div className='text-muted-foreground'>{label}</div>
            <div className='font-medium'>{value}</div>
          </div>
        ))}
      </div>
    </div>
  )
}

function DingTalkSyncLogTable({
  logs,
  loading,
}: {
  logs: DingTalkSyncLog[]
  loading: boolean
}) {
  const { t } = useTranslation()

  if (loading) {
    return <Skeleton className='h-36 w-full' />
  }

  if (logs.length === 0) {
    return (
      <p className='text-muted-foreground rounded-md border p-4 text-sm'>
        {t('No DingTalk sync logs yet.')}
      </p>
    )
  }

  return (
    <div className='overflow-hidden rounded-md border'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Object')}</TableHead>
            <TableHead>{t('Action')}</TableHead>
            <TableHead>{t('Message')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {logs.map((log) => (
            <TableRow key={log.id}>
              <TableCell>
                <Badge
                  variant={log.status === 'failed' ? 'destructive' : 'outline'}
                >
                  {t(syncLogStatusLabel(log.status))}
                </Badge>
              </TableCell>
              <TableCell className='whitespace-nowrap'>
                {log.object_type}
                {log.object_external_id ? ` #${log.object_external_id}` : ''}
              </TableCell>
              <TableCell>{log.action}</TableCell>
              <TableCell className='max-w-sm truncate'>{log.message}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function syncStatusLabel(status: DingTalkSyncTask['status']) {
  switch (status) {
    case 'pending':
      return 'Pending'
    case 'running':
      return 'Running'
    case 'succeeded':
      return 'Succeeded'
    case 'failed':
      return 'Failed'
    default:
      return status
  }
}

function syncLogStatusLabel(status: string) {
  switch (status) {
    case 'success':
      return 'Success'
    case 'failed':
      return 'Failed'
    case 'skipped':
      return 'Skipped'
    case 'warning':
      return 'Warning'
    default:
      return status
  }
}

function syncConflictTypeLabel(type: string) {
  switch (type) {
    case 'email':
      return 'Email conflict'
    case 'mobile':
      return 'Mobile conflict'
    case 'email_mobile':
      return 'Email and mobile conflict'
    default:
      return 'Conflict'
  }
}

export function EnterpriseDingTalkConnectivityResult({
  result,
}: {
  result: DingTalkConnectivityResult
}) {
  const { t } = useTranslation()
  const passed = result.code === 'auth_success'
  const guidance = getConnectivityGuidance(t, result.code)

  return (
    <Alert
      className={cn(
        'items-start',
        passed ? 'border-green-500/40 bg-green-500/5' : undefined
      )}
      variant={passed ? 'default' : 'destructive'}
    >
      {passed ? (
        <CheckCircle2 className='size-4' />
      ) : (
        <AlertCircle className='size-4' />
      )}
      <AlertTitle>{guidance.title}</AlertTitle>
      <AlertDescription>
        <div className='space-y-1'>
          <p>{guidance.description}</p>
          <p>
            {t('Result code')}: <span className='font-mono'>{result.code}</span>
            {result.stage ? (
              <>
                {' '}
                · {t('Stage')}:{' '}
                <span className='font-mono'>{result.stage}</span>
              </>
            ) : null}
          </p>
        </div>
      </AlertDescription>
    </Alert>
  )
}

function getConnectivityGuidance(
  t: (key: string) => string,
  code: DingTalkConnectivityCode
) {
  switch (code) {
    case 'auth_success':
      return {
        title: t('DingTalk app is reachable'),
        description: t(
          'Credentials and address book permission are ready for login and sync.'
        ),
      }
    case 'auth_invalid_credentials':
      return {
        title: t('Check DingTalk app credentials'),
        description: t(
          'Confirm the Corp ID, App Key, and App Secret in the DingTalk developer console.'
        ),
      }
    case 'auth_permission_insufficient':
      return {
        title: t('Grant address book permission'),
        description: t(
          'Enable address book read permission for this DingTalk app, then publish the app again.'
        ),
      }
    case 'network_unreachable':
      return {
        title: t('DingTalk network is unreachable'),
        description: t(
          'Check outbound network access and retry after the DingTalk OpenAPI endpoint is reachable.'
        ),
      }
    case 'callback_misconfigured':
      return {
        title: t('Fix DingTalk callback URL'),
        description: t(
          'Use the same valid HTTPS callback URL in new-api and the DingTalk developer console.'
        ),
      }
  }
}

function ToggleField(props: {
  label: string
  description: string
  checked: boolean
  disabled?: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <FormItem className='flex min-h-24 items-start justify-between gap-4 rounded-lg border p-4'>
      <div className='flex min-w-0 gap-3'>
        <ShieldCheck className='text-muted-foreground mt-0.5 size-4 shrink-0' />
        <div className='min-w-0 space-y-1'>
          <FormLabel className='leading-5'>{props.label}</FormLabel>
          <FormDescription>{props.description}</FormDescription>
        </div>
      </div>
      <FormControl>
        <Switch
          checked={props.checked}
          disabled={props.disabled}
          onCheckedChange={props.onCheckedChange}
        />
      </FormControl>
    </FormItem>
  )
}

function DingTalkConfigSkeleton() {
  return (
    <div className='flex flex-col gap-4'>
      <Skeleton className='h-72 w-full' />
      <Skeleton className='h-36 w-full' />
    </div>
  )
}
