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
	MTimeUnix int64 // Unix 时间戳，用于过滤器快速比较，避免每次查询都 time.Parse
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
	paged, _ := utils.SlicePage(files, pageNo, pageSize)
	var volume int64
	for _, f := range paged {
		volume += f.Size
	}
	return paged, volume
}

// RenameAll 重命名主文件 + 当前文件夹中所有同名附属文件，含回滚
// newMainPath: 主文件新路径
// newBaseName: 附属文件不含后缀的新基本名（如 "/path/to/newfile"）
// 返回改名后的新 FileItem（Id 不变），失败返回空 FileItem + error
func (f FileItem) RenameAll(newMainPath, newBaseName string) (FileItem, error) {
	originalDir := filepath.Dir(f.Path)
	originalBase := strings.TrimSuffix(filepath.Base(f.Path), "."+utils.GetSuffix(f.Path))

	files, err := os.ReadDir(originalDir)
	if err != nil {
		return FileItem{}, err
	}

	var originalPaths []string
	var newPaths []string

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if strings.EqualFold(strings.TrimSuffix(name, filepath.Ext(name)), originalBase) {
			originalPaths = append(originalPaths, filepath.Join(originalDir, name))
			suffix := utils.GetSuffix(name)
			newPaths = append(newPaths, newBaseName+"."+suffix)
		}
	}

	if len(originalPaths) == 0 {
		return FileItem{}, fmt.Errorf("no files found with base name %s", originalBase)
	}

	successIndices := make([]int, 0, len(originalPaths))
	for i := range originalPaths {
		if !utils.ExistsFiles(originalPaths[i]) {
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

// SetNodeInfo 设置文件所属节点信息
func (f *FileItem) SetNodeInfo(nodeHost, nodeName string) {
	f.NodeHost = nodeHost
	f.NodeName = nodeName
}
