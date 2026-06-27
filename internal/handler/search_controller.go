// controller 包包含所有HTTP请求的处理函数
// 这些函数负责接收请求、处理业务逻辑并返回响应
package handler

import (
	"net/http"
	"search-gin/internal/model"
	"search-gin/internal/service"
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

	if !isRemote && UseApp().search.IsEmpty() {
		// 异步触发扫描，不阻塞当前请求——用户发起首搜时无需等待扫描完成
		go func() {
			defer utils.RecoverPanic()
			UseApp().files.ScanAll()
		}()
	}

	searchParam := model.SearchParam{}
	err := c.Bind(&searchParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}

	if isRemote {
		result := UseApp().search.Page(searchParam)
		if data, ok := result.Data.([]model.FileItem); ok {
			service.FillURLs(c, data)
			result.Data = data
		} else {
			result.Data = []model.FileItem{}
		}
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
	result := UseApp().search.Page(searchParam)
	if data, ok := result.Data.([]model.FileItem); ok {
		service.FillURLs(c, data)
		result.Data = data
	} else {
		result.Data = []model.FileItem{}
	}
	c.JSON(http.StatusOK, result)
}

// PostAuthor 作者搜索处理函数
// 负责处理作者信息的搜索请求
// @Summary 作者搜索
// @Description 根据搜索参数查询作者信息
// @Accept json
// @Produce json
// @Router /api/search/authors [post]
func PostAuthor(c *gin.Context) {
	// 远程转发：只查本地，不递归，不触发扫描（避免跨节点递归死锁）
	if c.GetHeader("X-Search-Gin-Remote") == "true" {
		param := model.SearchParam{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
			return
		}
		pageAuthorResultWrapper := UseApp().search.PageAuthor(param)
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
	if UseApp().search.IsEmpty() {
		UseApp().files.ScanAll()
	}

	// 调用搜索引擎获取作者分页搜索结果
	pageAuthorResultWrapper := UseApp().search.PageAuthor(param)

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
