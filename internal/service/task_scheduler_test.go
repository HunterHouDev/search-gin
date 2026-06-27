package service

import (
	"search-gin/internal/model"
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
	now := time.Now()

	TransferTaskMutex.Lock()
	TransferTask[now] = model.TransferTaskModel{
		Status: model.StatusPending,
		Path:   "/test/file.mp4",
	}
	TransferTaskMutex.Unlock()

	PendingTaskCount.Store(1)
	markTaskExecuting(now)

	TransferTaskMutex.RLock()
	task := TransferTask[now]
	TransferTaskMutex.RUnlock()
	assert.Equal(t, model.StatusExecuting, task.Status)
	assert.Equal(t, int32(0), PendingTaskCount.Load())

	// cleanup
	TransferTaskMutex.Lock()
	delete(TransferTask, now)
	TransferTaskMutex.Unlock()
}

func TestMarkTaskExecuting_NonExistentNoop(t *testing.T) {
	PendingTaskCount.Store(0)
	markTaskExecuting(time.Now())
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
