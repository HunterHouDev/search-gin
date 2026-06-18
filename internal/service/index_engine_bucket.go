package service

import (
	"search-gin/internal/model"
	"strings"
	"sync"
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

	fs.mu.RLock()

	if keyWord == "" || keyWord == model.UndefinedStr {
		// 无关键词，按类型筛选后直接返回
		if movieType != "" && movieType != model.UndefinedStr {
			if fileIds, ok := fs.TypeIndex[movieType]; ok {
				for id := range fileIds {
					if file, ok := fs.FileLib[id]; ok {
						resultWrapper.AddWrapperItem(file)
					}
				}
			}
		} else {
			for _, file := range fs.FileLib {
				resultWrapper.AddWrapperItem(file)
			}
		}
		fs.mu.RUnlock()
		return resultWrapper
	}

	// 预处理关键词
	keyWord = strings.TrimSpace(keyWord)
	keywords := []string{}
	for _, word := range strings.Split(keyWord, " ") {
		word = strings.TrimSpace(word)
		if len(word) > 0 {
			keywords = append(keywords, strings.ToUpper(word))
		}
	}

	// 遍历时直接过滤，不构建中间切片
	if movieType != "" && movieType != model.UndefinedStr {
		if fileIds, ok := fs.TypeIndex[movieType]; ok {
			for id := range fileIds {
				if file, ok := fs.FileLib[id]; ok {
					if matchKeywords(file, keywords) {
						resultWrapper.AddWrapperItem(file)
					}
				}
			}
		}
	} else {
		for _, file := range fs.FileLib {
			if matchKeywords(file, keywords) {
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
