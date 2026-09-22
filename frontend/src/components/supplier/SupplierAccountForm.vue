<template>
  <BaseDialog :show="show" :title="account ? t('supplier.accounts.edit') : t('supplier.accounts.create')" width="wide" @close="emit('close')">
    <form id="supplier-account-form" class="space-y-5" @submit.prevent="submit">
      <p class="rounded-lg bg-gray-50 p-3 text-sm text-gray-600 dark:bg-dark-800 dark:text-gray-300">{{ t('supplier.accounts.configurationScope') }}</p>
      <div class="grid gap-4 sm:grid-cols-2">
        <label class="block">
          <span class="input-label">{{ t('common.name') }}</span>
          <input v-model.trim="name" class="input" required maxlength="100" autocomplete="off" />
        </label>
        <label class="block">
          <span class="input-label">{{ t('supplier.accounts.externalId') }}</span>
          <input v-model.trim="externalId" class="input" maxlength="191" autocomplete="off" />
        </label>
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <label class="block">
          <span class="input-label">{{ t('supplier.accounts.platform') }} / {{ t('supplier.accounts.type') }}</span>
          <select :value="kindKey" class="input" :disabled="Boolean(account)" required @change="changeKind">
            <option v-for="kind in kinds" :key="kindValue(kind)" :value="kindValue(kind)">
              {{ t(`monitorCommon.providers.${kind.platform}`) }} / {{ t(`supplier.admin.kindTypes.${kind.type}`) }}
            </option>
          </select>
        </label>
        <label class="block">
          <span class="input-label">{{ t('supplier.accounts.expiresAt') }}</span>
          <input v-model="expiresAt" class="input" type="datetime-local" />
        </label>
      </div>

      <label class="block">
        <span class="input-label">{{ t('supplier.accounts.notes') }}</span>
        <textarea v-model="notes" class="input" rows="2" maxlength="2000"></textarea>
      </label>

      <div>
        <div class="mb-2 flex items-center justify-between gap-3">
          <span class="input-label mb-0">{{ t('supplier.accounts.credentials') }}</span>
          <div class="flex gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="saving" @click="toggleCredentialMode">{{ t(credentialMode === 'fields' ? 'supplier.accounts.jsonMode' : 'supplier.accounts.fieldsMode') }}</button>
            <button v-if="credentialMode === 'json'" type="button" class="btn btn-secondary btn-sm" :disabled="saving" @click="insertTemplate">{{ t('supplier.accounts.credentialTemplate') }}</button>
          </div>
        </div>
        <p v-if="selectedKind()?.platform === 'antigravity'" class="mb-3 text-sm text-amber-700 dark:text-amber-300">{{ t(selectedKind()?.type === 'apikey' ? 'supplier.accounts.antigravityKeyHint' : 'supplier.accounts.oauthHint') }}</p>
        <SupplierCredentialFields v-if="credentialMode === 'fields' && selectedKind()" :kind="selectedKind()!" :credentials="credentialValues" :disabled="saving" @update:credentials="credentialsText = JSON.stringify($event)" />
        <textarea
          v-else
          id="supplier-credentials"
          v-model="credentialsText"
          class="input min-h-48 font-mono text-sm"
          :required="!account"
          spellcheck="false"
          autocomplete="off"
          :aria-label="t('supplier.accounts.credentials')"
          :disabled="saving"
          placeholder="{&#10;  &quot;api_key&quot;: &quot;...&quot;&#10;}"
        ></textarea>
        <p class="input-hint">{{ t('supplier.accounts.credentialsHint') }}</p>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ t('supplier.accounts.requiredCredentials', { fields: requiredFields.join(', ') }) }}</p>
        <p v-if="requiredFields.includes('base_url')" class="mt-1 text-sm text-amber-700 dark:text-amber-300">{{ t('supplier.accounts.baseUrlHint') }}</p>
        <p v-if="formError" role="alert" class="mt-2 text-sm text-red-600 dark:text-red-400">{{ formError }}</p>
        <p v-else-if="serverError" role="alert" class="mt-2 text-sm text-red-600 dark:text-red-400">{{ serverError }}</p>
      </div>
    </form>

    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="saving" @click="emit('close')">{{ t('common.cancel') }}</button>
      <button type="submit" form="supplier-account-form" class="btn btn-primary" :disabled="saving || kinds.length === 0">
        {{ saving ? t('common.saving') : t('common.save') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import SupplierCredentialFields from './SupplierCredentialFields.vue'
import { formatDateTimeLocalInput, parseDateTimeLocalInput } from '@/utils/format'
import { supplierCredentialTemplate, supplierRequiredCredentials } from '@/features/supplier/accountKinds'
import type {
  SupplierAccount,
  SupplierAccountInput,
  SupplierAccountKind,
  SupplierAccountUpdateInput,
} from '@/api/supplier'

const props = defineProps<{
  show: boolean
  kinds: SupplierAccountKind[]
  account?: SupplierAccount | null
  saving?: boolean
  serverError?: string
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'submit', accountId: number | null, input: SupplierAccountInput | SupplierAccountUpdateInput): void
}>()

const { t } = useI18n()
const name = ref('')
const externalId = ref('')
const notes = ref('')
const expiresAt = ref('')
const kindKey = ref('')
const credentialsText = ref('')
const credentialMode = ref<'fields' | 'json'>('fields')
const formError = ref('')

const kindValue = (kind: SupplierAccountKind) => `${kind.platform}:${kind.type}`

function selectedKind(): SupplierAccountKind | undefined {
  return props.kinds.find((kind) => kindValue(kind) === kindKey.value)
}

const requiredFields = computed(() => {
  const kind = selectedKind()
  if (!kind) return []
  try { return supplierRequiredCredentials(kind, JSON.parse(credentialsText.value)) }
  catch { return supplierRequiredCredentials(kind) }
})
const credentialValues = computed<Record<string, unknown>>(() => {
  try {
    const value = JSON.parse(credentialsText.value || '{}')
    return value && typeof value === 'object' && !Array.isArray(value) ? value : {}
  } catch { return {} }
})

function toggleCredentialMode() {
  if (credentialMode.value === 'json' && credentialsText.value.trim()) {
    try {
      const value = JSON.parse(credentialsText.value)
      if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error()
    } catch { formError.value = t('supplier.accounts.invalidCredentials'); return }
  }
  credentialMode.value = credentialMode.value === 'fields' ? 'json' : 'fields'
  formError.value = ''
}

function confirmReplaceCredentials() {
  const template = JSON.stringify(supplierCredentialTemplate(selectedKind()), null, 2)
  return !credentialsText.value.trim() || credentialsText.value === template || window.confirm(t('supplier.accounts.confirmReplaceCredentials'))
}

function changeKind(event: Event) {
  const select = event.target as HTMLSelectElement
  if (!confirmReplaceCredentials()) { select.value = kindKey.value; return }
  kindKey.value = select.value
  credentialsText.value = JSON.stringify(supplierCredentialTemplate(selectedKind()), null, 2)
  formError.value = ''
}

function insertTemplate() {
  if (!confirmReplaceCredentials()) return
  credentialsText.value = JSON.stringify(supplierCredentialTemplate(selectedKind()), null, 2)
  formError.value = ''
}

function reset() {
  const account = props.account
  name.value = account?.name ?? ''
  externalId.value = account?.external_id ?? ''
  notes.value = account?.notes ?? ''
  expiresAt.value = account?.expires_at ? formatDateTimeLocalInput(new Date(account.expires_at).getTime() / 1000) : ''
  kindKey.value = account ? `${account.platform}:${account.type}` : props.kinds[0] ? kindValue(props.kinds[0]) : ''
  credentialsText.value = account ? '' : JSON.stringify(supplierCredentialTemplate(selectedKind()), null, 2)
  credentialMode.value = 'fields'
  formError.value = ''
}

watch(() => [props.show, props.account, props.kinds] as const, ([show]) => {
  if (show) reset()
}, { immediate: true, deep: true })

function parseCredentials(): Record<string, unknown> | undefined {
  if (!credentialsText.value.trim()) return undefined
  let parsed: unknown
  try {
    parsed = JSON.parse(credentialsText.value)
  } catch {
    throw new Error(t('supplier.accounts.invalidCredentials'))
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error(t('supplier.accounts.invalidCredentials'))
  }
  const credentials = parsed as Record<string, unknown>
  const kind = selectedKind()
  if (!kind) throw new Error(t('supplier.accounts.invalidCredentials'))
  const missing = supplierRequiredCredentials(kind, credentials).filter(field => {
    const value = credentials[field]
    return field === 'service_account_json'
      ? !(typeof value === 'string' ? value.trim() : value && typeof value === 'object' && !Array.isArray(value) && Object.keys(value).length)
      : typeof value !== 'string' || !value.trim()
  })
  if (missing.length) throw new Error(t('supplier.accounts.missingCredentials', { fields: missing.join(', ') }))
  if (credentials.base_url !== undefined && credentials.base_url !== '') {
    try {
      const url = new URL(String(credentials.base_url))
      if (url.protocol !== 'https:' || !url.hostname || url.username || url.password) throw new Error()
    } catch { throw new Error(t('supplier.accounts.invalidBaseUrl')) }
  }
  return credentials
}

function submit() {
  if (props.saving) return
  formError.value = ''
  try {
    const credentials = props.account && Object.keys(credentialValues.value).length === 0 && credentialMode.value === 'fields' ? undefined : parseCredentials()
    const expiry = parseDateTimeLocalInput(expiresAt.value)
    if (props.account) {
      const input: SupplierAccountUpdateInput = {
        name: name.value,
        external_id: externalId.value,
        notes: notes.value || null,
        expires_at: expiry,
      }
      if (credentials) input.credentials = credentials
      emit('submit', props.account.id, input)
      return
    }
    const kind = selectedKind()
    if (!kind || !credentials) throw new Error(t('supplier.accounts.invalidCredentials'))
    emit('submit', null, {
      name: name.value,
      external_id: externalId.value,
      notes: notes.value || null,
      platform: kind.platform,
      type: kind.type,
      credentials,
      expires_at: expiry,
    })
  } catch (error) {
    formError.value = error instanceof Error ? error.message : t('supplier.accounts.invalidCredentials')
  }
}
</script>
