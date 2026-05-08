package middleware

import (
	"search-gin/pkg/utils"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	ginlogrus "github.com/toorop/gin-logrus"
)

func CORSConfig() cors.Config {
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowCredentials = true
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
