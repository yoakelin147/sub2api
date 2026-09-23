import { createI18n } from 'vue-i18n'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { createPinia } from 'pinia'
import OAuthAuthorizationFlow from '@/components/account/OAuthAuthorizationFlow.vue'
import SupplierAccountForm from '../SupplierAccountForm.vue'

const { generateSupplierOAuthURL, exchangeSupplierOAuthCode, exchangeSupplierOAuthCredential, getSupplierGrokOAuthCapabilities } = vi.hoisted(() => ({
  generateSupplierOAuthURL: vi.fn(),
  exchangeSupplierOAuthCode: vi.fn(),
  exchangeSupplierOAuthCredential: vi.fn(),
  getSupplierGrokOAuthCapabilities: vi.fn(),
}))
vi.mock('@/api/supplier', () => ({ generateSupplierOAuthURL, exchangeSupplierOAuthCode, exchangeSupplierOAuthCredential, getSupplierGrokOAuthCapabilities }))

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
  it('lets the supplier authorize OpenAI and fills tokens without bypassing login credentials', async () => {
    generateSupplierOAuthURL.mockResolvedValue({ auth_url: 'https://auth.example.test/authorize?state=expected', session_id: 'owned-session' })
    exchangeSupplierOAuthCode.mockResolvedValue({ access_token: 'access-from-oauth', refresh_token: 'refresh-from-oauth', email: 'supplier@example.test' })
    const wrapper = mount(SupplierAccountForm, {
      props: { show: true, kinds: [{ platform: 'openai', type: 'oauth' }], proxies: [{ id: 3, name: 'Platform proxy' }] },
      global: { plugins: [createPinia(), i18n], stubs: { BaseDialog: BaseDialogStub } },
    })
    await wrapper.get('[data-testid="oauth-generate-url"]').trigger('click')
    await flushPromises()
    expect(generateSupplierOAuthURL).toHaveBeenCalledWith('openai', { proxy_id: 3, type: 'oauth' })
    expect(wrapper.get('input[readonly]').element).toHaveProperty('value', 'https://auth.example.test/authorize?state=expected')
    expect(wrapper.get('[data-testid="oauth-open-url"]').attributes()).toMatchObject({ href: 'https://auth.example.test/authorize?state=expected', target: '_blank', rel: 'noopener noreferrer' })
    await wrapper.get('[data-testid="oauth-auth-code"]').setValue('http://localhost:1455/auth/callback?code=oauth-code&state=expected')
    await wrapper.get('[data-testid="supplier-oauth-complete"]').trigger('click')
    await flushPromises()
    expect(exchangeSupplierOAuthCode).toHaveBeenCalledWith('openai', 'oauth', 'owned-session', 'oauth-code', 'expected')
    expect(wrapper.get('[data-field="access_token"]').element).toHaveProperty('value', 'access-from-oauth')
    await wrapper.get('input[required]').setValue('My OpenAI account')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')).toBeUndefined()
    await wrapper.get('[data-field="password"]').setValue('web-secret')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')?.[0]?.[1]).toMatchObject({
      credentials: { access_token: 'access-from-oauth', refresh_token: 'refresh-from-oauth', email: 'supplier@example.test', password: 'web-secret' },
    })
  })

  it.each([
    ['anthropic', 'oauth'], ['anthropic', 'setup-token'], ['gemini', 'oauth'], ['antigravity', 'oauth'], ['grok', 'oauth'],
  ])('lets a supplier complete %s %s authorization using its selected kind', async (platform, type) => {
    generateSupplierOAuthURL.mockResolvedValue({ auth_url: 'https://auth.example.test/?state=expected', session_id: 'owned-session' })
    exchangeSupplierOAuthCode.mockResolvedValue({ access_token: 'issued-token', refresh_token: 'issued-refresh' })
    const wrapper = mount(SupplierAccountForm, {
      props: { show: true, kinds: [{ platform, type }], proxies: [{ id: 3, name: 'Proxy' }] },
      global: { plugins: [createPinia(), i18n], stubs: { BaseDialog: BaseDialogStub } },
    })
    await wrapper.get('[data-testid="oauth-generate-url"]').trigger('click')
    await flushPromises()
    expect(generateSupplierOAuthURL).toHaveBeenCalledWith(platform, {
      proxy_id: 3, type, ...(platform === 'gemini' ? { oauth_type: 'code_assist', project_id: '', tier_id: '' } : {}),
    })
    await wrapper.get('[data-testid="oauth-auth-code"]').setValue(platform === 'anthropic' ? 'issued-code' : 'http://localhost/callback?code=issued-code&state=expected')
    await wrapper.get('[data-testid="supplier-oauth-complete"]').trigger('click')
    await flushPromises()
    expect(exchangeSupplierOAuthCode).toHaveBeenCalledWith(platform, type, 'owned-session', 'issued-code', 'expected')
    expect(wrapper.get('[data-field="access_token"]').element).toHaveProperty('value', 'issued-token')
    wrapper.unmount()
  })

  it('uses the shared Claude Cookie option and a supplier-scoped exchange', async () => {
    exchangeSupplierOAuthCredential.mockResolvedValueOnce({ access_token: 'cookie-token' })
    const wrapper = mount(SupplierAccountForm, {
      props: { show: true, kinds: [{ platform: 'anthropic', type: 'setup-token' }], proxies: [{ id: 3, name: 'Proxy' }] },
      global: { plugins: [createPinia(), i18n], stubs: { BaseDialog: BaseDialogStub } },
    })
    expect(wrapper.get('input[value="cookie"]').exists()).toBe(true)
    wrapper.getComponent(OAuthAuthorizationFlow).vm.$emit('cookie-auth', 'session-secret')
    await flushPromises()
    expect(exchangeSupplierOAuthCredential).toHaveBeenCalledWith('anthropic', { type: 'setup-token', proxy_id: 3, method: 'cookie', session_key: 'session-secret' })
    expect(wrapper.get('[data-field="access_token"]').element).toHaveProperty('value', 'cookie-token')
    wrapper.unmount()
  })

  it('only offers Grok password authorization when the supplier capability enables it', async () => {
    getSupplierGrokOAuthCapabilities.mockResolvedValueOnce({ password_auth_enabled: false }).mockResolvedValueOnce({ password_auth_enabled: true })
    const props = { show: true, kinds: [{ platform: 'grok', type: 'oauth' }], proxies: [{ id: 3, name: 'Proxy' }] }
    const global = { plugins: [createPinia(), i18n], stubs: { BaseDialog: BaseDialogStub } }
    const disabled = mount(SupplierAccountForm, { props, global })
    await flushPromises()
    expect(disabled.find('input[value="email_password"]').exists()).toBe(false)
    disabled.unmount()
    const enabled = mount(SupplierAccountForm, { props, global })
    await flushPromises()
    expect(enabled.get('input[value="email_password"]').exists()).toBe(true)
    enabled.unmount()
  })

  it('requires web login email and password for OAuth uploads', async () => {
    const wrapper = mount(SupplierAccountForm, {
      props: { show: true, kinds: [{ platform: 'openai', type: 'oauth' }], proxies: [{ id: 3, name: 'Platform proxy' }] },
      global: { plugins: [createPinia(), i18n], stubs: { BaseDialog: BaseDialogStub } },
    })
    await wrapper.get('input[required]').setValue('OAuth Account')
    await wrapper.get('[data-field="access_token"]').setValue('access')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')).toBeUndefined()
    await wrapper.get('[data-field="email"]').setValue('login@example.test')
    await wrapper.get('[data-field="password"]').setValue('web-secret')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')?.[0]?.[1]).toMatchObject({ credentials: { access_token: 'access', email: 'login@example.test', password: 'web-secret' } })
  })
  it('emits only the supplier-safe create contract', async () => {
    const wrapper = mount(SupplierAccountForm, {
      props: { show: true, kinds: [{ platform: 'openai', type: 'apikey' }], proxies: [{ id: 3, name: 'Platform proxy' }] },
      global: { plugins: [createPinia(), i18n], stubs: { BaseDialog: BaseDialogStub } },
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
      proxy_id: 3,
      concurrency: 1,
      priority: 50,
      auto_pause_on_expired: true,
    })
    expect(JSON.stringify(payload)).not.toMatch(/supplier_id|group_ids|rate_multiplier|schedulable|review_status|extra/)
  })

  it('does not require or refill credentials while editing', async () => {
    const wrapper = mount(SupplierAccountForm, {
      props: {
        show: true,
        kinds: [{ platform: 'openai', type: 'apikey' }],
        proxies: [{ id: 3, name: 'Platform proxy' }],
        account: {
          id: 9, external_id: 'ext-9', name: 'Existing', notes: null, platform: 'openai', type: 'apikey',
          credential_status: { api_key: true }, has_credentials: true, status: 'disabled', schedulable: false,
          review_status: 'pending', review_note: null, expires_at: null, last_used_at: null,
          proxy_id: 3, concurrency: 1, priority: 50, load_factor: null, auto_pause_on_expired: true,
          created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
        },
      },
      global: { plugins: [createPinia(), i18n], stubs: { BaseDialog: BaseDialogStub } },
    })

    expect(wrapper.get('[data-field="api_key"]').element).toHaveProperty('value', '')
    await wrapper.get('form').trigger('submit')
    const [, payload] = wrapper.emitted('submit')?.[0] ?? []
    expect(payload).not.toHaveProperty('credentials')
  })

  it('switches to Antigravity fields and requires a valid upstream URL before submission', async () => {
    const wrapper = mount(SupplierAccountForm, {
      props: { show: true, kinds: [{ platform: 'openai', type: 'apikey' }, { platform: 'antigravity', type: 'apikey' }], proxies: [{ id: 3, name: 'Platform proxy' }] },
      global: { plugins: [createPinia(), i18n], stubs: { BaseDialog: BaseDialogStub } },
    })
    await wrapper.findAll('select')[1].setValue('antigravity:apikey')
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
      props: { show: true, kinds: [{ platform: 'openai', type: 'apikey' }, { platform: 'antigravity', type: 'oauth' }], proxies: [{ id: 3, name: 'Platform proxy' }] },
      global: { plugins: [createPinia(), i18n], stubs: { BaseDialog: BaseDialogStub } },
    })
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')).toBeUndefined()
    await wrapper.get('[data-field="api_key"]').setValue('keep-this-value')
    const confirm = vi.spyOn(window, 'confirm').mockReturnValueOnce(false)
    await wrapper.findAll('select')[1].setValue('antigravity:oauth')
    expect(wrapper.findAll('select')[1].element).toHaveProperty('value', 'openai:apikey')
    expect(wrapper.get('[data-field="api_key"]').element).toHaveProperty('value', 'keep-this-value')
    confirm.mockRestore()
  })
})
