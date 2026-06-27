package service

import (
	"context"
	"encoding/gob"
	"os"
	"path/filepath"
	"sync"
	"time"

	"search-gin/internal/model"
	"search-gin/pkg/utils"
)

// ─── 可序列化的缓存结构（不含 sync.RWMutex 等不可序列化字段） ───

// cacheBucket — bucketFile 的数据副本
type cacheBucket struct {
	InstanceName string
	TotalSize    int64
	TotalCount   int
	FileLib      map[string]*model.FileItem
	TypeIndex    map[string]map[string]struct{}
}

// cacheData — searchIndex 的数据副本
type cacheData struct {
	Buckets     []cacheBucket
	RepeatFiles []model.FileItem
	AuthorMap   map[string]*model.Author
	TypeMenu    map[string]model.FileInfo
	TagMenu     map[string]model.FileInfo
	SeriesCount map[string]model.FileInfo
}

const cacheFileName = "search_cache.gob"

// ── 去抖写入 ──────────────────────────────────────────────────────

// 多次 rebuild 在 cacheDebounceInterval 内合并为一次磁盘写入
const cacheDebounceInterval = 5 * time.Second

var (
	cacheDebounceMu    sync.Mutex
	cacheDebounceTimer *time.Timer
	cacheDebounceIndex *searchIndex
)

// saveIndexToCache 带去抖的异步缓存写入
// 30 秒窗口内的多次 rebuild 合并为一次 gob 序列化 + 磁盘写入
// 空快照（无 bucket）不保存，避免 Reset() 等路径清空磁盘缓存
func saveIndexToCache(index *searchIndex) {
	if GetWorkDir() == "" {
		return
	}
	if len(index.buckets) == 0 {
		return
	}

	cacheDebounceMu.Lock()
	cacheDebounceIndex = index
	if cacheDebounceTimer != nil {
		cacheDebounceTimer.Stop()
	}
	cacheDebounceTimer = time.AfterFunc(cacheDebounceInterval, flushCache)
	cacheDebounceMu.Unlock()
}

// flushCache 去抖超时后异步执行一次序列化 + 磁盘写入
func flushCache() {
	cacheDebounceMu.Lock()
	idx := cacheDebounceIndex
	cacheDebounceTimer = nil
	cacheDebounceMu.Unlock()

	if idx == nil {
		return
	}

	go func() {
		defer utils.RecoverPanic()
		doFlushCache(idx)
	}()
}

// FlushCache 同步强制写入缓存（用于 shutdown 信号处理，确保退出前持久化）
func FlushCache() {
	cacheDebounceMu.Lock()
	idx := cacheDebounceIndex
	// 取消待处理的定时器，避免重复写入
	if cacheDebounceTimer != nil {
		cacheDebounceTimer.Stop()
		cacheDebounceTimer = nil
	}
	cacheDebounceIndex = nil
	cacheDebounceMu.Unlock()

	if idx == nil {
		return
	}

	doFlushCache(idx)
}

// doFlushCache 执行实际的序列化 + 磁盘写入（同步，带 30s 超时）
func doFlushCache(idx *searchIndex) {
	cachePath := filepath.Join(GetWorkDir(), cacheFileName)

	// 转换为可序列化的 cacheData
	data := cacheData{
		RepeatFiles: idx.repeatFiles,
		AuthorMap:   idx.authorMap,
		TypeMenu:    idx.typeMenu,
		TagMenu:     idx.tagMenu,
		SeriesCount: idx.seriesCount,
		Buckets:     make([]cacheBucket, 0, len(idx.buckets)),
	}
	for _, b := range idx.buckets {
		b.mu.RLock()
		data.Buckets = append(data.Buckets, cacheBucket{
			InstanceName: b.InstanceName,
			TotalSize:    b.TotalSize,
			TotalCount:   b.TotalCount,
			FileLib:      b.FileLib,
			TypeIndex:    b.TypeIndex,
		})
		b.mu.RUnlock()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer utils.RecoverPanic()
		defer close(done)
		// 检查 context 是否已超时，避免超时后继续写盘
		if ctx.Err() != nil {
			return
		}
		tmpPath := cachePath + ".tmp"
		f, err := os.Create(tmpPath)
		if err != nil {
			utils.InfoFormat("保存索引缓存失败(创建临时文件): %v", err)
			return
		}

		enc := gob.NewEncoder(f)
		if err := enc.Encode(data); err != nil {
			f.Close()
			os.Remove(tmpPath)
			utils.InfoFormat("保存索引缓存失败(编码): %v", err)
			return
		}
		f.Close()

		if err := os.Rename(tmpPath, cachePath); err != nil {
			utils.InfoFormat("保存索引缓存失败(重命名): %v", err)
			return
		}
		LogMem.Add("索引缓存已保存: %s (%d buckets, %d files)", cachePath, len(data.Buckets), idx.totalCount)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		utils.ErrorFormat("保存索引缓存超时(30s): %s", cachePath)
	}
}

// LoadCachedIndex 从缓存文件加载快照，成功则安装到搜索引擎并返回 true
func (se *searchEngineCore) LoadCachedIndex() bool {
	if GetWorkDir() == "" {
		return false
	}
	cachePath := filepath.Join(GetWorkDir(), cacheFileName)

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return false
	}

	f, err := os.Open(cachePath)
	if err != nil {
		utils.InfoFormat("加载索引缓存失败(打开文件): %v", err)
		return false
	}
	defer f.Close()

	var data cacheData
	dec := gob.NewDecoder(f)
	if err := dec.Decode(&data); err != nil {
		utils.InfoFormat("加载索引缓存失败(解码): %v，将重新扫描", err)
		return false
	}

	// 转换为 searchIndex
	buckets := make(map[string]*bucketFile, len(data.Buckets))
	var totalSize int64
	var totalCount int
	idIndex := make(map[string]*model.FileItem, 5000)
	for _, cb := range data.Buckets {
		b := &bucketFile{
			InstanceName: cb.InstanceName,
			TotalSize:    cb.TotalSize,
			TotalCount:   cb.TotalCount,
			FileLib:      cb.FileLib,
			TypeIndex:    cb.TypeIndex,
		}
		buckets[cb.InstanceName] = b
		totalSize += cb.TotalSize
		totalCount += cb.TotalCount
		// 重建 idIndex（O(n)，缓存启动时一次性执行）
		for id, f := range cb.FileLib {
			idIndex[id] = f
		}
	}

	index := &searchIndex{
		buckets:     buckets,
		bucketCount: int32(len(data.Buckets)),
		totalSize:   totalSize,
		totalCount:  totalCount,
		repeatFiles: data.RepeatFiles,
		authorMap:   data.AuthorMap,
		idIndex:     idIndex,
		typeMenu:    data.TypeMenu,
		tagMenu:     data.TagMenu,
		seriesCount: data.SeriesCount,
	}

	se.installIndex(index)
	LogMem.Add("索引缓存已加载: %s (%d buckets, %d files)", cachePath, len(data.Buckets), totalCount)
	return true
}
