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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { I18nextProvider } from 'react-i18next'
import { describe, expect, it, vi } from 'vitest'

import { Command, CommandList } from '@/components/ui/command'
import type { DepartmentTreeNode } from '@/features/enterprise-organization/types'
import i18n from '@/i18n/config'

import { DepartmentGrantTreeOptions } from '../department-grant-tree-options'

function departmentNode(
  overrides: Partial<DepartmentTreeNode>
): DepartmentTreeNode {
  return {
    id: overrides.id ?? 1,
    tenant_id: 0,
    name: overrides.name ?? 'Department',
    parent_id: overrides.parent_id ?? null,
    status: 1,
    source_type: 1,
    external_id: '',
    sync_status: 0,
    sync_error: '',
    name_history: [],
    created_at: 0,
    updated_at: 0,
    deleted_at: 0,
    children: overrides.children ?? [],
  }
}

const departmentTree = [
  departmentNode({
    id: 1,
    name: 'Headquarters',
    children: [
      departmentNode({
        id: 2,
        parent_id: 1,
        name: 'Engineering',
        children: [
          departmentNode({
            id: 3,
            parent_id: 2,
            name: 'Platform',
          }),
        ],
      }),
    ],
  }),
]

function renderDepartmentGrantTree(props?: {
  keyword?: string
  onToggleSelected?: (departmentId: string) => void
}) {
  return render(
    <I18nextProvider i18n={i18n}>
      <Command shouldFilter={false}>
        <CommandList>
          <DepartmentGrantTreeOptions
            emptyLabel='No departments found'
            keyword={props?.keyword ?? ''}
            nodes={departmentTree}
            selectedIds={[]}
            onToggleSelected={props?.onToggleSelected ?? vi.fn()}
          />
        </CommandList>
      </Command>
    </I18nextProvider>
  )
}

describe('DepartmentGrantTreeOptions', () => {
  it('collapses and expands all department descendants from the bulk actions', async () => {
    renderDepartmentGrantTree()
    const user = userEvent.setup()

    await waitFor(() =>
      expect(screen.getByRole('treeitem', { name: /Platform/ })).toBeVisible()
    )
    expect(
      screen.getByRole('treeitem', { name: /Headquarters/ })
    ).toHaveAttribute('aria-level', '1')
    expect(
      screen.getByRole('treeitem', { name: /Engineering/ })
    ).toHaveAttribute('aria-level', '2')
    expect(screen.getByRole('treeitem', { name: /Platform/ })).toHaveAttribute(
      'aria-level',
      '3'
    )

    await user.click(screen.getByRole('button', { name: 'Collapse All' }))

    expect(
      screen.getByRole('treeitem', { name: /Headquarters/ })
    ).toHaveAttribute('aria-expanded', 'false')
    expect(
      screen.queryByRole('treeitem', { name: /Engineering/ })
    ).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Expand All' }))

    expect(screen.getByRole('treeitem', { name: /Platform/ })).toBeVisible()
  })

  it('expands matching ancestors when searching a nested department', async () => {
    renderDepartmentGrantTree({ keyword: 'platform' })

    await waitFor(() =>
      expect(screen.getByRole('treeitem', { name: /Platform/ })).toBeVisible()
    )
    expect(
      screen.getByRole('treeitem', { name: /Headquarters/ })
    ).toHaveAttribute('aria-expanded', 'true')
    expect(
      screen.getByRole('treeitem', { name: /Engineering/ })
    ).toHaveAttribute('aria-expanded', 'true')
  })

  it('keeps department selection on the clicked tree node', async () => {
    const onToggleSelected = vi.fn()
    renderDepartmentGrantTree({ onToggleSelected })
    const user = userEvent.setup()

    await waitFor(() =>
      expect(screen.getByRole('treeitem', { name: /Platform/ })).toBeVisible()
    )
    await user.click(screen.getByRole('treeitem', { name: /Platform/ }))

    expect(onToggleSelected).toHaveBeenCalledWith('3')
  })
})
