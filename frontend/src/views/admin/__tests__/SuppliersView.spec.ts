import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import SuppliersView from '../SuppliersView.vue'

const list = vi.fn().mockResolvedValue({
  items: [{
    id: 7, code: 'vendor-a', name: 'Vendor A', status: 'active', notes: null,
    allowed_account_kinds: [{ platform: 'openai', type: 'apikey' }],
    access_token: { exists: false, masked_key: null, created_at: null, last_used_at: null },
    stats: { member_count: 1, account_count: 2, pending_count: 1, schedulable_count: 0, error_count: 0 },
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  }], total: 1, page: 1, page_size: 20, pages: 1,
})

vi.mock('@/api/admin/suppliers', () => ({ default: {
  list: (...args: unknown[]) => list(...args), get: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn(),
} }))

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} }, missingWarn: false, fallbackWarn: false })

describe('SuppliersView', () => {
  it('lists suppliers and opens a selected tenant detail', async () => {
    const wrapper = mount(SuppliersView, {
      global: {
        plugins: [createPinia(), i18n],
        stubs: {
          AppLayout: { template: '<main><slot/></main>' }, Icon: true, Pagination: true,
          SupplierFormDialog: true, SupplierDetail: { props: ['supplier'], template: '<div data-testid="supplier-detail">{{ supplier.code }}</div>' },
        },
      },
    })
    await flushPromises()

    expect(list).toHaveBeenCalledWith(1, 20, { search: '', status: '' })
    expect(wrapper.text()).toContain('Vendor A')
    await wrapper.findAll('button').find((button) => button.text().includes('Vendor A'))!.trigger('click')
    expect(wrapper.get('[data-testid="supplier-detail"]').text()).toContain('vendor-a')
  })
})
