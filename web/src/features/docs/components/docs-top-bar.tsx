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
import { ThemeSwitch } from '@/components/theme-switch'
import { useSystemConfig } from '@/hooks/use-system-config'

export function DocsTopBar() {
  const { systemName, logo } = useSystemConfig()

  return (
    <header className='bg-background/95 supports-[backdrop-filter]:bg-background/80 sticky top-0 z-30 border-b backdrop-blur'>
      <div className='mx-auto flex h-14 max-w-7xl items-center justify-between gap-3 px-4 md:px-8'>
        <a href='/' className='flex min-w-0 items-center gap-2.5'>
          <span className='bg-primary/10 flex size-7 shrink-0 items-center justify-center rounded-lg'>
            <img
              src={logo}
              alt={systemName}
              className='size-5 rounded-md object-contain'
            />
          </span>
          <span className='truncate text-base font-semibold'>{systemName}</span>
        </a>

        <div className='flex min-w-0 items-center gap-3'>
          <ThemeSwitch />
        </div>
      </div>
    </header>
  )
}
