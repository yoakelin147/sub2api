import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const source = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../AccountsView.vue'), 'utf8')

describe('AccountsView supplier source', () => {
  it('links supplier-owned accounts to supplier governance and labels platform-owned accounts', () => {
    expect(source).toContain("path: '/admin/suppliers'")
    expect(source).toContain('row.supplier_id')
    expect(source).toContain('supplier.admin.supplierSource')
    expect(source).toContain('supplier.admin.platformOwned')
  })
})
