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
	TaskCtx, TaskCancel = context.WithCancel(context.Background())
)

var FullScanInProgress atomic.Bool

// HeartBeat 心跳定时扫描
func (s *searchService) HeartBeat() {
	ticker := time.NewTicker(180 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-TaskCtx.Done():
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
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		s.pollTasks()
		select {
		case <-TaskCtx.Done():
			utils.InfoFormat("任务调度器已停止")
			return
		case <-ticker.C:
		}
	}
}

// pollTasks 轮询并执行待处理任务
func (s *searchService) pollTasks() {
	taskGroups := struct {
		todos           []model.TransferTaskModel
		todosCuts       []model.TransferTaskModel
		todosMerges     []model.TransferTaskModel
		executing       []model.TransferTaskModel
		executingCuts   []model.TransferTaskModel
		executingMerges []model.TransferTaskModel
	}{}

	TransferTaskMutex.RLock()
	for _, t := range TransferTask {
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
	TransferTaskMutex.RUnlock()

	// 启动前在 map 中原子标记为 "执行中"，防止下次 poll 周期重复启动
	if len(taskGroups.executing) == 0 && len(taskGroups.todos) > 0 {
		markTaskExecuting(taskGroups.todos[0].CreateTime)
		go TransferFormatter(taskGroups.todos[0])
	}
	if len(taskGroups.executingCuts) == 0 && len(taskGroups.todosCuts) > 0 {
		markTaskExecuting(taskGroups.todosCuts[0].CreateTime)
		go CutFormatter(taskGroups.todosCuts[0])
	}
	if len(taskGroups.executingMerges) == 0 && len(taskGroups.todosMerges) > 0 {
		markTaskExecuting(taskGroups.todosMerges[0].CreateTime)
		go MergeFiles(taskGroups.todosMerges[0])
	}
}

// markTaskExecuting 在 TransferTask map 中原子地将任务标记为执行中
// 在启动 goroutine 之前调用，消除 pollTasks 竞态窗口
func markTaskExecuting(key time.Time) {
	TransferTaskMutex.Lock()
	if t, ok := TransferTask[key]; ok {
		t.Status = model.StatusExecuting
		TransferTask[key] = t
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
