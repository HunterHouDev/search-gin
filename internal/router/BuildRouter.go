package router

import (
	"search-gin/internal/handler"
	"search-gin/internal/env"
	"search-gin/pkg/utils"
	"search-gin/middleware"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	ginlogrus "github.com/toorop/gin-logrus"
)

func BuildRouter(tempDir string) *gin.Engine {
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowCredentials = true

	if env.IsProd {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.Default()
	router.Use(cors.New(config))
	router.Use(ginlogrus.Logger(utils.NewLogger()))
	router.Use(middleware.CustomRecovery())

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

	indexHtml := filepath.Join(tempDir, "dist", "index.html")
	if utils.ExistsFiles(indexHtml) {
		utils.InfoFormat("static exists:%s \n", indexHtml)
		router.LoadHTMLFiles(indexHtml)
		staticFs := map[string]string{
			"/css":    filepath.Join(tempDir, "dist", "css"),
			"/js":     filepath.Join(tempDir, "dist", "js"),
			"/assets": filepath.Join(tempDir, "dist", "assets"),
		}
		for k, v := range staticFs {
			router.StaticFS(k, http.Dir(v))
			utils.InfoFormat("static exists:%s \n", k)
		}
	} else {
		utils.InfoFormat("static not exists:%s \n", indexHtml)
	}

	router.NoRoute(handler.Index)
	router.GET("/", handler.Index)
	router.POST("/api/movieList", handler.PostSearch)

	router.GET("/api/transferTasks", handler.GetTransferTask)
	router.GET("/api/delTransferTasks/:create", handler.GetDelTransferTask)
	router.POST("/api/actressList", handler.PostActress)
	router.GET("/api/actressImgae/:path", handler.GetActressImage)

	router.GET("/api/play/:id", handler.GetPlay)
	router.GET("/api/mergeSrt/:id", handler.GetMergeSrt)
	router.GET("/api/tranferToMp4/:id/:xcode", handler.GetTransferToMp4)
	router.POST("/api/mergeFiles", handler.PostMerge)
	router.GET("/api/cutMovie/:id/:start/:end", handler.GetCutMovie)
	router.GET("/api/file/:id", handler.GetFile)
	router.GET("/api/setMovieType/:id/:movieType", handler.SetMovieType)
	router.GET("/api/info/:id", handler.GetInfo)
	router.POST("/api/file/rename", handler.PostRename)
	router.GET("/api/file/addTag/:id/:tag", handler.GetAddTag)
	router.GET("/api/file/clearTag/:id/:tag", handler.GetClearTag)
	router.GET("/api/imageList/:id", handler.GetImageList)
	router.GET("/api/dir/:id/:sort", handler.GetDirInfo)
	router.GET("/api/delete/:id", handler.GetDelete)

	router.POST("/api/sync", handler.PostSync)

	router.GET("/api/openFolder/:id", handler.GetOpenFolder)
	router.POST("/api/OpenFolerByPath", handler.PostOpenFolderByPath)
	router.POST("/api/DeleteFolerByPath", handler.PostDeleteFolerByPath)
	router.POST("/api/file/move", handler.PostMove)

	router.GET("/api/png/:path", handler.GetPng)
	router.GET("/api/jpg/:path", handler.GetJpg)
	router.GET("/api/GetFileByPathUseEncode/:path", handler.GetFileByPathUseEncode)
	router.GET("/api/DeleteFileByPathUseEncode/:path", handler.GetDeleteFileByPathUseEncode)
	router.GET("/api/tempimage/:path", handler.GetTempImage)

	router.GET("/api/buttoms", handler.GetSettingInfo)
	router.GET("/api/refreshTargetIndex/:dir", handler.GetRefreshTargetIndex)
	router.GET("/api/refreshIndex", handler.GetRefreshIndex)
	router.GET("/api/settingInfo", handler.GetSettingInfo)
	router.POST("/api/setting", handler.PostSetting)
	router.GET("/api/GetIpAddr", handler.GetIpAddr2)
	router.GET("/api/shutDown", handler.GetShutdown)

	router.GET("/api/typeSizeMap", handler.GetTypeSize)
	router.GET("/api/tagSizeMap", handler.GetTagSize)
	router.GET("/api/seriesCount", handler.GetSeriesSize)
	router.GET("/api/scanTime", handler.GetScanTime)
	router.GET("/api/heartBeat", handler.GetHeartBeat)
	router.GET("/api/logMemery", handler.GetLogMemery)

	router.GET("/api/cutImage/:id/:typeImage/:downFlag/:start", handler.GetCutImage)

	router.POST("/api/torrent/add", handler.PostAddMagnet)
	router.GET("/api/torrent/stream/:infoHash", handler.GetTorrentStream)
	router.GET("/api/torrent/status/:infoHash", handler.GetTorrentStatus)
	router.DELETE("/api/torrent/:infoHash", handler.DeleteTorrent)

	return router
}
