package service

import (
	"os"
	"path/filepath"
	"search-gin/internal/model"
	"search-gin/internal/sse"
	"search-gin/pkg/consts"
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
	setting := GetOSSetting()
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

	sse.BroadcastEvent("scan_start", map[string]interface{}{
		"totalDirs": dirCount,
	})

	ClearSmallDir()
	InitFolderTime()

	queryTypes := make([]string, 0)
	queryTypes = utils.ExtendsItems(queryTypes, setting.VideoTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.DocsTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.ImageTypes)

	// 扫描阶段：并发扫描目录，收集文件
	buckets := s.ScanDirs(dirList, queryTypes)

	// 切换到索引构建阶段
	Sp.SetPhase("building", "正在构建索引...")

	// 构建阶段：批量重建索引
	SearchEngine.rebuildWithBuckets(buckets)

	// 一致性检查：验证 bucket 数量和目录数量
	bucketCount := SearchEngine.BucketCount()
	indexNumber := consts.IndexNumber.Load()
	LogMem.Add("ScanAll 一致性检查: BucketCount=%d, IndexNumber=%d, Expected=%d", bucketCount, indexNumber, dirCount)
	if bucketCount != int32(dirCount) {
		LogMem.Add("警告: BucketCount(%d) != Expected(%d)，可能存在并发问题", bucketCount, dirCount)
	}

	consts.SetLastScanTime(time.Now())

	// 扫描完成
	Sp.Complete()

	sse.BroadcastEvent("scan_complete", map[string]interface{}{
		"dirCount":  dirCount,
		"fileCount": SearchEngine.GetTotalCount(),
	})
	sse.BroadcastEvent("index_health", map[string]interface{}{
		"bucketCount":  bucketCount,
		"indexNumber":  indexNumber,
		"totalCount":   SearchEngine.GetTotalCount(),
		"lastScanTime": consts.GetLastScanTime(),
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
	for i := 0; i < dirSize; i++ {
		go func(dir string) {
			defer wg.Done()
			defer utils.RecoverPanic()
			s.goWalkWithResult(dir, types, resultChan)

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

// Walks 并发扫描多文件夹并返回所有文件（扫描 + 重建索引）
func (s *searchService) Walks(baseDir []string, types []string) []model.FileItem {
	dirSize := len(baseDir)

	if dirSize == 0 {
		return nil
	}

	buckets := s.ScanDirs(baseDir, types)

	var result []model.FileItem
	for _, b := range buckets {
		for _, m := range b.FileLib {
			result = append(result, m)
		}
	}

	LogMem.Add("Walks: 准备重建索引")
	SearchEngine.rebuildWithBuckets(buckets)
	LogMem.Add("Walks: 索引重建完成")

	return result
}

// goWalkWithResult 协程方法扫描单个文件夹并返回结果
func (s *searchService) goWalkWithResult(baseDir string, types []string, resultChan chan<- scanResult) {
	defer func() {
		Sp.IncrementCompletedDirs()
	}()

	// 更新当前正在扫描的目录
	Sp.SetCurrentDir(baseDir)

	LogMem.Add("goWalkWithResult: 开始扫描目录 %s", baseDir)
	start := time.Now()
	files, size := s.WalkInner(baseDir, types, true, baseDir)

	LogMem.Add("goWalkWithResult: 扫描完成 %s, 发现 %d 个文件", baseDir, len(files))
	// 更新已扫描文件计数
	Sp.AddScannedFiles(int64(len(files)))

	bucket := newInstanceWithFiles(baseDir, files)

	ti := time.Since(start)
	thisTime := MenuSize{
		Name:    baseDir,
		Cnt:     ti.Milliseconds(),
		Size:    int64(len(files)),
		SizeStr: utils.GetSizeStr(size),
	}
	LogMem.Add("扫描目录:[%s] 耗时:[%d] 大小:[%s],剩余目录数:%d", baseDir, ti.Milliseconds(), utils.GetSizeStr(size), consts.IndexNumber.Load())
	AddFolderTime(thisTime)
	sse.BroadcastEvent("scan_one_done", map[string]interface{}{
		"dir":     baseDir,
		"time":    ti.Milliseconds(),
		"size":    int64(len(files)),
		"sizeStr": utils.GetSizeStr(size),
		"remain":  consts.IndexNumber.Load(),
	})
	resultChan <- scanResult{dir: baseDir, bucket: bucket}
}

// Walk 遍历目录，获取指定类型文件列表（轻量版，不建索引）
func (s *searchService) Walk(dirPath string, types []string, deep bool) []model.FileItem {
	files, _ := s.WalkInner(dirPath, types, deep, dirPath)
	return files
}

// WalkInner 递归遍历目录获取文件列表
func (s *searchService) WalkInner(currentDir string, types []string, queryChild bool, basePath string) ([]model.FileItem, int64) {
	typeSet := utils.ToSet(types)

	dirStack := []stackItem{{path: currentDir, queryChild: queryChild, visited: false}}

	var allFiles []model.FileItem
	sizeMap := make(map[string]int64)
	sizeMap[currentDir] = 0

	for len(dirStack) > 0 {
		current := dirStack[len(dirStack)-1]
		dirStack = dirStack[:len(dirStack)-1]
		currentPath := current.path
		currentQueryChild := current.queryChild
		visited := current.visited

		if !visited {
			files, err := os.ReadDir(currentPath)
			if err != nil {
				utils.InfoFormat("读取目录失败: %s, 错误: %v", currentPath, err)
				continue
			}

			dirStack = append(dirStack, stackItem{path: currentPath, queryChild: currentQueryChild, visited: true})

			for i := len(files) - 1; i >= 0; i-- {
				f := files[i]
				p := filepath.Join(currentPath, f.Name())

				if f.IsDir() && currentQueryChild {
					dirStack = append(dirStack, stackItem{path: p, queryChild: currentQueryChild, visited: false})
					sizeMap[p] = 0
				} else if !f.IsDir() {
					name := f.Name()
					suffix := utils.GetSuffix(name)

					info, err := f.Info()
					if err != nil {
						utils.InfoFormat("获取文件信息失败: %s, 错误: %v", p, err)
						continue
					}
					if utils.HasItemSet(typeSet, suffix) {
						movie := model.EasyFile(currentPath, p, name, suffix,
							info.Size(), info.ModTime(), basePath)
						SetMovieNode(&movie)
						allFiles = append(allFiles, movie)
					}
					sizeMap[currentPath] += info.Size()
				}
			}

			if len(files) == 0 {
				if emptyFile, err := os.Stat(currentPath); err == nil {
					yesterday := time.Now().AddDate(0, 0, -1)
					modDate := time.Date(emptyFile.ModTime().Year(), emptyFile.ModTime().Month(), emptyFile.ModTime().Day(), 0, 0, 0, 0, time.Local)
					yesterdayDate := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.Local)
					if modDate.Equal(yesterdayDate) {
						if utils.IndexOf(GetOSSetting().Dirs, currentPath) < 0 {
							if err := os.RemoveAll(currentPath); err != nil {
								utils.InfoFormat("删除空目录失败: %s, 错误: %v", currentPath, err)
							}
						}
					}
				}
			}
		} else {
			currentSize := sizeMap[currentPath]
			if currentSize <= 20000000 && utils.IndexOf(GetOSSetting().Dirs, currentPath) < 0 {
				AppendSmallDir(NewMenuSizeFold(currentPath, currentSize, true))
			}

			if currentPath != currentDir {
				parentPath := filepath.Dir(currentPath)
				sizeMap[parentPath] += currentSize
			}
		}
	}

	return allFiles, sizeMap[currentDir]
}
