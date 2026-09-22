import { describe, expect, it } from 'vitest'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'

describe('supplier role locale', () => {
  it('uses a human-readable supplier role label', () => {
    expect(zh.admin.users.roles.supplier).toBe('供应商')
    expect(en.admin.users.roles.supplier).toBe('Supplier')
  })
})
