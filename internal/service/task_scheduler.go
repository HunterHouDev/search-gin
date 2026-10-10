package service

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// defaultTaskParallel 未配置时的并行任务数（与 setting 的默认值一致）
	defaultTaskParallel = 4
	// maxTaskParallel 并行任务数上限：再高也压不出吞吐，只会让任务互相抢带宽
	maxTaskParallel = 16
)

// 任务并发控制
var (
	taskSlots      chan struct{} // 总并发槽位信号量
	taskSlotsMu    sync.Mutex    // 保护 taskSlots 的替换（并行任务数支持运行时调整）
	transcodeCount atomic.Int32  // 转码（h264/h265）执行数，共用 1 个槽
	taskSlotsOnce  sync.Once
)

var PendingTaskCount atomic.Int32

var FullScanInProgress atomic.Bool

// taskSignal 任务调度信号：有新任务创建或任务完成时唤醒调度器
var taskSignal = make(chan struct{}, 1)

// InitTaskSlots 初始化任务并发槽位（由 StartBackgroundTasks 调用）
func InitTaskSlots(maxConcurrent int) {
	taskSlotsOnce.Do(func() {
		resizeTaskSlots(maxConcurrent)
	})
}

// ResizeTaskSlots 调整并行任务槽位总数（下载任务提交时可带上该值）。
// 已占用的槽位会迁移到新的信号量：否则调整瞬间会漏算在跑的任务，
// 实际并行数可能超过新上限。
func ResizeTaskSlots(maxConcurrent int) {
	taskSlotsMu.Lock()
	defer taskSlotsMu.Unlock()
	resizeTaskSlots(maxConcurrent)
}

// resizeTaskSlots 重建槽位信号量，调用方需持有 taskSlotsMu
func resizeTaskSlots(maxConcurrent int) {
	if maxConcurrent <= 0 {
		maxConcurrent = defaultTaskParallel
	}
	if maxConcurrent > maxTaskParallel {
		maxConcurrent = maxTaskParallel
	}
	// 排空旧信号量即可得到当前已占用的槽位数
	used := 0
	if taskSlots != nil {
	drain:
		for {
			select {
			case <-taskSlots:
				used++
			default:
				break drain
			}
		}
	}
	next := make(chan struct{}, maxConcurrent)
	// 新旧上限不一致时（例如 8 → 2 且已占用 5），多出的占用无法迁移，
	// 这里按新上限截断——在跑的任务不受影响，只是调度器暂时不再放行新任务
	for i := 0; i < used && i < maxConcurrent; i++ {
		next <- struct{}{}
	}
	taskSlots = next
}

// applyTaskParallel 调整全局并行任务数上限：写入设置并立即重建槽位。
// ≤0 或超过上限时忽略，保持服务端现有配置不变。
func applyTaskParallel(n int) {
	if n <= 0 || n > maxTaskParallel {
		return
	}
	setting := GetOSSetting()
	if setting.TaskMaxConcurrent == n && taskSlots != nil {
		return
	}
	setting.TaskMaxConcurrent = n
	SetOSSetting(setting)
	if err := FlushDictionary(SettingFileName); err != nil {
		utils.ErrorFormat("applyTaskParallel: 设置落盘失败: %v", err)
	}
	ResizeTaskSlots(n)
	utils.InfoFormat("applyTaskParallel: 并行任务数调整为 %d", n)
}

// acquireTaskSlot 占用一个并发槽位，无可用槽位时返回 false（不阻塞）
func acquireTaskSlot(max int) bool {
	if max <= 0 {
		return true // 不限制
	}
	slots := currentTaskSlots(max)
	if slots == nil {
		return false
	}
	select {
	case slots <- struct{}{}:
		return true
	default:
		return false // 槽位满
	}
}

// currentTaskSlots 取当前槽位信号量；未初始化时按需建一个
func currentTaskSlots(max int) chan struct{} {
	taskSlotsMu.Lock()
	slots := taskSlots
	taskSlotsMu.Unlock()
	if slots == nil {
		InitTaskSlots(max)
		taskSlotsMu.Lock()
		slots = taskSlots
		taskSlotsMu.Unlock()
	}
	return slots
}

// releaseTaskSlot 释放一个并发槽位
func releaseTaskSlot() {
	taskSlotsMu.Lock()
	slots := taskSlots
	taskSlotsMu.Unlock()
	if slots == nil {
		return
	}
	select {
	case <-slots:
	default:
	}
}

// taskVCode 从任务中提取编码器类型
func taskVCode(task model.TransferTaskModel) string {
	if strings.EqualFold(task.Type, model.TaskTypeTrans) {
		return task.VCode
	}
	return ""
}

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

// pollTasks 轮询并执行待处理任务（槽位调度）
func (s *searchService) pollTasks() {
	if PendingTaskCount.Load() == 0 {
		return
	}
	maxSlot := s.settings.Get().TaskMaxConcurrent
	if maxSlot <= 0 {
		maxSlot = 4
	}
	if taskSlots == nil {
		InitTaskSlots(maxSlot)
	}

	// 索引为空时，依赖索引的任务（分切/合并/转码）不调度；
	// 分片下载不依赖索引，仍可正常执行
	engineEmpty := GetEngine().IsEmpty()

	var toStart []model.TransferTaskModel

	TransferTaskMutex.RLock()
	for _, t := range TransferTask {
		if !strings.EqualFold(t.Status, model.StatusPending) {
			continue
		}

		task := t
		canStart := false

		switch {
		case strings.EqualFold(task.Type, model.TaskTypeHls):
			// 分片下载：纯网络 I/O，不依赖索引
			canStart = acquireTaskSlot(maxSlot)

		case strings.EqualFold(task.Type, model.TaskTypeCut):
			canStart = !engineEmpty && acquireTaskSlot(maxSlot)

		case strings.EqualFold(task.Type, model.TaskTypeMerge):
			canStart = !engineEmpty && acquireTaskSlot(maxSlot)

		case strings.EqualFold(task.Type, model.TaskTypeTrans):
			canStart = !engineEmpty && transcodeCount.Load() == 0 && acquireTaskSlot(maxSlot)
		}

		if !canStart {
			continue
		}

		toStart = append(toStart, task)
	}
	TransferTaskMutex.RUnlock()

	for _, task := range toStart {
		markTaskExecuting(task.ID)
		LogMem.Add("pollTasks: 启动任务 ID=%v, type=%s, path=%s", task.CreateTime, task.Type, task.Path)

		t := task
		go func() {
			defer utils.RecoverPanic()
			defer wakeTaskScheduler()
			defer releaseTaskSlot()

			isTranscode := strings.EqualFold(t.Type, model.TaskTypeTrans)
			if isTranscode {
				transcodeCount.Add(1)
				defer transcodeCount.Add(-1)
			}

			switch {
			case strings.EqualFold(t.Type, model.TaskTypeTrans):
				TransferFormatter(t)
			case strings.EqualFold(t.Type, model.TaskTypeCut):
				CutFormatter(t)
			case strings.EqualFold(t.Type, model.TaskTypeMerge):
				MergeFiles(t)
			case strings.EqualFold(t.Type, model.TaskTypeHls):
				HlsDownloader(t)
			}
		}()
	}
}

// markTaskExecuting 在 TransferTask map 中原子地将任务标记为执行中
func markTaskExecuting(key string) {
	TransferTaskMutex.Lock()
	if t, ok := TransferTask[key]; ok {
		if t.Status != model.StatusPending {
			TransferTaskMutex.Unlock()
			return
		}
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
	tasks     map[string]*scanTask
	mutex     sync.Mutex
	taskChan  chan *scanTask
	engine    *searchEngineCore
	settings  Settings
	walkInner func(string, []string, bool) ([]model.FileItem, int64)
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

func SetScanWalkInner(walkInner func(string, []string, bool) ([]model.FileItem, int64)) {
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
	// 清空搜索引擎缓存
	q.engine.ClearCache()

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
