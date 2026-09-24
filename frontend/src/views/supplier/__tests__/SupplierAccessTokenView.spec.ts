import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SupplierAccessTokenView from '../SupplierAccessTokenView.vue'

const getStatus = vi.fn()
const getProfile = vi.fn()
const regenerate = vi.fn()
const revoke = vi.fn()

vi.mock('@/api/supplier', () => ({
  getAccessTokenStatus: (...args: unknown[]) => getStatus(...args),
  getProfile: (...args: unknown[]) => getProfile(...args),
  regenerateAccessToken: (...args: unknown[]) => regenerate(...args),
  revokeAccessToken: (...args: unknown[]) => revoke(...args),
}))

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: { en: { common: { notAvailable: 'N/A', copy: 'Copy', unknownError: 'Error' }, supplier: { token: {
    title: 'Token', description: 'Description', exists: 'Active', missing: 'Missing', masked: 'Masked', createdAt: 'Created',
    lastUsedAt: 'Used', generate: 'Generate token', rotate: 'Rotate', revoke: 'Revoke', oneTimeTitle: 'Copy now',
    oneTimeHint: 'Only once', rotated: 'Rotated', revoked: 'Revoked', confirmRotate: 'Rotate?', confirmRevoke: 'Revoke?', loadFailed: 'Failed',
  } } } },
})

describe('SupplierAccessTokenView', () => {
  beforeEach(() => {
    localStorage.clear()
    getStatus.mockResolvedValue({ exists: false, masked_key: null, created_at: null, last_used_at: null })
    getProfile.mockResolvedValue({ review_required: true, auto_approve_groups: {} })
    regenerate.mockResolvedValue({ key: 'supplier_selector_secret' })
    revoke.mockResolvedValue(undefined)
  })

  it('shows the generated token once without persisting it', async () => {
    const wrapper = mount(SupplierAccessTokenView, {
      global: { plugins: [createPinia(), i18n], stubs: { AppLayout: { template: '<main><slot/></main>' }, Icon: true, SupplierApiGuide: true } },
    })
    await flushPromises()
    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="plaintext-token"]').text()).toBe('supplier_selector_secret')
    expect([...Array(localStorage.length)].map((_, index) => localStorage.key(index))).not.toContain('supplier_access_token')
    expect([...Array(localStorage.length)].map((_, index) => localStorage.getItem(localStorage.key(index)!))).not.toContain('supplier_selector_secret')
  })
})
