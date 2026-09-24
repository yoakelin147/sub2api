import { apiClient } from '../client'
import type { AdminUser, PaginatedResponse } from '@/types'
import type {
  SupplierAccountFilters,
  SupplierAccountKind,
  SupplierTokenStatus,
  SupplierStats,
} from '@/api/supplier'

export interface AdminSupplierAccount {
  id: number
  name: string
  notes: string | null
  platform: string
  type: string
  supplier_id: number
  supplier_external_id?: string | null
  review_status: 'pending' | 'approved' | 'rejected'
  reviewed_at?: string | null
  reviewed_by?: number | null
  review_note?: string | null
  credentials_status?: Record<string, boolean>
  credentials?: Record<string, unknown>
  extra?: Record<string, unknown>
  proxy_id?: number | null
  concurrency?: number
  priority?: number
  load_factor?: number | null
  rate_multiplier?: number
  auto_pause_on_expired?: boolean
  status: 'active' | 'disabled'
  schedulable: boolean
  error_message?: string
  expires_at?: number | null
  created_at: string
  updated_at: string
}

export interface AdminSupplier {
  id: number
  code: string
  name: string
  status: 'active' | 'disabled'
  notes: string | null
  allowed_account_kinds: SupplierAccountKind[]
  review_required: boolean
  auto_approve_groups: Record<string, number[]>
  access_token: SupplierTokenStatus
  stats: SupplierStats
  created_at: string
  updated_at: string
}

export interface SupplierWriteInput {
  code?: string
  name: string
  status?: 'active' | 'disabled'
  notes?: string | null
  allowed_account_kinds: SupplierAccountKind[]
  review_required: boolean
  auto_approve_groups: Record<string, number[]>
}

export interface SupplierMemberInput {
  user_id?: number
  email?: string
  password?: string
  username?: string
  concurrency?: number
}

export interface SupplierReviewInput {
  account_ids: number[]
  group_ids?: number[]
  note?: string | null
}

export interface AdminSupplierAccountUpdateInput {
  name: string
  external_id: string
  notes: string
  proxy_id: number
  concurrency: number
  priority: number
  load_factor?: number
  clear_load_factor?: boolean
  rate_multiplier?: number
  auto_pause_on_expired: boolean
  expires_at?: number
  clear_expires_at?: boolean
  credentials: Record<string, unknown>
  extra?: Record<string, unknown>
}

export async function list(page = 1, pageSize = 20, filters: { status?: string; search?: string } = {}): Promise<PaginatedResponse<AdminSupplier>> {
  const { data } = await apiClient.get<PaginatedResponse<AdminSupplier>>('/admin/suppliers', {
    params: { page, page_size: pageSize, ...filters },
  })
  return data
}

export async function get(id: number): Promise<AdminSupplier> {
  const { data } = await apiClient.get<AdminSupplier>(`/admin/suppliers/${id}`)
  return data
}

export async function create(input: SupplierWriteInput): Promise<AdminSupplier> {
  const { data } = await apiClient.post<AdminSupplier>('/admin/suppliers', input)
  return data
}

export async function update(id: number, input: SupplierWriteInput): Promise<AdminSupplier> {
  const { data } = await apiClient.put<AdminSupplier>(`/admin/suppliers/${id}`, input)
  return data
}

export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/suppliers/${id}`)
}

export async function listMembers(id: number, page = 1, pageSize = 100): Promise<PaginatedResponse<AdminUser>> {
  const { data } = await apiClient.get<PaginatedResponse<AdminUser>>(`/admin/suppliers/${id}/members`, {
    params: { page, page_size: pageSize },
  })
  return data
}

export async function addMember(id: number, input: SupplierMemberInput): Promise<AdminUser> {
  const { data } = await apiClient.post<AdminUser>(`/admin/suppliers/${id}/members`, input)
  return data
}

export async function removeMember(id: number, userId: number): Promise<void> {
  await apiClient.delete(`/admin/suppliers/${id}/members/${userId}`)
}

export async function getAccessTokenStatus(id: number): Promise<SupplierTokenStatus> {
  const { data } = await apiClient.get<SupplierTokenStatus>(`/admin/suppliers/${id}/access-token`)
  return data
}

export async function regenerateAccessToken(id: number): Promise<{ key: string }> {
  const { data } = await apiClient.post<{ key: string }>(`/admin/suppliers/${id}/access-token/regenerate`)
  return data
}

export async function revokeAccessToken(id: number): Promise<void> {
  await apiClient.delete(`/admin/suppliers/${id}/access-token`)
}

export async function listAccounts(
  id: number,
  page = 1,
  pageSize = 20,
  filters: SupplierAccountFilters = {},
): Promise<PaginatedResponse<AdminSupplierAccount>> {
  const { data } = await apiClient.get<PaginatedResponse<AdminSupplierAccount>>(`/admin/suppliers/${id}/accounts`, {
    params: { page, page_size: pageSize, ...filters },
  })
  return data
}

export async function getAccount(id: number, accountId: number): Promise<AdminSupplierAccount> {
  const { data } = await apiClient.get<AdminSupplierAccount>(`/admin/suppliers/${id}/accounts/${accountId}`)
  return data
}

export async function updateAccount(id: number, accountId: number, input: AdminSupplierAccountUpdateInput): Promise<AdminSupplierAccount> {
  const { data } = await apiClient.put<AdminSupplierAccount>(`/admin/suppliers/${id}/accounts/${accountId}`, input)
  return data
}

export async function revealAccountPassword(id: number, accountId: number): Promise<string> {
  const { data } = await apiClient.get<{ password: string }>(`/admin/suppliers/${id}/accounts/${accountId}/password`)
  return data.password
}

async function review(id: number, action: 'approve' | 'reject' | 'pause', input: SupplierReviewInput): Promise<void> {
  await apiClient.post(`/admin/suppliers/${id}/accounts/${action}`, input)
}

export const approveAccounts = (id: number, input: SupplierReviewInput) => review(id, 'approve', input)
export const rejectAccounts = (id: number, input: SupplierReviewInput) => review(id, 'reject', input)
export const pauseAccounts = (id: number, input: SupplierReviewInput) => review(id, 'pause', input)

export default {
  list,
  get,
  create,
  update,
  remove,
  listMembers,
  addMember,
  removeMember,
  getAccessTokenStatus,
  regenerateAccessToken,
  revokeAccessToken,
  listAccounts,
  getAccount,
  updateAccount,
  revealAccountPassword,
  approveAccounts,
  rejectAccounts,
  pauseAccounts,
}
