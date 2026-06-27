export interface PermissionDef {
  key: string
  name: string
  group: string
  description: string
}

// 菜单权限
export const PERM_MENU_HOME = 'menu:home'
export const PERM_MENU_SEARCH = 'menu:search'
export const PERM_MENU_PICTURE = 'menu:picture'
export const PERM_MENU_SETTING = 'menu:setting'
export const PERM_MENU_SYSTEM = 'menu:system'
export const PERM_MENU_IMMERSIVE = 'menu:immersive'

// 操作权限
export const PERM_OP_EDIT = 'op:edit'
export const PERM_OP_TAG = 'op:tag'
export const PERM_OP_MOVIE_TYPE = 'op:movie:type'
export const PERM_OP_TRANSCODE = 'op:transcode'
export const PERM_OP_MERGE = 'op:merge'
export const PERM_OP_CUT = 'op:cut'
export const PERM_OP_TORRENT = 'op:torrent'
export const PERM_OP_SCAN = 'op:scan'
export const PERM_OP_CHAT = 'op:chat'
export const PERM_OP_NETWORK = 'op:network'

export const ALL_PERMISSION_KEYS = [
  PERM_MENU_HOME,
  PERM_MENU_SEARCH,
  PERM_MENU_PICTURE,
  PERM_MENU_SETTING,
  PERM_MENU_SYSTEM,
  PERM_MENU_IMMERSIVE,
  PERM_OP_EDIT,
  PERM_OP_TAG,
  PERM_OP_MOVIE_TYPE,
  PERM_OP_TRANSCODE,
  PERM_OP_MERGE,
  PERM_OP_CUT,
  PERM_OP_TORRENT,
  PERM_OP_SCAN,
  PERM_OP_CHAT,
  PERM_OP_NETWORK,
] as const

export const DEFAULT_USER_PERMISSIONS = [
  PERM_MENU_HOME,
  PERM_MENU_SEARCH,
  PERM_MENU_PICTURE,
  PERM_MENU_IMMERSIVE,
  PERM_OP_TORRENT,
  PERM_OP_CHAT,
]

// super_admin 固定角色，拥有全部权限
export const SUPER_ADMIN_ROLE = 'super_admin'

export interface RoleDef {
  name: string
  label: string
  permissions: string[]
}
