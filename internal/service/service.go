package service

import (
	"search-gin/internal/model"
	"search-gin/internal/sse"
	"search-gin/pkg/utils"
)

// ── searchService ──────────────────────────────────────────────────

type searchService struct {
	engine    *searchEngineCore
	settings  Settings
	events    EventBus
	scanQueue *taskQueue
}

// ── 包级变量（仅保留必要的） ──────────────────────────────────────

var workDir string

func SetWorkDir(dir string) { workDir = dir }
func GetWorkDir() string    { return workDir }

// ── 全局引擎/应用（内部使用，由 InitService 设置） ────────────────

var (
	globalEngine *searchEngineCore
	globalSearch *searchService
)

func InitService(engine *searchEngineCore, search *searchService) {
	globalEngine = engine
	globalSearch = search
}

func GetEngine() *searchEngineCore { return globalEngine }
func GetSearch() *searchService    { return globalSearch }

// ── 构造函数 ──────────────────────────────────────────────────────

func NewSearchEngine() *searchEngineCore {
	return &searchEngineCore{
		KeywordHistoryCache: utils.NewLRUCache(500),
	}
}

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

// DefaultSettings 返回默认配置适配器，供 handler 等外部包使用
func DefaultSettings() Settings {
	return settingsAdapter{}
}

// DefaultEventBus 返回默认事件总线适配器
func DefaultEventBus() EventBus {
	return sseAdapter{}

}
