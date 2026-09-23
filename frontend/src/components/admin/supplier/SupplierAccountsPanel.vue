<template>
  <div>
    <form class="grid gap-3 p-4 sm:grid-cols-2 xl:grid-cols-6" @submit.prevent="applyFilters">
      <div class="flex min-w-0 gap-2 sm:col-span-2">
        <input v-model.trim="search" class="input min-w-0 flex-1" :aria-label="t('common.search')" :placeholder="t('supplier.accounts.searchPlaceholder')" />
        <button type="submit" class="btn btn-secondary px-3" :aria-label="t('common.search')"><Icon name="search" size="sm" /></button>
      </div>
      <select v-model="filters.platform" class="input" :aria-label="t('supplier.accounts.platform')" @change="filters.type = ''; applyFilters()">
        <option value="">{{ t('supplier.accounts.allPlatforms') }}</option>
        <option v-for="platform in platforms" :key="platform" :value="platform">{{ t(`monitorCommon.providers.${platform}`) }}</option>
      </select>
      <select v-model="filters.type" class="input" :aria-label="t('supplier.accounts.type')" @change="applyFilters">
        <option value="">{{ t('supplier.accounts.allTypes') }}</option>
        <option v-for="type in accountTypes" :key="type" :value="type">{{ t(`supplier.admin.kindTypes.${type}`) }}</option>
      </select>
      <select v-model="filters.review_status" class="input" :aria-label="t('supplier.accounts.reviewStatus')" @change="applyFilters">
        <option value="">{{ t('supplier.accounts.allReviews') }}</option>
        <option v-for="status in ['pending', 'approved', 'rejected']" :key="status" :value="status">{{ t(`supplier.accounts.${status}`) }}</option>
      </select>
      <select v-model="filters.status" class="input" :aria-label="t('supplier.accounts.runtimeStatus')" @change="applyFilters">
        <option value="">{{ t('supplier.accounts.allStatuses') }}</option>
        <option value="active">{{ t('common.active') }}</option>
        <option value="disabled">{{ t('common.disabled') }}</option>
      </select>
    </form>
    <div v-if="selectedIds.length" class="flex flex-wrap items-center justify-between gap-3 border-t border-gray-100 bg-primary-50 px-4 py-3 dark:border-dark-700 dark:bg-primary-900/20">
      <span class="text-sm text-primary-700 dark:text-primary-300">{{ t('supplier.admin.selectedAccounts', { count: selectedIds.length }) }}</span>
      <div class="flex flex-wrap gap-2">
        <button class="btn btn-ghost btn-sm" @click="selectedIds = []">{{ t('common.clear') }}</button>
        <button class="btn btn-secondary btn-sm" @click="openReview('pause')">{{ t('supplier.admin.pause') }}</button>
        <button class="btn btn-secondary btn-sm" @click="openReview('reject')">{{ t('supplier.admin.reject') }}</button>
        <button class="btn btn-primary btn-sm" @click="openReview('approve')">{{ t('supplier.admin.approve') }}</button>
      </div>
    </div>
    <div v-if="error" role="alert" class="px-4 py-8 text-center text-sm text-red-600">
      <p>{{ error }}</p><button class="btn btn-secondary btn-sm mt-3" @click="load">{{ t('common.tryAgain') }}</button>
    </div>
    <div v-else class="max-h-96 overflow-auto" :aria-busy="loading" role="region" :aria-label="t('supplier.admin.accounts')" tabindex="0">
      <table class="w-full min-w-[720px] text-left text-sm">
        <thead class="sticky top-0 z-10 bg-gray-50 text-xs text-gray-500 dark:bg-dark-800 dark:text-gray-400">
          <tr>
            <th class="w-12 px-4 py-3"><input type="checkbox" :checked="allSelected" :indeterminate="selectedIds.length > 0 && !allSelected" :disabled="loading || accounts.length === 0" :aria-label="t('supplier.admin.selectPage')" @change="selectedIds = ($event.target as HTMLInputElement).checked ? accounts.map(account => account.id) : []" /></th>
            <th class="px-4 py-3">{{ t('common.name') }}</th>
            <th class="px-4 py-3">{{ t('supplier.accounts.platform') }}</th>
            <th class="px-4 py-3">{{ t('supplier.accounts.reviewStatus') }}</th>
            <th class="px-4 py-3">{{ t('supplier.accounts.runtimeStatus') }}</th>
            <th class="px-4 py-3">{{ t('supplier.admin.loginEmail') }}</th>
            <th class="px-4 py-3">{{ t('supplier.admin.loginPassword') }}</th>
            <th class="px-4 py-3">{{ t('supplier.admin.configuration') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
          <tr v-if="loading"><td colspan="8" class="px-4 py-12 text-center text-gray-500">{{ t('common.loading') }}</td></tr>
          <template v-else>
            <tr v-for="account in accounts" :key="account.id">
              <td class="px-4 py-3"><input v-model="selectedIds" type="checkbox" :value="account.id" :aria-label="account.name" /></td>
              <td class="px-4 py-3"><p class="max-w-xs truncate font-medium text-gray-900 dark:text-white" :title="account.name">{{ account.name }}</p><p class="max-w-xs truncate text-xs text-gray-500" :title="account.supplier_external_id || ''">{{ account.supplier_external_id || `#${account.id}` }}</p></td>
              <td class="px-4 py-3"><p>{{ t(`monitorCommon.providers.${account.platform}`) }}</p><p class="text-xs text-gray-500">{{ t(`supplier.admin.kindTypes.${account.type}`) }}</p></td>
              <td class="px-4 py-3"><span :class="account.review_status === 'approved' ? 'badge badge-success' : account.review_status === 'rejected' ? 'badge badge-danger' : 'badge badge-warning'">{{ t(`supplier.accounts.${account.review_status}`) }}</span><p v-if="account.review_note" class="mt-1 max-w-xs truncate text-xs text-gray-500" :title="account.review_note">{{ account.review_note }}</p></td>
              <td class="px-4 py-3"><span :class="account.status === 'active' ? 'badge badge-success' : 'badge badge-gray'">{{ account.status === 'active' ? t('common.active') : t('common.disabled') }}</span></td>
              <td class="px-4 py-3">{{ account.credentials?.email || '—' }}</td>
              <td class="px-4 py-3">
                <template v-if="account.credentials_status?.has_login_password_encrypted">
                  <span v-if="revealedPassword?.id === account.id" class="break-all">{{ revealedPassword.value }}</span>
                  <span v-else>••••••</span>
                  <button type="button" class="btn btn-secondary btn-sm ml-2" :disabled="revealingId === account.id" @click="togglePassword(account)">{{ t(revealedPassword?.id === account.id ? 'supplier.admin.hidePassword' : 'supplier.admin.showPassword') }}</button>
                </template>
                <span v-else>—</span>
              </td>
              <td class="px-4 py-3"><button type="button" class="btn btn-secondary btn-sm" @click="openDetail(account)">{{ t('supplier.admin.configuration') }}</button></td>
            </tr>
            <tr v-if="accounts.length === 0"><td colspan="8" class="px-4 py-12 text-center text-gray-500">{{ t('common.noData') }}</td></tr>
          </template>
        </tbody>
      </table>
    </div>
    <div v-if="total > 0 && !error" class="overflow-x-auto">
      <Pagination :total="total" :page="page" :page-size="pageSize" @update:page="changePage" @update:page-size="changePageSize" />
    </div>

    <BaseDialog :show="reviewAction !== null" :title="reviewAction ? t(`supplier.admin.${reviewAction}`) : ''" width="normal" :close-on-escape="!working" :show-close-button="!working" @close="reviewAction = null">
      <form id="supplier-review-form" class="space-y-4" @submit.prevent="review">
        <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('supplier.admin.reviewSelection', { count: selectedIds.length }) }}</p>
        <fieldset v-if="reviewAction === 'approve'">
          <legend class="input-label">{{ t('supplier.admin.groupIds') }}</legend>
          <p v-if="groupsLoading" role="status" class="py-4 text-sm text-gray-500">{{ t('common.loading') }}</p>
          <div v-else-if="groupsError" role="alert" class="text-sm text-red-600"><p>{{ groupsError }}</p><button type="button" class="btn btn-secondary btn-sm mt-2" @click="loadGroups">{{ t('common.tryAgain') }}</button></div>
          <div v-else class="max-h-48 space-y-2 overflow-auto rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <label v-for="group in eligibleGroups" :key="group.id" class="flex items-center gap-2 text-sm">
              <input v-model="selectedGroupIds" type="checkbox" :value="group.id" :disabled="working" />{{ group.name }} (#{{ group.id }})
            </label>
            <p v-if="eligibleGroups.length === 0" class="text-sm text-gray-500">{{ t('common.noData') }}</p>
          </div>
          <p class="input-hint">{{ t('supplier.admin.groupIdsHint') }}</p>
        </fieldset>
        <label class="block"><span class="input-label">{{ t('supplier.admin.reviewNote') }}</span><textarea v-model="reviewNote" class="input" rows="3" :disabled="working"></textarea></label>
      </form>
      <template #footer>
        <button class="btn btn-secondary" :disabled="working" @click="reviewAction = null">{{ t('common.cancel') }}</button>
        <button type="submit" form="supplier-review-form" class="btn btn-primary" :disabled="!canReview">{{ working ? t('common.saving') : t('common.confirm') }}</button>
      </template>
    </BaseDialog>
    <BaseDialog :show="detailId !== null" :title="t('supplier.admin.configuration')" width="wide" @close="closeDetail">
      <p v-if="detailLoading" role="status">{{ t('common.loading') }}</p>
      <p v-else-if="detailError" role="alert" class="text-red-600">{{ detailError }}</p>
      <div v-else-if="detail" class="space-y-3 text-sm">
        <p><strong>{{ detail.name }}</strong> · {{ detail.platform }} / {{ detail.type }}</p>
        <dl class="grid grid-cols-2 gap-2 rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
          <dt>{{ t('supplier.accounts.proxy') }}</dt><dd>{{ detail.proxy_id ?? '—' }}</dd>
          <dt>{{ t('supplier.accounts.concurrency') }}</dt><dd>{{ detail.concurrency ?? '—' }}</dd>
          <dt>{{ t('supplier.accounts.priority') }}</dt><dd>{{ detail.priority ?? '—' }}</dd>
          <dt>{{ t('supplier.accounts.loadFactor') }}</dt><dd>{{ detail.load_factor ?? '—' }}</dd>
          <dt>{{ t('supplier.accounts.autoPauseOnExpired') }}</dt><dd>{{ detail.auto_pause_on_expired ? '✓' : '—' }}</dd>
          <dt>{{ t('supplier.accounts.expiresAt') }}</dt><dd>{{ detail.expires_at ? new Date(detail.expires_at * 1000).toLocaleString() : '—' }}</dd>
          <dt>{{ t('supplier.accounts.notes') }}</dt><dd>{{ detail.notes || '—' }}</dd>
        </dl>
        <p>{{ t('supplier.admin.credentialSummary') }}</p>
        <pre class="max-h-60 overflow-auto rounded-lg bg-gray-50 p-3 dark:bg-dark-800">{{ JSON.stringify(detail.credentials || {}, null, 2) }}</pre>
        <p v-if="detail.credentials_status">{{ t('supplier.admin.secretStatus') }}: {{ Object.keys(detail.credentials_status).filter(key => detail?.credentials_status?.[key]).map(key => key.replace(/^has_/, '')).join(', ') }}</p>
        <p v-if="detail.credentials_status?.has_login_password_encrypted">
          {{ t('supplier.admin.loginPassword') }}:
          <span v-if="revealedPassword?.id === detail.id" class="break-all">{{ revealedPassword.value }}</span><span v-else>••••••</span>
          <button type="button" class="btn btn-secondary btn-sm ml-2" :disabled="revealingId === detail.id" @click="togglePassword(detail)">{{ t(revealedPassword?.id === detail.id ? 'supplier.admin.hidePassword' : 'supplier.admin.showPassword') }}</button>
        </p>
      </div>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import { useAppStore } from '@/stores/app'
import groupsAPI from '@/api/admin/groups'
import suppliersAPI, { type AdminSupplierAccount } from '@/api/admin/suppliers'
import { SUPPLIER_ACCOUNT_KINDS } from '@/features/supplier/accountKinds'
import type { AdminGroup } from '@/types'

const props = defineProps<{ supplierId: number }>()
const emit = defineEmits<{ (event: 'changed'): void }>()
const { t } = useI18n()
const appStore = useAppStore()
const accounts = ref<AdminSupplierAccount[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const error = ref('')
const search = ref('')
const filters = reactive({ search: '', platform: '', type: '', review_status: 'pending', status: '' })
const selectedIds = ref<number[]>([])
const groups = ref<AdminGroup[]>([])
const groupsLoading = ref(false)
const groupsError = ref('')
const selectedGroupIds = ref<number[]>([])
const reviewNote = ref('')
const reviewAction = ref<'approve' | 'reject' | 'pause' | null>(null)
const working = ref(false)
const detailId = ref<number | null>(null)
const detail = ref<AdminSupplierAccount | null>(null)
const detailLoading = ref(false)
const detailError = ref('')
const revealedPassword = ref<{ id: number; value: string } | null>(null)
const revealingId = ref<number | null>(null)
let requestId = 0
let disposed = false
onBeforeUnmount(() => { disposed = true; requestId++; revealedPassword.value = null })

async function openDetail(account: AdminSupplierAccount) {
  detailId.value = account.id
  detail.value = null
  detailError.value = ''
  detailLoading.value = true
  try {
    const result = await suppliersAPI.getAccount(props.supplierId, account.id)
    if (detailId.value === account.id) detail.value = result
  } catch (err) {
    if (detailId.value === account.id) detailError.value = (err as Error).message || t('supplier.admin.loadFailed')
  } finally {
    if (detailId.value === account.id) detailLoading.value = false
  }
}

function closeDetail() {
  detailId.value = null
  detail.value = null
  revealedPassword.value = null
}

async function togglePassword(account: AdminSupplierAccount) {
  if (revealedPassword.value?.id === account.id) { revealedPassword.value = null; return }
  const openedDetail = detailId.value
  revealingId.value = account.id
  revealedPassword.value = null
  try {
    const value = await suppliersAPI.revealAccountPassword(props.supplierId, account.id)
    if (!disposed && revealingId.value === account.id && detailId.value === openedDetail) revealedPassword.value = { id: account.id, value }
  } catch (err) {
    appStore.showError((err as Error).message || t('supplier.admin.loadFailed'))
  } finally {
    revealingId.value = null
  }
}

// Include historical account types even when a supplier can no longer submit them.
const platforms = [...new Set(SUPPLIER_ACCOUNT_KINDS.map(kind => kind.platform))]
const accountTypes = computed(() => [...new Set(SUPPLIER_ACCOUNT_KINDS.filter(kind => !filters.platform || kind.platform === filters.platform).map(kind => kind.type))])
const allSelected = computed(() => accounts.value.length > 0 && accounts.value.every(account => selectedIds.value.includes(account.id)))
const selectedPlatforms = computed(() => new Set(accounts.value.filter(account => selectedIds.value.includes(account.id)).map(account => account.platform)))
const eligibleGroups = computed(() => groups.value.filter(group => selectedPlatforms.value.has(group.platform)))
const canReview = computed(() => !working.value && !loading.value && selectedIds.value.length > 0 && reviewAction.value !== null && (reviewAction.value !== 'approve' || (!groupsLoading.value && !groupsError.value && selectedGroupIds.value.length > 0)))

async function load() {
  const request = ++requestId
  loading.value = true
  error.value = ''
  selectedIds.value = []
  revealedPassword.value = null
  reviewAction.value = null
  try {
    const result = await suppliersAPI.listAccounts(props.supplierId, page.value, pageSize.value, { ...filters })
    if (request !== requestId) return
    total.value = result.total
    const lastPage = Math.max(1, Math.ceil(result.total / pageSize.value))
    if (page.value > lastPage) {
      page.value = lastPage
      await load()
      return
    }
    accounts.value = result.items
  } catch (err) {
    if (request === requestId) error.value = (err as Error).message || t('supplier.admin.loadFailed')
  } finally {
    if (request === requestId) loading.value = false
  }
}

function changePage(value: number) {
  page.value = value
  void load()
}

function applyFilters() {
  filters.search = search.value
  changePage(1)
}

function changePageSize(value: number) {
  pageSize.value = value
  changePage(1)
}

async function loadGroups() {
  groupsLoading.value = true
  groupsError.value = ''
  try {
    groups.value = await groupsAPI.getAll()
  } catch (err) {
    groupsError.value = (err as Error).message || t('supplier.admin.loadFailed')
  } finally {
    groupsLoading.value = false
  }
}

function openReview(action: 'approve' | 'reject' | 'pause') {
  selectedGroupIds.value = []
  reviewNote.value = ''
  reviewAction.value = action
  if (action === 'approve') void loadGroups()
}

async function review() {
  if (!canReview.value) return
  working.value = true
  try {
    const input = { account_ids: [...selectedIds.value], note: reviewNote.value || null, group_ids: reviewAction.value === 'approve' ? [...selectedGroupIds.value] : undefined }
    if (reviewAction.value === 'approve') await suppliersAPI.approveAccounts(props.supplierId, input)
    else if (reviewAction.value === 'reject') await suppliersAPI.rejectAccounts(props.supplierId, input)
    else await suppliersAPI.pauseAccounts(props.supplierId, input)
    if (disposed) return
    reviewAction.value = null
    appStore.showSuccess(t('supplier.admin.reviewCompleted'))
    await load()
    if (!disposed) emit('changed')
  } catch (err) {
    if (!disposed) appStore.showError((err as Error).message || t('common.unknownError'))
  } finally {
    working.value = false
  }
}

onMounted(load)
</script>
