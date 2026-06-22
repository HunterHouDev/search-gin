package service

import (
	"context"
	"encoding/gob"
	"os"
	"path/filepath"
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
	FileLib      map[string]model.FileItem
	TypeIndex    map[string]map[string]struct{}
}

// cacheData — searchIndex 的数据副本
type cacheData struct {
	Buckets     []cacheBucket
	RepeatFiles []model.FileItem
	AuthorMap   map[string]model.Author
	TypeMenu    map[string]model.FileInfo
	TagMenu     map[string]model.FileInfo
	SeriesCount map[string]model.FileInfo
}

const cacheFileName = "search_cache.gob"

// saveIndexToCache 将当前快照异步保存到缓存文件
// 空快照（无 bucket）不保存，避免 Reset() 等路径清空磁盘缓存
func saveIndexToCache(index *searchIndex) {
	if GetWorkDir() == "" {
		return
	}
	if len(index.buckets) == 0 {
		return
	}
	cachePath := filepath.Join(GetWorkDir(), cacheFileName)

	// 转换为可序列化的 cacheData
	data := cacheData{
		RepeatFiles: index.repeatFiles,
		AuthorMap:   index.authorMap,
		TypeMenu:    index.typeMenu,
		TagMenu:     index.tagMenu,
		SeriesCount: index.seriesCount,
		Buckets:     make([]cacheBucket, 0, len(index.buckets)),
	}
	for _, b := range index.buckets {
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

	// 异步写入，带超时保护，不阻塞扫描流程
	go func() {
		defer utils.RecoverPanic()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		done := make(chan struct{})
		go func() {
			defer close(done)
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
			LogMem.Add("索引缓存已保存: %s (%d buckets, %d files)", cachePath, len(data.Buckets), index.totalCount)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			utils.ErrorFormat("保存索引缓存超时(30s): %s", cachePath)
		}
	}()
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
	}

	index := &searchIndex{
		buckets:     buckets,
		bucketCount: int32(len(data.Buckets)),
		totalSize:   totalSize,
		totalCount:  totalCount,
		repeatFiles: data.RepeatFiles,
		authorMap:   data.AuthorMap,
		typeMenu:    data.TypeMenu,
		tagMenu:     data.TagMenu,
		seriesCount: data.SeriesCount,
	}

	se.installIndex(index)
	LogMem.Add("索引缓存已加载: %s (%d buckets, %d files)", cachePath, len(data.Buckets), totalCount)
	return true
}
