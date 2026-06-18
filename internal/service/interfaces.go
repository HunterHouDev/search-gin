package service

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
)

type SearchEngineInterface interface {
	Page(searchParam model.SearchParam) utils.Page
	PageAuthor(searchParam model.SearchParam) model.PageAuthorResultWrapper
	FindById(id string) model.FileItem
	FindAuthorByName(name string) model.Author
	GetAuthorCount() int
	IsEmpty() bool
	GetTotalCount() int
	GetTotalSize() int64
	BucketCount() int32
	DeleteFile(file model.FileItem)
	ReplaceFile(oldFile, newFile model.FileItem)
}

type FileServiceInterface interface {
	SetMovieType(movie model.FileItem, movieType string) utils.Result
	AddTag(id string, tag string) utils.Result
	ClearTag(id string, tag string) utils.Result
	Rename(movie model.FileEdit) utils.Result
	Move(id string, newDir string, title string) utils.Result
	Delete(id string)
	ScanAll() int
	ScanTarget(baseDir string)
	Walk(dir string, types []string, withSub bool) []model.FileItem
	DeleteOne(dirName string, fileName string)
	DownDeleteDir(dirname string)
}

type VideoEncoderInterface interface {
	CutImage(path string, typeImage string, start string) utils.Result
	TransferFormatter(task model.TransferTaskModel) utils.Result
	CutFormatter(task model.TransferTaskModel) utils.Result
	MergeFiles(task model.TransferTaskModel) utils.Result
}
