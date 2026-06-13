package service

import (
	"context"
	"runtime/debug"
	"search-gin/internal/model"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type repeatModel struct {
	Code  string
	Files model.Movie
	Count int
}

// searchSnapshot 搜索引擎的快照（不可变，通过 atomic.Value 原子替换）
type searchSnapshot struct {
	buckets     map[string]*bucketFile // baseDir → bucket
	bucketCount int32
	totalSize   int64
	totalCount  int
	repeatFiles []model.Movie
	actressMap  map[string]model.Actress

	// 预聚合的菜单数据（写入 consts.* 前暂存）
	typeMenu      map[string]consts.MenuSize
	tagMenu       map[string]consts.MenuSize
	seriesCount   map[string]consts.MenuSize
}

// searchEngineCore 搜索引擎：只保留快照指针 + 不变的辅助字段
type searchEngineCore struct {
	snapshot            atomic.Value     // *searchSnapshot
	KeywordHistoryCache *utils.LRUCache // 搜索结果缓存
	searchPool          *utils.GoroutinePool
	rebuildMu           sync.Mutex // 防止并发 rebuildWithBucket（ScanAll 与 taskQueue 可能重叠）
	// PageActress 空关键词结果缓存（减少重复排序）
	actressSizeCache  []model.Actress
	actressCountCache []model.Actress
}

// loadSnapshot 线程安全地获取当前快照；尚未初始化时返回空快照
func (se *searchEngineCore) loadSnapshot() *searchSnapshot {
	s := se.snapshot.Load()
	if s == nil {
		return &searchSnapshot{
			buckets:    make(map[string]*bucketFile),
			actressMap: make(map[string]model.Actress),
		}
	}
	return s.(*searchSnapshot)
}

// init 初始化 goroutine 池
func init() {
	SearchEngine.searchPool = utils.NewGoroutinePool(20)
}

// installSnapshot 原子替换搜索引擎快照，并同步全局菜单
func (se *searchEngineCore) installSnapshot(snap *searchSnapshot) {
	se.snapshot.Store(snap)
	se.KeywordHistoryCache.Clear()
	// 清空演员缓存（新快照的数据可能变化）
	se.actressSizeCache = nil
	se.actressCountCache = nil

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
}

// Reset 清空搜索引擎全部状态和缓存
func (se *searchEngineCore) Reset() {
	empty := &searchSnapshot{
		buckets:    make(map[string]*bucketFile),
		actressMap: make(map[string]model.Actress),
	}
	se.installSnapshot(empty)
}

// IsEmpty 检查是否有 bucket 数据
func (se *searchEngineCore) IsEmpty() bool {
	snap := se.loadSnapshot()
	return len(snap.buckets) == 0
}

// GetTotalCount 获取文件总数
func (se *searchEngineCore) GetTotalCount() int {
	return se.loadSnapshot().totalCount
}

// GetTotalSize 获取文件总大小
func (se *searchEngineCore) GetTotalSize() int64 {
	return se.loadSnapshot().totalSize
}

// BucketCount 返回 bucket 数量
func (se *searchEngineCore) BucketCount() int32 {
	return se.loadSnapshot().bucketCount
}

// rebuildWithBuckets 批量重建：一次性替换所有 bucket，O(N) 聚合
func (se *searchEngineCore) rebuildWithBuckets(entries map[string]*bucketFile) {
	defer func() {
		if r := recover(); r != nil {
			AddLogMemory("rebuildWithBuckets 异常: %v", r)
			AddLogMemory("堆栈: %s", string(debug.Stack()))
		}
	}()

	se.rebuildMu.Lock()
	defer se.rebuildMu.Unlock()

	AddLogMemory("rebuildWithBuckets: 开始批量重建, %d 个目录", len(entries))
	start := time.Now()

	newSnap := buildSnapshotFromBuckets(entries)
	se.installSnapshot(newSnap)

	ti := time.Since(start)
	AddLogMemory("rebuildWithBuckets: 完成, 耗时 %dms, 文件数 %d", ti.Milliseconds(), newSnap.totalCount)
}

// rebuildWithBucket 影子索引核心：用指定目录的新 bucket 构造一份新快照，原子替换
// 这是 executeTask 的唯一入口，替代了旧的 setBucket + buildIndexEngin
func (se *searchEngineCore) rebuildWithBucket(baseDir string, newBucket *bucketFile) {
	defer func() {
		if r := recover(); r != nil {
			AddLogMemory("rebuildWithBucket 异常: %v", r)
			AddLogMemory("堆栈: %s", string(debug.Stack()))
		}
	}()

	se.rebuildMu.Lock()
	defer se.rebuildMu.Unlock()

	AddLogMemory("rebuildWithBucket: 开始处理目录 %s", baseDir)
	start := time.Now()

	// 1. 读取当前配置的允许目录集合
	dirs := consts.GetOSSetting().Dirs
	dirSet := make(map[string]struct{}, len(dirs))
	for _, d := range dirs {
		dirSet[d] = struct{}{}
	}

	// 2. 复制旧 bucket（跳过不在配置中的 + 即将被替换的目录）
	old := se.loadSnapshot()
	newBuckets := make(map[string]*bucketFile, len(old.buckets)+1)
	for k, v := range old.buckets {
		if k == baseDir {
			continue // 即将替换
		}
		if _, ok := dirSet[k]; !ok {
			continue // 已不在配置中
		}
		newBuckets[k] = v
	}
	// 放入新 bucket（未满配置目录数也接受——可能只有部分目录扫描完）
	if newBucket != nil && !newBucket.isEmpty() {
		newBuckets[baseDir] = newBucket
	}

	AddLogMemory("rebuildWithBucket: bucket 数量 %d -> %d", len(old.buckets), len(newBuckets))

	// 3. 在新数据上跑聚合，构造快照
	newSnap := buildSnapshotFromBuckets(newBuckets)

	// 4. 原子替换
	se.installSnapshot(newSnap)

	ti := time.Since(start)
	AddLogMemory("rebuildWithBucket: 完成, 耗时 %dms, 文件数 %d", ti.Milliseconds(), newSnap.totalCount)
}

// buildSnapshotFromBuckets 遍历所有 bucket，构造完整的 searchSnapshot
func buildSnapshotFromBuckets(buckets map[string]*bucketFile) *searchSnapshot {
	snap := &searchSnapshot{
		buckets:    make(map[string]*bucketFile, len(buckets)),
		actressMap: make(map[string]model.Actress),
		typeMenu:   make(map[string]consts.MenuSize),
		tagMenu:    make(map[string]consts.MenuSize),
		seriesCount: make(map[string]consts.MenuSize),
	}

	// 复制 bucket 引用（不需要深拷贝，bucket 本身是稳定的）
	for k, v := range buckets {
		snap.buckets[k] = v
	}

	sizeRepeats := make(map[int64]repeatModel, 1000)
	codeRepeats := make(map[string]repeatModel, 1000)
	fileRepeats := make(map[string]model.Movie, 2000)

	for _, bucket := range snap.buckets {
		if bucket.isEmpty() {
			continue
		}
		bucket.mu.RLock()

		snap.totalSize += bucket.TotalSize
		snap.totalCount += bucket.TotalCount
		snap.bucketCount++

		for _, movie := range bucket.FileLib {
			// 演员聚合
			if len(movie.Actress) > 0 {
				if cur, ok := snap.actressMap[movie.Actress]; ok {
					cur.PlusCnt()
					cur.PlusSize(movie.Size)
					cur.AddImage(movie.Png)
					cur.AddImage(movie.Jpg)
					snap.actressMap[movie.Actress] = cur
				} else {
					snap.actressMap[movie.Actress] = model.NewActress(movie.Actress, movie.Jpg, movie.Size)
				}
			}

			// 重复检测（大小 + 番号）
			if !movie.IsNull() {
				pkSize := movie.Size
				rs, ok := sizeRepeats[pkSize]
				if ok {
					rs.Count++
					fileRepeats[rs.Files.Path] = rs.Files
					fileRepeats[movie.Path] = movie
					sizeRepeats[pkSize] = rs
				} else {
					sizeRepeats[pkSize] = repeatModel{Code: movie.Code, Files: movie, Count: 1}
				}

				pkCode := strings.ReplaceAll(movie.Code, "-", "")
				pkCode = strings.ReplaceAll(pkCode, "_", "")
				rc, ok := codeRepeats[pkCode]
				if ok {
					rc.Count++
					fileRepeats[rc.Files.Path] = rc.Files
					fileRepeats[movie.Path] = movie
					codeRepeats[pkCode] = rc
				} else {
					codeRepeats[pkCode] = repeatModel{Code: movie.Code, Files: movie, Count: 1}
				}
			}

			// 类型/标签/系列菜单
			mt := movie.MovieType
			if mt == "" {
				mt = "无"
			}
			if v, ok := snap.typeMenu[mt]; ok {
				snap.typeMenu[mt] = v.Plus(movie.Size)
			} else {
				snap.typeMenu[mt] = consts.MenuSize{Name: mt, Cnt: 1, Size: movie.Size}
			}
			if v, ok := snap.typeMenu["全部"]; ok {
				snap.typeMenu["全部"] = v.Plus(movie.Size)
			} else {
				snap.typeMenu["全部"] = consts.MenuSize{Name: "全部", Cnt: 1, Size: movie.Size}
			}

			if len(movie.Tags) > 0 {
				for i := range movie.Tags {
					if v, ok := snap.tagMenu[movie.Tags[i]]; ok {
						snap.tagMenu[movie.Tags[i]] = v.Plus(movie.Size)
					} else {
						snap.tagMenu[movie.Tags[i]] = consts.MenuSize{Name: movie.Tags[i], Cnt: 1, Size: movie.Size, IsDir: true}
					}
				}
			}
			if len(movie.Studio) > 0 {
				if v, ok := snap.seriesCount[movie.Studio]; ok {
					snap.seriesCount[movie.Studio] = v.Plus(movie.Size)
				} else {
					snap.seriesCount[movie.Studio] = consts.MenuSize{Name: movie.Studio, Cnt: 1, Size: movie.Size, IsDir: true}
				}
			}
		}

		bucket.mu.RUnlock()
	}

	// 构建重复文件列表
	repeatSearch := make([]model.Movie, 0, len(fileRepeats))
	for _, m := range fileRepeats {
		repeatSearch = append(repeatSearch, m)
	}
	sort.Slice(repeatSearch, func(i, j int) bool {
		return repeatSearch[i].Size > repeatSearch[j].Size
	})
	snap.repeatFiles = repeatSearch

	return snap
}

// PageActress 演员搜索
func (se *searchEngineCore) PageActress(searchParam model.SearchParam) model.PageActressResultWrapper {
	snap := se.loadSnapshot()

	// 空关键词：优先走缓存
	if searchParam.Keyword == "" {
		switch searchParam.SortField {
		case "Size":
			if se.actressSizeCache != nil {
				resultWrapper := model.PageActressResultWrapper{}
				list, size := model.GetActressPageOfFiles(se.actressSizeCache, searchParam.Page, searchParam.PageSize)
				resultWrapper.FileList = list
				resultWrapper.Size = size
				resultWrapper.ResultCount = len(list)
				return resultWrapper
			}
		case "Cnt":
			if se.actressCountCache != nil {
				resultWrapper := model.PageActressResultWrapper{}
				list, size := model.GetActressPageOfFiles(se.actressCountCache, searchParam.Page, searchParam.PageSize)
				resultWrapper.FileList = list
				resultWrapper.Size = size
				resultWrapper.ResultCount = len(list)
				return resultWrapper
			}
		}
	}

	result := make([]model.Actress, 0, len(snap.actressMap))
	for _, actress := range snap.actressMap {
		if searchParam.Keyword == "" || strings.Contains(actress.Name, searchParam.Keyword) {
			result = append(result, actress)
		}
	}

	switch searchParam.SortField {
	case "Size":
		sort.Slice(result, func(i, j int) bool {
			return result[i].Size > result[j].Size
		})
		if searchParam.Keyword == "" {
			se.actressSizeCache = result
		}
	case "Cnt":
		sort.Slice(result, func(i, j int) bool {
			return result[i].Cnt > result[j].Cnt
		})
		if searchParam.Keyword == "" {
			se.actressCountCache = result
		}
	}

	resultWrapper := model.PageActressResultWrapper{}
	list, size := model.GetActressPageOfFiles(result, searchParam.Page, searchParam.PageSize)
	resultWrapper.FileList = list
	resultWrapper.Size = size
	resultWrapper.ResultCount = len(list)
	return resultWrapper
}

// returnRepeatSearch 返回重复文件
func (se *searchEngineCore) returnRepeatSearch() model.PageResultWrapper {
	snap := se.loadSnapshot()
	resultWrapper := model.NewPageWrapper()
	if len(snap.repeatFiles) > 0 {
		resultWrapper.FileList = make([]model.Movie, len(snap.repeatFiles))
		copy(resultWrapper.FileList, snap.repeatFiles)
	}
	resultWrapper.ResultCount = len(snap.repeatFiles)
	resultWrapper.LibCount = len(snap.repeatFiles)
	resultWrapper.SearchCount = len(snap.repeatFiles)
	return resultWrapper
}

// PageAsync 异步分页搜索
func (se *searchEngineCore) PageAsync(searchParam model.SearchParam) model.PageResultWrapper {
	if searchParam.OnlyRepeat {
		return se.returnRepeatSearch()
	}
	resultWrapper := model.NewPageWrapper()

	cacheKey := searchParam.UniWords()
	matchValue, ok := se.KeywordHistoryCache.Get(cacheKey)
	if ok {
		resultWrapper = matchValue.(model.PageResultWrapper)
		resultWrapper.FileList, resultWrapper.ResultSize = model.GetPageOfFiles(
			resultWrapper.FileList, searchParam.Page, searchParam.PageSize)
		return resultWrapper
	}

	snap := se.loadSnapshot()
	bucketCount := len(snap.buckets)
	if bucketCount <= 0 {
		AddLogMemory("警告: bucketCount=0, 跳过搜索")
		resultWrapper.FileList = []model.Movie{}
		return resultWrapper
	}

	poolSize := se.searchPool.Cap()
	if bucketCount > 0 && bucketCount < poolSize {
		poolSize = bucketCount
	}

	resultWrapper.ResultCount = searchParam.PageSize
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resultChan := make(chan model.SearchResultWrapper, bucketCount*2)

	for _, bucket := range snap.buckets {
		if bucket.isEmpty() {
			continue
		}
		b := bucket // 闭包捕获
		se.searchPool.Submit(func() {
			defer func() {
				if r := recover(); r != nil {
					AddLogMemory("搜索 bucket 异常: %v", r)
				}
			}()
			wrapper := b.searchBucket(searchParam)
			if wrapper.IsNotEmpty() {
				select {
				case resultChan <- wrapper:
				case <-ctx.Done():
				}
			}
		})
	}

	go func() {
		se.searchPool.Wait()
		close(resultChan)
	}()

loop:
	for {
		select {
		case data, ok := <-resultChan:
			if !ok {
				break loop
			}
			resultWrapper.FileList = append(resultWrapper.FileList, data.FileList...)
			resultWrapper.SearchCount += len(data.FileList)
			resultWrapper.SearchSize += data.Size
		case <-ctx.Done():
			AddLogMemory("搜索超时，部分结果可能未返回")
			se.searchPool.Wait()
			for data := range resultChan {
				resultWrapper.FileList = append(resultWrapper.FileList, data.FileList...)
				resultWrapper.SearchCount += len(data.FileList)
				resultWrapper.SearchSize += data.Size
			}
			break loop
		}
	}

	model.SortMoviesUtils(resultWrapper.FileList, searchParam.SortField, searchParam.SortType)
	se.addHistory(cacheKey, resultWrapper)

	resultWrapper.FileList, resultWrapper.ResultSize = model.GetPageOfFiles(
		resultWrapper.FileList, searchParam.Page, searchParam.PageSize)
	return resultWrapper
}

func (se *searchEngineCore) addHistory(uniqueWords string, resultWrapper model.PageResultWrapper) {
	se.KeywordHistoryCache.Set(uniqueWords, resultWrapper)
}

// FindById 查找文件——先查当前快照，未找到返回空
func (se *searchEngineCore) FindById(id string) model.Movie {
	snap := se.loadSnapshot()
	for _, bucket := range snap.buckets {
		if bucket.isEmpty() {
			continue
		}
		result := bucket.get(id)
		if !result.IsNull() {
			return result
		}
	}
	return model.Movie{}
}

// FindActressByName 按名称查找演员
func (se *searchEngineCore) FindActressByName(name string) model.Actress {
	snap := se.loadSnapshot()
	if a, ok := snap.actressMap[name]; ok {
		return a
	}
	return model.Actress{}
}

// GetActorCount 获取演员总数
func (se *searchEngineCore) GetActorCount() int {
	return len(se.loadSnapshot().actressMap)
}
