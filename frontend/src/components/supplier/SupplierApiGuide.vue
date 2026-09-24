<template>
  <section id="api-guide" class="card overflow-hidden">
    <div class="border-b border-gray-100 p-5 dark:border-dark-700">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('supplier.guide.title') }}</h3>
      <p class="mt-1 text-sm text-gray-500">{{ t('supplier.guide.description') }}</p>
      <ol class="mt-4 grid gap-3 text-sm text-gray-600 dark:text-gray-300 sm:grid-cols-2">
        <li v-for="step in ['prepare', 'submit']" :key="step" class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">{{ t(`supplier.guide.steps.${step}`) }}</li>
      </ol>
    </div>
    <div class="space-y-4 p-5">
      <details class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <summary class="cursor-pointer text-sm font-medium text-gray-900 dark:text-white">{{ t('supplier.guide.webTitle') }}</summary>
        <p class="mt-3 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t('supplier.guide.webSteps') }}</p>
        <RouterLink to="/supplier/accounts" class="mt-2 inline-block text-sm text-primary-600 hover:underline">{{ t('supplier.accounts.title') }} →</RouterLink>
      </details>
      <div>
        <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('supplier.guide.apiAddress') }}</p>
        <code class="mt-1 block break-all text-sm text-primary-700 dark:text-primary-300">{{ apiBase }}</code>
        <p class="mt-2 text-sm text-gray-500">{{ t('supplier.guide.authHint') }}</p>
      </div>
      <div class="grid items-end gap-3 sm:grid-cols-3">
        <label><span class="input-label">{{ t('supplier.guide.operation') }}</span><select v-model="operation" class="input" data-test="guide-operation"><option v-for="item in operations" :key="item" :value="item">{{ t(`supplier.guide.operations.${item}`) }}</option></select></label>
        <label><span class="input-label">{{ t('supplier.guide.exampleKind') }}</span><select v-model="kindKey" class="input" :disabled="!isWrite && !isOAuthOperation" data-test="guide-kind"><option v-for="item in selectableKinds" :key="item.platform + ':' + item.type" :value="item.platform + ':' + item.type">{{ t(`monitorCommon.providers.${item.platform}`) }} / {{ t(`supplier.admin.kindTypes.${item.type}`) }}</option></select></label>
        <label><span class="input-label">{{ t('supplier.guide.language') }}</span><select v-model="language" class="input" data-test="guide-language"><option value="bash">cURL / Bash</option><option value="powershell">PowerShell</option></select></label>
      </div>
      <label v-if="operation === 'oauthCredential' && kind.platform === 'grok'" class="block"><span class="input-label">{{ t('supplier.guide.credentialMethod') }}</span><select v-model="grokMethod" class="input" data-test="guide-credential-method"><option value="sso">SSO</option><option value="password">{{ t('supplier.guide.passwordMethod') }}</option></select></label>
      <p class="text-xs leading-5 text-gray-500">{{ t('supplier.guide.exampleHint') }}</p>
      <div class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
        <div class="flex items-center justify-between bg-gray-50 px-4 py-2 dark:bg-dark-800">
          <span class="text-xs font-medium text-gray-500">{{ method }} {{ endpoint }}</span>
          <button class="btn btn-ghost btn-sm" @click="copyExample"><Icon name="copy" size="sm" />{{ t('supplier.guide.copyExample') }}</button>
        </div>
        <pre class="max-h-80 overflow-auto bg-gray-950 p-4 text-xs leading-6 text-gray-100" tabindex="0" :aria-label="t('supplier.guide.codeExample')"><code>{{ example }}</code></pre>
      </div>
      <p class="rounded-lg bg-primary-50 p-3 text-sm leading-6 text-primary-800 dark:bg-primary-900/20 dark:text-primary-200">{{ t(`supplier.guide.results.${operation}`) }}</p>
      <details class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <summary class="cursor-pointer text-sm font-medium text-gray-900 dark:text-white">{{ t('supplier.guide.rulesTitle') }}</summary>
        <ul class="mt-3 list-disc space-y-2 pl-5 text-sm leading-6 text-gray-600 dark:text-gray-300"><li v-for="rule in ['fields', 'idempotency', 'batch', 'url', 'permissions', 'credentials']" :key="rule">{{ t(`supplier.guide.rules.${rule}`) }}</li></ul>
      </details>
      <details v-if="profile && !profile.review_required" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700" data-test="supplier-extra-guide">
        <summary class="cursor-pointer text-sm font-medium text-gray-900 dark:text-white">{{ t('supplier.guide.extraTitle') }}</summary>
        <p class="mt-3 text-sm text-gray-600 dark:text-gray-300">{{ t('supplier.guide.extraIntro') }}</p>
        <dl class="mt-3 space-y-3 text-sm text-gray-600 dark:text-gray-300">
          <div><dt class="font-mono text-gray-900 dark:text-white">openai_compact_mode <span class="text-xs">(auto / force_on / force_off)</span></dt><dd>{{ t('supplier.guide.extraCompact') }}</dd></div>
          <div><dt class="font-mono text-gray-900 dark:text-white">web_search_emulation <span class="text-xs">(default / enabled / disabled)</span></dt><dd>{{ t('supplier.guide.extraWebSearch') }}</dd></div>
          <div><dt class="font-mono text-gray-900 dark:text-white">upstream_request_id_header</dt><dd>{{ t('supplier.guide.extraRequestId') }}</dd></div>
        </dl>
      </details>
      <details class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <summary class="cursor-pointer text-sm font-medium text-gray-900 dark:text-white">{{ t('supplier.guide.errorsTitle') }}</summary>
        <dl class="mt-3 space-y-3 text-sm"><div v-for="code in ['400', '401', '403', '409', '422', '429']" :key="code" class="flex gap-3"><dt class="font-mono text-gray-900 dark:text-white">{{ code }}</dt><dd class="text-gray-600 dark:text-gray-300">{{ t(`supplier.guide.errors.${code}`) }}</dd></div></dl>
      </details>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import { getAPIBaseURL } from '@/api/url'
import { SUPPLIER_ACCOUNT_KINDS, supplierCredentialTemplate } from '@/features/supplier/accountKinds'
import { useAppStore } from '@/stores/app'
import type { SupplierProfile } from '@/api/supplier'

const props = defineProps<{ profile?: SupplierProfile | null }>()

const { t } = useI18n()
const appStore = useAppStore()
const operations = ['verify', 'proxies', 'oauthUrl', 'oauthExchange', 'oauthCredential', 'create', 'batch', 'update', 'list'] as const
const operation = ref<typeof operations[number]>('verify')
const language = ref('bash')
const kindKey = ref('openai:apikey')
const grokMethod = ref<'sso' | 'password'>('sso')
const kind = computed(() => SUPPLIER_ACCOUNT_KINDS.find(item => item.platform + ':' + item.type === kindKey.value)!)
const isOAuthOperation = computed(() => operation.value === 'oauthUrl' || operation.value === 'oauthExchange' || operation.value === 'oauthCredential')
const selectableKinds = computed(() => isOAuthOperation.value ? SUPPLIER_ACCOUNT_KINDS.filter(item => (item.type === 'oauth' || (item.platform === 'anthropic' && item.type === 'setup-token')) && (operation.value !== 'oauthCredential' || item.platform === 'anthropic' || item.platform === 'grok')) : SUPPLIER_ACCOUNT_KINDS)
watch(operation, () => { if (isOAuthOperation.value && !selectableKinds.value.some(item => item.platform + ':' + item.type === kindKey.value)) kindKey.value = operation.value === 'oauthCredential' ? 'anthropic:oauth' : 'openai:oauth' })
const apiBase = new URL(getAPIBaseURL(), window.location.origin).href.replace(/\/$/, '')
const isWrite = computed(() => operation.value === 'create' || operation.value === 'batch' || operation.value === 'update')
const hasBody = computed(() => isWrite.value || isOAuthOperation.value)
const method = computed(() => operation.value === 'update' ? 'PUT' : hasBody.value ? 'POST' : 'GET')
const endpoint = computed(() => ({
  verify: '/supplier/me',
  proxies: '/supplier/proxies',
  oauthUrl: `/supplier/oauth/${kind.value.platform}/auth-url`,
  oauthExchange: `/supplier/oauth/${kind.value.platform}/exchange-code`,
  oauthCredential: `/supplier/oauth/${kind.value.platform}/credential-exchange`,
  create: '/supplier/accounts',
  batch: '/supplier/accounts/batch',
  update: '/supplier/accounts/123',
  list: '/supplier/accounts?page=1&page_size=20',
})[operation.value])
const example = computed(() => {
  const credentials = Object.fromEntries(Object.entries(supplierCredentialTemplate(kind.value)).map(([key, value]) => [key, value === '' ? 'REPLACE_WITH_' + key.toUpperCase() : value === 'https://' ? 'https://YOUR_ALLOWED_UPSTREAM_HOST' : value]))
  if (kind.value.type === 'oauth' || kind.value.type === 'setup-token') delete credentials.refresh_token
  if (kind.value.type === 'service_account') credentials.service_account_json = JSON.stringify({ project_id: 'YOUR_PROJECT_ID', client_email: 'YOUR_SERVICE_ACCOUNT_EMAIL', private_key: 'REPLACE_WITH_PEM_PRIVATE_KEY' })
  if (props.profile && !props.profile.review_required) credentials.model_mapping = { 'YOUR_MODEL': 'UPSTREAM_MODEL' }
  const operational = props.profile && !props.profile.review_required ? {
    group_ids: props.profile.auto_approve_groups?.[kind.value.platform]?.slice() || ['REPLACE_WITH_AUTHORIZED_GROUP_ID'],
    concurrency: 1, priority: 50, auto_pause_on_expired: true,
  } : {}
  const advanced = props.profile && !props.profile.review_required && kind.value.platform === 'openai' ? { extra: { openai_compact_mode: 'auto' } } : {}
  const account = { name: 'Account 10001', platform: kind.value.platform, type: kind.value.type, proxy_id: 123, credentials, ...operational, ...advanced }
  const update = { name: 'Account 10001', proxy_id: 123, ...operational, ...advanced, ...(props.profile && !props.profile.review_required ? { credentials: { model_mapping: { YOUR_MODEL: 'UPSTREAM_MODEL' } } } : {}) }
  const requestBody = operation.value === 'oauthUrl' ? { proxy_id: 123, type: kind.value.type, ...(kind.value.platform === 'gemini' ? { oauth_type: 'code_assist' } : {}) }
    : operation.value === 'oauthExchange' ? { type: kind.value.type, session_id: 'REPLACE_WITH_SESSION_ID', code: 'REPLACE_WITH_AUTH_CODE', state: 'REPLACE_WITH_STATE' }
    : operation.value === 'oauthCredential' ? { type: kind.value.type, proxy_id: 123, ...(kind.value.platform === 'anthropic' ? { method: 'cookie', session_key: 'REPLACE_WITH_SESSION_KEY' } : grokMethod.value === 'sso' ? { method: 'sso', sso_token: 'REPLACE_WITH_SSO_TOKEN' } : { method: 'password', email: 'REPLACE_WITH_EMAIL', password: 'REPLACE_WITH_PASSWORD' }) }
    : operation.value === 'batch' ? { accounts: [account, { ...account, name: 'Account 10002' }] }
      : operation.value === 'update' ? update : account
  const body = JSON.stringify(requestBody, null, 2)
  const requestKey = kind.value.platform + '-' + kind.value.type + '-' + operation.value + '-10001-v1'
  if (language.value === 'powershell') {
    return [
      "$API_BASE = '" + apiBase.replace(/'/g, "''") + "'",
      "$SUPPLIER_TOKEN = 'REPLACE_WITH_FULL_SUPPLIER_TOKEN'",
      "$headers = @{ 'x-api-key' = $SUPPLIER_TOKEN }",
      ...(operation.value === 'create' || operation.value === 'batch' ? ["$headers['Idempotency-Key'] = '" + requestKey + "'"] : []),
      ...(hasBody.value ? ["$body = @'", body, "'@"] : []),
      'Invoke-RestMethod -Method ' + (method.value === 'GET' ? 'Get' : method.value === 'PUT' ? 'Put' : 'Post') + ' -Uri "$API_BASE' + endpoint.value + '" -Headers $headers' + (hasBody.value ? " -ContentType 'application/json' -Body ([System.Text.Encoding]::UTF8.GetBytes($body))" : ''),
    ].join('\n')
  }
  const quote = (value: string) => "'" + value.replace(/'/g, "'\\''") + "'"
  return [
    'API_BASE=' + quote(apiBase),
    "SUPPLIER_TOKEN='REPLACE_WITH_FULL_SUPPLIER_TOKEN'",
    'curl --fail-with-body -sS -X ' + method.value + ' "$API_BASE' + endpoint.value + '" \\',
    '  -H "x-api-key: $SUPPLIER_TOKEN"' + (hasBody.value ? ' \\' : ''),
    ...(isWrite.value ? [
      ...(operation.value === 'update' ? [] : ['  -H ' + quote('Idempotency-Key: ' + requestKey) + ' \\']),
      '  -H ' + quote('Content-Type: application/json') + ' \\',
      '  --data-raw ' + quote(body),
    ] : []),
    ...(!isWrite.value && hasBody.value ? ["  -H 'Content-Type: application/json' \\", '  --data-raw ' + quote(body)] : []),
  ].join('\n')
})

async function copyExample() {
  try {
    await navigator.clipboard.writeText(example.value)
    appStore.showSuccess(t('common.copiedToClipboard'))
  } catch { appStore.showError(t('common.copyFailed')) }
}
</script>
