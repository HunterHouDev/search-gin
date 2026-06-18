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
		service.SearchApp.ScanAll()
	}

	searchParam := model.SearchParam{}
	err := c.Bind(&searchParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}

	if isRemote {
		result := toPageResult(service.SearchEngine.PageAsync(searchParam), searchParam.Page)
		movies, ok := result.Data.([]model.FileItem)
		if ok {
			service.FillURLs(c, movies)
			result.Data = movies
		}
		result.SetProgress(consts.IndexNumber)
		c.JSON(http.StatusOK, result)
		return
	}

	// 前端请求
	if searchParam.SearchNode != "" {
		peer := service.GetPeer(searchParam.SearchNode)
		if peer == nil {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("节点不存在"))
			return
		}
		result, err := service.SearchRemotePeer(peer, searchParam)
		if err != nil {
			c.JSON(http.StatusBadGateway, utils.NewFailByMsg("远程搜索失败: "+err.Error()))
			return
		}
		c.JSON(http.StatusOK, result)
		return
	}

	// 搜索本机
	result := toPageResult(service.SearchEngine.PageAsync(searchParam), searchParam.Page)
	movies, ok := result.Data.([]model.FileItem)
	if ok {
		service.FillURLs(c, movies)
		result.Data = movies
	}
	result.SetProgress(consts.IndexNumber)
	c.JSON(http.StatusOK, result)
}

// PostAuthor 演员搜索处理函数
// 负责处理演员信息的搜索请求
// @Summary 演员搜索
// @Description 根据搜索参数查询演员信息
// @Accept json
// @Produce json
// @Router /api/search/authors [post]
func PostAuthor(c *gin.Context) {
	// 远程转发：只查本地，不递归
	if c.GetHeader("X-Search-Gin-Remote") == "true" {
		param := model.SearchParam{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
			return
		}
		if service.SearchEngine.IsEmpty() {
			service.SearchApp.ScanAll()
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
		service.SearchApp.ScanAll()
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

// toPageResult 将搜索引擎的 PageResultWrapper 转换为 API 响应 Page
func toPageResult(sr model.PageResultWrapper, pageNo int) utils.Page {
	result := utils.NewPage()
	result.TotalCnt = sr.SearchCount
	result.TotalSize = utils.GetSizeStr(sr.SearchSize)
	result.ResultSize = utils.GetSizeStr(sr.SearchSize)
	result.SetResultCnt(sr.SearchCount, pageNo)
	result.CurSize = utils.GetSizeStr(sr.ResultSize)
	result.CurCnt = sr.ResultCount
	for i := range sr.FileList {
		sr.FileList[i].PageNo = pageNo
	}
	result.Data = sr.FileList
	return result
}
