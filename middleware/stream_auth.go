package middleware

import (
	"net/http"
	"search-gin/pkg/utils"
	"time"

	"github.com/gin-gonic/gin"
)

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
