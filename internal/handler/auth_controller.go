package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"search-gin/pkg/consts"
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

	issueToken := func(username, role string) {
		tokenBytes := make([]byte, 16)
		if _, err := rand.Read(tokenBytes); err != nil {
			c.JSON(http.StatusInternalServerError, utils.NewFailByMsg("生成token失败，系统错误"))
			return
		}
		token := hex.EncodeToString(tokenBytes)
		consts.SetToken(token, time.Now().Add(2*time.Hour), username, role)
		res := utils.NewSuccess()
		res.Data = gin.H{
			"token":    token,
			"expireIn": 2 * 3600,
			"role":     role,
			"username": username,
		}
		c.JSON(http.StatusOK, res)
	}

	if req.Username == consts.AdminUsername && consts.VerifyPassword(req.Password, consts.AdminPasswordHash()) {
		issueToken(consts.AdminUsername, consts.AdminRole)
		return
	}

	for _, user := range consts.GetOSSettingUsers() {
		if user.Username == req.Username && consts.VerifyPassword(req.Password, user.Password) {
			if user.ExpireDate != "" {
				expireTime, err := time.Parse("2006-01-02", user.ExpireDate)
				if err == nil && time.Now().After(expireTime) {
					c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("用户已过期，请联系管理员"))
					return
				}
			}
			issueToken(user.Username, user.Role)
			return
		}
	}

	c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("用户名或密码错误"))
}

func requireAdmin(c *gin.Context) bool {
	role, _ := c.Get("role")
	if r, ok := role.(string); !ok || r != consts.AdminRole {
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("无权限执行此操作"))
		return false
	}
	return true
}
