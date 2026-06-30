package middleware

import (
	"net/http"
	"search-gin/pkg/utils"
	"strings"
	"time"

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
			c.JSON(http.StatusForbidden, utils.NewFailByMsg("签名无效或已过期"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// StreamTokenAuth 文件流 token 校验中间件（用于 :10082 端口）
// 解密 streamToken（AES-256-GCM 加密的过期时间戳），只校验是否过期，不依赖 session map。
// 跨机器节点共享同一密钥（配置在 setting.json streamSecret 字段），可互相验证。
func StreamTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("streamToken")
		if token == "" {
			c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("缺少 streamToken"))
			c.Abort()
			return
		}
		expire, err := utils.DecryptStreamToken(token)
		if err != nil || time.Now().Unix() > expire {
			c.JSON(http.StatusForbidden, utils.NewFailByMsg("streamToken 无效或已过期"))
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
