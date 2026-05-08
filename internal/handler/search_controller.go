// controller 包包含所有HTTP请求的处理函数
// 这些函数负责接收请求、处理业务逻辑并返回响应
package handler

import (
	"search-gin/pkg/consts"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/utils"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

// PostSearch 文件搜索入口
// 此函数是文件搜索的统一入口点，内部重定向到PostMovies函数
// @Summary 搜索入口
// @Description 统一的搜索入口，转发到电影搜索处理
// @Router /api/search [post]
func PostSearch(c *gin.Context) {
	// 直接调用PostMovies函数进行实际的搜索处理
	PostMovies(c)
}

// PostMovies 电影文件搜索处理函数
// 接收搜索参数并调用搜索服务获取结果
// @Summary 电影搜索
// @Description 根据搜索参数查询电影文件信息
// @Accept json
// @Produce json
// @Router /api/search/movies [post]
func PostMovies(c *gin.Context) {
	// 初始化搜索参数结构体
	searchParam := model.SearchParam{}

	// 绑定HTTP请求体到结构体
	err := c.Bind(&searchParam)
	if err != nil {
		// 绑定失败时直接返回
		return
	}

	// 记录搜索请求日志
	utils.InfoFormat("PostMovies： [%v]", searchParam)

	// 调用搜索服务执行实际搜索操作
	result := service.SearchApp.SearchDataSource(searchParam)

	// 设置搜索完成进度状态
	result.SetProgress(consts.IndexDone)

	// 返回HTTP 200状态码和搜索结果
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
	// 初始化搜索参数结构体
	param := model.SearchParam{}

	// 绑定HTTP请求体到结构体
	err := c.Bind(&param)
	if err != nil {
		// 记录参数绑定错误日志
		utils.InfoNormal(param, err)
	}

	// 检查搜索引擎索引是否为空，如果为空则执行扫描
	if service.SearchEngin.IsEmpty() {
		// 检查是否已经有索引构建任务在执行
		if atomic.LoadInt32(&consts.IndexDone) == 0 {
			// 执行全量文件扫描以构建索引
			service.FileApp.ScanAll()
		}
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
