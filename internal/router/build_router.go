package router

import (
	"os"
	"os/exec"
	"search-gin/internal/env"
	"search-gin/internal/handler"
	"search-gin/internal/service"
	"search-gin/middleware"
	"strings"
	"syscall"
	"time"

	"search-gin/pkg/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// buildCORSConfig 构建 CORS 配置
//
// AllowCredentials 已注释（仅开发模式注释，生产模式未启用），因此不存在
// AllowCredentials + AllowOrigins[*] 冲突。局域网场景 AllowOrigins[*] 合理。
// 生产环境可通过 ALLOWED_ORIGINS 环境变量限制来源。
func buildCORSConfig() cors.Config {
	config := cors.DefaultConfig()
	if env.IsProd {
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		if allowedOrigins != "" {
			config.AllowOrigins = strings.Split(allowedOrigins, ",")
		} else {
			config.AllowOrigins = []string{"*"}
		}
		// config.AllowCredentials = true
	} else {
		config.AllowOrigins = []string{"*"}
	}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Range", "Accept-Ranges", "Content-Range"}
	config.ExposeHeaders = []string{"Content-Length", "Content-Range", "Accept-Ranges", "Content-Type"}
	return config
}

// buildCommonMiddleware 构建通用中间件（CORS + 日志 + Recovery）
func buildCommonMiddleware(router *gin.Engine) {
	router.Use(cors.New(buildCORSConfig()))
	router.Use(middleware.CustomRecovery())
}

func buildStreamMiddleware(router *gin.Engine) {
	router.GET("/api/stream/file/:id", handler.GetFile)
	router.GET("/api/stream/png/:path", handler.GetPng)
	router.GET("/api/stream/jpg/:path", handler.GetJpg)
	router.GET("/api/stream/GetFileByPathUseEncode/:path", handler.GetFileByPathUseEncode)
}

func buildShutdownRoutes(router *gin.Engine, sigChan chan os.Signal) {
	router.GET("api/close", func(c *gin.Context) {
		role, _ := c.Get("role")
		if r, ok := role.(string); !ok || r != service.AdminRole {
			c.JSON(403, utils.NewFailByMsg("无权限执行此操作"))
			return
		}
		c.String(200, "即将关闭服务器")
		sigChan <- syscall.SIGTERM
	})
	router.GET("api/restart", func(c *gin.Context) {
		role, _ := c.Get("role")
		if r, ok := role.(string); !ok || r != service.AdminRole {
			c.JSON(403, utils.NewFailByMsg("无权限执行此操作"))
			return
		}
		c.String(200, "正在重启服务器")
		go func() {
			defer utils.RecoverPanic()
			time.Sleep(200 * time.Millisecond)
			sigChan <- syscall.SIGTERM
			time.Sleep(2 * time.Second)
			exe, err := os.Executable()
			if err != nil {
				utils.ErrorFormat("获取可执行文件路径失败: %v", err)
				return
			}
			cmd := exec.Command(exe, os.Args[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Start(); err != nil {
				utils.ErrorFormat("重启自身失败: %v", err)
			}
		}()
	})
}

// BuildAPIRouter 构建 API 业务路由（端口 10081）：需要认证
func BuildAPIRouter(sigChan chan os.Signal) *gin.Engine {
	if env.IsProd {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	buildCommonMiddleware(router)

	// 初始化接口（无需认证，首次部署时使用）
	router.GET("/api/init", handler.GetInitStatus)
	router.POST("/api/init/setup", handler.PostInitSetup)

	router.Use(middleware.AuthMiddleware())

	if !env.IsProd {
		router.Use(middleware.SlowRequestLogger())
	}

	router.NoRoute(handler.Index)
	router.GET("/", handler.Index)

	router.POST("/api/login", handler.Login)
	router.POST("/api/movieList", handler.PostMovies)

	// 多节点：在线节点列表（无认证，仅用于前端识别局域网内其他节点）
	router.GET("/api/lanPeers", handler.GetLanPeers)
	router.GET("/api/lanPeersWithStats", handler.GetLanPeersWithStats)
	router.GET("/api/peerStats", handler.GetPeerStats)
	router.POST("/api/discoverPeers", handler.DiscoverLanPeers)
	router.POST("/api/addPeer", handler.AddLanPeer)
	router.POST("/api/removePeer", handler.RemoveLanPeer)
	router.POST("/api/togglePeer", handler.TogglePeer)
	router.POST("/api/cleanLanPeers", handler.CleanLanPeers)

	router.GET("/api/transferTasks", handler.GetTransferTask)
	router.GET("/api/delTransferTasks/:create", handler.GetDelTransferTask)
	router.POST("/api/clearCompletedTasks", handler.PostClearCompletedTasks)
	router.POST("/api/clearFailedTasks", handler.PostClearFailedTasks)
	router.POST("/api/clearAllTasks", handler.PostClearAllTasks)
	router.POST("/api/authorList", handler.PostAuthor)
	router.GET("/api/authorImage/:path", handler.GetAuthorImage)

	router.GET("/api/play/:id", handler.GetPlay)
	router.GET("/api/tranferToMp4/:id/:xcode", handler.GetTransferToMp4)
	router.POST("/api/mergeFiles", handler.PostMerge)
	router.GET("/api/cutMovie/:id/:start/:end", handler.GetCutMovie)
	router.POST("/api/setMovieType/:id/:movieType", handler.SetMovieType)
	router.GET("/api/info/:id", handler.GetInfo)
	router.POST("/api/renameFile", handler.PostRename)
	router.POST("/api/addFileTag/:id/:tag", handler.GetAddTag)
	router.POST("/api/clearFileTag/:id/:tag", handler.GetClearTag)
	router.GET("/api/dir/:id/:sort", handler.GetDirInfo)
	router.GET("/api/delete/:id", handler.GetDelete)
	router.DELETE("/api/delete/:id", handler.GetDelete)

	router.GET("/api/openFolder/:id", handler.GetOpenFolder)
	router.POST("/api/OpenFolderByPath", handler.PostOpenFolderByPath)
	router.POST("/api/DeleteFolderByPath", handler.PostDeleteFolderByPath)
	router.POST("/api/moveFile", handler.PostMove)

	router.GET("/api/DeleteFileByPathUseEncode/:path", handler.GetDeleteFileByPathUseEncode)

	router.GET("/api/refreshTargetIndex/:dir", handler.GetRefreshTargetIndex)
	router.GET("/api/refreshIndex", handler.GetRefreshIndex)
	router.GET("/api/settingInfo", handler.GetSettingInfo)
	router.POST("/api/setting", handler.PostSetting)
	router.GET("/api/serverPort", handler.GetServerPort)
	router.GET("/api/GetIpAddr", handler.GetIpAddr2)
	router.GET("/api/shutDown", handler.GetShutdown)

	router.GET("/api/users", handler.GetUsers)
	router.POST("/api/user/add", handler.AddUser)
	router.POST("/api/user/delete", handler.DeleteUser)

	// 权限管理
	router.GET("/api/permissions", handler.GetAllPermissions)
	router.GET("/api/user/:username/permissions", handler.GetUserPermissions)
	router.POST("/api/user/permissions", handler.UpdateUserPermissions)
	// 角色管理
	router.GET("/api/roles", handler.GetRoles)
	router.POST("/api/roles", handler.CreateRole)
	router.POST("/api/roles/:name", handler.UpdateRole)
	router.DELETE("/api/roles/:name", handler.DeleteRole)
	router.POST("/api/user/role", handler.UpdateUserRole)

	router.GET("/api/typeSizeMap", handler.GetTypeSize)
	router.GET("/api/tagSizeMap", handler.GetTagSize)
	router.GET("/api/seriesCount", handler.GetSeriesSize)
	router.GET("/api/scanTime", handler.GetScanTime)
	router.GET("/api/diskUsage", handler.GetDiskUsage)
	router.GET("/api/heartBeat", handler.GetHeartBeat)
	router.GET("/api/pingHost", handler.PingHost)
	router.GET("/api/logMemory", handler.GetLogMemory)
	router.GET("/api/localLog", handler.GetLocalLog)
	router.GET("/api/indexHealth", handler.GetIndexHealthCheck)
	router.POST("/api/chat/deepseek", handler.PostChatDeepSeek)
	router.GET("/api/ws", handler.HandleWebSocket)
	router.GET("/api/events", func(c *gin.Context) {
		handler.HandleSSE(c.Writer, c.Request)
	})

	router.GET("/api/cutImage/:id/:typeImage/:downFlag/:start", handler.GetCutImage)

	router.POST("/api/torrent/add", handler.PostAddMagnet)
	router.POST("/api/torrent/startDownload", handler.PostStartDownload)
	router.GET("/api/torrent/files/:infoHash", handler.GetTorrentFiles)
	router.GET("/api/torrent/stream/:infoHash", handler.GetTorrentStream)
	router.GET("/api/torrent/status/:infoHash", handler.GetTorrentStatus)
	router.DELETE("/api/torrent/:infoHash", handler.DeleteTorrent)

	// 文件流路由：在 :10081 上使用 StreamTokenAuth 中间件（与 :10082 一致的 token 校验）
	// AuthMiddleware 已跳过 /api/stream/ 路径，不会要求 Bearer Token
	streamGroup := router.Group("/api/stream")
	streamGroup.Use(middleware.StreamTokenAuth())
	{
	 streamGroup.GET("/file/:id", handler.GetFile)
	 streamGroup.GET("/png/:path", handler.GetPng)
	 streamGroup.GET("/jpg/:path", handler.GetJpg)
	 streamGroup.GET("/GetFileByPathUseEncode/:path", handler.GetFileByPathUseEncode)
	}
	buildShutdownRoutes(router, sigChan)

	return router
}

// BuildFileRouter 构建文件/图片流路由（端口 10082）
// 使用 StreamTokenAuth 校验 URL 中的 token 参数（复用 API 侧 ValidateTokenWithInfo）
// NOTE: SignAuthMiddleware（HMAC 签名）对多节点集群不可用，故未注册
func BuildFileRouter() *gin.Engine {
	if env.IsProd {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	buildCommonMiddleware(router)
	router.Use(middleware.StreamTokenAuth())
	buildStreamMiddleware(router)

	return router
}
