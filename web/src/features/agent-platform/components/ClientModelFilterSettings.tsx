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
import { Filter } from 'lucide-react'
import { useId, useMemo, useState, type ReactNode } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { MultiSelect } from '@/components/multi-select'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup } from '@/components/ui/field'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { getEnabledModels } from '@/features/channels/api'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { ROLE } from '@/lib/roles'
import {
  createServerError,
  getServerErrorMessage,
  requireServerSuccess,
} from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

import {
  CLIENT_MODEL_FILTER_QUERY_KEY,
  getClientModelFilter,
  updateClientModelFilter,
  type ClientModelFilter,
} from '../api'
import {
  clientModelFilterFormSchema,
  type ClientModelFilterFormValues,
} from '../lib/model-filter-form'

function ClientModelFilterDialog(props: {
  config: ClientModelFilter
  onClose: () => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const formId = useId()
  const queryClient = useQueryClient()
  const form = useForm<ClientModelFilterFormValues>({
    resolver: zodResolver(clientModelFilterFormSchema),
    defaultValues: { excluded_models: props.config.excluded_models },
  })
  const selected = useWatch({ control: form.control, name: 'excluded_models' })
  const candidatesQuery = useQuery({
    queryKey: ['enabled-models'],
    queryFn: async () => requireServerSuccess(await getEnabledModels()),
    meta: { errorToast: false },
  })
  const options = useMemo(
    () =>
      (candidatesQuery.data?.data ?? []).map((id) => ({
        label: id,
        value: id,
      })),
    [candidatesQuery.data]
  )
  const saveMutation = useMutation({
    mutationFn: async (values: ClientModelFilterFormValues) => {
      const result = requireServerSuccess(await updateClientModelFilter(values))
      if (!result.data) {
        throw createServerError(result, t('Failed to save model filter'))
      }
      return result.data
    },
    onSuccess: (config) => {
      queryClient.setQueryData(CLIENT_MODEL_FILTER_QUERY_KEY, config)
      toast.success(t('Model filter saved'))
      props.onClose()
    },
    onError: (error) =>
      handleServerError(error, t('Failed to save model filter')),
  })

  return (
    <Dialog
      open
      onOpenChange={(open) =>
        !open && !saveMutation.isPending && props.onClose()
      }
      title={t('Client model filter')}
      description={t(
        'Hide selected models from the Huaqing AI Assistant model list.'
      )}
      contentClassName='sm:max-w-xl'
      footer={
        <>
          <Button
            type='button'
            variant='ghost'
            disabled={saveMutation.isPending || selected.length === 0}
            onClick={() =>
              form.setValue('excluded_models', [], {
                shouldDirty: true,
                shouldValidate: true,
              })
            }
          >
            {t('Clear all')}
          </Button>
          <Button
            type='button'
            variant='outline'
            disabled={saveMutation.isPending}
            onClick={props.onClose}
          >
            {t('Cancel')}
          </Button>
          <Button type='submit' form={formId} disabled={saveMutation.isPending}>
            {saveMutation.isPending ? t('Saving...') : t('Save')}
          </Button>
        </>
      }
    >
      <Form {...form}>
        <form
          id={formId}
          onSubmit={form.handleSubmit((values) => saveMutation.mutate(values))}
        >
          <FieldGroup>
            <FormField
              control={form.control}
              name='excluded_models'
              render={({ field, fieldState }) => (
                <Field
                  data-invalid={fieldState.invalid}
                  data-disabled={saveMutation.isPending}
                >
                  <FormLabel>{t('Excluded models')}</FormLabel>
                  <FormControl>
                    <MultiSelect
                      options={options}
                      selected={field.value}
                      onChange={field.onChange}
                      onBlur={field.onBlur}
                      ref={field.ref}
                      allowCreate
                      maxVisibleChips={8}
                      disabled={saveMutation.isPending}
                      placeholder={t('Search or enter an exact model ID')}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Leave the list empty to show all available models.')}{' '}
                    {t('Press Enter to add a model ID.')}
                  </FormDescription>
                  <FormMessage />
                </Field>
              )}
            />
            <p className='text-muted-foreground text-sm'>
              {t('{{count}} excluded models', {
                count: formatNumber(selected.length, locale),
              })}
            </p>
            {candidatesQuery.isLoading && (
              <LoadingState inline message={t('Loading...')} />
            )}
            {candidatesQuery.isError && (
              <Alert>
                <AlertDescription>
                  {getServerErrorMessage(
                    candidatesQuery.error,
                    t('Failed to load model suggestions')
                  )}
                  <Button
                    type='button'
                    variant='link'
                    size='sm'
                    onClick={() => void candidatesQuery.refetch()}
                  >
                    {t('Retry')}
                  </Button>
                </AlertDescription>
              </Alert>
            )}
            {selected.includes(props.config.default_model) && (
              <Alert>
                <AlertDescription>
                  {t(
                    'The default model {{model}} is filtered. Creating a client session may fail.',
                    { model: props.config.default_model }
                  )}
                </AlertDescription>
              </Alert>
            )}
          </FieldGroup>
        </form>
      </Form>
    </Dialog>
  )
}

export function ClientModelFilterSettings() {
  const { t, i18n } = useTranslation()
  const isSuperAdmin = useAuthStore(
    (state) => (state.auth.user?.role ?? 0) >= ROLE.SUPER_ADMIN
  )
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const [open, setOpen] = useState(false)
  const configQuery = useQuery({
    queryKey: CLIENT_MODEL_FILTER_QUERY_KEY,
    queryFn: async () => {
      const result = requireServerSuccess(await getClientModelFilter())
      if (!result.data) {
        throw createServerError(result, t('Failed to load model filter'))
      }
      return result.data
    },
    enabled: isSuperAdmin,
    meta: { errorToast: false },
  })

  if (!isSuperAdmin) return null

  let dialog: ReactNode = null
  if (open && configQuery.data) {
    dialog = (
      <ClientModelFilterDialog
        config={configQuery.data}
        onClose={() => setOpen(false)}
      />
    )
  }
  if (open && !configQuery.data) {
    dialog = (
      <Dialog open onOpenChange={setOpen} title={t('Client model filter')}>
        {configQuery.isError ? (
          <ErrorState
            title={t('Failed to load model filter')}
            description={getServerErrorMessage(
              configQuery.error,
              t('Failed to load model filter')
            )}
            onRetry={() => void configQuery.refetch()}
            className='min-h-40'
          />
        ) : (
          <LoadingState />
        )}
      </Dialog>
    )
  }

  return (
    <>
      <Button
        type='button'
        variant='outline'
        size='sm'
        disabled={configQuery.isLoading}
        onClick={() => setOpen(true)}
      >
        <Filter className='size-4' aria-hidden='true' />
        {t('Client model filter')}
        {configQuery.isLoading && <LoadingState inline size='sm' />}
        {!configQuery.isLoading && configQuery.data && (
          <Badge variant='secondary'>
            {formatNumber(configQuery.data.excluded_models.length, locale)}
          </Badge>
        )}
      </Button>
      {dialog}
    </>
  )
}
