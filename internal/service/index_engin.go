package service

import (
	"context"
	"search-gin/pkg/consts"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"runtime/debug"
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
 RepeatSearch                   []model.Movie
 ActressSizeWrapperNullKeyword  []model.Actress
 ActressCountWrapperNullKeyword []model.Actress
 ActressMap                     map[string]model.Actress
 // TotalSizeMutex 保护 TotalSize/TotalCount 的并发读写（buildOthersData 在 goroutine 中修改）
 TotalSizeMutex   sync.RWMutex
 TotalSize        int64
 TotalCount       int
 KeywordHistoryCache *utils.LRUCache // 替换 sync.Map 为 LRU 缓存
 searchPool       *utils.GoroutinePool // 全局 goroutine 池，避免每次搜索重复创建
 BucketCount      int32
}

// Reset 清空搜索引擎全部状态和缓存
func (se *searchEnginCore) Reset() {
	se.SearchIndexMap.Range(func(key, value interface{}) bool {
		se.SearchIndexMap.Delete(key)
		return true
	})
	// 清空所有状态
	se.RepeatSearch = nil
	se.ActressMap = nil
	se.KeywordHistoryCache.Clear()
	se.TotalSize = 0
	se.TotalCount = 0
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
	act, ok := se.ActressMap[id]
	if ok {
		return act
	}
	return model.Actress{}
}

func (se *searchEnginCore) setBucket(baseDir string, bucket *bucketFile) {
	se.SearchIndexMap.Store(baseDir, bucket)
	atomic.AddInt32(&se.BucketCount, 1)
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

	start := time.Now()
	se.KeywordHistoryCache.Clear()

	// 重置总计信息
	se.TotalCount = 0
	se.TotalSize = 0

	// 使用局部变量聚合，避免在 Range 中频繁写入全局 sync.Map
	se.ActressMap = make(map[string]model.Actress)
	localTypeMenu := make(map[string]consts.MenuSize)
	localTagMenu := make(map[string]consts.MenuSize)
	localSeriesCount := make(map[string]consts.MenuSize)
	sizeRepeats := make(map[int64]repeatModel, 1000)
	codeRepeats := make(map[string]repeatModel, 1000)
	fileRepeats := make(map[string]model.Movie, 2000)

	var bucketCount int32

	se.SearchIndexMap.Range(func(key, value any) bool {
		index := value.(*bucketFile)
		if index.isEmpty() {
			return true
		}
		index.mu.RLock()

		// 1. 总计信息
		se.TotalSize += index.TotalSize
		se.TotalCount += index.TotalCount
		bucketCount++

		// 2. 遍历文件中逐条处理
		for _, movie := range index.FileLib {
			// ---- 演员数据 ----
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
		return true
	})

	atomic.StoreInt32(&se.BucketCount, bucketCount)

	// 批量写入全局 sync.Map
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

	// 构建重复结果
	sizeRepeats = nil
	codeRepeats = nil
	se.RepeatSearch = make([]model.Movie, 0, len(fileRepeats))
	for _, m := range fileRepeats {
		se.RepeatSearch = append(se.RepeatSearch, m)
	}
	sort.Slice(se.RepeatSearch, func(i, j int) bool {
		return se.RepeatSearch[i].Size > se.RepeatSearch[j].Size
	})
	fileRepeats = nil

	ti := time.Since(start)
	AddLogMemory("buildIndexEngin (single-pass) time:%d", ti.Milliseconds())
}


