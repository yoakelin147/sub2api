import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAuthStore } from '@/stores/auth'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

const listKeys = vi.fn()

vi.mock('@/api/keys', () => ({
  keysAPI: { list: (...args: unknown[]) => listKeys(...args) },
}))

describe('useBatchImageAccess supplier isolation', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('does not query user API keys for supplier members', async () => {
    const authStore = useAuthStore()
    authStore.$patch({
      token: 'supplier-jwt',
      user: {
        id: 7,
        username: 'supplier',
        email: 'supplier@example.test',
        role: 'supplier',
        supplier_id: 3,
        balance: 0,
        concurrency: 1,
        status: 'active',
        allowed_groups: null,
        balance_notify_enabled: false,
        balance_notify_threshold: null,
        balance_notify_extra_emails: [],
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
    })

    const { refreshBatchImageAccess } = useBatchImageAccess()
    await expect(refreshBatchImageAccess(true)).resolves.toBe(false)
    expect(listKeys).not.toHaveBeenCalled()
  })
})
