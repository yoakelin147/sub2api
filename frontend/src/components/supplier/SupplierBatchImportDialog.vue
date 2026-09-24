<template>
  <BaseDialog :show="show" :title="t('supplier.batch.title')" width="wide" @close="close">
    <div class="space-y-5">
      <fieldset>
        <legend class="input-label">{{ t('supplier.batch.format') }}</legend>
        <div class="flex flex-wrap gap-2">
          <label v-for="option in formats" :key="option.value" class="cursor-pointer">
            <input v-model="format" class="peer sr-only" type="radio" :value="option.value" />
            <span class="inline-flex rounded-lg border border-gray-200 px-4 py-2 text-sm peer-checked:border-primary-500 peer-checked:bg-primary-50 peer-checked:text-primary-700 dark:border-dark-600 dark:peer-checked:bg-primary-900/20 dark:peer-checked:text-primary-300">
              {{ option.label }}
            </span>
          </label>
        </div>
      </fieldset>

      <label class="block">
        <span class="input-label">{{ t('supplier.batch.input') }}</span>
        <textarea v-model="input" class="input min-h-72 font-mono text-xs" spellcheck="false"></textarea>
        <p class="input-hint">{{ formatHint }}</p>
      </label>
      <label class="block">
        <span class="input-label">{{ t('supplier.accounts.proxy') }}</span>
        <select v-model.number="proxyId" class="input" required>
          <option :value="0">{{ proxies.length ? t('supplier.accounts.selectProxy') : t('supplier.accounts.directConnection') }}</option>
          <option v-for="proxy in proxies" :key="proxy.id" :value="proxy.id">{{ proxy.name }}</option>
        </select>
        <p class="input-hint">{{ t('supplier.batch.proxyHint') }}</p>
      </label>

      <div v-if="parseError" role="alert" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300">
        {{ parseError }}
      </div>
      <div v-else-if="accounts.length" class="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-900/50 dark:bg-green-900/20 dark:text-green-300">
        {{ t('supplier.batch.parsed', { count: accounts.length }) }}
      </div>

      <section v-if="result" aria-live="polite" class="rounded-lg border border-gray-200 dark:border-dark-700">
        <div class="border-b border-gray-200 px-4 py-3 dark:border-dark-700">
          <h4 class="font-medium text-gray-900 dark:text-white">{{ t('supplier.batch.result') }}</h4>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('supplier.batch.resultSummary', { succeeded: result.succeeded, failed: result.failed }) }}
          </p>
        </div>
        <ul class="max-h-56 divide-y divide-gray-100 overflow-y-auto dark:divide-dark-700">
          <li v-for="item in result.results" :key="item.index" class="flex items-start justify-between gap-4 px-4 py-2 text-sm">
            <span>{{ t('supplier.batch.row', { index: item.index + 1 }) }} · {{ item.external_id || '-' }}</span>
            <span :class="item.success ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">
              {{ item.success ? `#${item.account_id}` : `${item.error_code || ''} ${item.message || ''}` }}
            </span>
          </li>
        </ul>
      </section>
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="submitting" @click="close">{{ t('common.close') }}</button>
      <button type="button" class="btn btn-secondary" :disabled="submitting || !input.trim()" @click="parse">{{ t('supplier.batch.parse') }}</button>
      <button type="button" class="btn btn-primary" :disabled="submitting || accounts.length === 0 || (proxies.length > 0 && !proxyId)" @click="submit">
        {{ submitting ? t('common.submitting') : t('supplier.batch.submit', { count: accounts.length }) }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { batchCreateAccounts, type SupplierAccountInput, type SupplierBatchResult, type SupplierProxyOption } from '@/api/supplier'
import { parseSupplierAccountImport, type SupplierImportFormat } from '@/features/supplier/accountImport'

const props = defineProps<{ show: boolean; proxies: SupplierProxyOption[]; reviewRequired?: boolean; approvedGroups?: Record<string, number[]> }>()
const emit = defineEmits<{ (event: 'close'): void; (event: 'completed'): void }>()
const { t } = useI18n()

const format = ref<SupplierImportFormat>('json')
const input = ref('')
const accounts = ref<SupplierAccountInput[]>([])
const parseError = ref('')
const submitting = ref(false)
const result = ref<SupplierBatchResult | null>(null)
const proxyId = ref(0)

const formats = computed(() => [
  { value: 'json' as const, label: t('supplier.batch.json') },
  { value: 'csv' as const, label: t('supplier.batch.csv') },
  { value: 'text' as const, label: t('supplier.batch.text') },
])
const formatHint = computed(() => t(`supplier.batch.formatHint${format.value[0].toUpperCase()}${format.value.slice(1)}`))

watch(() => props.show, (show) => {
  if (!show) return
  accounts.value = []
  parseError.value = ''
  result.value = null
  proxyId.value = props.proxies[0]?.id ?? 0
})

watch(() => props.proxies, (proxies) => {
  if (!proxyId.value) proxyId.value = proxies[0]?.id ?? 0
})

watch([format, input], () => {
  accounts.value = []
  parseError.value = ''
  result.value = null
})

function parse() {
  try {
    accounts.value = parseSupplierAccountImport(format.value, input.value)
    if (props.reviewRequired && accounts.value.some(account => ['notes', 'expires_at', 'concurrency', 'priority', 'load_factor', 'auto_pause_on_expired', 'group_ids', 'extra'].some(field => field in account))) {
      throw new Error(t('supplier.batch.restrictedFields'))
    }
    parseError.value = ''
  } catch (error) {
    accounts.value = []
    parseError.value = error instanceof Error ? error.message : t('supplier.batch.failed')
  }
}

async function submit() {
  if (!accounts.value.length) return
  submitting.value = true
  parseError.value = ''
  try {
    if (props.proxies.length && !props.proxies.some(proxy => proxy.id === proxyId.value)) throw new Error(t('supplier.accounts.noProxy'))
    result.value = await batchCreateAccounts(accounts.value.map(account => ({
      ...account,
      ...(account.proxy_id || proxyId.value ? { proxy_id: account.proxy_id || proxyId.value } : {}),
      ...(!props.reviewRequired ? { group_ids: account.group_ids || props.approvedGroups?.[account.platform]?.slice(0, 1) || [] } : {}),
    })))
    if (result.value.succeeded > 0) emit('completed')
  } catch (error) {
    parseError.value = (error as { message?: string }).message || t('supplier.batch.failed')
  } finally {
    submitting.value = false
  }
}

function close() {
  if (!submitting.value) emit('close')
}
</script>
