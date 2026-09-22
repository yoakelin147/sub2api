import { createI18n } from 'vue-i18n'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import SupplierAccountForm from '../SupplierAccountForm.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      common: { name: 'Name', cancel: 'Cancel', save: 'Save', saving: 'Saving' },
      supplier: { accounts: {
        create: 'Add Account', edit: 'Edit Account', externalId: 'External ID', platform: 'Platform', type: 'Type',
        expiresAt: 'Expires', notes: 'Notes', credentials: 'Credentials', credentialsHint: 'Write only',
        credentialTemplate: 'Template', invalidCredentials: 'Invalid credentials',
      } },
    },
  },
})

const BaseDialogStub = {
  props: ['show', 'title'],
  template: '<section v-if="show"><slot/><footer><slot name="footer"/></footer></section>',
}

describe('SupplierAccountForm', () => {
  it('emits only the supplier-safe create contract', async () => {
    const wrapper = mount(SupplierAccountForm, {
      props: { show: true, kinds: [{ platform: 'openai', type: 'apikey' }] },
      global: { plugins: [i18n], stubs: { BaseDialog: BaseDialogStub } },
    })

    await wrapper.get('input[required]').setValue('Vendor Account')
    await wrapper.get('[data-field="api_key"]').setValue('sk-test')
    await wrapper.get('form').trigger('submit')

    const [, payload] = wrapper.emitted('submit')?.[0] ?? []
    expect(payload).toEqual({
      name: 'Vendor Account',
      external_id: '',
      notes: null,
      platform: 'openai',
      type: 'apikey',
      credentials: { api_key: 'sk-test' },
      expires_at: null,
    })
    expect(JSON.stringify(payload)).not.toMatch(/supplier_id|group_ids|proxy_id|priority|concurrency|schedulable|review_status|extra/)
  })

  it('does not require or refill credentials while editing', async () => {
    const wrapper = mount(SupplierAccountForm, {
      props: {
        show: true,
        kinds: [{ platform: 'openai', type: 'apikey' }],
        account: {
          id: 9, external_id: 'ext-9', name: 'Existing', notes: null, platform: 'openai', type: 'apikey',
          credential_status: { api_key: true }, has_credentials: true, status: 'disabled', schedulable: false,
          review_status: 'pending', review_note: null, expires_at: null, last_used_at: null,
          created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
        },
      },
      global: { plugins: [i18n], stubs: { BaseDialog: BaseDialogStub } },
    })

    expect(wrapper.get('[data-field="api_key"]').element).toHaveProperty('value', '')
    await wrapper.get('form').trigger('submit')
    const [, payload] = wrapper.emitted('submit')?.[0] ?? []
    expect(payload).not.toHaveProperty('credentials')
  })

  it('switches to Antigravity fields and requires a valid upstream URL before submission', async () => {
    const wrapper = mount(SupplierAccountForm, {
      props: { show: true, kinds: [{ platform: 'openai', type: 'apikey' }, { platform: 'antigravity', type: 'apikey' }] },
      global: { plugins: [i18n], stubs: { BaseDialog: BaseDialogStub } },
    })
    await wrapper.get('select').setValue('antigravity:apikey')
    expect(wrapper.get('[data-field="base_url"]').element).toHaveProperty('value', 'https://')
    await wrapper.get('input[required]').setValue('Upstream account')
    await wrapper.get('[data-field="api_key"]').setValue('example-key')
    await wrapper.get('[data-field="base_url"]').setValue('')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')).toBeUndefined()
    expect(wrapper.get('[role="alert"]').text()).toContain('supplier.accounts.missingCredentials')
    await wrapper.get('[data-field="base_url"]').setValue('http://upstream.example.test')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')).toBeUndefined()
    await wrapper.get('[data-field="base_url"]').setValue('https://upstream.example.test')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')?.[0]?.[1]).toMatchObject({ platform: 'antigravity', type: 'apikey', credentials: { api_key: 'example-key', base_url: 'https://upstream.example.test' } })
  })

  it('preserves entered credentials when a type switch is cancelled and rejects empty credentials', async () => {
    const wrapper = mount(SupplierAccountForm, {
      props: { show: true, kinds: [{ platform: 'openai', type: 'apikey' }, { platform: 'antigravity', type: 'oauth' }] },
      global: { plugins: [i18n], stubs: { BaseDialog: BaseDialogStub } },
    })
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')).toBeUndefined()
    await wrapper.get('[data-field="api_key"]').setValue('keep-this-value')
    const confirm = vi.spyOn(window, 'confirm').mockReturnValueOnce(false)
    await wrapper.get('select').setValue('antigravity:oauth')
    expect(wrapper.get('select').element).toHaveProperty('value', 'openai:apikey')
    expect(wrapper.get('[data-field="api_key"]').element).toHaveProperty('value', 'keep-this-value')
    confirm.mockRestore()
  })
})
