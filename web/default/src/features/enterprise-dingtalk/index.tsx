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
import { useEffect } from 'react'
import * as z from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useForm } from 'react-hook-form'
import { AlertCircle, Building2, RefreshCw, Save, ShieldCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import { SectionPageLayout } from '@/components/layout'
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from '@/components/ui/alert'
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
import { Textarea } from '@/components/ui/textarea'
import {
  enterpriseDingTalkQueryKey,
  getDingTalkConfig,
  saveDingTalkConfig,
} from './api'
import type { DingTalkConfig } from './types'

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
    })
    .superRefine((values, ctx) => {
      const enabling = values.login_enabled || values.sync_enabled
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

type DingTalkConfigFormValues = z.infer<
  ReturnType<typeof dingTalkConfigSchema>
>

const emptyConfig: DingTalkConfig = {
  id: 0,
  tenant_id: 0,
  corp_id: '',
  app_key: '',
  callback_url: '',
  sync_scope: '',
  login_enabled: false,
  sync_enabled: false,
  has_app_secret: false,
  created_at: 0,
  updated_at: 0,
}

function configToFormValues(
  config: DingTalkConfig
): DingTalkConfigFormValues {
  return {
    corp_id: config.corp_id ?? '',
    app_key: config.app_key ?? '',
    app_secret: '',
    callback_url: config.callback_url ?? '',
    sync_scope: config.sync_scope ?? '',
    login_enabled: Boolean(config.login_enabled),
    sync_enabled: Boolean(config.sync_enabled),
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
      if (!result.success) throw new Error(result.message || t('Request failed'))
      return result.data ?? emptyConfig
    },
  })

  const config = query.data ?? emptyConfig
  const form = useForm<DingTalkConfigFormValues>({
    resolver: zodResolver(dingTalkConfigSchema(t)),
    defaultValues: configToFormValues(config),
  })

  useEffect(() => {
    if (!query.data) return
    form.reset(configToFormValues(query.data))
  }, [form, query.data])

  const mutation = useMutation({
    mutationFn: async (values: DingTalkConfigFormValues) => {
      if ((values.login_enabled || values.sync_enabled) && !config.has_app_secret) {
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
      }
      const appSecret = values.app_secret.trim()
      const result = await saveDingTalkConfig(
        appSecret ? { ...payload, app_secret: appSecret } : payload
      )
      if (!result.success) throw new Error(result.message || t('Request failed'))
      return result.data ?? emptyConfig
    },
    onSuccess: async (saved) => {
      queryClient.setQueryData(enterpriseDingTalkQueryKey, saved)
      form.reset(configToFormValues(saved))
      await queryClient.invalidateQueries({ queryKey: enterpriseDingTalkQueryKey })
      toast.success(t('DingTalk configuration saved'))
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const onSubmit = (values: DingTalkConfigFormValues) => {
    mutation.mutate(values)
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('DingTalk Integration')}</SectionPageLayout.Title>
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
                {t('Only root administrators can edit DingTalk app credentials.')}
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
                          {t('Configure the DingTalk app used for login and address book sync.')}
                        </CardDescription>
                      </div>
                      <Badge variant={config.has_app_secret ? 'secondary' : 'outline'}>
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
                            {t('Saved secrets are never returned by the API or stored in page state.')}
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
                            {t('Use the same HTTPS callback URL in the DingTalk developer console.')}
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
                              placeholder={t('One department ID or rule per line')}
                              disabled={!canEdit}
                              {...field}
                            />
                          </FormControl>
                          <FormDescription>
                            {t('Limit address book sync to selected DingTalk departments or leave blank for all.')}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle>{t('Enablement')}</CardTitle>
                    <CardDescription>
                      {t('Credentials and a valid callback URL are required before enabling either switch.')}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className='grid gap-4 md:grid-cols-2'>
                    <FormField
                      control={form.control}
                      name='login_enabled'
                      render={({ field }) => (
                        <ToggleField
                          label={t('Enable DingTalk login')}
                          description={t('Allow employees to sign in with DingTalk after OAuth is configured.')}
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
                          description={t('Allow manual and scheduled DingTalk department synchronization.')}
                          checked={field.value}
                          disabled={!canEdit}
                          onCheckedChange={field.onChange}
                        />
                      )}
                    />
                  </CardContent>
                </Card>

                <div className='flex flex-wrap items-center justify-between gap-3'>
                  <Button variant='outline' render={<Link to='/enterprise-organization' />}>
                    <Building2 className='size-4' />
                    {t('Enterprise Organization')}
                  </Button>
                  <Button type='submit' disabled={!canEdit || mutation.isPending}>
                    <Save className='size-4' />
                    {mutation.isPending ? t('Saving...') : t('Save')}
                  </Button>
                </div>
              </form>
            </Form>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
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
