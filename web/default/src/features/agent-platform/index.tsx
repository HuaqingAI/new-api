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
import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  BookOpen,
  Bot,
  Boxes,
  CircleOff,
  FileJson2,
  Layers3,
  Pencil,
  PanelRightOpen,
  Plus,
  Puzzle,
  RefreshCw,
  Rocket,
  Save,
  ShieldOff,
  SquarePen,
  Unplug,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import { formatTimestamp } from '@/lib/format'
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import {
  sideDrawerContentClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import type {
  AgentPlatformItem,
  AgentPlatformLifecycleAction,
  AgentPlatformListResponse,
  AgentPlatformResourceType,
  CreateAgentPlatformResourceRequest,
  CreateAgentPlatformVersionRequest,
} from './api'
import {
  createAgentPlatformResource,
  createAgentPlatformResourceVersion,
  getAgentPlatformAgents,
  getAgentPlatformKnowledge,
  getAgentPlatformResource,
  getAgentPlatformResourceVersion,
  getAgentPlatformSkills,
  runAgentPlatformLifecycleAction,
  updateAgentPlatformResource,
} from './api'

type ResourceListCardProps = {
  badgeLabel: string
  createLabel: string
  description: string
  emptyDescription: string
  error?: unknown
  errorPrefix: string
  icon: React.ElementType
  isError: boolean
  isLoading: boolean
  onCreate: () => void
  onOpenDetails: (item: AgentPlatformItem) => void
  onOpenEdit: (item: AgentPlatformItem) => void
  onRetry: () => void
  response?: AgentPlatformListResponse
  title: string
  typeLabel: string
}

type ResourceFormState = {
  type: AgentPlatformResourceType
  displayName: string
  ownerUserId: string
  tenantId: string
}

type ResourceEditorState =
  | {
      mode: 'create'
      type: AgentPlatformResourceType
      item?: undefined
    }
  | {
      mode: 'edit'
      type: AgentPlatformResourceType
      item: AgentPlatformItem
    }

type VersionFormState = {
  version: string
  contractVersion: string
  summary: string
  schemaJson: string
  invokeMode: string
  timeoutSeconds: string
  invokeSchemaJson: string
  outputSchemaJson: string
  bindingConfigJson: string
  knowledgeMode: string
  providerType: string
  providerAdapterKey: string
  providerConfigJson: string
  querySchemaJson: string
  citationSchemaJson: string
  freshnessRulesJson: string
  providerCapabilitiesJson: string
  manifestJson: string
  dependenciesJson: string
  promptMetadataJson: string
  compatibilityMetadataJson: string
}

const CREATE_QUERY_KEYS = {
  skill: ['agent-platform', 'skills', 'summary'] as const,
  knowledge: ['agent-platform', 'knowledge', 'summary'] as const,
  agent: ['agent-platform', 'agents', 'summary'] as const,
}

const DEFAULT_CONTRACT_VERSION = '2026-06'

type SummaryMetric = {
  key: string
  label: string
  value: string | number
  helper: string
  icon: React.ElementType
}

function formatStatusLabel(
  status: string,
  t: ReturnType<typeof useTranslation>['t']
) {
  const value = status.trim().toLowerCase()
  if (!value) {
    return t('Unknown')
  }
  return t(value)
}

function statusVariantFor(value: string) {
  switch (value.trim().toLowerCase()) {
    case 'published':
      return 'success' as const
    case 'draft':
      return 'warning' as const
    case 'disabled':
    case 'offline':
      return 'neutral' as const
    case 'revoked':
      return 'danger' as const
    default:
      return 'info' as const
  }
}

function getItems(response?: AgentPlatformListResponse) {
  return response?.success && Array.isArray(response.data?.items)
    ? response.data.items
    : []
}

function getErrorMessage(error: unknown, fallback: string) {
  if (error instanceof Error && error.message) {
    return error.message
  }
  if (
    error != null &&
    typeof error === 'object' &&
    'message' in error &&
    typeof error.message === 'string' &&
    error.message.length > 0
  ) {
    return error.message
  }
  return fallback
}

function resourceTypeIcon(type: AgentPlatformResourceType) {
  switch (type) {
    case 'skill':
      return Puzzle
    case 'knowledge':
      return BookOpen
    case 'agent':
      return Bot
  }
}

function defaultResourceFormState(
  type: AgentPlatformResourceType,
  item?: AgentPlatformItem
): ResourceFormState {
  return {
    type,
    displayName: item?.display_name ?? '',
    ownerUserId: item ? String(item.owner_user_id) : '',
    tenantId: item && item.tenant_id > 0 ? String(item.tenant_id) : '',
  }
}

function defaultVersionFormState(
  resource?: AgentPlatformItem
): VersionFormState {
  const nextVersion = resource?.latest_version ? '' : '1.0.0'

  return {
    version: nextVersion,
    contractVersion: DEFAULT_CONTRACT_VERSION,
    summary: '',
    schemaJson: '{}',
    invokeMode: 'sync',
    timeoutSeconds: '30',
    invokeSchemaJson:
      '{\n  "type": "object",\n  "properties": {},\n  "additionalProperties": true\n}',
    outputSchemaJson:
      '{\n  "type": "object",\n  "properties": {},\n  "additionalProperties": true\n}',
    bindingConfigJson: '{}',
    knowledgeMode: 'retrieval',
    providerType: 'http_retrieval',
    providerAdapterKey: '',
    providerConfigJson: '{}',
    querySchemaJson:
      '{\n  "type": "object",\n  "required": ["query"],\n  "properties": {\n    "query": { "type": "string" }\n  }\n}',
    citationSchemaJson:
      '{\n  "type": "array",\n  "items": {\n    "type": "object"\n  }\n}',
    freshnessRulesJson: '{}',
    providerCapabilitiesJson: '{}',
    manifestJson: '{}',
    dependenciesJson:
      '[\n  {\n    "resource_type": "skill",\n    "resource_id": "res_replace_me"\n  }\n]',
    promptMetadataJson: '{}',
    compatibilityMetadataJson: '{}',
  }
}

function parseOptionalInt(value: string) {
  const trimmed = value.trim()
  if (!trimmed) {
    return undefined
  }
  const parsed = Number(trimmed)
  if (!Number.isInteger(parsed) || parsed < 0) {
    throw new Error('agent_platform.validation.non_negative_integer')
  }
  return parsed
}

function parseRequiredPositiveInt(value: string) {
  const trimmed = value.trim()
  const parsed = Number(trimmed)
  if (!Number.isInteger(parsed) || parsed <= 0) {
    throw new Error('agent_platform.validation.positive_integer')
  }
  return parsed
}

function parseJsonField(value: string) {
  const trimmed = value.trim()
  if (!trimmed) {
    return undefined
  }
  try {
    return JSON.parse(trimmed)
  } catch {
    throw new Error('agent_platform.validation.json')
  }
}

function translateFormError(
  error: unknown,
  t: ReturnType<typeof useTranslation>['t']
) {
  const message = getErrorMessage(error, t('Request failed'))
  switch (message) {
    case 'agent_platform.validation.non_negative_integer':
      return t('Enter a non-negative integer')
    case 'agent_platform.validation.positive_integer':
      return t('Enter a positive integer')
    case 'agent_platform.validation.json':
      return t('JSON fields must contain valid JSON')
    default:
      return message
  }
}

function formatJsonPreview(value: unknown) {
  if (value == null || value === '') {
    return ''
  }
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

function buildCreatePayload(
  form: ResourceFormState
): CreateAgentPlatformResourceRequest {
  return {
    display_name: form.displayName.trim(),
    owner_user_id: parseRequiredPositiveInt(form.ownerUserId),
    tenant_id: parseOptionalInt(form.tenantId),
  }
}

function buildVersionPayload(
  resourceType: AgentPlatformResourceType,
  form: VersionFormState
): CreateAgentPlatformVersionRequest {
  const base: CreateAgentPlatformVersionRequest = {
    version: form.version.trim(),
    contract_version: form.contractVersion.trim(),
    summary: form.summary.trim(),
    schema: parseJsonField(form.schemaJson),
  }

  switch (resourceType) {
    case 'skill':
      base.skill = {
        invoke_mode: form.invokeMode.trim(),
        timeout_seconds: parseRequiredPositiveInt(form.timeoutSeconds),
        invoke_schema: parseJsonField(form.invokeSchemaJson),
        output_schema: parseJsonField(form.outputSchemaJson),
        binding_config: parseJsonField(form.bindingConfigJson),
      }
      break
    case 'knowledge':
      base.knowledge = {
        knowledge_mode: form.knowledgeMode.trim(),
        provider_type: form.providerType.trim(),
        provider_adapter_key: form.providerAdapterKey.trim(),
        provider_config: parseJsonField(form.providerConfigJson),
        query_schema: parseJsonField(form.querySchemaJson),
        citation_schema: parseJsonField(form.citationSchemaJson),
        freshness_rules: parseJsonField(form.freshnessRulesJson),
        provider_capabilities: parseJsonField(form.providerCapabilitiesJson),
      }
      break
    case 'agent':
      base.agent = {
        manifest: parseJsonField(form.manifestJson),
        dependencies: parseJsonField(form.dependenciesJson),
        prompt_metadata: parseJsonField(form.promptMetadataJson),
        compatibility_metadata: parseJsonField(form.compatibilityMetadataJson),
      }
      break
  }

  return base
}

function ResourceListCard(props: ResourceListCardProps) {
  const { t } = useTranslation()
  const Icon = props.icon
  const items = getItems(props.response)
  const loadFailed =
    props.isError ||
    (props.response != null && props.response.success === false)
  const errorMessage =
    props.response?.message ?? getErrorMessage(props.error, t('Request failed'))

  return (
    <Card>
      <CardHeader className='gap-3 border-b'>
        <div className='flex items-start justify-between gap-3'>
          <div className='space-y-1'>
            <CardTitle className='flex items-center gap-2'>
              <span className='bg-primary/10 text-primary inline-flex size-8 items-center justify-center rounded-lg'>
                <Icon className='size-4' />
              </span>
              {props.title}
            </CardTitle>
            <CardDescription>{props.description}</CardDescription>
          </div>
          <div className='flex shrink-0 items-center gap-2'>
            <Badge variant='outline' className='hidden sm:inline-flex'>
              {props.badgeLabel}
            </Badge>
            <Button size='sm' onClick={props.onCreate}>
              <Plus className='size-4' />
              {props.createLabel}
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className='pt-4'>
        {props.isLoading ? (
          <ResourceTableSkeleton />
        ) : loadFailed ? (
          <ErrorState
            className='min-h-[240px]'
            title={t('Failed to load data')}
            description={`${props.errorPrefix}: ${errorMessage}`}
            onRetry={props.onRetry}
          />
        ) : items.length === 0 ? (
          <Empty className='min-h-[240px] border'>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <Icon className='text-muted-foreground size-5' />
              </EmptyMedia>
              <EmptyTitle>{t('No resources found')}</EmptyTitle>
              <EmptyDescription>{props.emptyDescription}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent />
          </Empty>
        ) : (
          <ResourceTable
            items={items}
            onOpenDetails={props.onOpenDetails}
            onOpenEdit={props.onOpenEdit}
            typeLabel={props.typeLabel}
          />
        )}
      </CardContent>
    </Card>
  )
}

function ResourceTable(props: {
  items: AgentPlatformItem[]
  onOpenDetails: (item: AgentPlatformItem) => void
  onOpenEdit: (item: AgentPlatformItem) => void
  typeLabel: string
}) {
  const { t } = useTranslation()

  return (
    <div className='overflow-x-auto rounded-md border'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Resource')}</TableHead>
            <TableHead>{t('Type')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Owner')}</TableHead>
            <TableHead>{t('Latest version')}</TableHead>
            <TableHead>{t('Tenant')}</TableHead>
            <TableHead>{t('Updated At')}</TableHead>
            <TableHead className='text-right'>{t('Actions')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {props.items.map((item) => (
            <ResourceRow
              key={item.resource_id}
              item={item}
              onOpenDetails={props.onOpenDetails}
              onOpenEdit={props.onOpenEdit}
              typeLabel={props.typeLabel}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function ResourceTableSkeleton() {
  return (
    <div className='space-y-3'>
      <Skeleton className='h-10 w-full' />
      <Skeleton className='h-12 w-full' />
      <Skeleton className='h-12 w-full' />
      <Skeleton className='h-12 w-4/5' />
    </div>
  )
}

function ResourceRow(props: {
  item: AgentPlatformItem
  onOpenDetails: (item: AgentPlatformItem) => void
  onOpenEdit: (item: AgentPlatformItem) => void
  typeLabel: string
}) {
  const { t } = useTranslation()
  const item = props.item

  return (
    <TableRow>
      <TableCell>
        <div className='min-w-[220px] space-y-1'>
          <div className='font-medium'>{item.display_name}</div>
          <div className='text-muted-foreground text-xs'>
            {item.resource_id}
          </div>
        </div>
      </TableCell>
      <TableCell>
        <Badge variant='outline'>{props.typeLabel}</Badge>
      </TableCell>
      <TableCell>
        <StatusBadge
          label={formatStatusLabel(item.status, t)}
          variant={statusVariantFor(item.status)}
          copyable={false}
        />
      </TableCell>
      <TableCell>#{item.owner_user_id}</TableCell>
      <TableCell>{item.latest_version || t('Not versioned')}</TableCell>
      <TableCell>{item.tenant_id || t('Global')}</TableCell>
      <TableCell>{formatTimestamp(item.updated_at)}</TableCell>
      <TableCell className='text-right'>
        <div className='flex justify-end gap-2'>
          <Button
            variant='outline'
            size='sm'
            onClick={() => props.onOpenDetails(item)}
          >
            <PanelRightOpen className='size-4' />
            {t('Details')}
          </Button>
          <Button
            variant='ghost'
            size='sm'
            onClick={() => props.onOpenEdit(item)}
          >
            <SquarePen className='size-4' />
            {t('Edit')}
          </Button>
        </div>
      </TableCell>
    </TableRow>
  )
}

function countResourcesByStatus(
  resourceSets: AgentPlatformItem[][],
  status: string
) {
  return resourceSets.reduce(
    (total, items) =>
      total +
      items.filter((item) => item.status.trim().toLowerCase() === status)
        .length,
    0
  )
}

function resourceTypeLabel(
  type: AgentPlatformResourceType,
  t: (key: string) => string
) {
  switch (type) {
    case 'skill':
      return t('Skill')
    case 'knowledge':
      return t('Knowledge')
    case 'agent':
      return t('Agent')
  }
}

function ResourceEditorDialog(props: {
  form: ResourceFormState
  mode: 'create' | 'edit'
  onChange: (form: ResourceFormState) => void
  onClose: () => void
  onSubmit: () => void
  open: boolean
  pending: boolean
}) {
  const { t } = useTranslation()
  const Icon = resourceTypeIcon(props.form.type)
  const isEdit = props.mode === 'edit'
  const title = isEdit
    ? t('Edit {{type}} resource', {
        type: resourceTypeLabel(props.form.type, t),
      })
    : t('Create {{type}} resource', {
        type: resourceTypeLabel(props.form.type, t),
      })

  return (
    <Dialog open={props.open} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle className='flex items-center gap-2'>
            <span className='bg-primary/10 text-primary inline-flex size-8 items-center justify-center rounded-lg'>
              <Icon className='size-4' />
            </span>
            {title}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? t('Update the resource display name.')
              : t(
                  'Create a control-plane resource shell before adding versions.'
                )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <Field>
            <FieldLabel>{t('Display name')}</FieldLabel>
            <Input
              value={props.form.displayName}
              onChange={(event) =>
                props.onChange({
                  ...props.form,
                  displayName: event.target.value,
                })
              }
              placeholder={t('Resource display name')}
            />
          </Field>
          <div className='grid gap-4 sm:grid-cols-2'>
            <Field data-disabled={isEdit ? true : undefined}>
              <FieldLabel>{t('Owner user ID')}</FieldLabel>
              <Input
                type='number'
                min={1}
                value={props.form.ownerUserId}
                onChange={(event) =>
                  props.onChange({
                    ...props.form,
                    ownerUserId: event.target.value,
                  })
                }
                disabled={isEdit}
                placeholder='1'
              />
              {isEdit ? (
                <FieldDescription>
                  {t('Owner is fixed after creation.')}
                </FieldDescription>
              ) : null}
            </Field>
            <Field data-disabled={isEdit ? true : undefined}>
              <FieldLabel>{t('Tenant ID')}</FieldLabel>
              <Input
                type='number'
                min={0}
                value={props.form.tenantId}
                onChange={(event) =>
                  props.onChange({
                    ...props.form,
                    tenantId: event.target.value,
                  })
                }
                disabled={isEdit}
                placeholder={t('Global')}
              />
              {isEdit ? (
                <FieldDescription>
                  {t('Tenant is fixed after creation.')}
                </FieldDescription>
              ) : null}
            </Field>
          </div>
        </FieldGroup>

        <DialogFooter>
          <Button variant='outline' onClick={props.onClose}>
            {t('Cancel')}
          </Button>
          <Button onClick={props.onSubmit} disabled={props.pending}>
            {isEdit ? <Save className='size-4' /> : <Plus className='size-4' />}
            {isEdit ? t('Save changes') : t('Create resource')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function DetailField(props: { label: string; value: React.ReactNode }) {
  return (
    <div className='rounded-lg border px-3 py-2'>
      <div className='text-muted-foreground text-xs'>{props.label}</div>
      <div className='mt-1 min-h-5 text-sm font-medium break-all'>
        {props.value}
      </div>
    </div>
  )
}

function JsonPreviewBlock(props: { label: string; value: unknown }) {
  const formatted = formatJsonPreview(props.value)
  if (!formatted) {
    return null
  }

  return (
    <div className='space-y-2'>
      <div className='text-sm font-medium'>{props.label}</div>
      <pre className='bg-muted/40 max-h-56 overflow-auto rounded-lg border p-3 text-xs leading-relaxed'>
        {formatted}
      </pre>
    </div>
  )
}

function ResourceDetailSheet(props: {
  detailResponse?: Awaited<ReturnType<typeof getAgentPlatformResource>>
  detailLoading: boolean
  lifecyclePending: boolean
  onClose: () => void
  onLifecycle: (action: AgentPlatformLifecycleAction) => void
  onOpenEdit: (item: AgentPlatformItem) => void
  onOpenVersionForm: () => void
  onRefreshDetail: () => void
  onVersionLookup: () => void
  open: boolean
  selected: AgentPlatformItem | null
  versionResponse?: Awaited<ReturnType<typeof getAgentPlatformResourceVersion>>
  versionLoading: boolean
}) {
  const { t } = useTranslation()
  const item = props.selected ?? props.detailResponse?.data
  const Icon = item ? resourceTypeIcon(item.resource_type) : Boxes
  const version = props.versionResponse?.success
    ? props.versionResponse.data
    : undefined

  return (
    <Sheet open={props.open} onOpenChange={(open) => !open && props.onClose()}>
      <SheetContent className={sideDrawerContentClassName('sm:max-w-3xl')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle className='flex items-center gap-2 pr-8'>
            <span className='bg-primary/10 text-primary inline-flex size-8 items-center justify-center rounded-lg'>
              <Icon className='size-4' />
            </span>
            {item?.display_name ?? t('Resource details')}
          </SheetTitle>
          <SheetDescription className='pr-8 break-all'>
            {item?.resource_id ?? t('Select a resource to inspect.')}
          </SheetDescription>
        </SheetHeader>

        <div className='flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-4 py-4 sm:px-6'>
          {item ? (
            <>
              <section className='space-y-3'>
                <div className='flex flex-wrap items-center justify-between gap-2'>
                  <div className='flex items-center gap-2'>
                    <Badge variant='outline'>
                      {resourceTypeLabel(item.resource_type, t)}
                    </Badge>
                    <StatusBadge
                      label={formatStatusLabel(item.status, t)}
                      variant={statusVariantFor(item.status)}
                      copyable={false}
                    />
                  </div>
                  <div className='flex flex-wrap gap-2'>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={props.onRefreshDetail}
                      disabled={props.detailLoading}
                    >
                      <RefreshCw className='size-4' />
                      {t('Refresh')}
                    </Button>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={() => props.onOpenEdit(item)}
                    >
                      <Pencil className='size-4' />
                      {t('Edit')}
                    </Button>
                  </div>
                </div>
                <div className='grid gap-3 sm:grid-cols-2'>
                  <DetailField
                    label={t('Resource ID')}
                    value={item.resource_id}
                  />
                  <DetailField
                    label={t('Latest version')}
                    value={item.latest_version || t('Not versioned')}
                  />
                  <DetailField
                    label={t('Owner user ID')}
                    value={`#${item.owner_user_id}`}
                  />
                  <DetailField
                    label={t('Tenant ID')}
                    value={item.tenant_id || t('Global')}
                  />
                  <DetailField
                    label={t('Created At')}
                    value={formatTimestamp(item.created_at)}
                  />
                  <DetailField
                    label={t('Updated At')}
                    value={formatTimestamp(item.updated_at)}
                  />
                </div>
              </section>

              <section className='space-y-3 border-t pt-5'>
                <div className='flex flex-wrap items-center justify-between gap-3'>
                  <div>
                    <h3 className='text-sm font-semibold'>{t('Version')}</h3>
                    <p className='text-muted-foreground text-sm'>
                      {t('Create and inspect typed contract versions.')}
                    </p>
                  </div>
                  <div className='flex flex-wrap gap-2'>
                    <Button size='sm' onClick={props.onOpenVersionForm}>
                      <FileJson2 className='size-4' />
                      {t('Create version')}
                    </Button>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={props.onVersionLookup}
                      disabled={!item.latest_version || props.versionLoading}
                    >
                      <PanelRightOpen className='size-4' />
                      {t('Load latest version')}
                    </Button>
                  </div>
                </div>
                {version ? (
                  <div className='space-y-4 rounded-lg border p-3'>
                    <div className='flex flex-wrap items-center gap-2'>
                      <Badge variant='outline'>{version.version}</Badge>
                      <Badge variant='outline'>
                        {version.contract_version}
                      </Badge>
                      <StatusBadge
                        label={formatStatusLabel(version.status, t)}
                        variant={statusVariantFor(version.status)}
                        copyable={false}
                      />
                    </div>
                    <div className='grid gap-3 sm:grid-cols-2'>
                      <DetailField
                        label={t('Created by')}
                        value={`#${version.created_by}`}
                      />
                      <DetailField
                        label={t('Published At')}
                        value={
                          version.published_at
                            ? formatTimestamp(version.published_at)
                            : t('Not published')
                        }
                      />
                    </div>
                    {version.summary ? (
                      <p className='text-muted-foreground text-sm'>
                        {version.summary}
                      </p>
                    ) : null}
                    <JsonPreviewBlock
                      label={t('Schema JSON')}
                      value={version.schema}
                    />
                    <JsonPreviewBlock
                      label={t('Typed detail JSON')}
                      value={
                        version.skill ??
                        version.knowledge ??
                        version.agent ??
                        null
                      }
                    />
                  </div>
                ) : (
                  <div className='text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-sm'>
                    {props.versionLoading
                      ? t('Loading version details...')
                      : t('No version detail loaded.')}
                  </div>
                )}
              </section>

              <section className='space-y-3 border-t pt-5'>
                <div>
                  <h3 className='text-sm font-semibold'>{t('Lifecycle')}</h3>
                  <p className='text-muted-foreground text-sm'>
                    {t(
                      'Publish or move the resource through governance states.'
                    )}
                  </p>
                </div>
                <div className='grid gap-2 sm:grid-cols-4'>
                  <Button
                    size='sm'
                    onClick={() => props.onLifecycle('publish')}
                    disabled={!item.latest_version || props.lifecyclePending}
                  >
                    <Rocket className='size-4' />
                    {t('Publish')}
                  </Button>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => props.onLifecycle('disable')}
                    disabled={props.lifecyclePending}
                  >
                    <CircleOff className='size-4' />
                    {t('Disable')}
                  </Button>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => props.onLifecycle('offline')}
                    disabled={props.lifecyclePending}
                  >
                    <Unplug className='size-4' />
                    {t('Offline')}
                  </Button>
                  <Button
                    variant='destructive'
                    size='sm'
                    onClick={() => props.onLifecycle('revoke')}
                    disabled={props.lifecyclePending}
                  >
                    <ShieldOff className='size-4' />
                    {t('Revoke')}
                  </Button>
                </div>
              </section>
            </>
          ) : (
            <ResourceTableSkeleton />
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

function JsonTextareaField(props: {
  label: string
  value: string
  onChange: (value: string) => void
  rows?: number
}) {
  return (
    <Field>
      <FieldLabel>{props.label}</FieldLabel>
      <Textarea
        className='font-mono text-xs'
        rows={props.rows ?? 5}
        value={props.value}
        onChange={(event) => props.onChange(event.target.value)}
      />
    </Field>
  )
}

function VersionFormDialog(props: {
  form: VersionFormState
  onChange: (form: VersionFormState) => void
  onClose: () => void
  onSubmit: () => void
  open: boolean
  pending: boolean
  resource: AgentPlatformItem | null
}) {
  const { t } = useTranslation()
  const resource = props.resource
  const Icon = resource ? resourceTypeIcon(resource.resource_type) : FileJson2

  const update = <K extends keyof VersionFormState>(
    key: K,
    value: VersionFormState[K]
  ) => props.onChange({ ...props.form, [key]: value })

  return (
    <Dialog open={props.open} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle className='flex items-center gap-2'>
            <span className='bg-primary/10 text-primary inline-flex size-8 items-center justify-center rounded-lg'>
              <Icon className='size-4' />
            </span>
            {t('Create version')}
          </DialogTitle>
          <DialogDescription className='break-all'>
            {resource?.resource_id ?? t('No resource selected')}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <div className='grid gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel>{t('Version')}</FieldLabel>
              <Input
                value={props.form.version}
                onChange={(event) => update('version', event.target.value)}
                placeholder='1.0.0'
              />
            </Field>
            <Field>
              <FieldLabel>{t('Contract version')}</FieldLabel>
              <Input
                value={props.form.contractVersion}
                onChange={(event) =>
                  update('contractVersion', event.target.value)
                }
                placeholder={DEFAULT_CONTRACT_VERSION}
              />
            </Field>
          </div>
          <Field>
            <FieldLabel>{t('Summary')}</FieldLabel>
            <Textarea
              rows={3}
              value={props.form.summary}
              onChange={(event) => update('summary', event.target.value)}
              placeholder={t('Version summary')}
            />
          </Field>
          <JsonTextareaField
            label={t('Schema JSON')}
            value={props.form.schemaJson}
            onChange={(value) => update('schemaJson', value)}
          />

          {resource?.resource_type === 'skill' ? (
            <div className='space-y-4 rounded-lg border p-3'>
              <div className='text-sm font-semibold'>{t('Skill detail')}</div>
              <div className='grid gap-4 sm:grid-cols-2'>
                <Field>
                  <FieldLabel>{t('Invoke mode')}</FieldLabel>
                  <select
                    className='border-input bg-background h-8 rounded-lg border px-2.5 text-sm'
                    value={props.form.invokeMode}
                    onChange={(event) =>
                      update('invokeMode', event.target.value)
                    }
                  >
                    <option value='sync'>{t('sync')}</option>
                    <option value='async'>{t('async')}</option>
                  </select>
                </Field>
                <Field>
                  <FieldLabel>{t('Timeout seconds')}</FieldLabel>
                  <Input
                    type='number'
                    min={1}
                    value={props.form.timeoutSeconds}
                    onChange={(event) =>
                      update('timeoutSeconds', event.target.value)
                    }
                  />
                </Field>
              </div>
              <JsonTextareaField
                label={t('Invoke schema')}
                value={props.form.invokeSchemaJson}
                onChange={(value) => update('invokeSchemaJson', value)}
              />
              <JsonTextareaField
                label={t('Output schema')}
                value={props.form.outputSchemaJson}
                onChange={(value) => update('outputSchemaJson', value)}
              />
              <JsonTextareaField
                label={t('Binding config')}
                value={props.form.bindingConfigJson}
                onChange={(value) => update('bindingConfigJson', value)}
              />
            </div>
          ) : null}

          {resource?.resource_type === 'knowledge' ? (
            <div className='space-y-4 rounded-lg border p-3'>
              <div className='text-sm font-semibold'>
                {t('Knowledge detail')}
              </div>
              <div className='grid gap-4 sm:grid-cols-3'>
                <Field>
                  <FieldLabel>{t('Knowledge mode')}</FieldLabel>
                  <Input
                    value={props.form.knowledgeMode}
                    onChange={(event) =>
                      update('knowledgeMode', event.target.value)
                    }
                  />
                </Field>
                <Field>
                  <FieldLabel>{t('Provider type')}</FieldLabel>
                  <select
                    className='border-input bg-background h-8 rounded-lg border px-2.5 text-sm'
                    value={props.form.providerType}
                    onChange={(event) =>
                      update('providerType', event.target.value)
                    }
                  >
                    <option value='http_retrieval'>
                      {t('http_retrieval')}
                    </option>
                    <option value='native'>{t('native')}</option>
                  </select>
                </Field>
                <Field>
                  <FieldLabel>{t('Provider adapter key')}</FieldLabel>
                  <Input
                    value={props.form.providerAdapterKey}
                    onChange={(event) =>
                      update('providerAdapterKey', event.target.value)
                    }
                  />
                </Field>
              </div>
              <JsonTextareaField
                label={t('Provider config')}
                value={props.form.providerConfigJson}
                onChange={(value) => update('providerConfigJson', value)}
              />
              <JsonTextareaField
                label={t('Query schema')}
                value={props.form.querySchemaJson}
                onChange={(value) => update('querySchemaJson', value)}
              />
              <JsonTextareaField
                label={t('Citation schema')}
                value={props.form.citationSchemaJson}
                onChange={(value) => update('citationSchemaJson', value)}
              />
              <JsonTextareaField
                label={t('Freshness rules')}
                value={props.form.freshnessRulesJson}
                onChange={(value) => update('freshnessRulesJson', value)}
              />
              <JsonTextareaField
                label={t('Provider capabilities')}
                value={props.form.providerCapabilitiesJson}
                onChange={(value) => update('providerCapabilitiesJson', value)}
              />
            </div>
          ) : null}

          {resource?.resource_type === 'agent' ? (
            <div className='space-y-4 rounded-lg border p-3'>
              <div className='text-sm font-semibold'>{t('Agent detail')}</div>
              <JsonTextareaField
                label={t('Manifest')}
                value={props.form.manifestJson}
                onChange={(value) => update('manifestJson', value)}
              />
              <JsonTextareaField
                label={t('Dependencies')}
                value={props.form.dependenciesJson}
                onChange={(value) => update('dependenciesJson', value)}
              />
              <JsonTextareaField
                label={t('Prompt metadata')}
                value={props.form.promptMetadataJson}
                onChange={(value) => update('promptMetadataJson', value)}
              />
              <JsonTextareaField
                label={t('Compatibility metadata')}
                value={props.form.compatibilityMetadataJson}
                onChange={(value) => update('compatibilityMetadataJson', value)}
              />
            </div>
          ) : null}
        </FieldGroup>

        <DialogFooter>
          <Button variant='outline' onClick={props.onClose}>
            {t('Cancel')}
          </Button>
          <Button
            onClick={props.onSubmit}
            disabled={props.pending || !resource}
          >
            <FileJson2 className='size-4' />
            {t('Create version')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function AgentPlatformShell() {
  const { t } = useTranslation()
  const userRole = useAuthStore((state) => state.auth.user?.role ?? ROLE.GUEST)
  const isSuperAdmin = userRole >= ROLE.SUPER_ADMIN
  const queryClient = useQueryClient()
  const [editor, setEditor] = useState<ResourceEditorState | null>(null)
  const [resourceForm, setResourceForm] = useState<ResourceFormState>(
    defaultResourceFormState('skill')
  )
  const [selectedResource, setSelectedResource] =
    useState<AgentPlatformItem | null>(null)
  const [versionFormOpen, setVersionFormOpen] = useState(false)
  const [versionForm, setVersionForm] = useState<VersionFormState>(
    defaultVersionFormState()
  )

  const skillsQuery = useQuery({
    queryKey: ['agent-platform', 'skills', 'summary'],
    queryFn: getAgentPlatformSkills,
  })
  const knowledgeQuery = useQuery({
    queryKey: ['agent-platform', 'knowledge', 'summary'],
    queryFn: getAgentPlatformKnowledge,
  })
  const agentsQuery = useQuery({
    queryKey: ['agent-platform', 'agents', 'summary'],
    queryFn: getAgentPlatformAgents,
  })
  const detailQuery = useQuery({
    queryKey: [
      'agent-platform',
      selectedResource?.resource_type,
      selectedResource?.resource_id,
      'detail',
    ],
    queryFn: () => {
      if (!selectedResource) {
        throw new Error('No resource selected')
      }
      return getAgentPlatformResource(
        selectedResource.resource_type,
        selectedResource.resource_id
      )
    },
    enabled: selectedResource != null,
  })
  const versionQuery = useQuery({
    queryKey: [
      'agent-platform',
      selectedResource?.resource_id,
      selectedResource?.latest_version,
      'version',
    ],
    queryFn: () => {
      if (!selectedResource?.latest_version) {
        throw new Error('No version selected')
      }
      return getAgentPlatformResourceVersion(
        selectedResource.resource_id,
        selectedResource.latest_version
      )
    },
    enabled: false,
  })

  const invalidateResourceList = async (type: AgentPlatformResourceType) => {
    await queryClient.invalidateQueries({ queryKey: CREATE_QUERY_KEYS[type] })
  }

  const createMutation = useMutation({
    mutationFn: (form: ResourceFormState) =>
      createAgentPlatformResource(form.type, buildCreatePayload(form)),
    onSuccess: async (response, form) => {
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      toast.success(t('Resource created'))
      setEditor(null)
      await invalidateResourceList(form.type)
      if (response.data) {
        setSelectedResource(response.data)
      }
    },
    onError: (error) => {
      toast.error(translateFormError(error, t))
    },
  })

  const updateMutation = useMutation({
    mutationFn: (input: { item: AgentPlatformItem; displayName: string }) =>
      updateAgentPlatformResource(
        input.item.resource_type,
        input.item.resource_id,
        { display_name: input.displayName.trim() }
      ),
    onSuccess: async (response, input) => {
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      toast.success(t('Resource updated'))
      setEditor(null)
      await invalidateResourceList(input.item.resource_type)
      if (response.data) {
        setSelectedResource(response.data)
      }
    },
    onError: (error) => {
      toast.error(translateFormError(error, t))
    },
  })

  const versionMutation = useMutation({
    mutationFn: (input: {
      resource: AgentPlatformItem
      form: VersionFormState
    }) =>
      createAgentPlatformResourceVersion(
        input.resource.resource_id,
        buildVersionPayload(input.resource.resource_type, input.form)
      ),
    onSuccess: async (response, input) => {
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      toast.success(t('Version created'))
      setVersionFormOpen(false)
      await invalidateResourceList(input.resource.resource_type)
      if (response.data) {
        setSelectedResource({
          ...input.resource,
          latest_version: response.data.version,
        })
        queryClient.setQueryData(
          [
            'agent-platform',
            input.resource.resource_id,
            response.data.version,
            'version',
          ],
          response
        )
      }
    },
    onError: (error) => {
      toast.error(getErrorMessage(error, t('Request failed')))
    },
  })

  const lifecycleMutation = useMutation({
    mutationFn: (input: {
      action: AgentPlatformLifecycleAction
      resource: AgentPlatformItem
    }) =>
      runAgentPlatformLifecycleAction(
        input.resource.resource_id,
        input.action,
        {
          version:
            input.action === 'publish'
              ? input.resource.latest_version
              : undefined,
          request_id: `web-${Date.now()}`,
        }
      ),
    onSuccess: async (response, input) => {
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      toast.success(t('Lifecycle action queued'))
      await invalidateResourceList(input.resource.resource_type)
      if (response.data) {
        queryClient.setQueryData(
          [
            'agent-platform',
            input.resource.resource_id,
            response.data.current_version || input.resource.latest_version,
            'version',
          ],
          (
            current: Awaited<ReturnType<typeof getAgentPlatformResourceVersion>>
          ) =>
            current?.success && current.data
              ? {
                  ...current,
                  data: {
                    ...current.data,
                    status:
                      response.data?.current_status ?? current.data.status,
                  },
                }
              : current
        )
        setSelectedResource({
          ...input.resource,
          status: response.data.current_status,
          latest_version:
            response.data.current_version || input.resource.latest_version,
        })
      }
    },
    onError: (error) => {
      toast.error(getErrorMessage(error, t('Request failed')))
    },
  })

  const openCreateEditor = (type: AgentPlatformResourceType) => {
    setEditor({ mode: 'create', type })
    setResourceForm(defaultResourceFormState(type))
  }

  const openEditEditor = (item: AgentPlatformItem) => {
    setEditor({ mode: 'edit', type: item.resource_type, item })
    setResourceForm(defaultResourceFormState(item.resource_type, item))
  }

  const handleResourceSubmit = () => {
    if (!resourceForm.displayName.trim()) {
      toast.error(t('Display name is required'))
      return
    }
    try {
      if (editor?.mode === 'edit') {
        updateMutation.mutate({
          item: editor.item,
          displayName: resourceForm.displayName,
        })
      } else if (editor?.mode === 'create') {
        createMutation.mutate(resourceForm)
      }
    } catch (error) {
      toast.error(translateFormError(error, t))
    }
  }

  const openVersionForm = () => {
    setVersionForm(defaultVersionFormState(selectedResource ?? undefined))
    setVersionFormOpen(true)
  }

  const handleVersionSubmit = () => {
    if (!selectedResource) {
      return
    }
    if (!versionForm.version.trim() || !versionForm.contractVersion.trim()) {
      toast.error(t('Version and contract version are required'))
      return
    }
    try {
      versionMutation.mutate({
        resource: selectedResource,
        form: versionForm,
      })
    } catch (error) {
      toast.error(translateFormError(error, t))
    }
  }

  const handleLifecycle = (action: AgentPlatformLifecycleAction) => {
    if (!selectedResource) {
      return
    }
    if (action === 'publish' && !selectedResource.latest_version) {
      toast.error(t('Create a version before publishing'))
      return
    }
    lifecycleMutation.mutate({ action, resource: selectedResource })
  }

  const skillItems = getItems(skillsQuery.data)
  const knowledgeItems = getItems(knowledgeQuery.data)
  const agentItems = getItems(agentsQuery.data)
  const resourceSets = [skillItems, knowledgeItems, agentItems]

  const metrics: SummaryMetric[] = [
    {
      key: 'skills',
      label: t('Skills'),
      value: skillItems.length,
      helper: t('Live control-plane entries'),
      icon: Puzzle,
    },
    {
      key: 'knowledge',
      label: t('Knowledge'),
      value: knowledgeItems.length,
      helper: t('Provider-backed resources'),
      icon: BookOpen,
    },
    {
      key: 'agents',
      label: t('Agents'),
      value: agentItems.length,
      helper: t('Definition templates'),
      icon: Bot,
    },
    {
      key: 'published',
      label: t('Published resources'),
      value: countResourcesByStatus(resourceSets, 'published'),
      helper: t('Ready for open capability exposure'),
      icon: Layers3,
    },
  ]

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Agent Platform')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <div className='flex items-center gap-2'>
            <Button
              variant='outline'
              size='sm'
              onClick={() => {
                void skillsQuery.refetch()
                void knowledgeQuery.refetch()
                void agentsQuery.refetch()
              }}
              disabled={
                skillsQuery.isFetching ||
                knowledgeQuery.isFetching ||
                agentsQuery.isFetching
              }
            >
              <RefreshCw className='size-4' />
              {t('Refresh')}
            </Button>
            {isSuperAdmin && (
              <Button
                variant='outline'
                size='sm'
                render={
                  <Link to='/system-settings/site'>{t('System Settings')}</Link>
                }
              />
            )}
          </div>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='space-y-6'>
            <Card>
              <CardHeader className='gap-3 border-b'>
                <div className='flex items-start gap-3'>
                  <span className='bg-primary/10 text-primary inline-flex size-10 items-center justify-center rounded-xl'>
                    <Boxes className='size-5' />
                  </span>
                  <div className='space-y-1'>
                    <CardTitle>{t('Resource control plane')}</CardTitle>
                    <CardDescription>
                      {t(
                        'Manage Skill, Knowledge, and Agent resources from the live Agent Platform control-plane endpoints.'
                      )}
                    </CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent className='pt-4'>
                <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
                  {metrics.map((metric) => {
                    const Icon = metric.icon
                    return (
                      <div
                        key={metric.key}
                        className='bg-muted/20 rounded-xl border px-4 py-3'
                      >
                        <div className='flex items-center justify-between gap-3'>
                          <span className='text-muted-foreground text-sm font-medium'>
                            {metric.label}
                          </span>
                          <Icon className='text-muted-foreground size-4' />
                        </div>
                        <div className='mt-2 text-2xl font-semibold'>
                          {metric.value}
                        </div>
                        <p className='text-muted-foreground mt-1 text-xs'>
                          {metric.helper}
                        </p>
                      </div>
                    )
                  })}
                </div>
              </CardContent>
            </Card>

            <Tabs defaultValue='skills' className='space-y-6'>
              <TabsList className='grid w-full grid-cols-3 md:w-[420px]'>
                <TabsTrigger value='skills'>{t('Skills')}</TabsTrigger>
                <TabsTrigger value='knowledge'>{t('Knowledge')}</TabsTrigger>
                <TabsTrigger value='agents'>{t('Agents')}</TabsTrigger>
              </TabsList>

              <TabsContent value='skills'>
                <ResourceListCard
                  title={t('Skill management')}
                  description={t(
                    'Skill definitions exposed by the Agent Platform control plane.'
                  )}
                  badgeLabel={t('Epic 3 active')}
                  createLabel={t('New Skill')}
                  icon={Puzzle}
                  typeLabel={resourceTypeLabel('skill', t)}
                  response={skillsQuery.data}
                  isError={skillsQuery.isError}
                  error={skillsQuery.error}
                  isLoading={skillsQuery.isLoading}
                  onCreate={() => openCreateEditor('skill')}
                  onOpenDetails={setSelectedResource}
                  onOpenEdit={openEditEditor}
                  onRetry={() => void skillsQuery.refetch()}
                  errorPrefix={t('Skill API request failed')}
                  emptyDescription={t(
                    'No Skill resources are currently available in the control plane.'
                  )}
                />
              </TabsContent>

              <TabsContent value='knowledge'>
                <ResourceListCard
                  title={t('Knowledge management')}
                  description={t(
                    'Provider-backed retrieval resources managed by the Agent Platform control plane.'
                  )}
                  badgeLabel={t('Epic 4 active')}
                  createLabel={t('New Knowledge')}
                  icon={BookOpen}
                  typeLabel={resourceTypeLabel('knowledge', t)}
                  response={knowledgeQuery.data}
                  isError={knowledgeQuery.isError}
                  error={knowledgeQuery.error}
                  isLoading={knowledgeQuery.isLoading}
                  onCreate={() => openCreateEditor('knowledge')}
                  onOpenDetails={setSelectedResource}
                  onOpenEdit={openEditEditor}
                  onRetry={() => void knowledgeQuery.refetch()}
                  errorPrefix={t('Knowledge API request failed')}
                  emptyDescription={t(
                    'No Knowledge resources are currently available in the control plane.'
                  )}
                />
              </TabsContent>

              <TabsContent value='agents'>
                <ResourceListCard
                  title={t('Agent definitions')}
                  description={t(
                    'Agent definition templates managed by the control plane without implying server-side runtime ownership.'
                  )}
                  badgeLabel={t('Epic 5 active')}
                  createLabel={t('New Agent')}
                  icon={Bot}
                  typeLabel={resourceTypeLabel('agent', t)}
                  response={agentsQuery.data}
                  isError={agentsQuery.isError}
                  error={agentsQuery.error}
                  isLoading={agentsQuery.isLoading}
                  onCreate={() => openCreateEditor('agent')}
                  onOpenDetails={setSelectedResource}
                  onOpenEdit={openEditEditor}
                  onRetry={() => void agentsQuery.refetch()}
                  errorPrefix={t('Agent API request failed')}
                  emptyDescription={t(
                    'No Agent definition resources are currently available in the control plane.'
                  )}
                />
              </TabsContent>
            </Tabs>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>
      <ResourceEditorDialog
        form={resourceForm}
        mode={editor?.mode ?? 'create'}
        onChange={setResourceForm}
        onClose={() => setEditor(null)}
        onSubmit={handleResourceSubmit}
        open={editor != null}
        pending={createMutation.isPending || updateMutation.isPending}
      />
      <ResourceDetailSheet
        detailResponse={detailQuery.data}
        detailLoading={detailQuery.isFetching}
        lifecyclePending={lifecycleMutation.isPending}
        onClose={() => setSelectedResource(null)}
        onLifecycle={handleLifecycle}
        onOpenEdit={openEditEditor}
        onOpenVersionForm={openVersionForm}
        onRefreshDetail={() => void detailQuery.refetch()}
        onVersionLookup={() => void versionQuery.refetch()}
        open={selectedResource != null}
        selected={selectedResource}
        versionResponse={versionQuery.data}
        versionLoading={versionQuery.isFetching}
      />
      <VersionFormDialog
        form={versionForm}
        onChange={setVersionForm}
        onClose={() => setVersionFormOpen(false)}
        onSubmit={handleVersionSubmit}
        open={versionFormOpen}
        pending={versionMutation.isPending}
        resource={selectedResource}
      />
    </>
  )
}
