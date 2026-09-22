import { apiClient } from '../client'
import type { AdminUser, PaginatedResponse } from '@/types'
import type {
  SupplierAccount,
  SupplierAccountFilters,
  SupplierAccountKind,
  SupplierTokenStatus,
} from '@/api/supplier'

export interface AdminSupplier {
  id: number
  code: string
  name: string
  status: 'active' | 'disabled'
  notes: string | null
  allowed_account_kinds: SupplierAccountKind[]
  access_token: SupplierTokenStatus
  created_at: string
  updated_at: string
}

export interface SupplierWriteInput {
  code: string
  name: string
  status?: 'active' | 'disabled'
  notes?: string | null
  allowed_account_kinds: SupplierAccountKind[]
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
): Promise<PaginatedResponse<SupplierAccount>> {
  const { data } = await apiClient.get<PaginatedResponse<SupplierAccount>>(`/admin/suppliers/${id}/accounts`, {
    params: { page, page_size: pageSize, ...filters },
  })
  return data
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
  approveAccounts,
  rejectAccounts,
  pauseAccounts,
}
