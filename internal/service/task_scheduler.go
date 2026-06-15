package service

import (
	"context"
	"search-gin/internal/model"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"strings"
	"sync/atomic"
	"time"
)

var (
	TaskCtx, TaskCancel = context.WithCancel(context.Background())
)

var FullScanInProgress atomic.Int32

// TaskExecuting 任务执行调度器
func (fs *fileService) TaskExecuting() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		fs.pollTasks()
		select {
		case <-TaskCtx.Done():
			utils.InfoFormat("任务调度器已停止")
			return
		case <-ticker.C:
		}
	}
}

// pollTasks 轮询并执行待处理任务
func (fs *fileService) pollTasks() {
	taskGroups := struct {
		todos           []model.TransferTaskModel
		todosCuts       []model.TransferTaskModel
		todosMerges     []model.TransferTaskModel
		executing       []model.TransferTaskModel
		executingCuts   []model.TransferTaskModel
		executingMerges []model.TransferTaskModel
	}{}

	consts.TransferTaskMutex.RLock()
	for _, t := range consts.TransferTask {
		switch {
		case strings.EqualFold(t.Status, model.StatusPending):
			switch {
			case strings.EqualFold(t.Type, model.TaskTypeCut):
				taskGroups.todosCuts = append(taskGroups.todosCuts, t)
			case strings.EqualFold(t.Type, model.TaskTypeMerge):
				taskGroups.todosMerges = append(taskGroups.todosMerges, t)
			case strings.EqualFold(t.Type, model.TaskTypeTrans):
				taskGroups.todos = append(taskGroups.todos, t)
			}
		case strings.EqualFold(t.Status, model.StatusExecuting):
			switch {
			case strings.EqualFold(t.Type, model.TaskTypeCut):
				taskGroups.executingCuts = append(taskGroups.executingCuts, t)
			case strings.EqualFold(t.Type, model.TaskTypeMerge):
				taskGroups.executingMerges = append(taskGroups.executingMerges, t)
			case strings.EqualFold(t.Type, model.TaskTypeTrans):
				taskGroups.executing = append(taskGroups.executing, t)
			}
		}
	}
	consts.TransferTaskMutex.RUnlock()

	if len(taskGroups.executing) == 0 && len(taskGroups.todos) > 0 {
		go fs.TransferFormatter(taskGroups.todos[0])
	}
	if len(taskGroups.executingCuts) == 0 && len(taskGroups.todosCuts) > 0 {
		go fs.CutFormatter(taskGroups.todosCuts[0])
	}
	if len(taskGroups.executingMerges) == 0 && len(taskGroups.todosMerges) > 0 {
		go fs.MergeFiles(taskGroups.todosMerges[0])
	}
}

// HeartBeat 心跳定时扫描
func (fs *fileService) HeartBeat() {
	ticker := time.NewTicker(180 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-TaskCtx.Done():
			return
		case <-ticker.C:
			if !consts.GetOSSetting().EnableTimeScan || time.Since(consts.LastScanTime).Seconds() <= 180 {
				continue
			}
			for _, dir := range consts.GetOSSetting().Dirs {
				removeWalk(dir, true)
			}
		}
	}
}
