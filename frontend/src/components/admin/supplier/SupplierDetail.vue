<template>
  <div class="min-w-0 space-y-4">
    <section class="card p-4 sm:p-5">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2">
            <h2 class="break-all text-xl font-semibold text-gray-900 dark:text-white">{{ supplier.name }}</h2>
            <span :class="supplier.status === 'active' ? 'badge badge-success' : 'badge badge-danger'">{{ supplier.status === 'active' ? t('common.active') : t('common.disabled') }}</span>
          </div>
          <p class="mt-1 break-all font-mono text-xs text-gray-500">{{ supplier.code }} · #{{ supplier.id }}</p>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-secondary btn-sm" @click="emit('edit')"><Icon name="edit" size="sm" />{{ t('common.edit') }}</button>
          <button class="btn btn-ghost btn-sm text-red-600" @click="emit('delete')"><Icon name="trash" size="sm" />{{ t('common.delete') }}</button>
        </div>
      </div>
      <dl class="mt-4 grid grid-cols-2 gap-4 border-t border-gray-100 pt-4 dark:border-dark-700 sm:grid-cols-4">
        <div v-for="item in stats" :key="item.label">
          <dt class="text-xs text-gray-500 dark:text-gray-400">{{ item.label }}</dt>
          <dd class="mt-1 text-xl font-semibold tabular-nums text-gray-900 dark:text-white">{{ item.value }}</dd>
        </div>
      </dl>
    </section>
    <section class="card min-w-0 overflow-hidden">
      <div role="tablist" :aria-label="t('supplier.admin.details')" class="flex overflow-x-auto border-b border-gray-200 px-4 dark:border-dark-700">
        <button v-for="(tab, index) in tabs" :id="'supplier-tab-' + tab.value" :key="tab.value"
          type="button" role="tab" :aria-selected="activeTab === tab.value" :aria-controls="'supplier-panel-' + tab.value"
          :tabindex="activeTab === tab.value ? 0 : -1"
          class="flex shrink-0 items-center gap-2 whitespace-nowrap border-b-2 px-4 py-3 text-sm font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-500"
          :class="activeTab === tab.value ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'"
          @click="activeTab = tab.value" @keydown="navigateTab($event, index)">
          {{ tab.label }}<span v-if="tab.count !== undefined" class="badge badge-gray">{{ tab.count }}</span>
        </button>
      </div>
      <div v-for="tab in tabs" v-show="activeTab === tab.value" :id="'supplier-panel-' + tab.value" :key="tab.value" role="tabpanel" :aria-labelledby="'supplier-tab-' + tab.value" tabindex="0">
        <SupplierAccountsPanel v-if="tab.value === 'accounts' && activeTab === 'accounts'" :supplier-id="supplier.id" @changed="emit('changed')" />
        <SupplierMembersPanel v-if="tab.value === 'members' && activeTab === 'members'" :supplier-id="supplier.id" @changed="emit('changed')" />
        <div v-if="tab.value === 'settings'" class="grid gap-6 p-4 sm:p-5 lg:grid-cols-2">
          <section class="min-w-0">
            <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('supplier.admin.basic') }}</h3>
            <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ supplier.review_required ? t('supplier.admin.reviewRequired') : t('supplier.admin.autoApproveGroups') }}</p>
            <p v-if="supplier.notes" class="mt-2 max-h-24 overflow-auto break-words text-sm text-gray-600 dark:text-gray-300">{{ supplier.notes }}</p>
            <p class="mb-2 mt-4 text-sm text-gray-500">{{ t('supplier.admin.allowedKinds') }}</p>
            <div class="flex max-h-48 flex-wrap gap-2 overflow-auto">
              <span v-for="kind in supplier.allowed_account_kinds ?? []" :key="kind.platform + ':' + kind.type" class="badge badge-gray">{{ t('monitorCommon.providers.' + kind.platform) }} / {{ t('supplier.admin.kindTypes.' + kind.type) }}</span>
              <p v-if="!supplier.allowed_account_kinds?.length" class="text-sm text-amber-600">{{ t('supplier.admin.noPermission') }}</p>
            </div>
          </section>
          <section class="min-w-0 border-t border-gray-100 pt-4 dark:border-dark-700 lg:border-l lg:border-t-0 lg:pl-6 lg:pt-0">
            <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('supplier.admin.token') }}</h3>
            <p class="mt-2 text-sm text-gray-500">{{ t('supplier.token.description') }}</p>
            <p class="mt-3 break-all font-mono text-sm text-gray-700 dark:text-gray-300">{{ tokenStatus.exists ? tokenStatus.masked_key : t('supplier.token.missing') }}</p>
            <p v-if="tokenStatus.last_used_at" class="mt-1 text-xs text-gray-500">{{ t('supplier.token.lastUsedAt') }}: {{ new Date(tokenStatus.last_used_at).toLocaleString() }}</p>
            <div class="mt-4 flex flex-wrap gap-2">
              <button class="btn btn-secondary btn-sm" :disabled="working" @click="generateToken">{{ tokenStatus.exists ? t('supplier.token.rotate') : t('supplier.token.generate') }}</button>
              <button v-if="tokenStatus.exists" class="btn btn-danger btn-sm" :disabled="working" @click="revokeToken">{{ t('supplier.token.revoke') }}</button>
            </div>
            <div v-if="plaintextToken" class="mt-4 rounded-lg border border-amber-300 bg-amber-50 p-3 dark:border-amber-700 dark:bg-amber-900/20">
              <p class="text-sm font-medium text-amber-900 dark:text-amber-200">{{ t('supplier.token.oneTimeTitle') }}</p>
              <p class="mt-1 text-xs text-gray-500">{{ t('supplier.token.oneTimeHint') }}</p>
              <code class="mt-2 block break-all text-sm">{{ plaintextToken }}</code>
              <button class="btn btn-secondary btn-sm mt-3" @click="copyToken"><Icon name="copy" size="sm" />{{ t('common.copy') }}</button>
            </div>
          </section>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import SupplierAccountsPanel from './SupplierAccountsPanel.vue'
import SupplierMembersPanel from './SupplierMembersPanel.vue'
import { useAppStore } from '@/stores/app'
import suppliersAPI, { type AdminSupplier } from '@/api/admin/suppliers'

const props = defineProps<{ supplier: AdminSupplier }>()
const emit = defineEmits<{ (event: 'edit'): void; (event: 'delete'): void; (event: 'changed'): void }>()
const { t } = useI18n()
const appStore = useAppStore()
const activeTab = ref('accounts')
const working = ref(false)
const plaintextToken = ref('')
const tokenStatus = ref(props.supplier.access_token)
let disposed = false
onBeforeUnmount(() => { disposed = true; plaintextToken.value = '' })

const tabs = computed(() => [
  { value: 'accounts', label: t('supplier.admin.accounts'), count: props.supplier.stats.account_count },
  { value: 'members', label: t('supplier.admin.members'), count: props.supplier.stats.member_count },
  { value: 'settings', label: t('supplier.admin.settings') },
])
const stats = computed(() => [
  { label: t('supplier.admin.accountCount'), value: props.supplier.stats.account_count },
  { label: t('supplier.accounts.pendingCount'), value: props.supplier.stats.pending_count },
  { label: t('supplier.accounts.schedulableCount'), value: props.supplier.stats.schedulable_count },
  { label: t('supplier.accounts.errorCount'), value: props.supplier.stats.error_count },
])
watch(() => props.supplier.access_token, value => { tokenStatus.value = value })

async function navigateTab(event: KeyboardEvent, index: number) {
  let next = index
  if (event.key === 'ArrowRight') next = (index + 1) % tabs.value.length
  else if (event.key === 'ArrowLeft') next = (index + tabs.value.length - 1) % tabs.value.length
  else if (event.key === 'Home') next = 0
  else if (event.key === 'End') next = tabs.value.length - 1
  else return
  event.preventDefault()
  activeTab.value = tabs.value[next].value
  await nextTick()
  document.getElementById('supplier-tab-' + activeTab.value)?.focus()
}

async function generateToken() {
  if (working.value || (tokenStatus.value.exists && !window.confirm(t('supplier.token.confirmRotate')))) return
  working.value = true
  try {
    const result = await suppliersAPI.regenerateAccessToken(props.supplier.id)
    if (disposed) return
    plaintextToken.value = result.key
    tokenStatus.value = await suppliersAPI.getAccessTokenStatus(props.supplier.id)
    if (!disposed) emit('changed')
  } catch (error) {
    if (!disposed) appStore.showError((error as Error).message || t('common.unknownError'))
  } finally {
    working.value = false
  }
}

async function revokeToken() {
  if (working.value || !window.confirm(t('supplier.token.confirmRevoke'))) return
  working.value = true
  try {
    await suppliersAPI.revokeAccessToken(props.supplier.id)
    if (disposed) return
    plaintextToken.value = ''
    tokenStatus.value = await suppliersAPI.getAccessTokenStatus(props.supplier.id)
    if (!disposed) emit('changed')
  } catch (error) {
    if (!disposed) appStore.showError((error as Error).message || t('common.unknownError'))
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
</script>
