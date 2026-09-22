import type { UserRole } from '@/types'

export function resolveAuthenticatedHome(role: UserRole | null | undefined): string {
  if (role === 'admin') return '/admin/dashboard'
  if (role === 'supplier') return '/supplier/accounts'
  return '/dashboard'
}

export function canAccessRoleRoute(
  role: UserRole | null | undefined,
  route: { requiresAdmin?: boolean; requiresSupplier?: boolean; allowsSupplier?: boolean },
): boolean {
  if (route.requiresAdmin) return role === 'admin'
  if (route.requiresSupplier) return role === 'supplier'
  return role !== 'supplier' || route.allowsSupplier === true
}

export function resolveCompletedSetupRedirectPath(
  isAuthenticated: boolean,
  role: UserRole | null | undefined,
): string {
  if (!isAuthenticated) {
    return '/login'
  }

  return resolveAuthenticatedHome(role)
}
