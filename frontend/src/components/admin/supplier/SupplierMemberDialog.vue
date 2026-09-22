<template>
  <BaseDialog :show="show" :title="t('supplier.admin.memberDialogTitle')" width="normal" @close="emit('close')">
    <form id="supplier-member-form" class="space-y-5" @submit.prevent="submit">
      <fieldset>
        <legend class="input-label">{{ t('supplier.admin.memberMode') }}</legend>
        <div class="grid grid-cols-2 gap-2 rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
          <label class="cursor-pointer">
            <input v-model="mode" data-test="mode-create" class="peer sr-only" type="radio" value="create" />
            <span class="block rounded-md px-3 py-2 text-center text-sm font-medium text-gray-600 peer-checked:bg-white peer-checked:text-primary-600 peer-checked:shadow-sm dark:text-gray-300 dark:peer-checked:bg-dark-700 dark:peer-checked:text-primary-400">
              {{ t('supplier.admin.createNewMember') }}
            </span>
          </label>
          <label class="cursor-pointer">
            <input v-model="mode" data-test="mode-bind" class="peer sr-only" type="radio" value="bind" />
            <span class="block rounded-md px-3 py-2 text-center text-sm font-medium text-gray-600 peer-checked:bg-white peer-checked:text-primary-600 peer-checked:shadow-sm dark:text-gray-300 dark:peer-checked:bg-dark-700 dark:peer-checked:text-primary-400">
              {{ t('supplier.admin.bindExistingMember') }}
            </span>
          </label>
        </div>
      </fieldset>

      <template v-if="mode === 'create'">
        <label class="block">
          <span class="input-label">{{ t('admin.users.email') }}</span>
          <input
            v-model.trim="email"
            data-test="member-email"
            type="email"
            required
            autocomplete="off"
            class="input"
            :placeholder="t('admin.users.enterEmail')"
          />
        </label>

        <label class="block">
          <span class="input-label">{{ t('admin.users.password') }}</span>
          <div class="flex gap-2">
            <input
              v-model="password"
              data-test="member-password"
              type="text"
              required
              minlength="6"
              autocomplete="new-password"
              class="input flex-1"
              :placeholder="t('admin.users.enterPassword')"
            />
            <button type="button" class="btn btn-secondary px-3" :aria-label="t('supplier.admin.generatePassword')" @click="generatePassword">
              <Icon name="refresh" size="md" />
            </button>
          </div>
        </label>

        <label class="block">
          <span class="input-label">{{ t('admin.users.username') }}</span>
          <input
            v-model.trim="username"
            data-test="member-username"
            type="text"
            class="input"
            :placeholder="t('admin.users.enterUsername')"
          />
        </label>

        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block">
            <span class="input-label">{{ t('admin.users.form.roleLabel') }}</span>
            <input class="input bg-gray-50 dark:bg-dark-800" :value="t('admin.users.roles.supplier')" disabled />
            <span class="input-hint">{{ t('supplier.admin.supplierRoleHint') }}</span>
          </label>
          <label class="block">
            <span class="input-label">{{ t('admin.users.columns.concurrency') }}</span>
            <input v-model.number="concurrency" data-test="member-concurrency" type="number" min="0" class="input" />
          </label>
        </div>
      </template>

      <label v-else class="block">
        <span class="input-label">{{ t('supplier.admin.bindUserId') }}</span>
        <input
          v-model.number="userId"
          data-test="member-user-id"
          type="number"
          min="1"
          required
          class="input"
          :placeholder="t('supplier.admin.bindUserId')"
        />
        <span class="input-hint">{{ t('supplier.admin.bindUserIdHint') }}</span>
      </label>
    </form>

    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="saving" @click="emit('close')">{{ t('common.cancel') }}</button>
      <button type="submit" form="supplier-member-form" class="btn btn-primary" :disabled="saving">
        {{ saving ? t('common.saving') : t('supplier.admin.addMember') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import type { SupplierMemberInput } from '@/api/admin/suppliers'

const props = defineProps<{ show: boolean; saving?: boolean }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'submit', input: SupplierMemberInput): void
}>()
const { t } = useI18n()

const mode = ref<'create' | 'bind'>('create')
const userId = ref<number | null>(null)
const email = ref('')
const password = ref('')
const username = ref('')
const concurrency = ref(1)

function reset() {
  mode.value = 'create'
  userId.value = null
  email.value = ''
  password.value = ''
  username.value = ''
  concurrency.value = 1
}

watch(() => props.show, (show) => {
  if (show) reset()
}, { immediate: true })

function generatePassword() {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789!@#$%^&*'
  const random = new Uint32Array(16)
  globalThis.crypto.getRandomValues(random)
  password.value = Array.from(random, (value) => chars[value % chars.length]).join('')
}

function submit() {
  if (mode.value === 'bind') {
    if (userId.value && userId.value > 0) emit('submit', { user_id: Number(userId.value) })
    return
  }
  emit('submit', {
    email: email.value,
    password: password.value,
    username: username.value,
    concurrency: Number(concurrency.value) || 0,
  })
}
</script>
