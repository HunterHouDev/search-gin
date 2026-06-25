package model

import (
	"fmt"
	"os"
	"path/filepath"
	"search-gin/pkg/utils"
	"sort"
	"strings"
	"time"
)

// FileItem 文件条目 — 表示一个被索引的文件
type FileItem struct {
	Id        string
	Code      string
	Name      string
	Path      string
	BaseDir   string
	Png       string
	Srt       string `json:"Srt,omitempty"`
	Jpg       string
	Gif       string
	Author    string
	FileType  string
	DirPath   string
	Size      int64
	Flag      int64
	SizeStr   string
	MTime     string
	MTimeUnix int64  // Unix 时间戳，用于过滤器快速比较，避免每次查询都 time.Parse
	MovieType string
	PathUpper string
	Tags      []string

	Studio string `json:"Studio,omitempty"`
	Title  string
	PngUrl string `json:"PngUrl,omitempty"`
	JpgUrl string `json:"JpgUrl,omitempty"`

	// 多节点字段
	NodeHost  string `json:"NodeHost,omitempty"`  // "PC-A:10081" 文件所属节点
	NodeName  string `json:"NodeName,omitempty"`  // "书房电脑" 节点可读别名
	StreamUrl string `json:"StreamUrl,omitempty"` // 文件流直连 URL

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
	fileKey := utils.DirpathForId(path)
	movieType := utils.GetMovieType(name)
	author := utils.GetAuthor(name)
	code := utils.GetCode(name)
	result := FileItem{
		Id:        fileKey,
		Code:      code,
		Title:     utils.GetTitle(name),
		Name:      name,
		Path:      path,
		PathUpper: strings.ToUpper(path),
		Png:       utils.ConcatSuffix(path, "png"),
		Jpg:       utils.ConcatSuffix(path, "jpg"),
		Srt:       utils.ConcatSuffix(path, "srt"),
		Gif:       utils.ConcatSuffix(path, "gif"),
		Tags:      utils.GetTags(path, ""),
		Author:    author,
		FileType:  fileType,
		DirPath:   dir,
		Size:      size,
		Flag:      1,
		Studio:    utils.GetSeriesByCode(code),
		SizeStr:   utils.GetSizeStr(size),
		MTime:     modTime.Format("2006-01-02 15:04:05"),
		MTimeUnix: modTime.Unix(),
		MovieType: movieType,
		BaseDir:   baseDir,
	}
	return result
}

// NewFile 创建文件条目（完整参数）
func NewFile(dir string, path string, name string, fileType string, size int64, modTime time.Time, movieType string, baseDir string) FileItem {
	generateId := utils.DirpathForId(path)
	code := utils.GetCode(name)
	author := utils.GetAuthor(name)
	result := FileItem{
		Id:        generateId,
		Code:      code,
		Title:     utils.GetTitle(name),
		Name:      name,
		Path:      path,
		PathUpper: strings.ToUpper(path),
		Png:       utils.ConcatSuffix(path, "png"),
		Jpg:       utils.ConcatSuffix(path, "jpg"),
		Srt:       utils.ConcatSuffix(path, "srt"),
		Gif:       utils.ConcatSuffix(path, "gif"),
		Tags:      utils.GetTags(path, ""),
		Author:    author,
		FileType:  fileType,
		DirPath:   dir,
		Size:      size,
		Flag:      1,
		Studio:    utils.GetSeriesByCode(code),
		SizeStr:   utils.GetSizeStr(size),
		MTime:     modTime.Format("2006-01-02 15:04:05"),
		MTimeUnix: modTime.Unix(),
		MovieType: movieType,
		BaseDir:   baseDir,
	}
	return result
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

// RenameAll 重命名主文件 + 附属图片 + 字幕文件（jpg/png/gif/srt），含回滚
// newMainPath: 主文件新路径
// newBaseName: 附属文件不含后缀的新基本名（如 "/path/to/newfile"）
// 返回改名后的新 FileItem（Id 不变），失败返回空 FileItem + error
func (f FileItem) RenameAll(newMainPath, newBaseName string) (FileItem, error) {
	originalPaths := []string{f.Path, f.Jpg, f.Png, f.Gif, f.Srt}
	newPaths := make([]string, 5)
	newPaths[0] = newMainPath
	if f.Jpg != "" {
		newPaths[1] = newBaseName + "." + utils.GetSuffix(f.Jpg)
	}
	if f.Png != "" {
		newPaths[2] = newBaseName + "." + utils.GetSuffix(f.Png)
	}
	if f.Gif != "" {
		newPaths[3] = newBaseName + "." + utils.GetSuffix(f.Gif)
	}
	if f.Srt != "" {
		newPaths[4] = newBaseName + "." + utils.GetSuffix(f.Srt)
	}

	successIndices := make([]int, 0, 5)
	for i := range originalPaths {
		if originalPaths[i] == "" || !utils.ExistsFiles(originalPaths[i]) {
			continue
		}
		if err := os.Rename(originalPaths[i], newPaths[i]); err != nil {
			utils.InfoFormat("rename failed: %v", err)
			for _, j := range successIndices {
				if utils.ExistsFiles(newPaths[j]) {
					if rerr := os.Rename(newPaths[j], originalPaths[j]); rerr != nil {
						utils.InfoFormat("rollback rename failed: %v", rerr)
					}
				}
			}
			return FileItem{}, err
		}
		successIndices = append(successIndices, i)
	}

	// 构建改名后的新 FileItem，Id 保持不变
	info, err := os.Stat(newMainPath)
	if err != nil {
		return FileItem{}, err
	}
	suffix := utils.GetSuffix(newMainPath)
	name := filepath.Base(newMainPath)
	newFile := EasyFile(filepath.Dir(newMainPath), newMainPath, name, suffix,
		info.Size(), info.ModTime(), f.BaseDir)
	newFile.Id = f.Id
	return newFile, nil
}
