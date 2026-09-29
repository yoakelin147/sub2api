import { createI18n } from 'vue-i18n'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SupplierBatchImportDialog from '../SupplierBatchImportDialog.vue'
import { parseSupplierAccountImport } from '@/features/supplier/accountImport'
import zhSupplier from '@/i18n/locales/zh/supplier'

const i18n = createI18n({ legacy: false, locale: 'zh', messages: { zh: zhSupplier } })
const BaseDialogStub = {
  props: ['show', 'title'],
  template: '<section v-if="show"><slot/><slot name="footer"/></section>',
}

describe('SupplierBatchImportDialog', () => {
  it('shows parseable examples and required-field guidance for each format', async () => {
    const wrapper = mount(SupplierBatchImportDialog, {
      props: { show: true, proxies: [{ id: 3, name: '库存代理' }] },
      global: { plugins: [i18n], stubs: { BaseDialog: BaseDialogStub } },
    })
    const textarea = wrapper.get('textarea')
    expect(wrapper.text()).toContain('supplier.batch.inputGuide')
    expect(zhSupplier.supplier.batch.inputGuide).toContain('填写获准的平台与账号类型')
    expect(zhSupplier.supplier.batch.formatHintJson).toContain('必填 name、platform、type、credentials 对象')
    expect(textarea.attributes('placeholder')).toBe(zhSupplier.supplier.batch.exampleJson)
    expect(parseSupplierAccountImport('json', textarea.attributes('placeholder')!)).toHaveLength(1)

    await wrapper.get('input[type="radio"][value="csv"]').setValue()
    expect(textarea.attributes('placeholder')).toBe(zhSupplier.supplier.batch.exampleCsv)
    expect(zhSupplier.supplier.batch.formatHintCsv).toContain('credential.api_key 列')
    expect(parseSupplierAccountImport('csv', zhSupplier.supplier.batch.exampleCsv)).toHaveLength(1)

    await wrapper.get('input[type="radio"][value="text"]').setValue()
    expect(textarea.attributes('placeholder')).toBe(zhSupplier.supplier.batch.exampleText)
    expect(zhSupplier.supplier.batch.formatHintText).toContain('external_id 可留空')
    expect(parseSupplierAccountImport('text', zhSupplier.supplier.batch.exampleText)).toHaveLength(1)
    wrapper.unmount()
  })
})
