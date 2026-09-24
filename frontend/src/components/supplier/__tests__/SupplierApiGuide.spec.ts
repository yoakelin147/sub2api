import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import SupplierApiGuide from '../SupplierApiGuide.vue'
import zhSupplier from '@/i18n/locales/zh/supplier'

vi.mock('@/api/url', () => ({ getAPIBaseURL: () => '/api/v1' }))
const i18n = createI18n({ legacy: false, locale: 'zh', messages: { zh: zhSupplier }, missingWarn: false, fallbackWarn: false })
const options = { global: { plugins: [createPinia(), i18n], stubs: { Icon: true, RouterLink: { template: '<a><slot/></a>' } } } }

describe('SupplierApiGuide', () => {
  it('documents allowed account options with platform scope and accepted values', () => {
    const wrapper = mount(SupplierApiGuide, { ...options, props: { profile: { review_required: false, auto_approve_groups: { openai: [42] } } as never } })
    const optionsText = wrapper.get('[data-test="supplier-extra-guide"]').text()
    expect(optionsText).toContain('openai_compact_mode')
    expect(optionsText).toContain('force_on')
    expect(optionsText).toContain('web_search_emulation')
    expect(optionsText).toContain('enabled')
    expect(optionsText).toContain('upstream_request_id_header')
    wrapper.unmount()
  })

  it('builds tenant API examples with full-token placeholders, required upstream fields and idempotency', async () => {
    const wrapper = mount(SupplierApiGuide, options)
    expect(wrapper.get('pre').text()).toContain('/supplier/me')
    await wrapper.get('[data-test="guide-operation"]').setValue('create')
    await wrapper.get('[data-test="guide-kind"]').setValue('antigravity:apikey')
    const command = wrapper.get('pre').text()
    expect(command).toContain('x-api-key: $SUPPLIER_TOKEN')
    expect(command).toContain('Idempotency-Key:')
    expect(command).toContain('REPLACE_WITH_FULL_SUPPLIER_TOKEN')
    const body = JSON.parse(command.split("--data-raw '")[1].slice(0, -1))
    expect(body).toEqual({ name: 'Account 10001', platform: 'antigravity', type: 'apikey', proxy_id: 123, credentials: { api_key: 'REPLACE_WITH_API_KEY', base_url: 'https://YOUR_ALLOWED_UPSTREAM_HOST' } })
    expect(command).not.toMatch(/supplier_id|group_ids|review_status/)
    wrapper.unmount()
  })

  it('provides PowerShell batch JSON with generated platform IDs and UTF-8 encoding', async () => {
    const wrapper = mount(SupplierApiGuide, options)
    await wrapper.get('[data-test="guide-operation"]').setValue('batch')
    await wrapper.get('[data-test="guide-language"]').setValue('powershell')
    const command = wrapper.get('pre').text()
    expect(command).toContain('/supplier/accounts/batch')
    expect(command).toContain('UTF8.GetBytes($body)')
    const body = JSON.parse(command.split("$body = @'\n")[1].split("\n'@")[0])
    expect(body.accounts.map((account: { name: string }) => account.name)).toEqual(['Account 10001', 'Account 10002'])
    expect(body.accounts.every((account: object) => !('external_id' in account))).toBe(true)
    wrapper.unmount()
  })

  it('shows supplier-owned OpenAI OAuth URL and token exchange examples without create-only headers', async () => {
    const wrapper = mount(SupplierApiGuide, options)
    await wrapper.get('[data-test="guide-operation"]').setValue('proxies')
    expect(wrapper.get('pre').text()).toContain('GET "$API_BASE/supplier/proxies"')
    await wrapper.get('[data-test="guide-operation"]').setValue('oauthUrl')
    const urlCommand = wrapper.get('pre').text()
    expect(urlCommand).toContain('POST "$API_BASE/supplier/oauth/openai/auth-url"')
    expect(JSON.parse(urlCommand.split("--data-raw '")[1].slice(0, -1))).toEqual({ proxy_id: 123, type: 'oauth' })
    expect(urlCommand).not.toContain('Idempotency-Key')
    await wrapper.get('[data-test="guide-operation"]').setValue('oauthExchange')
    await wrapper.get('[data-test="guide-language"]').setValue('powershell')
    const exchangeCommand = wrapper.get('pre').text()
    expect(exchangeCommand).toContain('/supplier/oauth/openai/exchange-code')
    expect(exchangeCommand).toContain('Invoke-RestMethod -Method Post')
    expect(exchangeCommand).not.toContain('Idempotency-Key')
    expect(JSON.parse(exchangeCommand.split("$body = @'\n")[1].split("\n'@")[0])).toEqual({
      type: 'oauth', session_id: 'REPLACE_WITH_SESSION_ID', code: 'REPLACE_WITH_AUTH_CODE', state: 'REPLACE_WITH_STATE',
    })
    wrapper.unmount()
  })

  it('shows Claude Setup Token and Gemini authorization options using their own endpoints', async () => {
    const wrapper = mount(SupplierApiGuide, options)
    await wrapper.get('[data-test="guide-operation"]').setValue('oauthUrl')
    await wrapper.get('[data-test="guide-kind"]').setValue('anthropic:setup-token')
    expect(wrapper.get('pre').text()).toContain('/supplier/oauth/anthropic/auth-url')
    expect(wrapper.get('pre').text()).toContain('"type": "setup-token"')
    await wrapper.get('[data-test="guide-kind"]').setValue('gemini:oauth')
    expect(wrapper.get('pre').text()).toContain('/supplier/oauth/gemini/auth-url')
    expect(wrapper.get('pre').text()).toContain('"oauth_type": "code_assist"')
    wrapper.unmount()
  })

  it('shows supplier-scoped Cookie, SSO and password exchange examples', async () => {
    const wrapper = mount(SupplierApiGuide, options)
    await wrapper.get('[data-test="guide-operation"]').setValue('oauthCredential')
    expect(wrapper.get('pre').text()).toContain('/supplier/oauth/anthropic/credential-exchange')
    expect(wrapper.get('pre').text()).toContain('REPLACE_WITH_SESSION_KEY')
    await wrapper.get('[data-test="guide-kind"]').setValue('grok:oauth')
    expect(wrapper.get('pre').text()).toContain('REPLACE_WITH_SSO_TOKEN')
    await wrapper.get('[data-test="guide-credential-method"]').setValue('password')
    expect(wrapper.get('pre').text()).toContain('REPLACE_WITH_PASSWORD')
    expect(wrapper.get('pre').text()).not.toContain('Idempotency-Key')
    wrapper.unmount()
  })

  it('requires login credentials in OAuth examples but does not demand an optional refresh token', async () => {
    const wrapper = mount(SupplierApiGuide, options)
    await wrapper.get('[data-test="guide-operation"]').setValue('create')
    await wrapper.get('[data-test="guide-kind"]').setValue('openai:oauth')
    const command = wrapper.get('pre').text()
    const body = JSON.parse(command.split("--data-raw '")[1].slice(0, -1))
    expect(body.credentials).toEqual({ email: 'REPLACE_WITH_EMAIL', password: 'REPLACE_WITH_PASSWORD', access_token: 'REPLACE_WITH_ACCESS_TOKEN' })
    expect(body).not.toHaveProperty('external_id')
    wrapper.unmount()
  })

  it('shows only current permissions and uses PUT without create-only headers', async () => {
    const wrapper = mount(SupplierApiGuide, { ...options, props: { profile: { review_required: false, auto_approve_groups: { openai: [42, 43] } } as never } })
    await wrapper.get('[data-test="guide-operation"]').setValue('create')
    const create = wrapper.get('pre').text()
    const body = JSON.parse(create.split("--data-raw '")[1].slice(0, -1))
    expect(body.group_ids).toEqual([42, 43])
    expect(body.credentials.model_mapping).toEqual({ YOUR_MODEL: 'UPSTREAM_MODEL' })
    expect(body.extra).toEqual({ openai_compact_mode: 'auto' })
    expect(body).not.toHaveProperty('rate_multiplier')
    await wrapper.get('[data-test="guide-operation"]').setValue('update')
    const update = wrapper.get('pre').text()
    expect(update).toContain(' -X PUT ')
    expect(update).not.toContain('Idempotency-Key')
    expect(JSON.parse(update.split("--data-raw '")[1].slice(0, -1)).group_ids).toEqual([42, 43])
    expect(JSON.parse(update.split("--data-raw '")[1].slice(0, -1)).credentials).toEqual({ model_mapping: { YOUR_MODEL: 'UPSTREAM_MODEL' } })
    await wrapper.get('[data-test="guide-language"]').setValue('powershell')
    expect(wrapper.get('pre').text()).toContain('Invoke-RestMethod -Method Put')
    wrapper.unmount()
  })
})
