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
	role, _ := c.Get("role")
	username, _ := c.Get("username")
	r, _ := role.(string)
	u, _ := username.(string)
	utils.InfoFormat("requireAdmin 检查: role=%q(username=%q), RequireAdminWithName=%v", r, u, service.RequireAdminWithName(r, u))

	// 兼容旧 token：中间件未设 role/username 时放行（已通过认证即合法）
	if r == "" && u == "" {
		utils.InfoFormat("requireAdmin: 旧 token 兼容放行")
		return true
	}

	if !service.RequireAdminWithName(r, u) {
		utils.InfoFormat("requireAdmin 拒绝: 中间件设置的 role=%v, username=%v", c.Keys)
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("无权限执行此操作"))
		return false
	}
	return true
}
