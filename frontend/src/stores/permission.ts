import { defineStore } from 'pinia'
import { SUPER_ADMIN_ROLE, ALL_PERMISSION_KEYS } from 'src/types/permission'

export const usePermissionStore = defineStore('permission', {
  state: () => ({
    permissions: [] as string[],
    role: '',
    username: '',
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
    loadFromSession() {
      this.role = sessionStorage.getItem('userRole') || ''
      this.username = sessionStorage.getItem('username') || ''
      try {
        const stored = sessionStorage.getItem('userPermissions')
        this.permissions = stored ? JSON.parse(stored) : []
      } catch {
        this.permissions = []
      }
    },

    setFromLogin(role: string, username: string, permissions: string[]) {
      this.role = role
      this.username = username
      this.permissions = this.isSuperAdmin ? [...ALL_PERMISSION_KEYS] : permissions
      sessionStorage.setItem('userRole', role)
      sessionStorage.setItem('username', username)
      sessionStorage.setItem('userPermissions', JSON.stringify(this.permissions))
    },

    clear() {
      this.role = ''
      this.username = ''
      this.permissions = []
      sessionStorage.removeItem('userRole')
      sessionStorage.removeItem('username')
      sessionStorage.removeItem('userPermissions')
    },
  },
})
