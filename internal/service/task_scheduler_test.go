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

// ── InitTaskSlots（两套独立槽位池）──

// resetTaskSlotsForTest 把槽位池与 Once 守卫全部清空，避免全局状态跨用例泄漏
// （rebuildSlots 会从旧 channel 迁移已占用的槽位，不清 nil 会让上一条用例遗留的占用带过来）。
func resetTaskSlotsForTest() {
	taskSlotsOnce = sync.Once{}
	processSlots = nil
	downloadSlots = nil
}

func TestInitTaskSlots_SetsCapacity(t *testing.T) {
	resetTaskSlotsForTest()
	InitTaskSlots(3, 5)
	assert.NotNil(t, processSlots)
	assert.NotNil(t, downloadSlots)
	assert.Equal(t, 3, cap(processSlots))
	assert.Equal(t, 5, cap(downloadSlots))
}

func TestInitTaskSlots_ZeroDefaultsTo4(t *testing.T) {
	resetTaskSlotsForTest()
	InitTaskSlots(0, 0)
	assert.Equal(t, 4, cap(processSlots))
	assert.Equal(t, 4, cap(downloadSlots))
}

func TestInitTaskSlots_NegativeDefaultsTo4(t *testing.T) {
	resetTaskSlotsForTest()
	InitTaskSlots(-1, -1)
	assert.Equal(t, 4, cap(processSlots))
	assert.Equal(t, 4, cap(downloadSlots))
}

func TestInitTaskSlots_OnceGuard(t *testing.T) {
	resetTaskSlotsForTest()
	InitTaskSlots(2, 7)
	prevProc := processSlots
	prevDl := downloadSlots
	InitTaskSlots(10, 10) // should be ignored
	assert.Equal(t, prevProc, processSlots)
	assert.Equal(t, prevDl, downloadSlots)
	assert.Equal(t, 2, cap(processSlots))
	assert.Equal(t, 7, cap(downloadSlots))
}

// ── acquire/release 槽位：处理池与下载池各自独立 ──

func TestAcquireProcessSlot_ZeroOrNegativeReturnsTrue(t *testing.T) {
	assert.True(t, acquireProcessSlot(0))
	assert.True(t, acquireProcessSlot(-1))
}

func TestAcquireDownloadSlot_ZeroOrNegativeReturnsTrue(t *testing.T) {
	assert.True(t, acquireDownloadSlot(0))
	assert.True(t, acquireDownloadSlot(-1))
}

func TestAcquireProcessSlot_BlocksWhenFull(t *testing.T) {
	resetTaskSlotsForTest()
	InitTaskSlots(1, 1)

	assert.True(t, acquireProcessSlot(1)) // fill the process slot
	assert.False(t, acquireProcessSlot(1))
}

func TestAcquireDownloadSlot_BlocksWhenFull(t *testing.T) {
	resetTaskSlotsForTest()
	InitTaskSlots(1, 1)

	assert.True(t, acquireDownloadSlot(1)) // fill the download slot
	assert.False(t, acquireDownloadSlot(1))
}

// 处理池被占满时，下载池仍能正常调度——验证两者互不阻塞
func TestPools_IndependentlyBlocked(t *testing.T) {
	resetTaskSlotsForTest()
	InitTaskSlots(1, 1)

	assert.True(t, acquireProcessSlot(1))  // 处理池占满
	assert.False(t, acquireProcessSlot(1)) // 处理池无法再取
	assert.True(t, acquireDownloadSlot(1)) // 下载池仍可取，不受处理池影响
}

func TestReleaseProcessSlot_FreesSlot(t *testing.T) {
	resetTaskSlotsForTest()
	InitTaskSlots(1, 1)

	acquireProcessSlot(1) // fill
	releaseProcessSlot()

	assert.True(t, acquireProcessSlot(1), "after release should be able to acquire again")
}

func TestReleaseDownloadSlot_FreesSlot(t *testing.T) {
	resetTaskSlotsForTest()
	InitTaskSlots(1, 1)

	acquireDownloadSlot(1) // fill
	releaseDownloadSlot()

	assert.True(t, acquireDownloadSlot(1), "after release should be able to acquire again")
}

func TestReleaseSlot_NilChannelNoPanic(t *testing.T) {
	processSlots = nil
	downloadSlots = nil
	releaseProcessSlot()  // should not panic
	releaseDownloadSlot() // should not panic
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

// TestResizeTaskSlotsKeepsUsed 校验调整并行任务数时已占用的处理槽位不会丢失：
// 否则在跑的任务会被漏算，实际并行数可能超过新上限。
func TestResizeTaskSlotsKeepsUsed(t *testing.T) {
	resetTaskSlotsForTest()
	InitTaskSlots(2, 4)
	assert.True(t, acquireProcessSlot(2))
	assert.True(t, acquireProcessSlot(2))
	assert.False(t, acquireProcessSlot(2)) // 两个槽位已占满

	ResizeTaskSlots(5)
	assert.Equal(t, 5, cap(processSlots))
	// 已占用 2 个，新上限 5 → 只剩 3 个可用
	assert.True(t, acquireProcessSlot(5))
	assert.True(t, acquireProcessSlot(5))
	assert.True(t, acquireProcessSlot(5))
	assert.False(t, acquireProcessSlot(5))
}

// TestResizeDownloadSlotsKeepsUsed 校验调整下载并行上限时已占用的下载槽位不丢失
func TestResizeDownloadSlotsKeepsUsed(t *testing.T) {
	resetTaskSlotsForTest()
	InitTaskSlots(4, 2)
	assert.True(t, acquireDownloadSlot(2))
	assert.True(t, acquireDownloadSlot(2))
	assert.False(t, acquireDownloadSlot(2)) // 两个槽位已占满

	ResizeDownloadSlots(5)
	assert.Equal(t, 5, cap(downloadSlots))
	assert.True(t, acquireDownloadSlot(5))
	assert.True(t, acquireDownloadSlot(5))
	assert.True(t, acquireDownloadSlot(5))
	assert.False(t, acquireDownloadSlot(5))
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

// 分片下载队列里还有执行中的下载任务时不抢占：自动重跑只发生在下载真正空闲时。
// 注意：转码/剪辑/合并属于另一套槽位池，它们的忙碌不应再阻塞下载的自动重跑
// （这正是「下载与转码互不阻塞」要达成的效果）。
func TestMaybeAutoRetryHls_SkipsWhenBusy(t *testing.T) {
	now := time.Now()
	task := newFailedHlsTask(hlsTestID("hls-busy-"), now.Add(-time.Minute))
	replaceTransferTasks(t, map[string]model.TransferTaskModel{
		task.ID: task,
		"busy-dl": {
			ID:         "busy-dl",
			Type:       model.TaskTypeHls,
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

// 转码忙碌时，下载空闲的失败任务仍应被自动重跑——两套槽位池互不阻塞
func TestMaybeAutoRetryHls_RunsWhileTranscodeBusy(t *testing.T) {
	now := time.Now()
	task := newFailedHlsTask(hlsTestID("hls-while-trans-"), now.Add(-time.Minute))
	replaceTransferTasks(t, map[string]model.TransferTaskModel{
		task.ID: task,
		"busy-trans": {
			ID:         "busy-trans",
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
	assert.Equal(t, model.StatusPending, got.Status)
	assert.Equal(t, 1, hlsAutoRetryCount(task.ID))
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
