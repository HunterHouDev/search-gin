package service

import (
	"path/filepath"
	"testing"
	"time"

	"search-gin/internal/model"

	"github.com/stretchr/testify/assert"
)

// TestHlsTaskStoreRoundTrip 校验下载列表的落盘 / 恢复：
// 只存分片下载；执行中的任务恢复后回到等待并重新排队（进程退出时它并没有真的跑完）。
func TestHlsTaskStoreRoundTrip(t *testing.T) {
	oldDir := GetWorkDir()
	SetWorkDir(t.TempDir())
	t.Cleanup(func() { SetWorkDir(oldDir) })

	// 清空全局任务表，测试结束后还原
	TransferTaskMutex.Lock()
	orig := TransferTask
	TransferTask = map[string]model.TransferTaskModel{
		"done-1": {
			ID: "done-1", Type: model.TaskTypeHls, Status: model.StatusCompleted,
			Path: filepath.Join(GetWorkDir(), "a.mp4"), CreateTime: time.Now().Add(-time.Hour),
		},
		"run-1": {
			ID: "run-1", Type: model.TaskTypeHls, Status: model.StatusExecuting,
			Path: filepath.Join(GetWorkDir(), "b.mp4"), CreateTime: time.Now(),
		},
		// 非分片下载不入库：转码依赖进程内的 ffmpeg 子进程，恢复出来是僵尸任务
		"trans-1": {ID: "trans-1", Type: model.TaskTypeTrans, Status: model.StatusCompleted},
	}
	TransferTaskMutex.Unlock()
	PendingTaskCount.Store(0)
	t.Cleanup(func() {
		TransferTaskMutex.Lock()
		TransferTask = orig
		TransferTaskMutex.Unlock()
		PendingTaskCount.Store(0)
	})

	doFlushHlsTaskStore()

	// 模拟进程重启：内存任务表清空
	TransferTaskMutex.Lock()
	TransferTask = map[string]model.TransferTaskModel{}
	TransferTaskMutex.Unlock()
	PendingTaskCount.Store(0)

	restored := LoadHlsTaskStore()
	assert.Equal(t, 2, restored)

	TransferTaskMutex.RLock()
	gotDone := TransferTask["done-1"]
	gotRun := TransferTask["run-1"]
	_, hasTrans := TransferTask["trans-1"]
	TransferTaskMutex.RUnlock()

	assert.Equal(t, model.StatusCompleted, gotDone.Status)
	// 执行中 → 等待：重新启动后由调度器接着跑（有 .part 则续传）
	assert.Equal(t, model.StatusPending, gotRun.Status)
	assert.False(t, hasTrans)
	assert.Equal(t, int32(1), PendingTaskCount.Load())

	// 重复加载不会把同一个任务插两遍
	assert.Equal(t, 0, LoadHlsTaskStore())
}

// TestHlsTaskStoreMissingFile 快照不存在时直接返回 0，不报错
func TestHlsTaskStoreMissingFile(t *testing.T) {
	oldDir := GetWorkDir()
	SetWorkDir(t.TempDir())
	t.Cleanup(func() { SetWorkDir(oldDir) })

	assert.Equal(t, 0, LoadHlsTaskStore())
}
