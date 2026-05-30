package model

import (
	"search-gin/pkg/utils"
	"strings"
)

type Actress struct {
	Name    string
	Url     string
	Cnt     int
	Size    int64
	SizeStr string
	Images  []string
}

func NewActress(name string, url string, size int64) Actress {
	return Actress{
		Name:    name,
		Url:     url,
		Cnt:     1,
		Size:    size,
		SizeStr: utils.GetSizeStr(size),
		Images:  []string{url},
	}
}
func (act *Actress) PlusCnt() {
	act.Cnt = act.Cnt + 1
}

func (act *Actress) IsEmpty() bool {
	return act.Name == ""
}

func (act *Actress) IsNotEmpty() bool {
	return !act.IsEmpty()
}

func (act *Actress) PlusSize(size int64) {
	act.Size = act.Size + size
	act.SizeStr = utils.GetSizeStr(act.Size)
}
func (act *Actress) AddImage(image string) {
	if !utils.HasItem(act.Images, image) {
		act.Images = append(act.Images, image)
	}
}

func GetActressPageOfFiles(files []Actress, pageNo int, pageSize int) ([]Actress, int64) {
	if len(files) == 0 {
		return files, 0
	}
	if pageNo <= 0 {
		pageNo = 1
	}
	length := len(files)
	start := (pageNo - 1) * pageSize

	if start >= length {
		return []Actress{}, 0
	}

	end := length
	if length-start >= pageSize {
		end = start + pageSize
	}

	var volume int64
	data := make([]Actress, end-start)
	for i := start; i < end; i++ {
		data[i-start] = files[i]
		volume += files[i].Size
	}
	return data, volume
}

func SearchActressByKeyWord(files map[string]Actress, keyWord string) []Actress {
	keywordUpper := strings.ToUpper(keyWord)
	resultWrapper := make([]Actress, 0, len(files))
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
