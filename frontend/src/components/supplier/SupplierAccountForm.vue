<template>
  <BaseDialog :show="show" :title="account ? t('supplier.accounts.edit') : t('supplier.accounts.create')" width="wide" @close="emit('close')">
    <form id="supplier-account-form" class="space-y-5" @submit.prevent="submit">
      <label class="block">
        <span class="input-label">{{ t('common.name') }}</span>
        <input v-model.trim="name" class="input" required maxlength="100" autocomplete="off" />
      </label>
      <details class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <summary class="cursor-pointer text-sm">{{ t('supplier.accounts.externalIdOptional') }}</summary>
        <label class="mt-3 block">
          <span class="input-label">{{ t('supplier.accounts.externalId') }}</span>
          <input v-model.trim="externalId" class="input" maxlength="191" autocomplete="off" />
        </label>
        <p class="input-hint">{{ t('supplier.accounts.externalIdHint') }}</p>
      </details>

      <div class="grid gap-4 sm:grid-cols-2">
        <label class="block">
          <span class="input-label">{{ t('supplier.accounts.proxy') }}</span>
          <select v-model.number="proxyId" class="input" :disabled="oauthBusy" required>
            <option :value="0">{{ proxies.length ? t('supplier.accounts.selectProxy') : t('supplier.accounts.directConnection') }}</option>
            <option v-for="proxy in proxies" :key="proxy.id" :value="proxy.id">{{ proxy.name }}</option>
          </select>
        </label>
        <label v-if="!reviewRequired" class="block">
          <span class="input-label">{{ t('supplier.accounts.concurrency') }}</span>
          <input v-model.number="concurrency" class="input" type="number" min="1" max="10000" required />
        </label>
        <label v-if="!reviewRequired" class="block">
          <span class="input-label">{{ t('supplier.accounts.priority') }}</span>
          <input v-model.number="priority" class="input" type="number" min="0" max="10000" required />
        </label>
        <label v-if="!reviewRequired" class="block">
          <span class="input-label">{{ t('supplier.accounts.loadFactor') }}</span>
          <input v-model.number="loadFactor" class="input" type="number" min="1" max="10000" :placeholder="t('supplier.accounts.loadFactorHint')" />
        </label>
      </div>
      <label v-if="!reviewRequired" class="flex items-center gap-2 text-sm"><input v-model="autoPauseOnExpired" type="checkbox" />{{ t('supplier.accounts.autoPauseOnExpired') }}</label>

      <div class="grid gap-4 sm:grid-cols-2">
        <label class="block">
          <span class="input-label">{{ t('supplier.accounts.platform') }} / {{ t('supplier.accounts.type') }}</span>
          <select :value="kindKey" class="input" :disabled="Boolean(account) || oauthBusy" required @change="changeKind">
            <option v-for="kind in kinds" :key="kindValue(kind)" :value="kindValue(kind)">
              {{ t(`monitorCommon.providers.${kind.platform}`) }} / {{ t(`supplier.admin.kindTypes.${kind.type}`) }}
            </option>
          </select>
        </label>
        <label v-if="!reviewRequired" class="block">
          <span class="input-label">{{ t('supplier.accounts.expiresAt') }}</span>
          <input v-model="expiresAt" class="input" type="datetime-local" />
        </label>
      </div>

      <label v-if="!reviewRequired" class="block">
        <span class="input-label">{{ t('supplier.accounts.notes') }}</span>
        <textarea v-model="notes" class="input" rows="2" maxlength="2000"></textarea>
      </label>
      <fieldset v-if="!reviewRequired">
        <legend class="input-label">{{ t('supplier.accounts.group') }}</legend>
        <details class="rounded-lg border border-gray-300 dark:border-dark-600" data-test="supplier-group-dropdown">
          <summary class="flex cursor-pointer items-center gap-2 px-3 py-2 text-sm text-gray-800 dark:text-gray-200">
            <span class="min-w-0 flex-1 truncate">{{ selectedGroupNames || t('supplier.accounts.selectGroup') }}</span>
            <span class="text-xs text-gray-400">{{ selectedGroupIds.length }} / {{ availableGroups.length }}</span>
            <span aria-hidden="true">▾</span>
          </summary>
          <div class="max-h-48 space-y-1 overflow-y-auto border-t border-gray-200 p-2 dark:border-dark-600">
            <label v-for="group in availableGroups" :key="group.id" class="flex cursor-pointer items-start gap-2 rounded px-2 py-2 text-sm hover:bg-gray-50 dark:hover:bg-dark-700">
              <input v-model="selectedGroupIds" type="checkbox" :value="group.id" class="mt-1" />
              <span class="min-w-0"><span class="block break-words">{{ group.name }}</span><span v-if="group.description" class="block break-words text-xs text-gray-500 dark:text-gray-400">{{ group.description }}</span></span>
            </label>
            <p v-if="!availableGroups.length" class="px-2 py-2 text-sm text-gray-500">{{ t('supplier.accounts.noGroup') }}</p>
          </div>
        </details>
      </fieldset>

      <div>
        <div class="mb-2 flex items-center justify-between gap-3">
          <span class="input-label mb-0">{{ t('supplier.accounts.credentials') }}</span>
          <div class="flex gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="saving" @click="toggleCredentialMode">{{ t(credentialMode === 'fields' ? 'supplier.accounts.jsonMode' : 'supplier.accounts.fieldsMode') }}</button>
            <button v-if="credentialMode === 'json'" type="button" class="btn btn-secondary btn-sm" :disabled="saving" @click="insertTemplate">{{ t('supplier.accounts.credentialTemplate') }}</button>
          </div>
        </div>
        <p v-if="selectedKind()?.platform === 'antigravity'" class="mb-3 text-sm text-amber-700 dark:text-amber-300">{{ t(selectedKind()?.type === 'apikey' ? 'supplier.accounts.antigravityKeyHint' : 'supplier.accounts.oauthHint') }}</p>
        <div v-if="canSelfAuthorize" class="mb-4 space-y-3 rounded-lg border border-blue-200 p-3 dark:border-blue-700">
          <p class="text-sm">{{ t('supplier.accounts.selfOAuthHint') }}</p>
          <div v-if="selectedKind()?.platform === 'gemini'" class="grid gap-3 sm:grid-cols-2">
            <label><span class="input-label">{{ t('supplier.accounts.geminiOAuthType') }}</span><select v-model="geminiOAuthType" class="input" :disabled="oauthBusy"><option value="code_assist">Code Assist</option><option value="google_one">Google One</option><option value="ai_studio">AI Studio</option></select></label>
            <label><span class="input-label">{{ t('supplier.accounts.geminiTierId') }}</span><input v-model.trim="geminiTierID" class="input" :disabled="oauthBusy" autocomplete="off" /></label>
          </div>
          <OAuthAuthorizationFlow
            :key="kindKey"
            ref="oauthFlow"
            supplier-mode
            :add-method="selectedKind()?.type === 'setup-token' ? 'setup-token' : 'oauth'"
            :platform="selectedKind()!.platform as AccountPlatform"
            :auth-url="oauthAuthURL"
            :session-id="oauthSessionID"
            :loading="oauthBusy"
            :error="formError"
            :show-help="false"
            :show-proxy-warning="false"
            :show-cookie-option="selectedKind()?.platform === 'anthropic'"
            :show-sso-option="selectedKind()?.platform === 'grok'"
            :show-email-password-option="selectedKind()?.platform === 'grok'"
            :show-project-id="selectedKind()?.platform === 'gemini' && geminiOAuthType === 'code_assist'"
            @generate-url="startSupplierOAuth"
            @cookie-auth="secret => completeAlternateOAuth('cookie', secret)"
            @import-sso="secret => completeAlternateOAuth('sso', secret)"
            @authorize-password="secret => completeAlternateOAuth('password', secret)"
          />
          <button v-if="oauthAuthURL && oauthInputMethod === 'manual'" type="button" data-testid="supplier-oauth-complete" class="btn btn-secondary btn-sm" :disabled="saving || oauthBusy || !oauthFlow?.authCode" @click="completeSupplierOAuth">{{ t('supplier.accounts.completeOAuth') }}</button>
          <p v-if="oauthComplete" role="status" class="text-sm text-green-700 dark:text-green-400">{{ t('supplier.accounts.oauthReady') }}</p>
        </div>
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
        <fieldset v-if="!reviewRequired" class="mt-4 space-y-4 rounded-lg border border-gray-200 p-4 dark:border-dark-700">
          <legend class="input-label px-1">{{ t('supplier.accounts.optionalFeatures') }}</legend>
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('supplier.accounts.optionalFeaturesHint') }}</p>
          <label v-if="selectedKind()?.platform === 'openai'" class="block">
            <span class="input-label">{{ t('supplier.accounts.compactMode') }} <code class="text-xs font-normal text-gray-500">openai_compact_mode</code></span>
            <select v-model="compactMode" class="input" data-test="supplier-compact-mode" :disabled="saving">
              <option value="auto">{{ t('supplier.accounts.compactAuto') }}</option>
              <option value="force_on">{{ t('supplier.accounts.compactForceOn') }}</option>
              <option value="force_off">{{ t('supplier.accounts.compactForceOff') }}</option>
            </select>
            <span class="input-hint">{{ t('supplier.accounts.compactModeHint') }}</span>
          </label>
          <label v-if="selectedKind()?.platform === 'anthropic' && selectedKind()?.type === 'apikey'" class="block">
            <span class="input-label">{{ t('supplier.accounts.webSearch') }} <code class="text-xs font-normal text-gray-500">web_search_emulation</code></span>
            <select v-model="webSearchMode" class="input" data-test="supplier-web-search" :disabled="saving">
              <option value="default">{{ t('supplier.accounts.webSearchDefault') }}</option>
              <option value="enabled">{{ t('supplier.accounts.webSearchEnabled') }}</option>
              <option value="disabled">{{ t('supplier.accounts.webSearchDisabled') }}</option>
            </select>
            <span class="input-hint">{{ t('supplier.accounts.webSearchHint') }}</span>
          </label>
          <label class="block">
            <span class="input-label">{{ t('supplier.accounts.upstreamRequestIdHeader') }} <code class="text-xs font-normal text-gray-500">upstream_request_id_header</code></span>
            <input v-model.trim="upstreamRequestIDHeader" class="input" data-test="supplier-request-id-header" maxlength="64" autocomplete="off" :disabled="saving" :placeholder="t('supplier.accounts.upstreamRequestIdHeaderPlaceholder')" />
            <span class="input-hint">{{ t('supplier.accounts.upstreamRequestIdHeaderHint') }}</span>
          </label>
        </fieldset>
        <p v-if="formError" role="alert" class="mt-2 text-sm text-red-600 dark:text-red-400">{{ formError }}</p>
        <p v-else-if="serverError" role="alert" class="mt-2 text-sm text-red-600 dark:text-red-400">{{ serverError }}</p>
      </div>
    </form>

    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="saving" @click="emit('close')">{{ t('common.cancel') }}</button>
      <button type="submit" form="supplier-account-form" class="btn btn-primary" :disabled="saving || oauthBusy || kinds.length === 0 || (!reviewRequired && !selectedGroupIds.length)">
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
import OAuthAuthorizationFlow from '@/components/account/OAuthAuthorizationFlow.vue'
import type { AccountPlatform } from '@/types'
import { formatDateTimeLocalInput, parseDateTimeLocalInput } from '@/utils/format'
import { supplierCredentialTemplate, supplierRequiredCredentials } from '@/features/supplier/accountKinds'
import { exchangeSupplierOAuthCode, exchangeSupplierOAuthCredential, generateSupplierOAuthURL } from '@/api/supplier'
import { extractApiErrorMessage } from '@/utils/apiError'
import type {
  SupplierAccount,
  SupplierAccountInput,
  SupplierAccountKind,
  SupplierAccountUpdateInput,
  SupplierProxyOption,
  SupplierGroupOption,
} from '@/api/supplier'

const props = defineProps<{
  show: boolean
  kinds: SupplierAccountKind[]
  proxies: SupplierProxyOption[]
  reviewRequired?: boolean
  approvedGroups?: Record<string, number[]>
  groupOptions?: SupplierGroupOption[]
  account?: SupplierAccount | null
  saving?: boolean
  serverError?: string
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'submit', accountId: number | null, input: SupplierAccountInput | SupplierAccountUpdateInput): void
}>()

const { t } = useI18n()
const reviewRequired = computed(() => props.reviewRequired ?? false)
const approvedGroups = computed(() => props.approvedGroups ?? {})
const selectedGroupIds = ref<number[]>([])
const availableGroups = computed(() => {
  const kind = selectedKind()
  if (!kind) return []
  const allowed = approvedGroups.value[kind.platform] ?? []
  return allowed.map(id => props.groupOptions?.find(group => group.id === id && group.platform === kind.platform))
    .filter((group): group is SupplierGroupOption => group !== undefined && (!group.require_oauth_only || kind.type === 'oauth'))
})
const selectedGroupNames = computed(() => availableGroups.value.filter(group => selectedGroupIds.value.includes(group.id)).map(group => group.name).join(', '))
const name = ref('')
const externalId = ref('')
const notes = ref('')
const expiresAt = ref('')
const kindKey = ref('')
const credentialsText = ref('')
const compactMode = ref<'auto' | 'force_on' | 'force_off'>('auto')
const webSearchMode = ref<'default' | 'enabled' | 'disabled'>('default')
const upstreamRequestIDHeader = ref('')
const credentialMode = ref<'fields' | 'json'>('fields')
const formError = ref('')
const proxyId = ref(0)
const concurrency = ref(1)
const priority = ref(50)
const loadFactor = ref<number | ''>('')
const autoPauseOnExpired = ref(true)
const oauthBusy = ref(false)
const oauthAuthURL = ref('')
const oauthSessionID = ref('')
const oauthComplete = ref(false)
const oauthFlow = ref<InstanceType<typeof OAuthAuthorizationFlow> | null>(null)
const oauthInputMethod = computed(() => oauthFlow.value?.inputMethod ?? 'manual')
const geminiOAuthType = ref<'code_assist' | 'google_one' | 'ai_studio'>('code_assist')
const geminiTierID = ref('')

const kindValue = (kind: SupplierAccountKind) => `${kind.platform}:${kind.type}`

function selectedKind(): SupplierAccountKind | undefined {
  return props.kinds.find((kind) => kindValue(kind) === kindKey.value)
}

const canSelfAuthorize = computed(() => {
  const kind = selectedKind()
  return kind?.type === 'oauth' || (kind?.platform === 'anthropic' && kind.type === 'setup-token')
})

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
  selectedGroupIds.value = availableGroups.value.slice(0, 1).map(group => group.id)
  compactMode.value = 'auto'
  webSearchMode.value = 'default'
  upstreamRequestIDHeader.value = ''
  credentialsText.value = JSON.stringify(supplierCredentialTemplate(selectedKind()), null, 2)
  formError.value = ''
  oauthAuthURL.value = ''
  oauthSessionID.value = ''
  oauthComplete.value = false
  oauthFlow.value?.reset()
}

async function startSupplierOAuth() {
  if (props.proxies.length && !props.proxies.some(proxy => proxy.id === proxyId.value)) {
    formError.value = t('supplier.accounts.noProxy')
    return
  }
  oauthBusy.value = true
  formError.value = ''
  oauthAuthURL.value = ''
  oauthSessionID.value = ''
  oauthComplete.value = false
  try {
    const kind = selectedKind()!
    const result = await generateSupplierOAuthURL(kind.platform, {
      proxy_id: proxyId.value, type: kind.type as 'oauth' | 'setup-token',
      ...(kind.platform === 'gemini' ? { oauth_type: geminiOAuthType.value, project_id: oauthFlow.value?.projectId || '', tier_id: geminiTierID.value } : {}),
    })
    oauthAuthURL.value = result.auth_url
    oauthSessionID.value = result.session_id
  } catch (error) {
    formError.value = extractApiErrorMessage(error, t('supplier.accounts.oauthFailed'))
  } finally {
    oauthBusy.value = false
  }
}

async function completeSupplierOAuth() {
  oauthBusy.value = true
  formError.value = ''
  try {
    const code = oauthFlow.value?.authCode?.trim() || ''
    const state = oauthFlow.value?.oauthState || new URL(oauthAuthURL.value).searchParams.get('state') || ''
    if (!code || !state || !oauthSessionID.value) throw new Error(t('supplier.accounts.oauthCallbackInvalid'))
    const kind = selectedKind()!
    const selected = kindKey.value
    const tokens = await exchangeSupplierOAuthCode(kind.platform, kind.type as 'oauth' | 'setup-token', oauthSessionID.value, code, state)
    if (!props.show || kindKey.value !== selected) return
    applyOAuthTokens(kind, tokens)
  } catch (error) {
    formError.value = extractApiErrorMessage(error, t('supplier.accounts.oauthFailed'))
  } finally {
    oauthBusy.value = false
  }
}

async function completeAlternateOAuth(method: 'cookie' | 'sso' | 'password', secret: string) {
  const kind = selectedKind()
  if (!kind || props.proxies.length && !props.proxies.some(proxy => proxy.id === proxyId.value)) {
    formError.value = t('supplier.accounts.noProxy')
    return
  }
  const separator = method === 'password' ? secret.indexOf('----') : -1
  const email = separator < 0 ? '' : secret.slice(0, separator).trim()
  const password = separator < 0 ? '' : secret.slice(separator + 4).trim()
  if (method === 'password' && (!email || !password)) { formError.value = t('supplier.accounts.missingCredentials', { fields: 'email, password' }); return }
  oauthBusy.value = true
  formError.value = ''
  const selected = kindKey.value
  try {
    const tokens = await exchangeSupplierOAuthCredential(kind.platform, {
      type: kind.type as 'oauth' | 'setup-token', proxy_id: proxyId.value, method,
      ...(method === 'cookie' ? { session_key: secret } : {}),
      ...(method === 'sso' ? { sso_token: secret } : {}),
      ...(method === 'password' ? { email, password } : {}),
    })
    if (!props.show || kindKey.value !== selected) return
    applyOAuthTokens(kind, tokens)
  } catch (error) {
    formError.value = extractApiErrorMessage(error, t('supplier.accounts.oauthFailed'))
  } finally {
    oauthBusy.value = false
  }
}

function applyOAuthTokens(kind: SupplierAccountKind, tokens: Record<string, unknown>) {
    const credentials = { ...credentialValues.value }
    const fields = kind.platform === 'openai' ? ['access_token', 'refresh_token', 'id_token', 'expires_at', 'email', 'chatgpt_account_id', 'chatgpt_user_id', 'organization_id', 'plan_type', 'subscription_expires_at', 'client_id']
      : kind.platform === 'anthropic' ? ['access_token', 'refresh_token', 'token_type', 'scope', 'expires_at']
      : kind.platform === 'gemini' ? ['access_token', 'refresh_token', 'token_type', 'expires_at', 'scope', 'project_id', 'oauth_type', 'tier_id']
      : kind.platform === 'antigravity' ? ['access_token', 'refresh_token', 'token_type', 'expires_at', 'project_id', 'plan_type']
      : ['access_token', 'refresh_token', 'id_token', 'token_type', 'expires_at', 'client_id', 'scope', 'sub', 'team_id', 'subscription_tier', 'entitlement_status']
    for (const key of fields) {
      if (tokens[key] && !(reviewRequired.value && ['plan_type', 'subscription_expires_at', 'tier_id'].includes(key))) credentials[key] = tokens[key]
    }
    if (typeof tokens.email === 'string' && !credentials.email) credentials.email = tokens.email
    if (kind.platform === 'anthropic' && typeof tokens.email_address === 'string' && !credentials.email) credentials.email = tokens.email_address
    credentialsText.value = JSON.stringify(credentials)
    oauthComplete.value = true
    oauthAuthURL.value = ''
    oauthSessionID.value = ''
    oauthFlow.value?.reset()
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
  selectedGroupIds.value = account?.group_ids?.filter(id => availableGroups.value.some(group => group.id === id)) ?? availableGroups.value.slice(0, 1).map(group => group.id)
  credentialsText.value = account ? '' : JSON.stringify(supplierCredentialTemplate(selectedKind()), null, 2)
  const savedCompactMode = account?.extra?.openai_compact_mode
  compactMode.value = savedCompactMode === 'force_on' || savedCompactMode === 'force_off' ? savedCompactMode : 'auto'
  const savedWebSearchMode = account?.extra?.web_search_emulation
  webSearchMode.value = savedWebSearchMode === 'enabled' || savedWebSearchMode === 'disabled' ? savedWebSearchMode : 'default'
  upstreamRequestIDHeader.value = typeof account?.extra?.upstream_request_id_header === 'string' ? account.extra.upstream_request_id_header : ''
  credentialMode.value = 'fields'
  formError.value = ''
  proxyId.value = account?.proxy_id ?? props.proxies[0]?.id ?? 0
  concurrency.value = account?.concurrency ?? 1
  priority.value = account?.priority ?? 50
  loadFactor.value = account?.load_factor ?? ''
  autoPauseOnExpired.value = account?.auto_pause_on_expired ?? true
  oauthAuthURL.value = ''
  oauthSessionID.value = ''
  oauthComplete.value = false
  oauthFlow.value?.reset()
  geminiOAuthType.value = 'code_assist'
  geminiTierID.value = ''
}

watch(proxyId, () => {
  oauthAuthURL.value = ''
  oauthSessionID.value = ''
  oauthFlow.value?.reset()
})

watch([geminiOAuthType, geminiTierID], () => {
  oauthAuthURL.value = ''
  oauthSessionID.value = ''
  oauthFlow.value?.reset()
})

watch(() => [props.show, props.account, props.kinds, props.proxies] as const, ([show]) => {
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
  const configurationOnly = props.account && !reviewRequired.value && Object.keys(credentials).length > 0 && Object.keys(credentials).every(key => ['model_mapping', 'compact_model_mapping', 'protocol_rules'].includes(key))
  const missing = (configurationOnly ? [] : supplierRequiredCredentials(kind, credentials)).filter(field => {
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
    if (props.proxies.length && !props.proxies.some(proxy => proxy.id === proxyId.value)) throw new Error(t('supplier.accounts.noProxy'))
    const operational = reviewRequired.value ? {} : {
      concurrency: concurrency.value,
      priority: priority.value,
      ...(loadFactor.value === '' ? {} : { load_factor: loadFactor.value }),
      auto_pause_on_expired: autoPauseOnExpired.value,
    }
    const connection = proxyId.value ? { proxy_id: proxyId.value } : {}
    const credentials = props.account && Object.keys(credentialValues.value).length === 0 && credentialMode.value === 'fields' ? undefined : parseCredentials()
    const kind = selectedKind()
    const extra: Record<string, string> = {}
    if (!reviewRequired.value) {
      if (kind?.platform === 'openai' && compactMode.value !== 'auto') extra.openai_compact_mode = compactMode.value
      if (kind?.platform === 'anthropic' && kind.type === 'apikey' && webSearchMode.value !== 'default') extra.web_search_emulation = webSearchMode.value
      if (upstreamRequestIDHeader.value) extra.upstream_request_id_header = upstreamRequestIDHeader.value
    }
    const extraPayload = !reviewRequired.value && (Object.keys(extra).length > 0 || props.account?.extra && Object.keys(props.account.extra).length > 0) ? { extra } : {}
    const expiry = reviewRequired.value ? null : parseDateTimeLocalInput(expiresAt.value)
    if (props.account) {
      const input: SupplierAccountUpdateInput = {
        ...operational,
        ...extraPayload,
        ...connection,
        name: name.value,
        external_id: externalId.value,
        ...(!reviewRequired.value ? { notes: notes.value || null, expires_at: expiry, group_ids: selectedGroupIds.value } : {}),
      }
      if (credentials) input.credentials = credentials
      emit('submit', props.account.id, input)
      return
    }
    if (!kind || !credentials) throw new Error(t('supplier.accounts.invalidCredentials'))
    emit('submit', null, {
      ...operational,
      ...extraPayload,
      ...connection,
      name: name.value,
      external_id: externalId.value,
      ...(!reviewRequired.value ? { notes: notes.value || null, expires_at: expiry, group_ids: selectedGroupIds.value } : {}),
      platform: kind.platform,
      type: kind.type,
      credentials,
    })
  } catch (error) {
    formError.value = error instanceof Error ? error.message : t('supplier.accounts.invalidCredentials')
  }
}
</script>
