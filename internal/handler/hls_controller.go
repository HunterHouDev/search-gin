package handler

import (
	"net/http"

	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

// PostHlsDownload 创建服务端分片下载任务。
// 实际下载在服务端执行，前端关闭弹窗 / 刷新页面都不会中断任务。
func PostHlsDownload(c *gin.Context) {
	if !requirePermission(c, "op:download") {
		return
	}
	req, err := BindJSON[service.HlsDownloadParam](c, "参数绑定失败")
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, service.CreateHlsDownloadTask(req))
}

// PostHlsCancel 取消进行中的分片下载任务
func PostHlsCancel(c *gin.Context) {
	if !requirePermission(c, "op:download") {
		return
	}
	taskID := c.Param("taskID")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("缺少 taskID"))
		return
	}
	c.JSON(http.StatusOK, service.CancelHlsDownload(taskID))
}

// PostHlsRestart 重启失败/已取消的分片下载任务（复用落盘的播放列表与目标路径）
func PostHlsRestart(c *gin.Context) {
	if !requirePermission(c, "op:download") {
		return
	}
	taskID := c.Param("taskID")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("缺少 taskID"))
		return
	}
	c.JSON(http.StatusOK, service.RestartHlsDownload(taskID))
}

// GetHlsPlaylist 返回任务创建时落盘的播放列表文本。
// 前端「浏览器直下」备用通道使用：浏览器拿到播放列表后直接从源站拉分片、
// 本地解密合并另存，全程不经过服务端下载。
func GetHlsPlaylist(c *gin.Context) {
	if !requirePermission(c, "op:download") {
		return
	}
	taskID := c.Param("taskID")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("缺少 taskID"))
		return
	}
	c.JSON(http.StatusOK, service.ReadHlsPlaylist(taskID))
}
