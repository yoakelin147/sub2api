import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'

export interface SupplierAccountKind {
  platform: string
  type: string
}

export interface SupplierTokenStatus {
  exists: boolean
  masked_key: string | null
  created_at: string | null
  last_used_at: string | null
}

export interface SupplierStats {
  member_count: number
  account_count: number
  pending_count: number
  schedulable_count: number
  error_count: number
}

export interface SupplierProfile {
  id: number
  code: string
  name: string
  status: 'active' | 'disabled'
  allowed_account_kinds: SupplierAccountKind[]
  access_token: SupplierTokenStatus
  stats: SupplierStats
}

export interface SupplierAccount {
  id: number
  external_id: string | null
  name: string
  notes: string | null
  platform: string
  type: string
  credential_status: Record<string, boolean>
  has_credentials: boolean
  status: 'active' | 'disabled'
  schedulable: boolean
  review_status: 'pending' | 'approved' | 'rejected'
  review_note: string | null
  expires_at: string | null
  last_used_at: string | null
  created_at: string
  updated_at: string
}

export interface SupplierAccountInput {
  external_id?: string
  name: string
  notes?: string | null
  platform: string
  type: string
  credentials: Record<string, unknown>
  expires_at?: number | null
}

export interface SupplierAccountUpdateInput {
  external_id?: string
  name?: string
  notes?: string | null
  credentials?: Record<string, unknown>
  expires_at?: number | null
  status?: 'active' | 'disabled'
}

export interface SupplierAccountFilters {
  platform?: string
  type?: string
  status?: string
  review_status?: string
  search?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export interface SupplierBatchItemResult {
  index: number
  external_id?: string
  account_id?: number
  success: boolean
  error_code?: string
  message?: string
}

export interface SupplierBatchResult {
  total: number
  succeeded: number
  failed: number
  results: SupplierBatchItemResult[]
}

export function createSupplierIdempotencyKey(): string {
  if (!globalThis.crypto?.randomUUID) {
    throw new Error('Secure random UUID generation is unavailable')
  }
  return globalThis.crypto.randomUUID()
}

export async function getProfile(): Promise<SupplierProfile> {
  const { data } = await apiClient.get<SupplierProfile>('/supplier/me')
  return data
}

export async function listAccounts(
  page = 1,
  pageSize = 20,
  filters: SupplierAccountFilters = {},
): Promise<PaginatedResponse<SupplierAccount>> {
  const { data } = await apiClient.get<PaginatedResponse<SupplierAccount>>('/supplier/accounts', {
    params: { page, page_size: pageSize, ...filters },
  })
  return data
}

export async function getAccount(id: number): Promise<SupplierAccount> {
  const { data } = await apiClient.get<SupplierAccount>(`/supplier/accounts/${id}`)
  return data
}

export async function createAccount(
  input: SupplierAccountInput,
  idempotencyKey = createSupplierIdempotencyKey(),
): Promise<SupplierAccount> {
  const { data } = await apiClient.post<SupplierAccount>('/supplier/accounts', input, {
    headers: { 'Idempotency-Key': idempotencyKey },
  })
  return data
}

export async function batchCreateAccounts(
  accounts: SupplierAccountInput[],
  idempotencyKey = createSupplierIdempotencyKey(),
): Promise<SupplierBatchResult> {
  const { data } = await apiClient.post<SupplierBatchResult>(
    '/supplier/accounts/batch',
    { accounts },
    { headers: { 'Idempotency-Key': idempotencyKey } },
  )
  return data
}

export async function updateAccount(id: number, input: SupplierAccountUpdateInput): Promise<SupplierAccount> {
  const { data } = await apiClient.put<SupplierAccount>(`/supplier/accounts/${id}`, input)
  return data
}

export async function deleteAccount(id: number): Promise<void> {
  await apiClient.delete(`/supplier/accounts/${id}`)
}

export async function testAccount(
  id: number,
  input: { model?: string; prompt?: string; mode?: string } = {},
): Promise<boolean> {
  const { data } = await apiClient.post<string>(`/supplier/accounts/${id}/test`, input, { responseType: 'text' })
  // Account tests return SSE, including failures sent with HTTP 200.
  if (typeof data !== 'string') return false
  let completed = false
  for (const block of data.split(/\r?\n\r?\n/)) {
    const payload = block.split(/\r?\n/).filter(line => line.startsWith('data:')).map(line => line.slice(5).trimStart()).join('\n')
    if (!payload) continue
    try {
      const event = JSON.parse(payload)
      if (!event || typeof event !== 'object' || event.type === 'error') return false
      if (event.type === 'test_complete') completed = event.success === true
    } catch { return false }
  }
  return completed
}

export async function getAccessTokenStatus(): Promise<SupplierTokenStatus> {
  const { data } = await apiClient.get<SupplierTokenStatus>('/supplier/access-token')
  return data
}

export async function regenerateAccessToken(): Promise<{ key: string }> {
  const { data } = await apiClient.post<{ key: string }>('/supplier/access-token/regenerate')
  return data
}

export async function revokeAccessToken(): Promise<void> {
  await apiClient.delete('/supplier/access-token')
}

export default {
  getProfile,
  listAccounts,
  getAccount,
  createAccount,
  batchCreateAccounts,
  updateAccount,
  deleteAccount,
  testAccount,
  getAccessTokenStatus,
  regenerateAccessToken,
  revokeAccessToken,
}
