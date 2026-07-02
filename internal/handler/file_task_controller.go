package handler

import (
	"bufio"
	"net/http"
	"os"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

func PostMerge(c *gin.Context) {
	if !requirePermission(c, "op:merge") {
		return
	}
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

// GetTaskLog 查询单任务日志（从日志文件读取后 1000 行）
func GetTaskLog(c *gin.Context) {
	taskID := c.Param("taskID")
	service.TransferTaskMutex.RLock()
	task, found := service.TransferTask[taskID]
	service.TransferTaskMutex.RUnlock()
	if !found {
		c.JSON(http.StatusOK, utils.NewFailByMsg("任务不存在"))
		return
	}

	// 从文件读取后 1000 行
	logPath := service.TaskLogPath(taskID)
	logContent := ""
	if f, err := os.Open(logPath); err == nil {
		defer f.Close()
		var lines []string
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
			if len(lines) > 1000 {
				lines = lines[len(lines)-1000:]
			}
		}
		for _, l := range lines {
			logContent += l + "\n"
		}
	}

	result := utils.NewSuccess()
	result.Data = map[string]interface{}{
		"createTime": task.CreateTime,
		"status":     task.Status,
		"command":    task.Command,
		"log":        logContent,
	}
	c.JSON(http.StatusOK, result)
}

func GetDelTransferTask(c *gin.Context) {
	taskID := c.Param("taskID")
	service.TransferTaskMutex.Lock()
	task, found := service.TransferTask[taskID]
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
	delete(service.TransferTask, taskID)
	service.TransferTaskMutex.Unlock()
	service.DeleteTaskLog(taskID)
	service.CleanupExpiredTaskLogs(7)
	c.JSON(http.StatusOK, utils.NewSuccess())
}

func GetTransferToMp4(c *gin.Context) {
	if !requirePermission(c, "op:transcode") {
		return
	}
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
	if !requirePermission(c, "op:cut") {
		return
	}
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
