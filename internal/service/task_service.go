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

// TransferTask 以 ID (string, RFC3339Nano 格式) 为 key
var TransferTask = map[string]model.TransferTaskModel{}
var TransferTaskMutex sync.RWMutex // 保护 TransferTask 的并发访问

const MaxTransferTaskCount = 1000

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

// ClearCompletedTasks 清除所有已完成的任务
func ClearCompletedTasks() utils.Result {
	TransferTaskMutex.Lock()
	defer TransferTaskMutex.Unlock()

	count := 0
	for key, task := range TransferTask {
		if task.Status == model.StatusCompleted {
			delete(TransferTask, key)
			DeleteTaskLog(key)
			count++
		}
	}
	CleanupExpiredTaskLogs(7)
	return utils.NewSuccessByMsg(fmt.Sprintf("已清除 %d 个已完成任务", count))
}

// ClearFailedTasks 清除所有失败的任务
func ClearFailedTasks() utils.Result {
	TransferTaskMutex.Lock()
	defer TransferTaskMutex.Unlock()

	count := 0
	for key, task := range TransferTask {
		if task.Status == model.StatusFailed {
			delete(TransferTask, key)
			DeleteTaskLog(key)
			count++
		}
	}
	CleanupExpiredTaskLogs(7)
	return utils.NewSuccessByMsg(fmt.Sprintf("已清除 %d 个失败任务", count))
}

// ClearAllTasks 清除所有任务（执行中的除外）
func ClearAllTasks() utils.Result {
	TransferTaskMutex.Lock()
	defer TransferTaskMutex.Unlock()

	count := 0
	for key, task := range TransferTask {
		if task.Status != model.StatusExecuting {
			delete(TransferTask, key)
			DeleteTaskLog(key)
			count++
		}
	}
	PendingTaskCount.Store(0)
	// 重新统计 pending 任务
	for _, task := range TransferTask {
		if task.Status == model.StatusPending {
			PendingTaskCount.Add(1)
		}
	}
	CleanupExpiredTaskLogs(7)
	return utils.NewSuccessByMsg(fmt.Sprintf("已清除 %d 个任务", count))
}

// DeleteTaskLog 删除单任务日志文件
func DeleteTaskLog(taskID string) {
	if err := os.Remove(taskLogPath(taskID)); err != nil && !os.IsNotExist(err) {
		utils.InfoFormat("删除任务日志文件失败: %s, 错误: %v", taskID, err)
	}
}

// CleanupExpiredTaskLogs 清理 task_logs 目录中创建时间超过 keepDays 天的日志文件
func CleanupExpiredTaskLogs(keepDays int) {
	dir := taskLogDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			utils.InfoFormat("读取日志目录失败: %v", err)
		}
		return
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err == nil {
				removed++
			}
		}
	}
	if removed > 0 {
		utils.InfoFormat("清理了 %d 个过期任务日志文件（超过 %d 天）", removed, keepDays)
	}
}

// DeleteIndexByPath 按路径删除文件：索引移除 + 物理删除 + 附属文件清理 + 空目录清理
func DeleteIndexByPath(validatedPath string) utils.Result {
	id := utils.DirpathForId(validatedPath)
	file := GetEngine().FindById(id)
	if !file.IsNull() {
		GetEngine().DeleteOnIndex(file)
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
	if len(TransferTask) >= MaxTransferTaskCount {
		TransferTaskMutex.Unlock()
		return utils.NewFailByMsg("任务队列已满（最多1000个），请清理已完成任务后再试")
	}
	TransferTask[task.ID] = task
	PendingTaskCount.Add(1)
	TransferTaskMutex.Unlock()

	wakeTaskScheduler()

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

	// 读锁检查重复任务
	TransferTaskMutex.RLock()
	for _, taskModel := range TransferTask {
		if taskModel.Path == movieFile.Path &&
			(taskModel.Status == model.StatusPending || taskModel.Status == model.StatusExecuting) {
			TransferTaskMutex.RUnlock()
			return utils.NewFailByMsg("该文件已有转码任务在执行，请等待完成")
		}
	}
	TransferTaskMutex.RUnlock()

	// 写锁创建任务
	TransferTaskMutex.Lock()
	if len(TransferTask) >= MaxTransferTaskCount {
		TransferTaskMutex.Unlock()
		return utils.NewFailByMsg("任务队列已满（最多1000个），请清理已完成任务后再试")
	}
	task := model.NewTask(movieFile.Path, movieFile.Name, from, to)
	task.SetStatus(model.StatusPending)
	if xcode != "" {
		task.VCode = xcode
	}
	TransferTask[task.ID] = task
	PendingTaskCount.Add(1)
	TransferTaskMutex.Unlock()
	wakeTaskScheduler()
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
	if len(TransferTask) >= MaxTransferTaskCount {
		TransferTaskMutex.Unlock()
		return utils.NewFailByMsg("任务队列已满（最多1000个），请清理已完成任务后再试")
	}
	TransferTask[task.ID] = task
	PendingTaskCount.Add(1)
	TransferTaskMutex.Unlock()

	wakeTaskScheduler()

	pending, executing := pendingExecutingCount()
	LogMem.Add("CreateCutTask: 创建成功 path=%s, start=%s, end=%s, CreateTime=%v, pending=%d, executing=%d", task.Path, start, end, task.CreateTime, pending, executing)
	return utils.NewSuccessByMsg("任务创建成功")
}
