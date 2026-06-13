package model

import (
	"fmt"
	"search-gin/pkg/utils"
	"sort"
	"time"
)

// FileItem 文件条目 — 表示一个被索引的文件
type FileItem struct {
	Id        string `xorm:"Varchar(255) pk"  `
	Code      string `xorm:"Varchar(255)"`
	Name      string `xorm:"Text"`
	Path      string `xorm:"Text"`
	BaseDir   string `xorm:"Text"`
	Png       string `xorm:"Text"`
	Srt       string `xorm:"Text" json:"Srt,omitempty"`
	Jpg       string `xorm:"Text"`
	Gif       string `xorm:"Text"`
	Author   string `xorm:"Text"`
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

	// 多节点字段
	NodeHost  string `json:"nodeHost,omitempty"`  // "PC-A:10081" 文件所属节点
	NodeName  string `json:"nodeName,omitempty"`  // "书房电脑" 节点可读别名
	StreamUrl string `json:"streamUrl,omitempty"` // 文件流直连 URL

	PageNo int
}

// FileEdit 文件修改模型
type FileEdit struct {
	FileItem
	MoveOut   bool
	NoRefresh bool
}

// EasyFile 快速创建文件条目（无指定类型，自动推断）
func EasyFile(dir string, path string, name string, fileType string, size int64, modTime time.Time, baseDir string) FileItem {
	fileKey, _ := utils.DirpathForId(path)
	movieType := utils.GetMovieType(name)
	author := utils.GetAuthor(name)
	code := utils.GetCode(name)
	result := FileItem{
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
		Author:   author,
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

// NewFile 创建文件条目（完整参数）
func NewFile(dir string, path string, name string, fileType string, size int64, modTime time.Time, movieType string, baseDir string) FileItem {
	generateId, _ := utils.DirpathForId(path)
	code := utils.GetCode(name)
	author := utils.GetAuthor(name)
	result := FileItem{
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
		Author:   author,
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

func (f *FileItem) SetId(id string) FileItem {
	f.Id = id
	return *f
}

func (f FileItem) GetFileInfo() string {
	info := fmt.Sprintf("name: %v\t code:%v\t fileType:%v\t sizeStr:%v\t author:%v\t path:%v\t",
		f.Name, f.Code, f.FileType, f.SizeStr, f.Author, f.Path)
	return info
}

// IsNull 检查文件条目是否有效
func (f FileItem) IsNull() bool {
	if f.Id == "" || f.Path == "" {
		return true
	}
	return false
}

// SortFileItems 文件排序工具
func SortFileItems(sortModels []FileItem, sF string, sT string) {
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

// GetPageOfFiles 分页
func GetPageOfFiles(files []FileItem, pageNo int, pageSize int) ([]FileItem, int64) {
	if len(files) == 0 {
		return files, 0
	}
	if pageNo <= 0 {
		pageNo = 1
	}
	length := len(files)
	start := (pageNo - 1) * pageSize

	if start >= length {
		return []FileItem{}, 0
	}

	end := length
	if length-start >= pageSize {
		end = start + pageSize
	}
	data := make([]FileItem, 0, end-start)
	var volume int64
	for i := start; i < end; i++ {
		curFile := files[i]
		volume += curFile.Size
		data = append(data, curFile)
	}
	return data, volume
}
