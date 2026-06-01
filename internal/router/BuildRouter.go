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

func BuildRouter(tempDir string) *gin.Engine {
	config := cors.DefaultConfig()
	// 限制CORS允许的起源，防止CSRF攻击
	// 生产环境应该明确指定允许的域名
	if env.IsProd {
		// 生产环境：从环境变量读取允许的起源
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		if allowedOrigins != "" {
			config.AllowOrigins = strings.Split(allowedOrigins, ",")
		} else {
			config.AllowOrigins = []string{"*"}
		}
		config.AllowCredentials = true
	} else {
		// 开发环境：允许所有 HTTP 来源（IP 地址访问需要）
		config.AllowOrigins = []string{"*"}
		// 注意：AllowOrigins 为 "*" 时 AllowCredentials 自动设为 false
		// 如需携带 cookie/Authorization，前端需要配置 withCredentials
	}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Range", "Accept-Ranges", "Content-Range"}
	config.ExposeHeaders = []string{"Content-Length", "Content-Range", "Accept-Ranges", "Content-Type"}

	if env.IsProd {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	router.Use(cors.New(config))
	router.Use(ginlogrus.Logger(utils.NewLogger()))
	router.Use(middleware.CustomRecovery())
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

	router.GET("/api/transferTasks", handler.GetTransferTask)
	router.GET("/api/delTransferTasks/:create", handler.GetDelTransferTask)
	router.POST("/api/actressList", handler.PostActress)
	router.GET("/api/actressImgae/:path", handler.GetActressImage)

	router.GET("/api/play/:id", handler.GetPlay)
	router.GET("/api/tranferToMp4/:id/:xcode", handler.GetTransferToMp4)
	router.POST("/api/mergeFiles", handler.PostMerge)
	router.GET("/api/cutMovie/:id/:start/:end", handler.GetCutMovie)
	router.GET("/api/file/:id", handler.GetFile)
	router.GET("/api/setMovieType/:id/:movieType", handler.SetMovieType)
	router.GET("/api/info/:id", handler.GetInfo)
	router.POST("/api/file/rename", handler.PostRename)
	router.GET("/api/file/addTag/:id/:tag", handler.GetAddTag)
	router.GET("/api/file/clearTag/:id/:tag", handler.GetClearTag)
	router.GET("/api/dir/:id/:sort", handler.GetDirInfo)
	router.GET("/api/delete/:id", handler.GetDelete)

	router.GET("/api/openFolder/:id", handler.GetOpenFolder)
	router.POST("/api/OpenFolerByPath", handler.PostOpenFolderByPath)
	router.POST("/api/DeleteFolerByPath", handler.PostDeleteFolerByPath)
	router.POST("/api/file/move", handler.PostMove)

	router.GET("/api/png/:path", handler.GetPng)
	router.GET("/api/jpg/:path", handler.GetJpg)
	router.GET("/api/GetFileByPathUseEncode/:path", handler.GetFileByPathUseEncode)
	router.GET("/api/DeleteFileByPathUseEncode/:path", handler.GetDeleteFileByPathUseEncode)
	router.GET("/api/tempimage/:path", handler.GetTempImage)

	router.GET("/api/refreshTargetIndex/:dir", handler.GetRefreshTargetIndex)
	router.GET("/api/refreshIndex", handler.GetRefreshIndex)
	router.GET("/api/settingInfo", handler.GetSettingInfo)
	router.POST("/api/setting", handler.PostSetting)
	router.GET("/api/GetIpAddr", handler.GetIpAddr2)
	router.GET("/api/shutDown", handler.GetShutdown)

	// 用户管理路由（需要认证）
	router.GET("/api/users", handler.GetUsers)
	router.POST("/api/user/add", handler.AddUser)
	router.POST("/api/user/delete", handler.DeleteUser)
	router.POST("/api/user/changePassword", handler.ChangePassword)

	router.GET("/api/typeSizeMap", handler.GetTypeSize)
	router.GET("/api/tagSizeMap", handler.GetTagSize)
	router.GET("/api/seriesCount", handler.GetSeriesSize)
	router.GET("/api/scanTime", handler.GetScanTime)
	router.GET("/api/heartBeat", handler.GetHeartBeat)
	router.GET("/api/logMemery", handler.GetLogMemery)
	router.GET("/api/indexHealth", handler.GetIndexHealthCheck)

	router.GET("/api/cutImage/:id/:typeImage/:downFlag/:start", handler.GetCutImage)

	router.POST("/api/torrent/add", handler.PostAddMagnet)
	router.POST("/api/torrent/startDownload", handler.PostStartDownload)
	router.GET("/api/torrent/files/:infoHash", handler.GetTorrentFiles)
	router.GET("/api/torrent/stream/:infoHash", handler.GetTorrentStream)
	router.GET("/api/torrent/status/:infoHash", handler.GetTorrentStatus)
	router.DELETE("/api/torrent/:infoHash", handler.DeleteTorrent)

	return router
}
