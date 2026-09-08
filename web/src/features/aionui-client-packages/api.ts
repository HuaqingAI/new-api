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

export type AionUiClientPlatform = 'windows_x64' | 'mac_arm64' | 'mac_x64'

export type AionUiClientPackageStatus = 'draft' | 'published' | 'disabled'

export type AionUiClientPackageRolloutMode = 'global' | 'targeted'

export type AionUiClientPackageScope = {
  subject_type: 'user' | 'department'
  subject_id: string
}

export type AionUiClientPackageRollout = {
  rollout_mode: AionUiClientPackageRolloutMode
  scopes: AionUiClientPackageScope[]
}

export type AionUiClientPackage = {
  id: number
  platform: AionUiClientPlatform
  version: string
  status: AionUiClientPackageStatus
  rollout_mode: AionUiClientPackageRolloutMode
  scope_count: number
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
  rolloutMode: AionUiClientPackageRolloutMode
  scopes: AionUiClientPackageScope[]
  file: File | null
  updateFile: File | null
  updateMetadataFile: File | null
}

type ClientPackageUploadKind = 'download' | 'update' | 'metadata'

type ClientPackageUploadFile = {
  kind: ClientPackageUploadKind
  file_name: string
  sha256: string
  sha512: string
  size: number
}

type ClientPackageUploadTarget = ClientPackageUploadFile & {
  object_uri: string
  object_key: string
  upload_url: string
  content_type: string
  headers: Record<string, string>
  expires_at: number
}

type ClientPackageUploadInitResponse = {
  success: boolean
  message?: string
  data?: {
    files: ClientPackageUploadTarget[]
  }
}

type ClientPackageUploadCompleteResponse = {
  success: boolean
  message?: string
  data?: AionUiClientPackage
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
  if (!payload.file) {
    throw new Error('Download installer is required')
  }

  const files: Array<{ kind: ClientPackageUploadKind; file: File }> = [
    { kind: 'download', file: payload.file },
  ]
  if (payload.updateFile) {
    files.push({ kind: 'update', file: payload.updateFile })
  }
  if (payload.updateMetadataFile) {
    files.push({ kind: 'metadata', file: payload.updateMetadataFile })
  }

  const uploadFiles = await Promise.all(
    files.map(async (item) => ({
      kind: item.kind,
      file_name: item.file.name,
      sha256: await hashFile(item.file, 'SHA-256', 'hex'),
      sha512: await hashFile(item.file, 'SHA-512', 'base64'),
      size: item.file.size,
    }))
  )

  const initRes = await api.post<ClientPackageUploadInitResponse>(
    '/api/aionui/client-packages/uploads/init',
    {
      platform: payload.platform,
      version: payload.version,
      publish: payload.publish,
      files: uploadFiles,
    }
  )
  if (!initRes.data.success || !initRes.data.data) {
    throw new Error(initRes.data.message || 'Failed to create upload')
  }

  for (const target of initRes.data.data.files) {
    const source = files.find((item) => item.kind === target.kind)?.file
    if (!source) {
      throw new Error('Upload target does not match selected files')
    }
    const uploadRes = await fetch(target.upload_url, {
      method: 'PUT',
      headers: target.headers,
      body: source,
    })
    if (!uploadRes.ok) {
      throw new Error(`OSS upload failed: HTTP ${uploadRes.status}`)
    }
  }

  const completeRes = await api.post<ClientPackageUploadCompleteResponse>(
    '/api/aionui/client-packages/uploads/complete',
    {
      platform: payload.platform,
      version: payload.version,
      release_note: payload.releaseNote,
      publish: payload.publish,
      rollout_mode: payload.rolloutMode,
      scopes: payload.scopes,
      file: initRes.data.data.files.find((item) => item.kind === 'download'),
      update_file: initRes.data.data.files.find(
        (item) => item.kind === 'update'
      ),
      update_metadata_file: initRes.data.data.files.find(
        (item) => item.kind === 'metadata'
      ),
    }
  )
  return completeRes.data
}

async function hashFile(
  file: File,
  algorithm: 'SHA-256' | 'SHA-512',
  encoding: 'base64' | 'hex'
) {
  const digest = await crypto.subtle.digest(algorithm, await file.arrayBuffer())
  const bytes = [...new Uint8Array(digest)]
  if (encoding === 'hex') {
    return bytes.map((item) => item.toString(16).padStart(2, '0')).join('')
  }
  let binary = ''
  for (const byte of bytes) {
    binary += String.fromCharCode(byte)
  }
  return btoa(binary)
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

export async function getClientPackageRollout(id: number): Promise<{
  success: boolean
  message?: string
  data?: AionUiClientPackageRollout
}> {
  const res = await api.get<{
    success: boolean
    message?: string
    data?: AionUiClientPackageRollout
  }>(`/api/aionui/client-packages/${id}/rollout`)
  return res.data
}

export async function updateClientPackageRollout(
  id: number,
  payload: AionUiClientPackageRollout
): Promise<{
  success: boolean
  message?: string
  data?: AionUiClientPackageRollout
}> {
  const res = await api.put<{
    success: boolean
    message?: string
    data?: AionUiClientPackageRollout
  }>(`/api/aionui/client-packages/${id}/rollout`, payload)
  return res.data
}

export async function deleteClientPackage(id: number) {
  const res = await api.delete<{ success: boolean; message?: string }>(
    `/api/aionui/client-packages/${id}`
  )
  return res.data
}

export async function getClientPackageDownloadUrl(id: number): Promise<string> {
  const res = await api.get<{
    success: boolean
    message?: string
    data?: { url?: string }
  }>(`/api/aionui/client-packages/${id}/download-url`)
  const url = res.data.data?.url
  if (!res.data.success || !url) {
    throw new Error(res.data.message || 'Failed to prepare download')
  }
  return url
}

export function clientPackageDownloadUrl(id: number) {
  return `/api/aionui/client-packages/${id}/download`
}
