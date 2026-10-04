package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"search-gin/internal/service"
	"search-gin/pkg/utils"
)

var initialized atomic.Bool

// InitInitializedFlag 启动时调用，根据配置赋值初始化状态
func InitInitializedFlag() { initialized.Store(service.GetOSSetting().AdminPassword != "") }

// MarkInitialized 初始化完成后翻转标记（PostInitSetup 成功时调用）
func MarkInitialized() { initialized.Store(true) }

// InitCheckMiddleware 检查系统是否已完成初始化，未初始化时返回 412
func InitCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 非 API 路径放行（静态资源/页面入口）
		if !strings.HasPrefix(path, "/api/") {
			c.Next()
			return
		}

		// 已初始化后，/api/init/setup 返回 403
		if path == "/api/init/setup" {
			if initialized.Load() {
				c.AbortWithStatusJSON(http.StatusForbidden, utils.NewFailByMsg("系统已初始化"))
				return
			}
			c.Next()
			return
		}

		if !initialized.Load() {
			utils.InfoFormat("系统未初始化，拒绝请求: %s", path)
			c.AbortWithStatusJSON(http.StatusPreconditionFailed, utils.NewFailByMsg("系统未初始化"))
			return
		}
		c.Next()
	}
}

// RequestLogger 记录失败的 HTTP 请求，以及（verbose 时的）慢请求：
//   - 状态码 >= 400：写入内存日志（前端系统日志页可查）与 gin.log，便于排查失败原因；
//     只记录路径不带 query，避免 streamToken 等凭据落进日志
//   - 耗时 > 5s：仅 verbose（非生产）记录，避免生产环境日志膨胀
//
// 静态资源（非 /api/）的失败不记录，防止前端资源 404 刷屏。
func RequestLogger(verbose bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		if status >= http.StatusBadRequest && strings.HasPrefix(path, "/api/") {
			service.LogMem.Add("HTTP 请求失败: %s %s → %d (%v)",
				c.Request.Method, path, status, duration.Round(time.Millisecond))
			if status >= http.StatusInternalServerError {
				utils.ErrorFormat("HTTP 请求失败: %s %s → %d (%v), 客户端=%s",
					c.Request.Method, path, status, duration.Round(time.Millisecond), c.ClientIP())
			} else {
				utils.WarnFormat("HTTP 请求失败: %s %s → %d (%v), 客户端=%s",
					c.Request.Method, path, status, duration.Round(time.Millisecond), c.ClientIP())
			}
			return
		}

		if verbose && duration > 5*time.Second {
			utils.InfoFormat("慢请求 [%s] %s %d %v",
				c.Request.Method, path, status, duration)
		}
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		skipPaths := []string{
			"/api/login",
			"/api/init/setup",
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
			// 文件流 token 路径在 :10081 上也需要通过 streamToken 而非 Bearer Token 访问，
			// 此处跳过 AuthMiddleware，由随后注册的 StreamTokenAuth 中间件校验。
			"/api/stream/",
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
		// 注意：必须在 skip path 检查之后，避免跨节点认证递归
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

			// 未知 IP → 尝试反向心跳验证，通过则自动加入集群
			if service.TryVerifyAndAddPeer(host) {
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
		if !valid {
			c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("认证失败"))
			c.Abort()
			return
		}
		c.Set("username", tokenInfo.Username)
		c.Set("role", tokenInfo.Role)
		c.Set("permissions", tokenInfo.Permissions)
		// 原始 token 一并存入，供登录态吊销（/api/logout）取用
		c.Set("token", token)
		c.Next()
	}
}
