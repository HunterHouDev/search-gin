package service

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
)

type searchService struct{}

// SearchDataSource 搜索数据源
func (fs *searchService) SearchDataSource(searchParam model.SearchParam) utils.Page {
	result := utils.NewPage()
	searchResult := SearchEngine.PageAsync(searchParam)
	result.TotalCnt = searchResult.SearchCount
	result.TotalSize = utils.GetSizeStr(searchResult.SearchSize)
	result.PageSize = searchParam.PageSize
	result.ResultSize = utils.GetSizeStr(searchResult.SearchSize)
	result.SetResultCnt(searchResult.SearchCount, searchParam.Page)
	result.CurSize = utils.GetSizeStr(searchResult.ResultSize)
	result.CurCnt = searchResult.ResultCount
	for i := range searchResult.FileList {
		searchResult.FileList[i].PageNo = searchParam.Page
	}
	result.Data = searchResult.FileList
	return result
}

// FindOne 查找单个文件
func (fs *searchService) FindOne(Id string) model.FileItem {
	return SearchEngine.FindById(Id)
}
