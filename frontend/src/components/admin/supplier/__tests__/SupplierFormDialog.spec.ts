import { createI18n } from 'vue-i18n'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SupplierFormDialog from '../SupplierFormDialog.vue'
import { SUPPLIER_ACCOUNT_KINDS } from '@/features/supplier/accountKinds'

const i18n = createI18n({ legacy: false, locale: 'zh', messages: { zh: {} }, missingWarn: false, fallbackWarn: false })
const BaseDialogStub = {
  props: ['show', 'title'],
  template: '<section v-if="show"><slot/><footer><slot name="footer"/></footer></section>',
}

function mountDialog() {
  return mount(SupplierFormDialog, {
    props: { show: true },
    global: { plugins: [i18n], stubs: { BaseDialog: BaseDialogStub } },
  })
}

describe('SupplierFormDialog account kind hierarchy', () => {
  it('selects every supported child with the global checkbox', async () => {
    const wrapper = mountDialog()

    await wrapper.get('[data-test="select-all-kinds"]').setValue(true)
    await wrapper.get('form').trigger('submit')

    const payload = wrapper.emitted('submit')?.[0]?.[0] as { allowed_account_kinds: unknown[] }
    expect(payload.allowed_account_kinds).toEqual(SUPPLIER_ACCOUNT_KINDS)
  })

  it('selects and clears one platform without changing sibling platforms', async () => {
    const wrapper = mountDialog()
    const openAIParent = wrapper.get('[data-test="platform-openai"]')
    const anthropicChild = wrapper.get('[data-test="kind-anthropic:apikey"]')

    await anthropicChild.setValue(true)
    await openAIParent.setValue(true)
    await wrapper.get('form').trigger('submit')

    let payload = wrapper.emitted('submit')?.at(-1)?.[0] as { allowed_account_kinds: Array<{ platform: string; type: string }> }
    expect(payload.allowed_account_kinds.filter((kind) => kind.platform === 'openai')).toHaveLength(4)
    expect(payload.allowed_account_kinds).toContainEqual({ platform: 'anthropic', type: 'apikey' })

    await openAIParent.setValue(false)
    await wrapper.get('form').trigger('submit')
    payload = wrapper.emitted('submit')?.at(-1)?.[0] as { allowed_account_kinds: Array<{ platform: string; type: string }> }
    expect(payload.allowed_account_kinds.filter((kind) => kind.platform === 'openai')).toHaveLength(0)
    expect(payload.allowed_account_kinds).toEqual([{ platform: 'anthropic', type: 'apikey' }])
  })
})
