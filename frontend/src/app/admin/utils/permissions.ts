import type { AuthUser } from '../../../api'

export type AdminPermission =
  | 'contest:read'
  | 'contest:write'
  | 'challenge:read'
  | 'challenge:write'
  | 'attachment:write'
  | 'announcement:read'
  | 'announcement:write'
  | 'submission:read'
  | 'instance:read'
  | 'instance:write'
  | 'user:read'
  | 'user:write'
  | 'audit:read'

export function canAccessAdmin(user: AuthUser | null): boolean {
  if (!user) return false
  return user.role === 'admin' || user.role === 'ops' || user.role === 'author'
}

export function hasAdminPermission(user: AuthUser | null, permission: AdminPermission): boolean {
  if (!user) return false
  const role = user.role

  const permissions: Record<string, Record<AdminPermission, boolean>> = {
    admin: {
      'contest:read': true,
      'contest:write': true,
      'challenge:read': true,
      'challenge:write': true,
      'attachment:write': true,
      'announcement:read': true,
      'announcement:write': true,
      'submission:read': true,
      'instance:read': true,
      'instance:write': true,
      'user:read': true,
      'user:write': true,
      'audit:read': true,
    },
    ops: {
      'contest:read': true,
      'contest:write': false,
      'challenge:read': true,
      'challenge:write': false,
      'attachment:write': true,
      'announcement:read': true,
      'announcement:write': false,
      'submission:read': true,
      'instance:read': true,
      'instance:write': true,
      'user:read': false,
      'user:write': false,
      'audit:read': true,
    },
    author: {
      'contest:read': false,
      'contest:write': false,
      'challenge:read': true,
      'challenge:write': true,
      'attachment:write': true,
      'announcement:read': false,
      'announcement:write': false,
      'submission:read': false,
      'instance:read': false,
      'instance:write': false,
      'user:read': false,
      'user:write': false,
      'audit:read': false,
    },
  }

  const rolePerms = permissions[role]
  if (!rolePerms) return false
  return Boolean(rolePerms[permission])
}

