package model

import (
	"fmt"
	"search-gin/pkg/utils"
	"sort"
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

func SortMoviesUtils(sortModels []Movie, sF string, sT string) {
	sort.Slice(sortModels, func(i, j int) bool {
		switch sF {
		case "Code":
			if sT == "desc" {
				return sortModels[i].Code > sortModels[j].Code
			}
			return sortModels[i].Code < sortModels[j].Code
		case "Size":
			if sT == "desc" {
				return sortModels[i].Size > sortModels[j].Size
			}
			return sortModels[i].Size < sortModels[j].Size
		case "MTime":
			if sT == "desc" {
				return sortModels[i].MTime > sortModels[j].MTime
			}
			return sortModels[i].MTime < sortModels[j].MTime
		default:
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
	data := make([]Movie, 0, end-start)
	var volume int64
	for i := start; i < end; i++ {
		curFile := files[i]
		volume += curFile.Size
		data = append(data, curFile)
	}
	return data, volume
}
