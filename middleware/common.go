package middleware

import (
	"net/http"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	ginlogrus "github.com/toorop/gin-logrus"
)

func CORSConfig() cors.Config {
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowCredentials = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Range", "Accept-Ranges", "Content-Range"}
	config.ExposeHeaders = []string{"Content-Length", "Content-Range", "Accept-Ranges", "Content-Type"}
	return config
}

func LoggerMiddleware() gin.HandlerFunc {
	return ginlogrus.Logger(utils.NewLogger())
}

func SlowRequestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		duration := time.Since(start)

		if duration > 5*time.Second {
			utils.InfoFormat("慢请求 [%s] %s %d %v",
					c.Request.Method, path, c.Writer.Status(), duration)
		}
	}
}

// AuthMiddleware token验证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 不需要认证的路径
		skipPaths := []string{
			"/api/login",
			"/login",
			"/",
			"/index.html",
			"/api/file/",   // 文件流不需要token（由视频播放器使用）
			"/api/png/",    // 图片资源
			"/api/jpg/",    // 图片资源
			"/api/tempimage/", // 临时图片
		}
		
		// 检查是否在跳过列表中
		for _, skipPath := range skipPaths {
			if strings.HasPrefix(c.Request.URL.Path, skipPath) {
				c.Next()
				return
			}
		}
		
		// 从请求头获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("未授权访问"))
			c.Abort()
			return
		}
		
		// 提取token（格式：Bearer <token>）
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("无效的认证格式"))
			c.Abort()
			return
		}
		
		token := strings.TrimPrefix(authHeader, "Bearer ")
		
		// 验证token
		if !consts.ValidateToken(token) {
			c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("token无效或已过期"))
			c.Abort()
			return
		}
		
		c.Next()
	}
}
