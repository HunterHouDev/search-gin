package service

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var PendingTaskCount atomic.Int32

var FullScanInProgress atomic.Bool

// taskSignal 任务调度信号：有新任务创建或任务完成时唤醒调度器
var taskSignal = make(chan struct{}, 1)

// wakeTaskScheduler 非阻塞通知调度器检查任务
func wakeTaskScheduler() {
	select {
	case taskSignal <- struct{}{}:
	default:
	}
}

// HeartBeat 心跳定时触发增量扫描（goroutine 随进程退出，无需 cancel）
func (s *searchService) HeartBeat() {
	ticker := time.NewTicker(180 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !s.settings.Get().EnableTimeScan || time.Since(GetLastScanTime()).Seconds() <= 180 {
			continue
		}
		for _, dir := range s.settings.Get().Dirs {
			s.ScanTarget(dir)
		}
	}
}

// TaskScheduler 任务调度器：有信号则检查并启动待处理任务，无信号则阻塞休眠
// goroutine 随进程退出自动清理，无需 cancel
func (s *searchService) TaskScheduler() {
	if s == nil {
		utils.ErrorFormat("TaskScheduler: s 为 nil，调度器无法启动")
		return
	}
	utils.InfoFormat("TaskScheduler: 调度器已启动")

	// 启动时立即检查一次，处理可能已有任务
	s.pollTasks()

	for range taskSignal {
		s.pollTasks()
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
					task := t
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
			hasAllExec = hasExecTrans && hasExecCut && hasExecMerge
		}

		if hasAllPending && hasAllExec {
			break
		}
	}
	TransferTaskMutex.RUnlock()

	if pendingTrans != nil && !hasExecTrans {
		LogMem.Add("pollTasks: 启动转码任务 CreateTime=%v, path=%s", pendingTrans.CreateTime, pendingTrans.Path)
		markTaskExecuting(pendingTrans.CreateTime)
		go func() {
			defer utils.RecoverPanic()
			TransferFormatter(*pendingTrans)
		}()
	}
	if pendingCut != nil && !hasExecCut {
		LogMem.Add("pollTasks: 启动分切任务 CreateTime=%v, path=%s", pendingCut.CreateTime, pendingCut.Path)
		markTaskExecuting(pendingCut.CreateTime)
		go func() {
			defer utils.RecoverPanic()
			CutFormatter(*pendingCut)
		}()
	}
	if pendingMerge != nil && !hasExecMerge {
		LogMem.Add("pollTasks: 启动合并任务 CreateTime=%v, path=%s", pendingMerge.CreateTime, pendingMerge.Path)
		markTaskExecuting(pendingMerge.CreateTime)
		go func() {
			defer utils.RecoverPanic()
			MergeFiles(*pendingMerge)
		}()
	}
}

// markTaskExecuting 在 TransferTask map 中原子地将任务标记为执行中
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

type scanTask struct {
	baseDir   string
	cancel    chan struct{}
	canceled  atomic.Bool
	createdAt time.Time
}

type taskQueue struct {
	tasks    map[string]*scanTask
	mutex    sync.Mutex
	taskChan chan *scanTask
	engine   *searchEngineCore
	settings Settings
	walkInner func(string, []string, bool, string) ([]model.FileItem, int64)
}

var scanQueue *taskQueue

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

func SetScanWalkInner(walkInner func(string, []string, bool, string) ([]model.FileItem, int64)) {
	if scanQueue != nil {
		scanQueue.walkInner = walkInner
	}
}

func (q *taskQueue) processTasks() {
	defer utils.RecoverPanic()
	for task := range q.taskChan {
		func() {
			defer utils.RecoverPanic()
			q.executeTask(task)
		}()
	}
}

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

	if FullScanInProgress.Load() {
		LogMem.Add("全量扫描中，跳过队列任务: %s", task.baseDir)
		return
	}

	IndexNumber.Add(1)
	defer IndexNumber.Add(-1)

	LogMem.Add("开始扫描文件夹: %s", task.baseDir)

	setting := q.settings.Get()
	queryTypes := make([]string, 0)
	queryTypes = utils.ExtendsItems(queryTypes, setting.VideoTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.DocsTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.ImageTypes)

	// 一次遍历：收集文件 + 清理空目录
	dirs := setting.Dirs
	files, _ := WalkInner(task.baseDir,
		WalkOptions{Recursive: true, Types: queryTypes, RootDirs: dirs, IsCleanEmpty: true})
	newBucket := newInstanceWithFiles(task.baseDir, files)
	q.engine.rebuildWithBucketIncremental(task.baseDir, newBucket)

	q.mutex.Lock()
	delete(q.tasks, task.baseDir)
	q.mutex.Unlock()

	LogMem.Add("扫描完成: %s", task.baseDir)
}

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

func (q *taskQueue) GetTaskCount() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return len(q.tasks)
}
