package service

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"strings"
	"sync"
	"time"
)

type bucketFile struct {
	InstanceName string
	TotalSize    int64
	TotalCount   int
	FileLib      map[string]*model.FileItem
	// 倒排索引，类型 -> 文件ID集合 (O(1) 去重)
	TypeIndex map[string]map[string]struct{}
	mu        sync.RWMutex
}

// clone 深拷贝 bucket（Copy-on-Write 用）
func (fs *bucketFile) clone() *bucketFile {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	newFileLib := make(map[string]*model.FileItem, len(fs.FileLib))
	for k, v := range fs.FileLib {
		f := *v // 值拷贝每个 FileItem
		newFileLib[k] = &f
	}

	newTypeIndex := make(map[string]map[string]struct{}, len(fs.TypeIndex))
	for typ, ids := range fs.TypeIndex {
		newIds := make(map[string]struct{}, len(ids))
		for id := range ids {
			newIds[id] = struct{}{}
		}
		newTypeIndex[typ] = newIds
	}

	return &bucketFile{
		InstanceName: fs.InstanceName,
		TotalSize:    fs.TotalSize,
		TotalCount:   fs.TotalCount,
		FileLib:      newFileLib,
		TypeIndex:    newTypeIndex,
	}
}

func newInstance(name string) *bucketFile {
	return &bucketFile{
		InstanceName: name,
		TotalSize:    0,
		TotalCount:   0,
		FileLib:      map[string]*model.FileItem{},
		TypeIndex:    map[string]map[string]struct{}{},
	}
}

func newInstanceWithFiles(baseDir string, files []model.FileItem) *bucketFile {
	bucket := newInstance(baseDir)
	bucket.putBatch(files)
	return bucket
}

func (fs *bucketFile) isNotEmpty() bool {
	return len(fs.FileLib) > 0
}

func (fs *bucketFile) isEmpty() bool {
	return !fs.isNotEmpty()
}

func (fs *bucketFile) put(m model.FileItem) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if m.PathUpper == "" {
		m.PathUpper = strings.ToUpper(m.Path)
	}
	f := new(model.FileItem)
	*f = m
	fs.FileLib[f.Id] = f
	fs.TotalSize = fs.TotalSize + m.Size
	fs.TotalCount = fs.TotalCount + 1

	// 构建类型倒排索引
	fs.buildTypeIndex(m)
}

// putBatch 批量写入文件，一次加锁
func (fs *bucketFile) putBatch(files []model.FileItem) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	for _, file := range files {
		if file.PathUpper == "" {
			file.PathUpper = strings.ToUpper(file.Path)
		}
		f := new(model.FileItem)
		*f = file
		fs.FileLib[f.Id] = f
		fs.TotalSize += file.Size
		fs.TotalCount++
		fs.buildTypeIndex(file)
	}
}

// buildTypeIndex 为文件构建类型倒排索引（O(1) 去重）
func (fs *bucketFile) buildTypeIndex(m model.FileItem) {
	if m.MovieType == "" {
		return
	}
	if _, ok := fs.TypeIndex[m.MovieType]; !ok {
		fs.TypeIndex[m.MovieType] = map[string]struct{}{}
	}
	fs.TypeIndex[m.MovieType][m.Id] = struct{}{}
}

func (fs *bucketFile) get(id string) *model.FileItem {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	movieFile, ok := fs.FileLib[id]
	if ok {
		return movieFile
	}
	return nil
}

func (fs *bucketFile) searchBucket(searchParam model.SearchParam) model.PageResultWrapper {
	resultWrapper := model.NewPageWrapper()
	keyWord := searchParam.Keyword
	movieType := searchParam.MovieType

	// 预处理关键词（仅执行一次，避免在每个文件的 filter 闭包中重复计算）
	var keywords []string
	if keyWord != "" && keyWord != model.UndefinedStr {
		keywords = strings.Fields(strings.ToUpper(keyWord))
	}

	// 预解析日期过滤器（避免每个文件都解析一次）
	var dateFrom, dateTo *time.Time
	if searchParam.DateFrom != "" {
		if t, err := time.Parse("2006-01-02", searchParam.DateFrom); err == nil {
			dateFrom = &t
		}
	}
	if searchParam.DateTo != "" {
		if t, err := time.Parse("2006-01-02", searchParam.DateTo); err == nil {
			toEnd := t.Add(24*time.Hour - time.Nanosecond)
			dateTo = &toEnd
		}
	}

	// 预处理扩展名集合（O(1) 查找替代线性扫描）
	var extSet map[string]struct{}
	if len(searchParam.FileExts) > 0 {
		extSet = make(map[string]struct{}, len(searchParam.FileExts))
		for _, ext := range searchParam.FileExts {
			extSet[strings.ToLower(ext)] = struct{}{}
		}
	}

	// 定义公共过滤函数：同时检查关键词 + 高级过滤
	filter := func(file *model.FileItem) bool {
		if !matchAdvancedFiltersFast(file, searchParam.MinSize, searchParam.MaxSize, dateFrom, dateTo, extSet) {
			return false
		}
		if keywords == nil {
			return true
		}
		return matchKeywords(file, keywords)
	}

	// 快照模式：先拷贝指针列表后释放锁，过滤在锁外执行
	// 避免搜索持读锁时间过长阻塞文件操作（clone 需写锁）
	if movieType != "" && movieType != model.UndefinedStr {
		fs.mu.RLock()
		if fileIds, ok := fs.TypeIndex[movieType]; ok {
			for id := range fileIds {
				if file, ok := fs.FileLib[id]; ok {
					if filter(file) {
						resultWrapper.AddWrapperItem(file)
					}
				}
			}
		}
		fs.mu.RUnlock()
	} else {
		// 无类型过滤时，拷贝指针列表到本地快照
		fs.mu.RLock()
		snapshot := make([]*model.FileItem, 0, len(fs.FileLib))
		for _, file := range fs.FileLib {
			snapshot = append(snapshot, file)
		}
		fs.mu.RUnlock()

		// 预分配 FileList 容量，避免 append 多次扩容
		resultWrapper.FileList = make([]model.FileItem, 0, len(snapshot))
		for _, file := range snapshot {
			if filter(file) {
				resultWrapper.FileList = append(resultWrapper.FileList, *file)
				resultWrapper.Size += file.Size
			}
		}
		return resultWrapper
	}

	return resultWrapper
}

// matchKeywords 检查文件路径是否匹配所有关键词
func matchKeywords(file *model.FileItem, keywords []string) bool {
	filePath := file.PathUpper
	for _, keyword := range keywords {
		if !strings.Contains(filePath, keyword) {
			return false
		}
	}
	return true
}

// matchAdvancedFiltersFast 优化版：使用预解析参数避免每文件重复计算
func matchAdvancedFiltersFast(file *model.FileItem, minSize, maxSize int64, dateFrom, dateTo *time.Time, extSet map[string]struct{}) bool {
	if minSize > 0 && file.Size < minSize {
		return false
	}
	if maxSize > 0 && file.Size > maxSize {
		return false
	}
	if extSet != nil {
		suffix := strings.ToLower(utils.GetSuffix(file.Name))
		if _, ok := extSet[suffix]; !ok {
			return false
		}
	}
	if dateFrom != nil || dateTo != nil {
		var fileTime time.Time
		// 优先使用预计算的 Unix 时间戳，回退到字符串解析
		if file.MTimeUnix > 0 {
			fileTime = time.Unix(file.MTimeUnix, 0)
		} else if file.MTime != "" {
			var err error
			fileTime, err = time.Parse("2006-01-02 15:04:05", file.MTime)
			if err != nil {
				return true
			}
		} else {
			return true
		}
		if dateFrom != nil && fileTime.Before(*dateFrom) {
			return false
		}
		if dateTo != nil && fileTime.After(*dateTo) {
			return false
		}
	}
	return true
}

// matchAdvancedFilters 已废弃，请使用 matchAdvancedFiltersFast
// 保留仅用于编译兼容，实际已无调用点
