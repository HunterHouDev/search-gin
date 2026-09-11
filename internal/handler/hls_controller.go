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
