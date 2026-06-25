package handler

import (
	"net/http"

	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的请求"))
		return
	}

	result := service.LoginUser(req.Username, req.Password)
	if !result.Success {
		c.JSON(http.StatusUnauthorized, utils.NewFailByMsg(result.Message))
		return
	}

	res := utils.NewSuccess()
	res.Data = gin.H{
		"token":    result.Token,
		"expireIn": result.ExpireIn,
		"role":     result.Role,
		"username": result.Username,
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
