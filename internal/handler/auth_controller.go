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
	if r, ok := role.(string); !ok || !service.RequireAdmin(r) {
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("无权限执行此操作"))
		return false
	}
	return true
}
