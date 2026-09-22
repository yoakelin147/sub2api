import { beforeEach, describe, expect, it, vi } from 'vitest'

const post = vi.fn()
const get = vi.fn()
const put = vi.fn()
const remove = vi.fn()

vi.mock('@/api/client', () => ({ apiClient: { post, get, put, delete: remove } }))

describe('supplier API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('sends a caller-visible idempotency key for single and batch creates', async () => {
    post.mockResolvedValue({ data: {} })
    const { createAccount, batchCreateAccounts } = await import('@/api/supplier')
    const account = { name: 'A', platform: 'openai', type: 'apikey', credentials: { api_key: 'secret' } }

    await createAccount(account, 'single-key')
    await batchCreateAccounts([account], 'batch-key')

    expect(post).toHaveBeenNthCalledWith(1, '/supplier/accounts', account, {
      headers: { 'Idempotency-Key': 'single-key' },
    })
    expect(post).toHaveBeenNthCalledWith(2, '/supplier/accounts/batch', { accounts: [account] }, {
      headers: { 'Idempotency-Key': 'batch-key' },
    })
  })
})
