package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"search-gin/pkg/consts"
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no token"})
			c.Abort()
			return
		}
		tokenInfo, valid := consts.ValidateTokenWithInfo(token)
		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		c.Set("username", tokenInfo.Username)
		c.Set("role", tokenInfo.Role)
		c.Next()
	}
}
