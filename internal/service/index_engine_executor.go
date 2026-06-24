package service

import (
	"context"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"sort"
	"strings"
	"time"
)

// ── searchEngineCore 类型与搜索 ────────────────────────────────────

// cachedResult wraps search result with epoch to detect stale cache entries
type cachedResult struct {
	epoch int64
	data  model.PageResultWrapper
}

// Page 搜索并返回 API 分页结果
func (se *searchEngineCore) Page(searchParam model.SearchParam) utils.Page {
	sr := se.pageAsync(searchParam)
	result := utils.NewPage()
	result.TotalCnt = sr.SearchCount
	result.TotalSize = utils.GetSizeStr(sr.SearchSize)
	result.ResultSize = utils.GetSizeStr(sr.SearchSize)
	result.SetResultCnt(sr.SearchCount, searchParam.Page)
	result.CurSize = utils.GetSizeStr(sr.ResultSize)
	result.CurCnt = sr.ResultCount
	files := make([]model.FileItem, len(sr.FileList))
	copy(files, sr.FileList)
	for i := range files {
		files[i].PageNo = searchParam.Page
	}
	result.Data = files
	result.SetProgress(IndexNumber.Load())
	return result
}

// pageAsync 异步分页搜索：先获取索引快照，再按路径分发
func (se *searchEngineCore) pageAsync(p model.SearchParam) model.PageResultWrapper {
	index := se.loadIndex()
	if p.OnlyRepeat {
		return se.returnRepeatSearch(index)
	}
	if cached, ok := se.tryCache(p); ok {
		return cached
	}
	return se.doSearch(index, p)
}

// tryCache 命中缓存则返回已分页的结果，否则返回 false
func (se *searchEngineCore) tryCache(p model.SearchParam) (model.PageResultWrapper, bool) {
	cacheKey := p.UniWords()
	v, ok := se.KeywordHistoryCache.Get(cacheKey)
	if !ok {
		return model.PageResultWrapper{}, false
	}
	cr, ok2 := v.(cachedResult)
	if !ok2 || cr.epoch != se.cacheEpoch.Load() {
		se.KeywordHistoryCache.Delete(cacheKey)
		return model.PageResultWrapper{}, false
	}
	w := cr.data
	paged, size := model.GetPageOfFiles(w.FileList, p.Page, p.PageSize)
	w.FileList = paged
	w.ResultSize = size
	return w, true
}

// doSearch 执行搜索：分发 bucket → 收集结果 → 排序 → 缓存 → 分页
func (se *searchEngineCore) doSearch(index *searchIndex, p model.SearchParam) model.PageResultWrapper {
	bucketCount := len(index.buckets)
	if bucketCount <= 0 {
		LogMem.Add("警告: bucketCount=0, 跳过搜索")
		return model.PageResultWrapper{FileList: []model.FileItem{}}
	}

	wrapper := model.NewPageWrapper()
	wrapper.ResultCount = p.PageSize
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// stopped 在 collectResults 返回后关闭，通知所有生产者停止发送结果
	stopped := make(chan struct{})
	resultChan := make(chan model.PageResultWrapper, bucketCount*2)

	// 分发搜索
	for _, bucket := range index.buckets {
		if bucket.isEmpty() {
			continue
		}
		b := bucket
		se.searchPool.Submit(func() {
			defer func() {
				if r := recover(); r != nil {
					LogMem.Add("搜索 bucket 异常: %v", r)
				}
			}()
			w := b.searchBucket(p)
			if w.IsNotEmpty() {
				select {
				case resultChan <- w:
				case <-ctx.Done():
				case <-stopped:
				}
			}
		})
	}

	go func() {
		defer utils.RecoverPanic()
		se.searchPool.Wait()
		// 若已超时（stopped 已关闭），不再关闭 resultChan
		select {
		case <-stopped:
		default:
			close(resultChan)
		}
	}()

	// 收集结果
	se.collectResults(&wrapper, resultChan, ctx)
	close(stopped) // 通知所有生产者停止发送

	model.SortFileItems(wrapper.FileList, p.SortField, p.SortType)

	if p.Keyword != "" && wrapper.SearchCount <= 2000 {
		se.KeywordHistoryCache.Set(p.UniWords(), cachedResult{epoch: se.cacheEpoch.Load(), data: wrapper})
	}

	wrapper.FileList, wrapper.ResultSize = model.GetPageOfFiles(wrapper.FileList, p.Page, p.PageSize)
	return wrapper
}

// collectResults 从 channel 收集搜索结果，含超时处理
func (se *searchEngineCore) collectResults(w *model.PageResultWrapper, ch <-chan model.PageResultWrapper, ctx context.Context) {
loop:
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				break loop
			}
			w.FileList = append(w.FileList, data.FileList...)
			w.SearchCount += len(data.FileList)
			w.SearchSize += data.Size
		case <-ctx.Done():
			LogMem.Add("搜索超时，返回部分结果")
			// 超时后尽力收集已在 channel 中的结果，不等待正在执行的 goroutine
			for done := false; !done; {
				select {
				case data, ok := <-ch:
					if !ok {
						done = true
						break
					}
					w.FileList = append(w.FileList, data.FileList...)
					w.SearchCount += len(data.FileList)
					w.SearchSize += data.Size
				default:
					done = true
				}
			}
			break loop
		}
	}
}

// returnRepeatSearch 返回重复文件（惰性重算：脏标记为 true 时触发全量重算）
func (se *searchEngineCore) returnRepeatSearch(index *searchIndex) model.PageResultWrapper {
	// 若单文件操作导致重复列表过期，惰性重算并安装新快照
	if se.repeatsDirty.CompareAndSwap(true, false) {
		newIndex := shallowCopyIndex(index)
		recomputeRepeats(newIndex)
		se.installIndexSkipDisk(newIndex)
		index = newIndex
	}

	wrapper := model.NewPageWrapper()
	if len(index.repeatFiles) > 0 {
		wrapper.FileList = make([]model.FileItem, len(index.repeatFiles))
		copy(wrapper.FileList, index.repeatFiles)
	}
	wrapper.ResultCount = len(index.repeatFiles)
	wrapper.LibCount = len(index.repeatFiles)
	wrapper.SearchCount = len(index.repeatFiles)
	return wrapper
}

// ── 作者搜索 ────────────────────────────────────────────────────────

// PageAuthor 作者搜索
func (se *searchEngineCore) PageAuthor(searchParam model.SearchParam) model.PageAuthorResultWrapper {
	index := se.loadIndex()

	if searchParam.Keyword == "" {
		switch searchParam.SortField {
		case "Size":
			se.authorCacheMu.RLock()
			cache := se.authorSizeCache
			se.authorCacheMu.RUnlock()
			if cache != nil {
				return buildAuthorResult(cache, searchParam)
			}
		case "Cnt":
			se.authorCacheMu.RLock()
			cache := se.authorCountCache
			se.authorCacheMu.RUnlock()
			if cache != nil {
				return buildAuthorResult(cache, searchParam)
			}
		}
	}

	var result []model.Author
	if searchParam.Keyword == "" {
		result = make([]model.Author, 0, len(index.authorMap))
		for _, author := range index.authorMap {
			result = append(result, *author)
		}
	} else {
		result = make([]model.Author, 0)
		for _, author := range index.authorMap {
			if strings.Contains(author.Name, searchParam.Keyword) {
				result = append(result, *author)
			}
		}
	}

	switch searchParam.SortField {
	case "Size":
		sort.Slice(result, func(i, j int) bool { return result[i].Size > result[j].Size })
		if searchParam.Keyword == "" {
			se.authorCacheMu.Lock()
			se.authorSizeCache = result
			se.authorCacheMu.Unlock()
		}
	case "Cnt":
		sort.Slice(result, func(i, j int) bool { return result[i].Cnt > result[j].Cnt })
		if searchParam.Keyword == "" {
			se.authorCacheMu.Lock()
			se.authorCountCache = result
			se.authorCacheMu.Unlock()
		}
	}

	return buildAuthorResult(result, searchParam)
}

// buildAuthorResult 构造作者搜索结果
func buildAuthorResult(authors []model.Author, param model.SearchParam) model.PageAuthorResultWrapper {
	wrapper := model.PageAuthorResultWrapper{}
	list, size := model.GetAuthorPageOfFiles(authors, param.Page, param.PageSize)
	wrapper.FileList = list
	wrapper.Size = size
	wrapper.ResultCount = len(list)
	return wrapper
}

// ── 查询方法 ──────────────────────────────────────────────────────

// FindById O(1) 查找文件，使用 idIndex 全局索引替代全桶线性扫描
func (se *searchEngineCore) FindById(id string) model.FileItem {
	index := se.loadIndex()
	if f, ok := index.idIndex[id]; ok {
		return *f
	}
	return model.FileItem{}
}

// FindAuthorByName 按名称查找作者
func (se *searchEngineCore) FindAuthorByName(name string) model.Author {
	index := se.loadIndex()
	if a, ok := index.authorMap[name]; ok {
		return *a
	}
	return model.Author{}
}

// GetAuthorCount 获取作者总数
func (se *searchEngineCore) GetAuthorCount() int {
	return len(se.loadIndex().authorMap)
}
