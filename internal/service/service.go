package service

import (
	"search-gin/internal/model"
	"search-gin/internal/sse"
	"search-gin/pkg/utils"
)

// ── searchService ──────────────────────────────────────────────────

// searchService 文件操作/扫描/流媒体的业务实现。
// 所有方法通过字段访问依赖（s.engine / s.settings / s.events），不使用全局变量。
type searchService struct {
	engine    *searchEngineCore
	settings  Settings
	events    EventBus
	scanQueue *taskQueue
}

// ── 包级变量（仅保留必要的） ──────────────────────────────────────

var workDir string

// SetWorkDir 设置工作目录路径，由 main.go 启动时调用。
func SetWorkDir(dir string) { workDir = dir }

// GetWorkDir 获取工作目录路径。
func GetWorkDir() string { return workDir }

// ── 全局引擎/应用（内部使用，由 InitService 设置） ────────────────

var (
	globalEngine *searchEngineCore
	globalSearch *searchService
)

// InitService 注册全局引擎和应用实例。
// 供包内辅助函数（remote_operation.go、task_service.go 等）通过 Getter 访问。
func InitService(engine *searchEngineCore, search *searchService) {
	globalEngine = engine
	globalSearch = search
}

func GetEngine() *searchEngineCore { return globalEngine }
func GetSearch() *searchService    { return globalSearch }

// ── 构造函数（由 main.go 显式调用） ──────────────────────────────

// NewSearchEngine 创建搜索引擎实例。
func NewSearchEngine() *searchEngineCore {
	return &searchEngineCore{
		KeywordHistoryCache: utils.NewLRUCache(500),
	}
}

// NewSearchService 创建搜索服务实例，注入所有依赖。
func NewSearchService(engine *searchEngineCore, settings Settings, events EventBus, scanQueue *taskQueue) *searchService {
	return &searchService{
		engine:    engine,
		settings:  settings,
		events:    events,
		scanQueue: scanQueue,
	}
}

// ── Settings / EventBus 默认适配器 ────────────────────────

type settingsAdapter struct{}

func (settingsAdapter) Get() model.Setting  { return GetOSSetting() }
func (settingsAdapter) Set(s model.Setting) { SetOSSetting(s) }
func (settingsAdapter) Flush(path string)   { FlushDictionary(path) }

type sseAdapter struct{}

func (sseAdapter) Broadcast(event string, data map[string]interface{}) {
	sse.BroadcastEvent(event, data)
}

// DefaultSettings 返回默认配置适配器（桥接到全局 GetOSSetting()）。
func DefaultSettings() Settings {
	return settingsAdapter{}
}

// DefaultEventBus 返回默认事件总线适配器（桥接到 sse.BroadcastEvent()）。
func DefaultEventBus() EventBus {
	return sseAdapter{}
}
