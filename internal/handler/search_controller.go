// controller 包包含所有HTTP请求的处理函数
// 这些函数负责接收请求、处理业务逻辑并返回响应
package handler

import (
	"net/http"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

// PostMovies 电影文件搜索处理函数
// 接收搜索参数并调用搜索服务获取结果
// @Summary 电影搜索
// @Description 根据搜索参数查询电影文件信息
// @Accept json
// @Produce json
// @Router /api/search/movies [post]
func PostMovies(c *gin.Context) {
	// 检查是否为远程转发请求（X-Search-Gin-Remote: true）
	isRemote := c.GetHeader("X-Search-Gin-Remote") == "true"

	if service.SearchEngine.IsEmpty() {
		service.FileApp.ScanAll()
	}

	searchParam := model.SearchParam{}
	err := c.Bind(&searchParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}

	if isRemote {
		// 远程转发：只返回本地结果，不递归搜索其他节点
		result := service.SearchApp.SearchDataSource(searchParam)
		movies, ok := result.Data.([]model.FileItem)
		if ok {
			service.FillURLs(c, movies)
			result.Data = movies
		}
		result.SetProgress(consts.IndexNumber)
		c.JSON(http.StatusOK, result)
		return
	}

	// 前端请求：根据 SearchMode 决定搜索范围
	var merged []model.FileItem
	var remoteTotalCnt int
	var remoteTotalSize int64
	var localResult utils.Page

	searchMode := searchParam.SearchMode
	doLocal := searchMode == "" || searchMode == "mixed" || searchMode == "local"
	doRemote := service.IsClusterEnabled() && (searchMode == "" || searchMode == "mixed" || searchMode == "remote")

	if doLocal {
		// 本地搜索获取全量结果，再由最终 PaginateMovies 统一分页
		fullParam := searchParam
		fullParam.Page = 1
		fullParam.PageSize = 99999
		localResult = service.SearchApp.SearchDataSource(fullParam)
		localMovies, ok := localResult.Data.([]model.FileItem)
		if !ok {
			localMovies = []model.FileItem{}
		}
		if doRemote {
			remoteMovies, rCnt, rSize := service.SearchPeers(searchParam)
			merged = service.MergeResults(localMovies, remoteMovies)
			remoteTotalCnt = rCnt
			remoteTotalSize = rSize
		} else {
			merged = localMovies
		}
	} else if doRemote {
		// 仅远程搜索
		remoteMovies, rCnt, rSize := service.SearchPeers(searchParam)
		merged = remoteMovies
		remoteTotalCnt = rCnt
		remoteTotalSize = rSize
		localResult = utils.NewPage()
	} else {
		merged = []model.FileItem{}
		localResult = utils.NewPage()
	}

	// 对合并结果重新分页
	pageMovies, _ := service.PaginateMovies(merged, searchParam.Page, searchParam.PageSize)

	// 计算当前页大小
	var curSize int64
	for _, m := range pageMovies {
		curSize += m.Size
	}

	// 填充 URL
	service.FillURLs(c, pageMovies)

	// 构造返回结果：TotalCnt = 本地总数 + 远程节点总数
	totalCnt := localResult.TotalCnt + remoteTotalCnt
	totalSize := service.ParseTotalSize(localResult.TotalSize) + remoteTotalSize

	result := utils.NewPage()
	result.PageNo = searchParam.Page
	result.PageSize = searchParam.PageSize
	result.TotalCnt = totalCnt
	result.TotalSize = utils.GetSizeStr(totalSize)
	result.ResultSize = utils.GetSizeStr(totalSize)
	result.ResultCnt = totalCnt
	result.CurSize = utils.GetSizeStr(curSize)
	result.CurCnt = len(pageMovies)
	result.Data = pageMovies
	result.SetResultCnt(totalCnt, searchParam.Page)
	result.SetProgress(consts.IndexNumber)

	c.JSON(http.StatusOK, result)
}

// PostAuthor 演员搜索处理函数
// 负责处理演员信息的搜索请求
// @Summary 演员搜索
// @Description 根据搜索参数查询演员信息
// @Accept json
// @Produce json
// @Router /api/search/actresses [post]
func PostAuthor(c *gin.Context) {
	// 远程转发：只查本地，不递归
	if c.GetHeader("X-Search-Gin-Remote") == "true" {
		param := model.SearchParam{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
			return
		}
		if service.SearchEngine.IsEmpty() {
			service.FileApp.ScanAll()
		}
		pageAuthorResultWrapper := service.SearchEngine.PageAuthor(param)
		result := utils.NewPage()
		result.CurCnt = pageAuthorResultWrapper.ResultCount
		result.TotalCnt = pageAuthorResultWrapper.SearchCount
		result.ResultCnt = pageAuthorResultWrapper.SearchCount
		result.Data = pageAuthorResultWrapper.FileList
		c.JSON(http.StatusOK, result)
		return
	}

	// 初始化搜索参数结构体
	param := model.SearchParam{}

	// 绑定HTTP请求体到结构体
	err := c.Bind(&param)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}

	// 检查搜索引擎索引是否为空，如果为空则执行扫描
	if service.SearchEngine.IsEmpty() {
		service.FileApp.ScanAll()
	}

	// 调用搜索引擎获取演员分页搜索结果
	pageAuthorResultWrapper := service.SearchEngine.PageAuthor(param)

	// 初始化分页结果对象
	result := utils.NewPage()

	// 设置分页相关数据
	result.CurCnt = pageAuthorResultWrapper.ResultCount    // 当前页结果数量
	result.TotalCnt = pageAuthorResultWrapper.SearchCount  // 总匹配数量
	result.ResultCnt = pageAuthorResultWrapper.SearchCount // 总结果数量
	result.Data = pageAuthorResultWrapper.FileList         // 结果数据列表

	// 返回HTTP 200状态码和搜索结果
	c.JSON(http.StatusOK, result)
}
