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

// Logout 登出：立即吊销当前请求携带的 token，使其后续无法再通过认证。
// 已过期或已吊销的 token 会在中间件层被拦下，此处只处理有效 token。
func Logout(c *gin.Context) {
	service.RevokeToken(c.GetString("token"))
	c.JSON(http.StatusOK, utils.NewSuccessByMsg("已退出登录"))
}

// requireAdmin 校验管理员身份。
// 旧 token 兼容（role 为空但 username 为 admin）由 RequireAdminWithName 统一覆盖，
// 非管理员与信息不全的 token 同样落到这里被拒绝。
func requireAdmin(c *gin.Context) bool {
	roleVal, _ := c.Get("role")
	usernameVal, _ := c.Get("username")
	r, _ := roleVal.(string)
	u, _ := usernameVal.(string)

	if !service.RequireAdminWithName(r, u) {
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
