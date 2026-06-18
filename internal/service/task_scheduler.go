package service

import (
	"context"
	"search-gin/internal/model"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	TaskCtx, TaskCancel = context.WithCancel(context.Background())
)

var FullScanInProgress atomic.Int32

// HeartBeat 心跳定时扫描
func (fs *searchService) HeartBeat() {
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

// TaskExecuting 任务执行调度器
func (fs *searchService) TaskExecuting() {
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
func (fs *searchService) pollTasks() {
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
		go VideoEncoder.TransferFormatter(taskGroups.todos[0])
	}
	if len(taskGroups.executingCuts) == 0 && len(taskGroups.todosCuts) > 0 {
		go VideoEncoder.CutFormatter(taskGroups.todosCuts[0])
	}
	if len(taskGroups.executingMerges) == 0 && len(taskGroups.todosMerges) > 0 {
		go VideoEncoder.MergeFiles(taskGroups.todosMerges[0])
	}
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
	tasks    map[string]*scanTask // baseDir -> task
	mutex    sync.Mutex
	taskChan chan *scanTask
}

var scanQueue = &taskQueue{
	tasks:    make(map[string]*scanTask),
	taskChan: make(chan *scanTask, 100),
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
		consts.LogMem.Add("扫描任务已取消: %s", task.baseDir)
		return
	}
	select {
	case <-task.cancel:
		consts.LogMem.Add("扫描任务已取消: %s", task.baseDir)
		return
	default:
	}

	// 全量扫描互斥检查
	if FullScanInProgress.Load() != 0 {
		consts.LogMem.Add("全量扫描中，跳过队列任务: %s", task.baseDir)
		return
	}

	// 设置索引构建状态
	atomic.AddInt32(&consts.IndexNumber, 1)
	defer atomic.AddInt32(&consts.IndexNumber, -1)

	consts.LogMem.Add("开始扫描文件夹: %s", task.baseDir)

	// 统计初始化
	consts.TypeMenu.Clear()
	consts.SeriesCount.Clear()
	consts.TagMenu.Clear()
	consts.ClearSmallDir()

	setting := consts.GetOSSetting()
	queryTypes := make([]string, 0)
	queryTypes = utils.ExtendsItems(queryTypes, setting.VideoTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.DocsTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.ImageTypes)

	// 执行扫描
	files, _ := SearchApp.WalkInner(task.baseDir, queryTypes, true, task.baseDir)
	newBucket := newInstanceWithFiles(task.baseDir, files)
	SearchEngine.rebuildWithBucketIncremental(task.baseDir, newBucket)

	clear(queryTypes)

	q.mutex.Lock()
	delete(q.tasks, task.baseDir)
	q.mutex.Unlock()

	consts.LogMem.Add("扫描完成: %s", task.baseDir)
}

// AddTask 添加扫描任务到队列
func (q *taskQueue) AddTask(baseDir string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if existingTask, exists := q.tasks[baseDir]; exists {
		if existingTask.canceled.CompareAndSwap(false, true) {
			close(existingTask.cancel)
		}
		consts.LogMem.Add("取消现有扫描任务，执行新任务: %s", baseDir)
	}

	newTask := &scanTask{
		baseDir:   baseDir,
		cancel:    make(chan struct{}),
		createdAt: time.Now(),
	}
	q.tasks[baseDir] = newTask
	q.taskChan <- newTask

	consts.LogMem.Add("添加扫描任务到队列: %s", baseDir)
}

// GetTaskCount 获取队列中的任务数
func (q *taskQueue) GetTaskCount() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return len(q.tasks)
}
