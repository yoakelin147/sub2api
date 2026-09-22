<template>
  <BaseDialog :show="show" :title="account ? t('supplier.accounts.edit') : t('supplier.accounts.create')" width="wide" @close="emit('close')">
    <form id="supplier-account-form" class="space-y-5" @submit.prevent="submit">
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
          <select v-model="kindKey" class="input" :disabled="Boolean(account)" required>
            <option v-for="kind in kinds" :key="kindValue(kind)" :value="kindValue(kind)">
              {{ kind.platform }} / {{ kind.type }}
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
          <label for="supplier-credentials" class="input-label mb-0">{{ t('supplier.accounts.credentials') }}</label>
          <button type="button" class="btn btn-secondary btn-sm" @click="insertTemplate">
            {{ t('supplier.accounts.credentialTemplate') }}
          </button>
        </div>
        <textarea
          id="supplier-credentials"
          v-model="credentialsText"
          class="input min-h-48 font-mono text-sm"
          :required="!account"
          spellcheck="false"
          autocomplete="off"
          placeholder="{&#10;  &quot;api_key&quot;: &quot;...&quot;&#10;}"
        ></textarea>
        <p class="input-hint">{{ t('supplier.accounts.credentialsHint') }}</p>
        <p v-if="formError" role="alert" class="mt-2 text-sm text-red-600 dark:text-red-400">{{ formError }}</p>
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
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
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
const formError = ref('')

const kindValue = (kind: SupplierAccountKind) => `${kind.platform}:${kind.type}`

function selectedKind(): SupplierAccountKind | undefined {
  return props.kinds.find((kind) => kindValue(kind) === kindKey.value)
}

function credentialTemplate(kind: SupplierAccountKind | undefined): Record<string, unknown> {
  if (!kind) return {}
  if (kind.type === 'oauth' || kind.type === 'setup-token') return { access_token: '', refresh_token: '' }
  if (kind.type === 'bedrock') return { auth_mode: 'api_key', aws_region: 'us-east-1', api_key: '' }
  if (kind.type === 'service_account') {
    return { service_account_json: '{"project_id":"","client_email":"","private_key":""}', location: 'us-central1' }
  }
  if (['kimi', 'zhipu', 'deepseek', 'minimax', 'opencode_go'].includes(kind.platform)) {
    return { api_key: '', account_mode: kind.platform === 'opencode_go' ? 'go' : 'payg', api_protocol: 'chat_completions' }
  }
  const template: Record<string, unknown> = { api_key: '' }
  if (kind.type === 'upstream') template.base_url = 'https://'
  return template
}

function insertTemplate() {
  credentialsText.value = JSON.stringify(credentialTemplate(selectedKind()), null, 2)
  formError.value = ''
}

function reset() {
  const account = props.account
  name.value = account?.name ?? ''
  externalId.value = account?.external_id ?? ''
  notes.value = account?.notes ?? ''
  expiresAt.value = account?.expires_at ? new Date(account.expires_at).toISOString().slice(0, 16) : ''
  kindKey.value = account ? `${account.platform}:${account.type}` : props.kinds[0] ? kindValue(props.kinds[0]) : ''
  credentialsText.value = account ? '' : JSON.stringify(credentialTemplate(selectedKind()), null, 2)
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
  return parsed as Record<string, unknown>
}

function submit() {
  formError.value = ''
  try {
    const credentials = parseCredentials()
    const expiry = expiresAt.value ? Math.floor(new Date(expiresAt.value).getTime() / 1000) : null
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
