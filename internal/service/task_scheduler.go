package service

import (
	"context"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	TaskCtx, TaskCancel           = context.WithCancel(context.Background())
	HeartBeatCtx, HeartBeatCancel = context.WithCancel(context.Background())
)

var PendingTaskCount atomic.Int32

var FullScanInProgress atomic.Bool

// TaskNotify 任务变更通知通道：有任务加入或完成时发送信号，消除空闲轮询
var TaskNotify = make(chan struct{}, 1)

// notifyTaskChange 非阻塞通知调度器有任务变更
func notifyTaskChange() {
	select {
	case TaskNotify <- struct{}{}:
	default:
	}
}

// HeartBeat 心跳定时扫描
func (s *searchService) HeartBeat() {
	ticker := time.NewTicker(180 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-HeartBeatCtx.Done():
			return
		case <-ticker.C:
			if !s.settings.Get().EnableTimeScan || time.Since(GetLastScanTime()).Seconds() <= 180 {
				continue
			}
			for _, dir := range s.settings.Get().Dirs {
				removeWalk(dir, true)
			}
		}
	}
}

// TaskExecuting 任务执行调度器
func (s *searchService) TaskExecuting() {
	if s == nil {
		utils.ErrorFormat("TaskExecuting: s 为 nil，调度器无法启动")
		return
	}
	utils.InfoFormat("TaskExecuting: 调度器已启动")

	// 启动时立即扫描一次，处理可能已有任务
	s.pollTasks()

	// fallback ticker：channel 通知为主信号，ticker 作为兜底（任务完成通知等边界情况）
	fallback := time.NewTicker(10 * time.Second)
	defer fallback.Stop()

	for {
		select {
		case <-TaskCtx.Done():
			utils.InfoFormat("TaskExecuting: 调度器已停止")
			return
		case <-TaskNotify:
			s.pollTasks()
		case <-fallback.C:
			s.pollTasks()
		}
	}
}

// pollTasks 轮询并执行待处理任务
func (s *searchService) pollTasks() {
	// 快速路径：没有待处理任务则直接返回
	if PendingTaskCount.Load() == 0 {
		return
	}

	// 只需要每个类别的第一个 pending 任务 + 检查是否有 executing 防止重复启动
	var pendingTrans, pendingCut, pendingMerge *model.TransferTaskModel
	var hasExecTrans, hasExecCut, hasExecMerge bool

	TransferTaskMutex.RLock()
	for _, t := range TransferTask {
		hasAllPending := false
		hasAllExec := false

		switch {
		case strings.EqualFold(t.Status, model.StatusPending):
			switch {
			case strings.EqualFold(t.Type, model.TaskTypeCut):
				if pendingCut == nil {
					task := t // 复制，range 变量在下一轮会变
					pendingCut = &task
				}
			case strings.EqualFold(t.Type, model.TaskTypeMerge):
				if pendingMerge == nil {
					task := t
					pendingMerge = &task
				}
			case strings.EqualFold(t.Type, model.TaskTypeTrans):
				if pendingTrans == nil {
					task := t
					pendingTrans = &task
				}
			}
			// 3 个 pending 都找到了？
			hasAllPending = pendingTrans != nil && pendingCut != nil && pendingMerge != nil

		case strings.EqualFold(t.Status, model.StatusExecuting):
			switch {
			case strings.EqualFold(t.Type, model.TaskTypeCut):
				hasExecCut = true
			case strings.EqualFold(t.Type, model.TaskTypeMerge):
				hasExecMerge = true
			case strings.EqualFold(t.Type, model.TaskTypeTrans):
				hasExecTrans = true
			}
			// 3 个 executing 状态都知道了？
			hasAllExec = hasExecTrans && hasExecCut && hasExecMerge
		}

		// 信息收集完毕，提前退出遍历
		if hasAllPending && hasAllExec {
			break
		}
	}
	TransferTaskMutex.RUnlock()

	// 按类别启动第一个待处理任务（无同类执行中任务时）
	if pendingTrans != nil && !hasExecTrans {
		LogMem.Add("pollTasks: 启动转码任务 CreateTime=%v, path=%s", pendingTrans.CreateTime, pendingTrans.Path)
		markTaskExecuting(pendingTrans.CreateTime)
		go TransferFormatter(*pendingTrans)
	}
	if pendingCut != nil && !hasExecCut {
		LogMem.Add("pollTasks: 启动分切任务 CreateTime=%v, path=%s", pendingCut.CreateTime, pendingCut.Path)
		markTaskExecuting(pendingCut.CreateTime)
		go CutFormatter(*pendingCut)
	}
	if pendingMerge != nil && !hasExecMerge {
		LogMem.Add("pollTasks: 启动合并任务 CreateTime=%v, path=%s", pendingMerge.CreateTime, pendingMerge.Path)
		markTaskExecuting(pendingMerge.CreateTime)
		go MergeFiles(*pendingMerge)
	}
}

// markTaskExecuting 在 TransferTask map 中原子地将任务标记为执行中
// 在启动 goroutine 之前调用，消除 pollTasks 竞态窗口
func markTaskExecuting(key time.Time) {
	TransferTaskMutex.Lock()
	if t, ok := TransferTask[key]; ok {
		t.Status = model.StatusExecuting
		TransferTask[key] = t
		PendingTaskCount.Add(-1)
	}
	TransferTaskMutex.Unlock()
}

// ── 扫描任务队列 ──────────────────────────────────────────────────

// 扫描任务
type scanTask struct {
	baseDir   string
	cancel    chan struct{}
	canceled  atomic.Bool
	createdAt time.Time
}

type taskQueue struct {
	tasks     map[string]*scanTask // baseDir -> task
	mutex     sync.Mutex
	taskChan  chan *scanTask
	engine    *searchEngineCore
	settings  Settings
	walkInner func(string, []string, bool, string) ([]model.FileItem, int64)
}

var scanQueue *taskQueue

// NewScanQueue 创建扫描任务队列（由 main.go 显式调用）
func NewScanQueue(engine *searchEngineCore, settings Settings) *taskQueue {
	q := &taskQueue{
		tasks:    make(map[string]*scanTask),
		taskChan: make(chan *scanTask, 100),
		engine:   engine,
		settings: settings,
	}
	scanQueue = q
	return q
}

// SetScanWalkInner 设置扫描队列的 WalkInner 回调（需要 searchService 实例化后才能调用）
func SetScanWalkInner(walkInner func(string, []string, bool, string) ([]model.FileItem, int64)) {
	if scanQueue != nil {
		scanQueue.walkInner = walkInner
	}
}

// processTasks 处理任务队列
func (q *taskQueue) processTasks() {
	defer utils.RecoverPanic()
	for task := range q.taskChan {
		func() {
			defer utils.RecoverPanic()
			q.executeTask(task)
		}()
	}
}

// executeTask 执行单个扫描任务
func (q *taskQueue) executeTask(task *scanTask) {
	if task.canceled.Load() {
		LogMem.Add("扫描任务已取消: %s", task.baseDir)
		return
	}
	select {
	case <-task.cancel:
		LogMem.Add("扫描任务已取消: %s", task.baseDir)
		return
	default:
	}

	// 全量扫描互斥检查
	if FullScanInProgress.Load() {
		LogMem.Add("全量扫描中，跳过队列任务: %s", task.baseDir)
		return
	}

	// 设置索引构建状态
	IndexNumber.Add(1)
	defer IndexNumber.Add(-1)

	LogMem.Add("开始扫描文件夹: %s", task.baseDir)

	setting := q.settings.Get()
	queryTypes := make([]string, 0)
	queryTypes = utils.ExtendsItems(queryTypes, setting.VideoTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.DocsTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.ImageTypes)

	// 执行扫描
	files, _ := q.walkInner(task.baseDir, queryTypes, true, task.baseDir)
	newBucket := newInstanceWithFiles(task.baseDir, files)
	q.engine.rebuildWithBucketIncremental(task.baseDir, newBucket)

	q.mutex.Lock()
	delete(q.tasks, task.baseDir)
	q.mutex.Unlock()

	LogMem.Add("扫描完成: %s", task.baseDir)
}

// AddTask 添加扫描任务到队列
func (q *taskQueue) AddTask(baseDir string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if existingTask, exists := q.tasks[baseDir]; exists {
		if existingTask.canceled.CompareAndSwap(false, true) {
			close(existingTask.cancel)
		}
		LogMem.Add("取消现有扫描任务，执行新任务: %s", baseDir)
	}

	newTask := &scanTask{
		baseDir:   baseDir,
		cancel:    make(chan struct{}),
		createdAt: time.Now(),
	}
	q.tasks[baseDir] = newTask
	q.taskChan <- newTask

	LogMem.Add("添加扫描任务到队列: %s", baseDir)
}

// GetTaskCount 获取队列中的任务数
func (q *taskQueue) GetTaskCount() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return len(q.tasks)
}
