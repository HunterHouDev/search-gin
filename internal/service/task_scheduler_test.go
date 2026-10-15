package service

import (
	"search-gin/internal/model"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewScanQueue(t *testing.T) {
	engine := NewSearchEngine()
	q := NewScanQueue(engine, DefaultSettings())
	if q == nil {
		t.Fatal("NewScanQueue returned nil")
	}
	if q.engine == nil {
		t.Error("engine should not be nil")
	}
}

func TestScanQueue_AddTaskAndGetCount(t *testing.T) {
	engine := NewSearchEngine()
	q := NewScanQueue(engine, DefaultSettings())
	if c := q.GetTaskCount(); c != 0 {
		t.Errorf("initial count = %d, want 0", c)
	}

	q.AddTask("/path/to/dir1")
	if c := q.GetTaskCount(); c != 1 {
		t.Errorf("after add count = %d, want 1", c)
	}

	q.AddTask("/path/to/dir2")
	if c := q.GetTaskCount(); c != 2 {
		t.Errorf("after second add count = %d, want 2", c)
	}
}

func TestScanQueue_AddSameDirCancelsPrevious(t *testing.T) {
	engine := NewSearchEngine()
	q := NewScanQueue(engine, DefaultSettings())
	q.AddTask("/path/to/dir")
	c1 := q.GetTaskCount()

	q.AddTask("/path/to/dir")
	c2 := q.GetTaskCount()

	if c2 != c1 {
		t.Errorf("count should stay same after re-add, got %d -> %d", c1, c2)
	}
}

func TestScanQueue_SetWalkInner(t *testing.T) {
	SetScanWalkInner(nil)
	// should not panic when engine is nil
	engine := NewSearchEngine()
	q := NewScanQueue(engine, DefaultSettings())
	SetScanWalkInner(q.walkInner)
}

// ── InitTaskSlots ──

func TestInitTaskSlots_SetsCapacity(t *testing.T) {
	// reset once for test
	taskSlotsOnce = sync.Once{}
	InitTaskSlots(3)
	assert.NotNil(t, taskSlots)
	assert.Equal(t, 3, cap(taskSlots))
}

func TestInitTaskSlots_ZeroDefaultsTo4(t *testing.T) {
	taskSlotsOnce = sync.Once{}
	InitTaskSlots(0)
	assert.Equal(t, 4, cap(taskSlots))
}

func TestInitTaskSlots_NegativeDefaultsTo4(t *testing.T) {
	taskSlotsOnce = sync.Once{}
	InitTaskSlots(-1)
	assert.Equal(t, 4, cap(taskSlots))
}

func TestInitTaskSlots_OnceGuard(t *testing.T) {
	taskSlotsOnce = sync.Once{}
	InitTaskSlots(2)
	prev := taskSlots
	InitTaskSlots(10) // should be ignored
	assert.Equal(t, prev, taskSlots)
	assert.Equal(t, 2, cap(taskSlots))
}

// ── acquireTaskSlot / releaseTaskSlot ──

func TestAcquireSlot_ZeroOrNegativeReturnsTrue(t *testing.T) {
	assert.True(t, acquireTaskSlot(0))
	assert.True(t, acquireTaskSlot(-1))
}

func TestAcquireSlot_BlocksWhenFull(t *testing.T) {
	taskSlotsOnce = sync.Once{}
	InitTaskSlots(1)

	assert.True(t, acquireTaskSlot(1)) // fill the slot
	assert.False(t, acquireTaskSlot(1))
}

func TestReleaseSlot_FreesSlot(t *testing.T) {
	taskSlotsOnce = sync.Once{}
	InitTaskSlots(1)

	acquireTaskSlot(1) // fill
	releaseTaskSlot()

	assert.True(t, acquireTaskSlot(1), "after release should be able to acquire again")
}

func TestReleaseSlot_NilChannelNoPanic(t *testing.T) {
	taskSlots = nil
	releaseTaskSlot() // should not panic
}

// ── taskVCode ──

func TestTaskVCode_TransType(t *testing.T) {
	task := model.TransferTaskModel{Type: model.TaskTypeTrans, VCode: "h264"}
	assert.Equal(t, "h264", taskVCode(task))
}

func TestTaskVCode_NotTransType(t *testing.T) {
	task := model.TransferTaskModel{Type: "cut", VCode: "h264"}
	assert.Empty(t, taskVCode(task))
}

func TestTaskVCode_CaseInsensitive(t *testing.T) {
	task := model.TransferTaskModel{Type: "转码", VCode: "h265"}
	assert.Equal(t, "h265", taskVCode(task))
}

// ── wakeTaskScheduler ──

func TestWakeTaskScheduler_NonBlocking(t *testing.T) {
	// fill the signal channel
	select {
	case <-taskSignal:
	default:
	}
	taskSignal <- struct{}{}

	// second send should not block (non-blocking select)
	wakeTaskScheduler()
}

// ── markTaskExecuting ──

func TestMarkTaskExecuting_UpdatesStatus(t *testing.T) {
	id := time.Now().Format(time.RFC3339Nano)

	TransferTaskMutex.Lock()
	TransferTask[id] = model.TransferTaskModel{
		ID:     id,
		Status: model.StatusPending,
		Path:   "/test/file.mp4",
	}
	TransferTaskMutex.Unlock()

	PendingTaskCount.Store(1)
	markTaskExecuting(id)

	TransferTaskMutex.RLock()
	task := TransferTask[id]
	TransferTaskMutex.RUnlock()
	assert.Equal(t, model.StatusExecuting, task.Status)
	assert.Equal(t, int32(0), PendingTaskCount.Load())

	// cleanup
	TransferTaskMutex.Lock()
	delete(TransferTask, id)
	TransferTaskMutex.Unlock()
}

func TestMarkTaskExecuting_NonExistentNoop(t *testing.T) {
	PendingTaskCount.Store(0)
	markTaskExecuting(time.Now().Format(time.RFC3339Nano))
	assert.Equal(t, int32(0), PendingTaskCount.Load())
}

// ── SetScanWalkInner with nil queue ──

func TestSetScanWalkInner_NilQueue(t *testing.T) {
	oldQueue := scanQueue
	scanQueue = nil
	defer func() { scanQueue = oldQueue }()

	SetScanWalkInner(func(string, []string, bool) ([]model.FileItem, int64) {
		return nil, 0
	})
	// should not panic
}

// TestResizeTaskSlotsKeepsUsed 校验调整并行任务数时已占用的槽位不会丢失：
// 否则在跑的任务会被漏算，实际并行数可能超过新上限。
func TestResizeTaskSlotsKeepsUsed(t *testing.T) {
	taskSlotsOnce = sync.Once{}
	InitTaskSlots(2)
	assert.True(t, acquireTaskSlot(2))
	assert.True(t, acquireTaskSlot(2))
	assert.False(t, acquireTaskSlot(2)) // 两个槽位已占满

	ResizeTaskSlots(5)
	assert.Equal(t, 5, cap(taskSlots))
	// 已占用 2 个，新上限 5 → 只剩 3 个可用
	assert.True(t, acquireTaskSlot(5))
	assert.True(t, acquireTaskSlot(5))
	assert.True(t, acquireTaskSlot(5))
	assert.False(t, acquireTaskSlot(5))
}

// ── 空闲自动重跑失败的分片下载 ────────────────────────────────────────────

// replaceTransferTasks 临时替换全局任务列表，测试结束后还原
func replaceTransferTasks(t *testing.T, tasks map[string]model.TransferTaskModel) {
	t.Helper()
	TransferTaskMutex.Lock()
	orig := TransferTask
	TransferTask = tasks
	TransferTaskMutex.Unlock()
	t.Cleanup(func() {
		TransferTaskMutex.Lock()
		TransferTask = orig
		TransferTaskMutex.Unlock()
		PendingTaskCount.Store(0)
	})
}

// hlsTestID 生成与真实任务同格式、且不含 Windows 非法字符的任务 ID
func hlsTestID(prefix string) string {
	return strings.ReplaceAll(prefix+time.Now().Format(time.RFC3339Nano), ":", "-")
}

// newFailedHlsTask 造一个已失败的分片下载任务，并落盘播放列表（重启依赖它）
func newFailedHlsTask(id string, created time.Time) model.TransferTaskModel {
	saveHlsPlaylist(id, "#EXTM3U\nhttps://example.com/1.ts\n")
	return model.TransferTaskModel{
		ID:         id,
		Type:       model.TaskTypeHls,
		Status:     model.StatusFailed,
		CreateTime: created,
	}
}

func TestEarliestFailedHlsTask_PicksOldest(t *testing.T) {
	now := time.Now()
	older := newFailedHlsTask(hlsTestID("hls-older-"), now.Add(-2*time.Hour))
	newer := newFailedHlsTask(hlsTestID("hls-newer-"), now.Add(-time.Minute))
	replaceTransferTasks(t, map[string]model.TransferTaskModel{
		older.ID: older,
		newer.ID: newer,
		// 其它类型 / 其它状态都不应被选中
		"cut": {
			ID:         "cut",
			Type:       model.TaskTypeCut,
			Status:     model.StatusFailed,
			CreateTime: now.Add(-3 * time.Hour),
		},
		"done": {
			ID:         "done",
			Type:       model.TaskTypeHls,
			Status:     model.StatusCompleted,
			CreateTime: now.Add(-4 * time.Hour),
		},
	})

	got, found := earliestFailedHlsTask()
	assert.True(t, found)
	assert.Equal(t, older.ID, got.ID)
}

func TestMaybeAutoRetryHls_RestartsWhenIdle(t *testing.T) {
	now := time.Now()
	task := newFailedHlsTask(hlsTestID("hls-idle-"), now.Add(-time.Minute))
	replaceTransferTasks(t, map[string]model.TransferTaskModel{task.ID: task})
	ResetHlsAutoRetry(task.ID)

	maybeAutoRetryHls()

	TransferTaskMutex.RLock()
	got := TransferTask[task.ID]
	TransferTaskMutex.RUnlock()
	assert.Equal(t, model.StatusPending, got.Status)
	assert.Equal(t, 1, hlsAutoRetryCount(task.ID))
}

// 队列里还有等待 / 执行中的任务时不抢占：自动重跑只发生在真正空闲的时候
func TestMaybeAutoRetryHls_SkipsWhenBusy(t *testing.T) {
	now := time.Now()
	task := newFailedHlsTask(hlsTestID("hls-busy-"), now.Add(-time.Minute))
	replaceTransferTasks(t, map[string]model.TransferTaskModel{
		task.ID: task,
		"busy": {
			ID:         "busy",
			Type:       model.TaskTypeTrans,
			Status:     model.StatusExecuting,
			CreateTime: now,
		},
	})
	ResetHlsAutoRetry(task.ID)

	maybeAutoRetryHls()

	TransferTaskMutex.RLock()
	got := TransferTask[task.ID]
	TransferTaskMutex.RUnlock()
	assert.Equal(t, model.StatusFailed, got.Status)
	assert.Equal(t, 0, hlsAutoRetryCount(task.ID))
}

// 重跑次数到上限后停手：源站持续失败时不能无限循环
func TestMaybeAutoRetryHls_StopsAfterMax(t *testing.T) {
	now := time.Now()
	task := newFailedHlsTask(hlsTestID("hls-max-"), now.Add(-time.Minute))
	replaceTransferTasks(t, map[string]model.TransferTaskModel{task.ID: task})
	hlsAutoRetry.Store(task.ID, hlsAutoRetryMax)

	maybeAutoRetryHls()

	TransferTaskMutex.RLock()
	got := TransferTask[task.ID]
	TransferTaskMutex.RUnlock()
	assert.Equal(t, model.StatusFailed, got.Status)
	assert.Equal(t, hlsAutoRetryMax, hlsAutoRetryCount(task.ID))
}
