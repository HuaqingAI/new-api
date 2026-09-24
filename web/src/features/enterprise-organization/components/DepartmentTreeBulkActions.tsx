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
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

export function DepartmentTreeBulkActions(props: {
  onExpandAll: () => void
  onCollapse: () => void
}) {
  const { t } = useTranslation()

  return (
    <div className='flex shrink-0 flex-wrap gap-2'>
      <Button variant='outline' size='sm' onClick={props.onExpandAll}>
        <ChevronDown className='size-4' />
        {t('Expand All')}
      </Button>
      <Button variant='outline' size='sm' onClick={props.onCollapse}>
        <ChevronRight className='size-4' />
        {t('Collapse All')}
      </Button>
    </div>
  )
}
