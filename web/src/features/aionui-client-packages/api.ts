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

export type AionUiClientPlatform =
  | 'windows_x64'
  | 'mac_arm64'
  | 'mac_x64'

export type AionUiClientPackageStatus = 'draft' | 'published' | 'disabled'

export type AionUiClientPackage = {
  id: number
  platform: AionUiClientPlatform
  version: string
  status: AionUiClientPackageStatus
  file_name: string
  file_sha256: string
  file_sha512: string
  file_size: number
  update_file_name?: string
  update_file_sha256?: string
  update_file_sha512?: string
  update_file_size?: number
  update_metadata_file_name?: string
  update_metadata_sha256?: string
  release_note?: string
  created_by: number
  published_at?: string
  created_at: string
  updated_at: string
}

export type ClientPackageListResponse = {
  success: boolean
  message?: string
  data?: {
    items: AionUiClientPackage[]
    total: number
    page: number
    page_size: number
  }
}

export type ClientPackageLatestResponse = {
  success: boolean
  message?: string
  data?: {
    items: Array<{
      platform: AionUiClientPlatform
      available: boolean
      release: AionUiClientPackage | null
    }>
  }
}

export type UploadClientPackagePayload = {
  platform: AionUiClientPlatform
  version: string
  releaseNote: string
  publish: boolean
  file: File | null
  updateFile: File | null
  updateMetadataFile: File | null
}

export async function getClientPackages(params: {
  platform?: string
  status?: string
}) {
  const res = await api.get<ClientPackageListResponse>(
    '/api/aionui/client-packages',
    {
      params: {
        page: 1,
        page_size: 100,
        platform: params.platform || undefined,
        status: params.status || undefined,
      },
    }
  )
  return res.data
}

export async function getLatestClientPackages() {
  const res = await api.get<ClientPackageLatestResponse>(
    '/api/aionui/client-packages/latest'
  )
  return res.data
}

export async function uploadClientPackage(payload: UploadClientPackagePayload) {
  const formData = new FormData()
  formData.set('platform', payload.platform)
  formData.set('version', payload.version)
  formData.set('release_note', payload.releaseNote)
  formData.set('publish', payload.publish ? 'true' : 'false')
  if (payload.file) {
    formData.set('file', payload.file)
  }
  if (payload.updateFile) {
    formData.set('update_file', payload.updateFile)
  }
  if (payload.updateMetadataFile) {
    formData.set('update_metadata_file', payload.updateMetadataFile)
  }
  const res = await api.post<{ success: boolean; message?: string }>(
    '/api/aionui/client-packages',
    formData
  )
  return res.data
}

export async function updateClientPackageStatus(
  id: number,
  status: AionUiClientPackageStatus
) {
  const res = await api.patch<{ success: boolean; message?: string }>(
    `/api/aionui/client-packages/${id}/status`,
    { status }
  )
  return res.data
}

export async function deleteClientPackage(id: number) {
  const res = await api.delete<{ success: boolean; message?: string }>(
    `/api/aionui/client-packages/${id}`
  )
  return res.data
}

export function clientPackageDownloadUrl(id: number) {
  return `/api/aionui/client-packages/${id}/download`
}
