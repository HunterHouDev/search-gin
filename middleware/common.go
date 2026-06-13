package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		skipPaths := []string{
		"/api/login",
		 "/login",
		 "/",
		 "/index.html",
		 "/api/ws",
		 "/api/lanPeers",
		 "/api/heartBeat",
		}

		// 集群节点间转发携带此头，跳过认证
		if c.GetHeader("X-Search-Gin-Remote") == "true" {
			c.Next()
			return
		}

		for _, sp := range skipPaths {
			if strings.HasSuffix(sp, "/") {
				if strings.HasPrefix(path, sp) {
					c.Next()
					return
				}
			} else {
				if path == sp {
					c.Next()
					return
				}
			}
		}
		authHeader := c.GetHeader("Authorization")
		token := ""
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("未认证"))
			c.Abort()
			return
		}
		tokenInfo, valid := consts.ValidateTokenWithInfo(token)
		if !valid {
			c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("认证失败"))
			c.Abort()
			return
		}
		c.Set("username", tokenInfo.Username)
		c.Set("role", tokenInfo.Role)
		c.Next()
	}
}
