package service

import (
	"search-gin/internal/model"
	"search-gin/pkg/consts"
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
	actorMap    map[string]model.Author

	// 预聚合的菜单数据（写入 consts.* 前暂存）
	typeMenu    map[string]consts.MenuSize
	tagMenu     map[string]consts.MenuSize
	seriesCount map[string]consts.MenuSize
}

// searchEngineCore 搜索引擎：只保留快照指针 + 不变的辅助字段
type searchEngineCore struct {
	index               atomic.Value    // *searchIndex
	KeywordHistoryCache *utils.LRUCache // 搜索结果缓存
	searchPool          *utils.GoroutinePool
	rebuildMu           sync.Mutex     // 防止并发 rebuildWithBucket
	cacheEpoch          atomic.Int64   // 缓存失效纪元，递增触发 cache 清空
	actorSizeCache      []model.Author // PageAuthor 空关键词缓存（按Size排序）
	actorCountCache     []model.Author // PageAuthor 空关键词缓存（按Cnt排序）
}

// loadIndex 线程安全地获取当前快照
func (se *searchEngineCore) loadIndex() *searchIndex {
	s := se.index.Load()
	if s == nil {
		return &searchIndex{
			buckets:  make(map[string]*bucketFile),
			actorMap: make(map[string]model.Author),
		}
	}
	snap, ok := s.(*searchIndex)
	if !ok {
		// 类型不匹配，返回空快照
		return &searchIndex{
			buckets:  make(map[string]*bucketFile),
			actorMap: make(map[string]model.Author),
		}
	}
	return snap
}

// installIndex 原子替换搜索引擎快照，并同步全局菜单
func (se *searchEngineCore) installIndex(snap *searchIndex) {
	se.index.Store(snap)
	se.KeywordHistoryCache.Clear()
	se.cacheEpoch.Add(1)
	se.actorSizeCache = nil
	se.actorCountCache = nil

	// 同步菜单到全局 consts（首页等模块使用）
	consts.TypeMenu.Clear()
	for k, v := range snap.typeMenu {
		consts.TypeMenu.Store(k, v)
	}
	consts.TagMenu.Clear()
	for k, v := range snap.tagMenu {
		consts.TagMenu.Store(k, v)
	}
	consts.SeriesCount.Clear()
	for k, v := range snap.seriesCount {
		consts.SeriesCount.Store(k, v)
	}
	consts.LastScanTime = time.Now()

	// 异步保存缓存快照，保证下次启动时能恢复本次索引状态
	saveIndexToCache(snap)
}

// Reset 清空搜索引擎全部状态和缓存
func (se *searchEngineCore) Reset() {
	empty := &searchIndex{
		buckets:  make(map[string]*bucketFile),
		actorMap: make(map[string]model.Author),
	}
	se.installIndex(empty)
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
