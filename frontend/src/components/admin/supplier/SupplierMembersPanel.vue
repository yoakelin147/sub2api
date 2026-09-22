<template>
  <div>
    <div class="flex flex-wrap items-center justify-between gap-3 p-4">
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('supplier.admin.membersDescription') }}</p>
      <button class="btn btn-primary btn-sm" :disabled="working" @click="showDialog = true">
        <Icon name="userPlus" size="sm" />{{ t('supplier.admin.addMember') }}
      </button>
    </div>
    <div v-if="error" role="alert" class="px-4 py-8 text-center text-sm text-red-600">
      <p>{{ error }}</p>
      <button class="btn btn-secondary btn-sm mt-3" @click="load">{{ t('common.tryAgain') }}</button>
    </div>
    <div v-else class="max-h-96 overflow-auto" :aria-busy="loading" role="region" :aria-label="t('supplier.admin.members')" tabindex="0">
      <table class="w-full min-w-[640px] text-left text-sm">
        <thead class="sticky top-0 z-10 bg-gray-50 text-xs text-gray-500 dark:bg-dark-800 dark:text-gray-400">
          <tr>
            <th class="px-4 py-3">{{ t('common.email') }}</th>
            <th class="px-4 py-3">{{ t('admin.users.username') }}</th>
            <th class="px-4 py-3">{{ t('admin.users.form.roleLabel') }}</th>
            <th class="px-4 py-3">{{ t('common.status') }}</th>
            <th class="px-4 py-3 text-right">{{ t('common.actions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
          <tr v-if="loading"><td colspan="5" class="px-4 py-12 text-center text-gray-500">{{ t('common.loading') }}</td></tr>
          <template v-else>
            <tr v-for="member in members" :key="member.id">
              <td class="px-4 py-3"><p class="max-w-xs truncate font-medium text-gray-900 dark:text-white" :title="member.email">{{ member.email }}</p><p class="text-xs text-gray-500">#{{ member.id }}</p></td>
              <td class="max-w-48 truncate px-4 py-3" :title="member.username">{{ member.username || '-' }}</td>
              <td class="px-4 py-3"><span class="badge badge-gray">{{ t('admin.users.roles.supplier') }}</span></td>
              <td class="px-4 py-3"><span :class="member.status === 'active' ? 'badge badge-success' : 'badge badge-danger'">{{ member.status === 'active' ? t('common.active') : t('common.disabled') }}</span></td>
              <td class="px-4 py-3 text-right"><button class="btn btn-ghost btn-sm text-red-600" :disabled="working" @click="remove(member.id)">{{ t('supplier.admin.removeMember') }}</button></td>
            </tr>
            <tr v-if="members.length === 0"><td colspan="5" class="px-4 py-12 text-center text-gray-500">{{ t('common.noData') }}</td></tr>
          </template>
        </tbody>
      </table>
    </div>
    <div v-if="total > 0 && !error" class="overflow-x-auto" :class="{ 'pointer-events-none opacity-50': working }" :inert="working || undefined">
      <Pagination :total="total" :page="page" :page-size="pageSize" @update:page="changePage" @update:page-size="changePageSize" />
    </div>
    <SupplierMemberDialog :show="showDialog" :saving="working" @close="showDialog = false" @submit="add" />
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import SupplierMemberDialog from './SupplierMemberDialog.vue'
import suppliersAPI, { type SupplierMemberInput } from '@/api/admin/suppliers'
import type { AdminUser } from '@/types'
import { useAppStore } from '@/stores/app'

const props = defineProps<{ supplierId: number }>()
const emit = defineEmits<{ (event: 'changed'): void }>()
const { t } = useI18n()
const appStore = useAppStore()
const members = ref<AdminUser[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const error = ref('')
const working = ref(false)
const showDialog = ref(false)
let requestId = 0
let disposed = false
onBeforeUnmount(() => { disposed = true; requestId++ })

async function load() {
  const request = ++requestId
  loading.value = true
  error.value = ''
  try {
    const result = await suppliersAPI.listMembers(props.supplierId, page.value, pageSize.value)
    if (request !== requestId) return
    total.value = result.total
    const lastPage = Math.max(1, Math.ceil(result.total / pageSize.value))
    if (page.value > lastPage) {
      page.value = lastPage
      await load()
      return
    }
    members.value = result.items
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

function changePageSize(value: number) {
  pageSize.value = value
  changePage(1)
}

async function add(input: SupplierMemberInput) {
  if (working.value) return
  working.value = true
  try {
    await suppliersAPI.addMember(props.supplierId, input)
    if (disposed) return
    showDialog.value = false
    appStore.showSuccess(t('supplier.admin.memberAdded'))
    page.value = 1
    await load()
    if (!disposed) emit('changed')
  } catch (err) {
    if (!disposed) appStore.showError((err as Error).message || t('common.unknownError'))
  } finally {
    working.value = false
  }
}

async function remove(userId: number) {
  if (working.value || !window.confirm(t('supplier.admin.confirmRemoveMember'))) return
  working.value = true
  try {
    await suppliersAPI.removeMember(props.supplierId, userId)
    if (disposed) return
    appStore.showSuccess(t('supplier.admin.memberRemoved'))
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
