<template>
  <div class="space-y-6">
    <section class="card p-5">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div class="flex items-center gap-2">
            <h3 class="text-xl font-semibold text-gray-900 dark:text-white">{{ supplier.name }}</h3>
            <span :class="supplier.status === 'active' ? 'badge badge-success' : 'badge badge-danger'">{{ supplier.status }}</span>
          </div>
          <p class="mt-1 font-mono text-sm text-gray-500">{{ supplier.code }} · #{{ supplier.id }}</p>
          <p v-if="supplier.notes" class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ supplier.notes }}</p>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-secondary" @click="emit('edit')"><Icon name="edit" size="sm" />{{ t('common.edit') }}</button>
          <button class="btn btn-danger" @click="emit('delete')"><Icon name="trash" size="sm" />{{ t('common.delete') }}</button>
        </div>
      </div>
      <div class="mt-4 flex flex-wrap gap-2">
        <span v-for="kind in supplier.allowed_account_kinds" :key="`${kind.platform}:${kind.type}`" class="badge badge-gray">{{ kind.platform }}/{{ kind.type }}</span>
        <span v-if="supplier.allowed_account_kinds.length === 0" class="text-sm text-amber-600">{{ t('supplier.admin.noPermission') }}</span>
      </div>
    </section>

    <section class="card p-5">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('supplier.admin.token') }}</h3>
          <p class="mt-1 text-sm text-gray-500">{{ tokenStatus.exists ? tokenStatus.masked_key : t('supplier.token.missing') }}</p>
          <p v-if="tokenStatus.last_used_at" class="mt-1 text-xs text-gray-500">{{ t('supplier.token.lastUsedAt') }}: {{ formatDate(tokenStatus.last_used_at) }}</p>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-secondary" :disabled="working" @click="generateToken">{{ tokenStatus.exists ? t('supplier.token.rotate') : t('supplier.token.generate') }}</button>
          <button v-if="tokenStatus.exists" class="btn btn-danger" :disabled="working" @click="revokeToken">{{ t('supplier.token.revoke') }}</button>
        </div>
      </div>
      <div v-if="plaintextToken" class="mt-4 rounded-lg border border-amber-300 bg-amber-50 p-4 dark:border-amber-700 dark:bg-amber-900/20">
        <p class="font-medium text-amber-900 dark:text-amber-200">{{ t('supplier.token.oneTimeTitle') }}</p>
        <div class="mt-2 flex flex-col gap-2 sm:flex-row">
          <code class="min-w-0 flex-1 break-all rounded bg-white p-3 text-sm dark:bg-dark-900">{{ plaintextToken }}</code>
          <button class="btn btn-secondary" @click="copyToken"><Icon name="copy" size="sm" />{{ t('common.copy') }}</button>
        </div>
      </div>
    </section>

    <section class="card p-5">
      <div class="mb-4 flex items-center justify-between">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('supplier.admin.members') }}</h3>
        <span class="text-sm text-gray-500">{{ members.length }}</span>
      </div>
      <form class="grid gap-3 rounded-lg bg-gray-50 p-4 dark:bg-dark-800 sm:grid-cols-2 lg:grid-cols-5" @submit.prevent="addMember">
        <input v-model.number="memberForm.user_id" class="input" type="number" min="1" :placeholder="t('supplier.admin.bindUserId')" />
        <input v-model.trim="memberForm.email" class="input" type="email" :placeholder="t('common.email')" :required="!memberForm.user_id" />
        <input v-model="memberForm.password" class="input" type="password" :placeholder="t('common.password')" :required="!memberForm.user_id" autocomplete="new-password" />
        <input v-model.trim="memberForm.username" class="input" :placeholder="t('common.name')" />
        <button class="btn btn-primary" :disabled="working" type="submit">{{ t('supplier.admin.addMember') }}</button>
        <p class="text-xs text-gray-500 sm:col-span-2 lg:col-span-5">{{ t('supplier.admin.newMemberHint') }}</p>
      </form>
      <div class="mt-4 divide-y divide-gray-100 dark:divide-dark-700">
        <div v-for="member in members" :key="member.id" class="flex items-center justify-between gap-4 py-3">
          <div class="min-w-0">
            <p class="truncate font-medium text-gray-900 dark:text-white">{{ member.username || member.email }}</p>
            <p class="truncate text-sm text-gray-500">{{ member.email }} · #{{ member.id }}</p>
          </div>
          <button class="btn btn-ghost btn-sm text-red-600" :disabled="working" @click="removeMemberById(member.id)">{{ t('supplier.admin.removeMember') }}</button>
        </div>
        <p v-if="members.length === 0" class="py-6 text-center text-sm text-gray-500">{{ t('common.noData') }}</p>
      </div>
    </section>

    <section class="card overflow-hidden">
      <div class="space-y-4 border-b border-gray-200 p-5 dark:border-dark-700">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('supplier.admin.accounts') }}</h3>
          <div class="flex flex-wrap gap-2">
            <select v-model="accountFilters.review_status" class="input w-36" @change="loadAccounts">
              <option value="">{{ t('supplier.accounts.allReviews') }}</option>
              <option value="pending">{{ t('supplier.accounts.pending') }}</option>
              <option value="approved">{{ t('supplier.accounts.approved') }}</option>
              <option value="rejected">{{ t('supplier.accounts.rejected') }}</option>
            </select>
            <button class="btn btn-secondary" :disabled="selectedIds.length === 0 || working" @click="review('pause')">{{ t('supplier.admin.pause') }}</button>
            <button class="btn btn-secondary" :disabled="selectedIds.length === 0 || working" @click="review('reject')">{{ t('supplier.admin.reject') }}</button>
            <button class="btn btn-primary" :disabled="selectedIds.length === 0 || selectedGroupIds.length === 0 || working" @click="review('approve')">{{ t('supplier.admin.approve') }}</button>
          </div>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <label>
            <span class="input-label">{{ t('supplier.admin.groupIds') }}</span>
            <div class="flex max-h-28 flex-wrap gap-2 overflow-y-auto rounded-lg border border-gray-200 p-3 dark:border-dark-700">
              <label v-for="group in eligibleGroups" :key="group.id" class="inline-flex items-center gap-2 text-sm">
                <input v-model="selectedGroupIds" type="checkbox" :value="group.id" class="h-4 w-4 rounded border-gray-300 text-primary-600" />
                {{ group.name }} (#{{ group.id }})
              </label>
              <span v-if="eligibleGroups.length === 0" class="text-sm text-gray-500">{{ t('common.noData') }}</span>
            </div>
            <span class="input-hint">{{ t('supplier.admin.groupIdsHint') }}</span>
          </label>
          <label>
            <span class="input-label">{{ t('supplier.admin.reviewNote') }}</span>
            <textarea v-model="reviewNote" class="input" rows="3"></textarea>
          </label>
        </div>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full min-w-[850px] divide-y divide-gray-200 dark:divide-dark-700">
          <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-800">
            <tr>
              <th class="w-12 px-4 py-3"><input type="checkbox" :checked="allSelected" :aria-label="t('common.selectAll')" @change="toggleAll(($event.target as HTMLInputElement).checked)" /></th>
              <th class="px-4 py-3">{{ t('common.name') }}</th>
              <th class="px-4 py-3">{{ t('supplier.accounts.platform') }}</th>
              <th class="px-4 py-3">{{ t('supplier.accounts.reviewStatus') }}</th>
              <th class="px-4 py-3">{{ t('supplier.accounts.runtimeStatus') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-800">
            <tr v-for="account in accounts" :key="account.id">
              <td class="px-4 py-3"><input v-model="selectedIds" type="checkbox" :value="account.id" :aria-label="account.name" /></td>
              <td class="px-4 py-3"><p class="font-medium text-gray-900 dark:text-white">{{ account.name }}</p><p class="text-xs text-gray-500">{{ account.supplier_external_id || `#${account.id}` }}</p></td>
              <td class="px-4 py-3 text-sm">{{ account.platform }} / {{ account.type }}</td>
              <td class="px-4 py-3"><span :class="reviewBadge(account.review_status)">{{ t(`supplier.accounts.${account.review_status}`) }}</span><p v-if="account.review_note" class="mt-1 max-w-xs text-xs text-gray-500">{{ account.review_note }}</p></td>
              <td class="px-4 py-3"><span :class="account.status === 'active' ? 'badge badge-success' : 'badge badge-gray'">{{ account.status }}</span></td>
            </tr>
            <tr v-if="accounts.length === 0"><td colspan="5" class="px-4 py-10 text-center text-sm text-gray-500">{{ t('common.noData') }}</td></tr>
          </tbody>
        </table>
      </div>
      <Pagination v-if="accountTotal > 0" :total="accountTotal" :page="accountPage" :page-size="20" :show-page-size-selector="false" @update:page="setAccountPage" />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import { useAppStore } from '@/stores/app'
import groupsAPI from '@/api/admin/groups'
import suppliersAPI, { type AdminSupplier, type AdminSupplierAccount } from '@/api/admin/suppliers'
import type { AdminGroup, AdminUser } from '@/types'

const props = defineProps<{ supplier: AdminSupplier }>()
const emit = defineEmits<{ (event: 'edit'): void; (event: 'delete'): void; (event: 'changed'): void }>()
const { t } = useI18n()
const appStore = useAppStore()
const working = ref(false)
const members = ref<AdminUser[]>([])
const accounts = ref<AdminSupplierAccount[]>([])
const accountTotal = ref(0)
const accountPage = ref(1)
const selectedIds = ref<number[]>([])
const groups = ref<AdminGroup[]>([])
const selectedGroupIds = ref<number[]>([])
const reviewNote = ref('')
const plaintextToken = ref('')
const tokenStatus = ref(props.supplier.access_token)
const accountFilters = reactive({ review_status: 'pending' })
const memberForm = reactive<{ user_id?: number; email: string; password: string; username: string }>({ email: '', password: '', username: '' })

const allSelected = computed(() => accounts.value.length > 0 && accounts.value.every((account) => selectedIds.value.includes(account.id)))
const selectedPlatforms = computed(() => new Set(accounts.value.filter((account) => selectedIds.value.includes(account.id)).map((account) => account.platform)))
const eligibleGroups = computed(() => groups.value.filter((group) => selectedPlatforms.value.size === 0 || selectedPlatforms.value.has(group.platform)))

watch(() => props.supplier, async (supplier) => {
  tokenStatus.value = supplier.access_token
  plaintextToken.value = ''
  selectedIds.value = []
  await loadAll()
}, { deep: true })

async function loadAll() {
  const [memberResult, accountResult, groupResult] = await Promise.all([
    suppliersAPI.listMembers(props.supplier.id),
    suppliersAPI.listAccounts(props.supplier.id, accountPage.value, 20, accountFilters),
    groupsAPI.getAll(),
  ])
  members.value = memberResult.items
  accounts.value = accountResult.items
  accountTotal.value = accountResult.total
  groups.value = groupResult
  selectedIds.value = []
}

async function loadAccounts() {
  try {
    const result = await suppliersAPI.listAccounts(props.supplier.id, accountPage.value, 20, accountFilters)
    accounts.value = result.items
    accountTotal.value = result.total
    selectedIds.value = []
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('supplier.admin.loadFailed'))
  }
}

async function addMember() {
  working.value = true
  try {
    await suppliersAPI.addMember(props.supplier.id, memberForm.user_id
      ? { user_id: memberForm.user_id }
      : { email: memberForm.email, password: memberForm.password, username: memberForm.username })
    Object.assign(memberForm, { user_id: undefined, email: '', password: '', username: '' })
    appStore.showSuccess(t('supplier.admin.memberAdded'))
    await loadAll()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('common.unknownError'))
  } finally {
    working.value = false
  }
}

async function removeMemberById(userId: number) {
  working.value = true
  try {
    await suppliersAPI.removeMember(props.supplier.id, userId)
    appStore.showSuccess(t('supplier.admin.memberRemoved'))
    await loadAll()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('common.unknownError'))
  } finally {
    working.value = false
  }
}

async function generateToken() {
  if (tokenStatus.value.exists && !window.confirm(t('supplier.token.confirmRotate'))) return
  working.value = true
  try {
    plaintextToken.value = (await suppliersAPI.regenerateAccessToken(props.supplier.id)).key
    tokenStatus.value = await suppliersAPI.getAccessTokenStatus(props.supplier.id)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('common.unknownError'))
  } finally {
    working.value = false
  }
}

async function revokeToken() {
  if (!window.confirm(t('supplier.token.confirmRevoke'))) return
  working.value = true
  try {
    await suppliersAPI.revokeAccessToken(props.supplier.id)
    plaintextToken.value = ''
    tokenStatus.value = await suppliersAPI.getAccessTokenStatus(props.supplier.id)
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

async function review(action: 'approve' | 'reject' | 'pause') {
  if (selectedIds.value.length === 0) return
  working.value = true
  try {
    const input = { account_ids: selectedIds.value, note: reviewNote.value || null, group_ids: action === 'approve' ? selectedGroupIds.value : undefined }
    if (action === 'approve') await suppliersAPI.approveAccounts(props.supplier.id, input)
    else if (action === 'reject') await suppliersAPI.rejectAccounts(props.supplier.id, input)
    else await suppliersAPI.pauseAccounts(props.supplier.id, input)
    appStore.showSuccess(t('supplier.admin.reviewCompleted'))
    reviewNote.value = ''
    await loadAccounts()
    emit('changed')
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('common.unknownError'))
  } finally {
    working.value = false
  }
}

function toggleAll(checked: boolean) {
  selectedIds.value = checked ? accounts.value.map((account) => account.id) : []
}

function setAccountPage(value: number) {
  accountPage.value = value
  void loadAccounts()
}

function reviewBadge(status: AdminSupplierAccount['review_status']) {
  if (status === 'approved') return 'badge badge-success'
  if (status === 'rejected') return 'badge badge-danger'
  return 'badge badge-warning'
}

const formatDate = (value: string | null) => value ? new Date(value).toLocaleString() : t('common.notAvailable')

onMounted(async () => {
  try {
    await loadAll()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('supplier.admin.loadFailed'))
  }
})
</script>
