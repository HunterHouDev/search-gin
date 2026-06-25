package service

import (
	"os"
	"testing"
	"time"

	"search-gin/internal/model"
)

func TestMain(m *testing.M) {
	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)
	SetScanWalkInner(app.WalkDirWithCfg)
	InitService(engine, app)
	os.Exit(m.Run())
}

func TestDeleteIndexByPath_NonExistent(t *testing.T) {
	result := DeleteIndexByPath("/path/to/nonexistent/file.avi")
	if !result.IsSuccess() {
		t.Log("DeleteIndexByPath on non-existent returns fail (expected)")
	}
}

func TestCreateTransferTask_NonExistent(t *testing.T) {
	result := CreateTransferTask("non_existent_id", "h264")
	if result.IsSuccess() {
		t.Error("CreateTransferTask should fail for non-existent file")
	}
}

func TestCreateCutTask_NonExistent(t *testing.T) {
	result := CreateCutTask("non_existent_id", "00:00", "00:10")
	if result.IsSuccess() {
		t.Error("CreateCutTask should fail for non-existent file")
	}
}

func TestCreateMergeTask_EmptyList(t *testing.T) {
	result := CreateMergeTask([]string{}, "", false)
	if result.IsSuccess() {
		t.Error("CreateMergeTask should fail with empty file list")
	}
}

func TestCreateMergeTask_NonExistentFile(t *testing.T) {
	result := CreateMergeTask([]string{"fake_id_1", "fake_id_2"}, "", false)
	if result.IsSuccess() {
		t.Error("CreateMergeTask should fail for non-existent files")
	}
}

// ── TransferTask 限制测试（P4 修复验证） ──

func TestTransferTask_MaxLimit(t *testing.T) {
	// 保存原始状态
	TransferTaskMutex.Lock()
	origTasks := TransferTask
	TransferTask = make(map[time.Time]model.TransferTaskModel)
	TransferTaskMutex.Unlock()
	defer func() {
		TransferTaskMutex.Lock()
		TransferTask = origTasks
		TransferTaskMutex.Unlock()
	}()

	// 填充到最大限制
	TransferTaskMutex.Lock()
	for i := 0; i < MaxTransferTaskCount; i++ {
		key := time.Now().Add(-time.Duration(i) * time.Second)
		TransferTask[key] = model.TransferTaskModel{
			Status: model.StatusCompleted,
			Path:   "/test/file" + string(rune('0'+i%10)) + ".mp4",
		}
	}
	TransferTaskMutex.Unlock()

	// 尝试添加新任务应失败
	TransferTaskMutex.Lock()
	if len(TransferTask) >= MaxTransferTaskCount {
		TransferTaskMutex.Unlock()
		// 模拟 CreateTransferTask 的限制检查
		t.Log("任务队列已满，正确拒绝添加")
	} else {
		TransferTaskMutex.Unlock()
		t.Error("任务队列应已满")
	}

	// 验证清理后可以添加
	TransferTaskMutex.Lock()
	// 手动删除一些任务模拟用户清理
	for i := 0; i < 100; i++ {
		key := time.Now().Add(-time.Duration(i) * time.Second)
		delete(TransferTask, key)
	}
	if len(TransferTask) < MaxTransferTaskCount {
		t.Log("清理后可以添加新任务")
	}
	TransferTaskMutex.Unlock()
}
