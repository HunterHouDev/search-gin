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
	FileLib      map[string]model.Movie
	// 倒排索引，类型 -> 文件ID列表
	TypeIndex map[string][]string
	mu        sync.RWMutex
}

func newInstance(name string) bucketFile {
	return bucketFile{
		InstanceName: name,
		TotalSize:    0,
		TotalCount:   0,
		FileLib:      map[string]model.Movie{},
		TypeIndex:    map[string][]string{},
	}
}

func newInstanceWithFiles(baseDir string, files []model.Movie) bucketFile {
	bucket := newInstance(baseDir)
	for _, file := range files {
		bucket.put(file)
	}
	return bucket
}

func (fs *bucketFile) isNotEmpty() bool {
	return len(fs.FileLib) > 0
}

func (fs *bucketFile) isEmpty() bool {
	return !fs.isNotEmpty()
}

func (fs *bucketFile) put(model model.Movie) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.FileLib[model.Id] = model
	fs.TotalSize = fs.TotalSize + model.Size
	fs.TotalCount = fs.TotalCount + 1

	// 构建类型倒排索引
	fs.buildTypeIndex(model)
}

// buildTypeIndex 为文件构建类型倒排索引
func (fs *bucketFile) buildTypeIndex(model model.Movie) {
	if model.MovieType == "" {
		return
	}

	// 去重检查
	has := false
	fileIds, ok := fs.TypeIndex[model.MovieType]
	if ok {
		for _, id := range fileIds {
			if id == model.Id {
				has = true
				break
			}
		}
	}
	if !has {
		fs.TypeIndex[model.MovieType] = append(fileIds, model.Id)
	}
}

func (fs *bucketFile) get(id string) model.Movie {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	movieFile, ok := fs.FileLib[id]
	if ok {
		return movieFile
	}
	return model.Movie{}
}

func (fs *bucketFile) searchBucket(searchParam model.SearchParam) model.SearchResultWrapper {
	resultWrapper := model.NewSearchWrapper()
	keyWord := searchParam.Keyword
	movieType := searchParam.MovieType

	// 获取候选文件列表
	var candidates []model.Movie

	fs.mu.RLock()
	if movieType != "" && movieType != "undefined" {
		// 使用类型倒排索引筛选
		if fileIds, ok := fs.TypeIndex[movieType]; ok {
			for _, id := range fileIds {
				if file, ok := fs.FileLib[id]; ok {
					candidates = append(candidates, file)
				}
			}
		}
	} else {
		// 没有类型限制，遍历所有文件
		for _, file := range fs.FileLib {
			candidates = append(candidates, file)
		}
	}
	fs.mu.RUnlock()

	// 如果没有关键词，返回所有候选文件
	if keyWord == "" || keyWord == "undefined" {
		for _, file := range candidates {
			resultWrapper.AddWrapperItem(file)
		}
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

	// 对候选文件进行模糊匹配
	for _, file := range candidates {
		// 检查文件路径是否包含所有关键词
		filePath := strings.ToUpper(file.Path)
		matchAll := true
		for _, keyword := range keywords {
			if !strings.Contains(filePath, keyword) {
				matchAll = false
				break
			}
		}
		if matchAll {
			resultWrapper.AddWrapperItem(file)
		}
	}

	return resultWrapper
}
