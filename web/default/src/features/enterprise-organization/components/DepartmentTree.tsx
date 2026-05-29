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
import { AlertTriangle, GitBranch, History, Link2, Minus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import {
  DEPARTMENT_SOURCE_TYPE,
  DEPARTMENT_STATUS,
  DEPARTMENT_SYNC_STATUS,
} from '../constants'
import type { DepartmentTreeNode } from '../types'

interface DepartmentTreeProps {
  nodes: DepartmentTreeNode[]
}

export function DepartmentTree({ nodes }: DepartmentTreeProps) {
  const { t } = useTranslation()

  return (
    <div className='overflow-x-auto rounded-lg border'>
      <div className='bg-muted/60 text-muted-foreground grid min-w-[760px] grid-cols-[minmax(260px,1.5fr)_120px_130px_160px_120px] gap-3 border-b px-4 py-2 text-xs font-medium'>
        <span>{t('Department')}</span>
        <span>{t('Status')}</span>
        <span>{t('Source')}</span>
        <span>{t('External ID')}</span>
        <span>{t('Sync')}</span>
      </div>
      <div className='min-w-[760px] divide-y'>
        {nodes.map((node) => (
          <DepartmentTreeRow key={node.id} node={node} level={0} />
        ))}
      </div>
    </div>
  )
}

function DepartmentTreeRow({
  node,
  level,
}: {
  node: DepartmentTreeNode
  level: number
}) {
  const { t } = useTranslation()
  const hasChildren = node.children.length > 0

  return (
    <>
      <div className='grid grid-cols-[minmax(260px,1.5fr)_120px_130px_160px_120px] gap-3 px-4 py-3 text-sm'>
        <div className='flex min-w-0 items-center gap-2'>
          <div
            className='flex shrink-0 items-center'
            style={{ width: `${level * 22 + 18}px` }}
          >
            {hasChildren ? (
              <GitBranch className='text-muted-foreground size-4' />
            ) : (
              <Minus className='text-muted-foreground/60 size-4' />
            )}
          </div>
          <div className='min-w-0'>
            <div className='truncate font-medium'>{node.name}</div>
            <div className='text-muted-foreground mt-1 flex flex-wrap items-center gap-1.5 text-xs'>
              <span>{t('ID {{id}}', { id: node.id })}</span>
              {node.parent_id == null ? (
                <span>{t('Root department')}</span>
              ) : (
                <span>{t('Parent ID {{id}}', { id: node.parent_id })}</span>
              )}
              {node.name_history.length > 0 ? (
                <span className='inline-flex items-center gap-1'>
                  <History className='size-3' />
                  {t('{{count}} historical name(s)', {
                    count: node.name_history.length,
                  })}
                </span>
              ) : null}
            </div>
          </div>
        </div>
        <div className='flex items-center'>
          <DepartmentStatusBadge status={node.status} />
        </div>
        <div className='flex items-center'>
          <SourceBadge sourceType={node.source_type} />
        </div>
        <div className='text-muted-foreground flex min-w-0 items-center gap-1'>
          {node.external_id ? <Link2 className='size-3.5 shrink-0' /> : null}
          <span className='truncate'>
            {node.external_id || t('Local only')}
          </span>
        </div>
        <div className='flex min-w-0 items-center'>
          <SyncStatusBadge node={node} />
        </div>
      </div>
      {node.children.map((child) => (
        <DepartmentTreeRow key={child.id} node={child} level={level + 1} />
      ))}
    </>
  )
}

function DepartmentStatusBadge({ status }: { status: number }) {
  const { t } = useTranslation()
  let label = t('Enabled')
  if (status === DEPARTMENT_STATUS.DISABLED) {
    label = t('Disabled')
  } else if (status === DEPARTMENT_STATUS.DELETED) {
    label = t('Deleted upstream')
  }
  return (
    <Badge
      variant={status === DEPARTMENT_STATUS.ENABLED ? 'secondary' : 'outline'}
    >
      {label}
    </Badge>
  )
}

function SourceBadge({ sourceType }: { sourceType: number }) {
  const { t } = useTranslation()
  const label =
    sourceType === DEPARTMENT_SOURCE_TYPE.DINGTALK ? t('DingTalk') : t('Manual')
  return <Badge variant='outline'>{label}</Badge>
}

function SyncStatusBadge({ node }: { node: DepartmentTreeNode }) {
  const { t } = useTranslation()
  const isFailed = node.sync_status === DEPARTMENT_SYNC_STATUS.FAILED
  const isWarning = node.sync_status === DEPARTMENT_SYNC_STATUS.WARNING
  let label = t('Not synced')
  if (node.sync_status === DEPARTMENT_SYNC_STATUS.OK) {
    label = t('Synced')
  } else if (isFailed) {
    label = t('Sync failed')
  } else if (isWarning) {
    label = t('Sync warning')
  }

  return (
    <span
      className={cn(
        'inline-flex min-w-0 items-center gap-1 text-xs',
        isFailed || isWarning ? 'text-destructive' : 'text-muted-foreground'
      )}
      title={node.sync_error || label}
    >
      {isFailed || isWarning ? <AlertTriangle className='size-3.5' /> : null}
      <span className='truncate'>{label}</span>
    </span>
  )
}
