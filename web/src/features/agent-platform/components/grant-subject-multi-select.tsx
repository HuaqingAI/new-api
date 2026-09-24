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
import { ChevronsUpDown, XCircle } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

export type GrantSelectOption = {
  value: string
  label: string
  description: string
}

export function GrantSubjectMultiSelect(props: {
  disabled?: boolean
  emptyLabel: string
  loading: boolean
  onSearchChange: (value: string) => void
  onSelectedIdsChange: (ids: string[]) => void
  options: GrantSelectOption[]
  placeholder: string
  searchPlaceholder: string
  searchValue: string
  selectedIds: string[]
  selectedOptions?: GrantSelectOption[]
  renderOptions?: (params: {
    selectedIds: string[]
    toggleValue: (value: string) => void
  }) => ReactNode
}) {
  const { t } = useTranslation()
  const selectedOptions = props.selectedIds.map((id) => {
    const option =
      props.options.find((item) => item.value === id) ??
      props.selectedOptions?.find((item) => item.value === id)
    return (
      option ?? {
        value: id,
        label: `#${id}`,
        description: `#${id}`,
      }
    )
  })

  const toggleValue = (value: string) => {
    if (props.disabled) {
      return
    }
    if (props.selectedIds.includes(value)) {
      props.onSelectedIdsChange(props.selectedIds.filter((id) => id !== value))
      return
    }
    props.onSelectedIdsChange([...props.selectedIds, value])
  }

  let optionsContent: ReactNode
  if (props.loading) {
    optionsContent = (
      <div className='text-muted-foreground px-3 py-6 text-center text-sm'>
        {t('Loading')}
      </div>
    )
  } else if (props.renderOptions) {
    optionsContent = props.renderOptions({
      selectedIds: props.selectedIds,
      toggleValue,
    })
  } else {
    optionsContent = (
      <>
        <CommandEmpty>{props.emptyLabel}</CommandEmpty>
        <CommandGroup>
          {props.options.map((option) => {
            const selected = props.selectedIds.includes(option.value)
            return (
              <CommandItem
                key={option.value}
                value={`${option.label} ${option.description}`}
                data-checked={selected}
                onSelect={() => toggleValue(option.value)}
              >
                <Checkbox checked={selected} />
                <span className='min-w-0 flex-1'>
                  <span className='block truncate font-medium'>
                    {option.label}
                  </span>
                  <span className='text-muted-foreground block truncate text-xs'>
                    {option.description}
                  </span>
                </span>
              </CommandItem>
            )
          })}
        </CommandGroup>
      </>
    )
  }

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant='outline'
            disabled={props.disabled}
            className='h-auto min-h-9 w-full justify-between px-3 py-2'
          >
            <span className='flex min-w-0 flex-1 flex-wrap gap-1 text-left'>
              {selectedOptions.length === 0 ? (
                <span className='text-muted-foreground'>
                  {props.placeholder}
                </span>
              ) : (
                selectedOptions.map((option) => (
                  <Badge
                    key={option.value}
                    variant='secondary'
                    className='max-w-[220px] gap-1 rounded-md'
                  >
                    <span className='truncate'>{option.label}</span>
                    <span
                      role='button'
                      tabIndex={0}
                      aria-label={t('Remove')}
                      className='hover:bg-muted-foreground/20 inline-flex size-4 shrink-0 items-center justify-center rounded-full'
                      onClick={(event) => {
                        event.preventDefault()
                        event.stopPropagation()
                        toggleValue(option.value)
                      }}
                      onKeyDown={(event) => {
                        if (event.key !== 'Enter' && event.key !== ' ') {
                          return
                        }
                        event.preventDefault()
                        event.stopPropagation()
                        toggleValue(option.value)
                      }}
                    >
                      <XCircle className='size-3' />
                    </span>
                  </Badge>
                ))
              )}
            </span>
            <ChevronsUpDown className='text-muted-foreground ml-2 size-4 shrink-0' />
          </Button>
        }
      />
      <PopoverContent
        className='w-[420px] max-w-[calc(100vw-3rem)] p-0'
        align='start'
      >
        <Command shouldFilter={false}>
          <CommandInput
            disabled={props.disabled}
            value={props.searchValue}
            onValueChange={props.onSearchChange}
            placeholder={props.searchPlaceholder}
          />
          <CommandList>{optionsContent}</CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
