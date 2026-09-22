<template>
  <AppLayout>
    <div class="space-y-6">
      <header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
        <div>
          <h2 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('supplier.accounts.title') }}</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('supplier.accounts.description') }}</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="btn btn-secondary" @click="showBatch = true"><Icon name="upload" size="sm" />{{ t('supplier.accounts.batchImport') }}</button>
          <button class="btn btn-primary" :disabled="allowedKinds.length === 0" @click="openCreate"><Icon name="plus" size="sm" />{{ t('supplier.accounts.create') }}</button>
        </div>
      </header>

      <div v-if="stats" class="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <div v-for="item in statCards" :key="item.label" class="card p-4">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ item.label }}</p>
          <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</p>
        </div>
      </div>

      <section class="card p-4">
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-6">
          <label class="lg:col-span-2">
            <span class="sr-only">{{ t('common.search') }}</span>
            <input v-model.trim="filters.search" class="input" :placeholder="t('supplier.accounts.searchPlaceholder')" @keyup.enter="applyFilters" />
          </label>
          <select v-model="filters.platform" class="input" :aria-label="t('supplier.accounts.platform')" @change="applyFilters">
            <option value="">{{ t('supplier.accounts.allPlatforms') }}</option>
            <option v-for="platform in platforms" :key="platform" :value="platform">{{ platform }}</option>
          </select>
          <select v-model="filters.type" class="input" :aria-label="t('supplier.accounts.type')" @change="applyFilters">
            <option value="">{{ t('supplier.accounts.allTypes') }}</option>
            <option v-for="type in accountTypes" :key="type" :value="type">{{ type }}</option>
          </select>
          <select v-model="filters.review_status" class="input" :aria-label="t('supplier.accounts.reviewStatus')" @change="applyFilters">
            <option value="">{{ t('supplier.accounts.allReviews') }}</option>
            <option value="pending">{{ t('supplier.accounts.pending') }}</option>
            <option value="approved">{{ t('supplier.accounts.approved') }}</option>
            <option value="rejected">{{ t('supplier.accounts.rejected') }}</option>
          </select>
          <select v-model="filters.status" class="input" :aria-label="t('supplier.accounts.runtimeStatus')" @change="applyFilters">
            <option value="">{{ t('supplier.accounts.allStatuses') }}</option>
            <option value="active">{{ t('common.active') }}</option>
            <option value="disabled">{{ t('common.disabled') }}</option>
          </select>
        </div>
      </section>

      <section class="card overflow-hidden">
        <div v-if="loading" class="space-y-3 p-6" aria-busy="true">
          <div v-for="index in 5" :key="index" class="h-12 animate-pulse rounded bg-gray-100 dark:bg-dark-800"></div>
        </div>
        <div v-else-if="accounts.length === 0" class="p-12 text-center">
          <Icon name="inbox" size="xl" class="mx-auto text-gray-400" />
          <h3 class="mt-3 font-medium text-gray-900 dark:text-white">{{ t('supplier.accounts.noAccounts') }}</h3>
          <p class="mx-auto mt-1 max-w-xl text-sm text-gray-500 dark:text-gray-400">{{ t('supplier.accounts.noAccountsHint') }}</p>
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full min-w-[980px] divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-800 dark:text-gray-400">
              <tr>
                <th class="px-4 py-3">{{ t('common.name') }}</th>
                <th class="px-4 py-3">{{ t('supplier.accounts.platform') }}</th>
                <th class="px-4 py-3">{{ t('supplier.accounts.reviewStatus') }}</th>
                <th class="px-4 py-3">{{ t('supplier.accounts.runtimeStatus') }}</th>
                <th class="px-4 py-3">{{ t('supplier.accounts.expiresAt') }}</th>
                <th class="px-4 py-3 text-right">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-800 dark:bg-dark-900">
              <tr v-for="account in accounts" :key="account.id">
                <td class="px-4 py-3">
                  <div class="font-medium text-gray-900 dark:text-white">{{ account.name }}</div>
                  <div class="text-xs text-gray-500">{{ account.external_id || `#${account.id}` }}</div>
                  <div v-if="account.review_note" class="mt-1 max-w-xs text-xs text-amber-700 dark:text-amber-300">{{ account.review_note }}</div>
                </td>
                <td class="px-4 py-3 text-sm"><span class="badge badge-gray">{{ account.platform }}</span> <span class="ml-1 text-gray-500">{{ account.type }}</span></td>
                <td class="px-4 py-3"><span :class="reviewBadge(account.review_status)">{{ t(`supplier.accounts.${account.review_status}`) }}</span></td>
                <td class="px-4 py-3 text-sm">
                  <span :class="account.status === 'active' ? 'badge badge-success' : 'badge badge-gray'">{{ account.status === 'active' ? t('common.active') : t('common.disabled') }}</span>
                  <span v-if="account.schedulable" class="ml-1 text-xs text-green-600">{{ t('supplier.accounts.schedulable') }}</span>
                </td>
                <td class="px-4 py-3 text-sm text-gray-600 dark:text-gray-300">{{ formatDate(account.expires_at) }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end gap-1">
                    <button class="btn btn-ghost btn-sm" :title="t('supplier.accounts.test')" @click="test(account)"><Icon name="play" size="sm" /></button>
                    <button class="btn btn-ghost btn-sm" :title="t('common.edit')" @click="openEdit(account)"><Icon name="edit" size="sm" /></button>
                    <button
                      class="btn btn-ghost btn-sm"
                      :disabled="account.status !== 'active' && account.review_status !== 'approved'"
                      :title="account.status === 'active' ? t('supplier.accounts.pause') : t('supplier.accounts.enable')"
                      @click="toggleStatus(account)"
                    >
                      <Icon :name="account.status === 'active' ? 'xCircle' : 'checkCircle'" size="sm" />
                    </button>
                    <button class="btn btn-ghost btn-sm text-red-600" :title="t('common.delete')" @click="remove(account)"><Icon name="trash" size="sm" /></button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <Pagination v-if="total > 0" :total="total" :page="page" :page-size="pageSize" @update:page="changePage" @update:page-size="changePageSize" />
      </section>
    </div>

    <SupplierAccountForm :show="showForm" :kinds="allowedKinds" :account="editingAccount" :saving="saving" :server-error="saveError" @close="showForm = false" @submit="save" />
    <SupplierBatchImportDialog :show="showBatch" @close="showBatch = false" @completed="batchCompleted" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import SupplierAccountForm from '@/components/supplier/SupplierAccountForm.vue'
import SupplierBatchImportDialog from '@/components/supplier/SupplierBatchImportDialog.vue'
import { useAppStore } from '@/stores/app'
import {
  createAccount,
  deleteAccount,
  getProfile,
  listAccounts,
  testAccount,
  updateAccount,
  type SupplierAccount,
  type SupplierAccountFilters,
  type SupplierAccountInput,
  type SupplierAccountKind,
  type SupplierAccountUpdateInput,
  type SupplierStats,
} from '@/api/supplier'

const { t } = useI18n()
const appStore = useAppStore()
const accounts = ref<SupplierAccount[]>([])
const allowedKinds = ref<SupplierAccountKind[]>([])
const stats = ref<SupplierStats | null>(null)
const loading = ref(true)
const saving = ref(false)
const saveError = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const showForm = ref(false)
const showBatch = ref(false)
const editingAccount = ref<SupplierAccount | null>(null)
const filters = reactive<SupplierAccountFilters>({ search: '', platform: '', type: '', status: '', review_status: '' })

const platforms = computed(() => [...new Set(allowedKinds.value.map((kind) => kind.platform))].sort())
const accountTypes = computed(() => [...new Set(allowedKinds.value.filter((kind) => !filters.platform || kind.platform === filters.platform).map((kind) => kind.type))].sort())
const statCards = computed(() => stats.value ? [
  { label: t('supplier.accounts.totalCount'), value: stats.value.account_count },
  { label: t('supplier.accounts.pendingCount'), value: stats.value.pending_count },
  { label: t('supplier.accounts.schedulableCount'), value: stats.value.schedulable_count },
  { label: t('supplier.accounts.errorCount'), value: stats.value.error_count },
] : [])

async function load() {
  loading.value = true
  try {
    const result = await listAccounts(page.value, pageSize.value, filters)
    accounts.value = result.items
    total.value = result.total
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('supplier.accounts.loadFailed'))
  } finally {
    loading.value = false
  }
}

function applyFilters() {
  page.value = 1
  void load()
}

function changePage(value: number) {
  page.value = value
  void load()
}

function changePageSize(value: number) {
  pageSize.value = value
  page.value = 1
  void load()
}

function openCreate() {
  saveError.value = ''
  editingAccount.value = null
  showForm.value = true
}

function openEdit(account: SupplierAccount) {
  saveError.value = ''
  editingAccount.value = account
  showForm.value = true
}

async function save(accountId: number | null, input: SupplierAccountInput | SupplierAccountUpdateInput) {
  if (saving.value) return
  saving.value = true
  saveError.value = ''
  try {
    if (accountId) {
      await updateAccount(accountId, input as SupplierAccountUpdateInput)
      appStore.showSuccess(t('supplier.accounts.updated'))
    } else {
      await createAccount(input as SupplierAccountInput)
      appStore.showSuccess(t('supplier.accounts.created'))
    }
    showForm.value = false
    await load()
  } catch (error) {
    const failure = error as { message?: string; reason?: string }
    saveError.value = failure.reason === 'INVALID_CREDENTIALS' ? t('supplier.accounts.serverCredentialsInvalid')
      : failure.reason === 'ACCOUNT_KIND_NOT_ALLOWED' ? t('supplier.accounts.kindNotAllowed')
        : failure.reason === 'SUPPLIER_EXTERNAL_ID_EXISTS' ? t('supplier.accounts.externalIdExists')
          : failure.message || t('supplier.accounts.saveFailed')
  } finally {
    saving.value = false
  }
}

async function toggleStatus(account: SupplierAccount) {
  try {
    await updateAccount(account.id, { status: account.status === 'active' ? 'disabled' : 'active' })
    await load()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('common.unknownError'))
  }
}

async function test(account: SupplierAccount) {
  try {
    const success = await testAccount(account.id, {})
    if (success) appStore.showSuccess(t('supplier.accounts.testSucceeded'))
    else appStore.showError(t('supplier.accounts.testFailed'))
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('common.unknownError'))
  }
}

async function remove(account: SupplierAccount) {
  if (!window.confirm(t('supplier.accounts.confirmDelete'))) return
  try {
    await deleteAccount(account.id)
    appStore.showSuccess(t('supplier.accounts.deleted'))
    await load()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('common.unknownError'))
  }
}

function batchCompleted() {
  showBatch.value = false
  void load()
}

function reviewBadge(status: SupplierAccount['review_status']) {
  if (status === 'approved') return 'badge badge-success'
  if (status === 'rejected') return 'badge badge-danger'
  return 'badge badge-warning'
}

const formatDate = (value: string | null) => value ? new Date(value).toLocaleString() : t('common.notAvailable')

onMounted(async () => {
  try {
    const profile = await getProfile()
    allowedKinds.value = profile.allowed_account_kinds
    stats.value = profile.stats
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('supplier.accounts.loadFailed'))
  }
  await load()
})
</script>
