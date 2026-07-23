package handler

import (
	"net/http"

	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/middleware"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// PostInitSetup 初始化设置管理员密码（仅首次可调用）
// 已初始化时由 InitCheckMiddleware 返回 403
func PostInitSetup(c *gin.Context) {
	req, err := BindJSON[struct {
		Password string `json:"password"`
	}](c, "密码不能为空")
	if err != nil {
		return
	}
	if req.Password == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("密码不能为空"))
		return
	}

	service.UpdateOSSetting(func(s model.Setting) model.Setting {
		s.AdminPassword = req.Password
		return s
	})
	service.CacheAdminPasswordHash()
	middleware.MarkInitialized()
	if err := service.FlushDictionary(service.SettingFileName); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewFailByMsg("密码保存失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessByMsg("管理员密码设置成功，请登录"))
}

func Login(c *gin.Context) {
	req, err := BindJSON[LoginRequest](c, "无效的请求")
	if err != nil {
		return
	}

	result := service.LoginUser(req.Username, req.Password)
	if !result.Success {
		c.JSON(http.StatusUnauthorized, utils.NewFailByMsg(result.Message))
		return
	}

	res := utils.NewSuccess()
	res.Data = gin.H{
		"token":       result.Token,
		"expireIn":    result.ExpireIn,
		"role":        result.Role,
		"username":    result.Username,
		"permissions": result.Permissions,
	}
	c.JSON(http.StatusOK, res)
}

func requireAdmin(c *gin.Context) bool {
	roleVal, _ := c.Get("role")
	usernameVal, _ := c.Get("username")
	r, _ := roleVal.(string)
	u, _ := usernameVal.(string)
	utils.InfoFormat("requireAdmin 检查: role=%q(username=%q), RequireAdminWithName=%v", r, u, service.RequireAdminWithName(r, u))

	// 兼容旧 token：仅 admin 用户在中间件未设 role 时放行
	// （ValidateTokenWithInfo 会自动补全 admin 用户的 role，旧 token 首次校验后即不再进入此分支）
	if r == "" && u == service.AdminUsername {
		utils.InfoFormat("requireAdmin: 旧 token 兼容放行(admin)")
		return true
	}
	// role 和 username 均为空 → 拒绝（非 admin 用户必须带完整 token 信息）
	if r == "" && u == "" {
		utils.InfoFormat("requireAdmin: 拒绝无 role/username 的旧 token")
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("无权限执行此操作"))
		return false
	}

	if !service.RequireAdminWithName(r, u) {
		utils.InfoFormat("requireAdmin 拒绝: 中间件设置的 role=%v, username=%v", r, u)
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("无权限执行此操作"))
		return false
	}
	return true
}

// requirePermission 检查当前用户是否拥有指定权限
// super_admin 拥有所有权限，无需逐项检查
func requirePermission(c *gin.Context, perm string) bool {
	roleVal, _ := c.Get("role")
	usernameVal, _ := c.Get("username")
	permsVal, _ := c.Get("permissions")
	r, _ := roleVal.(string)
	u, _ := usernameVal.(string)
	perms, _ := permsVal.([]string)

	// super_admin 放行所有权限
	if service.RequireAdminWithName(r, u) {
		return true
	}

	if !service.HasPermission(perms, perm) {
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("无权限执行此操作"))
		return false
	}
	return true
}
