import { beforeEach, describe, expect, it, vi } from 'vitest'

const post = vi.fn()
const get = vi.fn()
const put = vi.fn()
const remove = vi.fn()

vi.mock('@/api/client', () => ({ apiClient: { post, get, put, delete: remove } }))

describe('supplier API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('sends Gemini options and a supplier-scoped code exchange without changing proxies', async () => {
    post.mockResolvedValue({ data: {} })
    const { generateSupplierOAuthURL, exchangeSupplierOAuthCode } = await import('@/api/supplier')
    await generateSupplierOAuthURL('gemini', { proxy_id: 3, type: 'oauth', oauth_type: 'google_one', project_id: 'my-project' })
    await exchangeSupplierOAuthCode('gemini', 'oauth', 'session', 'code', 'state')
    expect(post).toHaveBeenNthCalledWith(1, '/supplier/oauth/gemini/auth-url', { proxy_id: 3, type: 'oauth', oauth_type: 'google_one', project_id: 'my-project' })
    expect(post).toHaveBeenNthCalledWith(2, '/supplier/oauth/gemini/exchange-code', { type: 'oauth', session_id: 'session', code: 'code', state: 'state' })
  })

  it('reads Grok password capability from the supplier route', async () => {
    get.mockResolvedValueOnce({ data: { password_auth_enabled: false } })
    const { getSupplierGrokOAuthCapabilities } = await import('@/api/supplier')
    expect(await getSupplierGrokOAuthCapabilities()).toEqual({ password_auth_enabled: false })
    expect(get).toHaveBeenCalledWith('/supplier/oauth/grok/capabilities')
  })

  it('loads models only from the supplier-scoped account route', async () => {
    get.mockResolvedValueOnce({ data: [{ id: 'gpt-5.6-sol', display_name: 'GPT-5.6 Sol' }] })
    const { getAccountTestModels } = await import('@/api/supplier')
    expect(await getAccountTestModels(11)).toHaveLength(1)
    expect(get).toHaveBeenCalledWith('/supplier/accounts/11/models')
  })

  it('sends a caller-visible idempotency key for single and batch creates', async () => {
    post.mockResolvedValue({ data: {} })
    const { createAccount, batchCreateAccounts } = await import('@/api/supplier')
    const account = { name: 'A', platform: 'openai', type: 'apikey', credentials: { api_key: 'secret' }, proxy_id: 3 }

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
