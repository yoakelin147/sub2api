import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SupplierDetail from '../SupplierDetail.vue'
import SupplierMemberDialog from '../SupplierMemberDialog.vue'

const approveAccounts = vi.fn().mockResolvedValue(undefined)
const addMember = vi.fn().mockResolvedValue(undefined)
const listMembers = vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, page_size: 100, pages: 1 })
const listAccounts = vi.fn().mockResolvedValue({
  items: [{
    id: 11, name: 'Pending Account', notes: null, platform: 'openai', type: 'apikey', supplier_id: 7,
    supplier_external_id: 'ext-11', review_status: 'pending', review_note: null,
    credentials_status: { has_api_key: true }, status: 'disabled', schedulable: false,
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  }], total: 1, page: 1, page_size: 20, pages: 1,
})

vi.mock('@/api/admin/suppliers', () => ({ default: {
  listMembers: (...args: unknown[]) => listMembers(...args),
  listAccounts: (...args: unknown[]) => listAccounts(...args),
  approveAccounts: (...args: unknown[]) => approveAccounts(...args),
  rejectAccounts: vi.fn(), pauseAccounts: vi.fn(), addMember: (...args: unknown[]) => addMember(...args), removeMember: vi.fn(),
  regenerateAccessToken: vi.fn(), getAccessTokenStatus: vi.fn(), revokeAccessToken: vi.fn(),
} }))

vi.mock('@/api/admin/groups', () => ({ default: {
  getAll: vi.fn().mockResolvedValue([{ id: 3, name: 'OpenAI', platform: 'openai', status: 'active' }]),
} }))

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} }, missingWarn: false, fallbackWarn: false })
const supplier = {
  id: 7, code: 'vendor-a', name: 'Vendor A', status: 'active' as const, notes: null,
  allowed_account_kinds: [{ platform: 'openai', type: 'apikey' }],
  access_token: { exists: false, masked_key: null, created_at: null, last_used_at: null },
  stats: { member_count: 0, account_count: 1, pending_count: 1, schedulable_count: 0, error_count: 0 },
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

describe('SupplierDetail', () => {
  beforeEach(() => vi.clearAllMocks())

  it('approves only selected accounts with selected groups', async () => {
    const wrapper = mount(SupplierDetail, {
      props: { supplier },
      global: { plugins: [createPinia(), i18n], stubs: { Icon: true, Pagination: true } },
    })
    await flushPromises()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    await checkboxes.find((input) => input.attributes('aria-label') === 'Pending Account')!.setValue(true)
    await checkboxes.find((input) => input.element.getAttribute('value') === '3')!.setValue(true)
    const approveButton = wrapper.findAll('button').find((button) => button.text() === 'supplier.admin.approve')!
    await approveButton.trigger('click')
    await flushPromises()

    expect(approveAccounts).toHaveBeenCalledWith(7, {
      account_ids: [11],
      group_ids: [3],
      note: null,
    })
  })

  it('opens the member dialog and refreshes the member list after adding', async () => {
    const wrapper = mount(SupplierDetail, {
      props: { supplier },
      global: { plugins: [createPinia(), i18n], stubs: { Icon: true, Pagination: true } },
    })
    await flushPromises()

    await wrapper.findAll('button').find((button) => button.text() === 'supplier.admin.addMember')!.trigger('click')
    const dialog = wrapper.getComponent(SupplierMemberDialog)
    expect(dialog.props('show')).toBe(true)

    dialog.vm.$emit('submit', { email: 'new@example.test', password: 'Password!123', username: 'new', concurrency: 1 })
    await flushPromises()

    expect(addMember).toHaveBeenCalledWith(7, {
      email: 'new@example.test',
      password: 'Password!123',
      username: 'new',
      concurrency: 1,
    })
    expect(dialog.props('show')).toBe(false)
    expect(listMembers).toHaveBeenCalledTimes(2)
  })
})
