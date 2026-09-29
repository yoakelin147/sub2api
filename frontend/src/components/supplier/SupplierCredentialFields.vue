<template>
  <div class="grid gap-4 sm:grid-cols-2">
    <label v-for="field in fields" :key="field" class="block" :class="{ 'sm:col-span-2': field === 'service_account_json' }">
      <span class="input-label">{{ t(`supplier.fields.${field}`) }} <span v-if="required.includes(field)" class="text-red-500">*</span><code class="ml-1 text-xs font-normal text-gray-400">{{ field }}</code></span>
      <select v-if="choices[field]" class="input" :value="credentials[field] ?? ''" :disabled="disabled" :data-field="field" @change="update(field, ($event.target as HTMLSelectElement).value)">
        <option value="" disabled>{{ t('common.required') }}</option>
        <option v-if="field === 'api_protocol' && credentials[field] === 'adaptive'" value="adaptive">adaptive (JSON)</option>
        <option v-for="choice in choices[field]" :key="choice" :value="choice">{{ choice }}</option>
      </select>
      <textarea v-else-if="field === 'service_account_json'" class="input font-mono text-sm" rows="5" autocomplete="off" spellcheck="false" :value="typeof credentials[field] === 'object' ? JSON.stringify(credentials[field], null, 2) : credentials[field] as string" :disabled="disabled" :data-field="field" @input="update(field, ($event.target as HTMLTextAreaElement).value)"></textarea>
      <input v-else class="input" :type="secretFields.includes(field) ? 'password' : field === 'email' ? 'email' : 'text'" autocomplete="off" spellcheck="false" :value="credentials[field] ?? ''" :placeholder="field === 'base_url' ? 'https://…' : ''" :disabled="disabled" :data-field="field" @input="update(field, ($event.target as HTMLInputElement).value)" />
      <span v-if="field === 'base_url' && kind.type === 'apikey' && !required.includes(field)" class="input-hint" data-test="api-key-default-base-url-hint">{{ t('supplier.accounts.apiKeyDefaultBaseUrlHint') }}</span>
    </label>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SupplierAccountKind } from '@/api/supplier'
import { supplierRequiredCredentials } from '@/features/supplier/accountKinds'

const props = defineProps<{ kind: SupplierAccountKind; credentials: Record<string, unknown>; disabled?: boolean }>()
const emit = defineEmits<{ (event: 'update:credentials', value: Record<string, unknown>): void }>()
const { t } = useI18n()
const secretFields = ['api_key', 'access_token', 'refresh_token', 'password', 'aws_access_key_id', 'aws_secret_access_key']
const required = computed(() => supplierRequiredCredentials(props.kind, props.credentials))
const fields = computed(() => {
  const optional = props.kind.type === 'oauth' || props.kind.type === 'setup-token' ? ['refresh_token'] : []
  if (props.kind.type === 'apikey' && !required.value.includes('base_url')) optional.push('base_url')
  return [...required.value, ...optional]
})
const choices = computed<Record<string, string[]>>(() => ({
  auth_mode: ['api_key', 'sigv4'],
  account_mode: props.kind.platform === 'opencode_go' ? ['go', 'zen'] : ['payg', 'coding'],
  api_protocol: ['chat_completions', 'anthropic', 'responses'],
}))

function update(field: string, value: string) {
  const credentials = { ...props.credentials }
  if (value) credentials[field] = value
  else delete credentials[field]
  emit('update:credentials', credentials)
}
</script>
