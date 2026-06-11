package router

import (
	"os"
	"search-gin/internal/env"
	"search-gin/internal/handler"
	"search-gin/middleware"
	"search-gin/pkg/utils"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	ginlogrus "github.com/toorop/gin-logrus"
)

// buildCORSConfig 构建 CORS 配置
func buildCORSConfig() cors.Config {
	config := cors.DefaultConfig()
	if env.IsProd {
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		if allowedOrigins != "" {
			config.AllowOrigins = strings.Split(allowedOrigins, ",")
		} else {
			config.AllowOrigins = []string{"*"}
		}
		config.AllowCredentials = true
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
	router.Use(ginlogrus.Logger(utils.NewLogger()))
	router.Use(middleware.CustomRecovery())
}

func buildStreamMiddleware(router *gin.Engine) {
	router.GET("/api/stream/file/:id", handler.GetFile)
	router.GET("/api/stream/png/:path", handler.GetPng)
	router.GET("/api/stream/jpg/:path", handler.GetJpg)
	router.GET("/api/stream/GetFileByPathUseEncode/:path", handler.GetFileByPathUseEncode)
	router.GET("/api/stream/tempimage/:path", handler.GetTempImage)
}

// BuildAPIRouter 构建 API 业务路由（端口 10081）：需要认证
func BuildAPIRouter() *gin.Engine {
	if env.IsProd {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	buildCommonMiddleware(router)
	router.Use(middleware.AuthMiddleware())

	if !env.IsProd {
		router.Use(func(c *gin.Context) {
			start := time.Now()
			path := c.Request.URL.Path
			c.Next()
			duration := time.Since(start)
			if duration > 5*time.Second {
				utils.InfoFormat("慢请求 [%s] %s %d %v",
					c.Request.Method, path, c.Writer.Status(), duration)
			}
		})
	}

	router.NoRoute(handler.Index)
	router.GET("/", handler.Index)
	router.POST("/api/login", handler.Login)
	router.POST("/api/movieList", handler.PostMovies)

	// 多节点：在线节点列表（无认证，仅用于前端识别局域网内其他节点）
	router.GET("/api/lanPeers", handler.GetLanPeers)

	router.GET("/api/transferTasks", handler.GetTransferTask)
	router.GET("/api/delTransferTasks/:create", handler.GetDelTransferTask)
	router.POST("/api/actressList", handler.PostActress)
	router.GET("/api/actressImgae/:path", handler.GetActressImage)

	router.GET("/api/play/:id", handler.GetPlay)
	router.GET("/api/tranferToMp4/:id/:xcode", handler.GetTransferToMp4)
	router.POST("/api/mergeFiles", handler.PostMerge)
	router.GET("/api/cutMovie/:id/:start/:end", handler.GetCutMovie)
	router.GET("/api/setMovieType/:id/:movieType", handler.SetMovieType)
	router.GET("/api/info/:id", handler.GetInfo)
	router.POST("/api/renameFile", handler.PostRename)
	router.GET("/api/addFileTag/:id/:tag", handler.GetAddTag)
	router.GET("/api/clearFileTag/:id/:tag", handler.GetClearTag)
	router.GET("/api/dir/:id/:sort", handler.GetDirInfo)
	router.GET("/api/delete/:id", handler.GetDelete)

	router.GET("/api/openFolder/:id", handler.GetOpenFolder)
	router.POST("/api/OpenFolerByPath", handler.PostOpenFolderByPath)
	router.POST("/api/DeleteFolerByPath", handler.PostDeleteFolerByPath)
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

	router.GET("/api/typeSizeMap", handler.GetTypeSize)
	router.GET("/api/tagSizeMap", handler.GetTagSize)
	router.GET("/api/seriesCount", handler.GetSeriesSize)
	router.GET("/api/scanTime", handler.GetScanTime)
	router.GET("/api/heartBeat", handler.GetHeartBeat)
	router.GET("/api/logMemery", handler.GetLogMemery)
	router.GET("/api/indexHealth", handler.GetIndexHealthCheck)
	router.GET("/api/ws", handler.HandleWebSocket)

	router.GET("/api/cutImage/:id/:typeImage/:downFlag/:start", handler.GetCutImage)

	router.POST("/api/torrent/add", handler.PostAddMagnet)
	router.POST("/api/torrent/startDownload", handler.PostStartDownload)
	router.GET("/api/torrent/files/:infoHash", handler.GetTorrentFiles)
	router.GET("/api/torrent/stream/:infoHash", handler.GetTorrentStream)
	router.GET("/api/torrent/status/:infoHash", handler.GetTorrentStatus)
	router.DELETE("/api/torrent/:infoHash", handler.DeleteTorrent)

	buildStreamMiddleware(router)

	return router
}

// BuildFileRouter 构建文件/图片流路由（端口 10082）：无需认证
func BuildFileRouter() *gin.Engine {
	if env.IsProd {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	buildCommonMiddleware(router)
	// 文件流服务不需要认证中间件
	buildStreamMiddleware(router)

	return router
}
