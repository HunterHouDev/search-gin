package service

import (
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"sync"
	"sync/atomic"
	"time"
)

// 扫描任务队列
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

// StartScanQueue 启动扫描任务队列处理器（由 main.go 在初始化完成后显式调用）
func StartScanQueue() {
	go scanQueue.processTasks()
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
	// 检查任务是否已被取消（优先检查 canceled 标志，避免在已关闭 channel 上 select）
	if task.canceled.Load() {
		AddLogMemory("扫描任务已取消: %s", task.baseDir)
		return
	}
	select {
	case <-task.cancel:
		AddLogMemory("扫描任务已取消: %s", task.baseDir)
		return
	default:
	}

	// 设置索引构建状态
	atomic.AddInt32(&consts.IndexNumber, 1)
	defer atomic.AddInt32(&consts.IndexNumber, -1)

	AddLogMemory("开始扫描文件夹: %s", task.baseDir)

	// 统计初始化
	consts.TypeMenu.Clear()
	consts.SeriesCount.Clear()
	consts.TagMenu.Clear()
	consts.ClearSmallDir()

	// 线程安全：使用 GetOSSetting() 读取配置
	setting := consts.GetOSSetting()
	queryTypes := make([]string, 0)
	queryTypes = utils.ExtendsItems(queryTypes, setting.VideoTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.DocsTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.ImageTypes)

	// 执行扫描
	files, _ := FileApp.WalkInner(task.baseDir, queryTypes, true, task.baseDir)
	newBucket := newInstanceWithFiles(task.baseDir, files)
	// 影子索引：构造新快照并原子替换
	SearchEngine.rebuildWithBucket(task.baseDir, newBucket)

	// 清理
	clear(queryTypes)

	// 任务完成，从队列中移除
	q.mutex.Lock()
	delete(q.tasks, task.baseDir)
	q.mutex.Unlock()

	AddLogMemory("扫描完成: %s", task.baseDir)
}

// AddTask 添加扫描任务到队列
func (q *taskQueue) AddTask(baseDir string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// 检查是否已有相同baseDir的任务
	if existingTask, exists := q.tasks[baseDir]; exists {
		// 使用 CAS 防止重复 close cancel channel 导致 panic
		if existingTask.canceled.CompareAndSwap(false, true) {
			close(existingTask.cancel)
		}
		AddLogMemory("取消现有扫描任务，执行新任务: %s", baseDir)
	}

	// 创建新任务
	newTask := &scanTask{
		baseDir:   baseDir,
		cancel:    make(chan struct{}),
		createdAt: time.Now(),
	}

	// 存储任务
	q.tasks[baseDir] = newTask

	// 发送任务到处理队列
	q.taskChan <- newTask

	AddLogMemory("添加扫描任务到队列: %s", baseDir)
}

// GetTaskCount 获取队列中的任务数
func (q *taskQueue) GetTaskCount() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return len(q.tasks)
}

// TempDir 临时目录路径
var TempDir string

var FileApp = new(fileService)
var SearchApp = new(searchService)

// SearchEngine 搜索引擎
var SearchEngine = searchEngineCore{
 KeywordHistoryCache: utils.NewLRUCache(500),
}

func AddLogMemory(format string, msg ...any) {
	consts.AddLogMemory(format, msg...)
	utils.InfoFormat(format, msg...)

}
