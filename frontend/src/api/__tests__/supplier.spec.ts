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

  it('requires an explicit successful SSE completion instead of treating HTTP 200 as a passed test', async () => {
    const { testAccount } = await import('@/api/supplier')
    for (const [body, expected] of [
      ['data: {"type":"test_complete","success":true}\n\n', true],
      ['data: {"type":"error","error":"Upstream rejected request"}\n\n', false],
      ['data: {"type":"content","text":"incomplete"}\n\n', false],
      ['data: {"type":"test_complete","success":false}\r\n\r\n', false],
      ['data: {"type":"test_complete","success":true}\n\ndata: {"type":"error"}\n\n', false],
      ['invalid response', false],
    ] as const) {
      post.mockResolvedValueOnce({ data: body })
      expect(await testAccount(11)).toBe(expected)
    }
    expect(post).toHaveBeenLastCalledWith('/supplier/accounts/11/test', {}, { responseType: 'text' })
  })
})
