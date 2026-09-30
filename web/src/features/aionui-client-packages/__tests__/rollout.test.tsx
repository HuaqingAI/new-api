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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { I18nextProvider } from 'react-i18next'
import { describe, expect, it, vi } from 'vitest'

import { getDepartmentTree } from '@/features/enterprise-organization/api'
import type { DepartmentTreeNode } from '@/features/enterprise-organization/types'
import { searchUsers } from '@/features/users/api'
import type { User } from '@/features/users/types'
import i18n from '@/i18n/config'

import { ClientPackageRolloutFields } from '../components/client-package-rollout'
import {
  clientPackageRolloutScopes,
  type ClientPackageRolloutFormValue,
} from '../rollout'

vi.mock('@/features/enterprise-organization/api', () => ({
  getDepartmentTree: vi.fn(),
}))

vi.mock('@/features/users/api', () => ({
  searchUsers: vi.fn(),
}))

const departmentTree: DepartmentTreeNode[] = [
  {
    id: 1,
    tenant_id: 0,
    name: 'Headquarters',
    parent_id: null,
    status: 1,
    source_type: 1,
    external_id: '',
    sync_status: 0,
    sync_error: '',
    name_history: [],
    created_at: 0,
    updated_at: 0,
    deleted_at: 0,
    children: [
      {
        id: 2,
        tenant_id: 0,
        name: 'Engineering',
        parent_id: 1,
        status: 1,
        source_type: 1,
        external_id: '',
        sync_status: 0,
        sync_error: '',
        name_history: [],
        created_at: 0,
        updated_at: 0,
        deleted_at: 0,
        children: [
          {
            id: 3,
            tenant_id: 0,
            name: 'Platform',
            parent_id: 2,
            status: 1,
            source_type: 1,
            external_id: '',
            sync_status: 0,
            sync_error: '',
            name_history: [],
            created_at: 0,
            updated_at: 0,
            deleted_at: 0,
            children: [],
          },
        ],
      },
    ],
  },
]

const alice: User = {
  id: 7,
  username: 'alice',
  display_name: 'Alice Example',
  email: 'alice@example.com',
  quota: 0,
  used_quota: 0,
  request_count: 0,
  group: 'default',
  status: 1,
  role: 1,
}

const otherUser: User = {
  ...alice,
  id: 8,
  username: 'other',
  display_name: 'Other User',
  email: 'other@example.com',
}

const existingUser: User = {
  ...alice,
  id: 48,
  username: 'existing-user',
  display_name: 'Existing User',
  email: 'existing@example.com',
}

function TargetedRolloutFields(props: { userIds?: string[] }) {
  const [value, setValue] = useState<ClientPackageRolloutFormValue>({
    rolloutMode: 'targeted' as const,
    userIds: props.userIds ?? [],
    departmentIds: [],
  })

  return <ClientPackageRolloutFields value={value} onChange={setValue} />
}

function renderTargetedRolloutFields(props?: { userIds?: string[] }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <TargetedRolloutFields userIds={props?.userIds} />
      </I18nextProvider>
    </QueryClientProvider>
  )
}

function IndependentRolloutFields() {
  const [uploadValue, setUploadValue] = useState<ClientPackageRolloutFormValue>(
    {
      rolloutMode: 'global',
      userIds: [],
      departmentIds: [],
    }
  )
  const [editorValue, setEditorValue] = useState<ClientPackageRolloutFormValue>(
    {
      rolloutMode: 'global',
      userIds: [],
      departmentIds: [],
    }
  )

  return (
    <>
      <output data-testid='upload-rollout-mode'>
        {uploadValue.rolloutMode}
      </output>
      <ClientPackageRolloutFields
        value={uploadValue}
        onChange={setUploadValue}
      />
      <output data-testid='editor-rollout-mode'>
        {editorValue.rolloutMode}
      </output>
      <ClientPackageRolloutFields
        value={editorValue}
        onChange={setEditorValue}
      />
    </>
  )
}

describe('clientPackageRolloutScopes', () => {
  it('omits retained selections when rollout is global', () => {
    expect(
      clientPackageRolloutScopes({
        rolloutMode: 'global',
        userIds: ['7'],
        departmentIds: ['11'],
      })
    ).toEqual([])
  })

  it('serializes selected users and departments for targeted rollout', () => {
    expect(
      clientPackageRolloutScopes({
        rolloutMode: 'targeted',
        userIds: ['7', '9'],
        departmentIds: ['11'],
      })
    ).toEqual([
      { subject_type: 'user', subject_id: '7' },
      { subject_type: 'user', subject_id: '9' },
      { subject_type: 'department', subject_id: '11' },
    ])
  })

  it('keeps the upload rollout unchanged when editing a second rollout form', async () => {
    vi.mocked(searchUsers).mockResolvedValue({
      success: true,
      data: { items: [], total: 0, page: 1, page_size: 20 },
    })
    vi.mocked(getDepartmentTree).mockResolvedValue({
      success: true,
      data: departmentTree,
    })
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const user = userEvent.setup()

    render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <IndependentRolloutFields />
        </I18nextProvider>
      </QueryClientProvider>
    )

    await user.click(screen.getAllByText('Selected users and departments')[1])

    expect(screen.getByTestId('upload-rollout-mode')).toHaveTextContent(
      'global'
    )
    expect(screen.getByTestId('editor-rollout-mode')).toHaveTextContent(
      'targeted'
    )
  })

  it('shows matching remote users while searching targeted rollout grants', async () => {
    vi.mocked(searchUsers).mockImplementation(async ({ keyword }) => {
      let items: User[] = []
      if (keyword === 'alice') {
        items = [alice]
      } else if (!keyword) {
        items = [otherUser]
      }
      return {
        success: true,
        data: {
          items,
          total: items.length,
          page: 1,
          page_size: 20,
        },
      }
    })
    vi.mocked(getDepartmentTree).mockResolvedValue({
      success: true,
      data: departmentTree,
    })
    const user = userEvent.setup()

    renderTargetedRolloutFields()

    await user.click(screen.getByRole('button', { name: /Select users/ }))
    await user.type(
      screen.getByPlaceholderText('Search users by name or email'),
      'alice'
    )

    await waitFor(() =>
      expect(searchUsers).toHaveBeenLastCalledWith({
        keyword: 'alice',
        status: '1',
        p: 1,
        page_size: 20,
      })
    )
    expect(await screen.findByText('Alice Example')).toBeVisible()
    expect(screen.queryByText('Other User')).not.toBeInTheDocument()
  })

  it('filters the targeted rollout department selector as an organization tree', async () => {
    vi.mocked(searchUsers).mockResolvedValue({
      success: true,
      data: { items: [], total: 0, page: 1, page_size: 20 },
    })
    vi.mocked(getDepartmentTree).mockResolvedValue({
      success: true,
      data: departmentTree,
    })
    const user = userEvent.setup()

    renderTargetedRolloutFields()

    await user.click(screen.getByRole('button', { name: /Select departments/ }))
    await user.type(
      screen.getByPlaceholderText('Search departments'),
      'platform'
    )

    expect(
      await screen.findByRole('treeitem', { name: /Platform/ })
    ).toBeVisible()
    expect(
      screen.getByRole('treeitem', { name: /Headquarters/ })
    ).toHaveAttribute('aria-expanded', 'true')
  })

  it('expands and collapses all targeted rollout department branches', async () => {
    vi.mocked(searchUsers).mockResolvedValue({
      success: true,
      data: { items: [], total: 0, page: 1, page_size: 20 },
    })
    vi.mocked(getDepartmentTree).mockResolvedValue({
      success: true,
      data: departmentTree,
    })
    const user = userEvent.setup()

    renderTargetedRolloutFields()

    await user.click(screen.getByRole('button', { name: /Select departments/ }))
    await screen.findByRole('treeitem', { name: /Platform/ })

    await user.click(screen.getByRole('button', { name: 'Collapse All' }))
    expect(
      screen.queryByRole('treeitem', { name: /Engineering/ })
    ).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Expand All' }))
    expect(screen.getByRole('treeitem', { name: /Platform/ })).toBeVisible()
  })

  it('uses the selected department name in the rollout trigger', async () => {
    vi.mocked(searchUsers).mockResolvedValue({
      success: true,
      data: { items: [], total: 0, page: 1, page_size: 20 },
    })
    vi.mocked(getDepartmentTree).mockResolvedValue({
      success: true,
      data: departmentTree,
    })
    const user = userEvent.setup()

    renderTargetedRolloutFields()

    await user.click(screen.getByRole('button', { name: /Select departments/ }))
    await user.click(
      await screen.findByRole('treeitem', { name: /Headquarters/ })
    )

    expect(screen.getByRole('button', { name: /Headquarters/ })).toBeVisible()
    expect(screen.queryByText('Department #1')).not.toBeInTheDocument()
  })

  it('resolves selected users by ID when editing a targeted rollout', async () => {
    vi.mocked(searchUsers).mockImplementation(async ({ keyword }) => {
      const items = keyword === '48' ? [existingUser] : []
      return {
        success: true,
        data: {
          items,
          total: items.length,
          page: 1,
          page_size: 20,
        },
      }
    })
    vi.mocked(getDepartmentTree).mockResolvedValue({
      success: true,
      data: departmentTree,
    })

    renderTargetedRolloutFields({ userIds: ['48'] })

    await waitFor(() =>
      expect(searchUsers).toHaveBeenCalledWith({
        keyword: '48',
        p: 1,
        page_size: 20,
      })
    )
    expect(
      await screen.findByRole('button', { name: /Existing User/ })
    ).toBeVisible()
    expect(screen.queryByText('User #48')).not.toBeInTheDocument()
  })
})
