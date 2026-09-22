<template>
  <BaseDialog :show="show" :title="supplier ? t('supplier.admin.edit') : t('supplier.admin.create')" width="wide" @close="emit('close')">
    <form id="supplier-form" class="space-y-5" @submit.prevent="submit">
      <div class="grid gap-4 sm:grid-cols-2">
        <label>
          <span class="input-label">{{ t('supplier.admin.code') }}</span>
          <input v-model.trim="code" class="input" required maxlength="64" pattern="[a-z0-9][a-z0-9_-]{0,63}" :disabled="Boolean(supplier)" />
        </label>
        <label>
          <span class="input-label">{{ t('common.name') }}</span>
          <input v-model.trim="name" class="input" required maxlength="120" />
        </label>
      </div>
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
          <section
            v-for="group in kindGroups"
            :key="group.platform"
            class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900"
          >
            <label class="flex cursor-pointer items-center gap-2 border-b border-gray-100 px-4 py-3 font-medium text-gray-900 dark:border-dark-700 dark:text-white">
              <input
                :data-test="`platform-${group.platform}`"
                type="checkbox"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="platformSelected(group.kinds)"
                :indeterminate="platformPartiallySelected(group.kinds)"
                @change="togglePlatform(group.kinds, ($event.target as HTMLInputElement).checked)"
              />
              {{ platformLabel(group.platform) }}
              <span class="ml-auto text-xs font-normal text-gray-400">{{ group.kinds.length }}</span>
            </label>
            <div class="grid gap-1 p-2 sm:grid-cols-2">
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
          </section>
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

const props = defineProps<{ show: boolean; supplier?: AdminSupplier | null; saving?: boolean }>()
const emit = defineEmits<{ (event: 'close'): void; (event: 'submit', input: SupplierWriteInput): void }>()
const { t } = useI18n()

const code = ref('')
const name = ref('')
const status = ref<'active' | 'disabled'>('active')
const notes = ref('')
const selectedKinds = ref<string[]>([])
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
  code.value = props.supplier?.code ?? ''
  name.value = props.supplier?.name ?? ''
  status.value = props.supplier?.status ?? 'active'
  notes.value = props.supplier?.notes ?? ''
  selectedKinds.value = props.supplier?.allowed_account_kinds.map(kindKey) ?? []
}, { immediate: true, deep: true })

function submit() {
  emit('submit', {
    code: code.value,
    name: name.value,
    status: status.value,
    notes: notes.value || null,
    allowed_account_kinds: SUPPLIER_ACCOUNT_KINDS.filter((kind) => selectedKinds.value.includes(kindKey(kind))),
  })
}
</script>
