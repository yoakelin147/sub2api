import { createI18n } from 'vue-i18n'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SupplierAccountForm from '../SupplierAccountForm.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
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
    await wrapper.get('#supplier-credentials').setValue('{"api_key":"sk-test"}')
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

    expect(wrapper.get('#supplier-credentials').element).toHaveProperty('value', '')
    await wrapper.get('form').trigger('submit')
    const [, payload] = wrapper.emitted('submit')?.[0] ?? []
    expect(payload).not.toHaveProperty('credentials')
  })
})
