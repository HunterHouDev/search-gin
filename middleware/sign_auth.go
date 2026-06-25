package middleware

import (
	"net/http"
	"search-gin/internal/service"
	"search-gin/pkg/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// SignAuthMiddleware 签名 URL 校验中间件
// 校验请求中的 sign 和 expire 参数，防止未授权访问文件流
func SignAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawPath; raw != "" {
			path = raw
		}
		path = stripQuery(path)
		if !utils.VerifySignedRequest(path, c.Request.URL.Query()) {
			c.JSON(http.StatusForbidden, gin.H{"fail": true, "msg": "签名无效或已过期"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// StreamTokenAuth 文件流 token 校验中间件（用于 :10082 端口）
// 校验 URL 中的 token 查询参数，复用 API 侧的 ValidateTokenWithInfo
func StreamTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"fail": true, "msg": "缺少 token"})
			c.Abort()
			return
		}
		if _, valid := service.ValidateTokenWithInfo(token); !valid {
			c.JSON(http.StatusForbidden, gin.H{"fail": true, "msg": "token 无效或已过期"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func stripQuery(path string) string {
	if i := strings.Index(path, "?"); i >= 0 {
		return path[:i]
	}
	return path
}
