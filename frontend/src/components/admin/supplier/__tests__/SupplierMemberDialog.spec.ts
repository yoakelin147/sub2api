import { createI18n } from 'vue-i18n'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SupplierMemberDialog from '../SupplierMemberDialog.vue'

const i18n = createI18n({ legacy: false, locale: 'zh', messages: { zh: {} }, missingWarn: false, fallbackWarn: false })
const BaseDialogStub = {
  props: ['show', 'title'],
  template: '<section v-if="show"><slot/><footer><slot name="footer"/></footer></section>',
}

function mountDialog() {
  return mount(SupplierMemberDialog, {
    props: { show: true },
    global: { plugins: [i18n], stubs: { BaseDialog: BaseDialogStub, Icon: true } },
  })
}

describe('SupplierMemberDialog', () => {
  it('submits the fixed supplier-member create contract', async () => {
    const wrapper = mountDialog()

    await wrapper.get('[data-test="member-email"]').setValue('supplier@example.test')
    await wrapper.get('[data-test="member-password"]').setValue('StrongPassword!123')
    await wrapper.get('[data-test="member-username"]').setValue('supplier-one')
    await wrapper.get('[data-test="member-concurrency"]').setValue(3)
    await wrapper.get('form').trigger('submit')

    expect(wrapper.emitted('submit')?.[0]?.[0]).toEqual({
      email: 'supplier@example.test',
      password: 'StrongPassword!123',
      username: 'supplier-one',
      concurrency: 3,
    })
  })

  it('binds an existing user by id without sending create fields', async () => {
    const wrapper = mountDialog()

    await wrapper.get('[data-test="mode-bind"]').setValue()
    await wrapper.get('[data-test="member-user-id"]').setValue(42)
    await wrapper.get('form').trigger('submit')

    expect(wrapper.emitted('submit')?.[0]?.[0]).toEqual({ user_id: 42 })
  })
})
