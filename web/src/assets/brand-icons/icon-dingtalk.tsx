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
import { type SVGProps } from 'react'

import { cn } from '@/lib/utils'

export function IconDingTalk({ className, ...props }: SVGProps<SVGSVGElement>) {
  return (
    <svg
      role='img'
      viewBox='0 0 24 24'
      xmlns='http://www.w3.org/2000/svg'
      width='20'
      height='20'
      className={cn('fill-current', className)}
      {...props}
    >
      <title>DingTalk</title>
      <path d='M4.2 3.1c2.3 1.1 4.8 2 7.5 2.6 2.6.6 5.3.9 8.1.9.4 0 .7.3.7.7 0 3.2-.9 6.1-2.7 8.4-1.8 2.4-4.4 4-7.9 4.8-.6.1-.9-.6-.5-1l2.7-2.8-5.5-.4c-.6 0-.8-.8-.3-1.1l3.8-2.3-5.7-1.6c-.5-.1-.7-.8-.2-1.1l3.4-2.1-4-3.7c-.5-.5 0-1.4.6-1.1z' />
    </svg>
  )
}
