package model

import (
	"fmt"
	"search-gin/pkg/utils"
	"sort"
	"strings"
	"time"
)

// Movie 声明一个File结构体 表示一个文件信息
type Movie struct {
	Id        string `xorm:"Varchar(255) pk"  `
	Code      string `xorm:"Varchar(255)"`
	Name      string `xorm:"Text"`
	Path      string `xorm:"Text"`
	BaseDir   string `xorm:"Text"`
	Png       string `xorm:"Text"`
	Nfo       string `xorm:"Text"  json:"Nfo,omitempty"`
	Srt       string `xorm:"Text" json:"Srt,omitempty"`
	Jpg       string `xorm:"Text"`
	Gif       string `xorm:"Text"`
	Actress   string `xorm:"Text"`
	FileType  string `xorm:"Text"`
	DirPath   string `xorm:"Text"`
	Size      int64
	Flag      int64
	SizeStr   string
	CTime     string `xorm:"DateTime"`
	MTime     string `xorm:"DateTime"`
	PTime     string `xorm:"DateTime"`
	MovieType string
	PathUpper string
	ImageBase string   `json:"ImageBase,omitempty"`
	ImageList []string `json:"ImageList,omitempty"`
	Tags      []string

	Studio   string `json:"Studio,omitempty"`
	Supplier string `json:"Supplier,omitempty"`
	Length   string `json:"Length,omitempty"`
	Series   string `json:"Series,omitempty"`
	Director string `json:"Director,omitempty"`
	Title    string
	PngUrl   string `json:"PngUrl,omitempty" xorm:"Text" `
	JpgUrl   string `json:"JpgUrl,omitempty" xorm:"Text"`

	PageNo int
}

// MovieEdit 文件修改模型
type MovieEdit struct {
	Movie
	MoveOut   bool
	NoRefresh bool
}

func EasyFile(dir string, path string, name string, fileType string, size int64, modTime time.Time, baseDir string) Movie {
	// 使用工厂模式 返回一个 Movie 实例
	fileKey, _ := utils.DirpathForId(path)
	movieType := utils.GetMovieType(name)
	Actress := utils.GetActress(name)
	code := utils.GetCode(name)
	result := Movie{
		Id:        fileKey,
		Code:      code,
		Title:     utils.GetTitle(name),
		Name:      name,
		Path:      path,
		Png:       utils.ConcatSuffix(path, "png"),
		Nfo:       utils.ConcatSuffix(path, "nfo"),
		Jpg:       utils.ConcatSuffix(path, "jpg"),
		Srt:       utils.ConcatSuffix(path, "srt"),
		Gif:       utils.ConcatSuffix(path, "gif"),
		Tags:      utils.GetTags(path, ""),
		Actress:   Actress,
		FileType:  fileType,
		DirPath:   dir,
		Size:      size,
		Flag:      1,
		Studio:    utils.GetSeriesByCode(code),
		SizeStr:   utils.GetSizeStr(size),
		CTime:     "",
		MTime:     modTime.Format("2006-01-02 15:04:05"),
		MovieType: movieType,
		BaseDir:   baseDir,
	}
	return result
}

func NewFile(dir string, path string, name string, fileType string, size int64, modTime time.Time, movieType string, baseDir string) Movie {
	// 使用工厂模式 返回一个 Movie 实例
	generateId, _ := utils.DirpathForId(path)
	code := utils.GetCode(name)
	Actress := utils.GetActress(name)
	result := Movie{
		Id:        generateId,
		Code:      code,
		Title:     utils.GetTitle(name),
		Name:      name,
		Path:      path,
		Png:       utils.ConcatSuffix(path, "png"),
		Nfo:       utils.ConcatSuffix(path, "nfo"),
		Jpg:       utils.ConcatSuffix(path, "jpg"),
		Srt:       utils.ConcatSuffix(path, "srt"),
		Gif:       utils.ConcatSuffix(path, "gif"),
		Tags:      utils.GetTags(path, ""),
		Actress:   Actress,
		FileType:  fileType,
		DirPath:   dir,
		Size:      size,
		Flag:      1,
		Studio:    utils.GetSeriesByCode(code),
		SizeStr:   utils.GetSizeStr(size),
		CTime:     "",
		MTime:     modTime.Format("2006-01-02 15:04:05"),
		MovieType: movieType,
		BaseDir:   baseDir,
	}
	return result
}

func (f *Movie) SetId(id string) Movie {
	f.Id = id
	return *f
}

func (f Movie) GetFileInfo() string {
	//
	info := fmt.Sprintf("name: %v\t code:%v\t fileType:%v\t sizeStr:%v\t actress:%v\t path:%v\t",
		f.Name, f.Code, f.FileType, f.SizeStr, f.Actress, f.Path)
	return info
}

// IsNull
func (f Movie) IsNull() bool {
	//
	if f.Id == "" || f.Path == "" {
		return true
	}
	return false
}

func SortMoviesUtils(sortModels []Movie, sF string, sT string, lastSortField string, lastSortType string) {
	//if sF == lastSortField && sT == lastSortType {
	//	return
	//}
	sort.Slice(sortModels, func(i, j int) bool {
		if sF == "Code" && sT == "desc" {
			return sortModels[i].Code > sortModels[j].Code
		} else if sF == "Code" && sT == "asc" {
			return sortModels[i].Code < sortModels[j].Code
		} else if sF == "Size" && sT == "desc" {
			return sortModels[i].Size > sortModels[j].Size
		} else if sF == "Size" && sT == "asc" {
			return sortModels[i].Size < sortModels[j].Size
		} else if sF == "MTime" && sT == "desc" {
			return sortModels[i].MTime > sortModels[j].MTime
		} else if sF == "MTime" && sT == "asc" {
			return sortModels[i].MTime < sortModels[j].MTime
		} else {
			return sortModels[i].MTime > sortModels[j].MTime
		}
	})
}

func GetPageOfFiles(files []Movie, pageNo int, pageSize int) ([]Movie, int64) {
	if len(files) == 0 {
		return files, 0
	}
	if pageNo <= 0 {
		pageNo = 1
	}
	length := len(files)
	start := (pageNo - 1) * pageSize

	if start >= length {
		return []Movie{}, 0
	}

	end := length
	if length-start >= pageSize {
		end = start + pageSize
	}
	if len(files) <= pageSize {
		return files, 0
	}

	var data []Movie
	var volume int64
	for i := start; i < end; i++ {
		curFile := files[i]
		volume += curFile.Size
		data = append(data, curFile)
	}
	return data, volume
}

func SearchByKeyWord(files map[string]Movie, keyWord string, movieType string) SearchResultWrapper {

	resultWrapper := NewSearchWrapper()
	if (keyWord == "" || keyWord == UndefinedStr) && (movieType == "" || movieType == UndefinedStr) {
		for _, file := range files {
			resultWrapper.AddWrapperItem(file)
		}
		return resultWrapper
	}

	// 预处理关键词，提高搜索效率
	keyWord = strings.TrimSpace(keyWord)
	keywords := []string{}
	if len(keyWord) > 0 {
		for _, word := range strings.Split(keyWord, " ") {
			word = strings.TrimSpace(word)
			if len(word) > 0 {
				keywords = append(keywords, strings.ToUpper(word))
			}
		}
	}

	for _, file := range files {
		// 先检查电影类型
		if movieType != "" && file.MovieType != movieType {
			continue
		}

		// 如果没有关键词，直接添加
		if len(keywords) == 0 {
			resultWrapper.AddWrapperItem(file)
			continue
		}
		// 对每个关键词进行匹配
		filepath := strings.ToUpper(file.Path)
		matchAllKeywords := true
		for _, keyword := range keywords {
			// 只要关键词匹配任何一个字段即可
			keywordMatched := strings.Contains(filepath, keyword)
			// 如果有任何一个关键词不匹配，跳过当前文件
			if !keywordMatched {
				matchAllKeywords = false
				break
			}
		}
		// 只有当所有关键词都匹配时才添加文件
		if matchAllKeywords {
			resultWrapper.AddWrapperItem(file)
		}
	}

	return resultWrapper
}
