package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"search-gin/internal/service"
	"search-gin/pkg/utils"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		skipPaths := []string{
			"/api/login",
			"/login",
			"/index.html",
			"/api/ws",
			"/api/events",
			"/api/lanPeers",
			"/api/heartBeat",
			"/api/authorImage/",
			"/css/",
			"/js/",
			"/assets/",
			"/icons/",
			"/favicon.ico",
		}

		// 单独处理根路径（不能用前缀匹配，否则所有 /api/* 都会被跳过）
		if path == "/" {
			c.Next()
			return
		}

		// 免认证路径优先检查（心跳等），防止递归触发 X-Search-Gin-Remote 验证
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

		// 集群节点间转发携带此头，校验来源 IP 为已知 peer 后跳过认证
		// 注意：必须在 skip path 检查之后，避免 verifyPeer 反向心跳形成递归死锁
		if c.GetHeader("X-Search-Gin-Remote") == "true" {
			host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
			if err != nil {
				c.JSON(http.StatusForbidden, utils.NewFailByMsg("禁止访问"))
				c.Abort()
				return
			}

			// 已知 peer → 直接放行
			if service.IsKnownPeerIP(host) {
				c.Next()
				return
			}

			// 未知 IP → 反向心跳验证，通过则自动加入集群并放行
			utils.InfoFormat("首次检测到潜在集群节点 %s，尝试反向验证...", host)
			if service.TryVerifyAndAddPeer(host) {
				utils.InfoFormat("新节点已通过验证并加入集群: %s", host)
				c.Next()
				return
			}

			utils.InfoFormat("拒绝来自非集群节点的 X-Search-Gin-Remote 请求: %s", c.Request.RemoteAddr)
			c.JSON(http.StatusForbidden, utils.NewFailByMsg("禁止访问"))
			c.Abort()
			return
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
		tokenInfo, valid := service.ValidateTokenWithInfo(token)
		utils.InfoFormat("AuthMiddleware: token=%s valid=%v username=%q role=%q", token[:8]+"...", valid, tokenInfo.Username, tokenInfo.Role)
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
