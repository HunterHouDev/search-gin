package service

import (
	"encoding/gob"
	"os"
	"path/filepath"

	"search-gin/internal/model"
	"search-gin/pkg/consts"
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
	ActorMap    map[string]model.Author
	TypeMenu    map[string]consts.MenuSize
	TagMenu     map[string]consts.MenuSize
	SeriesCount map[string]consts.MenuSize
}

const cacheFileName = "search_cache.gob"

// saveIndexToCache 将当前快照异步保存到缓存文件
// 空快照（无 bucket）不保存，避免 Reset() 等路径清空磁盘缓存
func saveIndexToCache(snap *searchIndex) {
	if WorkDir == "" {
		return
	}
	if len(snap.buckets) == 0 {
		return
	}
	cachePath := filepath.Join(WorkDir, cacheFileName)

	// 转换为可序列化的 cacheData
	data := cacheData{
		RepeatFiles: snap.repeatFiles,
		ActorMap:    snap.actorMap,
		TypeMenu:    snap.typeMenu,
		TagMenu:     snap.tagMenu,
		SeriesCount: snap.seriesCount,
		Buckets:     make([]cacheBucket, 0, len(snap.buckets)),
	}
	for _, b := range snap.buckets {
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

	// 异步写入，不阻塞扫描流程
	go func() {
		defer utils.RecoverPanic()

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

		// 原子替换：先写 .tmp 再 Rename，防止写入中断导致文件损坏
		if err := os.Rename(tmpPath, cachePath); err != nil {
			utils.InfoFormat("保存索引缓存失败(重命名): %v", err)
			return
		}
		consts.LogMem.Add("索引缓存已保存: %s (%d buckets, %d files)", cachePath, len(data.Buckets), snap.totalCount)
	}()
}

// LoadCachedIndex 从缓存文件加载快照，成功则安装到搜索引擎并返回 true
func (se *searchEngineCore) LoadCachedIndex() bool {
	if WorkDir == "" {
		return false
	}
	cachePath := filepath.Join(WorkDir, cacheFileName)

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

	snap := &searchIndex{
		buckets:     buckets,
		bucketCount: int32(len(data.Buckets)),
		totalSize:   totalSize,
		totalCount:  totalCount,
		repeatFiles: data.RepeatFiles,
		actorMap:    data.ActorMap,
		typeMenu:    data.TypeMenu,
		tagMenu:     data.TagMenu,
		seriesCount: data.SeriesCount,
	}

	se.installIndex(snap)
	consts.LogMem.Add("索引缓存已加载: %s (%d buckets, %d files)", cachePath, len(data.Buckets), totalCount)
	return true
}
