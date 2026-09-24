import { createI18n } from 'vue-i18n'
import { mount } from '@vue/test-utils'
import { flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import SupplierFormDialog from '../SupplierFormDialog.vue'
import { SUPPLIER_ACCOUNT_KINDS } from '@/features/supplier/accountKinds'
import type { AdminSupplier } from '@/api/admin/suppliers'
import { getAll as getAllGroups } from '@/api/admin/groups'

vi.mock('@/api/admin/groups', () => ({ getAll: vi.fn().mockResolvedValue([]) }))

const i18n = createI18n({ legacy: false, locale: 'zh', messages: { zh: {} }, missingWarn: false, fallbackWarn: false })
const BaseDialogStub = {
  props: ['show', 'title'],
  template: '<section v-if="show"><slot/><footer><slot name="footer"/></footer></section>',
}

function mountDialog(supplier?: AdminSupplier) {
  return mount(SupplierFormDialog, {
    props: { show: true, supplier },
    global: { plugins: [i18n], stubs: { BaseDialog: BaseDialogStub } },
  })
}

describe('SupplierFormDialog account kind hierarchy', () => {
  it('keeps editing possible with no allowed kinds and authorizes multiple groups per platform', async () => {
    vi.mocked(getAllGroups).mockResolvedValueOnce([
      { id: 3, name: 'First', platform: 'openai', status: 'active' },
      { id: 4, name: 'Second', platform: 'openai', status: 'active' },
    ] as never)
    const wrapper = mountDialog({
      code: 'supplier-existing', name: 'Existing', status: 'active', notes: null,
      allowed_account_kinds: [], review_required: false, auto_approve_groups: {},
    } as AdminSupplier)
    await flushPromises()
    await wrapper.get('[data-test="kind-openai:apikey"]').setValue(true)
    const groupInputs = wrapper.findAll('input[type="checkbox"][value="3"], input[type="checkbox"][value="4"]')
    expect(groupInputs).toHaveLength(2)
    await groupInputs[0].setValue(true)
    await groupInputs[1].setValue(true)
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')?.[0]?.[0]).toMatchObject({ auto_approve_groups: { openai: [3, 4] } })
  })
  it('saves an unchecked review policy without groups while types are selected', async () => {
    const wrapper = mountDialog({
      code: 'supplier-existing', name: 'Existing', status: 'active', notes: null,
      allowed_account_kinds: SUPPLIER_ACCOUNT_KINDS, review_required: false, auto_approve_groups: {},
    } as AdminSupplier)
    await wrapper.get('form').trigger('submit')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.emitted('submit')?.[0]?.[0]).toMatchObject({
      review_required: false, allowed_account_kinds: SUPPLIER_ACCOUNT_KINDS, auto_approve_groups: {},
    })
  })

  it('keeps long group lists collapsed and selects a whole platform independently', async () => {
    vi.mocked(getAllGroups).mockResolvedValueOnce([
      ...Array.from({ length: 12 }, (_, index) => ({ id: index + 1, name: `OpenAI ${index + 1}`, platform: 'openai', status: 'active' })),
      { id: 30, name: 'Anthropic', platform: 'anthropic', status: 'active' },
    ] as never)
    const wrapper = mountDialog({
      code: 'supplier-existing', name: 'Existing', status: 'active', notes: null,
      allowed_account_kinds: SUPPLIER_ACCOUNT_KINDS, review_required: false, auto_approve_groups: {},
    } as AdminSupplier)
    await flushPromises()
    const openAI = wrapper.get('[data-test="group-platform-openai"]')
    const anthropic = wrapper.get('[data-test="group-platform-anthropic"]')
    expect((openAI.element.closest('details') as HTMLDetailsElement).open).toBe(false)
    expect(openAI.text()).toContain('0 / 12')
    await openAI.get('input[type="checkbox"]').setValue(true)
    expect(openAI.text()).toContain('12 / 12')
    expect(anthropic.text()).toContain('0 / 1')
    await openAI.get('summary').trigger('click')
    expect((openAI.element.closest('details') as HTMLDetailsElement).open).toBe(true)
    await wrapper.get('[data-test="group-openai-1"]').setValue(false)
    await wrapper.get('form').trigger('submit')
    const payload = wrapper.emitted('submit')?.[0]?.[0] as { auto_approve_groups: Record<string, number[]> }
    expect(payload.auto_approve_groups.openai).toEqual(Array.from({ length: 11 }, (_, index) => index + 2))
    expect(payload.auto_approve_groups.anthropic).toBeUndefined()
  })
  it('stacks the two platform columns independently when one platform expands', async () => {
    const wrapper = mountDialog()
    const [left, right] = wrapper.findAll('[data-test="kind-column"]')

    expect(left.find('[data-test="platform-openai"]').exists()).toBe(true)
    expect(right.find('[data-test="platform-anthropic"]').exists()).toBe(true)
    expect(left.find('[data-test="platform-gemini"]').exists()).toBe(true)

    await left.get('summary').trigger('click')
    expect((left.get('details').element as HTMLDetailsElement).open).toBe(true)
    expect((right.get('details').element as HTMLDetailsElement).open).toBe(false)
    expect(left.findAll('details')).toHaveLength(5)
    expect(right.findAll('details')).toHaveLength(5)
  })

  it('starts collapsed and reveals child types when the platform is opened', async () => {
    const wrapper = mountDialog()
    const openAI = wrapper.get('[data-test="platform-openai"]').element.closest('details') as HTMLDetailsElement

    expect(openAI.open).toBe(false)
    await wrapper.get('[data-test="platform-openai"]').element.closest('summary')?.click()
    expect(openAI.open).toBe(true)
    await wrapper.get('summary').trigger('click')
    expect(openAI.open).toBe(false)

    await wrapper.get('[data-test="kind-openai:apikey"]').setValue(true)
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')?.[0]?.[0]).toMatchObject({
      allowed_account_kinds: [{ platform: 'openai', type: 'apikey' }],
    })
    expect(wrapper.emitted('submit')?.[0]?.[0]).not.toHaveProperty('code')
  })

  it('preserves the generated code when editing', async () => {
    const wrapper = mountDialog({
      code: 'supplier-existing', name: 'Existing', status: 'active', notes: null,
      allowed_account_kinds: [], review_required: true, auto_approve_groups: {},
    } as AdminSupplier)

    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')?.[0]?.[0]).toMatchObject({ code: 'supplier-existing' })
  })

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
    const openAIDetails = openAIParent.element.closest('details') as HTMLDetailsElement
    const anthropicChild = wrapper.get('[data-test="kind-anthropic:apikey"]')

    await anthropicChild.setValue(true)
    await openAIParent.setValue(true)
    expect(openAIDetails.open).toBe(false)
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
