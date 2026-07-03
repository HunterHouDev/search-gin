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
	result.PageSize = searchParam.PageSize
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
	// 传递聚合数据
	agg := map[string]interface{}{
		"authors": sr.AuthorAgg,
		"tags":    sr.TagAgg,
		"series":  sr.SeriesAgg,
	}
	if sr.ResultMinSize > 0 && sr.ResultMaxSize > 0 {
		agg["minSize"] = sr.ResultMinSize
		agg["maxSize"] = sr.ResultMaxSize
	}
	if sr.ResultMinDate > 0 {
		agg["minDate"] = sr.ResultMinDate
		agg["maxDate"] = sr.ResultMaxDate
	}
	if len(sr.ExtAgg) > 0 {
		agg["exts"] = sr.ExtAgg
	}
	result.Aggregates = agg
	return result
}

// pageAsync 异步分页搜索：先获取索引快照，再按路径分发
func (se *searchEngineCore) pageAsync(p model.SearchParam) model.PageResultWrapper {
	index := se.loadIndex()
	if p.OnlyRepeat {
		w := se.returnRepeatSearch(index)
		model.SortFileItems(w.FileList, p.SortField, p.SortType)
		return w
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
	defer close(stopped)
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

	model.SortFileItems(wrapper.FileList, p.SortField, p.SortType)

	// 计算搜索结果中的聚合数据（基于全量匹配结果，分页前计算）
	wrapper.AuthorAgg, wrapper.TagAgg, wrapper.SeriesAgg, wrapper.ExtAgg, wrapper.ResultMinSize, wrapper.ResultMaxSize, wrapper.ResultMinDate, wrapper.ResultMaxDate = computeAggregates(wrapper.FileList)

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

// computeAggregates 从搜索结果文件中统计聚合数据
// 内部使用指针 map 避免 struct 值拷贝，返回时转换为值 map 供 JSON 序列化
func computeAggregates(files []model.FileItem) (authorAgg, tagAgg, seriesAgg, extAgg map[string]model.AggItem, minSize, maxSize, minDate, maxDate int64) {
	type aggPtr = *model.AggItem
	authorPtrs := make(map[string]aggPtr)
	tagPtrs := make(map[string]aggPtr)
	seriesPtrs := make(map[string]aggPtr)
	extPtrs := make(map[string]aggPtr)

	for _, f := range files {
		// 大小范围
		if f.Size > maxSize {
			maxSize = f.Size
		}
		if minSize == 0 || f.Size < minSize {
			minSize = f.Size
		}

		// 日期范围
		if f.MTimeUnix > maxDate {
			maxDate = f.MTimeUnix
		}
		if minDate == 0 || f.MTimeUnix < minDate {
			minDate = f.MTimeUnix
		}

		// 按作者聚合
		if f.Author != "" {
			if entry, ok := authorPtrs[f.Author]; ok {
				entry.Cnt++
				entry.Size += f.Size
			} else {
				authorPtrs[f.Author] = &model.AggItem{Cnt: 1, Size: f.Size}
			}
		}

		// 按标签聚合
		for _, tag := range f.Tags {
			if tag == "" {
				continue
			}
			if entry, ok := tagPtrs[tag]; ok {
				entry.Cnt++
			} else {
				tagPtrs[tag] = &model.AggItem{Cnt: 1}
			}
		}

		// 按系列（Studio）聚合
		if f.Studio != "" {
			if entry, ok := seriesPtrs[f.Studio]; ok {
				entry.Cnt++
			} else {
				seriesPtrs[f.Studio] = &model.AggItem{Cnt: 1}
			}
		}

		// 按扩展名聚合
		if ext := utils.GetSuffix(f.Name); ext != "" {
			if entry, ok := extPtrs[ext]; ok {
				entry.Cnt++
			} else {
				extPtrs[ext] = &model.AggItem{Cnt: 1}
			}
		}
	}

	// 转换为值 map
	authorAgg = make(map[string]model.AggItem, len(authorPtrs))
	for k, v := range authorPtrs {
		authorAgg[k] = *v
	}
	tagAgg = make(map[string]model.AggItem, len(tagPtrs))
	for k, v := range tagPtrs {
		tagAgg[k] = *v
	}
	seriesAgg = make(map[string]model.AggItem, len(seriesPtrs))
	for k, v := range seriesPtrs {
		seriesAgg[k] = *v
	}
	extAgg = make(map[string]model.AggItem, len(extPtrs))
	for k, v := range extPtrs {
		extAgg[k] = *v
	}

	return
}

// ── 查询方法 ──────────────────────────────────────────────────────

// FindById O(1) 查找文件，优先使用 idIndex；未命中时遍历所有 bucket 兜底
func (se *searchEngineCore) FindById(id string) model.FileItem {
	return se.loadIndex().FindById(id)
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
