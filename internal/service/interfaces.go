package service

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
)

// IndexEngine 搜索引擎抽象，定义搜索、查找、索引管理等核心能力。
// 实现者：*searchEngineCore（通过 atomic.Value 持有 searchIndex 快照）
type IndexEngine interface {
	Page(searchParam model.SearchParam) utils.Page
	PageAuthor(searchParam model.SearchParam) model.PageAuthorResultWrapper
	FindById(id string) model.FileItem
	FindAuthorByName(name string) model.Author
	GetAuthorCount() int
	IsEmpty() bool
	GetTotalCount() int
	GetTotalSize() int64
	BucketCount() int32
	DeleteFile(file model.FileItem)
	ReplaceFile(oldFile, newFile model.FileItem)
	GetTypeMenu() map[string]model.FileInfo
	GetTagMenu() map[string]model.FileInfo
	GetSeriesCount() map[string]model.FileInfo
}

// FileService 文件操作抽象，定义扫描、标签、重命名、移动、删除等能力。
// 实现者：*searchService（组合 engine + settings + events + scanQueue）
type FileService interface {
	SetMovieType(movie model.FileItem, movieType string) utils.Result
	AddTag(id string, tag string) utils.Result
	ClearTag(id string, tag string) utils.Result
	Rename(movie model.FileEdit) utils.Result
	Move(id string, newDir string, title string) utils.Result
	Delete(id string)
	ScanAll() int
	ScanTarget(baseDir string)
	Walk(dir string, types []string, withSub bool) []model.FileItem
	DeleteOne(dirName string, fileName string)
	DownDeleteDir(dirname string)
}

// Settings 配置读写抽象，封装 setting.json 的读取、写入、持久化。
// 实现者：settingsAdapter（桥接全局 GetOSSetting()/SetOSSetting()/FlushDictionary()）
type Settings interface {
	Get() model.Setting
	Set(s model.Setting)
	Flush(path string)
}

// EventBus 事件广播抽象，用于服务层向外部（SSE、集群）发送事件通知。
// 实现者：sseAdapter（桥接 sse.BroadcastEvent()）
type EventBus interface {
	Broadcast(event string, data map[string]interface{})
}
