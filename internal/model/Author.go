package model

import (
	"search-gin/pkg/utils"
	"strings"
)

// Author 作者聚合信息
type Author struct {
	Name    string
	Url     string
	Cnt     int
	Size    int64
	SizeStr string
	Images  []string
}

// Clone 深拷贝 Author
func (a *Author) Clone() *Author {
	if a == nil {
		return nil
	}
	images := make([]string, len(a.Images))
	copy(images, a.Images)
	return &Author{
		Name:    a.Name,
		Url:     a.Url,
		Cnt:     a.Cnt,
		Size:    a.Size,
		SizeStr: a.SizeStr,
		Images:  images,
	}
}

// NewAuthor 创建作者聚合对象
func NewAuthor(name string, url string, size int64) *Author {
	return &Author{
		Name:    name,
		Url:     url,
		Cnt:     1,
		Size:    size,
		SizeStr: utils.GetSizeStr(size),
		Images:  []string{url},
	}
}

func (act *Author) PlusCnt() {
	act.Cnt = act.Cnt + 1
}

func (act *Author) IsEmpty() bool {
	return act.Name == ""
}

func (act *Author) IsNotEmpty() bool {
	return !act.IsEmpty()
}

func (act *Author) PlusSize(size int64) {
	act.Size = act.Size + size
	act.SizeStr = utils.GetSizeStr(act.Size)
}

func (act *Author) AddImage(image string) {
	if !utils.HasItem(act.Images, image) {
		act.Images = append(act.Images, image)
	}
}

// MinusCnt 减少计数（增量索引重建用）
func (act *Author) MinusCnt() {
	act.Cnt--
}

// MinusSize 减少大小（增量索引重建用）
func (act *Author) MinusSize(size int64) {
	act.Size -= size
	if act.Size < 0 {
		act.Size = 0
	}
	act.SizeStr = utils.GetSizeStr(act.Size)
}

// GetAuthorPageOfFiles 作者分页
func GetAuthorPageOfFiles(files []Author, pageNo int, pageSize int) ([]Author, int64) {
	paged, _ := utils.SlicePage(files, pageNo, pageSize)
	var volume int64
	for _, f := range paged {
		volume += f.Size
	}
	return paged, volume
}

// SearchAuthorByKeyWord 按关键词搜索作者
func SearchAuthorByKeyWord(files map[string]Author, keyWord string) []Author {
	keywordUpper := strings.ToUpper(keyWord)
	resultWrapper := make([]Author, 0, len(files))
	for _, file := range files {
		if len(keyWord) > 0 {
			if strings.Contains(strings.ToUpper(file.Name), keywordUpper) {
				resultWrapper = append(resultWrapper, file)
			}
		} else {
			resultWrapper = append(resultWrapper, file)
		}
	}
	return resultWrapper
}
