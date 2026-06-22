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
	FileLib      map[string]model.FileItem
	// 倒排索引，类型 -> 文件ID集合 (O(1) 去重)
	TypeIndex map[string]map[string]struct{}
	mu        sync.RWMutex
}

// clone 深拷贝 bucket（Copy-on-Write 用）
func (fs *bucketFile) clone() *bucketFile {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	newFileLib := make(map[string]model.FileItem, len(fs.FileLib))
	for k, v := range fs.FileLib {
		newFileLib[k] = v
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
		FileLib:      map[string]model.FileItem{},
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
	fs.FileLib[m.Id] = m
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
		fs.FileLib[file.Id] = file
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

func (fs *bucketFile) get(id string) model.FileItem {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	movieFile, ok := fs.FileLib[id]
	if ok {
		return movieFile
	}
	return model.FileItem{}
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

	// 定义公共过滤函数：同时检查关键词 + 高级过滤
	filter := func(file model.FileItem) bool {
		if !matchAdvancedFilters(file, searchParam) {
			return false
		}
		if keywords == nil {
			return true
		}
		return matchKeywords(file, keywords)
	}

	fs.mu.RLock()

	if movieType != "" && movieType != model.UndefinedStr {
		if fileIds, ok := fs.TypeIndex[movieType]; ok {
			for id := range fileIds {
				if file, ok := fs.FileLib[id]; ok {
					if filter(file) {
						resultWrapper.AddWrapperItem(file)
					}
				}
			}
		}
	} else {
		for _, file := range fs.FileLib {
			if filter(file) {
				resultWrapper.AddWrapperItem(file)
			}
		}
	}

	fs.mu.RUnlock()
	return resultWrapper
}

// matchKeywords 检查文件路径是否匹配所有关键词
func matchKeywords(file model.FileItem, keywords []string) bool {
	filePath := file.PathUpper
	for _, keyword := range keywords {
		if !strings.Contains(filePath, keyword) {
			return false
		}
	}
	return true
}

// matchAdvancedFilters 检查高级过滤条件：大小范围、日期范围、扩展名
func matchAdvancedFilters(file model.FileItem, p model.SearchParam) bool {
	if p.MinSize > 0 && file.Size < p.MinSize {
		return false
	}
	if p.MaxSize > 0 && file.Size > p.MaxSize {
		return false
	}
	if len(p.FileExts) > 0 {
		suffix := utils.GetSuffix(file.Name)
		matched := false
		for _, ext := range p.FileExts {
			if strings.EqualFold(suffix, ext) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if p.DateFrom != "" || p.DateTo != "" {
		// MTime 格式示例: "2006-01-02 15:04:05"
		fileTime, err := time.Parse("2006-01-02 15:04:05", file.MTime)
		if err != nil {
			return true // 无法解析日期时不过滤
		}
		if p.DateFrom != "" {
			from, err := time.Parse("2006-01-02", p.DateFrom)
			if err == nil && fileTime.Before(from) {
				return false
			}
		}
		if p.DateTo != "" {
			to, err := time.Parse("2006-01-02", p.DateTo)
			if err == nil {
				// 包含截止日期当天
				toEnd := to.Add(24*time.Hour - time.Nanosecond)
				if fileTime.After(toEnd) {
					return false
				}
			}
		}
	}
	return true
}
