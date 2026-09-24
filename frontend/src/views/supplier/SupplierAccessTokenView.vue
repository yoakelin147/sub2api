<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <header>
        <h2 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('supplier.token.title') }}</h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('supplier.token.description') }}</p>
      </header>

      <div v-if="loading" class="card p-8" aria-busy="true">
        <div class="h-24 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-800"></div>
      </div>

      <section v-else class="card space-y-5 p-6">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <span :class="['badge', status?.exists ? 'badge-success' : 'badge-gray']">
              {{ status?.exists ? t('supplier.token.exists') : t('supplier.token.missing') }}
            </span>
            <dl v-if="status?.exists" class="mt-4 grid gap-4 text-sm sm:grid-cols-3">
              <div>
                <dt class="text-gray-500 dark:text-gray-400">{{ t('supplier.token.masked') }}</dt>
                <dd class="mt-1 font-mono text-gray-900 dark:text-white">{{ status.masked_key }}</dd>
              </div>
              <div>
                <dt class="text-gray-500 dark:text-gray-400">{{ t('supplier.token.createdAt') }}</dt>
                <dd class="mt-1 text-gray-900 dark:text-white">{{ formatDate(status.created_at) }}</dd>
              </div>
              <div>
                <dt class="text-gray-500 dark:text-gray-400">{{ t('supplier.token.lastUsedAt') }}</dt>
                <dd class="mt-1 text-gray-900 dark:text-white">{{ formatDate(status.last_used_at) }}</dd>
              </div>
            </dl>
          </div>
          <div class="flex flex-wrap gap-2">
            <button class="btn btn-primary" :disabled="working" @click="generate">
              <Icon name="refresh" size="sm" />
              {{ status?.exists ? t('supplier.token.rotate') : t('supplier.token.generate') }}
            </button>
            <button v-if="status?.exists" class="btn btn-danger" :disabled="working" @click="revoke">
              {{ t('supplier.token.revoke') }}
            </button>
          </div>
        </div>
      </section>

      <section v-if="plaintextToken" class="rounded-xl border-2 border-amber-300 bg-amber-50 p-6 dark:border-amber-700 dark:bg-amber-900/20" aria-live="assertive">
        <h3 class="font-semibold text-amber-900 dark:text-amber-200">{{ t('supplier.token.oneTimeTitle') }}</h3>
        <p class="mt-1 text-sm text-amber-800 dark:text-amber-300">{{ t('supplier.token.oneTimeHint') }}</p>
        <div class="mt-4 flex flex-col gap-2 sm:flex-row">
          <code data-testid="plaintext-token" class="min-w-0 flex-1 break-all rounded-lg bg-white p-3 text-sm text-gray-900 ring-1 ring-amber-200 dark:bg-dark-900 dark:text-white dark:ring-amber-800">{{ plaintextToken }}</code>
          <button class="btn btn-secondary" @click="copyToken"><Icon name="copy" size="sm" />{{ t('common.copy') }}</button>
        </div>
      </section>
      <SupplierApiGuide :profile="profile" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import SupplierApiGuide from '@/components/supplier/SupplierApiGuide.vue'
import { useAppStore } from '@/stores/app'
import {
  getAccessTokenStatus,
  getProfile,
  regenerateAccessToken,
  revokeAccessToken,
  type SupplierTokenStatus,
  type SupplierProfile,
} from '@/api/supplier'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(true)
const working = ref(false)
const status = ref<SupplierTokenStatus | null>(null)
const profile = ref<SupplierProfile | null>(null)
const plaintextToken = ref('')

const formatDate = (value: string | null | undefined) => value ? new Date(value).toLocaleString() : t('common.notAvailable')

async function load() {
  loading.value = true
  try {
    status.value = await getAccessTokenStatus()
    profile.value = await getProfile()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('supplier.token.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function generate() {
  if (status.value?.exists && !window.confirm(t('supplier.token.confirmRotate'))) return
  working.value = true
  try {
    const result = await regenerateAccessToken()
    plaintextToken.value = result.key
    appStore.showSuccess(t('supplier.token.rotated'))
    await load()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('common.unknownError'))
  } finally {
    working.value = false
  }
}

async function revoke() {
  if (!window.confirm(t('supplier.token.confirmRevoke'))) return
  working.value = true
  try {
    await revokeAccessToken()
    plaintextToken.value = ''
    appStore.showSuccess(t('supplier.token.revoked'))
    await load()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('common.unknownError'))
  } finally {
    working.value = false
  }
}

async function copyToken() {
  try {
    await navigator.clipboard.writeText(plaintextToken.value)
    appStore.showSuccess(t('common.copiedToClipboard'))
  } catch {
    appStore.showError(t('common.copyFailed'))
  }
}

onMounted(load)
</script>
