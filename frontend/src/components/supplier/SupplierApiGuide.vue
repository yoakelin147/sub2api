<template>
  <section id="api-guide" class="card overflow-hidden">
    <div class="border-b border-gray-100 p-5 dark:border-dark-700">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('supplier.guide.title') }}</h3>
      <p class="mt-1 text-sm text-gray-500">{{ t('supplier.guide.description') }}</p>
      <ol class="mt-4 grid gap-3 text-sm text-gray-600 dark:text-gray-300 sm:grid-cols-3">
        <li v-for="step in ['prepare', 'submit', 'review']" :key="step" class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">{{ t(`supplier.guide.steps.${step}`) }}</li>
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
        <label><span class="input-label">{{ t('supplier.guide.exampleKind') }}</span><select v-model="kindKey" class="input" :disabled="!isWrite" data-test="guide-kind"><option v-for="item in SUPPLIER_ACCOUNT_KINDS" :key="item.platform + ':' + item.type" :value="item.platform + ':' + item.type">{{ t(`monitorCommon.providers.${item.platform}`) }} / {{ t(`supplier.admin.kindTypes.${item.type}`) }}</option></select></label>
        <label><span class="input-label">{{ t('supplier.guide.language') }}</span><select v-model="language" class="input" data-test="guide-language"><option value="bash">cURL / Bash</option><option value="powershell">PowerShell</option></select></label>
      </div>
      <p class="text-xs leading-5 text-gray-500">{{ t('supplier.guide.exampleHint') }}</p>
      <div class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
        <div class="flex items-center justify-between bg-gray-50 px-4 py-2 dark:bg-dark-800">
          <span class="text-xs font-medium text-gray-500">{{ isWrite ? 'POST' : 'GET' }} {{ endpoint }}</span>
          <button class="btn btn-ghost btn-sm" @click="copyExample"><Icon name="copy" size="sm" />{{ t('supplier.guide.copyExample') }}</button>
        </div>
        <pre class="max-h-80 overflow-auto bg-gray-950 p-4 text-xs leading-6 text-gray-100" tabindex="0" :aria-label="t('supplier.guide.codeExample')"><code>{{ example }}</code></pre>
      </div>
      <p class="rounded-lg bg-primary-50 p-3 text-sm leading-6 text-primary-800 dark:bg-primary-900/20 dark:text-primary-200">{{ t(`supplier.guide.results.${operation}`) }}</p>
      <details class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <summary class="cursor-pointer text-sm font-medium text-gray-900 dark:text-white">{{ t('supplier.guide.rulesTitle') }}</summary>
        <ul class="mt-3 list-disc space-y-2 pl-5 text-sm leading-6 text-gray-600 dark:text-gray-300"><li v-for="rule in ['fields', 'idempotency', 'batch', 'url', 'permissions', 'credentials']" :key="rule">{{ t(`supplier.guide.rules.${rule}`) }}</li></ul>
      </details>
      <details class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <summary class="cursor-pointer text-sm font-medium text-gray-900 dark:text-white">{{ t('supplier.guide.errorsTitle') }}</summary>
        <dl class="mt-3 space-y-3 text-sm"><div v-for="code in ['400', '401', '403', '409', '422', '429']" :key="code" class="flex gap-3"><dt class="font-mono text-gray-900 dark:text-white">{{ code }}</dt><dd class="text-gray-600 dark:text-gray-300">{{ t(`supplier.guide.errors.${code}`) }}</dd></div></dl>
      </details>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import { getAPIBaseURL } from '@/api/url'
import { SUPPLIER_ACCOUNT_KINDS, supplierCredentialTemplate } from '@/features/supplier/accountKinds'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()
const operations = ['verify', 'create', 'batch', 'list'] as const
const operation = ref<typeof operations[number]>('verify')
const language = ref('bash')
const kindKey = ref('openai:apikey')
const kind = computed(() => SUPPLIER_ACCOUNT_KINDS.find(item => item.platform + ':' + item.type === kindKey.value)!)
const apiBase = new URL(getAPIBaseURL(), window.location.origin).href.replace(/\/$/, '')
const isWrite = computed(() => operation.value === 'create' || operation.value === 'batch')
const endpoint = computed(() => operation.value === 'verify' ? '/supplier/me' : operation.value === 'batch' ? '/supplier/accounts/batch' : operation.value === 'list' ? '/supplier/accounts?page=1&page_size=20' : '/supplier/accounts')
const example = computed(() => {
  const credentials = Object.fromEntries(Object.entries(supplierCredentialTemplate(kind.value)).map(([key, value]) => [key, value === '' ? 'REPLACE_WITH_' + key.toUpperCase() : value === 'https://' ? 'https://YOUR_ALLOWED_UPSTREAM_HOST' : value]))
  if (kind.value.type === 'service_account') credentials.service_account_json = JSON.stringify({ project_id: 'YOUR_PROJECT_ID', client_email: 'YOUR_SERVICE_ACCOUNT_EMAIL', private_key: 'REPLACE_WITH_PEM_PRIVATE_KEY' })
  const account = { external_id: 'ext-10001', name: 'Account 10001', platform: kind.value.platform, type: kind.value.type, credentials }
  const body = JSON.stringify(operation.value === 'batch' ? { accounts: [account, { ...account, external_id: 'ext-10002', name: 'Account 10002' }] } : account, null, 2)
  const requestKey = kind.value.platform + '-' + kind.value.type + '-' + operation.value + '-10001-v1'
  if (language.value === 'powershell') {
    return [
      "$API_BASE = '" + apiBase.replace(/'/g, "''") + "'",
      "$SUPPLIER_TOKEN = 'REPLACE_WITH_FULL_SUPPLIER_TOKEN'",
      "$headers = @{ 'x-api-key' = $SUPPLIER_TOKEN }",
      ...(isWrite.value ? ["$headers['Idempotency-Key'] = '" + requestKey + "'", "$body = @'", body, "'@"] : []),
      'Invoke-RestMethod -Method ' + (isWrite.value ? 'Post' : 'Get') + ' -Uri "$API_BASE' + endpoint.value + '" -Headers $headers' + (isWrite.value ? " -ContentType 'application/json' -Body ([System.Text.Encoding]::UTF8.GetBytes($body))" : ''),
    ].join('\n')
  }
  const quote = (value: string) => "'" + value.replace(/'/g, "'\\''") + "'"
  return [
    'API_BASE=' + quote(apiBase),
    "SUPPLIER_TOKEN='REPLACE_WITH_FULL_SUPPLIER_TOKEN'",
    'curl --fail-with-body -sS -X ' + (isWrite.value ? 'POST' : 'GET') + ' "$API_BASE' + endpoint.value + '" \\',
    '  -H "x-api-key: $SUPPLIER_TOKEN"' + (isWrite.value ? ' \\' : ''),
    ...(isWrite.value ? ['  -H ' + quote('Idempotency-Key: ' + requestKey) + ' \\', "  -H 'Content-Type: application/json' \\", '  --data-raw ' + quote(body)] : []),
  ].join('\n')
})

async function copyExample() {
  try {
    await navigator.clipboard.writeText(example.value)
    appStore.showSuccess(t('common.copiedToClipboard'))
  } catch { appStore.showError(t('common.copyFailed')) }
}
</script>
