package service

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"sync"
	"time"
)

// scanResult 目录扫描结果
type scanResult struct {
	dir    string
	bucket *bucketFile
}

// ScanAll 全局扫描
func (s *searchService) ScanAll() int {
	setting := s.settings.Get()
	dirCount := len(setting.Dirs)
	dirList := make([]string, dirCount)
	copy(dirList, setting.Dirs)
	LogMem.Add("Plan to ScanAll dirTotal: %d, dirList: %v", dirCount, dirList)
	if !FullScanInProgress.CompareAndSwap(false, true) {
		LogMem.Add("全量扫描正在进行中")
		return dirCount
	}
	defer FullScanInProgress.Store(false)

	// 初始化扫描进度
	Sp.Init(dirCount)

	s.events.Broadcast(model.SSEScanStart, map[string]interface{}{
		"totalDirs": dirCount,
	})

	ClearSmallDir()
	// 清空搜索引擎缓存
	s.engine.ClearCache()

	queryTypes := make([]string, 0)
	queryTypes = utils.ExtendsItems(queryTypes, setting.VideoTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.DocsTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.ImageTypes)

	// 扫描阶段：并发扫描目录，收集文件
	buckets := s.ScanDirs(dirList, queryTypes)

	// 切换到索引构建阶段
	Sp.SetPhase("building", "正在构建索引...")

	// 构建阶段：批量重建索引
	s.engine.rebuildWithBuckets(buckets)

	// 一致性检查：验证 bucket 数量和目录数量
	bucketCount := s.engine.BucketCount()
	indexNumber := IndexNumber.Load()
	LogMem.Add("ScanAll 一致性检查: BucketCount=%d, IndexNumber=%d, Expected=%d", bucketCount, indexNumber, dirCount)
	if bucketCount != int32(dirCount) {
		LogMem.Add("警告: BucketCount(%d) != Expected(%d)，可能存在并发问题", bucketCount, dirCount)
	}

	SetLastScanTime(time.Now())

	// 扫描完成
	Sp.Complete()

	s.events.Broadcast(model.SSEScanComplete, map[string]interface{}{
		"dirCount":  dirCount,
		"fileCount": s.engine.GetTotalCount(),
	})
	s.events.Broadcast(model.SSEIndexHealth, map[string]interface{}{
		"bucketCount":  bucketCount,
		"indexNumber":  indexNumber,
		"totalCount":   s.engine.GetTotalCount(),
		"lastScanTime": GetLastScanTime(),
	})

	return dirCount
}

// ScanTarget 扫描指定文件夹
func (s *searchService) ScanTarget(baseDir string) {
	scanQueue.AddTask(baseDir)
}

// ScanDirs 并发扫描多文件夹，收集 bucket 结果（不重建索引）
func (s *searchService) ScanDirs(baseDir []string, types []string) map[string]*bucketFile {
	var wg sync.WaitGroup
	dirSize := len(baseDir)

	if dirSize == 0 {
		return make(map[string]*bucketFile)
	}

	resultChan := make(chan scanResult, dirSize)

	wg.Add(dirSize)
	for i := range dirSize {
		go func(dir string) {
			defer wg.Done()
			defer utils.RecoverPanic()
			s.scanDir(dir, types, resultChan)

		}(baseDir[i])
	}

	wg.Wait()
	close(resultChan)

	// 收集所有扫描结果
	buckets := make(map[string]*bucketFile, dirSize)
	for r := range resultChan {
		if r.bucket != nil && !r.bucket.isEmpty() {
			buckets[r.dir] = r.bucket
		}
	}

	LogMem.Add("ScanDirs: 扫描完成, 共 %d 个目录", len(buckets))
	return buckets
}

// scanDir 协程方法扫描单个文件夹并返回结果
func (s *searchService) scanDir(baseDir string, types []string, resultChan chan<- scanResult) {
	defer func() {
		Sp.IncrementCompletedDirs()
	}()

	// 更新当前正在扫描的目录
	Sp.SetCurrentDir(baseDir)

	LogMem.Add("scanDir: 开始扫描目录 %s", baseDir)
	start := time.Now()
	files, size := WalkInner(baseDir, WalkOptions{Recursive: true, Types: types, RootDirs: []string{baseDir}})

	LogMem.Add("scanDir: 扫描完成 %s, 发现 %d 个文件", baseDir, len(files))
	// 更新已扫描文件计数
	Sp.AddScannedFiles(int64(len(files)))

	bucket := newInstanceWithFiles(baseDir, files)

	ti := time.Since(start)
	thisTime := model.FileInfo{
		Name:    baseDir,
		Cnt:     ti.Milliseconds(),
		Size:    int64(len(files)),
		SizeStr: utils.GetSizeStr(size),
	}
	LogMem.Add("扫描目录:[%s] 耗时:[%d] 大小:[%s],剩余目录数:%d", baseDir, ti.Milliseconds(), utils.GetSizeStr(size), IndexNumber.Load())
	AddFolderTime(thisTime)
	s.events.Broadcast(model.SSEScanOneDone, map[string]interface{}{
		"dir":     baseDir,
		"time":    ti.Milliseconds(),
		"size":    int64(len(files)),
		"sizeStr": utils.GetSizeStr(size),
		"remain":  IndexNumber.Load(),
	})
	resultChan <- scanResult{dir: baseDir, bucket: bucket}
}

// Walk 遍历目录，获取指定类型文件列表（轻量版，不建索引）
func (s *searchService) Walk(dirPath string, types []string, deep bool) []model.FileItem {
	files, _ := s.WalkDirWithCfg(dirPath, types, deep)
	return files
}

// WalkDirWithCfg 适配旧调用方，注入 settings.Dirs 后转发到包级 WalkInner。
func (s *searchService) WalkDirWithCfg(currentDir string, types []string, queryChild bool) ([]model.FileItem, int64) {
	rootDirs := s.settings.Get().Dirs
	return WalkInner(currentDir, WalkOptions{Recursive: queryChild, Types: types, RootDirs: rootDirs})
}
