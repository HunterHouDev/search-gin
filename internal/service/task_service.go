package service

import (
	"fmt"
	"os"
	"path/filepath"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"sync"
	"time"
)

var TransferTask = map[time.Time]model.TransferTaskModel{}
var TransferTaskMutex sync.RWMutex // 保护 TransferTask 的并发访问

// pendingExecutingCount 统计当前等待中和执行中的任务数量
func pendingExecutingCount() (pending, executing int) {
	TransferTaskMutex.RLock()
	defer TransferTaskMutex.RUnlock()
	for _, t := range TransferTask {
		switch t.Status {
		case model.StatusPending:
			pending++
		case model.StatusExecuting:
			executing++
		}
	}
	return
}

// DeleteFileByPath 按路径删除文件：索引移除 + 物理删除 + 附属文件清理 + 空目录清理
func DeleteFileByPath(validatedPath string) utils.Result {
	id := utils.DirpathForId(validatedPath)
	file := GetEngine().FindById(id)
	if !file.IsNull() {
		GetEngine().DeleteFile(file)
		utils.InfoFormat("已从索引中删除: %s", file.Path)
	}

	dir := filepath.Dir(validatedPath)
	if err := os.Remove(validatedPath); err != nil {
		utils.InfoFormat("删除文件失败: %s, 错误: %v", validatedPath, err)
		return utils.NewFailByMsg("删除失败")
	}

	for _, companion := range []string{file.Jpg, file.Png, file.Gif} {
		if companion != "" && utils.ExistsFiles(companion) {
			if e := os.Remove(companion); e != nil {
				utils.InfoFormat("删除附属文件失败: %s, 错误: %v", companion, e)
			}
		}
	}

	if entries, e := os.ReadDir(dir); e == nil && len(entries) == 0 {
		os.Remove(dir)
	}

	return utils.NewSuccessByMsg("删除成功")
}

// CreateMergeTask 创建合并任务
func CreateMergeTask(fileIds []string, dest string, deleteSource bool) utils.Result {
	var paths []string
	var dir string
	for _, id := range fileIds {
		curFile := GetEngine().FindById(id)
		if curFile.IsNull() {
			return utils.NewFailByMsg("文件不存在: " + id)
		}
		dir = curFile.DirPath
		paths = append(paths, curFile.Path)
	}

	if len(paths) == 0 {
		return utils.NewFailByMsg("没有找到要合并的文件")
	}

	listPath := dir + string(filepath.Separator) + "list.txt"
	f, err := os.Create(listPath)
	if err != nil {
		utils.InfoFormat("创建文件 list.txt 时出错: %v", err)
		return utils.NewFailByMsg("创建合并列表文件失败")
	}
	defer f.Close()

	for _, filePath := range paths {
		if _, err := f.WriteString("file '" + filePath + "'\n"); err != nil {
			utils.InfoFormat("写入文件 list.txt 时出错: %v", err)
			return utils.NewFailByMsg("写入合并列表失败")
		}
	}

	if dest == "" {
		suffix := utils.GetSuffix(paths[0])
		dest = dir + string(filepath.Separator) + fmt.Sprintf("%d.%s", time.Now().UnixMilli(), suffix)
	}

	task := model.NewMergeTask(paths, dest, listPath, deleteSource)
	task.SetStatus(model.StatusPending)
	TransferTaskMutex.Lock()
	TransferTask[task.CreateTime] = task
	PendingTaskCount.Add(1)
	TransferTaskMutex.Unlock()

	notifyTaskChange()

	pending, executing := pendingExecutingCount()
	LogMem.Add("CreateMergeTask: 创建成功 path=%s, CreateTime=%v, pending=%d, executing=%d", task.Path, task.CreateTime, pending, executing)
	return utils.NewSuccessByMsg("任务创建成功")
}

// CreateTransferTask 创建转码任务（含重复检查）
func CreateTransferTask(id string, xcode string) utils.Result {
	movieFile := GetEngine().FindById(id)
	if !utils.ExistsFiles(movieFile.Path) {
		return utils.NewFailByMsg("文件不存在")
	}

	from := utils.GetSuffix(movieFile.Path)
	to := "mp4"

	TransferTaskMutex.RLock()
	for _, taskModel := range TransferTask {
		if taskModel.Path == movieFile.Path &&
			(taskModel.Status == model.StatusPending || taskModel.Status == model.StatusExecuting) {
			TransferTaskMutex.RUnlock()
			return utils.NewFailByMsg("该文件已有转码任务在执行，请等待完成")
		}
	}
	TransferTaskMutex.RUnlock()

	task := model.NewTask(movieFile.Path, movieFile.Name, from, to)
	task.SetStatus(model.StatusPending)
	if xcode != "" {
		task.VCode = xcode
	}
	TransferTaskMutex.Lock()
	TransferTask[task.CreateTime] = task
	PendingTaskCount.Add(1)
	TransferTaskMutex.Unlock()
	notifyTaskChange()
	pending, executing := pendingExecutingCount()
	LogMem.Add("CreateTransferTask: 创建成功 id=%s, xcode=%s, path=%s, CreateTime=%v, pending=%d, executing=%d", id, xcode, task.Path, task.CreateTime, pending, executing)
	return utils.NewSuccessByMsg("任务创建成功")
}

// CreateCutTask 创建剪切任务
func CreateCutTask(id string, start string, end string) utils.Result {
	movieFile := GetEngine().FindById(id)
	if !utils.ExistsFiles(movieFile.Path) {
		return utils.NewFailByMsg("文件不存在")
	}

	from := utils.GetSuffix(movieFile.Path)
	task := model.NewCutTask(movieFile.Path, movieFile.Name, start, end, from)
	task.SetStatus(model.StatusPending)
	TransferTaskMutex.Lock()
	TransferTask[task.CreateTime] = task
	PendingTaskCount.Add(1)
	TransferTaskMutex.Unlock()

	notifyTaskChange()

	pending, executing := pendingExecutingCount()
	LogMem.Add("CreateCutTask: 创建成功 path=%s, start=%s, end=%s, CreateTime=%v, pending=%d, executing=%d", task.Path, start, end, task.CreateTime, pending, executing)
	return utils.NewSuccessByMsg("任务创建成功")
}
