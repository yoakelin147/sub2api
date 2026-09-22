<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <header>
        <h2 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('supplier.security.title') }}</h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('supplier.security.description') }}</p>
      </header>
      <ProfilePasswordForm />
      <ProfileTotpCard />
      <ProfilePasskeyCard :enabled="passkeyEnabled" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ProfilePasswordForm from '@/components/user/profile/ProfilePasswordForm.vue'
import ProfileTotpCard from '@/components/user/profile/ProfileTotpCard.vue'
import ProfilePasskeyCard from '@/components/user/profile/ProfilePasskeyCard.vue'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()
const passkeyEnabled = ref(false)

onMounted(async () => {
  try {
    const settings = await appStore.fetchPublicSettings()
    passkeyEnabled.value = settings?.passkey_enabled === true
  } catch {
    passkeyEnabled.value = false
  }
})
</script>
