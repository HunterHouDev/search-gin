package service

import (
	"search-gin/pkg/consts"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
)

type repeatModel struct {
	Code  string
	Files model.Movie
	Count int
}

type searchEnginCore struct {
	LastSortField                  string
	LastSortType                   string
	SearchIndexMap                 sync.Map // map[string]*bucketFile
	RepeatSearch                   []model.Movie
	ActressSizeWrapperNullKeyword  []model.Actress
	ActressCountWrapperNullKeyword []model.Actress
	ActressMap                     map[string]model.Actress
	// TotalSizeMutex 保护 TotalSize/TotalCount 的并发读写（buildOthersData 在 goroutine 中修改）
	TotalSizeMutex   sync.RWMutex
	TotalSize        int64
	TotalCount       int
	KeywordHistoryCache *utils.LRUCache // 替换 sync.Map 为 LRU 缓存
}

func (se *searchEnginCore) Reset() {
	se.SearchIndexMap.Range(func(key, value interface{}) bool {
		se.SearchIndexMap.Delete(key)
		return true
	})
	// 清空所有状态
	se.LastSortField = ""
	se.LastSortType = ""
	se.RepeatSearch = nil
	se.ActressMap = nil
	se.KeywordHistoryCache.Clear()
	se.TotalSize = 0
	se.TotalCount = 0
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
	return se.TotalCount == 0
}

func (se *searchEnginCore) PageActress(searchParam model.SearchParam) model.PageActressResultWrapper {
	var result = []model.Actress{}
	if len((searchParam.Keyword)) == 0 {
		// 如果没有关键字，考虑缓存读取
		if searchParam.SortField == "Size" && len(se.ActressSizeWrapperNullKeyword) > 0 {
			result = se.ActressSizeWrapperNullKeyword
		} else if searchParam.SortField == "Cnt" && len(se.ActressCountWrapperNullKeyword) > 0 {
			result = se.ActressCountWrapperNullKeyword
		}

	}
	if len(result) == 0 {
		result = model.SearchActressByKeyWord(se.ActressMap, searchParam.Keyword)
		if searchParam.SortField == "Size" {
			sort.Slice(result, func(i, j int) bool {
				return result[i].Size > result[j].Size
			})
			if len((searchParam.Keyword)) == 0 {
				se.ActressSizeWrapperNullKeyword = result
			}

		} else if searchParam.SortField == "Cnt" {
			sort.Slice(result, func(i, j int) bool {
				return result[i].Cnt > result[j].Cnt
			})
			if len((searchParam.Keyword)) == 0 {
				se.ActressCountWrapperNullKeyword = result
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
	// 拷贝切片，避免与缓存共享底层数组
	if len(se.RepeatSearch) > 0 {
		resultWrapper.FileList = make([]model.Movie, len(se.RepeatSearch))
		copy(resultWrapper.FileList, se.RepeatSearch)
	}
	resultWrapper.ResultCount = len(se.RepeatSearch)
	resultWrapper.LibCount = len(se.RepeatSearch)
	resultWrapper.SearchCount = len(se.RepeatSearch)
	return resultWrapper
}

func (se *searchEnginCore) Page(searchParam model.SearchParam) model.PageResultWrapper {
	if searchParam.OnlyRepeat {
		return se.returnRepeatSearch()
	}
	resultWrapper := model.NewPageWrapper()

	// 使用缓存键包含排序信息，避免排序不一致问题
	cacheKey := searchParam.UniWords()
	matchValue, ok := se.KeywordHistoryCache.Get(cacheKey)
	if ok {
		resultWrapper = matchValue.(model.PageResultWrapper)
		// 对缓存结果重新分页，确保页面大小正确
		resultWrapper.FileList, resultWrapper.ResultSize = model.GetPageOfFiles(
			resultWrapper.FileList, searchParam.Page, searchParam.PageSize)
		return resultWrapper
	}

	resultWrapper.ResultCount = searchParam.PageSize
	// 优化遍历性能，避免不必要的搜索
	se.SearchIndexMap.Range(func(key, value interface{}) bool {
		index := value.(*bucketFile)
		if index.isEmpty() {
		 return true
		}
		indexWrapper := index.searchBucket(searchParam)
		if !indexWrapper.IsNotEmpty() {
			return true
		}
		// 直接追加到结果列表，减少内存分配
		resultWrapper.FileList = append(resultWrapper.FileList, indexWrapper.FileList...)
		resultWrapper.SearchCount += len(indexWrapper.FileList)
		resultWrapper.SearchSize += indexWrapper.Size
		return true
	})

	// 对结果进行排序
	model.SortMoviesUtils(resultWrapper.FileList, searchParam.SortField, searchParam.SortType, se.LastSortField, se.LastSortType)
	se.LastSortField = searchParam.SortField
	se.LastSortType = searchParam.SortType

	// 直接缓存完整结果集
	se.addHistory(cacheKey, resultWrapper)

	// 进行分页
	resultWrapper.FileList, resultWrapper.ResultSize = model.GetPageOfFiles(
		resultWrapper.FileList, searchParam.Page, searchParam.PageSize)
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
	bucketCount := 0
	se.SearchIndexMap.Range(func(key, value interface{}) bool {
		bucketCount++
		return true
	})

	// 根据 bucket 数量动态调整 goroutine 池大小
	poolSize := 10
	if bucketCount > 0 {
		if bucketCount < 5 {
			poolSize = bucketCount
		} else if bucketCount > 20 {
			poolSize = 20 // 最大 20 个并发
		} else {
			poolSize = bucketCount
		}
	}

	resultWrapper.ResultCount = searchParam.PageSize

	// 使用 goroutine 池控制并发数量
	pool := utils.NewGoroutinePool(poolSize)
	// 预估结果大小，减少通道内存分配
	resultChan := make(chan model.SearchResultWrapper, bucketCount*2)

	// 并发搜索所有 bucket
	se.SearchIndexMap.Range(func(key, value interface{}) bool {
		index := value.(*bucketFile)
		if index.isEmpty() {
			return true
		}
		pool.Submit(func() {
			defer func() {
				if r := recover(); r != nil {
					AddLogMemory("搜索 bucket 发生异常：%v", r)
				}
			}()
			indexWrapper := index.searchBucket(searchParam)
			if indexWrapper.IsNotEmpty() {
				select {
				case resultChan <- indexWrapper:
				case <-time.After(30 * time.Second):
					AddLogMemory("发送搜索结果超时，丢弃结果")
				}
			}
		})
		return true
	})

	// 等待所有搜索完成并关闭通道
	done := make(chan struct{})
	go func() {
		pool.Wait()
		close(resultChan)
		close(done)
	}()

	// 收集搜索结果，添加超时控制
	timeout := time.After(30 * time.Second) // 30秒超时
	for {
		select {
		case data, ok := <-resultChan:
			if !ok {
				// 通道已关闭，搜索完成
				goto searchDone
			}
			// 直接追加到结果列表，减少内存分配
			resultWrapper.FileList = append(resultWrapper.FileList, data.FileList...)
			resultWrapper.SearchCount += len(data.FileList)
			resultWrapper.SearchSize += data.Size
		case <-done:
			// 所有 goroutine 已完成，通道将关闭
			for data := range resultChan {
				resultWrapper.FileList = append(resultWrapper.FileList, data.FileList...)
				resultWrapper.SearchCount += len(data.FileList)
				resultWrapper.SearchSize += data.Size
			}
			goto searchDone
		case <-timeout:
			// 搜索超时，等待 goroutine 完成
			AddLogMemory("搜索超时，部分结果可能未返回")
			<-done // 等待所有 goroutine 完成
			for data := range resultChan {
				resultWrapper.FileList = append(resultWrapper.FileList, data.FileList...)
				resultWrapper.SearchCount += len(data.FileList)
				resultWrapper.SearchSize += data.Size
			}
			goto searchDone
		}
	}

searchDone:
	// 对结果进行排序
	model.SortMoviesUtils(resultWrapper.FileList, searchParam.SortField, searchParam.SortType, se.LastSortField, se.LastSortType)
	se.LastSortField = searchParam.SortField
	se.LastSortType = searchParam.SortType

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
	act, ok := se.ActressMap[id]
	if ok {
		return act
	}
	return model.Actress{}
}

func (se *searchEnginCore) setBucket(baseDir string, bucket *bucketFile) {
	se.SearchIndexMap.Store(baseDir, bucket)
}

// buildIndexEngin 构建索引
// 1. 遍历所有的索引文件，将文件信息存储到SearchIndex中
// 2. 遍历所有的索引文件，将重复的文件信息存储到CodeRepeat中
// 3. 遍历所有的索引文件，将演员信息存储到ActressLib中
func (se *searchEnginCore) buildIndexEngin() {
	defer func() {
		if r := recover(); r != nil {
			AddLogMemory("构建索引发生异常: %v", r)
			AddLogMemory("堆栈信息: %s", string(debug.Stack()))
		}
	}()

	se.KeywordHistoryCache.Clear()

	se.buildIndexEnginTotalInfo()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				AddLogMemory("构建演员数据发生异常: %v", r)
				AddLogMemory("堆栈信息: %s", string(debug.Stack()))
			}
		}()
		se.buildActressData()
	}()
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				AddLogMemory("构建重复数据发生异常: %v", r)
				AddLogMemory("堆栈信息: %s", string(debug.Stack()))
			}
		}()
		se.buildRepeatData()
	}()
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				AddLogMemory("构建其他数据发生异常: %v", r)
				AddLogMemory("堆栈信息: %s", string(debug.Stack()))
			}
		}()
		se.buildOthersData()
	}()
	wg.Wait()

}

func (se *searchEnginCore) buildIndexEnginTotalInfo() {
	se.TotalCount = 0
	se.TotalSize = 0
	se.SearchIndexMap.Range(func(key, value any) bool {
		index := value.(*bucketFile)
		if index.isNotEmpty() {
			index.mu.RLock()
			se.TotalSize += index.TotalSize
			se.TotalCount += index.TotalCount
			index.mu.RUnlock()
		}
		return true
	})
}

func (se *searchEnginCore) buildActressData() {
	start := time.Now()
	se.ActressMap = make(map[string]model.Actress)
	se.SearchIndexMap.Range(func(key, value any) bool {
		index := value.(*bucketFile)
		if index.isNotEmpty() {
			index.mu.RLock()
			for _, movie := range index.FileLib {
				if len(movie.Actress) > 0 {
					curActress, ok := se.ActressMap[movie.Actress]
					if ok {
						curActress.PlusCnt()
						curActress.PlusSize(movie.Size)
						curActress.AddImage(movie.Png)
						curActress.AddImage(movie.Jpg)
						se.ActressMap[movie.Actress] = curActress
					} else {
						se.ActressMap[movie.Actress] = model.NewActress(movie.Actress, movie.Jpg, movie.Size)
					}
				}
			}
			index.mu.RUnlock()
		}
		return true
	})
	ti := time.Since(start)
	AddLogMemory("buildIndexEnginTotalInfo time:%d", ti.Milliseconds())
}

func (se *searchEnginCore) buildRepeatData() {
	start := time.Now()
	se.RepeatSearch = []model.Movie{}

	// 预分配 map 容量，提高性能
	sizeRepeats := make(map[int64]repeatModel, se.TotalCount/10)
	codeRepeats := make(map[string]repeatModel, se.TotalCount/10)
	fileRepeats := make(map[string]model.Movie, se.TotalCount/5)

	se.SearchIndexMap.Range(func(key, value any) bool {
		index := value.(*bucketFile)
		if index.isNotEmpty() {
			index.mu.RLock()
			for _, movie := range index.FileLib {
				if movie.IsNull() {
					continue
				}
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
			index.mu.RUnlock()
		}
		return true
	})
	sizeRepeats = nil
	codeRepeats = nil
	// 预分配切片容量
	se.RepeatSearch = make([]model.Movie, 0, len(fileRepeats))
	for _, model := range fileRepeats {
		se.RepeatSearch = append(se.RepeatSearch, model)
	}
	sort.Slice(se.RepeatSearch, func(i, j int) bool {
		return se.RepeatSearch[i].Size > se.RepeatSearch[j].Size
	})
	fileRepeats = nil
	ti := time.Since(start)
	AddLogMemory("buildRepeatData time:%d", ti.Milliseconds())

}

func (se *searchEnginCore) buildOthersData() {
	start := time.Now()
	se.SearchIndexMap.Range(func(key, value any) bool {
		index := value.(*bucketFile)
		if index.isNotEmpty() {
			index.mu.RLock()
			for _, movie := range index.FileLib {
				consts.TypeSizePlus(movie.MovieType, movie.Size)
				if len(movie.Tags) > 0 {
					for i := range movie.Tags {
						consts.TagSizePlus(movie.Tags[i], movie.Size)
					}

				}
				consts.SeriesPlus(movie.Studio, movie.Size)
			}
			index.mu.RUnlock()
		}
		return true
	})
	ti := time.Since(start)
	AddLogMemory("buildOthersData time:%d", ti.Milliseconds())
}
