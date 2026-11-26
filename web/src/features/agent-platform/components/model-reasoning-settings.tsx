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
import { useQuery } from '@tanstack/react-query'
import { Brain } from 'lucide-react'
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { ROLE } from '@/lib/roles'
import { getServerErrorMessage } from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

import {
  getModelReasoningPolicy,
  MODEL_REASONING_QUERY_KEY,
} from '../model-reasoning-api'
import { ModelReasoningEditor } from './model-reasoning-editor'

export function ModelReasoningSettings() {
  const { t } = useTranslation()
  const isRoot = useAuthStore(
    (state) => (state.auth.user?.role ?? 0) >= ROLE.SUPER_ADMIN
  )
  const [open, setOpen] = useState(false)
  const trigger = useRef<HTMLButtonElement>(null)
  const query = useQuery({
    queryKey: MODEL_REASONING_QUERY_KEY,
    queryFn: getModelReasoningPolicy,
    enabled: isRoot && open,
    refetchOnMount: 'always',
    meta: { errorToast: false },
  })

  if (!isRoot) return null

  const close = () => {
    setOpen(false)
    trigger.current?.focus()
  }

  return (
    <>
      <Button
        ref={trigger}
        type='button'
        variant='outline'
        size='sm'
        onClick={() => setOpen(true)}
      >
        <Brain className='size-4' aria-hidden='true' />
        {t('Model reasoning settings')}
      </Button>
      {open && query.data && (
        <ModelReasoningEditor config={query.data} onClose={close} />
      )}
      {open && !query.data && (
        <Dialog
          open
          title={t('Model reasoning settings')}
          onOpenChange={(value) => !value && close()}
        >
          {query.isError ? (
            <ErrorState
              title={t('Failed to load model reasoning settings')}
              description={getServerErrorMessage(query.error)}
              onRetry={() => void query.refetch()}
            />
          ) : (
            <LoadingState />
          )}
        </Dialog>
      )}
    </>
  )
}
