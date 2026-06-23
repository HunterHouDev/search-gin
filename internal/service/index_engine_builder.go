package service

import (
	"runtime/debug"
	"search-gin/internal/model"
	"search-gin/internal/sse"
	"sort"
	"strings"
	"time"
)

// ── 批量/增量索引重建 ──────────────────────────────────────────────

// rebuildWithBuckets 批量重建：一次性替换所有 bucket，O(N) 聚合
func (se *searchEngineCore) rebuildWithBuckets(entries map[string]*bucketFile) {
	defer func() {
		if r := recover(); r != nil {
			LogMem.Add("rebuildWithBuckets 异常: %v", r)
			LogMem.Add("堆栈: %s", string(debug.Stack()))
		}
	}()

	se.rebuildMu.Lock()
	defer se.rebuildMu.Unlock()

	LogMem.Add("rebuildWithBuckets: 开始批量重建, %d 个目录", len(entries))
	start := time.Now()

	newIndex := buildIndexFromBuckets(entries)
	se.installIndex(newIndex)
	se.repeatsDirty.Store(false) // 全量重建已计算最新重复列表，清除脏标记

	ti := time.Since(start)
	LogMem.Add("rebuildWithBuckets: 完成, 耗时 %dms, 文件数 %d", ti.Milliseconds(), newIndex.totalCount)
}

// rebuildWithBucket 用指定目录的新 bucket 构造新快照并原子替换
func (se *searchEngineCore) rebuildWithBucket(baseDir string, newBucket *bucketFile) {
	defer func() {
		if r := recover(); r != nil {
			LogMem.Add("rebuildWithBucket 异常: %v", r)
			LogMem.Add("堆栈: %s", string(debug.Stack()))
		}
	}()

	se.rebuildMu.Lock()
	defer se.rebuildMu.Unlock()

	LogMem.Add("rebuildWithBucket: 开始处理目录 %s", baseDir)
	start := time.Now()

	dirs := GetOSSetting().Dirs
	dirSet := make(map[string]struct{}, len(dirs))
	for _, d := range dirs {
		dirSet[d] = struct{}{}
	}

	old := se.loadIndex()
	newBuckets := make(map[string]*bucketFile, len(old.buckets)+1)
	for k, v := range old.buckets {
		if k == baseDir {
			continue
		}
		if _, ok := dirSet[k]; !ok {
			continue
		}
		newBuckets[k] = v
	}
	if newBucket != nil && !newBucket.isEmpty() {
		newBuckets[baseDir] = newBucket
	}

	LogMem.Add("rebuildWithBucket: bucket 数量 %d -> %d", len(old.buckets), len(newBuckets))

	newIndex := buildIndexFromBuckets(newBuckets)
	se.installIndex(newIndex)
	se.repeatsDirty.Store(false) // 全量重建已计算最新重复列表，清除脏标记

	ti := time.Since(start)
	LogMem.Add("rebuildWithBucket: 完成, 耗时 %dms, 文件数 %d", ti.Milliseconds(), newIndex.totalCount)
}

// rebuildWithBucketIncremental 增量重建：只遍历变化的 bucket（O(变化量)）
func (se *searchEngineCore) rebuildWithBucketIncremental(baseDir string, newBucket *bucketFile) {
	defer func() {
		if r := recover(); r != nil {
			LogMem.Add("rebuildWithBucketIncremental 异常: %v", r)
			LogMem.Add("堆栈: %s", string(debug.Stack()))
		}
	}()

	se.rebuildMu.Lock()
	defer se.rebuildMu.Unlock()

	start := time.Now()
	old := se.loadIndex()

	dirs := GetOSSetting().Dirs
	dirSet := make(map[string]struct{}, len(dirs))
	for _, d := range dirs {
		dirSet[d] = struct{}{}
	}

	newBuckets := make(map[string]*bucketFile, len(old.buckets)+1)
	for k, v := range old.buckets {
		if k == baseDir {
			continue
		}
		if _, ok := dirSet[k]; !ok {
			continue
		}
		newBuckets[k] = v
	}
	if newBucket != nil && !newBucket.isEmpty() {
		newBuckets[baseDir] = newBucket
	}

	index := &searchIndex{
		buckets:     newBuckets,
		bucketCount: int32(len(newBuckets)),
		totalSize:   old.totalSize,
		totalCount:  old.totalCount,
		authorMap:   cloneActorMap(old.authorMap),
		typeMenu:    cloneMenuMap(old.typeMenu),
		tagMenu:     cloneMenuMap(old.tagMenu),
		seriesCount: cloneMenuMap(old.seriesCount),
	}

	oldBucket := old.buckets[baseDir]
	if oldBucket != nil && !oldBucket.isEmpty() {
		subtractBucketFromIndex(index, oldBucket)
	}
	if newBucket != nil && !newBucket.isEmpty() {
		addBucketToIndex(index, newBucket)
	}

	recomputeRepeats(index)
	se.installIndex(index)
	se.repeatsDirty.Store(false) // 增量重建已计算最新重复列表，清除脏标记

	Sp.IncrementProcessedBuckets()
	prog := Sp.Get()
	sse.BroadcastEvent("index_update", map[string]interface{}{
		"processed": prog.ProcessedBuckets,
		"total":     prog.TotalBuckets,
	})

	ti := time.Since(start)
	LogMem.Add("rebuildWithBucketIncremental: 完成, 耗时 %dms, bucket %s, 文件数 %d", ti.Milliseconds(), baseDir, index.totalCount)
}

// ── 快照聚合操作 ──────────────────────────────────────────────────

// buildIndexFromBuckets 遍历所有 bucket，构造完整的 searchIndex
func buildIndexFromBuckets(buckets map[string]*bucketFile) *searchIndex {
	index := &searchIndex{
		buckets:     make(map[string]*bucketFile, len(buckets)),
		authorMap:   make(map[string]model.Author),
		typeMenu:    make(map[string]model.FileInfo),
		tagMenu:     make(map[string]model.FileInfo),
		seriesCount: make(map[string]model.FileInfo),
	}

	for k, v := range buckets {
		index.buckets[k] = v
	}

	sizeRepeats := make(map[int64]repeatModel, 1000)
	codeRepeats := make(map[string]repeatModel, 1000)
	fileRepeats := make(map[string]model.FileItem, 2000)

	for _, bucket := range index.buckets {
		if bucket.isEmpty() {
			continue
		}
		bucket.mu.RLock()

		index.bucketCount++

		for _, movie := range bucket.FileLib {
			addFileToIndex(index, movie)

			if !movie.IsNull() {
				pkSize := movie.Size
				rs, ok := sizeRepeats[pkSize]
				if ok {
					rs.Count++
					fileRepeats[rs.Files.Path] = rs.Files
					fileRepeats[movie.Path] = *movie
					sizeRepeats[pkSize] = rs
				} else {
					sizeRepeats[pkSize] = repeatModel{Code: movie.Code, Files: *movie, Count: 1}
				}

				pkCode := strings.ReplaceAll(movie.Code, "-", "")
				pkCode = strings.ReplaceAll(pkCode, "_", "")
				rc, ok := codeRepeats[pkCode]
				if ok {
					rc.Count++
					fileRepeats[rc.Files.Path] = rc.Files
					fileRepeats[movie.Path] = *movie
					codeRepeats[pkCode] = rc
				} else {
					codeRepeats[pkCode] = repeatModel{Code: movie.Code, Files: *movie, Count: 1}
				}
			}
		}

		bucket.mu.RUnlock()
	}

	repeatSearch := make([]model.FileItem, 0, len(fileRepeats))
	for _, m := range fileRepeats {
		repeatSearch = append(repeatSearch, m)
	}
	sort.Slice(repeatSearch, func(i, j int) bool {
		return repeatSearch[i].Size > repeatSearch[j].Size
	})
	index.repeatFiles = repeatSearch

	return index
}

// ── 增量操作辅助函数 ──────────────────────────────────────────────

func cloneActorMap(src map[string]model.Author) map[string]model.Author {
	dst := make(map[string]model.Author, len(src))
	for k, v := range src {
		images := make([]string, len(v.Images))
		copy(images, v.Images)
		dst[k] = model.Author{
			Name:    v.Name,
			Url:     v.Url,
			Cnt:     v.Cnt,
			Size:    v.Size,
			SizeStr: v.SizeStr,
			Images:  images,
		}
	}
	return dst
}

func cloneMenuMap(src map[string]model.FileInfo) map[string]model.FileInfo {
	dst := make(map[string]model.FileInfo, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func subtractBucketFromIndex(index *searchIndex, bucket *bucketFile) {
	bucket.mu.RLock()
	defer bucket.mu.RUnlock()
	for _, movie := range bucket.FileLib {
		subtractFileFromIndex(index, movie)
	}
}

func addBucketToIndex(index *searchIndex, bucket *bucketFile) {
	bucket.mu.RLock()
	defer bucket.mu.RUnlock()
	for _, movie := range bucket.FileLib {
		addFileToIndex(index, movie)
	}
}

// recomputeRepeats 在所有 bucket 上重新计算重复文件
func recomputeRepeats(index *searchIndex) {
	sizeRepeats := make(map[int64]repeatModel, 1000)
	codeRepeats := make(map[string]repeatModel, 1000)
	fileRepeats := make(map[string]model.FileItem, 2000)

	for _, bucket := range index.buckets {
		if bucket.isEmpty() {
			continue
		}
		bucket.mu.RLock()

		for _, movie := range bucket.FileLib {
			if movie.IsNull() {
				continue
			}

			rs, ok := sizeRepeats[movie.Size]
			if ok {
				rs.Count++
				fileRepeats[rs.Files.Path] = rs.Files
				fileRepeats[movie.Path] = *movie
				sizeRepeats[movie.Size] = rs
			} else {
				sizeRepeats[movie.Size] = repeatModel{Code: movie.Code, Files: *movie, Count: 1}
			}

			pkCode := strings.ReplaceAll(movie.Code, "-", "")
			pkCode = strings.ReplaceAll(pkCode, "_", "")
			rc, ok := codeRepeats[pkCode]
			if ok {
				rc.Count++
				fileRepeats[rc.Files.Path] = rc.Files
				fileRepeats[movie.Path] = *movie
				codeRepeats[pkCode] = rc
			} else {
				codeRepeats[pkCode] = repeatModel{Code: movie.Code, Files: *movie, Count: 1}
			}
		}

		bucket.mu.RUnlock()
	}

	repeatSearch := make([]model.FileItem, 0, len(fileRepeats))
	for _, m := range fileRepeats {
		repeatSearch = append(repeatSearch, m)
	}
	sort.Slice(repeatSearch, func(i, j int) bool {
		return repeatSearch[i].Size > repeatSearch[j].Size
	})
	index.repeatFiles = repeatSearch
}

// ── 单文件操作 ────────────────────────────────────────────────────

func subtractFileFromIndex(index *searchIndex, movie *model.FileItem) {
	index.totalCount--
	index.totalSize -= movie.Size

	if len(movie.Author) > 0 {
		if cur, ok := index.authorMap[movie.Author]; ok {
			cur.MinusCnt()
			cur.MinusSize(movie.Size)
			if cur.Cnt <= 0 {
				delete(index.authorMap, movie.Author)
			} else {
				index.authorMap[movie.Author] = cur
			}
		}
	}

	mt := movie.MovieType
	if mt == "" {
		mt = "无"
	}
	if v, ok := index.typeMenu[mt]; ok {
		updated := v.Minus(movie.Size)
		if updated.Cnt <= 0 {
			delete(index.typeMenu, mt)
		} else {
			index.typeMenu[mt] = updated
		}
	}
	if v, ok := index.typeMenu["全部"]; ok {
		updated := v.Minus(movie.Size)
		if updated.Cnt <= 0 {
			delete(index.typeMenu, "全部")
		} else {
			index.typeMenu["全部"] = updated
		}
	}

	for i := range movie.Tags {
		if v, ok := index.tagMenu[movie.Tags[i]]; ok {
			updated := v.Minus(movie.Size)
			if updated.Cnt <= 0 {
				delete(index.tagMenu, movie.Tags[i])
			} else {
				index.tagMenu[movie.Tags[i]] = updated
			}
		}
	}

	if len(movie.Studio) > 0 {
		if v, ok := index.seriesCount[movie.Studio]; ok {
			updated := v.Minus(movie.Size)
			if updated.Cnt <= 0 {
				delete(index.seriesCount, movie.Studio)
			} else {
				index.seriesCount[movie.Studio] = updated
			}
		}
	}
}

func addFileToIndex(index *searchIndex, movie *model.FileItem) {
	index.totalCount++
	index.totalSize += movie.Size

	if len(movie.Author) > 0 {
		if cur, ok := index.authorMap[movie.Author]; ok {
			cur.PlusCnt()
			cur.PlusSize(movie.Size)
			cur.AddImage(movie.Png)
			cur.AddImage(movie.Jpg)
			index.authorMap[movie.Author] = cur
		} else {
			index.authorMap[movie.Author] = model.NewAuthor(movie.Author, movie.Jpg, movie.Size)
		}
	}

	mt := movie.MovieType
	if mt == "" {
		mt = "无"
	}
	if v, ok := index.typeMenu[mt]; ok {
		index.typeMenu[mt] = v.Plus(movie.Size)
	} else {
		index.typeMenu[mt] = model.FileInfo{Name: mt, Cnt: 1, Size: movie.Size}
	}
	if v, ok := index.typeMenu["全部"]; ok {
		index.typeMenu["全部"] = v.Plus(movie.Size)
	} else {
		index.typeMenu["全部"] = model.FileInfo{Name: "全部", Cnt: 1, Size: movie.Size}
	}

	for i := range movie.Tags {
		if v, ok := index.tagMenu[movie.Tags[i]]; ok {
			index.tagMenu[movie.Tags[i]] = v.Plus(movie.Size)
		} else {
			index.tagMenu[movie.Tags[i]] = model.FileInfo{Name: movie.Tags[i], Cnt: 1, Size: movie.Size, IsDir: true}
		}
	}

	if len(movie.Studio) > 0 {
		if v, ok := index.seriesCount[movie.Studio]; ok {
			index.seriesCount[movie.Studio] = v.Plus(movie.Size)
		} else {
			index.seriesCount[movie.Studio] = model.FileInfo{Name: movie.Studio, Cnt: 1, Size: movie.Size, IsDir: true}
		}
	}
}

// ReplaceFile 同步替换索引中的单文件记录
func (se *searchEngineCore) ReplaceFile(oldFile, newFile model.FileItem) {
	op := fileOp{opType: "replace", oldFile: oldFile, newFile: newFile}
	se.flushPendingOp(op)
}

// DeleteFile 同步从索引中删除文件记录
func (se *searchEngineCore) DeleteFile(file model.FileItem) {
	op := fileOp{opType: "delete", oldFile: file}
	se.flushPendingOp(op)
}
func (se *searchEngineCore) flushPendingOp(ops fileOp) {
	se.flushPendingOps([]fileOp{ops})
}

// flushPendingOps 将一组文件操作同步应用到索引
// 合并为一次索引安装 + 一次缓存清除
func (se *searchEngineCore) flushPendingOps(ops []fileOp) {
	se.rebuildMu.Lock()
	defer se.rebuildMu.Unlock()

	start := time.Now()

	// 加载当前索引快照（持 rebuildMu，不会被并发重建覆盖）
	index := se.loadIndex()

	// ── 按 baseDir 分组收集操作，保证同 bucket 多操作顺序应用 ──
	type dirOps struct {
		baseDir string
		ops     []fileOp
	}
	ordered := make([]dirOps, 0)
	dirMap := make(map[string]int, len(ops)) // baseDir → ordered 下标

	for _, op := range ops {
		baseDir := op.oldFile.BaseDir
		if idx, ok := dirMap[baseDir]; ok {
			ordered[idx].ops = append(ordered[idx].ops, op)
		} else {
			dirMap[baseDir] = len(ordered)
			ordered = append(ordered, dirOps{baseDir: baseDir, ops: []fileOp{op}})
		}
	}

	// ── 构造新索引 ──
	newIndex := shallowCopyIndex(index)

	for _, dg := range ordered {
		bucket := index.buckets[dg.baseDir]
		if bucket == nil {
			continue
		}

		// 克隆 bucket，按顺序应用所有操作
		newBucket := bucket.clone()
		applied := false

		for _, op := range dg.ops {
			switch op.opType {
			case "replace":
				if _, exists := newBucket.FileLib[op.oldFile.Id]; !exists {
					continue
				}
				// 更新 bucket 数据
					f := op.newFile
					newBucket.FileLib[op.oldFile.Id] = &f
				sizeDiff := op.newFile.Size - op.oldFile.Size
				newBucket.TotalSize += sizeDiff
				// 更新 TypeIndex
				if op.oldFile.MovieType != op.newFile.MovieType {
					if op.oldFile.MovieType != "" {
						if ids, ok := newBucket.TypeIndex[op.oldFile.MovieType]; ok {
							delete(ids, op.oldFile.Id)
							if len(ids) == 0 {
								delete(newBucket.TypeIndex, op.oldFile.MovieType)
							}
						}
					}
					if op.newFile.MovieType != "" {
						if newBucket.TypeIndex[op.newFile.MovieType] == nil {
							newBucket.TypeIndex[op.newFile.MovieType] = map[string]struct{}{}
						}
						newBucket.TypeIndex[op.newFile.MovieType][op.newFile.Id] = struct{}{}
					}
				}
				// 更新 index 级聚合
					subtractFileFromIndex(newIndex, &op.oldFile)
					addFileToIndex(newIndex, &op.newFile)
				applied = true

			case "delete":
				entry, exists := newBucket.FileLib[op.oldFile.Id]
				if !exists {
					continue
				}
				// 更新 bucket 数据
				delete(newBucket.FileLib, op.oldFile.Id)
				newBucket.TotalCount--
				newBucket.TotalSize -= entry.Size
				if entry.MovieType != "" {
					if ids, ok := newBucket.TypeIndex[entry.MovieType]; ok {
						delete(ids, entry.Id)
						if len(ids) == 0 {
							delete(newBucket.TypeIndex, entry.MovieType)
						}
					}
				}
				// 更新 index 级聚合
				subtractFileFromIndex(newIndex, entry)
				applied = true
			}
		}

		if applied {
			newIndex.buckets[dg.baseDir] = newBucket
		}
	}

	// 单次安装 + 单次缓存清除（对比逐操作安装，大幅降低开销）
	se.installIndexSkipDisk(newIndex)
	// 重复文件列表延迟到下次 returnRepeatSearch 时惰性重算
	se.repeatsDirty.Store(true)

	ti := time.Since(start)
	LogMem.Add("flushPendingOps: 完成, 耗时 %dms, 操作数 %d", ti.Milliseconds(), len(ops))
}

// shallowCopyIndex 浅拷贝 searchIndex，共享未修改的 bucket 指针
func shallowCopyIndex(index *searchIndex) *searchIndex {
	newBuckets := make(map[string]*bucketFile, len(index.buckets))
	for k, v := range index.buckets {
		newBuckets[k] = v
	}

	newAuthorMap := make(map[string]model.Author, len(index.authorMap))
	for k, v := range index.authorMap {
		newAuthorMap[k] = v
	}

	newTypeMenu := make(map[string]model.FileInfo, len(index.typeMenu))
	for k, v := range index.typeMenu {
		newTypeMenu[k] = v
	}

	newTagMenu := make(map[string]model.FileInfo, len(index.tagMenu))
	for k, v := range index.tagMenu {
		newTagMenu[k] = v
	}

	newSeriesCount := make(map[string]model.FileInfo, len(index.seriesCount))
	for k, v := range index.seriesCount {
		newSeriesCount[k] = v
	}

	return &searchIndex{
		buckets:     newBuckets,
		bucketCount: index.bucketCount,
		totalSize:   index.totalSize,
		totalCount:  index.totalCount,
		repeatFiles: index.repeatFiles,
		authorMap:   newAuthorMap,
		typeMenu:    newTypeMenu,
		tagMenu:     newTagMenu,
		seriesCount: newSeriesCount,
	}
}
