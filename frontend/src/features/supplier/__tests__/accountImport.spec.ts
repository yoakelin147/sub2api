import { describe, expect, it } from 'vitest'
import { parseSupplierAccountImport } from '@/features/supplier/accountImport'

describe('parseSupplierAccountImport', () => {
  it('accepts the stable JSON batch envelope', () => {
    const accounts = parseSupplierAccountImport(
      'json',
      JSON.stringify({
        accounts: [{ external_id: 'ext-1', name: 'A', platform: 'openai', type: 'apikey', credentials: { api_key: 'sk-a' } }],
      }),
    )

    expect(accounts).toEqual([
      { external_id: 'ext-1', name: 'A', platform: 'openai', type: 'apikey', credentials: { api_key: 'sk-a' } },
    ])
  })

  it('parses quoted CSV cells and credential columns', () => {
    const accounts = parseSupplierAccountImport(
      'csv',
      'external_id,name,platform,type,credential.api_key,notes\r\n' +
        'ext-1,"Team, One",openai,apikey,sk-a,"quoted ""note"""',
    )

    expect(accounts).toEqual([
      {
        external_id: 'ext-1',
        name: 'Team, One',
        platform: 'openai',
        type: 'apikey',
        credentials: { api_key: 'sk-a' },
        notes: 'quoted "note"',
      },
    ])
  })

  it('parses line-oriented shorthand and JSON credentials', () => {
    const accounts = parseSupplierAccountImport(
      'text',
      [
        'openai|apikey|ext-1|OpenAI A|sk-a',
        'anthropic|bedrock|ext-2|Bedrock B|{"auth_mode":"api_key","aws_region":"us-east-1","api_key":"bedrock-key"}',
      ].join('\n'),
    )

    expect(accounts[0].credentials).toEqual({ api_key: 'sk-a' })
    expect(accounts[1].credentials).toEqual({ auth_mode: 'api_key', aws_region: 'us-east-1', api_key: 'bedrock-key' })
  })

  it('reports the source row for malformed input', () => {
    expect(() => parseSupplierAccountImport('csv', 'name,platform,type\nA,openai,apikey')).toThrow(
      'CSV row 1',
    )
    expect(() => parseSupplierAccountImport('text', 'openai|apikey|missing-fields')).toThrow(
      'Text line 1',
    )
  })

  it('rejects batches above the server limit', () => {
    const accounts = Array.from({ length: 501 }, (_, index) => ({
      name: `Account ${index}`,
      platform: 'openai',
      type: 'apikey',
      credentials: { api_key: `key-${index}` },
    }))

    expect(() => parseSupplierAccountImport('json', JSON.stringify(accounts))).toThrow('500')
  })
})
