package service

import (
	"fmt"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// defaultTaskParallel 未配置时的处理并发数（转码+剪辑+合并）
	defaultTaskParallel = 4
	// maxTaskParallel 处理并发数上限：再高也压不出吞吐，只会让任务互相抢 CPU
	maxTaskParallel = 16
	// defaultDownloadParallel 未配置时的下载并发数
	defaultDownloadParallel = 4
	// maxDownloadParallel 下载并发数上限
	maxDownloadParallel = 16
	// hlsAutoRetryMax 单个失败任务自动重跑的次数上限。
	// 源站彻底不可用 / 播放列表失效时必须有终点，
	// 否则「失败 → 空闲 → 重跑 → 再失败」会无限循环。
	hlsAutoRetryMax = 3
)

// ── 两套独立的并发槽位池 ────────────────────────────────────────────────
//
// 分片下载是纯网络 I/O，转码/剪辑/合并是 CPU/ffmpeg 密集，
// 两类任务的瓶颈完全不同。之前共用一个总槽位池，下载铺满时转码排不上、
// 转码占满时下载也被卡住。拆成两个池后两者各自独立调度、互不阻塞，
// 各自的并行上限也分开配置（setting.downloadMaxConcurrent / taskMaxConcurrent）。
var (
	downloadSlots  chan struct{} // 分片下载槽位池
	processSlots   chan struct{} // 转码/剪辑/合并槽位池
	taskSlotsMu    sync.Mutex    // 保护两个信号量的替换（并行数支持运行时调整）
	transcodeCount atomic.Int32  // 转码（h264/h265）执行数，共用 1 个处理槽
	taskSlotsOnce  sync.Once
)

// ensureSlotsInited 兜底：槽位池尚未初始化时按默认上限补建，避免首任务到来时拿到 nil
func ensureSlotsInited() {
	if downloadSlots == nil || processSlots == nil {
		InitTaskSlots(defaultTaskParallel, defaultDownloadParallel)
	}
}

var PendingTaskCount atomic.Int32

var FullScanInProgress atomic.Bool

// taskSignal 任务调度信号：有新任务创建或任务完成时唤醒调度器
var taskSignal = make(chan struct{}, 1)

// InitTaskSlots 初始化两类并发槽位（由 StartBackgroundTasks 调用）
func InitTaskSlots(processMax, downloadMax int) {
	taskSlotsOnce.Do(func() {
		resizeProcessSlots(processMax)
		resizeDownloadSlots(downloadMax)
	})
}

// ResizeTaskSlots 调整转码/剪辑/合并的并行上限（与下载槽位互不影响）。
// 已占用的槽位会迁移到新的信号量：否则调整瞬间会漏算在跑的任务，
// 实际并行数可能超过新上限。
func ResizeTaskSlots(maxConcurrent int) {
	taskSlotsMu.Lock()
	defer taskSlotsMu.Unlock()
	resizeProcessSlots(maxConcurrent)
}

// ResizeDownloadSlots 调整分片下载的并行上限（与处理槽位互不影响）。
func ResizeDownloadSlots(maxConcurrent int) {
	taskSlotsMu.Lock()
	defer taskSlotsMu.Unlock()
	resizeDownloadSlots(maxConcurrent)
}

// resizeProcessSlots 重建处理槽位信号量，调用方需持有 taskSlotsMu
func resizeProcessSlots(maxConcurrent int) {
	if maxConcurrent <= 0 {
		maxConcurrent = defaultTaskParallel
	}
	if maxConcurrent > maxTaskParallel {
		maxConcurrent = maxTaskParallel
	}
	processSlots = rebuildSlots(processSlots, maxConcurrent)
}

// resizeDownloadSlots 重建下载槽位信号量，调用方需持有 taskSlotsMu
func resizeDownloadSlots(maxConcurrent int) {
	if maxConcurrent <= 0 {
		maxConcurrent = defaultDownloadParallel
	}
	if maxConcurrent > maxDownloadParallel {
		maxConcurrent = maxDownloadParallel
	}
	downloadSlots = rebuildSlots(downloadSlots, maxConcurrent)
}

// rebuildSlots 用新上限重建信号量并迁移已占用的槽位，调用方需持有 taskSlotsMu
func rebuildSlots(old chan struct{}, maxConcurrent int) chan struct{} {
	// 排空旧信号量即可得到当前已占用的槽位数
	used := 0
	if old != nil {
	drain:
		for {
			select {
			case <-old:
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
	return next
}

// applyProcessParallel 调整转码/剪辑/合并的并行上限：写入设置并立即重建处理槽位。
// ≤0 或超过上限时忽略，保持服务端现有配置不变。
func applyProcessParallel(n int) {
	if n <= 0 || n > maxTaskParallel {
		return
	}
	setting := GetOSSetting()
	if setting.TaskMaxConcurrent == n && processSlots != nil {
		return
	}
	setting.TaskMaxConcurrent = n
	SetOSSetting(setting)
	if err := FlushDictionary(SettingFileName); err != nil {
		utils.ErrorFormat("applyProcessParallel: 设置落盘失败: %v", err)
	}
	ResizeTaskSlots(n)
	utils.InfoFormat("applyProcessParallel: 处理并行上限调整为 %d", n)
}

// applyDownloadParallel 调整分片下载的并行上限：写入设置并立即重建下载槽位。
// 与处理上限完全独立，互不干扰。
func applyDownloadParallel(n int) {
	if n <= 0 || n > maxDownloadParallel {
		return
	}
	setting := GetOSSetting()
	if setting.DownloadMaxConcurrent == n && downloadSlots != nil {
		return
	}
	setting.DownloadMaxConcurrent = n
	SetOSSetting(setting)
	if err := FlushDictionary(SettingFileName); err != nil {
		utils.ErrorFormat("applyDownloadParallel: 设置落盘失败: %v", err)
	}
	ResizeDownloadSlots(n)
	utils.InfoFormat("applyDownloadParallel: 下载并行上限调整为 %d", n)
}

// acquireDownloadSlot 占用一个下载槽位，无可用槽位时返回 false（不阻塞）
func acquireDownloadSlot(max int) bool {
	if max <= 0 {
		return true // 不限制
	}
	slots := currentDownloadSlots()
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

// currentDownloadSlots 取当前下载槽位信号量；未初始化时按需建一个
func currentDownloadSlots() chan struct{} {
	taskSlotsMu.Lock()
	slots := downloadSlots
	taskSlotsMu.Unlock()
	return slots
}

// releaseDownloadSlot 释放一个下载槽位
func releaseDownloadSlot() {
	taskSlotsMu.Lock()
	slots := downloadSlots
	taskSlotsMu.Unlock()
	if slots == nil {
		return
	}
	select {
	case <-slots:
	default:
	}
}

// acquireProcessSlot 占用一个处理（转码/剪辑/合并）槽位，无可用槽位时返回 false（不阻塞）
func acquireProcessSlot(max int) bool {
	if max <= 0 {
		return true // 不限制
	}
	slots := currentProcessSlots()
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

// currentProcessSlots 取当前处理槽位信号量；未初始化时按需建一个
func currentProcessSlots() chan struct{} {
	taskSlotsMu.Lock()
	slots := processSlots
	taskSlotsMu.Unlock()
	return slots
}

// releaseProcessSlot 释放一个处理槽位
func releaseProcessSlot() {
	taskSlotsMu.Lock()
	slots := processSlots
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

// ── 空闲自动重跑失败的分片下载 ────────────────────────────────────────────
//
// 下载是串行推进的：一批任务跑完后总有「队列空转」的间隙。
// 空闲时把最早失败的任务重新拉起，省去人工点重启；
// 手动重启会重置额度（见 RestartHlsDownload），失败太多次则停手不再打扰。

// hlsAutoRetry 任务 ID → 已自动重跑的次数
var hlsAutoRetry sync.Map

// hlsPendingExecutingCount 只统计分片下载的等待/执行中数量，
// 不被转码/剪辑/合并的任务干扰——否则一个长时间转码会让失败的下载迟迟不被自动重跑。
func hlsPendingExecutingCount() (pending, executing int) {
	TransferTaskMutex.RLock()
	defer TransferTaskMutex.RUnlock()
	for _, t := range TransferTask {
		if !strings.EqualFold(t.Type, model.TaskTypeHls) {
			continue
		}
		switch t.Status {
		case model.StatusPending:
			pending++
		case model.StatusExecuting:
			executing++
		}
	}
	return
}

// maybeAutoRetryHls 没有等待 / 执行中的分片下载任务时，自动重启最早失败的下载任务。
//
// 每次只补一个：哪怕有多个失败任务也先跑最早的，
// 避免一次性占满下载并行槽位，也让重试顺序对用户可预期。
func maybeAutoRetryHls() {
	pending, executing := hlsPendingExecutingCount()
	if pending > 0 || executing > 0 {
		return
	}
	task, found := earliestFailedHlsTask()
	if !found {
		return
	}
	used := hlsAutoRetryCount(task.ID)
	if used >= hlsAutoRetryMax {
		return
	}

	res := RestartHlsDownload(task.ID)
	// 重启失败同样计入次数：播放列表丢失这类问题重试也不会成功
	hlsAutoRetry.Store(task.ID, used+1)
	next := used + 1
	if res.IsSuccess() {
		utils.InfoFormat("maybeAutoRetryHls: 空闲自动重跑失败任务 id=%s（第 %d/%d 次）", task.ID, next, hlsAutoRetryMax)
		appendHlsLog(task.ID, fmt.Sprintf("空闲自动重跑：第 %d/%d 次", next, hlsAutoRetryMax))
		return
	}
	utils.InfoFormat("maybeAutoRetryHls: 自动重跑未成功 id=%s（第 %d/%d 次）: %s", task.ID, next, hlsAutoRetryMax, res.Message)
}

// earliestFailedHlsTask 找出创建时间最早的分片下载失败任务；
// 顺带清理已不在任务列表里的重跑计数，避免长期运行后无限堆积。
func earliestFailedHlsTask() (model.TransferTaskModel, bool) {
	TransferTaskMutex.RLock()
	defer TransferTaskMutex.RUnlock()

	alive := make(map[string]struct{}, len(TransferTask))
	var earliest model.TransferTaskModel
	found := false
	for id, t := range TransferTask {
		alive[id] = struct{}{}
		if !strings.EqualFold(t.Type, model.TaskTypeHls) || t.Status != model.StatusFailed {
			continue
		}
		if !found || t.CreateTime.Before(earliest.CreateTime) {
			earliest = t
			found = true
		}
	}
	hlsAutoRetry.Range(func(key, _ any) bool {
		if id, ok := key.(string); ok {
			if _, ok := alive[id]; !ok {
				hlsAutoRetry.Delete(id)
			}
		}
		return true
	})
	return earliest, found
}

// hlsAutoRetryCount 取任务已自动重跑的次数（未记录过时为 0）
func hlsAutoRetryCount(taskID string) int {
	if v, ok := hlsAutoRetry.Load(taskID); ok {
		if n, ok := v.(int); ok {
			return n
		}
	}
	return 0
}

// ResetHlsAutoRetry 清空任务的自动重跑次数。
// 手动重启意味着用户已经介入处理，此时应当重新给满自动重试的额度。
func ResetHlsAutoRetry(taskID string) {
	hlsAutoRetry.Delete(taskID)
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
	// 空闲补跑放在最前面：没有等待 / 执行中的下载任务时重启最早失败的下载，
	// 重启成功后 PendingTaskCount 变正，下面的流程会立刻把它拉起来
	maybeAutoRetryHls()

	if PendingTaskCount.Load() == 0 {
		return
	}
	setting := s.settings.Get()
	processMax := setting.TaskMaxConcurrent
	downloadMax := setting.DownloadMaxConcurrent
	if processMax <= 0 {
		processMax = 4
	}
	if downloadMax <= 0 {
		downloadMax = 4
	}
	ensureSlotsInited()

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
			// 分片下载：纯网络 I/O，走独立的下载槽位池，与转码/剪辑/合并互不阻塞
			canStart = acquireDownloadSlot(downloadMax)

		case strings.EqualFold(task.Type, model.TaskTypeCut):
			canStart = !engineEmpty && acquireProcessSlot(processMax)

		case strings.EqualFold(task.Type, model.TaskTypeMerge):
			canStart = !engineEmpty && acquireProcessSlot(processMax)

		case strings.EqualFold(task.Type, model.TaskTypeTrans):
			// 转码吃满 CPU，处理池内只允许 1 个在跑
			canStart = !engineEmpty && transcodeCount.Load() == 0 && acquireProcessSlot(processMax)
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

			isDownload := strings.EqualFold(t.Type, model.TaskTypeHls)
			isTranscode := strings.EqualFold(t.Type, model.TaskTypeTrans)
			if isDownload {
				defer releaseDownloadSlot()
			} else {
				defer releaseProcessSlot()
			}
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
	// 状态变为执行中：同步到下载列表快照
	SaveHlsTaskStore()
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
