import { apiClient } from './client'
import type { ClaudeModel, PaginatedResponse } from '@/types'

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
  review_required: boolean
  auto_approve_groups: Record<string, number[]>
  authorized_groups: SupplierGroupOption[]
  access_token: SupplierTokenStatus
  stats: SupplierStats
}

export interface SupplierGroupOption {
  id: number
  name: string
  description: string
  platform: string
  require_oauth_only: boolean
}

export interface SupplierAccount {
  id: number
  external_id: string | null
  name: string
  notes: string | null
  extra?: Record<string, unknown>
  platform: string
  type: string
  credential_status: Record<string, boolean>
  has_credentials: boolean
  status: 'active' | 'disabled'
  schedulable: boolean
  review_status: 'pending' | 'approved' | 'rejected'
  proxy_id: number | null
  group_ids?: number[]
  concurrency: number
  priority: number
  load_factor: number | null
  auto_pause_on_expired: boolean
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
  extra?: Record<string, unknown>
  expires_at?: number | null
  proxy_id?: number
  group_ids?: number[]
  concurrency?: number
  priority?: number
  load_factor?: number
  auto_pause_on_expired?: boolean
}

export interface SupplierAccountUpdateInput {
  external_id?: string
  name?: string
  notes?: string | null
  credentials?: Record<string, unknown>
  extra?: Record<string, unknown>
  group_ids?: number[]
  expires_at?: number | null
  status?: 'active' | 'disabled'
  proxy_id?: number
  concurrency?: number
  priority?: number
  load_factor?: number
  auto_pause_on_expired?: boolean
}

export interface SupplierProxyOption { id: number; name: string }

export async function listProxies(): Promise<SupplierProxyOption[]> {
  const { data } = await apiClient.get<SupplierProxyOption[]>('/supplier/proxies')
  return data
}

export interface SupplierOAuthOptions {
  proxy_id: number
  type: 'oauth' | 'setup-token'
  oauth_type?: 'code_assist' | 'google_one' | 'ai_studio'
  project_id?: string
  tier_id?: string
}

export async function getSupplierGrokOAuthCapabilities(): Promise<{ password_auth_enabled: boolean }> {
  const { data } = await apiClient.get('/supplier/oauth/grok/capabilities')
  return data
}

export async function generateSupplierOAuthURL(platform: string, options: SupplierOAuthOptions): Promise<{ auth_url: string; session_id: string; state?: string }> {
  const { data } = await apiClient.post(`/supplier/oauth/${platform}/auth-url`, options)
  return data
}

export async function exchangeSupplierOAuthCode(platform: string, type: 'oauth' | 'setup-token', sessionId: string, code: string, state: string): Promise<Record<string, unknown>> {
  const { data } = await apiClient.post(`/supplier/oauth/${platform}/exchange-code`, { type, session_id: sessionId, code, state })
  return data
}

export async function exchangeSupplierOAuthCredential(platform: string, input: {
  type: 'oauth' | 'setup-token'
  proxy_id: number
  method: 'cookie' | 'sso' | 'password'
  session_key?: string
  sso_token?: string
  email?: string
  password?: string
}): Promise<Record<string, unknown>> {
  const { data } = await apiClient.post(`/supplier/oauth/${platform}/credential-exchange`, input)
  return data
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
  const crypto = globalThis.crypto
  if (crypto?.randomUUID) return crypto.randomUUID()
  if (!crypto?.getRandomValues) {
    throw new Error('Secure random UUID generation is unavailable')
  }
  const bytes = crypto.getRandomValues(new Uint8Array(16))
  bytes[6] = (bytes[6] & 0x0f) | 0x40
  bytes[8] = (bytes[8] & 0x3f) | 0x80
  const hex = Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('')
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
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

export async function getAccountTestModels(id: number): Promise<ClaudeModel[]> {
  const { data } = await apiClient.get<ClaudeModel[]>(`/supplier/accounts/${id}/models`)
  return data
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
  getAccountTestModels,
  getAccessTokenStatus,
  regenerateAccessToken,
  revokeAccessToken,
}
