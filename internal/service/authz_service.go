package service

import (
	"fmt"
	"search-gin/internal/model"
)

// ─── 权限常量定义 ────────────────────────────────────────────────

type PermissionDef struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Group       string `json:"group"`
	Description string `json:"description"`
}

const (
	// 菜单权限
	PermMenuHome      = "menu:home"
	PermMenuSearch    = "menu:search"
	PermMenuPicture   = "menu:picture"
	PermMenuSetting   = "menu:setting"
	PermMenuSystem    = "menu:system"
	PermMenuImmersive = "menu:immersive"

	// 操作权限
	PermOpEdit      = "op:edit"
	PermOpTag       = "op:tag"
	PermOpMovieType = "op:movie:type"
	PermOpTranscode = "op:transcode"
	PermOpMerge     = "op:merge"
	PermOpCut       = "op:cut"
	PermOpTorrent   = "op:torrent"
	PermOpScan      = "op:scan"
	PermOpChat      = "op:chat"
	PermOpNetwork   = "op:network"
)

// AllPermissions 返回所有可用的权限定义
func AllPermissions() []PermissionDef {
	return []PermissionDef{
		{Key: PermMenuHome, Name: "首页", Group: "菜单", Description: "访问首页数据统计"},
		{Key: PermMenuSearch, Name: "搜索", Group: "菜单", Description: "访问多媒体搜索页面"},
		{Key: PermMenuPicture, Name: "图鉴", Group: "菜单", Description: "访问图片浏览页面"},
		{Key: PermMenuSetting, Name: "配置", Group: "菜单", Description: "访问系统配置页面"},
		{Key: PermMenuSystem, Name: "系统", Group: "菜单", Description: "访问系统信息页面"},
		{Key: PermMenuImmersive, Name: "沉浸", Group: "菜单", Description: "访问沉浸式播放页面"},
		{Key: PermOpEdit, Name: "编辑文件", Group: "操作", Description: "重命名、移动、删除文件"},
		{Key: PermOpTag, Name: "管理标签", Group: "操作", Description: "添加/清除文件标签"},
		{Key: PermOpMovieType, Name: "设置类型", Group: "操作", Description: "设置影片类型"},
		{Key: PermOpTranscode, Name: "转码", Group: "操作", Description: "视频转码操作"},
		{Key: PermOpMerge, Name: "合并", Group: "操作", Description: "文件合并操作"},
		{Key: PermOpCut, Name: "剪辑", Group: "操作", Description: "视频剪辑操作"},
		{Key: PermOpTorrent, Name: "磁力下载", Group: "操作", Description: "磁力链接下载管理"},
		{Key: PermOpScan, Name: "扫描索引", Group: "操作", Description: "触发文件索引扫描"},
		{Key: PermOpChat, Name: "AI 聊天", Group: "操作", Description: "使用 AI 聊天功能"},
		{Key: PermOpNetwork, Name: "网络管理", Group: "操作", Description: "节点发现和网络设置"},
	}
}

// AllPermissionKeys 返回所有权限 key
func AllPermissionKeys() []string {
	defs := AllPermissions()
	keys := make([]string, len(defs))
	for i, d := range defs {
		keys[i] = d.Key
	}
	return keys
}

// DefaultUserPermissions 返回普通用户的默认权限
func DefaultUserPermissions() []string {
	return []string{
		PermMenuHome,
		PermMenuSearch,
		PermMenuPicture,
		PermMenuImmersive,
		PermOpTorrent,
		PermOpChat,
	}
}

// HasPermission 检查指定权限切片中是否包含某个权限
func HasPermission(perms []string, key string) bool {
	for _, p := range perms {
		if p == key {
			return true
		}
	}
	return false
}

// resolveUserPermissions 合并角色权限和用户自定义权限
// 优先级：角色权限 + 用户自定义权限（叠加）→ 默认权限
func resolveUserPermissions(u model.User) []string {
	// 先解析角色权限
	rolePerms := resolveRolePermissions(u.Role)
	// 合并用户自定义权限
	if len(u.Permissions) > 0 {
		return mergePermissions(rolePerms, u.Permissions)
	}
	if len(rolePerms) > 0 {
		return rolePerms
	}
	// 都没有则用默认
	defaultPerms := GetOSSetting().DefaultPermissions
	if len(defaultPerms) > 0 {
		return defaultPerms
	}
	return DefaultUserPermissions()
}

// resolveRolePermissions 根据角色名解析权限
func resolveRolePermissions(role string) []string {
	if role == "" || role == "user" {
		return nil
	}
	setting := GetOSSetting()
	for _, r := range setting.Roles {
		if r.Name == role {
			return r.Permissions
		}
	}
	return nil
}

// mergePermissions 合并两个权限列表（去重）
func mergePermissions(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))
	for _, p := range a {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	for _, p := range b {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	return result
}

// GetUserPermissions 获取用户的有效权限列表
// super_admin 拥有所有权限，普通用户按：角色权限 + 自定义权限 → 默认权限 解析
func GetUserPermissions(username string, role string) []string {
	if role == AdminRole || username == AdminUsername {
		return AllPermissionKeys()
	}

	for _, u := range GetOSSettingUsers() {
		if u.Username == username {
			return resolveUserPermissions(u)
		}
	}

	// 用户不存在时按角色名解析
	if perms := resolveRolePermissions(role); len(perms) > 0 {
		return perms
	}
	defaultPerms := GetOSSetting().DefaultPermissions
	if len(defaultPerms) > 0 {
		return defaultPerms
	}
	return DefaultUserPermissions()
}

// ─── 角色管理 ────────────────────────────────────────────────────

// GetRole 按名称查找角色
func GetRole(name string) (model.Role, bool) {
	setting := GetOSSetting()
	for _, r := range setting.Roles {
		if r.Name == name {
			return r, true
		}
	}
	return model.Role{}, false
}

// AddRole 添加角色（name 不能与已有角色或内置角色冲突）
func AddRole(role model.Role) error {
	setting := GetOSSetting()
	for _, r := range setting.Roles {
		if r.Name == role.Name {
			return fmt.Errorf("角色 %q 已存在", role.Name)
		}
	}
	if role.Name == AdminRole || role.Name == "user" || role.Name == "" {
		return fmt.Errorf("角色名 %q 不可用", role.Name)
	}
	UpdateOSSetting(func(s model.Setting) model.Setting {
		s.Roles = append(s.Roles, role)
		return s
	})
	FlushDictionary(GetOSSetting().SelfPath)
	return nil
}

// UpdateRole 更新角色权限
func UpdateRole(name string, role model.Role) error {
	updated := false
	UpdateOSSetting(func(s model.Setting) model.Setting {
		for i, r := range s.Roles {
			if r.Name == name {
				s.Roles[i] = role
				updated = true
				break
			}
		}
		return s
	})
	if !updated {
		return fmt.Errorf("角色 %q 不存在", name)
	}
	FlushDictionary(GetOSSetting().SelfPath)
	return nil
}

// DeleteRole 删除角色
func DeleteRole(name string) error {
	if name == AdminRole || name == "user" || name == "" {
		return fmt.Errorf("不能删除内置角色")
	}
	deleted := false
	UpdateOSSetting(func(s model.Setting) model.Setting {
		idx := -1
		for i, r := range s.Roles {
			if r.Name == name {
				idx = i
				break
			}
		}
		if idx >= 0 {
			s.Roles = append(s.Roles[:idx], s.Roles[idx+1:]...)
			deleted = true
		}
		return s
	})
	if !deleted {
		return fmt.Errorf("角色 %q 不存在", name)
	}
	// 将该角色的用户的 role 重置为 "user"
	UpdateOSSetting(func(s model.Setting) model.Setting {
		for i, u := range s.Users {
			if u.Role == name {
				s.Users[i].Role = "user"
			}
		}
		return s
	})
	FlushDictionary(GetOSSetting().SelfPath)
	return nil
}
