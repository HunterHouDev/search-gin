package service

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
)

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

// ── Phase 2: 新增接口 ────────────────────────────────────────────

// Settings 配置读写抽象，替代全局 GetOSSetting()/SetOSSetting()
type Settings interface {
	Get() model.Setting
	Set(s model.Setting)
	Flush(path string)
}

// EventBus 事件广播抽象，替代 searchService 方法中的 sse.BroadcastEvent()
type EventBus interface {
	Broadcast(event string, data map[string]interface{})
}
