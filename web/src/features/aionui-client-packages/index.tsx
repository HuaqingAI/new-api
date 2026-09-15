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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Download,
  RefreshCw,
  SlidersHorizontal,
  Trash2,
  Upload,
} from 'lucide-react'
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'

import {
  type AionUiClientPackage,
  type AionUiClientPackageStatus,
  type AionUiClientPlatform,
  deleteClientPackage,
  getClientPackageDownloadUrl,
  getClientPackages,
  updateClientPackageStatus,
  uploadClientPackage,
} from './api'
import { ClientInstallations } from './components/client-installations'
import {
  ClientPackageRolloutDialog,
  ClientPackageRolloutFields,
} from './components/client-package-rollout'
import {
  clientPackageRolloutScopes,
  defaultClientPackageRollout,
} from './rollout'

const PLATFORM_OPTIONS: Array<{
  value: AionUiClientPlatform
  label: string
  fileAccept: string
  updateAccept: string
  metadataName: string
}> = [
  {
    value: 'windows_x64',
    label: 'Windows x64',
    fileAccept: '.exe',
    updateAccept: '.exe',
    metadataName: 'latest.yml',
  },
  {
    value: 'mac_arm64',
    label: 'macOS Apple Silicon',
    fileAccept: '.dmg',
    updateAccept: '.zip',
    metadataName: 'latest-arm64-mac.yml',
  },
  {
    value: 'mac_x64',
    label: 'macOS Intel',
    fileAccept: '.dmg',
    updateAccept: '.zip',
    metadataName: 'latest-mac.yml',
  },
]

type UploadFormState = {
  platform: AionUiClientPlatform
  version: string
  releaseNote: string
  rollout: typeof defaultClientPackageRollout
  file: File | null
  updateFile: File | null
  updateMetadataFile: File | null
}

const initialForm: UploadFormState = {
  platform: 'windows_x64',
  version: '',
  releaseNote: '',
  rollout: defaultClientPackageRollout,
  file: null,
  updateFile: null,
  updateMetadataFile: null,
}

type UploadAction = 'draft' | 'publish'

export function AionUiClientPackages() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [form, setForm] = useState<UploadFormState>(initialForm)
  const [uploadAction, setUploadAction] = useState<UploadAction | null>(null)
  const [rolloutItem, setRolloutItem] = useState<AionUiClientPackage | null>(
    null
  )
  const uploadToastId = useRef<string | number | null>(null)
  const selectedPlatform =
    PLATFORM_OPTIONS.find((item) => item.value === form.platform) ??
    PLATFORM_OPTIONS[0]

  const packagesQuery = useQuery({
    queryKey: ['aionui-client-packages'],
    queryFn: () => getClientPackages({}),
  })

  const dismissUploadToast = () => {
    if (uploadToastId.current !== null) {
      toast.dismiss(uploadToastId.current)
      uploadToastId.current = null
    }
  }

  const uploadMutation = useMutation({
    mutationFn: (publish: boolean) =>
      uploadClientPackage({
        platform: form.platform,
        version: form.version.trim(),
        releaseNote: form.releaseNote,
        publish,
        rolloutMode: form.rollout.rolloutMode,
        scopes: clientPackageRolloutScopes(form.rollout),
        file: form.file,
        updateFile: form.updateFile,
        updateMetadataFile: form.updateMetadataFile,
      }),
    onMutate: (publish) => {
      const nextAction = publish ? 'publish' : 'draft'
      setUploadAction(nextAction)
      uploadToastId.current = toast.loading(
        t(publish ? 'Uploading and publishing...' : 'Saving draft...')
      )
    },
    onSuccess: (res, publish) => {
      if (res.success) {
        dismissUploadToast()
        toast.success(
          publish
            ? t('Client package uploaded and published')
            : t('Draft saved')
        )
        setForm(initialForm)
        queryClient.invalidateQueries({ queryKey: ['aionui-client-packages'] })
      }
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Upload failed'))
    },
    onSettled: () => {
      dismissUploadToast()
      setUploadAction(null)
    },
  })

  const statusMutation = useMutation({
    mutationFn: (params: { id: number; status: AionUiClientPackageStatus }) =>
      updateClientPackageStatus(params.id, params.status),
    onSuccess: (res) => {
      if (res.success) {
        queryClient.invalidateQueries({ queryKey: ['aionui-client-packages'] })
      }
    },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteClientPackage,
    onSuccess: (res) => {
      if (res.success) {
        queryClient.invalidateQueries({ queryKey: ['aionui-client-packages'] })
      }
    },
  })

  const downloadMutation = useMutation({
    mutationFn: getClientPackageDownloadUrl,
    onSuccess: (url) => {
      window.location.assign(url)
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : t('Failed to prepare download')
      )
    },
  })

  const items = packagesQuery.data?.data?.items ?? []
  const isUploading = uploadMutation.isPending
  const canSubmitRollout =
    form.rollout.rolloutMode === 'global' ||
    form.rollout.userIds.length + form.rollout.departmentIds.length > 0

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Client Version')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          onClick={() =>
            queryClient.invalidateQueries({
              queryKey: ['aionui-client-packages'],
            })
          }
        >
          <RefreshCw className='size-4' />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='grid gap-4 lg:grid-cols-[360px_minmax(0,1fr)]'>
          <Card>
            <CardHeader>
              <CardTitle>{t('Upload client package')}</CardTitle>
            </CardHeader>
            <CardContent className='space-y-4'>
              <Field>
                <FieldLabel>{t('Platform')}</FieldLabel>
                <NativeSelect
                  disabled={isUploading}
                  value={form.platform}
                  onChange={(event) =>
                    setForm({
                      ...form,
                      platform: event.target.value as AionUiClientPlatform,
                      file: null,
                      updateFile: null,
                      updateMetadataFile: null,
                    })
                  }
                >
                  {PLATFORM_OPTIONS.map((item) => (
                    <NativeSelectOption key={item.value} value={item.value}>
                      {item.label}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
              <Field>
                <FieldLabel>{t('Version')}</FieldLabel>
                <Input
                  disabled={isUploading}
                  value={form.version}
                  onChange={(event) =>
                    setForm({ ...form, version: event.target.value })
                  }
                  placeholder='2.1.42'
                />
              </Field>
              <Field>
                <FieldLabel>{t('Download installer')}</FieldLabel>
                <Input
                  disabled={isUploading}
                  type='file'
                  accept={selectedPlatform.fileAccept}
                  onChange={(event) =>
                    setForm({ ...form, file: event.target.files?.[0] ?? null })
                  }
                />
              </Field>
              {form.platform !== 'windows_x64' ? (
                <Field>
                  <FieldLabel>{t('Auto update package')}</FieldLabel>
                  <Input
                    disabled={isUploading}
                    type='file'
                    accept={selectedPlatform.updateAccept}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        updateFile: event.target.files?.[0] ?? null,
                      })
                    }
                  />
                </Field>
              ) : null}
              <Field>
                <FieldLabel>
                  {t('Update metadata')} ({selectedPlatform.metadataName})
                </FieldLabel>
                <Input
                  disabled={isUploading}
                  type='file'
                  accept='.yml,.yaml,text/yaml'
                  onChange={(event) =>
                    setForm({
                      ...form,
                      updateMetadataFile: event.target.files?.[0] ?? null,
                    })
                  }
                />
              </Field>
              <Field>
                <FieldLabel>{t('Release notes')}</FieldLabel>
                <Textarea
                  disabled={isUploading}
                  value={form.releaseNote}
                  rows={4}
                  onChange={(event) =>
                    setForm({ ...form, releaseNote: event.target.value })
                  }
                />
              </Field>
              <ClientPackageRolloutFields
                disabled={isUploading}
                onChange={(rollout) => setForm({ ...form, rollout })}
                value={form.rollout}
              />
              <div className='flex flex-wrap gap-2'>
                <Button
                  variant='outline'
                  disabled={isUploading || !canSubmitRollout}
                  onClick={() => uploadMutation.mutate(false)}
                >
                  <Upload className='size-4' />
                  {uploadAction === 'draft' ? t('Saving...') : t('Save draft')}
                </Button>
                <Button
                  disabled={isUploading || !canSubmitRollout}
                  onClick={() => uploadMutation.mutate(true)}
                >
                  <Upload className='size-4' />
                  {uploadAction === 'publish'
                    ? t('Uploading and publishing...')
                    : t('Upload and publish')}
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t('Client packages')}</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Platform')}</TableHead>
                    <TableHead>{t('Version')}</TableHead>
                    <TableHead>{t('Status')}</TableHead>
                    <TableHead>{t('Release rollout')}</TableHead>
                    <TableHead>{t('Files')}</TableHead>
                    <TableHead>{t('Published At')}</TableHead>
                    <TableHead className='text-right'>{t('Actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((item) => (
                    <ClientPackageRow
                      key={item.id}
                      item={item}
                      onSetStatus={(status) =>
                        statusMutation.mutate({ id: item.id, status })
                      }
                      onDelete={() => deleteMutation.mutate(item.id)}
                      onDownload={() => downloadMutation.mutate(item.id)}
                      onEditRollout={() => setRolloutItem(item)}
                    />
                  ))}
                  {items.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={7} className='text-center'>
                        {packagesQuery.isLoading
                          ? t('Loading...')
                          : t('No data')}
                      </TableCell>
                    </TableRow>
                  ) : null}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>
        <ClientInstallations />
        <ClientPackageRolloutDialog
          item={rolloutItem}
          onClose={() => setRolloutItem(null)}
          onUpdated={() =>
            queryClient.invalidateQueries({
              queryKey: ['aionui-client-packages'],
            })
          }
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function ClientPackageRow(props: {
  item: AionUiClientPackage
  onSetStatus: (status: AionUiClientPackageStatus) => void
  onDelete: () => void
  onDownload: () => void
  onEditRollout: () => void
}) {
  const { t } = useTranslation()
  return (
    <TableRow>
      <TableCell>{platformLabel(props.item.platform)}</TableCell>
      <TableCell>{props.item.version}</TableCell>
      <TableCell>
        <Badge variant='outline'>
          {clientPackageStatusLabel(props.item.status, t)}
        </Badge>
      </TableCell>
      <TableCell>
        {clientPackageRolloutLabel(
          props.item.rollout_mode,
          props.item.scope_count,
          t
        )}
      </TableCell>
      <TableCell>
        <div className='space-y-1 text-xs'>
          <div>{props.item.file_name}</div>
          {props.item.update_file_name ? (
            <div>{props.item.update_file_name}</div>
          ) : null}
          {props.item.update_metadata_file_name ? (
            <div>{props.item.update_metadata_file_name}</div>
          ) : null}
        </div>
      </TableCell>
      <TableCell>{props.item.published_at || '-'}</TableCell>
      <TableCell>
        <div className='flex justify-end gap-2'>
          <Button
            variant='outline'
            size='sm'
            aria-label={t('Download installer')}
            title={t('Download installer')}
            onClick={props.onDownload}
          >
            <Download className='size-4' />
          </Button>
          <Button
            variant='outline'
            size='sm'
            aria-label={t('Edit release rollout')}
            title={t('Edit release rollout')}
            onClick={props.onEditRollout}
          >
            <SlidersHorizontal className='size-4' />
          </Button>
          {props.item.status !== 'published' ? (
            <Button size='sm' onClick={() => props.onSetStatus('published')}>
              {t('Publish')}
            </Button>
          ) : (
            <Button
              variant='outline'
              size='sm'
              onClick={() => props.onSetStatus('disabled')}
            >
              {t('Disable')}
            </Button>
          )}
          {props.item.status !== 'published' ? (
            <Button variant='destructive' size='sm' onClick={props.onDelete}>
              <Trash2 className='size-4' />
            </Button>
          ) : null}
        </div>
      </TableCell>
    </TableRow>
  )
}

function platformLabel(platform: AionUiClientPlatform) {
  const item = PLATFORM_OPTIONS.find((option) => option.value === platform)
  return item?.label ?? platform
}

function clientPackageStatusLabel(
  status: AionUiClientPackageStatus,
  t: (key: string) => string
) {
  switch (status) {
    case 'draft':
      return t('Draft')
    case 'published':
      return t('Published')
    case 'disabled':
      return t('Disabled')
    default:
      return status
  }
}

function clientPackageRolloutLabel(
  mode: 'global' | 'targeted',
  scopeCount: number,
  t: (key: string, values?: Record<string, number>) => string
) {
  if (mode === 'global') {
    return t('All users')
  }
  return t('{{count}} targets', { count: scopeCount })
}
