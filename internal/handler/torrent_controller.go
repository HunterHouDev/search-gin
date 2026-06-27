package handler

import (
	"net/http"

	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

type MagnetRequest struct {
	MagnetURI string `json:"magnetURI" binding:"required"`
}

func PostAddMagnet(c *gin.Context) {
	if !requirePermission(c, "op:torrent") {
		return
	}
	var req MagnetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("请提供有效的磁力链"))
		return
	}

	if service.TorrentApp == nil {
		c.JSON(http.StatusServiceUnavailable, utils.NewFailByMsg("Torrent 服务未启动"))
		return
	}

	result, err := service.TorrentApp.AddMagnet(req.MagnetURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewFailByMsg(err.Error()))
		return
	}

	res := utils.NewSuccess()
	res.Data = result
	c.JSON(http.StatusOK, res)
}

type StartDownloadRequest struct {
	InfoHash string `json:"infoHash" binding:"required"`
	FilePath string `json:"filePath"`
}

func PostStartDownload(c *gin.Context) {
	if !requirePermission(c, "op:torrent") {
		return
	}
	var req StartDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("请提供有效的参数"))
		return
	}

	if service.TorrentApp == nil {
		c.JSON(http.StatusServiceUnavailable, utils.NewFailByMsg("Torrent 服务未启动"))
		return
	}

	result, err := service.TorrentApp.StartDownload(req.InfoHash, req.FilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewFailByMsg(err.Error()))
		return
	}

	res := utils.NewSuccess()
	res.Data = result
	c.JSON(http.StatusOK, res)
}

func GetTorrentStream(c *gin.Context) {
	infoHash := c.Param("infoHash")
	if infoHash == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("缺少 infoHash"))
		return
	}

	if service.TorrentApp == nil {
		c.JSON(http.StatusServiceUnavailable, utils.NewFailByMsg("Torrent 服务未启动"))
		return
	}

	err := service.TorrentApp.StreamVideo(infoHash, c.Writer, c.Request)
	if err != nil {
		utils.ErrorNormal("流式播放失败:", err)
		c.JSON(http.StatusInternalServerError, utils.NewFailByMsg(err.Error()))
		return
	}
}

func GetTorrentStatus(c *gin.Context) {
	infoHash := c.Param("infoHash")
	if infoHash == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("缺少 infoHash"))
		return
	}

	if service.TorrentApp == nil {
		c.JSON(http.StatusServiceUnavailable, utils.NewFailByMsg("Torrent 服务未启动"))
		return
	}

	status, err := service.TorrentApp.GetStatus(infoHash)
	if err != nil {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg(err.Error()))
		return
	}

	res := utils.NewSuccess()
	res.Data = status
	c.JSON(http.StatusOK, res)
}

func DeleteTorrent(c *gin.Context) {
	if !requirePermission(c, "op:torrent") {
		return
	}
	infoHash := c.Param("infoHash")
	if infoHash == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("缺少 infoHash"))
		return
	}

	if service.TorrentApp == nil {
		c.JSON(http.StatusServiceUnavailable, utils.NewFailByMsg("Torrent 服务未启动"))
		return
	}

	err := service.TorrentApp.RemoveTorrent(infoHash)
	if err != nil {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg(err.Error()))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccess())
}

func GetTorrentFiles(c *gin.Context) {
	infoHash := c.Param("infoHash")
	if infoHash == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("缺少 infoHash"))
		return
	}

	if service.TorrentApp == nil {
		c.JSON(http.StatusServiceUnavailable, utils.NewFailByMsg("Torrent 服务未启动"))
		return
	}

	files, err := service.TorrentApp.GetFiles(infoHash)
	if err != nil {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg(err.Error()))
		return
	}

	res := utils.NewSuccess()
	res.Data = files
	c.JSON(http.StatusOK, res)
}
