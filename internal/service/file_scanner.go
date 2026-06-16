package service

import (
	"os"
	"path/filepath"
	"search-gin/internal/model"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"sync"
	"sync/atomic"
	"time"
)

// scanResult 目录扫描结果
type scanResult struct {
	dir    string
	bucket *bucketFile
}

// ScanAll 全局扫描
func (fs *fileService) ScanAll() int {
	setting := consts.GetOSSetting()
	dirCount := len(setting.Dirs)
	dirList := make([]string, dirCount)
	copy(dirList, setting.Dirs)
	AddLogMemory("Plan to ScanAll dirTotal: %d, dirList: %v", dirCount, dirList)
	if !FullScanInProgress.CompareAndSwap(0, 1) {
		AddLogMemory("全量扫描正在进行中")
		return dirCount
	}
	defer FullScanInProgress.Store(0)

	// 初始化扫描进度
	consts.SpMu.Lock()
	consts.Sp = consts.ScanProgress{
		Phase:            "scanning",
		TotalDirs:        dirCount,
		CompletedDirs:    0,
		CurrentDir:       "",
		ScannedFiles:     0,
		TotalBuckets:     dirCount,
		ProcessedBuckets: 0,
		CurrentPhase:     "正在扫描目录...",
	}
	consts.SpMu.Unlock()

	consts.TypeMenu.Clear()
	consts.SeriesCount.Clear()
	consts.TagMenu.Clear()
	consts.ClearSmallDir()

	queryTypes := make([]string, 0)
	queryTypes = utils.ExtendsItems(queryTypes, setting.VideoTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.DocsTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.ImageTypes)
	consts.InitFolderTime()
	fs.Walks(dirList, queryTypes)

	// 一致性检查：验证 bucket 数量和目录数量
	bucketCount := SearchEngine.BucketCount()
	indexNumber := atomic.LoadInt32(&consts.IndexNumber)
	AddLogMemory("ScanAll 一致性检查: BucketCount=%d, IndexNumber=%d, Expected=%d", bucketCount, indexNumber, dirCount)
	if bucketCount != int32(dirCount) {
		AddLogMemory("警告: BucketCount(%d) != Expected(%d)，可能存在并发问题", bucketCount, dirCount)
	}

	// 切换到索引构建阶段
	consts.SpMu.Lock()
	consts.Sp.Phase = "building"
	consts.Sp.CurrentPhase = "正在构建索引..."
	consts.SpMu.Unlock()

	consts.LastScanTime = time.Now()

	// 扫描完成
	consts.SpMu.Lock()
	consts.Sp.Phase = "done"
	consts.Sp.CompletedDirs = consts.Sp.TotalDirs
	consts.Sp.CurrentPhase = "扫描完成"
	consts.SpMu.Unlock()

	return dirCount
}

// ScanTarget 扫描指定文件夹
func (fs *fileService) ScanTarget(baseDir string) {
	scanQueue.AddTask(baseDir)
}

// Walks 并发扫描多文件夹并返回所有文件
func (fs *fileService) Walks(baseDir []string, types []string) []model.FileItem {
	var wg sync.WaitGroup
	var result []model.FileItem
	dirSize := len(baseDir)

	// 不提前 Reset，旧索引在扫描期间保持可用，避免正在播放的流媒体断连
	resultChan := make(chan scanResult, dirSize)

	wg.Add(dirSize)
	for i := 0; i < dirSize; i++ {
		go func(dir string) {
			defer wg.Done()
			defer utils.RecoverPanic()
			fs.goWalkWithResult(dir, types, resultChan)
		}(baseDir[i])
	}

	wg.Wait()
	close(resultChan)

	if dirSize == 0 {
		return result
	}

	// 收集所有扫描结果，批量重建索引
	buckets := make(map[string]*bucketFile, dirSize)
	for r := range resultChan {
		if r.bucket != nil && !r.bucket.isEmpty() {
			buckets[r.dir] = r.bucket
			for _, m := range r.bucket.FileLib {
				result = append(result, m)
			}
		}
	}

	AddLogMemory("Walks: 扫描完成, 共 %d 个目录, 准备重建索引", len(buckets))
	// rebuildWithBuckets 内部原子替换快照，无需提前 Reset，零窗口
	SearchEngine.rebuildWithBuckets(buckets)
	AddLogMemory("Walks: 索引重建完成")

	return result
}

// goWalkWithResult 协程方法扫描单个文件夹并返回结果
func (fs *fileService) goWalkWithResult(baseDir string, types []string, resultChan chan<- scanResult) {
	defer func() {
		consts.SpMu.Lock()
		consts.Sp.CompletedDirs++
		consts.SpMu.Unlock()
	}()

	// 更新当前正在扫描的目录
	consts.SpMu.Lock()
	consts.Sp.CurrentDir = baseDir
	consts.SpMu.Unlock()

	AddLogMemory("goWalkWithResult: 开始扫描目录 %s", baseDir)
	start := time.Now()
	files, size := fs.WalkInner(baseDir, types, true, baseDir)

	AddLogMemory("goWalkWithResult: 扫描完成 %s, 发现 %d 个文件", baseDir, len(files))
	// 更新已扫描文件计数
	consts.SpMu.Lock()
	consts.Sp.ScannedFiles += int64(len(files))
	consts.SpMu.Unlock()

	bucket := newInstanceWithFiles(baseDir, files)

	ti := time.Since(start)
	thisTime := consts.MenuSize{
		Name:    baseDir,
		Cnt:     ti.Milliseconds(),
		Size:    int64(len(files)),
		SizeStr: utils.GetSizeStr(size),
	}
	AddLogMemory("扫描目录:[%s] 耗时:[%d] 大小:[%s],剩余目录数:%d", baseDir, ti.Milliseconds(), utils.GetSizeStr(size), atomic.LoadInt32(&consts.IndexNumber))
	consts.AddFolderTime(thisTime)

	resultChan <- scanResult{dir: baseDir, bucket: bucket}
}

// Walk 遍历目录，获取指定类型文件列表（轻量版，不建索引）
func (fs *fileService) Walk(dirPath string, types []string, deep bool) []model.FileItem {
	files, _ := fs.WalkInner(dirPath, types, deep, dirPath)
	return files
}

// WalkInner 递归遍历目录获取文件列表
func (fs *fileService) WalkInner(currentDir string, types []string, queryChild bool, basePath string) ([]model.FileItem, int64) {
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
					if emptyFile.ModTime().Day() == yesterday.Day() &&
						emptyFile.ModTime().Month() == yesterday.Month() &&
						emptyFile.ModTime().Year() == yesterday.Year() {
						if utils.IndexOf(consts.GetOSSetting().Dirs, currentPath) < 0 {
							if err := os.RemoveAll(currentPath); err != nil {
								utils.InfoFormat("删除空目录失败: %s, 错误: %v", currentPath, err)
							}
						}
					}
				}
			}
		} else {
			currentSize := sizeMap[currentPath]
			if currentSize <= 20000000 && utils.IndexOf(consts.GetOSSetting().Dirs, currentPath) < 0 {
				consts.AppendSmallDir(consts.NewMenuSizeFold(currentPath, currentSize, true))
			}

			if currentPath != currentDir {
				parentPath := filepath.Dir(currentPath)
				sizeMap[parentPath] += currentSize
			}
		}
	}

	return allFiles, sizeMap[currentDir]
}
