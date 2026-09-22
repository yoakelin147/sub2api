import { describe, expect, it } from 'vitest'
import {
  canAccessRoleRoute,
  resolveAuthenticatedHome,
  resolveCompletedSetupRedirectPath,
} from '@/router/setupRedirect'

describe('role-aware routing', () => {
  it.each([
    ['admin', '/admin/dashboard'],
    ['user', '/dashboard'],
    ['supplier', '/supplier/accounts'],
  ] as const)('routes %s to its own home', (role, expected) => {
    expect(resolveAuthenticatedHome(role)).toBe(expected)
    expect(resolveCompletedSetupRedirectPath(true, role)).toBe(expected)
  })

  it('routes anonymous setup completion to login', () => {
    expect(resolveCompletedSetupRedirectPath(false, null)).toBe('/login')
  })

  it.each([
    ['admin', true, false, true],
    ['admin', false, true, false],
    ['user', false, false, true],
    ['user', true, false, false],
    ['user', false, true, false],
    ['supplier', false, true, true],
    ['supplier', false, false, false],
    ['supplier', true, false, false],
  ] as const)(
    'enforces route role for %s (admin=%s supplier=%s)',
    (role, requiresAdmin, requiresSupplier, expected) => {
      expect(canAccessRoleRoute(role, { requiresAdmin, requiresSupplier })).toBe(expected)
    },
  )
})
