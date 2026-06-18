package service

import (
	"fmt"
	"os"
	"path/filepath"
	"search-gin/internal/model"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"time"
)

// DeleteFileByPath 按路径删除文件：索引移除 + 物理删除 + 附属文件清理 + 空目录清理
func DeleteFileByPath(validatedPath string) utils.Result {
	id := utils.DirpathForId(validatedPath)
	file := SearchEngine.FindById(id)
	if !file.IsNull() {
		SearchEngine.DeleteFile(file)
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
		curFile := SearchEngine.FindById(id)
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
	consts.TransferTaskMutex.Lock()
	consts.TransferTask[task.CreateTime] = task
	consts.TransferTaskMutex.Unlock()

	return utils.NewSuccessByMsg("任务创建成功")
}

// CreateTransferTask 创建转码任务（含重复检查）
func CreateTransferTask(id string, xcode string) utils.Result {
	movieFile := SearchEngine.FindById(id)
	if !utils.ExistsFiles(movieFile.Path) {
		return utils.NewFailByMsg("文件不存在")
	}

	from := utils.GetSuffix(movieFile.Path)
	to := "mp4"

	consts.TransferTaskMutex.RLock()
	for _, taskModel := range consts.TransferTask {
		if taskModel.Path == movieFile.Path && taskModel.Status != "执行失败" {
			consts.TransferTaskMutex.RUnlock()
			return utils.NewFailByMsg("任务不可重复")
		}
	}
	consts.TransferTaskMutex.RUnlock()

	task := model.NewTask(movieFile.Path, movieFile.Name, from, to)
	task.SetStatus(model.StatusPending)
	if xcode != "" {
		task.VCode = xcode
	}
	consts.TransferTaskMutex.Lock()
	consts.TransferTask[task.CreateTime] = task
	consts.TransferTaskMutex.Unlock()

	return utils.NewSuccessByMsg("任务创建成功")
}

// CreateCutTask 创建剪切任务
func CreateCutTask(id string, start string, end string) utils.Result {
	movieFile := SearchEngine.FindById(id)
	if !utils.ExistsFiles(movieFile.Path) {
		return utils.NewFailByMsg("文件不存在")
	}

	from := utils.GetSuffix(movieFile.Path)
	task := model.NewCutTask(movieFile.Path, movieFile.Name, start, end, from)
	task.SetStatus(model.StatusPending)
	consts.TransferTaskMutex.Lock()
	consts.TransferTask[task.CreateTime] = task
	consts.TransferTaskMutex.Unlock()

	return utils.NewSuccessByMsg("任务创建成功")
}
