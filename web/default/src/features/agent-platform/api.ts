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

export type AgentPlatformSkillItem = {
  id: number
  resource_id: string
  resource_type: string
  display_name: string
  owner_user_id: number
  status: string
  latest_version: string
  tenant_id: number
  created_at: number
  updated_at: number
}

export async function getAgentPlatformSkills() {
  const res = await api.get('/api/agent-platform/skills', {
    params: { page: 1, page_size: 12 },
  })
  return res.data
}
