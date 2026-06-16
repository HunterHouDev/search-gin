package service

import (
	"context"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"sort"
	"strings"
	"time"
)

// PageAuthor 演员搜索
func (se *searchEngineCore) PageAuthor(searchParam model.SearchParam) model.PageAuthorResultWrapper {
	snap := se.loadSnapshot()

	if searchParam.Keyword == "" {
		switch searchParam.SortField {
		case "Size":
			if se.actorSizeCache != nil {
				return buildAuthorResult(se.actorSizeCache, searchParam)
			}
		case "Cnt":
			if se.actorCountCache != nil {
				return buildAuthorResult(se.actorCountCache, searchParam)
			}
		}
	}

	var result []model.Author
	if searchParam.Keyword == "" {
		result = make([]model.Author, 0, len(snap.actorMap))
		for _, author := range snap.actorMap {
			result = append(result, author)
		}
	} else {
		result = make([]model.Author, 0)
		for _, author := range snap.actorMap {
			if strings.Contains(author.Name, searchParam.Keyword) {
				result = append(result, author)
			}
		}
	}

	switch searchParam.SortField {
	case "Size":
		sort.Slice(result, func(i, j int) bool { return result[i].Size > result[j].Size })
		if searchParam.Keyword == "" {
			se.actorSizeCache = result
		}
	case "Cnt":
		sort.Slice(result, func(i, j int) bool { return result[i].Cnt > result[j].Cnt })
		if searchParam.Keyword == "" {
			se.actorCountCache = result
		}
	}

	return buildAuthorResult(result, searchParam)
}

// buildAuthorResult 构造演员搜索结果
func buildAuthorResult(authors []model.Author, param model.SearchParam) model.PageAuthorResultWrapper {
	wrapper := model.PageAuthorResultWrapper{}
	list, size := model.GetAuthorPageOfFiles(authors, param.Page, param.PageSize)
	wrapper.FileList = list
	wrapper.Size = size
	wrapper.ResultCount = len(list)
	return wrapper
}

// returnRepeatSearch 返回重复文件
func (se *searchEngineCore) returnRepeatSearch() model.PageResultWrapper {
	snap := se.loadSnapshot()
	wrapper := model.NewPageWrapper()
	if len(snap.repeatFiles) > 0 {
		wrapper.FileList = make([]model.FileItem, len(snap.repeatFiles))
		copy(wrapper.FileList, snap.repeatFiles)
	}
	wrapper.ResultCount = len(snap.repeatFiles)
	wrapper.LibCount = len(snap.repeatFiles)
	wrapper.SearchCount = len(snap.repeatFiles)
	return wrapper
}

// cachedResult wraps search result with epoch to detect stale cache entries
type cachedResult struct {
	epoch int64
	data  model.PageResultWrapper
}

// PageAsync 异步分页搜索
func (se *searchEngineCore) PageAsync(searchParam model.SearchParam) model.PageResultWrapper {
	if searchParam.OnlyRepeat {
		return se.returnRepeatSearch()
	}
	wrapper := model.NewPageWrapper()

	// 缓存命中
	cacheKey := searchParam.UniWords()
	if matchValue, ok := se.KeywordHistoryCache.Get(cacheKey); ok {
		cr, ok2 := matchValue.(cachedResult)
		if !ok2 {
			// 脏缓存：类型不匹配，删除后继续搜索
			se.KeywordHistoryCache.Delete(cacheKey)
		} else if cr.epoch != se.cacheEpoch.Load() {
			// 缓存过时：属于旧快照，删除后重新搜索
			se.KeywordHistoryCache.Delete(cacheKey)
		} else {
			wrapper = cr.data
			wrapper.FileList, wrapper.ResultSize = model.GetPageOfFiles(
				wrapper.FileList, searchParam.Page, searchParam.PageSize)
			return wrapper
		}
	}

	snap := se.loadSnapshot()
	bucketCount := len(snap.buckets)
	if bucketCount <= 0 {
		AddLogMemory("警告: bucketCount=0, 跳过搜索")
		wrapper.FileList = []model.FileItem{}
		return wrapper
	}

	poolSize := se.searchPool.Cap()
	if bucketCount < poolSize {
		poolSize = bucketCount
	}

	wrapper.ResultCount = searchParam.PageSize
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resultChan := make(chan model.SearchResultWrapper, bucketCount*2)

	for _, bucket := range snap.buckets {
		if bucket.isEmpty() {
			continue
		}
		b := bucket
		se.searchPool.Submit(func() {
			defer func() {
				if r := recover(); r != nil {
					AddLogMemory("搜索 bucket 异常: %v", r)
				}
			}()
			w := b.searchBucket(searchParam)
			if w.IsNotEmpty() {
				select {
				case resultChan <- w:
				case <-ctx.Done():
				}
			}
		})
	}

	go func() {
		defer utils.RecoverPanic()
		se.searchPool.Wait()
		close(resultChan)
	}()

	// 收集结果
loop:
	for {
		select {
		case data, ok := <-resultChan:
			if !ok {
				break loop
			}
			wrapper.FileList = append(wrapper.FileList, data.FileList...)
			wrapper.SearchCount += len(data.FileList)
			wrapper.SearchSize += data.Size
		case <-ctx.Done():
			AddLogMemory("搜索超时，部分结果可能未返回")
			se.searchPool.Wait()
			for data := range resultChan {
				wrapper.FileList = append(wrapper.FileList, data.FileList...)
				wrapper.SearchCount += len(data.FileList)
				wrapper.SearchSize += data.Size
			}
			break loop
		}
	}

	model.SortFileItems(wrapper.FileList, searchParam.SortField, searchParam.SortType)

	// 只缓存小结果集：空关键词不缓存，结果超过 2000 条不缓存
	if searchParam.Keyword != "" && wrapper.SearchCount <= 2000 {
		se.KeywordHistoryCache.Set(cacheKey, cachedResult{epoch: se.cacheEpoch.Load(), data: wrapper})
	}

	wrapper.FileList, wrapper.ResultSize = model.GetPageOfFiles(
		wrapper.FileList, searchParam.Page, searchParam.PageSize)
	return wrapper
}

// ── 查询方法 ──────────────────────────────────────────────────────

// FindById 查找文件
func (se *searchEngineCore) FindById(id string) model.FileItem {
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
	return model.FileItem{}
}

// FindAuthorByName 按名称查找演员
func (se *searchEngineCore) FindAuthorByName(name string) model.Author {
	snap := se.loadSnapshot()
	if a, ok := snap.actorMap[name]; ok {
		return a
	}
	return model.Author{}
}

// GetAuthorCount 获取演员总数
func (se *searchEngineCore) GetAuthorCount() int {
	return len(se.loadSnapshot().actorMap)
}
