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

 if service.SearchEngin.IsEmpty() {
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
  movies, ok := result.Data.([]model.Movie)
  if ok {
   service.FillURLs(c, movies)
   result.Data = movies
  }
  result.SetProgress(consts.IndexNumber)
  c.JSON(http.StatusOK, result)
  return
 }

 // 前端请求：查本地 + 并发查所有在线节点
 localResult := service.SearchApp.SearchDataSource(searchParam)
 localMovies, ok := localResult.Data.([]model.Movie)
 if !ok {
  localMovies = []model.Movie{}
 }

 remoteMovies := service.SearchPeers(searchParam)
 merged := service.MergeResults(localMovies, remoteMovies)

 // 对合并结果重新分页
 pageMovies, total := service.PaginateMovies(merged, searchParam.Page, searchParam.PageSize)

 // 填充 URL
 service.FillURLs(c, pageMovies)

 // 构造返回结果
 result := utils.NewPage()
 result.PageNo = searchParam.Page
 result.PageSize = searchParam.PageSize
 result.TotalCnt = total
 result.ResultCnt = total
 result.CurCnt = len(pageMovies)
 result.Data = pageMovies
 result.SetProgress(consts.IndexNumber)

 c.JSON(http.StatusOK, result)
}

// PostActress 演员搜索处理函数
// 负责处理演员信息的搜索请求
// @Summary 演员搜索
// @Description 根据搜索参数查询演员信息
// @Accept json
// @Produce json
// @Router /api/search/actresses [post]
func PostActress(c *gin.Context) {
	// 远程转发：只查本地，不递归
	if c.GetHeader("X-Search-Gin-Remote") == "true" {
		param := model.SearchParam{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
			return
		}
		if service.SearchEngin.IsEmpty() {
			service.FileApp.ScanAll()
		}
		pageActressResultWrapper := service.SearchEngin.PageActress(param)
		result := utils.NewPage()
		result.CurCnt = pageActressResultWrapper.ResultCount
		result.TotalCnt = pageActressResultWrapper.SearchCount
		result.ResultCnt = pageActressResultWrapper.SearchCount
		result.Data = pageActressResultWrapper.FileList
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
	if service.SearchEngin.IsEmpty() {
		service.FileApp.ScanAll()
	}

	// 调用搜索引擎获取演员分页搜索结果
	pageActressResultWrapper := service.SearchEngin.PageActress(param)

	// 初始化分页结果对象
	result := utils.NewPage()

	// 设置分页相关数据
	result.CurCnt = pageActressResultWrapper.ResultCount    // 当前页结果数量
	result.TotalCnt = pageActressResultWrapper.SearchCount  // 总匹配数量
	result.ResultCnt = pageActressResultWrapper.SearchCount // 总结果数量
	result.Data = pageActressResultWrapper.FileList         // 结果数据列表

	// 返回HTTP 200状态码和搜索结果
	c.JSON(http.StatusOK, result)
}
