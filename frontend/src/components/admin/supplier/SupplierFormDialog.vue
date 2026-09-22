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
        <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <label v-for="kind in SUPPLIER_ACCOUNT_KINDS" :key="kindKey(kind)" class="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-700">
            <input v-model="selectedKinds" type="checkbox" :value="kindKey(kind)" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            <span>{{ kind.platform }} / {{ kind.type }}</span>
          </label>
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
import { ref, watch } from 'vue'
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
