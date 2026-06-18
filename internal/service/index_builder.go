package service

import (
	"runtime/debug"
	"search-gin/internal/model"
	"search-gin/pkg/consts"
	"sort"
	"strings"
	"time"
)

// ── 批量/增量索引重建 ──────────────────────────────────────────────

// rebuildWithBuckets 批量重建：一次性替换所有 bucket，O(N) 聚合
func (se *searchEngineCore) rebuildWithBuckets(entries map[string]*bucketFile) {
	defer func() {
		if r := recover(); r != nil {
			consts.LogMem.Add("rebuildWithBuckets 异常: %v", r)
			consts.LogMem.Add("堆栈: %s", string(debug.Stack()))
		}
	}()

	se.rebuildMu.Lock()
	defer se.rebuildMu.Unlock()

	consts.LogMem.Add("rebuildWithBuckets: 开始批量重建, %d 个目录", len(entries))
	start := time.Now()

	newSnap := buildSnapshotFromBuckets(entries)
	se.installSnapshot(newSnap)

	ti := time.Since(start)
	consts.LogMem.Add("rebuildWithBuckets: 完成, 耗时 %dms, 文件数 %d", ti.Milliseconds(), newSnap.totalCount)
}

// rebuildWithBucket 用指定目录的新 bucket 构造新快照并原子替换
func (se *searchEngineCore) rebuildWithBucket(baseDir string, newBucket *bucketFile) {
	defer func() {
		if r := recover(); r != nil {
			consts.LogMem.Add("rebuildWithBucket 异常: %v", r)
			consts.LogMem.Add("堆栈: %s", string(debug.Stack()))
		}
	}()

	se.rebuildMu.Lock()
	defer se.rebuildMu.Unlock()

	consts.LogMem.Add("rebuildWithBucket: 开始处理目录 %s", baseDir)
	start := time.Now()

	dirs := consts.GetOSSetting().Dirs
	dirSet := make(map[string]struct{}, len(dirs))
	for _, d := range dirs {
		dirSet[d] = struct{}{}
	}

	old := se.loadSnapshot()
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

	consts.LogMem.Add("rebuildWithBucket: bucket 数量 %d -> %d", len(old.buckets), len(newBuckets))

	newSnap := buildSnapshotFromBuckets(newBuckets)
	se.installSnapshot(newSnap)

	ti := time.Since(start)
	consts.LogMem.Add("rebuildWithBucket: 完成, 耗时 %dms, 文件数 %d", ti.Milliseconds(), newSnap.totalCount)
}

// rebuildWithBucketIncremental 增量重建：只遍历变化的 bucket（O(变化量)）
func (se *searchEngineCore) rebuildWithBucketIncremental(baseDir string, newBucket *bucketFile) {
	defer func() {
		if r := recover(); r != nil {
			consts.LogMem.Add("rebuildWithBucketIncremental 异常: %v", r)
			consts.LogMem.Add("堆栈: %s", string(debug.Stack()))
		}
	}()

	se.rebuildMu.Lock()
	defer se.rebuildMu.Unlock()

	start := time.Now()
	old := se.loadSnapshot()

	dirs := consts.GetOSSetting().Dirs
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

	snap := &searchSnapshot{
		buckets:     newBuckets,
		bucketCount: int32(len(newBuckets)),
		totalSize:   old.totalSize,
		totalCount:  old.totalCount,
		actorMap:    cloneActorMap(old.actorMap),
		typeMenu:    cloneMenuMap(old.typeMenu),
		tagMenu:     cloneMenuMap(old.tagMenu),
		seriesCount: cloneMenuMap(old.seriesCount),
	}

	oldBucket := old.buckets[baseDir]
	if oldBucket != nil && !oldBucket.isEmpty() {
		subtractBucketFromSnapshot(snap, oldBucket)
	}
	if newBucket != nil && !newBucket.isEmpty() {
		addBucketToSnapshot(snap, newBucket)
	}

	recomputeRepeats(snap)
	se.installSnapshot(snap)

	ti := time.Since(start)
	consts.LogMem.Add("rebuildWithBucketIncremental: 完成, 耗时 %dms, bucket %s, 文件数 %d", ti.Milliseconds(), baseDir, snap.totalCount)
}

// ── 快照聚合操作 ──────────────────────────────────────────────────

// buildSnapshotFromBuckets 遍历所有 bucket，构造完整的 searchSnapshot
func buildSnapshotFromBuckets(buckets map[string]*bucketFile) *searchSnapshot {
	snap := &searchSnapshot{
		buckets:     make(map[string]*bucketFile, len(buckets)),
		actorMap:    make(map[string]model.Author),
		typeMenu:    make(map[string]consts.MenuSize),
		tagMenu:     make(map[string]consts.MenuSize),
		seriesCount: make(map[string]consts.MenuSize),
	}

	for k, v := range buckets {
		snap.buckets[k] = v
	}

	sizeRepeats := make(map[int64]repeatModel, 1000)
	codeRepeats := make(map[string]repeatModel, 1000)
	fileRepeats := make(map[string]model.FileItem, 2000)

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
			if len(movie.Author) > 0 {
				if cur, ok := snap.actorMap[movie.Author]; ok {
					cur.PlusCnt()
					cur.PlusSize(movie.Size)
					cur.AddImage(movie.Png)
					cur.AddImage(movie.Jpg)
					snap.actorMap[movie.Author] = cur
				} else {
					snap.actorMap[movie.Author] = model.NewAuthor(movie.Author, movie.Jpg, movie.Size)
				}
			}

			// 重复检测
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

			for i := range movie.Tags {
				if v, ok := snap.tagMenu[movie.Tags[i]]; ok {
					snap.tagMenu[movie.Tags[i]] = v.Plus(movie.Size)
				} else {
					snap.tagMenu[movie.Tags[i]] = consts.MenuSize{Name: movie.Tags[i], Cnt: 1, Size: movie.Size, IsDir: true}
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

	repeatSearch := make([]model.FileItem, 0, len(fileRepeats))
	for _, m := range fileRepeats {
		repeatSearch = append(repeatSearch, m)
	}
	sort.Slice(repeatSearch, func(i, j int) bool {
		return repeatSearch[i].Size > repeatSearch[j].Size
	})
	snap.repeatFiles = repeatSearch

	return snap
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

func cloneMenuMap(src map[string]consts.MenuSize) map[string]consts.MenuSize {
	dst := make(map[string]consts.MenuSize, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func subtractBucketFromSnapshot(snap *searchSnapshot, bucket *bucketFile) {
	bucket.mu.RLock()
	defer bucket.mu.RUnlock()

	for _, movie := range bucket.FileLib {
		movie := movie
		snap.totalCount--
		snap.totalSize -= movie.Size

		// 演员
		if len(movie.Author) > 0 {
			if cur, ok := snap.actorMap[movie.Author]; ok {
				cur.MinusCnt()
				cur.MinusSize(movie.Size)
				if cur.Cnt <= 0 {
					delete(snap.actorMap, movie.Author)
				} else {
					snap.actorMap[movie.Author] = cur
				}
			}
		}

		// 类型菜单
		mt := movie.MovieType
		if mt == "" {
			mt = "无"
		}
		if v, ok := snap.typeMenu[mt]; ok {
			updated := v.Minus(movie.Size)
			if updated.Cnt <= 0 {
				delete(snap.typeMenu, mt)
			} else {
				snap.typeMenu[mt] = updated
			}
		}
		if v, ok := snap.typeMenu["全部"]; ok {
			updated := v.Minus(movie.Size)
			if updated.Cnt <= 0 {
				delete(snap.typeMenu, "全部")
			} else {
				snap.typeMenu["全部"] = updated
			}
		}

		// 标签菜单
		for i := range movie.Tags {
			if v, ok := snap.tagMenu[movie.Tags[i]]; ok {
				updated := v.Minus(movie.Size)
				if updated.Cnt <= 0 {
					delete(snap.tagMenu, movie.Tags[i])
				} else {
					snap.tagMenu[movie.Tags[i]] = updated
				}
			}
		}

		// 系列菜单
		if len(movie.Studio) > 0 {
			if v, ok := snap.seriesCount[movie.Studio]; ok {
				updated := v.Minus(movie.Size)
				if updated.Cnt <= 0 {
					delete(snap.seriesCount, movie.Studio)
				} else {
					snap.seriesCount[movie.Studio] = updated
				}
			}
		}
	}
}

func addBucketToSnapshot(snap *searchSnapshot, bucket *bucketFile) {
	bucket.mu.RLock()
	defer bucket.mu.RUnlock()

	for _, movie := range bucket.FileLib {
		movie := movie
		snap.totalCount++
		snap.totalSize += movie.Size

		if len(movie.Author) > 0 {
			if cur, ok := snap.actorMap[movie.Author]; ok {
				cur.PlusCnt()
				cur.PlusSize(movie.Size)
				cur.AddImage(movie.Png)
				cur.AddImage(movie.Jpg)
				snap.actorMap[movie.Author] = cur
			} else {
				snap.actorMap[movie.Author] = model.NewAuthor(movie.Author, movie.Jpg, movie.Size)
			}
		}

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

		for i := range movie.Tags {
			if v, ok := snap.tagMenu[movie.Tags[i]]; ok {
				snap.tagMenu[movie.Tags[i]] = v.Plus(movie.Size)
			} else {
				snap.tagMenu[movie.Tags[i]] = consts.MenuSize{Name: movie.Tags[i], Cnt: 1, Size: movie.Size, IsDir: true}
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
}

// recomputeRepeats 在所有 bucket 上重新计算重复文件
func recomputeRepeats(snap *searchSnapshot) {
	sizeRepeats := make(map[int64]repeatModel, 1000)
	codeRepeats := make(map[string]repeatModel, 1000)
	fileRepeats := make(map[string]model.FileItem, 2000)

	for _, bucket := range snap.buckets {
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
				fileRepeats[movie.Path] = movie
				sizeRepeats[movie.Size] = rs
			} else {
				sizeRepeats[movie.Size] = repeatModel{Code: movie.Code, Files: movie, Count: 1}
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

		bucket.mu.RUnlock()
	}

	repeatSearch := make([]model.FileItem, 0, len(fileRepeats))
	for _, m := range fileRepeats {
		repeatSearch = append(repeatSearch, m)
	}
	sort.Slice(repeatSearch, func(i, j int) bool {
		return repeatSearch[i].Size > repeatSearch[j].Size
	})
	snap.repeatFiles = repeatSearch
}

// ── 单文件操作 ────────────────────────────────────────────────────

func subtractFileFromSnapshot(snap *searchSnapshot, movie model.FileItem) {
	snap.totalCount--
	snap.totalSize -= movie.Size

	if len(movie.Author) > 0 {
		if cur, ok := snap.actorMap[movie.Author]; ok {
			cur.MinusCnt()
			cur.MinusSize(movie.Size)
			if cur.Cnt <= 0 {
				delete(snap.actorMap, movie.Author)
			} else {
				snap.actorMap[movie.Author] = cur
			}
		}
	}

	mt := movie.MovieType
	if mt == "" {
		mt = "无"
	}
	if v, ok := snap.typeMenu[mt]; ok {
		updated := v.Minus(movie.Size)
		if updated.Cnt <= 0 {
			delete(snap.typeMenu, mt)
		} else {
			snap.typeMenu[mt] = updated
		}
	}
	if v, ok := snap.typeMenu["全部"]; ok {
		updated := v.Minus(movie.Size)
		if updated.Cnt <= 0 {
			delete(snap.typeMenu, "全部")
		} else {
			snap.typeMenu["全部"] = updated
		}
	}

	for i := range movie.Tags {
		if v, ok := snap.tagMenu[movie.Tags[i]]; ok {
			updated := v.Minus(movie.Size)
			if updated.Cnt <= 0 {
				delete(snap.tagMenu, movie.Tags[i])
			} else {
				snap.tagMenu[movie.Tags[i]] = updated
			}
		}
	}

	if len(movie.Studio) > 0 {
		if v, ok := snap.seriesCount[movie.Studio]; ok {
			updated := v.Minus(movie.Size)
			if updated.Cnt <= 0 {
				delete(snap.seriesCount, movie.Studio)
			} else {
				snap.seriesCount[movie.Studio] = updated
			}
		}
	}
}

func addFileToSnapshot(snap *searchSnapshot, movie model.FileItem) {
	snap.totalCount++
	snap.totalSize += movie.Size

	if len(movie.Author) > 0 {
		if cur, ok := snap.actorMap[movie.Author]; ok {
			cur.PlusCnt()
			cur.PlusSize(movie.Size)
			cur.AddImage(movie.Png)
			cur.AddImage(movie.Jpg)
			snap.actorMap[movie.Author] = cur
		} else {
			snap.actorMap[movie.Author] = model.NewAuthor(movie.Author, movie.Jpg, movie.Size)
		}
	}

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

	for i := range movie.Tags {
		if v, ok := snap.tagMenu[movie.Tags[i]]; ok {
			snap.tagMenu[movie.Tags[i]] = v.Plus(movie.Size)
		} else {
			snap.tagMenu[movie.Tags[i]] = consts.MenuSize{Name: movie.Tags[i], Cnt: 1, Size: movie.Size, IsDir: true}
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

// ReplaceFile 替换索引中的单文件记录（重命名/改标签/改类型后同步索引）
func (se *searchEngineCore) ReplaceFile(oldFile, newFile model.FileItem) {
	se.rebuildMu.Lock()
	defer se.rebuildMu.Unlock()

	snap := se.loadSnapshot()
	bucket := snap.buckets[oldFile.BaseDir]
	if bucket == nil || bucket.isEmpty() {
		return
	}

	bucket.mu.Lock()
	oldEntry, exists := bucket.FileLib[oldFile.Id]
	if !exists {
		bucket.mu.Unlock()
		return
	}
	delete(bucket.FileLib, oldFile.Id)
	bucket.TotalCount--
	bucket.TotalSize -= oldEntry.Size
	if oldEntry.MovieType != "" {
		if ids, ok := bucket.TypeIndex[oldEntry.MovieType]; ok {
			delete(ids, oldEntry.Id)
			if len(ids) == 0 {
				delete(bucket.TypeIndex, oldEntry.MovieType)
			}
		}
	}
	bucket.mu.Unlock()

	bucket.put(newFile)

	newSnap := &searchSnapshot{
		buckets:     snap.buckets,
		bucketCount: snap.bucketCount,
		totalSize:   snap.totalSize,
		totalCount:  snap.totalCount,
		actorMap:    cloneActorMap(snap.actorMap),
		typeMenu:    cloneMenuMap(snap.typeMenu),
		tagMenu:     cloneMenuMap(snap.tagMenu),
		seriesCount: cloneMenuMap(snap.seriesCount),
	}

	subtractFileFromSnapshot(newSnap, oldEntry)
	addFileToSnapshot(newSnap, newFile)
	recomputeRepeats(newSnap)

	se.installSnapshot(newSnap)
}

// DeleteFile 从索引中删除文件记录（删除文件后同步索引）
func (se *searchEngineCore) DeleteFile(file model.FileItem) {
	se.rebuildMu.Lock()
	defer se.rebuildMu.Unlock()

	snap := se.loadSnapshot()
	bucket := snap.buckets[file.BaseDir]
	if bucket == nil || bucket.isEmpty() {
		return
	}

	bucket.mu.Lock()
	entry, exists := bucket.FileLib[file.Id]
	if !exists {
		bucket.mu.Unlock()
		return
	}
	delete(bucket.FileLib, file.Id)
	bucket.TotalCount--
	bucket.TotalSize -= entry.Size
	if entry.MovieType != "" {
		if ids, ok := bucket.TypeIndex[entry.MovieType]; ok {
			delete(ids, entry.Id)
			if len(ids) == 0 {
				delete(bucket.TypeIndex, entry.MovieType)
			}
		}
	}
	bucket.mu.Unlock()

	newSnap := &searchSnapshot{
		buckets:     snap.buckets,
		bucketCount: snap.bucketCount,
		totalSize:   snap.totalSize,
		totalCount:  snap.totalCount,
		actorMap:    cloneActorMap(snap.actorMap),
		typeMenu:    cloneMenuMap(snap.typeMenu),
		tagMenu:     cloneMenuMap(snap.tagMenu),
		seriesCount: cloneMenuMap(snap.seriesCount),
	}

	subtractFileFromSnapshot(newSnap, entry)
	recomputeRepeats(newSnap)

	se.installSnapshot(newSnap)
}
