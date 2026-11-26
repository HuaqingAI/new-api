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
import { useId, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup } from '@/components/ui/field'
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
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { handleServerError } from '@/lib/handle-server-error'
import {
  getServerErrorMessage,
  getServerErrorStatus,
} from '@/lib/server-error-message'

import {
  clientNotificationFormSchema,
  type ClientNotificationFormValues,
} from '../lib/client-notification-form'
import {
  getClientNotification,
  NOTIFICATION_QUERY_KEY,
  saveClientNotification,
  type ClientNotification,
} from '../notification-api'

export function ClientNotificationEditor(props: {
  id?: string
  onClose: () => void
}) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: [...NOTIFICATION_QUERY_KEY, 'detail', props.id],
    queryFn: () => getClientNotification(props.id ?? ''),
    enabled: Boolean(props.id),
    staleTime: Infinity,
    refetchOnMount: 'always',
    meta: { errorToast: false },
  })
  if (!props.id || (query.data && !query.isFetching)) {
    return (
      <NotificationForm
        item={props.id ? query.data : undefined}
        onClose={props.onClose}
      />
    )
  }
  return (
    <Dialog
      open
      title={t('Edit notification')}
      onOpenChange={(open) => !open && props.onClose()}
    >
      {query.isError ? (
        <ErrorState
          title={t('Failed to load client notifications')}
          description={getServerErrorMessage(query.error)}
          onRetry={() => void query.refetch()}
        />
      ) : (
        <LoadingState />
      )}
    </Dialog>
  )
}

function NotificationForm(props: {
  item?: ClientNotification
  onClose: () => void
}) {
  const { t } = useTranslation()
  const formId = useId()
  const queryClient = useQueryClient()
  const [item, setItem] = useState(props.item)
  const [discard, setDiscard] = useState(false)
  const [conflict, setConflict] = useState(false)
  const form = useForm<ClientNotificationFormValues>({
    resolver: zodResolver(clientNotificationFormSchema),
    defaultValues: item ?? {
      title: '',
      summary: '',
      content: '',
      enabled: true,
    },
  })
  const save = useMutation({
    mutationFn: (values: ClientNotificationFormValues) =>
      saveClientNotification(values, item),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: NOTIFICATION_QUERY_KEY })
      toast.success(t('Client notification saved'))
      props.onClose()
    },
    onError: (error) => {
      setConflict(getServerErrorStatus(error) === 412)
      handleServerError(error, t('Failed to save client notification'))
    },
  })
  const reload = useMutation({
    mutationFn: () => getClientNotification(item?.id ?? ''),
    onSuccess: (fresh) => {
      setItem(fresh)
      form.reset({
        title: fresh.title,
        summary: fresh.summary,
        content: fresh.content,
        enabled: fresh.enabled,
      })
      setConflict(false)
    },
    onError: (error) => handleServerError(error),
  })
  const busy = save.isPending || reload.isPending
  const dirty = form.formState.isDirty
  const requestClose = () => {
    if (busy || discard) return
    if (dirty) setDiscard(true)
    else props.onClose()
  }
  return (
    <>
      <Dialog
        open
        title={item ? t('Edit notification') : t('Add notification')}
        contentClassName='sm:max-w-3xl'
        onOpenChange={(open) => !open && requestClose()}
        footer={
          <>
            <Button variant='outline' disabled={busy} onClick={requestClose}>
              {t('Cancel')}
            </Button>
            <Button type='submit' form={formId} disabled={busy || conflict}>
              {save.isPending ? t('Saving...') : t('Save')}
            </Button>
          </>
        }
      >
        <Form {...form}>
          <form
            id={formId}
            onSubmit={form.handleSubmit((values) => {
              if (!busy && !conflict) save.mutate(values)
            })}
          >
            <FieldGroup>
              {(
                [
                  { name: 'title', label: t('Title') },
                  { name: 'summary', label: t('Summary') },
                  { name: 'content', label: t('Content') },
                ] as const
              ).map(({ name, label }) => (
                <FormField
                  key={name}
                  control={form.control}
                  name={name}
                  render={({ field, fieldState }) => (
                    <FormItem>
                      <Field
                        data-invalid={fieldState.invalid}
                        data-disabled={busy}
                      >
                        <FormLabel>{label}</FormLabel>
                        <FormControl>
                          {name === 'title' ? (
                            <Input {...field} disabled={busy} />
                          ) : (
                            <Textarea
                              {...field}
                              rows={name === 'content' ? 12 : 2}
                              disabled={busy}
                              className='resize-y [overflow-wrap:anywhere]'
                            />
                          )}
                        </FormControl>
                        {name === 'content' && (
                          <FormDescription>
                            {t('Plain text only. Line breaks are preserved.')}
                          </FormDescription>
                        )}
                        <FormMessage />
                      </Field>
                    </FormItem>
                  )}
                />
              ))}
              <FormField
                control={form.control}
                name='enabled'
                render={({ field }) => (
                  <FormItem>
                    <Field data-disabled={busy}>
                      <FormLabel>{t('Status')}</FormLabel>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                          onBlur={field.onBlur}
                          ref={field.ref}
                          disabled={busy}
                        />
                      </FormControl>
                    </Field>
                  </FormItem>
                )}
              />
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Editing does not notify users who have already read this message. Create a new notification to remind them again.'
                )}
              </p>
              {save.isError && !item && (
                <Alert>
                  <AlertDescription>
                    {t(
                      'If saving timed out, refresh the list before trying again to avoid duplicates.'
                    )}
                  </AlertDescription>
                </Alert>
              )}
              {conflict && (
                <Alert>
                  <AlertDescription>
                    {t('Notification has changed. Reload and try again.')}
                    <p>{t('Reloading discards your unsaved changes.')}</p>
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      disabled={busy}
                      onClick={() => reload.mutate()}
                    >
                      {t('Reload')}
                    </Button>
                  </AlertDescription>
                </Alert>
              )}
            </FieldGroup>
          </form>
        </Form>
      </Dialog>
      <ConfirmDialog
        open={discard}
        onOpenChange={setDiscard}
        title={t('Discard unsaved changes?')}
        desc={t('Your unsaved notification changes will be lost.')}
        destructive
        confirmText={t('Discard')}
        handleConfirm={props.onClose}
      />
    </>
  )
}
