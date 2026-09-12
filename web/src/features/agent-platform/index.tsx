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
import { Link } from '@tanstack/react-router'
import {
  BookOpen,
  Bot,
  Boxes,
  CheckCircle2,
  Download,
  Image,
  PanelRightOpen,
  Plus,
  Puzzle,
  RefreshCw,
  Rocket,
  Save,
  SquarePen,
  Trash2,
  Unplug,
  XCircle,
} from 'lucide-react'
import {
  useEffect,
  useMemo,
  useState,
  type ElementType,
  type ReactNode,
} from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  sideDrawerContentClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
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
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
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
import { getDepartmentTree } from '@/features/enterprise-organization/api'
import type { DepartmentTreeNode } from '@/features/enterprise-organization/types'
import { searchUsers } from '@/features/users/api'
import type { User } from '@/features/users/types'
import { formatTimestamp } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import {
  type AgentPlatformAgentItem,
  type AgentPlatformAgentCliType,
  type AgentPlatformAgentVersion,
  type AgentPlatformGrantRequest,
  type AgentPlatformItem,
  type AgentPlatformItemResponse,
  type AgentPlatformJsonValue,
  type AgentPlatformListResponse,
  type AgentPlatformResourceType,
  createAgentPlatformAgent,
  createAgentPlatformKnowledge,
  createAgentPlatformMcp,
  createAgentPlatformSkill,
  deleteAgentPlatformResource,
  getAgentPlatformAgents,
  getAgentPlatformAgentVersionGrants,
  getAgentPlatformAgentVersions,
  getAgentPlatformKnowledge,
  getAgentPlatformMcps,
  getAgentPlatformPublishDefaults,
  getAgentPlatformResource,
  getAgentPlatformSkillPackageDownloadUrl,
  getAgentPlatformSkills,
  publishAgentPlatformAgent,
  setAgentPlatformResourceEnabled,
  updateAgentPlatformAgent,
  updateAgentPlatformKnowledge,
  updateAgentPlatformMcp,
  updateAgentPlatformSkill,
  uploadAgentPlatformAvatar,
  uploadAgentPlatformSkillPackage,
} from './api'
import { DepartmentGrantTreeOptions } from './components/department-grant-tree-options'
import {
  GrantSubjectMultiSelect,
  type GrantSelectOption,
} from './components/grant-subject-multi-select'

type ResourceListCardProps = {
  badgeLabel: string
  createLabel: string
  description: string
  emptyDescription: string
  error?: unknown
  errorPrefix: string
  icon: ElementType
  isError: boolean
  isLoading: boolean
  onCreate: () => void
  onOpenDetails: (item: AgentPlatformItem) => void
  onOpenEdit: (item: AgentPlatformItem) => void
  onRetry: () => void
  response?: AgentPlatformListResponse
  showCliType: boolean
  showLatestVersion: boolean
  title: string
  typeLabel: string
}

type ResourceFormState = {
  type: AgentPlatformResourceType
  cliType: AgentPlatformAgentCliType
  displayName: string
  description: string
  avatar: string
  avatarPreviewUrl: string
  mcpConfigJson: string
  skillFile: File | null
  externalKnowledgeId: string
  instructions: string
  categories: string[]
  recommendedPrompts: string
  mcpIds: string[]
  skillIds: string[]
  knowledgeIds: string[]
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

type AgentPlatformDetailItem = AgentPlatformItem &
  Partial<{
    config: AgentPlatformJsonValue
    file_name: string
    sha256: string
    size_bytes: number
    external_knowledge_id: string
    cli_type: AgentPlatformAgentCliType
    instructions: string
    categories: string[]
    recommended_prompts: string[]
    mcp_ids: string[]
    skill_ids: string[]
    knowledge_ids: string[]
  }>

type PublishFormState = {
  summary: string
  selectedUserIds: string[]
  selectedDepartmentIds: string[]
  selectedUserOptions: GrantSelectOption[]
  selectedDepartmentOptions: GrantSelectOption[]
}

type SummaryMetric = {
  key: string
  label: string
  value: string | number
  helper: string
  icon: ElementType
}

const RESOURCE_QUERY_KEYS = {
  mcp: ['agent-platform', 'mcps', 'summary'] as const,
  skill: ['agent-platform', 'skills', 'summary'] as const,
  knowledge: ['agent-platform', 'knowledge', 'summary'] as const,
  agent: ['agent-platform', 'agents', 'summary'] as const,
}

const MCP_CONFIG_JSON_PLACEHOLDER =
  '// Example JSON (stdio):\n' +
  '{\n' +
  '  "mcpServers": {\n' +
  '    "stdio-server-example": {\n' +
  '      "command": "npx",\n' +
  '      "args": ["-y", "mcp-server-example"]\n' +
  '    }\n' +
  '  }\n' +
  '}\n\n' +
  '// Example JSON (sse):\n' +
  '{\n' +
  '  "mcpServers": {\n' +
  '    "sse-server-example": {\n' +
  '      "type": "sse",\n' +
  '      "url": "http://localhost:3000"\n' +
  '    }\n' +
  '  }\n' +
  '}\n\n' +
  '// Example JSON (streamableHttp OAuth):\n' +
  '{\n' +
  '  "mcpServers": {\n' +
  '    "streamable-http-oauth-example": {\n' +
  '      "type": "streamableHttp",\n' +
  '      "url": "http://localhost:3002/mcp",\n' +
  '      "oauth": {\n' +
  '        "callback_port": 51786,\n' +
  '        "callback_url": "http://localhost:51786/callback",\n' +
  '        "client_id": "your-oauth-client-id"\n' +
  '      }\n' +
  '    }\n' +
  '  }\n' +
  '}\n\n' +
  '// Example JSON (streamableHttp):\n' +
  '{\n' +
  '  "mcpServers": {\n' +
  '    "streamable-http-example": {\n' +
  '      "type": "streamableHttp",\n' +
  '      "url": "http://localhost:3001",\n' +
  '      "headers": {\n' +
  '        "Content-Type": "application/json",\n' +
  '        "Authorization": "Bearer your-token"\n' +
  '      }\n' +
  '    }\n' +
  '  }\n' +
  '}'

const AGENT_INSTRUCTIONS_PLACEHOLDER = '你是一个有用的助手'
const AGENT_CATEGORY_VALUES = [
  'general',
  'amazon_operations',
  'dtc_operations',
  'marketing',
  'design',
  'customer_service',
  'logistics',
  'market',
  'finance',
  'hr',
  'administration',
] as const

type AgentCategoryValue = (typeof AGENT_CATEGORY_VALUES)[number]

function formatAgentCategoryLabel(
  value: AgentCategoryValue,
  t: ReturnType<typeof useTranslation>['t']
) {
  switch (value) {
    case 'general':
      return t('General Assistant', { defaultValue: '通用助手' })
    case 'amazon_operations':
      return t('Amazon Operations Assistant', {
        defaultValue: '亚马逊运营助手',
      })
    case 'dtc_operations':
      return t('DTC Operations Assistant', { defaultValue: 'DTC运营助手' })
    case 'marketing':
      return t('Marketing Assistant', { defaultValue: '营销助手' })
    case 'design':
      return t('Design Assistant', { defaultValue: '设计助手' })
    case 'customer_service':
      return t('Customer Service Assistant', { defaultValue: '客服助手' })
    case 'logistics':
      return t('Logistics Assistant', { defaultValue: '物流助手' })
    case 'market':
      return t('Market Assistant', { defaultValue: '市场助手' })
    case 'finance':
      return t('Finance Assistant', { defaultValue: '财务助手' })
    case 'hr':
      return t('HR Assistant', { defaultValue: '人事助手' })
    case 'administration':
      return t('Administration Assistant', { defaultValue: '行政助手' })
  }
}

function normalizeAgentCategories(values?: string[]) {
  const seen = new Set<string>()
  const categories: string[] = []
  for (const value of values ?? []) {
    const normalized = value.trim().toLowerCase()
    if (
      !normalized ||
      seen.has(normalized) ||
      !AGENT_CATEGORY_VALUES.includes(normalized as AgentCategoryValue)
    ) {
      continue
    }
    seen.add(normalized)
    categories.push(normalized)
  }
  return categories.length > 0 ? categories : ['general']
}

function normalizeRecommendedPrompts(value?: string | string[]) {
  const values = Array.isArray(value) ? value : (value?.split(/\r?\n/) ?? [])
  const seen = new Set<string>()
  const prompts: string[] = []
  for (const item of values) {
    const prompt = item.trim()
    if (!prompt || seen.has(prompt)) {
      continue
    }
    seen.add(prompt)
    prompts.push(prompt)
  }
  return prompts
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

function formatAgentCliTypeLabel(value?: string) {
  const normalized = value?.trim().toLowerCase()
  switch (normalized) {
    case 'opencode':
      return 'OpenCode'
    case 'codex':
      return 'Codex'
    default:
      return value?.trim() || ''
  }
}

function formatDateTimeText(value?: string) {
  if (!value) {
    return '-'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}

function isImageAvatar(value?: string) {
  const avatar = value?.trim() ?? ''
  return (
    avatar.startsWith('/api/') ||
    avatar.startsWith('http://') ||
    avatar.startsWith('https://') ||
    avatar.startsWith('data:image/')
  )
}

function AvatarPreview(props: { value?: string }) {
  if (!props.value) {
    return null
  }
  if (isImageAvatar(props.value)) {
    return (
      <img
        alt=''
        className='size-6 rounded-full object-cover'
        src={props.value}
      />
    )
  }
  return <span>{props.value}</span>
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

function getItems<TItem extends AgentPlatformItem>(
  response?: AgentPlatformListResponse<TItem>
) {
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

function userGrantOption(user: User): GrantSelectOption {
  const displayName = user.display_name?.trim()
  const email = user.email?.trim()
  const username = user.username.trim()
  const label = displayName || username || `#${user.id}`
  const descriptionParts = [`#${user.id}`]
  if (username && username !== label) {
    descriptionParts.push(username)
  }
  if (email) {
    descriptionParts.push(email)
  }
  return {
    value: String(user.id),
    label,
    description: descriptionParts.join(' · '),
  }
}

function grantOptionFromGrant(grant: AgentPlatformGrantRequest) {
  const subjectName = grant.subject_name?.trim()
  const label = subjectName || `#${grant.subject_id}`
  return {
    value: grant.subject_id,
    label,
    description: subjectName ? `#${grant.subject_id}` : label,
  }
}

function getDepartmentGrantOptions(nodes: DepartmentTreeNode[]) {
  const options: GrantSelectOption[] = []

  const visit = (items: DepartmentTreeNode[]) => {
    for (const item of items) {
      const label = item.name || `#${item.id}`
      options.push({
        value: String(item.id),
        label,
        description: `#${item.id}`,
      })
      visit(item.children ?? [])
    }
  }

  visit(nodes)
  return options
}

function resourceTypeIcon(type: AgentPlatformResourceType) {
  switch (type) {
    case 'mcp':
      return Unplug
    case 'skill':
      return Puzzle
    case 'knowledge':
      return BookOpen
    case 'agent':
      return Bot
  }
}

function resourceTypeLabel(
  type: AgentPlatformResourceType,
  t: (key: string) => string
) {
  switch (type) {
    case 'mcp':
      return t('MCP')
    case 'skill':
      return t('Skill')
    case 'knowledge':
      return t('Knowledge')
    case 'agent':
      return t('Agent')
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

function parseJsonField(value: string): AgentPlatformJsonValue {
  const trimmed = value.trim()
  if (!trimmed) {
    throw new Error('agent_platform.validation.json_required')
  }
  try {
    return JSON.parse(trimmed) as AgentPlatformJsonValue
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
    case 'agent_platform.validation.json_required':
      return t('JSON configuration is required')
    case 'agent_platform.validation.json':
      return t('JSON fields must contain valid JSON')
    case 'agent_platform.validation.zip_required':
      return t('Upload a zip package')
    case 'agent_platform.validation.grant_required':
      return t('Add at least one user or department grant')
    case 'agent_platform.validation.subject_required':
      return t('Grant subject ID is required')
    case 'agent_platform.validation.current_user_required':
      return t('Current user is required')
    default:
      return message
  }
}

function defaultResourceFormState(
  type: AgentPlatformResourceType,
  item?: AgentPlatformItem
): ResourceFormState {
  const detail = item as AgentPlatformDetailItem | undefined
  const agent = item as AgentPlatformAgentItem | undefined
  const mcpConfig =
    detail?.resource_type === 'mcp' ? formatJsonPreview(detail.config) : ''

  return {
    type,
    cliType: agent?.cli_type ?? 'opencode',
    displayName: item?.display_name ?? '',
    description: item?.description ?? '',
    avatar: item?.avatar ?? '',
    avatarPreviewUrl: item?.avatar_url ?? item?.avatar ?? '',
    mcpConfigJson: mcpConfig,
    skillFile: null,
    externalKnowledgeId: detail?.external_knowledge_id ?? '',
    instructions: agent?.instructions ?? '',
    categories: normalizeAgentCategories(agent?.categories),
    recommendedPrompts: normalizeRecommendedPrompts(
      agent?.recommended_prompts
    ).join('\n'),
    mcpIds: agent?.mcp_ids ?? [],
    skillIds: agent?.skill_ids ?? [],
    knowledgeIds: agent?.knowledge_ids ?? [],
  }
}

function buildCreateOrUpdateBase(form: ResourceFormState) {
  return {
    display_name: form.displayName.trim(),
    description: form.description.trim(),
  }
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
        {props.isLoading && <ResourceTableSkeleton />}
        {!props.isLoading && loadFailed && (
          <ErrorState
            className='min-h-[240px]'
            title={t('Failed to load data')}
            description={`${props.errorPrefix}: ${errorMessage}`}
            onRetry={props.onRetry}
          />
        )}
        {!props.isLoading && !loadFailed && items.length === 0 && (
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
        )}
        {!props.isLoading && !loadFailed && items.length > 0 && (
          <ResourceTable
            items={items}
            onOpenDetails={props.onOpenDetails}
            onOpenEdit={props.onOpenEdit}
            showCliType={props.showCliType}
            showLatestVersion={props.showLatestVersion}
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
  showCliType: boolean
  showLatestVersion: boolean
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
            {props.showCliType ? (
              <TableHead>{t('Agent CLI 类型')}</TableHead>
            ) : null}
            {props.showLatestVersion ? (
              <TableHead>{t('Status')}</TableHead>
            ) : null}
            <TableHead>{t('Owner')}</TableHead>
            {props.showLatestVersion ? (
              <TableHead>{t('Latest version')}</TableHead>
            ) : null}
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
              showCliType={props.showCliType}
              showLatestVersion={props.showLatestVersion}
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
  showCliType: boolean
  showLatestVersion: boolean
  typeLabel: string
}) {
  const { t } = useTranslation()
  const item = props.item
  const agentItem = item as AgentPlatformAgentItem

  return (
    <TableRow>
      <TableCell>
        <div className='min-w-[220px] space-y-1'>
          <div className='flex items-center gap-2 font-medium'>
            <AvatarPreview value={item.avatar_url || item.avatar} />
            {item.display_name}
          </div>
          <div className='text-muted-foreground text-xs'>
            {item.resource_id}
          </div>
          {item.description ? (
            <div className='text-muted-foreground max-w-[360px] truncate text-xs'>
              {item.description}
            </div>
          ) : null}
        </div>
      </TableCell>
      <TableCell>
        <Badge variant='outline'>{props.typeLabel}</Badge>
      </TableCell>
      {props.showCliType ? (
        <TableCell>
          <Badge variant='secondary'>
            {formatAgentCliTypeLabel(agentItem.cli_type) || t('Not configured')}
          </Badge>
        </TableCell>
      ) : null}
      {props.showLatestVersion ? (
        <TableCell>
          <StatusBadge
            label={formatStatusLabel(item.status, t)}
            variant={statusVariantFor(item.status)}
            copyable={false}
          />
        </TableCell>
      ) : null}
      <TableCell>{item.owner_name || `#${item.owner_user_id}`}</TableCell>
      {props.showLatestVersion ? (
        <TableCell>{item.latest_version || t('Not versioned')}</TableCell>
      ) : null}
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

function DetailField(props: { label: string; value: ReactNode }) {
  return (
    <div className='space-y-1'>
      <div className='text-muted-foreground text-xs font-medium'>
        {props.label}
      </div>
      <div className='text-sm break-words whitespace-pre-wrap'>
        {props.value}
      </div>
    </div>
  )
}

function JsonPreviewBlock(props: { label: string; value: unknown }) {
  const text = formatJsonPreview(props.value)
  if (!text) {
    return null
  }
  return (
    <div className='space-y-2'>
      <div className='text-muted-foreground text-xs font-medium'>
        {props.label}
      </div>
      <pre className='bg-muted/40 max-h-72 overflow-auto rounded-lg border p-3 text-xs'>
        {text}
      </pre>
    </div>
  )
}

function ResourceCheckboxList(props: {
  description?: string
  emptyLabel: string
  items: AgentPlatformItem[]
  label: string
  onChange: (ids: string[]) => void
  selectedIds: string[]
}) {
  return (
    <Field>
      <FieldLabel>{props.label}</FieldLabel>
      {props.description ? (
        <FieldDescription>{props.description}</FieldDescription>
      ) : null}
      {props.items.length === 0 ? (
        <div className='text-muted-foreground rounded-md border px-3 py-2 text-sm'>
          {props.emptyLabel}
        </div>
      ) : (
        <div className='grid max-h-48 gap-2 overflow-y-auto rounded-md border p-2 sm:grid-cols-2'>
          {props.items.map((item) => {
            const checked = props.selectedIds.includes(item.resource_id)
            return (
              <label
                key={item.resource_id}
                className='hover:bg-muted/40 flex cursor-pointer items-start gap-2 rounded-md px-2 py-2'
              >
                <Checkbox
                  checked={checked}
                  onCheckedChange={(value) => {
                    const enabled = value === true
                    if (enabled) {
                      props.onChange([...props.selectedIds, item.resource_id])
                      return
                    }
                    props.onChange(
                      props.selectedIds.filter((id) => id !== item.resource_id)
                    )
                  }}
                />
                <span className='min-w-0 space-y-0.5'>
                  <span className='block truncate text-sm font-medium'>
                    {item.display_name}
                  </span>
                  <span className='text-muted-foreground block truncate text-xs'>
                    {item.resource_id}
                  </span>
                </span>
              </label>
            )
          })}
        </div>
      )}
    </Field>
  )
}

function ResourceEditorDialog(props: {
  form: ResourceFormState
  knowledgeItems: AgentPlatformItem[]
  mcpItems: AgentPlatformItem[]
  mode: 'create' | 'edit'
  onChange: (form: ResourceFormState) => void
  onClose: () => void
  onSubmit: () => void
  open: boolean
  pending: boolean
  skillItems: AgentPlatformItem[]
}) {
  const { t } = useTranslation()
  const Icon = resourceTypeIcon(props.form.type)
  const isEdit = props.mode === 'edit'
  const [avatarUploading, setAvatarUploading] = useState(false)
  const update = <TKey extends keyof ResourceFormState>(
    key: TKey,
    value: ResourceFormState[TKey]
  ) => props.onChange({ ...props.form, [key]: value })
  const title = isEdit
    ? t('Edit {{type}} resource', {
        type: resourceTypeLabel(props.form.type, t),
      })
    : t('Create {{type}} resource', {
        type: resourceTypeLabel(props.form.type, t),
      })
  const handleAvatarFileChange = (file?: File) => {
    if (!file) {
      return
    }
    setAvatarUploading(true)
    void uploadAgentPlatformAvatar(file)
      .then((response) => {
        if (!response.success || !response.data?.url) {
          toast.error(response.message || t('Request failed'))
          return
        }
        props.onChange({
          ...props.form,
          avatar: response.data.url,
          avatarPreviewUrl: response.data.preview_url || response.data.url,
        })
        toast.success(t('Avatar uploaded'))
      })
      .catch((error: unknown) => {
        toast.error(getErrorMessage(error, t('Request failed')))
      })
      .finally(() => setAvatarUploading(false))
  }

  return (
    <Dialog open={props.open} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle className='flex items-center gap-2'>
            <span className='bg-primary/10 text-primary inline-flex size-8 items-center justify-center rounded-lg'>
              <Icon className='size-4' />
            </span>
            {title}
          </DialogTitle>
          <DialogDescription>
            {props.form.type === 'agent'
              ? t('Manage the Agent definition and dependency selection.')
              : t('Manage the platform resource metadata and configuration.')}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <Field>
            <FieldLabel>{t('Display name')}</FieldLabel>
            <Input
              value={props.form.displayName}
              onChange={(event) => update('displayName', event.target.value)}
              placeholder={t('Resource display name')}
            />
          </Field>

          <Field>
            <FieldLabel>{t('Description')}</FieldLabel>
            <Textarea
              value={props.form.description}
              onChange={(event) => update('description', event.target.value)}
              placeholder={t('Describe the resource purpose')}
              rows={3}
            />
          </Field>

          {props.form.type === 'mcp' ? (
            <Field>
              <FieldLabel>{t('MCP JSON configuration')}</FieldLabel>
              <Textarea
                value={props.form.mcpConfigJson}
                onChange={(event) =>
                  update('mcpConfigJson', event.target.value)
                }
                rows={12}
                className='font-mono'
                placeholder={MCP_CONFIG_JSON_PLACEHOLDER}
              />
            </Field>
          ) : null}

          {props.form.type === 'skill' ? (
            <Field>
              <FieldLabel>{t('Skill zip package')}</FieldLabel>
              <Input
                type='file'
                accept='.zip,application/zip'
                onChange={(event) =>
                  update('skillFile', event.target.files?.[0] ?? null)
                }
              />
              <FieldDescription>
                {isEdit
                  ? t('Uploading a file replaces the existing Skill package.')
                  : t('Upload the Skill package stored by the platform.')}
              </FieldDescription>
            </Field>
          ) : null}

          {props.form.type === 'knowledge' ? (
            <Field>
              <FieldLabel>{t('Knowledge base ID')}</FieldLabel>
              <Input
                value={props.form.externalKnowledgeId}
                onChange={(event) =>
                  update('externalKnowledgeId', event.target.value)
                }
                placeholder={t('External knowledge base ID')}
              />
            </Field>
          ) : null}

          {props.form.type === 'agent' ? (
            <>
              <Field>
                <FieldLabel>{t('Agent CLI 类型')}</FieldLabel>
                <NativeSelect
                  className='w-full'
                  value={props.form.cliType}
                  onChange={(event) =>
                    update(
                      'cliType',
                      event.target.value as AgentPlatformAgentCliType
                    )
                  }
                >
                  <NativeSelectOption value='opencode'>
                    OpenCode
                  </NativeSelectOption>
                  <NativeSelectOption value='codex'>Codex</NativeSelectOption>
                </NativeSelect>
                <FieldDescription>
                  {props.form.cliType === 'codex'
                    ? t('发布后将生成 codex.zip。')
                    : t('发布后将生成 opencode.zip。')}
                </FieldDescription>
              </Field>
              <Field>
                <FieldLabel>{t('Agent 分类')}</FieldLabel>
                <FieldDescription>
                  {t('Select at least one category for this agent.')}
                </FieldDescription>
                <div className='grid gap-2 sm:grid-cols-2'>
                  {AGENT_CATEGORY_VALUES.map((category) => {
                    const checked = props.form.categories.includes(category)
                    return (
                      <label
                        key={category}
                        className='hover:bg-muted/40 flex cursor-pointer items-start gap-2 rounded-md border px-3 py-2'
                      >
                        <Checkbox
                          checked={checked}
                          onCheckedChange={(value) => {
                            const enabled = value === true
                            if (enabled) {
                              update(
                                'categories',
                                normalizeAgentCategories([
                                  ...props.form.categories,
                                  category,
                                ])
                              )
                              return
                            }
                            update(
                              'categories',
                              props.form.categories.filter(
                                (item) => item !== category
                              )
                            )
                          }}
                        />
                        <span className='min-w-0 flex-1'>
                          <span className='block truncate font-medium'>
                            {formatAgentCategoryLabel(category, t)}
                          </span>
                        </span>
                      </label>
                    )
                  })}
                </div>
              </Field>
              <Field>
                <FieldLabel>{t('Recommended prompts')}</FieldLabel>
                <Textarea
                  value={props.form.recommendedPrompts}
                  onChange={(event) =>
                    update('recommendedPrompts', event.target.value)
                  }
                  rows={4}
                  placeholder={t('Enter one suggested question per line.')}
                />
                <FieldDescription>
                  {t('Enter one suggested question per line.')}
                </FieldDescription>
              </Field>
              <Field>
                <FieldLabel>{t('Avatar')}</FieldLabel>
                <div className='flex items-center gap-2'>
                  <Button
                    variant='outline'
                    disabled={avatarUploading}
                    render={
                      <label className='inline-flex cursor-pointer items-center gap-2'>
                        <Image className='size-4' />
                        {avatarUploading ? t('Uploading') : t('Upload image')}
                        <input
                          type='file'
                          accept='image/png,image/jpeg,image/gif,image/webp'
                          className='hidden'
                          onChange={(event) =>
                            handleAvatarFileChange(event.target.files?.[0])
                          }
                        />
                      </label>
                    }
                  />
                  <AvatarPreview
                    value={props.form.avatarPreviewUrl || props.form.avatar}
                  />
                </div>
              </Field>
              <Field>
                <FieldLabel>{t('Instructions')}</FieldLabel>
                <Textarea
                  value={props.form.instructions}
                  onChange={(event) =>
                    update('instructions', event.target.value)
                  }
                  rows={8}
                  placeholder={AGENT_INSTRUCTIONS_PLACEHOLDER}
                />
              </Field>
              <ResourceCheckboxList
                label={t('MCP dependencies')}
                emptyLabel={t('No MCP resources are available.')}
                items={props.mcpItems}
                selectedIds={props.form.mcpIds}
                onChange={(ids) => update('mcpIds', ids)}
              />
              <ResourceCheckboxList
                label={t('Skill dependencies')}
                emptyLabel={t('No Skill resources are available.')}
                items={props.skillItems}
                selectedIds={props.form.skillIds}
                onChange={(ids) => update('skillIds', ids)}
              />
              <ResourceCheckboxList
                label={t('Knowledge dependencies')}
                emptyLabel={t('No Knowledge resources are available.')}
                items={props.knowledgeItems}
                selectedIds={props.form.knowledgeIds}
                onChange={(ids) => update('knowledgeIds', ids)}
              />
            </>
          ) : null}
        </FieldGroup>

        <DialogFooter>
          <Button variant='outline' onClick={props.onClose}>
            {t('Cancel')}
          </Button>
          <Button onClick={props.onSubmit} disabled={props.pending}>
            <Save className='size-4' />
            {isEdit ? t('Save changes') : t('Create resource')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function ResourceDetailSheet(props: {
  detailResponse?: {
    success: boolean
    message?: string
    data?: AgentPlatformDetailItem
  }
  detailLoading: boolean
  operationPending: boolean
  onClose: () => void
  onDelete: (item: AgentPlatformItem) => void
  onOpenEdit: (item: AgentPlatformItem) => void
  onOpenPublish: (item: AgentPlatformItem) => void
  onRefreshDetail: () => void
  onSetEnabled: (item: AgentPlatformItem, enabled: boolean) => void
  open: boolean
  selected: AgentPlatformItem | null
}) {
  const { t } = useTranslation()
  const item = (props.detailResponse?.data ??
    props.selected) as AgentPlatformDetailItem | null
  const Icon = item ? resourceTypeIcon(item.resource_type) : Boxes
  const isDisabled = item?.status.trim().toLowerCase() === 'disabled'
  const detailFailed =
    props.detailResponse != null && props.detailResponse.success === false
  const [selectedVersion, setSelectedVersion] = useState('')
  const versionsQuery = useQuery({
    queryKey: ['agent-platform', item?.resource_id, 'versions'],
    queryFn: async () => {
      if (!item) {
        throw new Error('No resource selected')
      }
      return getAgentPlatformAgentVersions(item.resource_id)
    },
    enabled: props.open && item?.resource_type === 'agent',
  })
  const versionItems = useMemo(
    () => versionsQuery.data?.data?.items ?? [],
    [versionsQuery.data?.data?.items]
  )
  useEffect(() => {
    if (item?.resource_type !== 'agent') {
      setSelectedVersion('')
      return
    }
    const nextVersion = versionItems[0]?.version ?? item.latest_version ?? ''
    const selectedExists = versionItems.some(
      (version) => version.version === selectedVersion
    )
    if (nextVersion && (!selectedVersion || !selectedExists)) {
      setSelectedVersion(nextVersion)
    }
  }, [item?.latest_version, item?.resource_type, selectedVersion, versionItems])
  const grantsQuery = useQuery({
    queryKey: ['agent-platform', item?.resource_id, selectedVersion, 'grants'],
    queryFn: async () => {
      if (!item || !selectedVersion) {
        throw new Error('No version selected')
      }
      return getAgentPlatformAgentVersionGrants(
        item.resource_id,
        selectedVersion
      )
    },
    enabled:
      props.open && item?.resource_type === 'agent' && selectedVersion !== '',
  })
  const grantItems = grantsQuery.data?.data?.items ?? []

  return (
    <Sheet open={props.open} onOpenChange={(open) => !open && props.onClose()}>
      <SheetContent
        side='right'
        className={sideDrawerContentClassName('overflow-y-auto sm:max-w-xl')}
      >
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle className='flex items-center gap-2'>
            <span className='bg-primary/10 text-primary inline-flex size-8 items-center justify-center rounded-lg'>
              <Icon className='size-4' />
            </span>
            {item?.display_name ?? t('Resource details')}
          </SheetTitle>
          <SheetDescription>
            {item
              ? `${resourceTypeLabel(item.resource_type, t)} - ${item.resource_id}`
              : t('Inspect resource details and lifecycle state.')}
          </SheetDescription>
        </SheetHeader>
        <div className='space-y-5 p-4'>
          {props.detailLoading && <ResourceTableSkeleton />}
          {!props.detailLoading && detailFailed && (
            <ErrorState
              className='min-h-[220px]'
              title={t('Failed to load data')}
              description={props.detailResponse?.message ?? t('Request failed')}
              onRetry={props.onRefreshDetail}
            />
          )}
          {!props.detailLoading && !detailFailed && item && (
            <>
              <div className='flex flex-wrap gap-2'>
                <Button
                  size='sm'
                  variant='outline'
                  onClick={() => props.onOpenEdit(item)}
                >
                  <SquarePen className='size-4' />
                  {t('Edit')}
                </Button>
                {item.resource_type !== 'agent' ? (
                  <Button
                    size='sm'
                    variant='outline'
                    disabled={props.operationPending}
                    onClick={() => props.onSetEnabled(item, isDisabled)}
                  >
                    {isDisabled ? (
                      <CheckCircle2 className='size-4' />
                    ) : (
                      <XCircle className='size-4' />
                    )}
                    {isDisabled ? t('Enable') : t('Disable')}
                  </Button>
                ) : null}
                {item.resource_type === 'agent' ? (
                  <Button
                    size='sm'
                    disabled={props.operationPending}
                    onClick={() => props.onOpenPublish(item)}
                  >
                    <Rocket className='size-4' />
                    {t('Publish')}
                  </Button>
                ) : null}
                {item.resource_type === 'skill' ? (
                  <Button
                    size='sm'
                    variant='outline'
                    render={
                      <a
                        href={getAgentPlatformSkillPackageDownloadUrl(
                          item.resource_id
                        )}
                      >
                        <Download className='size-4' />
                        {t('Download package')}
                      </a>
                    }
                  />
                ) : null}
                <Button
                  size='sm'
                  variant='destructive'
                  disabled={props.operationPending}
                  onClick={() => props.onDelete(item)}
                >
                  <Trash2 className='size-4' />
                  {t('Delete')}
                </Button>
              </div>

              <div className='grid gap-4 rounded-lg border p-4 sm:grid-cols-2'>
                {item.resource_type === 'agent' ? (
                  <DetailField label={t('Status')} value={item.status} />
                ) : null}
                {item.resource_type === 'agent' ? (
                  <DetailField
                    label={t('Latest version')}
                    value={item.latest_version || t('Not versioned')}
                  />
                ) : null}
                <DetailField
                  label={t('Owner')}
                  value={item.owner_name || `#${item.owner_user_id}`}
                />
                <DetailField
                  label={t('Tenant')}
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

              {item.description ? (
                <DetailField
                  label={t('Description')}
                  value={item.description}
                />
              ) : null}

              {item.resource_type === 'mcp' ? (
                <JsonPreviewBlock
                  label={t('MCP JSON configuration')}
                  value={item.config}
                />
              ) : null}

              {item.resource_type === 'skill' && item.file_name ? (
                <div className='grid gap-4 rounded-lg border p-4 sm:grid-cols-2'>
                  <DetailField
                    label={t('Package file')}
                    value={item.file_name}
                  />
                  <DetailField
                    label={t('Package SHA256')}
                    value={item.sha256}
                  />
                  <DetailField
                    label={t('Package size')}
                    value={item.size_bytes ?? 0}
                  />
                </div>
              ) : null}

              {item.resource_type === 'knowledge' ? (
                <DetailField
                  label={t('Knowledge base ID')}
                  value={item.external_knowledge_id || t('Not configured')}
                />
              ) : null}

              {item.resource_type === 'agent' ? (
                <>
                  <DetailField
                    label={t('Agent CLI 类型')}
                    value={item.cli_type || t('未配置')}
                  />
                  {item.categories?.length ? (
                    <DetailField
                      label={t('Agent 分类')}
                      value={
                        <div className='flex flex-wrap gap-2'>
                          {item.categories.map((category) => (
                            <Badge key={category} variant='outline'>
                              {formatAgentCategoryLabel(
                                category as AgentCategoryValue,
                                t
                              )}
                            </Badge>
                          ))}
                        </div>
                      }
                    />
                  ) : null}
                  {item.recommended_prompts?.length ? (
                    <DetailField
                      label={t('Recommended prompts')}
                      value={
                        <div className='space-y-1'>
                          {item.recommended_prompts.map((prompt) => (
                            <div key={prompt} className='whitespace-pre-wrap'>
                              {prompt}
                            </div>
                          ))}
                        </div>
                      }
                    />
                  ) : null}
                  <DetailField
                    label={t('Instructions')}
                    value={
                      <pre className='bg-muted/40 max-h-72 overflow-auto rounded-lg border p-3 text-xs whitespace-pre-wrap'>
                        {item.instructions || t('Not configured')}
                      </pre>
                    }
                  />
                  <JsonPreviewBlock
                    label={t('Dependencies')}
                    value={{
                      mcp_ids: item.mcp_ids ?? [],
                      skill_ids: item.skill_ids ?? [],
                      knowledge_ids: item.knowledge_ids ?? [],
                    }}
                  />
                  <AgentVersionPanel
                    grants={grantItems}
                    grantsLoading={grantsQuery.isFetching}
                    onVersionChange={setSelectedVersion}
                    selectedVersion={selectedVersion}
                    versions={versionItems}
                    versionsLoading={versionsQuery.isFetching}
                  />
                </>
              ) : null}
            </>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

function AgentVersionPanel(props: {
  grants: AgentPlatformGrantRequest[]
  grantsLoading: boolean
  onVersionChange: (version: string) => void
  selectedVersion: string
  versions: AgentPlatformAgentVersion[]
  versionsLoading: boolean
}) {
  const { t } = useTranslation()
  const visibleVersions = props.versions.slice(0, 3)
  let versionContent: ReactNode
  if (props.versionsLoading) {
    versionContent = <Skeleton className='h-24 w-full' />
  } else if (visibleVersions.length === 0) {
    versionContent = (
      <div className='text-muted-foreground text-sm'>
        {t('No published versions')}
      </div>
    )
  } else {
    versionContent = (
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Version')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Summary')}</TableHead>
            <TableHead>{t('Published At')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {visibleVersions.map((version) => (
            <TableRow key={version.version}>
              <TableCell className='font-medium'>{version.version}</TableCell>
              <TableCell>
                <StatusBadge
                  label={formatStatusLabel(version.status, t)}
                  variant={statusVariantFor(version.status)}
                />
              </TableCell>
              <TableCell>{version.summary || '-'}</TableCell>
              <TableCell>
                {formatDateTimeText(version.published_at ?? version.created_at)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    )
  }

  let grantsContent: ReactNode
  if (props.grantsLoading) {
    grantsContent = <Skeleton className='h-14 w-full' />
  } else if (props.grants.length === 0) {
    grantsContent = (
      <div className='text-muted-foreground text-sm'>
        {t('No grants configured')}
      </div>
    )
  } else {
    grantsContent = (
      <div className='flex flex-wrap gap-2'>
        {props.grants.map((grant) => (
          <Badge
            key={`${grant.subject_type}:${grant.subject_id}`}
            variant='secondary'
          >
            {grant.subject_type === 'user' ? t('User') : t('Department')}{' '}
            {grant.subject_name || `#${grant.subject_id}`}
          </Badge>
        ))}
      </div>
    )
  }

  return (
    <div className='space-y-3 rounded-lg border p-4'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div>
          <div className='font-medium'>{t('Published versions')}</div>
          <div className='text-muted-foreground text-sm'>
            {t('View release records and active grants.')}
          </div>
        </div>
        <NativeSelect
          className='w-40'
          value={props.selectedVersion}
          onChange={(event) => props.onVersionChange(event.target.value)}
        >
          {visibleVersions.map((version) => (
            <NativeSelectOption key={version.version} value={version.version}>
              {version.version}
            </NativeSelectOption>
          ))}
        </NativeSelect>
      </div>

      {versionContent}

      <div className='space-y-2'>
        <div className='font-medium'>{t('Version grants')}</div>
        {grantsContent}
      </div>
    </div>
  )
}

function PublishAgentDialog(props: {
  form: PublishFormState
  onChange: (form: PublishFormState) => void
  onClose: () => void
  onSubmit: () => void
  open: boolean
  pending: boolean
  resource: AgentPlatformItem | null
}) {
  const { t } = useTranslation()
  const [userSearchValue, setUserSearchValue] = useState('')
  const [departmentSearchValue, setDepartmentSearchValue] = useState('')
  const usersQuery = useQuery({
    queryKey: ['agent-platform', 'grant-users', userSearchValue],
    queryFn: async () => {
      const result = await searchUsers({
        keyword: userSearchValue,
        status: '1',
        p: 1,
        page_size: 20,
      })
      if (!result.success) {
        throw new Error(result.message || t('Request failed'))
      }
      return result.data?.items ?? []
    },
    enabled: props.open,
  })
  const departmentsQuery = useQuery({
    queryKey: ['agent-platform', 'grant-departments'],
    queryFn: async () => {
      const result = await getDepartmentTree()
      if (!result.success) {
        throw new Error(result.message || t('Request failed'))
      }
      return result.data ?? []
    },
    enabled: props.open,
  })
  const userOptions = useMemo(
    () => (usersQuery.data ?? []).map(userGrantOption),
    [usersQuery.data]
  )
  const departmentOptions = useMemo(
    () => getDepartmentGrantOptions(departmentsQuery.data ?? []),
    [departmentsQuery.data]
  )
  const mergeSelectedOptions = (
    ids: string[],
    options: GrantSelectOption[],
    existingOptions: GrantSelectOption[]
  ) =>
    ids
      .map(
        (id) =>
          options.find((option) => option.value === id) ??
          existingOptions.find((option) => option.value === id)
      )
      .filter((option): option is GrantSelectOption => option != null)
  return (
    <Dialog open={props.open} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle className='flex items-center gap-2'>
            <span className='bg-primary/10 text-primary inline-flex size-8 items-center justify-center rounded-lg'>
              <Rocket className='size-4' />
            </span>
            {t('Publish Agent')}
          </DialogTitle>
          <DialogDescription>
            {props.resource
              ? t('Publish {{name}} and grant access by user or department.', {
                  name: props.resource.display_name,
                })
              : t('Publish the Agent and grant access.')}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <Field>
            <FieldLabel>{t('Release summary')}</FieldLabel>
            <Textarea
              value={props.form.summary}
              onChange={(event) =>
                props.onChange({ ...props.form, summary: event.target.value })
              }
              placeholder={t('Describe this Agent release')}
              rows={3}
            />
          </Field>

          <Field>
            <FieldLabel>{t('Access grants')}</FieldLabel>
            <FieldDescription>
              {t('Grant this published version to users and departments.')}
            </FieldDescription>
            <div className='grid gap-4'>
              <div className='grid gap-2'>
                <FieldLabel>{t('Users')}</FieldLabel>
                <GrantSubjectMultiSelect
                  emptyLabel={t('No users found')}
                  loading={usersQuery.isFetching}
                  onSearchChange={setUserSearchValue}
                  onSelectedIdsChange={(ids) => {
                    props.onChange({
                      ...props.form,
                      selectedUserIds: ids,
                      selectedUserOptions: mergeSelectedOptions(
                        ids,
                        userOptions,
                        props.form.selectedUserOptions
                      ),
                    })
                  }}
                  options={userOptions}
                  placeholder={t('Select users')}
                  searchPlaceholder={t('Search users by name or email')}
                  searchValue={userSearchValue}
                  selectedIds={props.form.selectedUserIds}
                  selectedOptions={props.form.selectedUserOptions}
                />
              </div>
              <div className='grid gap-2'>
                <FieldLabel>{t('Departments')}</FieldLabel>
                <GrantSubjectMultiSelect
                  emptyLabel={t('No departments found')}
                  loading={departmentsQuery.isFetching}
                  onSearchChange={setDepartmentSearchValue}
                  onSelectedIdsChange={(ids) =>
                    props.onChange({
                      ...props.form,
                      selectedDepartmentIds: ids,
                      selectedDepartmentOptions: mergeSelectedOptions(
                        ids,
                        departmentOptions,
                        props.form.selectedDepartmentOptions
                      ),
                    })
                  }
                  options={departmentOptions}
                  placeholder={t('Select departments')}
                  searchPlaceholder={t('Search departments')}
                  searchValue={departmentSearchValue}
                  selectedIds={props.form.selectedDepartmentIds}
                  selectedOptions={props.form.selectedDepartmentOptions}
                  renderOptions={({ selectedIds, toggleValue }) => (
                    <DepartmentGrantTreeOptions
                      emptyLabel={t('No departments found')}
                      keyword={departmentSearchValue}
                      nodes={departmentsQuery.data ?? []}
                      selectedIds={selectedIds}
                      onToggleSelected={toggleValue}
                    />
                  )}
                />
              </div>
            </div>
          </Field>
        </FieldGroup>

        <DialogFooter>
          <Button variant='outline' onClick={props.onClose}>
            {t('Cancel')}
          </Button>
          <Button onClick={props.onSubmit} disabled={props.pending}>
            <Rocket className='size-4' />
            {t('Publish')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function AgentPlatformShell() {
  const { t } = useTranslation()
  const currentUser = useAuthStore((state) => state.auth.user)
  const userRole = currentUser?.role ?? ROLE.GUEST
  const isSuperAdmin = userRole >= ROLE.SUPER_ADMIN
  const queryClient = useQueryClient()
  const [editor, setEditor] = useState<ResourceEditorState | null>(null)
  const [resourceForm, setResourceForm] = useState<ResourceFormState>(
    defaultResourceFormState('mcp')
  )
  const [selectedResource, setSelectedResource] =
    useState<AgentPlatformItem | null>(null)
  const [publishResource, setPublishResource] =
    useState<AgentPlatformItem | null>(null)
  const [publishForm, setPublishForm] = useState<PublishFormState>({
    summary: '',
    selectedUserIds: [],
    selectedDepartmentIds: [],
    selectedUserOptions: [],
    selectedDepartmentOptions: [],
  })

  const mcpsQuery = useQuery({
    queryKey: RESOURCE_QUERY_KEYS.mcp,
    queryFn: getAgentPlatformMcps,
  })
  const skillsQuery = useQuery({
    queryKey: RESOURCE_QUERY_KEYS.skill,
    queryFn: getAgentPlatformSkills,
  })
  const knowledgeQuery = useQuery({
    queryKey: RESOURCE_QUERY_KEYS.knowledge,
    queryFn: getAgentPlatformKnowledge,
  })
  const agentsQuery = useQuery({
    queryKey: RESOURCE_QUERY_KEYS.agent,
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

  const invalidateResourceList = async (type: AgentPlatformResourceType) => {
    await queryClient.invalidateQueries({ queryKey: RESOURCE_QUERY_KEYS[type] })
  }

  const createMutation = useMutation<
    AgentPlatformItemResponse,
    Error,
    ResourceFormState
  >({
    mutationFn: (form: ResourceFormState) => {
      const base = buildCreateOrUpdateBase(form)
      switch (form.type) {
        case 'mcp':
          return createAgentPlatformMcp({
            ...base,
            config: parseJsonField(form.mcpConfigJson),
          })
        case 'skill':
          if (!form.skillFile) {
            throw new Error('agent_platform.validation.zip_required')
          }
          return createAgentPlatformSkill({
            ...base,
            file: form.skillFile,
          })
        case 'knowledge':
          return createAgentPlatformKnowledge({
            ...base,
            external_knowledge_id: form.externalKnowledgeId.trim(),
          })
        case 'agent':
          return createAgentPlatformAgent({
            ...base,
            cli_type: form.cliType,
            avatar: form.avatar.trim(),
            instructions: form.instructions,
            categories: form.categories,
            recommended_prompts: normalizeRecommendedPrompts(
              form.recommendedPrompts
            ),
            mcp_ids: form.mcpIds,
            skill_ids: form.skillIds,
            knowledge_ids: form.knowledgeIds,
          })
      }
    },
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

  const updateMutation = useMutation<
    AgentPlatformItemResponse,
    Error,
    {
      item: AgentPlatformItem
      form: ResourceFormState
    }
  >({
    mutationFn: (input: {
      item: AgentPlatformItem
      form: ResourceFormState
    }) => {
      const base = buildCreateOrUpdateBase(input.form)
      switch (input.item.resource_type) {
        case 'mcp':
          return updateAgentPlatformMcp(input.item.resource_id, {
            ...base,
            config: parseJsonField(input.form.mcpConfigJson),
          })
        case 'skill':
          return updateAgentPlatformSkill(input.item.resource_id, base)
        case 'knowledge':
          return updateAgentPlatformKnowledge(input.item.resource_id, {
            ...base,
            external_knowledge_id: input.form.externalKnowledgeId.trim(),
          })
        case 'agent':
          return updateAgentPlatformAgent(input.item.resource_id, {
            ...base,
            cli_type: input.form.cliType,
            avatar: input.form.avatar.trim(),
            instructions: input.form.instructions,
            categories: input.form.categories,
            recommended_prompts: normalizeRecommendedPrompts(
              input.form.recommendedPrompts
            ),
            mcp_ids: input.form.mcpIds,
            skill_ids: input.form.skillIds,
            knowledge_ids: input.form.knowledgeIds,
          })
      }
    },
    onSuccess: async (response, input) => {
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      if (input.item.resource_type === 'skill' && input.form.skillFile) {
        const uploadResponse = await uploadAgentPlatformSkillPackage(
          input.item.resource_id,
          input.form.skillFile
        )
        if (!uploadResponse.success) {
          toast.error(uploadResponse.message || t('Request failed'))
          return
        }
      }
      toast.success(t('Resource updated'))
      setEditor(null)
      await invalidateResourceList(input.item.resource_type)
      await detailQuery.refetch()
      if (response.data) {
        setSelectedResource(response.data)
      }
    },
    onError: (error) => {
      toast.error(translateFormError(error, t))
    },
  })

  const enableMutation = useMutation({
    mutationFn: (input: { item: AgentPlatformItem; enabled: boolean }) => {
      if (input.item.resource_type === 'agent') {
        throw new Error('Agent status is managed by publish')
      }
      return setAgentPlatformResourceEnabled(
        input.item.resource_type,
        input.item.resource_id,
        input.enabled
      )
    },
    onSuccess: async (response, input) => {
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      toast.success(
        input.enabled ? t('Resource enabled') : t('Resource disabled')
      )
      await invalidateResourceList(input.item.resource_type)
      await detailQuery.refetch()
    },
    onError: (error) => {
      toast.error(getErrorMessage(error, t('Request failed')))
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (item: AgentPlatformItem) =>
      deleteAgentPlatformResource(item.resource_type, item.resource_id),
    onSuccess: async (response, item) => {
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      toast.success(t('Resource deleted'))
      setSelectedResource(null)
      await invalidateResourceList(item.resource_type)
    },
    onError: (error) => {
      toast.error(getErrorMessage(error, t('Request failed')))
    },
  })

  const publishMutation = useMutation({
    mutationFn: (input: {
      item: AgentPlatformItem
      form: PublishFormState
    }) => {
      if (
        input.form.selectedUserIds.length === 0 &&
        input.form.selectedDepartmentIds.length === 0
      ) {
        throw new Error('agent_platform.validation.grant_required')
      }
      return publishAgentPlatformAgent(input.item.resource_id, {
        summary: input.form.summary.trim(),
        grants: {
          users: input.form.selectedUserIds,
          departments: input.form.selectedDepartmentIds,
        },
      })
    },
    onSuccess: async (response, input) => {
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      toast.success(t('Agent published'))
      setPublishResource(null)
      await invalidateResourceList('agent')
      setSelectedResource({
        ...input.item,
        status: response.data?.status ?? 'published',
        latest_version: response.data?.version ?? input.item.latest_version,
      })
    },
    onError: (error) => {
      toast.error(translateFormError(error, t))
    },
  })

  const openCreateEditor = (type: AgentPlatformResourceType) => {
    setEditor({ mode: 'create', type })
    setResourceForm(defaultResourceFormState(type))
  }

  const openEditEditor = (item: AgentPlatformItem) => {
    void (async () => {
      try {
        const response = await queryClient.fetchQuery({
          queryKey: [
            'agent-platform',
            item.resource_type,
            item.resource_id,
            'detail',
          ],
          queryFn: () =>
            getAgentPlatformResource(item.resource_type, item.resource_id),
        })
        const detailItem =
          response.success && response.data ? response.data : item
        setEditor({ mode: 'edit', type: item.resource_type, item: detailItem })
        setResourceForm(
          defaultResourceFormState(item.resource_type, detailItem)
        )
      } catch (error) {
        toast.error(getErrorMessage(error, t('Request failed')))
      }
    })()
  }

  const handleResourceSubmit = () => {
    if (!resourceForm.displayName.trim()) {
      toast.error(t('Display name is required'))
      return
    }
    if (resourceForm.type === 'agent' && resourceForm.categories.length === 0) {
      toast.error(t('Select at least one category'))
      return
    }
    try {
      if (editor?.mode === 'edit') {
        updateMutation.mutate({
          item: editor.item,
          form: resourceForm,
        })
      } else if (editor?.mode === 'create') {
        createMutation.mutate(resourceForm)
      }
    } catch (error) {
      toast.error(translateFormError(error, t))
    }
  }

  const openPublishDialog = (item: AgentPlatformItem) => {
    setPublishResource(item)
    setPublishForm({
      summary: '',
      selectedUserIds: [],
      selectedDepartmentIds: [],
      selectedUserOptions: [],
      selectedDepartmentOptions: [],
    })
    void (async () => {
      try {
        const response = await queryClient.fetchQuery({
          queryKey: ['agent-platform', item.resource_id, 'publish-defaults'],
          queryFn: () => getAgentPlatformPublishDefaults(item.resource_id),
        })
        const grants = response.success ? (response.data?.grants ?? []) : []
        setPublishForm((current) => ({
          ...current,
          selectedUserIds: grants
            .filter((grant) => grant.subject_type === 'user')
            .map((grant) => grant.subject_id),
          selectedDepartmentIds: grants
            .filter((grant) => grant.subject_type === 'department')
            .map((grant) => grant.subject_id),
          selectedUserOptions: grants
            .filter((grant) => grant.subject_type === 'user')
            .map(grantOptionFromGrant),
          selectedDepartmentOptions: grants
            .filter((grant) => grant.subject_type === 'department')
            .map(grantOptionFromGrant),
        }))
      } catch (error) {
        toast.error(getErrorMessage(error, t('Request failed')))
      }
    })()
  }

  const handlePublishSubmit = () => {
    if (!publishResource) {
      return
    }
    if (!publishForm.summary.trim()) {
      toast.error(t('Release summary is required'))
      return
    }
    try {
      publishMutation.mutate({ item: publishResource, form: publishForm })
    } catch (error) {
      toast.error(translateFormError(error, t))
    }
  }

  const handleRefreshAll = () => {
    void mcpsQuery.refetch()
    void skillsQuery.refetch()
    void knowledgeQuery.refetch()
    void agentsQuery.refetch()
  }

  const mcpItems = getItems(mcpsQuery.data)
  const skillItems = getItems(skillsQuery.data)
  const knowledgeItems = getItems(knowledgeQuery.data)
  const agentItems = getItems(agentsQuery.data)

  const metrics: SummaryMetric[] = [
    {
      key: 'mcps',
      label: t('MCP'),
      value: mcpItems.length,
      helper: t('Tool server definitions'),
      icon: Unplug,
    },
    {
      key: 'skills',
      label: t('Skills'),
      value: skillItems.length,
      helper: t('Local zip packages'),
      icon: Puzzle,
    },
    {
      key: 'knowledge',
      label: t('Knowledge'),
      value: knowledgeItems.length,
      helper: t('External knowledge IDs'),
      icon: BookOpen,
    },
    {
      key: 'agents',
      label: t('Published Agents'),
      value: countResourcesByStatus([agentItems], 'published'),
      helper: t('Granted to users or departments'),
      icon: Bot,
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
              onClick={handleRefreshAll}
              disabled={
                mcpsQuery.isFetching ||
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
                        'Manage MCP, Skill, Knowledge, and Agent resources for AionUi opencode configuration delivery.'
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

            <Tabs defaultValue='mcps' className='space-y-6'>
              <TabsList className='grid w-full grid-cols-4 md:w-[560px]'>
                <TabsTrigger value='mcps'>{t('MCP')}</TabsTrigger>
                <TabsTrigger value='skills'>{t('Skills')}</TabsTrigger>
                <TabsTrigger value='knowledge'>{t('Knowledge')}</TabsTrigger>
                <TabsTrigger value='agents'>{t('Agents')}</TabsTrigger>
              </TabsList>

              <TabsContent value='mcps'>
                <ResourceListCard
                  title={t('MCP management')}
                  description={t(
                    'Manage mcpServers JSON definitions used by Agent package generation.'
                  )}
                  badgeLabel={t('Tooling')}
                  createLabel={t('New MCP')}
                  icon={Unplug}
                  showCliType={false}
                  showLatestVersion={false}
                  typeLabel={resourceTypeLabel('mcp', t)}
                  response={mcpsQuery.data}
                  isError={mcpsQuery.isError}
                  error={mcpsQuery.error}
                  isLoading={mcpsQuery.isLoading}
                  onCreate={() => openCreateEditor('mcp')}
                  onOpenDetails={setSelectedResource}
                  onOpenEdit={openEditEditor}
                  onRetry={() => void mcpsQuery.refetch()}
                  errorPrefix={t('MCP API request failed')}
                  emptyDescription={t(
                    'No MCP resources are currently available.'
                  )}
                />
              </TabsContent>

              <TabsContent value='skills'>
                <ResourceListCard
                  title={t('Skill management')}
                  description={t('Upload and manage local Skill zip packages.')}
                  badgeLabel={t('Package')}
                  createLabel={t('New Skill')}
                  icon={Puzzle}
                  showCliType={false}
                  showLatestVersion={false}
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
                    'No Skill resources are currently available.'
                  )}
                />
              </TabsContent>

              <TabsContent value='knowledge'>
                <ResourceListCard
                  title={t('Knowledge management')}
                  description={t(
                    'Register external knowledge base IDs for Agent package generation.'
                  )}
                  badgeLabel={t('Knowledge ID')}
                  createLabel={t('New Knowledge')}
                  icon={BookOpen}
                  showCliType={false}
                  showLatestVersion={false}
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
                    'No Knowledge resources are currently available.'
                  )}
                />
              </TabsContent>

              <TabsContent value='agents'>
                <ResourceListCard
                  title={t('Agent definitions')}
                  description={t(
                    'Compose MCP, Skill, and Knowledge dependencies, then publish with user or department grants.'
                  )}
                  badgeLabel={t('Release managed')}
                  createLabel={t('New Agent')}
                  icon={Bot}
                  showCliType
                  showLatestVersion
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
                    'No Agent definitions are currently available.'
                  )}
                />
              </TabsContent>
            </Tabs>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>
      <ResourceEditorDialog
        form={resourceForm}
        knowledgeItems={knowledgeItems}
        mcpItems={mcpItems}
        mode={editor?.mode ?? 'create'}
        onChange={setResourceForm}
        onClose={() => setEditor(null)}
        onSubmit={handleResourceSubmit}
        open={editor != null}
        pending={createMutation.isPending || updateMutation.isPending}
        skillItems={skillItems}
      />
      <ResourceDetailSheet
        detailResponse={detailQuery.data}
        detailLoading={detailQuery.isFetching}
        operationPending={
          enableMutation.isPending ||
          deleteMutation.isPending ||
          publishMutation.isPending
        }
        onClose={() => setSelectedResource(null)}
        onDelete={(item) => deleteMutation.mutate(item)}
        onOpenEdit={openEditEditor}
        onOpenPublish={openPublishDialog}
        onRefreshDetail={() => void detailQuery.refetch()}
        onSetEnabled={(item, enabled) =>
          enableMutation.mutate({ item, enabled })
        }
        open={selectedResource != null}
        selected={selectedResource}
      />
      <PublishAgentDialog
        form={publishForm}
        onChange={setPublishForm}
        onClose={() => setPublishResource(null)}
        onSubmit={handlePublishSubmit}
        open={publishResource != null}
        pending={publishMutation.isPending}
        resource={publishResource}
      />
    </>
  )
}
