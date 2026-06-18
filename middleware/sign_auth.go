package middleware

import (
	"net/http"
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

func stripQuery(path string) string {
	if i := strings.Index(path, "?"); i >= 0 {
		return path[:i]
	}
	return path
}
