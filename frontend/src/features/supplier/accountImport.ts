import type { SupplierAccountInput } from '@/api/supplier'

export type SupplierImportFormat = 'json' | 'csv' | 'text'

const MAX_BATCH_SIZE = 500
const ACCOUNT_FIELDS = new Set(['external_id', 'name', 'notes', 'platform', 'type', 'credentials', 'expires_at', 'proxy_id', 'concurrency', 'priority', 'load_factor', 'auto_pause_on_expired'])

function asAccount(value: unknown, source: string): SupplierAccountInput {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${source}: account must be an object`)
  }
  const record = value as Record<string, unknown>
  const name = typeof record.name === 'string' ? record.name.trim() : ''
  const platform = typeof record.platform === 'string' ? record.platform.trim().toLowerCase() : ''
  const type = typeof record.type === 'string' ? record.type.trim().toLowerCase() : ''
  const credentials = record.credentials
  if (
    !name ||
    !platform ||
    !type ||
    !credentials ||
    typeof credentials !== 'object' ||
    Array.isArray(credentials) ||
    Object.keys(credentials).length === 0
  ) {
    throw new Error(`${source}: name, platform, type and credentials are required`)
  }
  const proxyId = Number(record.proxy_id || 0)
  if (!Number.isSafeInteger(proxyId) || proxyId < 0) throw new Error(`${source}: proxy_id must be a positive integer`)
  const account: SupplierAccountInput = { name, platform, type, credentials: credentials as Record<string, unknown>, proxy_id: proxyId }
  for (const field of ['concurrency', 'priority', 'load_factor'] as const) {
    if (record[field] !== undefined && record[field] !== '') {
      const value = Number(record[field])
      if (!Number.isSafeInteger(value)) throw new Error(`${source}: ${field} must be an integer`)
      account[field] = value
    }
  }
  if (record.auto_pause_on_expired !== undefined && record.auto_pause_on_expired !== '') {
    if (record.auto_pause_on_expired !== true && record.auto_pause_on_expired !== false && record.auto_pause_on_expired !== 'true' && record.auto_pause_on_expired !== 'false') throw new Error(`${source}: auto_pause_on_expired must be a boolean`)
    account.auto_pause_on_expired = record.auto_pause_on_expired === true || record.auto_pause_on_expired === 'true'
  }
  if (typeof record.external_id === 'string' && record.external_id.trim()) account.external_id = record.external_id.trim()
  if (typeof record.notes === 'string') account.notes = record.notes.trim() || null
  if (record.expires_at !== undefined && record.expires_at !== null && record.expires_at !== '') {
    const expiresAt = Number(record.expires_at)
    if (!Number.isSafeInteger(expiresAt) || expiresAt <= 0) throw new Error(`${source}: expires_at must be a Unix timestamp`)
    account.expires_at = expiresAt
  }
  return account
}

function parseCSVRows(input: string): string[][] {
  const rows: string[][] = []
  let row: string[] = []
  let cell = ''
  let quoted = false
  for (let index = 0; index < input.length; index += 1) {
    const char = input[index]
    if (quoted) {
      if (char === '"' && input[index + 1] === '"') {
        cell += '"'
        index += 1
      } else if (char === '"') {
        quoted = false
      } else {
        cell += char
      }
    } else if (char === '"') {
      quoted = true
    } else if (char === ',') {
      row.push(cell.trim())
      cell = ''
    } else if (char === '\n') {
      row.push(cell.trim())
      if (row.some(Boolean)) rows.push(row)
      row = []
      cell = ''
    } else if (char !== '\r') {
      cell += char
    }
  }
  if (quoted) throw new Error('CSV has an unclosed quoted cell')
  row.push(cell.trim())
  if (row.some(Boolean)) rows.push(row)
  return rows
}

function parseCSV(input: string): SupplierAccountInput[] {
  const rows = parseCSVRows(input.replace(/^\uFEFF/, ''))
  if (rows.length < 2) throw new Error('CSV requires a header and at least one account row')
  const headers = rows[0].map((header) => header.trim().toLowerCase())
  if (new Set(headers).size !== headers.length) throw new Error('CSV header contains duplicates')
  for (const header of headers) {
    if (!ACCOUNT_FIELDS.has(header) && !header.startsWith('credential.')) {
      throw new Error(`CSV header is not supported: ${header}`)
    }
  }
  return rows.slice(1).map((cells, rowIndex) => {
    if (cells.length > headers.length) throw new Error(`CSV row ${rowIndex + 1}: too many columns`)
    const record: Record<string, unknown> = { credentials: {} }
    const credentials = record.credentials as Record<string, unknown>
    headers.forEach((header, columnIndex) => {
      const value = cells[columnIndex] ?? ''
      if (!value) return
      if (header === 'credentials') {
        let parsed: unknown
        try {
          parsed = JSON.parse(value)
        } catch {
          throw new Error(`CSV row ${rowIndex + 1}: credentials must be valid JSON`)
        }
        if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
          throw new Error(`CSV row ${rowIndex + 1}: credentials must be a JSON object`)
        }
        Object.assign(credentials, parsed)
      } else if (header.startsWith('credential.')) {
        credentials[header.slice('credential.'.length)] = value
      } else {
        record[header] = value
      }
    })
    return asAccount(record, `CSV row ${rowIndex + 1}`)
  })
}

function parseText(input: string): SupplierAccountInput[] {
  return input.split(/\r?\n/).flatMap((rawLine, index) => {
    const line = rawLine.trim()
    if (!line || line.startsWith('#')) return []
    if (line.startsWith('{')) {
      try {
        return [asAccount(JSON.parse(line), `Text line ${index + 1}`)]
      } catch (error) {
        if (error instanceof SyntaxError) throw new Error(`Text line ${index + 1}: invalid JSON`)
        throw error
      }
    }
    const parts = line.split('|')
    if (parts.length < 5) throw new Error(`Text line ${index + 1}: expected platform|type|external_id|name|credential`)
    const [platform, type, externalID, name] = parts
    const rawCredential = parts.slice(4).join('|').trim()
    let credentials: Record<string, unknown>
    if (rawCredential.startsWith('{')) {
      try {
        credentials = JSON.parse(rawCredential) as Record<string, unknown>
      } catch {
        throw new Error(`Text line ${index + 1}: credentials must be valid JSON`)
      }
    } else if (type.trim() === 'oauth' || type.trim() === 'setup-token') {
      credentials = { access_token: rawCredential }
    } else if (type.trim() === 'apikey') {
      credentials = { api_key: rawCredential }
    } else {
      throw new Error(`Text line ${index + 1}: this account type requires JSON credentials`)
    }
    return [asAccount({ external_id: externalID, name, platform, type, credentials }, `Text line ${index + 1}`)]
  })
}

export function parseSupplierAccountImport(format: SupplierImportFormat, input: string): SupplierAccountInput[] {
  let accounts: SupplierAccountInput[]
  if (format === 'csv') {
    accounts = parseCSV(input)
  } else if (format === 'text') {
    accounts = parseText(input)
  } else {
    let parsed: unknown
    try {
      parsed = JSON.parse(input)
    } catch {
      throw new Error('JSON input is invalid')
    }
    const values = Array.isArray(parsed)
      ? parsed
      : parsed && typeof parsed === 'object' && Array.isArray((parsed as { accounts?: unknown }).accounts)
        ? (parsed as { accounts: unknown[] }).accounts
        : null
    if (!values) throw new Error('JSON must be an array or an object with an accounts array')
    accounts = values.map((value, index) => asAccount(value, `JSON item ${index + 1}`))
  }
  if (accounts.length === 0) throw new Error('At least one account is required')
  if (accounts.length > MAX_BATCH_SIZE) throw new Error(`A batch can contain at most ${MAX_BATCH_SIZE} accounts`)
  return accounts
}
