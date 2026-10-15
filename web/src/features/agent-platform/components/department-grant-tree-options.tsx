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
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { DepartmentTreeBulkActions } from '@/features/enterprise-organization/components/DepartmentTreeBulkActions'
import type { DepartmentTreeNode } from '@/features/enterprise-organization/types'
import { cn } from '@/lib/utils'

type DepartmentGrantTreeOptionsProps = {
  emptyLabel: string
  keyword: string
  nodes: DepartmentTreeNode[]
  onToggleSelected: (departmentId: string) => void
  selectedIds: string[]
}

function filterDepartmentTree(
  nodes: DepartmentTreeNode[],
  keyword: string
): DepartmentTreeNode[] {
  const normalizedKeyword = keyword.trim().toLowerCase()
  if (!normalizedKeyword) {
    return nodes
  }

  return nodes.flatMap((node) => {
    const children = filterDepartmentTree(node.children ?? [], keyword)
    const label = node.name || `#${node.id}`
    const matches = `${node.id} ${label}`
      .toLowerCase()
      .includes(normalizedKeyword)
    if (!matches && children.length === 0) {
      return []
    }
    return [{ ...node, children }]
  })
}

function getExpandableDepartmentIds(nodes: DepartmentTreeNode[]): Set<number> {
  const ids = new Set<number>()
  const visit = (items: DepartmentTreeNode[]) => {
    for (const item of items) {
      if (item.children.length > 0) {
        ids.add(item.id)
        visit(item.children)
      }
    }
  }

  visit(nodes)
  return ids
}

export function DepartmentGrantTreeOptions(
  props: DepartmentGrantTreeOptionsProps
) {
  const { t } = useTranslation()
  const [expandedIds, setExpandedIds] = useState<Set<number>>(new Set())
  const [initialized, setInitialized] = useState(false)
  const filteredNodes = useMemo(
    () => filterDepartmentTree(props.nodes, props.keyword),
    [props.keyword, props.nodes]
  )
  const isSearching = props.keyword.trim().length > 0
  const searchableExpandedIds = useMemo(
    () => getExpandableDepartmentIds(filteredNodes),
    [filteredNodes]
  )
  const visibleExpandedIds = isSearching ? searchableExpandedIds : expandedIds

  useEffect(() => {
    if (props.nodes.length === 0) {
      setInitialized(false)
      return
    }
    if (!initialized) {
      setExpandedIds(getExpandableDepartmentIds(props.nodes))
      setInitialized(true)
    }
  }, [initialized, props.nodes])

  const toggleExpanded = (departmentId: number) => {
    setExpandedIds((current) => {
      const next = new Set(current)
      if (next.has(departmentId)) {
        next.delete(departmentId)
      } else {
        next.add(departmentId)
      }
      return next
    })
  }

  if (filteredNodes.length === 0) {
    return (
      <div className='text-muted-foreground px-3 py-6 text-center text-sm'>
        {props.emptyLabel}
      </div>
    )
  }

  return (
    <div>
      {!isSearching ? (
        <div className='border-b px-2 py-1.5'>
          <DepartmentTreeBulkActions
            onExpandAll={() =>
              setExpandedIds(getExpandableDepartmentIds(props.nodes))
            }
            onCollapse={() => setExpandedIds(new Set())}
          />
        </div>
      ) : null}
      <div role='tree' aria-label={t('Departments')} className='p-1'>
        {filteredNodes.map((node) => (
          <DepartmentGrantTreeItem
            key={node.id}
            node={node}
            level={0}
            expandedIds={visibleExpandedIds}
            selectedIds={props.selectedIds}
            onToggleExpanded={toggleExpanded}
            onToggleSelected={props.onToggleSelected}
          />
        ))}
      </div>
    </div>
  )
}

function DepartmentGrantTreeItem(props: {
  expandedIds: Set<number>
  level: number
  node: DepartmentTreeNode
  onToggleExpanded: (departmentId: number) => void
  onToggleSelected: (departmentId: string) => void
  selectedIds: string[]
}) {
  const { t } = useTranslation()
  const departmentId = String(props.node.id)
  const hasChildren = props.node.children.length > 0
  const isExpanded = hasChildren && props.expandedIds.has(props.node.id)
  const isSelected = props.selectedIds.includes(departmentId)
  const label = props.node.name || `#${props.node.id}`

  return (
    <>
      <div
        role='treeitem'
        tabIndex={0}
        aria-level={props.level + 1}
        aria-expanded={hasChildren ? isExpanded : undefined}
        aria-selected={isSelected}
        style={{ paddingInlineStart: `${props.level * 16 + 8}px` }}
        className={cn(
          'hover:bg-muted focus-visible:ring-ring flex cursor-pointer items-center gap-2 rounded-sm py-1.5 pr-2 text-sm outline-none focus-visible:ring-3',
          isSelected ? 'bg-muted' : undefined
        )}
        onClick={() => props.onToggleSelected(departmentId)}
        onKeyDown={(event) => {
          if (event.key !== 'Enter' && event.key !== ' ') {
            return
          }
          event.preventDefault()
          props.onToggleSelected(departmentId)
        }}
      >
        <span className='flex size-4 shrink-0 items-center justify-center'>
          {hasChildren ? (
            <Button
              type='button'
              variant='ghost'
              size='icon'
              className='size-5'
              aria-label={
                isExpanded ? t('Collapse department') : t('Expand department')
              }
              onPointerDown={(event) => {
                event.preventDefault()
                event.stopPropagation()
              }}
              onClick={(event) => {
                event.preventDefault()
                event.stopPropagation()
                props.onToggleExpanded(props.node.id)
              }}
            >
              {isExpanded ? (
                <ChevronDown className='size-3.5' />
              ) : (
                <ChevronRight className='size-3.5' />
              )}
            </Button>
          ) : null}
        </span>
        <Checkbox
          checked={isSelected}
          onClick={(event) => event.stopPropagation()}
          onCheckedChange={() => props.onToggleSelected(departmentId)}
        />
        <span className='min-w-0 flex-1'>
          <span className='block truncate font-medium'>{label}</span>
          <span className='text-muted-foreground block truncate text-xs'>
            #{props.node.id}
          </span>
        </span>
      </div>
      {isExpanded
        ? props.node.children.map((child) => (
            <DepartmentGrantTreeItem
              key={child.id}
              node={child}
              level={props.level + 1}
              expandedIds={props.expandedIds}
              selectedIds={props.selectedIds}
              onToggleExpanded={props.onToggleExpanded}
              onToggleSelected={props.onToggleSelected}
            />
          ))
        : null}
    </>
  )
}
