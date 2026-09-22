<template>
  <AppLayout>
    <div class="space-y-6">
      <header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
        <div>
          <h2 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('supplier.admin.title') }}</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('supplier.admin.description') }}</p>
        </div>
        <button class="btn btn-primary" @click="openCreate"><Icon name="plus" size="sm" />{{ t('supplier.admin.create') }}</button>
      </header>

      <section class="card p-4">
        <div class="flex flex-col gap-3 sm:flex-row">
          <input v-model.trim="search" class="input flex-1" :placeholder="t('common.searchPlaceholder')" @keyup.enter="applyFilters" />
          <select v-model="status" class="input sm:w-40" @change="applyFilters">
            <option value="">{{ t('common.all') }}</option>
            <option value="active">{{ t('common.active') }}</option>
            <option value="disabled">{{ t('common.disabled') }}</option>
          </select>
          <button class="btn btn-secondary" @click="applyFilters"><Icon name="search" size="sm" />{{ t('common.search') }}</button>
        </div>
      </section>

      <div class="grid gap-6 xl:grid-cols-[minmax(320px,0.8fr)_minmax(0,2.2fr)]">
        <section class="card overflow-hidden self-start">
          <div v-if="loading" class="space-y-3 p-5" aria-busy="true">
            <div v-for="index in 5" :key="index" class="h-16 animate-pulse rounded bg-gray-100 dark:bg-dark-800"></div>
          </div>
          <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
            <button
              v-for="item in suppliers"
              :key="item.id"
              type="button"
              class="flex w-full items-start justify-between gap-4 p-4 text-left transition-colors hover:bg-gray-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-500 dark:hover:bg-dark-800"
              :class="selected?.id === item.id ? 'bg-primary-50 dark:bg-primary-900/20' : ''"
              @click="selectSupplier(item)"
            >
              <span class="min-w-0">
                <span class="block truncate font-medium text-gray-900 dark:text-white">{{ item.name }}</span>
                <span class="block truncate font-mono text-xs text-gray-500">{{ item.code }}</span>
                <span class="mt-2 block text-xs text-gray-500">
                  {{ t('supplier.admin.memberCount') }} {{ item.stats.member_count }} ·
                  {{ t('supplier.admin.accountCount') }} {{ item.stats.account_count }} ·
                  {{ t('supplier.accounts.pendingCount') }} {{ item.stats.pending_count }}
                </span>
              </span>
              <span :class="item.status === 'active' ? 'badge badge-success' : 'badge badge-danger'">{{ item.status }}</span>
            </button>
            <div v-if="suppliers.length === 0" class="p-10 text-center text-sm text-gray-500">{{ t('common.noData') }}</div>
          </div>
          <Pagination v-if="total > 0" :total="total" :page="page" :page-size="pageSize" :show-page-size-selector="false" @update:page="changePage" />
        </section>

        <SupplierDetail v-if="selected" :supplier="selected" @edit="openEdit" @delete="deleteSelected" @changed="refreshSelected" />
        <section v-else class="card flex min-h-72 items-center justify-center p-10 text-center text-gray-500">
          {{ t('supplier.admin.details') }}
        </section>
      </div>
    </div>

    <SupplierFormDialog :show="showForm" :supplier="editing" :saving="saving" @close="showForm = false" @submit="save" />
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
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
const suppliers = ref<AdminSupplier[]>([])
const selected = ref<AdminSupplier | null>(null)
const editing = ref<AdminSupplier | null>(null)
const showForm = ref(false)
const loading = ref(true)
const saving = ref(false)
const search = ref('')
const status = ref('')
const page = ref(1)
const pageSize = 20
const total = ref(0)

async function load() {
  loading.value = true
  try {
    const result = await suppliersAPI.list(page.value, pageSize, { search: search.value, status: status.value })
    suppliers.value = result.items
    total.value = result.total
    if (selected.value) {
      selected.value = result.items.find((item) => item.id === selected.value?.id) ?? null
    }
    const requestedID = Number(route.query.supplier)
    if (!selected.value && Number.isSafeInteger(requestedID) && requestedID > 0) {
      selected.value = result.items.find((item) => item.id === requestedID) ?? await suppliersAPI.get(requestedID)
    }
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('supplier.admin.loadFailed'))
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

function selectSupplier(supplier: AdminSupplier) {
  selected.value = supplier
}

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
    selected.value = saved
    await load()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('common.unknownError'))
  } finally {
    saving.value = false
  }
}

async function refreshSelected() {
  if (!selected.value) return
  try {
    selected.value = await suppliersAPI.get(selected.value.id)
    await load()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('supplier.admin.loadFailed'))
  }
}

async function deleteSelected() {
  if (!selected.value || !window.confirm(t('supplier.admin.confirmDelete'))) return
  try {
    await suppliersAPI.remove(selected.value.id)
    selected.value = null
    appStore.showSuccess(t('supplier.admin.deleted'))
    await load()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('common.unknownError'))
  }
}

onMounted(load)
</script>
