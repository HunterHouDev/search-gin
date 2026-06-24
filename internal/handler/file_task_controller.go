package handler

import (
	"net/http"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/utils"
	"time"

	"github.com/gin-gonic/gin"
)

func PostMerge(c *gin.Context) {
	searchParam := model.MergeParam{}
	if err := c.Bind(&searchParam); err != nil {
		utils.InfoFormat("PostMerge 参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}
	utils.InfoFormat("PostMerge： [%v]", searchParam)
	c.JSON(http.StatusOK, service.CreateMergeTask(searchParam.Files, searchParam.Dest, searchParam.DeleteSource))
}

func GetTransferTask(c *gin.Context) {
	service.TransferTaskMutex.RLock()
	n := len(service.TransferTask)
	tasks := make([]model.TransferTaskModel, 0, n)
	counts := [5]int{} // [total, completed, failed, executing, pending]
	for _, v := range service.TransferTask {
		tasks = append(tasks, v)
		switch v.Status {
		case model.StatusCompleted:
			counts[1]++
		case model.StatusFailed:
			counts[2]++
		case model.StatusExecuting:
			counts[3]++
		case model.StatusPending:
			counts[4]++
		}
	}
	service.TransferTaskMutex.RUnlock()

	result := utils.NewSuccess()
	result.Data = map[string]interface{}{
		"tasks":  tasks,
		"counts": counts[:],
	}
	c.JSON(http.StatusOK, result)
}

func GetDelTransferTask(c *gin.Context) {
	createStr := c.Param("create")
	ti, err := time.Parse(time.RFC3339Nano, createStr)
	if err != nil {
		c.JSON(http.StatusOK, utils.NewFailByMsg("参数解析失败"))
		return
	}

	service.TransferTaskMutex.Lock()
	task, found := service.TransferTask[ti]
	if !found {
		service.TransferTaskMutex.Unlock()
		c.JSON(http.StatusOK, utils.NewFailByMsg("任务不存在"))
		return
	}
	if task.Status == model.StatusExecuting {
		service.TransferTaskMutex.Unlock()
		r := utils.Fail()
		r.Message = "执行中无法删除"
		c.JSON(http.StatusOK, r)
		return
	}
	delete(service.TransferTask, ti)
	service.TransferTaskMutex.Unlock()
	c.JSON(http.StatusOK, utils.NewSuccess())
}

func GetTransferToMp4(c *gin.Context) {
	id := c.Param("id")
	if service.HandleRemoteByID(c, id, "transferToMp4") {
		return
	}

	xcode := c.Param("xcode")
	utils.InfoFormat("GetTransferToMp4 newFile [%v][%v]", id, xcode)
	c.JSON(http.StatusOK, service.CreateTransferTask(id, xcode))
}

func GetCutImage(c *gin.Context) {
	idInt := c.Param("id")
	typeImage := c.Param("typeImage")
	start := c.Param("start")

	if service.HandleRemoteByID(c, idInt, "cutImage") {
		return
	}

	movieFile := UseApp().search.FindById(idInt)
	if movieFile.IsNull() {
		r := utils.Fail()
		r.Message = "文件不存在"
		c.JSON(http.StatusOK, r)
		return
	}
	c.JSON(http.StatusOK, service.CutImage(movieFile.Path, typeImage, start))
}

func GetCutMovie(c *gin.Context) {
	id := c.Param("id")
	if service.HandleRemoteByID(c, id, "cutMovie") {
		return
	}

	start := c.Param("start")
	end := c.Param("end")
	utils.InfoFormat("GetCutMovie [%v][%v][%v]", id, start, end)
	c.JSON(http.StatusOK, service.CreateCutTask(id, start, end))
}

// PostClearCompletedTasks 清除所有已完成任务
func PostClearCompletedTasks(c *gin.Context) {
	c.JSON(http.StatusOK, service.ClearCompletedTasks())
}

// PostClearFailedTasks 清除所有失败任务
func PostClearFailedTasks(c *gin.Context) {
	c.JSON(http.StatusOK, service.ClearFailedTasks())
}

// PostClearAllTasks 清除所有任务（执行中的除外）
func PostClearAllTasks(c *gin.Context) {
	c.JSON(http.StatusOK, service.ClearAllTasks())
}
