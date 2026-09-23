<template>
  <BaseDialog :show="show" :title="supplier ? t('supplier.admin.edit') : t('supplier.admin.create')" width="wide" @close="emit('close')">
    <form id="supplier-form" class="space-y-5" @submit.prevent="submit">
      <label class="block">
        <span class="input-label">{{ t('common.name') }}</span>
        <input v-model.trim="name" class="input" required maxlength="120" />
      </label>
      <label>
        <span class="input-label">{{ t('common.status') }}</span>
        <select v-model="status" class="input">
          <option value="active">{{ t('common.active') }}</option>
          <option value="disabled">{{ t('common.disabled') }}</option>
        </select>
      </label>
      <label>
        <span class="input-label">{{ t('supplier.accounts.notes') }}</span>
        <textarea v-model="notes" class="input" rows="2"></textarea>
      </label>
      <label class="flex items-start gap-3 rounded-lg border border-gray-200 p-4 dark:border-dark-700">
        <input v-model="reviewRequired" type="checkbox" class="mt-1" />
        <span><span class="block font-medium">{{ t('supplier.admin.reviewRequired') }}</span><span class="text-sm text-gray-500">{{ t('supplier.admin.reviewRequiredHint') }}</span></span>
      </label>
      <fieldset v-if="!reviewRequired" class="space-y-3">
        <legend class="input-label">{{ t('supplier.admin.autoApproveGroups') }}</legend>
        <p class="input-hint">{{ t('supplier.admin.autoApproveGroupsHint') }}</p>
        <label v-for="platform in selectedPlatforms" :key="platform" class="block">
          <span class="input-label">{{ platformLabel(platform) }}</span>
          <select v-model.number="autoApproveGroups[platform]" class="input" required>
            <option :value="0">{{ t('supplier.admin.selectGroup') }}</option>
            <option v-for="group in groups.filter(item => item.platform === platform && item.status === 'active')" :key="group.id" :value="group.id">{{ group.name }} (#{{ group.id }})</option>
          </select>
        </label>
      </fieldset>
      <p v-if="policyError" role="alert" class="text-sm text-red-600">{{ policyError }}</p>
      <fieldset>
        <legend class="input-label">{{ t('supplier.admin.allowedKinds') }}</legend>
        <p class="input-hint mb-3">{{ t('supplier.admin.noPermission') }}</p>

        <div class="mb-3 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-primary-200 bg-primary-50/60 px-4 py-3 dark:border-primary-900/60 dark:bg-primary-900/10">
          <label class="flex cursor-pointer items-center gap-2 font-medium text-gray-900 dark:text-white">
            <input
              data-test="select-all-kinds"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="allKindsSelected"
              :indeterminate="someKindsSelected"
              @change="toggleAllKinds(($event.target as HTMLInputElement).checked)"
            />
            {{ t('supplier.admin.selectAllKinds') }}
          </label>
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('supplier.admin.selectedKinds', { selected: selectedKinds.length, total: SUPPLIER_ACCOUNT_KINDS.length }) }}
          </span>
        </div>

        <div class="grid gap-3 sm:grid-cols-2">
          <div v-for="column in [0, 1]" :key="column" data-test="kind-column" class="flex flex-col gap-3">
            <details
              v-for="group in kindGroups.filter((_, index) => index % 2 === column)"
              :key="group.platform"
              class="group rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900"
            >
              <summary class="flex cursor-pointer items-center gap-2 px-4 py-3 font-medium text-gray-900 dark:text-white">
                <input
                  :data-test="`platform-${group.platform}`"
                  type="checkbox"
                  class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                  :checked="platformSelected(group.kinds)"
                  :indeterminate="platformPartiallySelected(group.kinds)"
                  @click.stop
                  @change="togglePlatform(group.kinds, ($event.target as HTMLInputElement).checked)"
                />
                {{ platformLabel(group.platform) }}
                <span class="ml-auto text-xs font-normal text-gray-400">{{ group.kinds.filter(kind => selectedKinds.includes(kindKey(kind))).length }} / {{ group.kinds.length }}</span>
                <span aria-hidden="true" class="text-xs text-gray-500 transition-transform group-open:rotate-180">▾</span>
              </summary>
              <div class="grid gap-1 border-t border-gray-100 p-2 dark:border-dark-700 sm:grid-cols-2">
                <label
                  v-for="kind in group.kinds"
                  :key="kindKey(kind)"
                  class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-2 text-sm text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-800"
                >
                  <input
                    v-model="selectedKinds"
                    :data-test="`kind-${kindKey(kind)}`"
                    type="checkbox"
                    :value="kindKey(kind)"
                    class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                  />
                  <span>{{ t(`supplier.admin.kindTypes.${kind.type}`) }}</span>
                </label>
              </div>
            </details>
          </div>
        </div>
      </fieldset>
    </form>
    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="saving" @click="emit('close')">{{ t('common.cancel') }}</button>
      <button type="submit" form="supplier-form" class="btn btn-primary" :disabled="saving">{{ saving ? t('common.saving') : t('common.save') }}</button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { AdminSupplier, SupplierWriteInput } from '@/api/admin/suppliers'
import type { SupplierAccountKind } from '@/api/supplier'
import { SUPPLIER_ACCOUNT_KINDS } from '@/features/supplier/accountKinds'
import { getAll as getAllGroups } from '@/api/admin/groups'
import type { AdminGroup } from '@/types'

const props = defineProps<{ show: boolean; supplier?: AdminSupplier | null; saving?: boolean }>()
const emit = defineEmits<{ (event: 'close'): void; (event: 'submit', input: SupplierWriteInput): void }>()
const { t } = useI18n()

const name = ref('')
const status = ref<'active' | 'disabled'>('active')
const notes = ref('')
const selectedKinds = ref<string[]>([])
const reviewRequired = ref(true)
const autoApproveGroups = ref<Record<string, number>>({})
const groups = ref<AdminGroup[]>([])
const policyError = ref('')
const selectedPlatforms = computed(() => [...new Set(SUPPLIER_ACCOUNT_KINDS.filter(kind => selectedKinds.value.includes(kindKey(kind))).map(kind => kind.platform))])
const kindKey = (kind: SupplierAccountKind) => `${kind.platform}:${kind.type}`
const kindGroups = computed(() => {
  const groups = new Map<string, SupplierAccountKind[]>()
  for (const kind of SUPPLIER_ACCOUNT_KINDS) {
    groups.set(kind.platform, [...(groups.get(kind.platform) ?? []), kind])
  }
  return [...groups].map(([platform, kinds]) => ({ platform, kinds }))
})
const allKindsSelected = computed(() => SUPPLIER_ACCOUNT_KINDS.every((kind) => selectedKinds.value.includes(kindKey(kind))))
const someKindsSelected = computed(() => SUPPLIER_ACCOUNT_KINDS.some((kind) => selectedKinds.value.includes(kindKey(kind))) && !allKindsSelected.value)

const platformNames: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
  kimi: 'Kimi',
  zhipu: '智谱 GLM',
  deepseek: 'DeepSeek',
  minimax: 'MiniMax',
  opencode_go: 'OpenCode Go',
}
const platformLabel = (platform: string) => platformNames[platform] ?? platform

function toggleKinds(kinds: SupplierAccountKind[], checked: boolean) {
  const keys = new Set(selectedKinds.value)
  for (const kind of kinds) {
    if (checked) keys.add(kindKey(kind))
    else keys.delete(kindKey(kind))
  }
  selectedKinds.value = [...keys]
}

function toggleAllKinds(checked: boolean) {
  selectedKinds.value = checked ? SUPPLIER_ACCOUNT_KINDS.map(kindKey) : []
}

function togglePlatform(kinds: SupplierAccountKind[], checked: boolean) {
  toggleKinds(kinds, checked)
}

function platformSelected(kinds: SupplierAccountKind[]) {
  return kinds.every((kind) => selectedKinds.value.includes(kindKey(kind)))
}

function platformPartiallySelected(kinds: SupplierAccountKind[]) {
  const selected = kinds.filter((kind) => selectedKinds.value.includes(kindKey(kind))).length
  return selected > 0 && selected < kinds.length
}

watch(() => [props.show, props.supplier] as const, ([show]) => {
  if (!show) return
  name.value = props.supplier?.name ?? ''
  status.value = props.supplier?.status ?? 'active'
  notes.value = props.supplier?.notes ?? ''
  selectedKinds.value = props.supplier?.allowed_account_kinds.map(kindKey) ?? []
  reviewRequired.value = props.supplier?.review_required ?? true
  autoApproveGroups.value = { ...props.supplier?.auto_approve_groups }
  policyError.value = ''
  void getAllGroups().then(items => { groups.value = items }).catch(() => { groups.value = [] })
}, { immediate: true, deep: true })

function submit() {
  if (!reviewRequired.value && selectedPlatforms.value.some(platform => !autoApproveGroups.value[platform])) {
    policyError.value = t('supplier.admin.selectGroup')
    return
  }
  policyError.value = ''
  emit('submit', {
    ...(props.supplier ? { code: props.supplier.code } : {}),
    name: name.value,
    status: status.value,
    notes: notes.value || null,
    allowed_account_kinds: SUPPLIER_ACCOUNT_KINDS.filter((kind) => selectedKinds.value.includes(kindKey(kind))),
    review_required: reviewRequired.value,
    auto_approve_groups: reviewRequired.value ? {} : Object.fromEntries(selectedPlatforms.value.map(platform => [platform, autoApproveGroups.value[platform]])),
  })
}
</script>
