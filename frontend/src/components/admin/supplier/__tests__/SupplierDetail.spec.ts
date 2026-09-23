import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SupplierDetail from '../SupplierDetail.vue'
import SupplierMemberDialog from '../SupplierMemberDialog.vue'
import SupplierAccountsPanel from '../SupplierAccountsPanel.vue'
import SupplierMembersPanel from '../SupplierMembersPanel.vue'
import Pagination from '@/components/common/Pagination.vue'

const approveAccounts = vi.fn().mockResolvedValue(undefined)
const addMember = vi.fn().mockResolvedValue(undefined)
const removeMember = vi.fn().mockResolvedValue(undefined)
const listMembers = vi.fn()
const listAccounts = vi.fn()
const accountResult = {
  items: [{
    id: 11, name: 'Pending Account', notes: null, platform: 'openai', type: 'apikey', supplier_id: 7,
    supplier_external_id: 'ext-11', review_status: 'pending', review_note: null,
    credentials_status: { has_api_key: true }, status: 'disabled', schedulable: false,
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  }], total: 1, page: 1, page_size: 20, pages: 1,
}
const memberResult = { items: [{ id: 21, email: 'member@example.test', username: 'Member', status: 'active', role: 'supplier' }], total: 101, page: 1, page_size: 20, pages: 6 }

vi.mock('@/api/admin/suppliers', () => ({ default: {
  listMembers: (...args: unknown[]) => listMembers(...args),
  listAccounts: (...args: unknown[]) => listAccounts(...args),
  getAccount: vi.fn().mockResolvedValue({ id: 11, name: 'Pending Account', platform: 'openai', type: 'oauth', credentials: { email: 'login@example.test' }, credentials_status: { has_access_token: true, has_login_password_encrypted: true }, proxy_id: 3, concurrency: 1, priority: 50, auto_pause_on_expired: true }),
  revealAccountPassword: vi.fn().mockResolvedValue('web-secret'),
  approveAccounts: (...args: unknown[]) => approveAccounts(...args),
  rejectAccounts: vi.fn(), pauseAccounts: vi.fn(), addMember: (...args: unknown[]) => addMember(...args), removeMember: (...args: unknown[]) => removeMember(...args),
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
const global = { plugins: [createPinia(), i18n], stubs: { Icon: true, Pagination: true, teleport: true } }
enableAutoUnmount(afterEach)

describe('SupplierDetail', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listMembers.mockReset().mockResolvedValue(memberResult)
    listAccounts.mockReset().mockResolvedValue(accountResult)
  })

  it('shows review configuration with a masked password until explicitly revealed', async () => {
    listAccounts.mockResolvedValue({ ...accountResult, items: [{ ...accountResult.items[0], credentials: { email: 'login@example.test' }, credentials_status: { has_login_password_encrypted: true } }] })
    const wrapper = mount(SupplierAccountsPanel, { props: { supplierId: 7 }, global })
    await flushPromises()
    expect(wrapper.text()).toContain('login@example.test')
    expect(wrapper.text()).not.toContain('web-secret')
    await wrapper.findAll('button').find(button => button.text() === 'supplier.admin.configuration')!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('access_token')
    expect(wrapper.text()).not.toContain('web-secret')
    await wrapper.findAll('button').find(button => button.text() === 'supplier.admin.showPassword')!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('web-secret')
    await wrapper.findAll('button').find(button => button.text() === 'supplier.admin.hidePassword')!.trigger('click')
    expect(wrapper.text()).not.toContain('web-secret')
  })

  it('approves only selected accounts with selected groups', async () => {
    const wrapper = mount(SupplierDetail, {
      props: { supplier },
      global,
    })
    await flushPromises()

    expect(listMembers).not.toHaveBeenCalled()
    expect(wrapper.find('#supplier-review-form').exists()).toBe(false)
    await wrapper.get('input[aria-label="Pending Account"]').setValue(true)
    const approveButton = wrapper.findAll('button').find((button) => button.text() === 'supplier.admin.approve')!
    await approveButton.trigger('click')
    await flushPromises()
    expect(wrapper.get('button[form="supplier-review-form"]').attributes('disabled')).toBeDefined()
    await wrapper.get('input[value="3"]').setValue(true)
    await wrapper.get('#supplier-review-form').trigger('submit')
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
      global,
    })
    await flushPromises()

    await wrapper.get('#supplier-tab-members').trigger('click')
    await flushPromises()
    expect(wrapper.findComponent(SupplierAccountsPanel).exists()).toBe(false)
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
    expect(listMembers).toHaveBeenLastCalledWith(7, 1, 20)
  })

  it('paginates beyond 100 members and resets to page one when changing page size', async () => {
    const wrapper = mount(SupplierMembersPanel, { props: { supplierId: 7 }, global })
    await flushPromises()
    const pagination = wrapper.getComponent(Pagination)
    expect(pagination.props('total')).toBe(101)
    pagination.vm.$emit('update:page', 6)
    await flushPromises()
    expect(listMembers).toHaveBeenLastCalledWith(7, 6, 20)
    pagination.vm.$emit('update:pageSize', 50)
    await flushPromises()
    expect(listMembers).toHaveBeenLastCalledWith(7, 1, 50)
  })

  it('returns to the previous page after removing the final member on the last page', async () => {
    vi.spyOn(window, 'confirm').mockReturnValueOnce(true)
    const wrapper = mount(SupplierMembersPanel, { props: { supplierId: 7 }, global })
    await flushPromises()
    wrapper.getComponent(Pagination).vm.$emit('update:page', 6)
    await flushPromises()
    listMembers.mockResolvedValue({ ...memberResult, total: 100 })
    await wrapper.findAll('button').find(button => button.text() === 'supplier.admin.removeMember')!.trigger('click')
    await flushPromises()
    expect(removeMember).toHaveBeenCalledWith(7, 21)
    expect(listMembers).toHaveBeenLastCalledWith(7, 5, 20)
    expect(wrapper.getComponent(Pagination).props('page')).toBe(5)
  })

  it('clears account selection across pages and resets filtered queries to page one', async () => {
    listAccounts.mockResolvedValue({ ...accountResult, total: 41 })
    const wrapper = mount(SupplierAccountsPanel, { props: { supplierId: 7 }, global })
    await flushPromises()
    await wrapper.get('input[aria-label="Pending Account"]').setValue(true)
    wrapper.getComponent(Pagination).vm.$emit('update:page', 2)
    await flushPromises()
    expect(listAccounts).toHaveBeenLastCalledWith(7, 2, 20, expect.any(Object))
    expect((wrapper.get('input[aria-label="Pending Account"]').element as HTMLInputElement).checked).toBe(false)
    await wrapper.get('select[aria-label="supplier.accounts.reviewStatus"]').setValue('approved')
    await flushPromises()
    expect(listAccounts).toHaveBeenLastCalledWith(7, 1, 20, expect.objectContaining({ review_status: 'approved' }))
    await wrapper.get('input[aria-label="common.search"]').setValue('external-41')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(listAccounts).toHaveBeenLastCalledWith(7, 1, 20, expect.objectContaining({ search: 'external-41' }))
    wrapper.getComponent(Pagination).vm.$emit('update:pageSize', 50)
    await flushPromises()
    expect(listAccounts).toHaveBeenLastCalledWith(7, 1, 50, expect.objectContaining({ search: 'external-41' }))
  })

  it('returns to a valid accounts page when approval empties the last pending page', async () => {
    listAccounts.mockResolvedValue({ ...accountResult, total: 21 })
    const wrapper = mount(SupplierAccountsPanel, { props: { supplierId: 7 }, global })
    await flushPromises()
    wrapper.getComponent(Pagination).vm.$emit('update:page', 2)
    await flushPromises()
    await wrapper.get('input[aria-label="Pending Account"]').setValue(true)
    await wrapper.findAll('button').find(button => button.text() === 'supplier.admin.approve')!.trigger('click')
    await flushPromises()
    await wrapper.get('input[value="3"]').setValue(true)
    listAccounts.mockResolvedValue({ ...accountResult, total: 20 })
    await wrapper.get('#supplier-review-form').trigger('submit')
    await flushPromises()
    expect(listAccounts).toHaveBeenLastCalledWith(7, 1, 20, expect.objectContaining({ review_status: 'pending' }))
    expect(wrapper.getComponent(Pagination).props('page')).toBe(1)
  })

  it('ignores late account responses after a newer filter query', async () => {
    let resolveOld!: (value: typeof accountResult) => void
    listAccounts.mockImplementationOnce(() => new Promise(resolve => { resolveOld = resolve }))
    const wrapper = mount(SupplierAccountsPanel, { props: { supplierId: 7 }, global })
    await wrapper.get('select[aria-label="supplier.accounts.reviewStatus"]').setValue('approved')
    await flushPromises()
    resolveOld({ ...accountResult, items: [{ ...accountResult.items[0], name: 'Stale Account' }] })
    await flushPromises()
    expect(wrapper.text()).toContain('Pending Account')
    expect(wrapper.text()).not.toContain('Stale Account')
  })

  it('shows a retry state on failure and loads the requested members page on retry', async () => {
    listMembers.mockRejectedValueOnce(new Error('Member request failed'))
    const wrapper = mount(SupplierMembersPanel, { props: { supplierId: 7 }, global })
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('Member request failed')
    expect(wrapper.text()).not.toContain('common.noData')
    await wrapper.get('[role="alert"] button').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('member@example.test')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('supports keyboard tab navigation and separates settings from the account table', async () => {
    const wrapper = mount(SupplierDetail, { props: { supplier: { ...supplier, stats: { ...supplier.stats, member_count: 101 } } }, global })
    await flushPromises()
    expect(wrapper.get('#supplier-tab-members').text()).toContain('101')
    await wrapper.get('#supplier-tab-accounts').trigger('keydown', { key: 'End' })
    expect(wrapper.get('#supplier-tab-settings').attributes('aria-selected')).toBe('true')
    expect(wrapper.findComponent(SupplierAccountsPanel).exists()).toBe(false)
    expect(wrapper.findComponent(SupplierMembersPanel).exists()).toBe(false)
    await wrapper.get('#supplier-tab-settings').trigger('keydown', { key: 'ArrowLeft' })
    await flushPromises()
    expect(wrapper.get('#supplier-tab-members').attributes('aria-selected')).toBe('true')
    expect(wrapper.findComponent(SupplierMembersPanel).exists()).toBe(true)
  })
})
