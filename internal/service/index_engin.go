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

type searchEnginCore struct {
	SearchIndexMap                 sync.Map // map[string]*bucketFile
	dataMu                         sync.RWMutex
	RepeatSearch                   []model.Movie
	ActressSizeWrapperNullKeyword  []model.Actress
	ActressCountWrapperNullKeyword []model.Actress
	ActressMap                     map[string]model.Actress
	// TotalSizeMutex 保护 TotalSize/TotalCount 的并发读写（buildOthersData 在 goroutine 中修改）
	TotalSizeMutex      sync.RWMutex
	TotalSize           int64
	TotalCount          int
	KeywordHistoryCache *utils.LRUCache      // 替换 sync.Map 为 LRU 缓存
	searchPool          *utils.GoroutinePool // 全局 goroutine 池，避免每次搜索重复创建
	BucketCount         int32
}

// Reset 清空搜索引擎全部状态和缓存
func (se *searchEnginCore) Reset() {
	se.SearchIndexMap.Range(func(key, value interface{}) bool {
		se.SearchIndexMap.Delete(key)
		return true
	})
	se.KeywordHistoryCache.Clear()
	se.dataMu.Lock()
	se.RepeatSearch = nil
	se.ActressMap = nil
	se.ActressSizeWrapperNullKeyword = nil
	se.ActressCountWrapperNullKeyword = nil
	se.dataMu.Unlock()
	se.TotalSizeMutex.Lock()
	se.TotalSize = 0
	se.TotalCount = 0
	se.TotalSizeMutex.Unlock()
	atomic.StoreInt32(&se.BucketCount, 0)
}

func init() {
	SearchEngin.searchPool = utils.NewGoroutinePool(20) // 全局 goroutine 池，最大 20 并发
}

func (se *searchEnginCore) Init(baseDirs []string) {
	for _, dir := range baseDirs {
		_, ok := se.SearchIndexMap.Load(dir)
		if !ok {
			se.SearchIndexMap.Store(dir, newInstance(dir))
		}
	}
}

func (se *searchEnginCore) IsEmpty() bool {
	se.TotalSizeMutex.RLock()
	defer se.TotalSizeMutex.RUnlock()
	return se.TotalCount == 0
}

func (se *searchEnginCore) GetTotalCount() int {
	se.TotalSizeMutex.RLock()
	defer se.TotalSizeMutex.RUnlock()
	return se.TotalCount
}

func (se *searchEnginCore) GetTotalSize() int64 {
	se.TotalSizeMutex.RLock()
	defer se.TotalSizeMutex.RUnlock()
	return se.TotalSize
}

func (se *searchEnginCore) PageActress(searchParam model.SearchParam) model.PageActressResultWrapper {
	var result = []model.Actress{}
	if len((searchParam.Keyword)) == 0 {
		se.dataMu.RLock()
		if searchParam.SortField == "Size" && len(se.ActressSizeWrapperNullKeyword) > 0 {
			result = se.ActressSizeWrapperNullKeyword
		} else if searchParam.SortField == "Cnt" && len(se.ActressCountWrapperNullKeyword) > 0 {
			result = se.ActressCountWrapperNullKeyword
		}
		se.dataMu.RUnlock()
	}
	if len(result) == 0 {
		se.dataMu.RLock()
		result = model.SearchActressByKeyWord(se.ActressMap, searchParam.Keyword)
		se.dataMu.RUnlock()
		switch searchParam.SortField {
		case "Size":
			sort.Slice(result, func(i, j int) bool {
				return result[i].Size > result[j].Size
			})
			if len((searchParam.Keyword)) == 0 {
				se.dataMu.Lock()
				se.ActressSizeWrapperNullKeyword = result
				se.dataMu.Unlock()
			}

		case "Cnt":
			sort.Slice(result, func(i, j int) bool {
				return result[i].Cnt > result[j].Cnt
			})
			if len((searchParam.Keyword)) == 0 {
				se.dataMu.Lock()
				se.ActressCountWrapperNullKeyword = result
				se.dataMu.Unlock()
			}
		}
	}

	resultWrapper := model.PageActressResultWrapper{}
	list, size := model.GetActressPageOfFiles(result, searchParam.Page, searchParam.PageSize)
	resultWrapper.FileList = list
	resultWrapper.Size = size
	resultWrapper.ResultCount = len(list)
	return resultWrapper
}

func (se *searchEnginCore) returnRepeatSearch() model.PageResultWrapper {
	resultWrapper := model.NewPageWrapper()
	se.dataMu.RLock()
	if len(se.RepeatSearch) > 0 {
		resultWrapper.FileList = make([]model.Movie, len(se.RepeatSearch))
		copy(resultWrapper.FileList, se.RepeatSearch)
	}
	resultWrapper.ResultCount = len(se.RepeatSearch)
	resultWrapper.LibCount = len(se.RepeatSearch)
	resultWrapper.SearchCount = len(se.RepeatSearch)
	se.dataMu.RUnlock()
	return resultWrapper
}

func (se *searchEnginCore) PageAsync(searchParam model.SearchParam) model.PageResultWrapper {
	if searchParam.OnlyRepeat {
		return se.returnRepeatSearch()
	}
	resultWrapper := model.NewPageWrapper()

	// 使用缓存键包含排序信息
	cacheKey := searchParam.UniWords()
	matchValue, ok := se.KeywordHistoryCache.Get(cacheKey)
	if ok {
		resultWrapper = matchValue.(model.PageResultWrapper)
		// 对缓存结果重新分页
		resultWrapper.FileList, resultWrapper.ResultSize = model.GetPageOfFiles(
			resultWrapper.FileList, searchParam.Page, searchParam.PageSize)
		return resultWrapper
	}

	// 异步搜索优化
	// 动态计算并发数量
	bucketCount := int(atomic.LoadInt32(&se.BucketCount))

	// 断言检查：防止零值导致死锁
	if bucketCount <= 0 {
		AddLogMemory("警告: BucketCount=%d <= 0，可能存在竞态条件，跳过搜索", bucketCount)
		resultWrapper.FileList = []model.Movie{}
		return resultWrapper
	}
	AddLogMemory("PageAsync: 开始搜索, bucketCount=%d", bucketCount)

	// 根据 bucket 数量动态调整 goroutine 池大小（不超过全局池容量）
	poolSize := se.searchPool.Cap()
	if bucketCount > 0 && bucketCount < poolSize {
		poolSize = bucketCount
	}

	resultWrapper.ResultCount = searchParam.PageSize

	// 使用 context 控制超时和取消
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 预估结果大小，减少通道内存分配
	resultChan := make(chan model.SearchResultWrapper, bucketCount*2)

	// 并发搜索所有 bucket
	se.SearchIndexMap.Range(func(key, value interface{}) bool {
		index := value.(*bucketFile)
		if index.isEmpty() {
			return true
		}
		se.searchPool.Submit(func() {
			defer func() {
				if r := recover(); r != nil {
					AddLogMemory("搜索 bucket 发生异常：%v", r)
				}
			}()
			indexWrapper := index.searchBucket(searchParam)
			if indexWrapper.IsNotEmpty() {
				select {
				case resultChan <- indexWrapper:
				case <-ctx.Done():
					// context 已取消（超时或主 goroutine 已退出），不再发送
				}
			}
		})
		return true
	})

	// 等待所有搜索完成并关闭通道
	go func() {
		se.searchPool.Wait()
		close(resultChan)
	}()

	// 收集搜索结果
loop:
	for {
		select {
		case data, ok := <-resultChan:
			if !ok {
				// 通道已关闭，搜索完成
				break loop
			}
			// 直接追加到结果列表，减少内存分配
			resultWrapper.FileList = append(resultWrapper.FileList, data.FileList...)
			resultWrapper.SearchCount += len(data.FileList)
			resultWrapper.SearchSize += data.Size
		case <-ctx.Done():
			// 搜索超时，取消所有进行中的发送，等待剩余结果
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

	// 对结果进行排序
	model.SortMoviesUtils(resultWrapper.FileList, searchParam.SortField, searchParam.SortType)

	// 直接缓存完整结果集
	se.addHistory(cacheKey, resultWrapper)

	// 进行分页
	resultWrapper.FileList, resultWrapper.ResultSize = model.GetPageOfFiles(
		resultWrapper.FileList, searchParam.Page, searchParam.PageSize)
	return resultWrapper
}

func (se *searchEnginCore) addHistory(uniqueWords string, resultWrapper model.PageResultWrapper) {
	// 使用LRU缓存自动管理缓存大小
	se.KeywordHistoryCache.Set(uniqueWords, resultWrapper)
}

func (se *searchEnginCore) FindById(id string) model.Movie {
	var result = model.Movie{}
	se.SearchIndexMap.Range(func(key, value any) bool {
		index := value.(*bucketFile)
		if index.isEmpty() {
			return true
		}
		result = index.get(id)
		return result.IsNull()
	})
	return result
}

func (se *searchEnginCore) FindActressByName(id string) model.Actress {
	se.dataMu.RLock()
	act, ok := se.ActressMap[id]
	se.dataMu.RUnlock()
	if ok {
		return act
	}
	return model.Actress{}
}

func (se *searchEnginCore) GetActorCount() int {
	se.dataMu.RLock()
	defer se.dataMu.RUnlock()
	return len(se.ActressMap)
}

func (se *searchEnginCore) setBucket(baseDir string, bucket *bucketFile) {
	before := atomic.LoadInt32(&se.BucketCount)
	se.SearchIndexMap.Store(baseDir, bucket)
	after := atomic.AddInt32(&se.BucketCount, 1)
	AddLogMemory("setBucket: %s, before=%d, after=%d", baseDir, before, after)
}

func (se *searchEnginCore) buildIndexEngin() {
	defer func() {
		if r := recover(); r != nil {
			AddLogMemory("构建索引发生异常: %v", r)
			AddLogMemory("堆栈信息: %s", string(debug.Stack()))
		}
	}()

	start := time.Now()
	AddLogMemory("buildIndexEngin: 开始构建索引")
	se.KeywordHistoryCache.Clear()

	// 清理不在当前配置中的 bucket
	dirs := consts.GetOSSetting().Dirs
	if len(dirs) > 0 {
		dirSet := make(map[string]struct{}, len(dirs))
		for _, d := range dirs {
			dirSet[d] = struct{}{}
		}
		se.SearchIndexMap.Range(func(key, value any) bool {
			if _, ok := dirSet[key.(string)]; !ok {
				se.SearchIndexMap.Delete(key)
			}
			return true
		})
	}

	// 重置总计信息
	se.TotalSizeMutex.Lock()
	se.TotalCount = 0
	se.TotalSize = 0
	se.TotalSizeMutex.Unlock()

	// 使用局部变量聚合，避免在 Range 中频繁写入全局 sync.Map
	actressMap := make(map[string]model.Actress)
	localTypeMenu := make(map[string]consts.MenuSize)
	localTagMenu := make(map[string]consts.MenuSize)
	localSeriesCount := make(map[string]consts.MenuSize)
	sizeRepeats := make(map[int64]repeatModel, 1000)
	codeRepeats := make(map[string]repeatModel, 1000)
	fileRepeats := make(map[string]model.Movie, 2000)

	var bucketCount int32
	var localTotalCount int
	var localTotalSize int64

	AddLogMemory("buildIndexEngin: 开始遍历 SearchIndexMap")

	// 预统计 bucket 总数用于进度追踪
	var totalBuckets int32
	se.SearchIndexMap.Range(func(key, value any) bool {
		if !value.(*bucketFile).isEmpty() {
			totalBuckets++
		}
		return true
	})
	consts.SpMu.Lock()
	consts.Sp.TotalBuckets = int(totalBuckets)
	consts.Sp.ProcessedBuckets = 0
	consts.SpMu.Unlock()

	se.SearchIndexMap.Range(func(key, value any) bool {
		index := value.(*bucketFile)
		if index.isEmpty() {
			AddLogMemory("buildIndexEngin: bucket 为空，跳过")
			return true
		}
		index.mu.RLock()

		// 1. 总计信息（聚合到局部变量，避免并发写入）
		localTotalSize += index.TotalSize
		localTotalCount += index.TotalCount
		bucketCount++
		AddLogMemory("buildIndexEngin: 处理 bucket %s, TotalCount=%d, TotalSize=%d", key.(string), index.TotalCount, index.TotalSize)

		// 2. 遍历文件中逐条处理
		for _, movie := range index.FileLib {
			// ---- 演员数据 ----
			if len(movie.Actress) > 0 {
				curActress, ok := actressMap[movie.Actress]
				if ok {
					curActress.PlusCnt()
					curActress.PlusSize(movie.Size)
					curActress.AddImage(movie.Png)
					curActress.AddImage(movie.Jpg)
					actressMap[movie.Actress] = curActress
				} else {
					actressMap[movie.Actress] = model.NewActress(movie.Actress, movie.Jpg, movie.Size)
				}
			}

			// ---- 重复检测 ----
			if !movie.IsNull() {
				pkSize := movie.Size
				repeatSize, ok := sizeRepeats[pkSize]
				if ok {
					repeatSize.Count = repeatSize.Count + 1
					fileRepeats[repeatSize.Files.Path] = repeatSize.Files
					fileRepeats[movie.Path] = movie
					sizeRepeats[pkSize] = repeatSize
				} else {
					sizeRepeats[pkSize] = repeatModel{
						Code:  movie.Code,
						Files: movie,
						Count: 1,
					}
				}

				pkCode := movie.Code
				pkCode = strings.ReplaceAll(pkCode, "-", "")
				pkCode = strings.ReplaceAll(pkCode, "_", "")
				repeatCode, ok := codeRepeats[pkCode]
				if ok {
					repeatCode.Count = repeatCode.Count + 1
					fileRepeats[repeatCode.Files.Path] = repeatCode.Files
					fileRepeats[movie.Path] = movie
					codeRepeats[pkCode] = repeatCode
				} else {
					codeRepeats[pkCode] = repeatModel{
						Code:  movie.Code,
						Files: movie,
						Count: 1,
					}
				}
			}

			// ---- 类型/标签/系列菜单 ----
			mt := movie.MovieType
			if mt == "" {
				mt = "无"
			}
			if v, ok := localTypeMenu[mt]; ok {
				localTypeMenu[mt] = v.Plus(movie.Size)
			} else {
				localTypeMenu[mt] = consts.MenuSize{Name: mt, Cnt: 1, Size: movie.Size}
			}
			if v, ok := localTypeMenu["全部"]; ok {
				localTypeMenu["全部"] = v.Plus(movie.Size)
			} else {
				localTypeMenu["全部"] = consts.MenuSize{Name: "全部", Cnt: 1, Size: movie.Size}
			}

			if len(movie.Tags) > 0 {
				for i := range movie.Tags {
					if v, ok := localTagMenu[movie.Tags[i]]; ok {
						localTagMenu[movie.Tags[i]] = v.Plus(movie.Size)
					} else {
						localTagMenu[movie.Tags[i]] = consts.MenuSize{Name: movie.Tags[i], Cnt: 1, Size: movie.Size, IsDir: true}
					}
				}
			}
			if len(movie.Studio) > 0 {
				if v, ok := localSeriesCount[movie.Studio]; ok {
					localSeriesCount[movie.Studio] = v.Plus(movie.Size)
				} else {
					localSeriesCount[movie.Studio] = consts.MenuSize{Name: movie.Studio, Cnt: 1, Size: movie.Size, IsDir: true}
				}
			}
		}

		index.mu.RUnlock()
		// 更新索引构建进度
		consts.SpMu.Lock()
		consts.Sp.ProcessedBuckets++
		consts.SpMu.Unlock()
		return true
	})

	atomic.StoreInt32(&se.BucketCount, bucketCount)
	se.TotalSizeMutex.Lock()
	se.TotalSize = localTotalSize
	se.TotalCount = localTotalCount
	se.TotalSizeMutex.Unlock()
	AddLogMemory("buildIndexEngin: 遍历完成, bucketCount=%d, TotalCount=%d", bucketCount, localTotalCount)

	// 批量写入全局 sync.Map
	AddLogMemory("buildIndexEngin: 开始写入全局菜单数据")
	consts.TypeMenu.Clear()
	for k, v := range localTypeMenu {
		consts.TypeMenu.Store(k, v)
	}
	consts.TagMenu.Clear()
	for k, v := range localTagMenu {
		consts.TagMenu.Store(k, v)
	}
	consts.SeriesCount.Clear()
	for k, v := range localSeriesCount {
		consts.SeriesCount.Store(k, v)
	}

	// 构建重复结果（局部变量）
	repeatSearch := make([]model.Movie, 0, len(fileRepeats))
	for _, m := range fileRepeats {
		repeatSearch = append(repeatSearch, m)
	}
	sort.Slice(repeatSearch, func(i, j int) bool {
		return repeatSearch[i].Size > repeatSearch[j].Size
	})

	// 原子性：一次锁定 swap ActressMap、RepeatSearch、缓存
	se.dataMu.Lock()
	se.ActressMap = actressMap
	se.RepeatSearch = repeatSearch
	se.ActressSizeWrapperNullKeyword = nil
	se.ActressCountWrapperNullKeyword = nil
	se.dataMu.Unlock()
	sizeRepeats = nil
	codeRepeats = nil
	fileRepeats = nil

	ti := time.Since(start)
	AddLogMemory("buildIndexEngin (single-pass) completed, time:%dms, files:%d, repeats:%d", ti.Milliseconds(), localTotalCount, len(se.RepeatSearch))
}
