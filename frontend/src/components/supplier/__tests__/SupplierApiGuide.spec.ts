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
    expect(body).toEqual({ external_id: 'ext-10001', name: 'Account 10001', platform: 'antigravity', type: 'apikey', credentials: { api_key: 'REPLACE_WITH_API_KEY', base_url: 'https://YOUR_ALLOWED_UPSTREAM_HOST' } })
    expect(command).not.toMatch(/proxy_id|supplier_id|group_ids|review_status/)
    wrapper.unmount()
  })

  it('provides PowerShell batch JSON with separate external IDs and UTF-8 encoding', async () => {
    const wrapper = mount(SupplierApiGuide, options)
    await wrapper.get('[data-test="guide-operation"]').setValue('batch')
    await wrapper.get('[data-test="guide-language"]').setValue('powershell')
    const command = wrapper.get('pre').text()
    expect(command).toContain('/supplier/accounts/batch')
    expect(command).toContain('UTF8.GetBytes($body)')
    const body = JSON.parse(command.split("$body = @'\n")[1].split("\n'@")[0])
    expect(body.accounts.map((account: { external_id: string }) => account.external_id)).toEqual(['ext-10001', 'ext-10002'])
    wrapper.unmount()
  })
})
