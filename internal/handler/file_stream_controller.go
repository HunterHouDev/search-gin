package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"time"

	"github.com/gin-gonic/gin"
)

func GetRefreshTargetIndex(c *gin.Context) {
	dir := c.Param("dir")
	baseDir, _ := url.QueryUnescape(dir)

	validatedDir, err := utils.ValidatePath(baseDir, consts.GetOSSetting().Dirs)
	if err != nil {
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("路径不在允许范围内"))
		return
	}

	fileHandler.fileSvc.ScanTarget(validatedDir)
	c.JSON(http.StatusOK, utils.NewSuccessByMsg("扫描任务执行中"))
}

func GetRefreshIndex(c *gin.Context) {
	cnt := fileHandler.fileSvc.ScanAll()
	c.JSON(http.StatusOK, utils.NewSuccessByMsg("计划扫描："+fmt.Sprint(cnt)))
}

func GetFileByPathUseEncode(c *gin.Context) {
	decodedPath, err := url.QueryUnescape(c.Param("path"))
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的文件路径"))
		return
	}

	validatedPath, err := utils.ValidatePath(decodedPath, consts.GetOSSetting().Dirs)
	if err != nil {
		utils.ErrorFormat("路径遍历攻击尝试: %s, 错误: %v", decodedPath, err)
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("访问被拒绝：路径不在允许范围内"))
		return
	}

	if utils.ExistsFiles(validatedPath) {
		c.File(validatedPath)
	} else {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
	}
}

func GetDeleteFileByPathUseEncode(c *gin.Context) {
	decodedPath, err := url.QueryUnescape(c.Param("path"))
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的文件路径"))
		return
	}

	validatedPath, err := utils.ValidatePath(decodedPath, consts.GetOSSetting().Dirs)
	if err != nil {
		utils.ErrorFormat("路径遍历攻击尝试: %s, 错误: %v", decodedPath, err)
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("删除被拒绝：路径不在允许范围内"))
		return
	}

	if !utils.ExistsFiles(validatedPath) {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
		return
	}

	c.JSON(http.StatusOK, service.DeleteFileByPath(validatedPath))
}

func GetFile(c *gin.Context) {
	service.SearchApp.GetFile(c)
}

func GetPng(c *gin.Context) {
	service.SearchApp.GetPng(c)
}

func GetJpg(c *gin.Context) {
	service.SearchApp.GetJpg(c)
}

func GetTransferTask(c *gin.Context) {
	result := utils.NewSuccess()
	consts.TransferTaskMutex.RLock()
	tasks := make(map[time.Time]model.TransferTaskModel, len(consts.TransferTask))
	for k, v := range consts.TransferTask {
		tasks[k] = v
	}
	consts.TransferTaskMutex.RUnlock()
	result.Data = tasks
	c.JSON(http.StatusOK, result)
}

func GetDelTransferTask(c *gin.Context) {
	create := c.Param("create")
	consts.TransferTaskMutex.Lock()
	var ti time.Time
	var task model.TransferTaskModel
	for k, v := range consts.TransferTask {
		if v.Name == create {
			ti = k
			task = v
			break
		}
	}
	if task.Status == "执行中" {
		consts.TransferTaskMutex.Unlock()
		r := utils.Fail()
		r.Message = "执行中无法删除"
		c.JSON(http.StatusOK, r)
		return
	}
	delete(consts.TransferTask, ti)
	consts.TransferTaskMutex.Unlock()
	c.JSON(http.StatusOK, utils.NewSuccess())
}
