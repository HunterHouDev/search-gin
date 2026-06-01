package middleware

import (
 "net/http"
 "search-gin/pkg/consts"
 "search-gin/pkg/utils"
 "strings"

 "github.com/gin-gonic/gin"
)

// AuthMiddleware token验证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 基础安全检测：拒绝路径遍历攻击
		if strings.Contains(path, "..") {
			c.JSON(http.StatusForbidden, utils.NewFailByMsg("非法的请求路径"))
			c.Abort()
			return
		}

		// 不需要认证的路径（精确前缀匹配）
		skipPaths := []string{
			"/api/login",
			"/login",
			"/",
			"/index.html",
			"/api/file/",      // 文件流不需要token（由视频播放器使用）
			"/api/png/",        // 图片资源
			"/api/jpg/",        // 图片资源
			"/api/tempimage/",  // 临时图片
		}

		// 检查是否在跳过列表中
		for _, skipPath := range skipPaths {
			if strings.HasPrefix(path, skipPath) {
				c.Next()
				return
			}
		}
		
		// 从请求头获取token
		authHeader := c.GetHeader("Authorization")
		token := ""

		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// WebSocket 无法自定义 Header，支持从 query 参数获取 token
		if token == "" {
			token = c.Query("token")
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("未授权访问"))
			c.Abort()
			return
		}
		
	// 验证token
	tokenInfo, valid := consts.ValidateTokenWithInfo(token)
	if !valid {
		c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("token无效或已过期"))
		c.Abort()
		return
	}

	// 将用户名和角色注入 context，供后续 handler 使用
	c.Set("username", tokenInfo.Username)
	c.Set("role", tokenInfo.Role)

	c.Next()
	}
}
