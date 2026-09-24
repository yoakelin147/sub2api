import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SupplierAccountsView from '../SupplierAccountsView.vue'

const profile = {
  id: 7, code: 'vendor', name: 'Vendor', status: 'active',
  allowed_account_kinds: [{ platform: 'openai', type: 'apikey' }],
  access_token: { exists: false, masked_key: null, created_at: null, last_used_at: null },
  stats: { member_count: 1, account_count: 1, pending_count: 1, schedulable_count: 0, error_count: 0 },
}
const getProfile = vi.fn().mockResolvedValue(profile)
const listAccounts = vi.fn().mockResolvedValue({
  items: [{
    id: 11, external_id: 'ext-11', name: 'Vendor Account', notes: null, platform: 'openai', type: 'apikey',
    credential_status: { api_key: true }, has_credentials: true, status: 'disabled', schedulable: false,
    review_status: 'pending', review_note: null, expires_at: null, last_used_at: null,
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  }], total: 1, page: 1, page_size: 20, pages: 1,
})

vi.mock('@/api/supplier', () => ({
  getProfile: (...args: unknown[]) => getProfile(...args),
  listAccounts: (...args: unknown[]) => listAccounts(...args),
  listProxies: vi.fn().mockResolvedValue([{ id: 3, name: 'Platform proxy' }]),
  createAccount: vi.fn(), updateAccount: vi.fn(), deleteAccount: vi.fn(), testAccount: vi.fn(),
}))

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} }, missingWarn: false, fallbackWarn: false })

describe('SupplierAccountsView', () => {
  beforeEach(() => { getProfile.mockClear(); listAccounts.mockClear() })
  it('shows the empty state rather than crashing for a supplier without account permissions', async () => {
    getProfile.mockResolvedValueOnce({ ...profile, allowed_account_kinds: null, review_required: false, auto_approve_groups: {} })
    listAccounts.mockResolvedValueOnce({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    const wrapper = mount(SupplierAccountsView, { global: {
      plugins: [createPinia(), i18n], stubs: { AppLayout: { template: '<main><slot/></main>' }, Icon: true, Pagination: true, SupplierAccountForm: true, SupplierBatchImportDialog: true },
    } })
    await flushPromises()
    expect(wrapper.text()).toContain('supplier.accounts.noAccounts')
    expect(wrapper.get('button.btn-primary').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
  it('loads tenant-scoped accounts and renders review state', async () => {
    const wrapper = mount(SupplierAccountsView, {
      global: {
        plugins: [createPinia(), i18n],
        stubs: {
          AppLayout: { template: '<main><slot/></main>' },
          Icon: true,
          Pagination: true,
          SupplierAccountForm: true,
          SupplierBatchImportDialog: true,
        },
      },
    })
    await flushPromises()

    expect(getProfile).toHaveBeenCalledOnce()
    expect(listAccounts).toHaveBeenCalledWith(1, 20, expect.objectContaining({ review_status: '' }))
    expect(wrapper.text()).toContain('Vendor Account')
    expect(wrapper.text()).toContain('ext-11')
    expect(wrapper.text()).toContain('supplier.accounts.pending')
    expect(wrapper.html()).not.toMatch(/group_ids|proxy_id|rate_multiplier|concurrency/)
  })
})
