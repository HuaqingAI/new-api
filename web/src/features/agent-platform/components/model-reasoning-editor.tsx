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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { useFieldArray, useForm, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { EmptyState } from '@/components/empty-state'
import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLegend, FieldSet } from '@/components/ui/field'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import {
  modelReasoningFormSchema,
  type ModelReasoningFormValues,
} from '../lib/model-reasoning-form'
import {
  MODEL_FAMILIES,
  MODEL_REASONING_QUERY_KEY,
  REASONING_EFFORTS,
  updateModelReasoningPolicy,
  type ModelReasoningPolicy,
} from '../model-reasoning-api'

const effortOptions = REASONING_EFFORTS.map((effort) => ({
  label: effort,
  value: effort,
}))
const familyLabels = {
  deepseek: 'DeepSeek',
  qwen: 'Qwen',
  glm: 'GLM',
  gpt: 'GPT',
}

export function ModelReasoningEditor(props: {
  config: ModelReasoningPolicy
  onClose: () => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const formId = useId()
  const queryClient = useQueryClient()
  const [discardOpen, setDiscardOpen] = useState(false)
  const form = useForm<ModelReasoningFormValues>({
    resolver: zodResolver(modelReasoningFormSchema),
    defaultValues: { rules: props.config.rules },
  })
  const fieldArray = useFieldArray({ control: form.control, name: 'rules' })
  const rules = useWatch({ control: form.control, name: 'rules' })
  const mutation = useMutation({
    mutationFn: updateModelReasoningPolicy,
    onSuccess: (config) => {
      queryClient.setQueryData(MODEL_REASONING_QUERY_KEY, config)
      toast.success(t('Model reasoning settings saved'))
      props.onClose()
    },
    onError: (error) =>
      handleServerError(error, t('Failed to save model reasoning settings')),
  })
  const close = () => {
    if (mutation.isPending) return
    if (form.formState.isDirty) setDiscardOpen(true)
    else props.onClose()
  }
  const rulesError =
    form.formState.errors.rules?.root?.message ??
    form.formState.errors.rules?.message

  return (
    <>
      <Dialog
        open
        title={t('Model reasoning settings')}
        description={t(
          'Configure the available model reasoning efforts in Huaqing AI Assistant, with exact model rules taking priority over family rules and unmatched models using low, medium, high by default.'
        )}
        contentClassName='sm:max-w-4xl'
        onOpenChange={(open) => !open && close()}
        footer={
          <>
            <Button
              type='button'
              variant='ghost'
              disabled={mutation.isPending || rules.length === 0}
              onClick={() => fieldArray.replace([])}
            >
              {t('Clear all')}
            </Button>
            <Button
              type='button'
              variant='outline'
              disabled={mutation.isPending}
              onClick={close}
            >
              {t('Cancel')}
            </Button>
            <Button type='submit' form={formId} disabled={mutation.isPending}>
              {mutation.isPending ? t('Saving...') : t('Save')}
            </Button>
          </>
        }
      >
        <Form {...form}>
          <form
            id={formId}
            onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
          >
            <FieldGroup>
              <Button
                type='button'
                variant='outline'
                className='w-fit'
                disabled={
                  mutation.isPending || fieldArray.fields.length >= 1000
                }
                onClick={() =>
                  fieldArray.append({
                    match_type: 'family',
                    match_value: '',
                    efforts: ['low', 'medium', 'high'],
                  })
                }
              >
                {t('Add reasoning rule')}
              </Button>
              {fieldArray.fields.length === 0 && (
                <EmptyState
                  title={t('No reasoning rules')}
                  description={t(
                    'All models use the default reasoning choices.'
                  )}
                  className='min-h-32'
                />
              )}
              {fieldArray.fields.map((row, index) => (
                <FieldSet
                  key={row.id}
                  className='min-w-0 gap-3 rounded-lg border p-3'
                  disabled={mutation.isPending}
                >
                  <FieldLegend className='mb-0' variant='label'>
                    {t('Rule {{number}}', {
                      number: formatNumber(index + 1, locale),
                    })}
                  </FieldLegend>
                  <Button
                    className='ml-auto'
                    type='button'
                    variant='ghost'
                    size='sm'
                    disabled={mutation.isPending}
                    onClick={() => fieldArray.remove(index)}
                    aria-label={t('Remove rule {{number}}', {
                      number: formatNumber(index + 1, locale),
                    })}
                  >
                    {t('Delete')}
                  </Button>
                  <FieldGroup className='grid min-w-0 gap-3 md:grid-cols-[9rem_1fr_1.5fr]'>
                    <FormField
                      control={form.control}
                      name={`rules.${index}.match_type`}
                      render={({ field, fieldState }) => (
                        <FormItem className='min-w-0'>
                          <Field
                            data-invalid={fieldState.invalid}
                            data-disabled={mutation.isPending}
                          >
                            <FormLabel>{t('Rule type')}</FormLabel>
                            <FormControl>
                              <NativeSelect
                                {...field}
                                className='w-full'
                                disabled={mutation.isPending}
                                onChange={(event) => {
                                  field.onChange(event)
                                  form.setValue(
                                    `rules.${index}.match_value`,
                                    '',
                                    { shouldDirty: true }
                                  )
                                }}
                              >
                                <NativeSelectOption value='family'>
                                  {t('Model family')}
                                </NativeSelectOption>
                                <NativeSelectOption value='model'>
                                  {t('Exact model')}
                                </NativeSelectOption>
                              </NativeSelect>
                            </FormControl>
                            <FormMessage />
                          </Field>
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name={`rules.${index}.match_value`}
                      render={({ field, fieldState }) => (
                        <FormItem className='min-w-0'>
                          <Field
                            data-invalid={fieldState.invalid}
                            data-disabled={mutation.isPending}
                          >
                            <FormLabel>
                              {rules[index]?.match_type === 'family'
                                ? t('Model family')
                                : t('Exact model ID')}
                            </FormLabel>
                            <FormControl>
                              {rules[index]?.match_type === 'family' ? (
                                <NativeSelect
                                  {...field}
                                  className='w-full'
                                  disabled={mutation.isPending}
                                >
                                  <NativeSelectOption value=''>
                                    {t('Select a model family')}
                                  </NativeSelectOption>
                                  {MODEL_FAMILIES.map((family) => (
                                    <NativeSelectOption
                                      key={family}
                                      value={family}
                                    >
                                      {familyLabels[family]}
                                    </NativeSelectOption>
                                  ))}
                                </NativeSelect>
                              ) : (
                                <Input
                                  {...field}
                                  disabled={mutation.isPending}
                                  placeholder={t('Enter an exact model ID')}
                                />
                              )}
                            </FormControl>
                            <FormMessage />
                          </Field>
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name={`rules.${index}.efforts`}
                      render={({ field, fieldState }) => (
                        <FormItem className='min-w-0'>
                          <Field
                            data-invalid={fieldState.invalid}
                            data-disabled={mutation.isPending}
                          >
                            <FormLabel>{t('Reasoning choices')}</FormLabel>
                            <FormControl>
                              <MultiSelect
                                options={effortOptions}
                                selected={field.value}
                                onChange={field.onChange}
                                onBlur={field.onBlur}
                                ref={field.ref}
                                disabled={mutation.isPending}
                                placeholder={t('Select reasoning efforts')}
                              />
                            </FormControl>
                            <FormMessage />
                          </Field>
                        </FormItem>
                      )}
                    />
                  </FieldGroup>
                </FieldSet>
              ))}
              {rulesError && (
                <p role='alert' className='text-destructive text-sm'>
                  {t(rulesError)}
                </p>
              )}
            </FieldGroup>
          </form>
        </Form>
      </Dialog>
      <ConfirmDialog
        open={discardOpen}
        onOpenChange={setDiscardOpen}
        title={t('Discard unsaved changes?')}
        desc={t('Your unsaved reasoning rules will be discarded.')}
        confirmText={t('Discard')}
        handleConfirm={props.onClose}
      />
    </>
  )
}
