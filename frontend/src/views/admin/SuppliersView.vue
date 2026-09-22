<template>
  <AppLayout>
    <div class="min-w-0 space-y-4">
      <header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
        <div>
          <h2 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('supplier.admin.title') }}</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('supplier.admin.description') }}</p>
        </div>
        <button v-if="!selected" class="btn btn-primary" @click="openCreate"><Icon name="plus" size="sm" />{{ t('supplier.admin.create') }}</button>
        <button v-else class="btn btn-secondary" @click="backToList"><Icon name="chevronLeft" size="sm" />{{ t('supplier.admin.backToList') }}</button>
      </header>

      <section v-if="!selected" class="card min-w-0 overflow-hidden">
        <form class="flex flex-col gap-3 p-4 sm:flex-row" @submit.prevent="applyFilters">
          <input v-model.trim="search" class="input min-w-0 flex-1" :aria-label="t('common.search')" :placeholder="t('common.searchPlaceholder')" />
          <select v-model="status" class="input sm:w-40" :aria-label="t('common.status')" @change="applyFilters">
            <option value="">{{ t('common.all') }}</option>
            <option value="active">{{ t('common.active') }}</option>
            <option value="disabled">{{ t('common.disabled') }}</option>
          </select>
          <button type="submit" class="btn btn-secondary"><Icon name="search" size="sm" />{{ t('common.search') }}</button>
        </form>
        <div v-if="loadError" role="alert" class="p-8 text-center text-sm text-red-600">
          <p>{{ loadError }}</p><button class="btn btn-secondary btn-sm mt-3" @click="load">{{ t('common.tryAgain') }}</button>
        </div>
        <div v-else class="max-h-96 overflow-auto" :aria-busy="loading" role="region" :aria-label="t('supplier.admin.title')" tabindex="0">
          <table class="w-full min-w-[800px] text-left text-sm">
            <thead class="sticky top-0 z-10 bg-gray-50 text-xs text-gray-500 dark:bg-dark-800 dark:text-gray-400">
              <tr>
                <th class="px-4 py-3">{{ t('common.name') }}</th><th class="px-4 py-3">{{ t('common.status') }}</th>
                <th class="px-4 py-3">{{ t('supplier.admin.memberCount') }}</th><th class="px-4 py-3">{{ t('supplier.admin.accountCount') }}</th>
                <th class="px-4 py-3">{{ t('supplier.accounts.pendingCount') }}</th><th class="px-4 py-3">{{ t('supplier.accounts.schedulableCount') }}</th>
                <th class="px-4 py-3">{{ t('supplier.accounts.errorCount') }}</th><th class="px-4 py-3">{{ t('supplier.token.lastUsedAt') }}</th>
                <th class="px-4 py-3 text-right">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-if="loading"><td colspan="9" class="px-4 py-12 text-center text-gray-500">{{ t('common.loading') }}</td></tr>
              <template v-else>
                <tr v-for="item in suppliers" :key="item.id" class="hover:bg-gray-50 dark:hover:bg-dark-800/50">
                  <td class="px-4 py-3"><button class="block max-w-xs truncate text-left font-medium text-primary-600 hover:underline" :title="item.name" @click="selectSupplier(item)">{{ item.name }}</button><p class="max-w-xs truncate font-mono text-xs text-gray-500" :title="item.code">{{ item.code }}</p></td>
                  <td class="px-4 py-3"><span :class="item.status === 'active' ? 'badge badge-success' : 'badge badge-danger'">{{ item.status === 'active' ? t('common.active') : t('common.disabled') }}</span></td>
                  <td class="px-4 py-3 tabular-nums">{{ item.stats.member_count }}</td><td class="px-4 py-3 tabular-nums">{{ item.stats.account_count }}</td>
                  <td class="px-4 py-3 tabular-nums">{{ item.stats.pending_count }}</td><td class="px-4 py-3 tabular-nums">{{ item.stats.schedulable_count }}</td>
                  <td class="px-4 py-3 tabular-nums">{{ item.stats.error_count }}</td><td class="whitespace-nowrap px-4 py-3 text-xs text-gray-500">{{ item.access_token.last_used_at ? new Date(item.access_token.last_used_at).toLocaleString() : '-' }}</td>
                  <td class="px-4 py-3 text-right"><button class="btn btn-ghost btn-sm" @click="selectSupplier(item)">{{ t('supplier.admin.details') }}<Icon name="chevronRight" size="sm" /></button></td>
                </tr>
                <tr v-if="suppliers.length === 0"><td colspan="9" class="px-4 py-12 text-center text-gray-500">{{ t('common.noData') }}</td></tr>
              </template>
            </tbody>
          </table>
        </div>
        <div v-if="total > 0 && !loadError" class="overflow-x-auto"><Pagination :total="total" :page="page" :page-size="pageSize" @update:page="changePage" @update:page-size="changePageSize" /></div>
      </section>
      <SupplierDetail v-if="selected" :key="selected.id" :supplier="selected" @edit="openEdit" @delete="deleteSelected" @changed="refreshSelected" />
    </div>

    <SupplierFormDialog :show="showForm" :supplier="editing" :saving="saving" @close="showForm = false" @submit="save" />
  </AppLayout>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import SupplierFormDialog from '@/components/admin/supplier/SupplierFormDialog.vue'
import SupplierDetail from '@/components/admin/supplier/SupplierDetail.vue'
import suppliersAPI, { type AdminSupplier, type SupplierWriteInput } from '@/api/admin/suppliers'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()
const route = useRoute()
const router = useRouter()
const suppliers = ref<AdminSupplier[]>([])
const selected = ref<AdminSupplier | null>(null)
const editing = ref<AdminSupplier | null>(null)
const showForm = ref(false)
const loading = ref(true)
const loadError = ref('')
const saving = ref(false)
const search = ref('')
const status = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
let listRequest = 0
let detailRequest = 0
onBeforeUnmount(() => { listRequest++; detailRequest++ })

async function load() {
  const request = ++listRequest
  loading.value = true
  loadError.value = ''
  try {
    const result = await suppliersAPI.list(page.value, pageSize.value, { search: search.value, status: status.value })
    if (request !== listRequest) return
    suppliers.value = result.items
    total.value = result.total
    const lastPage = Math.max(1, Math.ceil(result.total / pageSize.value))
    if (page.value > lastPage) {
      page.value = lastPage
      await load()
    }
  } catch (error) {
    if (request === listRequest) loadError.value = (error as Error).message || t('supplier.admin.loadFailed')
  } finally {
    if (request === listRequest) loading.value = false
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
  changePage(1)
}

function selectSupplier(supplier: AdminSupplier) {
  detailRequest++
  selected.value = supplier
  void router.push({ query: { ...route.query, supplier: supplier.id } })
}

function backToList() {
  detailRequest++
  selected.value = null
  void router.push({ query: { ...route.query, supplier: undefined } })
}

watch(() => route.query.supplier, async (value) => {
  const request = ++detailRequest
  const id = Number(value)
  if (!Number.isSafeInteger(id) || id <= 0) { selected.value = null; return }
  if (selected.value?.id === id) return
  selected.value = null
  try {
    const supplier = suppliers.value.find(item => item.id === id) ?? await suppliersAPI.get(id)
    if (request === detailRequest) selected.value = supplier
  } catch (error) {
    if (request === detailRequest) appStore.showError((error as Error).message || t('supplier.admin.loadFailed'))
  }
}, { immediate: true })

function openCreate() {
  editing.value = null
  showForm.value = true
}

function openEdit() {
  editing.value = selected.value
  showForm.value = true
}

async function save(input: SupplierWriteInput) {
  saving.value = true
  try {
    const saved = editing.value
      ? await suppliersAPI.update(editing.value.id, input)
      : await suppliersAPI.create(input)
    appStore.showSuccess(t(editing.value ? 'supplier.admin.updated' : 'supplier.admin.created'))
    showForm.value = false
    selectSupplier(saved)
    await load()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('common.unknownError'))
  } finally {
    saving.value = false
  }
}

async function refreshSelected() {
  if (!selected.value) return
  const id = selected.value.id
  const request = ++detailRequest
  try {
    const supplier = await suppliersAPI.get(id)
    if (request !== detailRequest || selected.value?.id !== id) return
    selected.value = supplier
    await load()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('supplier.admin.loadFailed'))
  }
}

async function deleteSelected() {
  if (!selected.value || !window.confirm(t('supplier.admin.confirmDelete'))) return
  try {
    await suppliersAPI.remove(selected.value.id)
    backToList()
    appStore.showSuccess(t('supplier.admin.deleted'))
    await load()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('common.unknownError'))
  }
}

onMounted(load)
</script>
