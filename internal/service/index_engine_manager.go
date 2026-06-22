package service

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"sync"
	"sync/atomic"
	"time"
)

// repeatModel 重复文件模型（内部使用）
type repeatModel struct {
	Code  string
	Files model.FileItem
	Count int
}

// searchIndex 搜索引擎的快照（不可变，通过 atomic.Value 原子替换）
type searchIndex struct {
	buckets     map[string]*bucketFile // baseDir → bucket
	bucketCount int32
	totalSize   int64
	totalCount  int
	repeatFiles []model.FileItem
	authorMap   map[string]model.Author

	// 预聚合的菜单数据（写入 consts.* 前暂存）
	typeMenu    map[string]model.FileInfo
	tagMenu     map[string]model.FileInfo
	seriesCount map[string]model.FileInfo
}

// searchEngineCore 搜索引擎：只保留快照指针 + 不变的辅助字段
type searchEngineCore struct {
	index               atomic.Value    // *searchIndex
	KeywordHistoryCache *utils.LRUCache // 搜索结果缓存
	searchPool          *utils.GoroutinePool
	rebuildMu           sync.Mutex     // 防止并发 rebuildWithBucket
	cacheEpoch          atomic.Int64   // 缓存失效纪元，递增触发 cache 清空
	repeatsDirty        atomic.Bool    // 单文件操作后标记重复文件列表需要重算
	authorCacheMu       sync.RWMutex   // 保护 authorSizeCache/authorCountCache
	authorSizeCache     []model.Author // PageAuthor 空关键词缓存（按Size排序）
	authorCountCache    []model.Author // PageAuthor 空关键词缓存（按Cnt排序）
}

// fileOp 延迟文件操作
type fileOp struct {
	opType  string // "replace" | "delete"
	oldFile model.FileItem
	newFile model.FileItem // 仅 replace 有效
}

// loadIndex 线程安全地获取当前快照
func (se *searchEngineCore) loadIndex() *searchIndex {
	s := se.index.Load()
	if s == nil {
		return &searchIndex{
			buckets:     make(map[string]*bucketFile),
			authorMap:   make(map[string]model.Author),
			typeMenu:    make(map[string]model.FileInfo),
			tagMenu:     make(map[string]model.FileInfo),
			seriesCount: make(map[string]model.FileInfo),
		}
	}
	index, ok := s.(*searchIndex)
	if !ok {
		return &searchIndex{
			buckets:     make(map[string]*bucketFile),
			authorMap:   make(map[string]model.Author),
			typeMenu:    make(map[string]model.FileInfo),
			tagMenu:     make(map[string]model.FileInfo),
			seriesCount: make(map[string]model.FileInfo),
		}
	}
	return index
}

// installIndex 原子替换索引 + 同步全局菜单 + 异步持久化磁盘缓存
func (se *searchEngineCore) installIndex(index *searchIndex) {
	se.syncIndex(index)
	saveIndexToCache(index)
}

// installIndexSkipDisk 原子替换索引 + 清 LRU 缓存（跳过磁盘持久化，单文件操作用）
func (se *searchEngineCore) installIndexSkipDisk(index *searchIndex) {
	se.syncIndex(index)
}

// syncIndex 原子替换索引 + 清 LRU 缓存 + 递增 epoch
func (se *searchEngineCore) syncIndex(index *searchIndex) {
	se.index.Store(index)
	se.KeywordHistoryCache.Clear()
	se.cacheEpoch.Add(1)
	se.authorCacheMu.Lock()
	se.authorSizeCache = nil
	se.authorCountCache = nil
	se.authorCacheMu.Unlock()

	SetLastScanTime(time.Now())
}

// GetTypeMenu 从当前索引快照获取类型菜单
func (se *searchEngineCore) GetTypeMenu() map[string]model.FileInfo {
	return se.loadIndex().typeMenu
}

// GetTagMenu 从当前索引快照获取标签菜单
func (se *searchEngineCore) GetTagMenu() map[string]model.FileInfo {
	return se.loadIndex().tagMenu
}

// GetSeriesCount 从当前索引快照获取系列统计
func (se *searchEngineCore) GetSeriesCount() map[string]model.FileInfo {
	return se.loadIndex().seriesCount
}

// IsEmpty 检查是否有 bucket 数据
func (se *searchEngineCore) IsEmpty() bool {
	return len(se.loadIndex().buckets) == 0
}

// GetTotalCount 获取文件总数
func (se *searchEngineCore) GetTotalCount() int {
	return se.loadIndex().totalCount
}

// GetTotalSize 获取文件总大小
func (se *searchEngineCore) GetTotalSize() int64 {
	return se.loadIndex().totalSize
}

// BucketCount 返回 bucket 数量
func (se *searchEngineCore) BucketCount() int32 {
	return se.loadIndex().bucketCount
}
