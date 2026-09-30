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
import { useMutation, useQuery } from '@tanstack/react-query'
import { type ReactNode, useEffect, useId, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { DepartmentGrantTreeOptions } from '@/features/agent-platform/components/department-grant-tree-options'
import {
  GrantSubjectMultiSelect,
  type GrantSelectOption,
} from '@/features/agent-platform/components/grant-subject-multi-select'
import { getDepartmentTree } from '@/features/enterprise-organization/api'
import type { DepartmentTreeNode } from '@/features/enterprise-organization/types'
import { searchUsers } from '@/features/users/api'
import type { User } from '@/features/users/types'

import {
  type AionUiClientPackage,
  type AionUiClientPackageRollout,
  type AionUiClientPackageRolloutMode,
  getClientPackageRollout,
  updateClientPackageRollout,
} from '../api'
import {
  clientPackageRolloutScopes,
  defaultClientPackageRollout,
  type ClientPackageRolloutFormValue,
} from '../rollout'

export function ClientPackageRolloutFields(props: {
  disabled?: boolean
  onChange: (value: ClientPackageRolloutFormValue) => void
  value: ClientPackageRolloutFormValue
}) {
  const { t } = useTranslation()
  const rolloutFieldId = useId()
  const [userSearch, setUserSearch] = useState('')
  const [departmentSearch, setDepartmentSearch] = useState('')
  const [knownUserOptions, setKnownUserOptions] = useState<GrantSelectOption[]>(
    []
  )
  const targeted = props.value.rolloutMode === 'targeted'
  const targetScopeCount =
    props.value.userIds.length + props.value.departmentIds.length
  const usersQuery = useQuery({
    queryKey: ['aionui-client-packages', 'rollout-users', userSearch],
    queryFn: async () => {
      const result = await searchUsers({
        keyword: userSearch,
        status: '1',
        p: 1,
        page_size: 20,
      })
      if (!result.success) {
        throw new Error(result.message || t('Request failed'))
      }
      return result.data?.items ?? []
    },
    enabled: targeted,
  })
  const departmentsQuery = useQuery({
    queryKey: ['aionui-client-packages', 'rollout-departments'],
    queryFn: async () => {
      const result = await getDepartmentTree()
      if (!result.success) {
        throw new Error(result.message || t('Request failed'))
      }
      return result.data ?? []
    },
    enabled: targeted,
  })
  const selectedUsersQuery = useQuery({
    queryKey: [
      'aionui-client-packages',
      'rollout-selected-users',
      props.value.userIds,
    ],
    queryFn: async () => {
      const options = await Promise.all(
        props.value.userIds.map(async (userID) => {
          const userId = Number(userID)
          if (!Number.isSafeInteger(userId) || userId <= 0) {
            return null
          }
          const result = await searchUsers({
            keyword: String(userId),
            p: 1,
            page_size: 20,
          })
          const user = result.data?.items.find((item) => item.id === userId)
          if (!result.success || !user) {
            return null
          }
          return userGrantOption(user)
        })
      )
      return options.filter(
        (option): option is GrantSelectOption => option != null
      )
    },
    enabled: targeted && props.value.userIds.length > 0,
  })
  const searchedUserOptions = useMemo(
    () => (usersQuery.data ?? []).map(userGrantOption),
    [usersQuery.data]
  )

  useEffect(() => {
    if (searchedUserOptions.length === 0) {
      return
    }
    setKnownUserOptions((current) => mergeOptions(current, searchedUserOptions))
  }, [searchedUserOptions])

  const selectedUserOptions = useMemo(
    () =>
      addSelectedFallbacks(
        mergeOptions(
          knownUserOptions,
          searchedUserOptions,
          selectedUsersQuery.data ?? []
        ),
        props.value.userIds,
        t('User #{{id}}')
      ),
    [
      knownUserOptions,
      props.value.userIds,
      searchedUserOptions,
      selectedUsersQuery.data,
      t,
    ]
  )
  const userOptions = useMemo(() => {
    if (!userSearch.trim()) {
      return selectedUserOptions
    }
    return searchedUserOptions
  }, [searchedUserOptions, selectedUserOptions, userSearch])
  const departmentOptions = useMemo(
    () =>
      addSelectedFallbacks(
        flattenDepartments(departmentsQuery.data ?? []),
        props.value.departmentIds,
        t('Department #{{id}}')
      ),
    [departmentsQuery.data, props.value.departmentIds, t]
  )

  return (
    <Field>
      <FieldLabel>{t('Release rollout')}</FieldLabel>
      <FieldDescription>
        {t('Choose who can discover this client update after it is published.')}
      </FieldDescription>
      <RadioGroup
        disabled={props.disabled}
        value={props.value.rolloutMode}
        onValueChange={(rolloutMode) =>
          props.onChange({
            ...props.value,
            rolloutMode: rolloutMode as AionUiClientPackageRolloutMode,
          })
        }
        className='gap-3'
      >
        <div className='flex items-center gap-2'>
          <RadioGroupItem
            id={`client-rollout-global-${rolloutFieldId}`}
            value='global'
          />
          <Label
            htmlFor={`client-rollout-global-${rolloutFieldId}`}
            className='cursor-pointer font-normal'
          >
            {t('All users')}
          </Label>
        </div>
        <div className='flex items-center gap-2'>
          <RadioGroupItem
            id={`client-rollout-targeted-${rolloutFieldId}`}
            value='targeted'
          />
          <Label
            htmlFor={`client-rollout-targeted-${rolloutFieldId}`}
            className='cursor-pointer font-normal'
          >
            {t('Selected users and departments')}
          </Label>
        </div>
      </RadioGroup>

      {targeted ? (
        <div className='grid gap-4 border-l pl-4'>
          <Field>
            <FieldLabel>{t('Users')}</FieldLabel>
            <GrantSubjectMultiSelect
              disabled={props.disabled}
              emptyLabel={t('No users found')}
              loading={usersQuery.isFetching}
              onSearchChange={setUserSearch}
              onSelectedIdsChange={(userIds) =>
                props.onChange({ ...props.value, userIds })
              }
              options={userOptions}
              placeholder={t('Select users')}
              searchPlaceholder={t('Search users by name or email')}
              searchValue={userSearch}
              selectedIds={props.value.userIds}
              selectedOptions={selectedUserOptions}
            />
          </Field>
          <Field>
            <FieldLabel>{t('Departments')}</FieldLabel>
            <GrantSubjectMultiSelect
              disabled={props.disabled}
              emptyLabel={t('No departments found')}
              loading={departmentsQuery.isFetching}
              onSearchChange={setDepartmentSearch}
              onSelectedIdsChange={(departmentIds) =>
                props.onChange({ ...props.value, departmentIds })
              }
              options={departmentOptions}
              placeholder={t('Select departments')}
              searchPlaceholder={t('Search departments')}
              searchValue={departmentSearch}
              selectedIds={props.value.departmentIds}
              renderOptions={({ selectedIds, toggleValue }) => (
                <DepartmentGrantTreeOptions
                  emptyLabel={t('No departments found')}
                  keyword={departmentSearch}
                  nodes={departmentsQuery.data ?? []}
                  selectedIds={selectedIds}
                  onToggleSelected={toggleValue}
                />
              )}
            />
          </Field>
        </div>
      ) : null}
      {targeted && targetScopeCount === 0 ? (
        <FieldDescription className='text-destructive'>
          {t('Add at least one user or department grant')}
        </FieldDescription>
      ) : null}
    </Field>
  )
}

export function ClientPackageRolloutDialog(props: {
  item: AionUiClientPackage | null
  onClose: () => void
  onUpdated: () => void
}) {
  const { t } = useTranslation()
  const item = props.item
  const packageID = item?.id
  const [value, setValue] = useState<ClientPackageRolloutFormValue>(
    defaultClientPackageRollout
  )
  const rolloutQuery = useQuery({
    queryKey: ['aionui-client-packages', 'rollout', packageID],
    queryFn: async () => {
      if (!packageID) {
        return null
      }
      const result = await getClientPackageRollout(packageID)
      if (!result.success || !result.data) {
        throw new Error(result.message || t('Request failed'))
      }
      return result.data
    },
    enabled: packageID != null,
  })

  useEffect(() => {
    if (!packageID) {
      setValue(defaultClientPackageRollout)
      return
    }
    if (rolloutQuery.data) {
      setValue(rolloutFormValue(rolloutQuery.data))
    }
  }, [packageID, rolloutQuery.data])

  const updateMutation = useMutation({
    mutationFn: async () => {
      if (!item) {
        throw new Error(t('Request failed'))
      }
      const result = await updateClientPackageRollout(item.id, {
        rollout_mode: value.rolloutMode,
        scopes: clientPackageRolloutScopes(value),
      })
      if (!result.success) {
        throw new Error(result.message || t('Request failed'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Release rollout updated'))
      props.onUpdated()
      props.onClose()
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })
  const scopeCount = value.userIds.length + value.departmentIds.length
  const submitDisabled =
    rolloutQuery.isLoading ||
    rolloutQuery.isError ||
    updateMutation.isPending ||
    (value.rolloutMode === 'targeted' && scopeCount === 0)
  let rolloutContent: ReactNode
  if (rolloutQuery.isLoading) {
    rolloutContent = (
      <div className='text-muted-foreground py-6 text-center text-sm'>
        {t('Loading...')}
      </div>
    )
  } else if (rolloutQuery.isError) {
    rolloutContent = (
      <div className='text-destructive py-6 text-center text-sm'>
        {rolloutQuery.error instanceof Error
          ? rolloutQuery.error.message
          : t('Request failed')}
      </div>
    )
  } else {
    rolloutContent = (
      <ClientPackageRolloutFields
        disabled={updateMutation.isPending}
        onChange={setValue}
        value={value}
      />
    )
  }

  return (
    <Dialog
      open={packageID != null}
      onOpenChange={(open) => !open && props.onClose()}
      title={t('Edit release rollout')}
      description={
        item
          ? t('Manage which users can discover version {{version}}.', {
              version: item.version,
            })
          : undefined
      }
      footer={
        <>
          <Button variant='outline' onClick={props.onClose}>
            {t('Cancel')}
          </Button>
          <Button
            disabled={submitDisabled}
            onClick={() => updateMutation.mutate()}
          >
            {updateMutation.isPending ? t('Saving...') : t('Save')}
          </Button>
        </>
      }
    >
      {rolloutContent}
    </Dialog>
  )
}

function rolloutFormValue(
  rollout: AionUiClientPackageRollout
): ClientPackageRolloutFormValue {
  return {
    rolloutMode: rollout.rollout_mode,
    userIds: rollout.scopes
      .filter((scope) => scope.subject_type === 'user')
      .map((scope) => scope.subject_id),
    departmentIds: rollout.scopes
      .filter((scope) => scope.subject_type === 'department')
      .map((scope) => scope.subject_id),
  }
}

function userGrantOption(user: User): GrantSelectOption {
  const displayName = user.display_name.trim()
  const username = user.username.trim()
  const email = user.email?.trim()
  const label = displayName || username || `#${user.id}`
  return {
    value: String(user.id),
    label,
    description: [
      `#${user.id}`,
      username !== label ? username : '',
      email ?? '',
    ]
      .filter(Boolean)
      .join(' · '),
  }
}

function flattenDepartments(nodes: DepartmentTreeNode[]): GrantSelectOption[] {
  const options: GrantSelectOption[] = []
  const visit = (items: DepartmentTreeNode[]) => {
    for (const node of items) {
      options.push({
        value: String(node.id),
        label: node.name || `#${node.id}`,
        description: `#${node.id}`,
      })
      visit(node.children ?? [])
    }
  }
  visit(nodes)
  return options
}

function mergeOptions(
  ...optionGroups: GrantSelectOption[][]
): GrantSelectOption[] {
  const values = new Map<string, GrantSelectOption>()
  for (const options of optionGroups) {
    for (const option of options) {
      values.set(option.value, option)
    }
  }
  return [...values.values()]
}

function addSelectedFallbacks(
  options: GrantSelectOption[],
  selected: string[],
  fallbackTemplate: string
): GrantSelectOption[] {
  return mergeOptions(
    selected.map((value) => ({
      value,
      label: fallbackTemplate.replace('{{id}}', value),
      description: `#${value}`,
    })),
    options
  )
}
