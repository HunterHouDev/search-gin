import { defineStore } from 'pinia'
import { SUPER_ADMIN_ROLE, ALL_PERMISSION_KEYS } from 'src/types/permission'
import {
  getAuthPermissions,
  getAuthRole,
  getAuthUsername,
  saveAuthSession,
} from 'src/utils/authStorage'

export const usePermissionStore = defineStore('permission', {
  // 直接以本窗口 sessionStorage 为数据源，消费方无需再显式恢复
  state: () => ({
    permissions: getAuthPermissions(),
    role: getAuthRole(),
    username: getAuthUsername(),
  }),

  getters: {
    isSuperAdmin(state): boolean {
      return state.role === SUPER_ADMIN_ROLE
    },

    hasPermission(state): (perm: string) => boolean {
      return (perm: string) => {
        if (this.isSuperAdmin) return true
        return state.permissions.includes(perm)
      }
    },

    hasAnyPermission(state): (perms: string[]) => boolean {
      return (perms: string[]) => {
        if (this.isSuperAdmin) return true
        return perms.some(p => state.permissions.includes(p))
      }
    },
  },

  actions: {
    setFromLogin(
      role: string,
      username: string,
      permissions: string[],
      token: string,
      expireIn?: number
    ) {
      this.role = role
      this.username = username
      this.permissions = this.isSuperAdmin ? [...ALL_PERMISSION_KEYS] : permissions
      // 由 authStorage 统一写入本窗口 sessionStorage
      saveAuthSession({ token, role, username, permissions: this.permissions, expireIn })
    },
  },
})
