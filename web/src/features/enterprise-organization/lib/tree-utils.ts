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
import type { DepartmentTreeNode } from '../types'

type DepartmentLookup = {
  orderedIds: number[]
  rootIds: number[]
  byId: Map<number, DepartmentTreeNode>
  parentById: Map<number, number | null>
}

function orderDepartmentIds(ids: Iterable<number>, orderedIds: number[]) {
  const included = new Set(ids)
  return orderedIds.filter((id) => included.has(id))
}

export type ResolvedDepartmentSelection = {
  selectedDepartmentId: number | null
  normalizedDepartmentId: number | null
  requiredExpandedIds: number[]
}

function createDepartmentLookup(nodes: DepartmentTreeNode[]): DepartmentLookup {
  const orderedIds: number[] = []
  const rootIds: number[] = []
  const byId = new Map<number, DepartmentTreeNode>()
  const parentById = new Map<number, number | null>()

  const walk = (items: DepartmentTreeNode[], parentId: number | null) => {
    for (const item of items) {
      orderedIds.push(item.id)
      byId.set(item.id, item)
      parentById.set(item.id, parentId)
      if (parentId === null) {
        rootIds.push(item.id)
      }
      if (item.children.length > 0) {
        walk(item.children, item.id)
      }
    }
  }

  walk(nodes, null)

  return {
    orderedIds,
    rootIds,
    byId,
    parentById,
  }
}

export function findDepartmentNode(
  nodes: DepartmentTreeNode[],
  departmentId: number | null | undefined
) {
  if (!departmentId) return null
  return createDepartmentLookup(nodes).byId.get(departmentId) ?? null
}

export function getFirstVisibleDepartmentId(nodes: DepartmentTreeNode[]) {
  return createDepartmentLookup(nodes).orderedIds[0] ?? null
}

export function getAncestorDepartmentIds(
  nodes: DepartmentTreeNode[],
  departmentId: number | null | undefined
) {
  if (!departmentId) return []

  const lookup = createDepartmentLookup(nodes)
  if (!lookup.byId.has(departmentId)) return []

  const ancestors: number[] = []
  let current = lookup.parentById.get(departmentId) ?? null

  while (current !== null) {
    ancestors.unshift(current)
    current = lookup.parentById.get(current) ?? null
  }

  return ancestors
}

export function getDefaultExpandedDepartmentIds(
  nodes: DepartmentTreeNode[],
  departmentId: number | null | undefined
) {
  const lookup = createDepartmentLookup(nodes)
  const expanded = new Set<number>(lookup.rootIds)
  const selected =
    departmentId && lookup.byId.has(departmentId)
      ? lookup.byId.get(departmentId)!
      : null

  for (const ancestorId of getAncestorDepartmentIds(nodes, departmentId)) {
    expanded.add(ancestorId)
  }

  if (selected?.children.length) {
    expanded.add(selected.id)
  }

  return orderDepartmentIds(expanded, lookup.orderedIds)
}

export function resolveDepartmentSelection(
  nodes: DepartmentTreeNode[],
  departmentId: number | null | undefined
): ResolvedDepartmentSelection {
  const fallbackDepartmentId = getFirstVisibleDepartmentId(nodes)
  const selectedDepartment =
    findDepartmentNode(nodes, departmentId) ??
    findDepartmentNode(nodes, fallbackDepartmentId)

  const normalizedDepartmentId = selectedDepartment?.id ?? null

  return {
    selectedDepartmentId: normalizedDepartmentId,
    normalizedDepartmentId,
    requiredExpandedIds: getDefaultExpandedDepartmentIds(
      nodes,
      normalizedDepartmentId
    ),
  }
}

export function syncExpandedDepartmentIds(
  expandedIds: number[],
  nodes: DepartmentTreeNode[],
  requiredExpandedIds: number[]
) {
  const lookup = createDepartmentLookup(nodes)
  const validIds = new Set(lookup.orderedIds)
  const next = new Set<number>()

  for (const departmentId of expandedIds) {
    if (validIds.has(departmentId)) {
      next.add(departmentId)
    }
  }

  for (const departmentId of requiredExpandedIds) {
    if (validIds.has(departmentId)) {
      next.add(departmentId)
    }
  }

  return orderDepartmentIds(next, lookup.orderedIds)
}

export function toggleExpandedDepartmentId(
  expandedIds: number[],
  departmentId: number
) {
  const next = new Set(expandedIds)
  if (next.has(departmentId)) {
    next.delete(departmentId)
  } else {
    next.add(departmentId)
  }
  return [...next]
}
