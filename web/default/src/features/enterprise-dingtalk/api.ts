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
import { api } from '@/lib/api'
import type {
  ApiResponse,
  DingTalkConfig,
  DingTalkConfigPayload,
} from './types'

export const enterpriseDingTalkQueryKey = [
  'enterprise',
  'dingtalk',
  'config',
] as const

export async function getDingTalkConfig(): Promise<
  ApiResponse<DingTalkConfig>
> {
  const res = await api.get('/api/enterprise/dingtalk/config')
  return res.data
}

export async function saveDingTalkConfig(
  payload: DingTalkConfigPayload
): Promise<ApiResponse<DingTalkConfig>> {
  const res = await api.put('/api/enterprise/dingtalk/config', payload)
  return res.data
}
