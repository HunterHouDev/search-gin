package handler

import (
	"net/http"

	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

// GetAllPermissions 返回所有可用的权限定义
func GetAllPermissions(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	res := utils.NewSuccess()
	res.Data = service.AllPermissions()
	c.JSON(http.StatusOK, res)
}

// GetUserPermissions 返回指定用户的权限列表
func GetUserPermissions(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("用户名不能为空"))
		return
	}

	perms := service.GetUserPermissions(username, "")
	res := utils.NewSuccess()
	res.Data = perms
	c.JSON(http.StatusOK, res)
}

// UpdateUserPermissions 更新指定用户的权限
func UpdateUserPermissions(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	type UpdatePermsRequest struct {
		Username    string   `json:"username"`
		Permissions []string `json:"permissions"`
	}

	req, err := BindJSON[UpdatePermsRequest](c, "无效的请求")
	if err != nil {
		return
	}
	if req.Username == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("用户名不能为空"))
		return
	}
	if req.Username == service.AdminUsername {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("不能修改管理员权限"))
		return
	}

	// 校验所有权限 key 是否合法
	validKeys := make(map[string]bool)
	for _, d := range service.AllPermissions() {
		validKeys[d.Key] = true
	}
	for _, p := range req.Permissions {
		if !validKeys[p] {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的权限: "+p))
			return
		}
	}

	updated := false
	service.UpdateOSSetting(func(s model.Setting) model.Setting {
		for i, u := range s.Users {
			if u.Username == req.Username {
				s.Users[i].Permissions = req.Permissions
				updated = true
				break
			}
		}
		return s
	})
	if !updated {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("用户不存在"))
		return
	}
	UseApp().config.Flush(service.SettingFileName)

	// 强制该用户的所有 token 重新登录（让下次请求时同步新权限）
	// 现有 token 在 ValidateTokenWithInfo 中会自动同步最新权限
	c.JSON(http.StatusOK, utils.NewSuccessByMsg("权限更新成功，用户下次请求时将生效"))
}

// ─── 角色管理 ────────────────────────────────────────────────────

// GetRoles 返回所有自定义角色
func GetRoles(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	setting := UseApp().config.Get()
	res := utils.NewSuccess()
	res.Data = setting.Roles
	c.JSON(http.StatusOK, res)
}

// CreateRole 创建自定义角色
func CreateRole(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	req, err := BindJSON[model.Role](c, "无效的请求")
	if err != nil {
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("角色名不能为空"))
		return
	}

	// 校验权限 key
	validKeys := make(map[string]bool)
	for _, d := range service.AllPermissions() {
		validKeys[d.Key] = true
	}
	for _, p := range req.Permissions {
		if !validKeys[p] {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的权限: "+p))
			return
		}
	}

	if err := service.AddRole(req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, utils.NewSuccessByMsg("角色创建成功"))
}

// UpdateRole 更新角色
func UpdateRole(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("角色名不能为空"))
		return
	}

	req, err := BindJSON[model.Role](c, "无效的请求")
	if err != nil {
		return
	}

	// 校验权限 key
	validKeys := make(map[string]bool)
	for _, d := range service.AllPermissions() {
		validKeys[d.Key] = true
	}
	for _, p := range req.Permissions {
		if !validKeys[p] {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的权限: "+p))
			return
		}
	}

	if err := service.UpdateRole(name, req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, utils.NewSuccessByMsg("角色更新成功"))
}

// DeleteRole 删除角色
func DeleteRole(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("角色名不能为空"))
		return
	}

	if err := service.DeleteRole(name); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, utils.NewSuccessByMsg("角色已删除"))
}

// UpdateUserRole 更新用户的角色
func UpdateUserRole(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	type UpdateUserRoleReq struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	req, err := BindJSON[UpdateUserRoleReq](c, "无效的请求")
	if err != nil {
		return
	}
	if req.Username == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("用户名不能为空"))
		return
	}
	if req.Username == service.AdminUsername {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("不能修改管理员角色"))
		return
	}

	updated := false
	service.UpdateOSSetting(func(s model.Setting) model.Setting {
		for i, u := range s.Users {
			if u.Username == req.Username {
				s.Users[i].Role = req.Role
				updated = true
				break
			}
		}
		return s
	})
	if !updated {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("用户不存在"))
		return
	}
	UseApp().config.Flush(service.SettingFileName)
	c.JSON(http.StatusOK, utils.NewSuccessByMsg("用户角色更新成功"))
}
