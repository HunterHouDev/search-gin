package handler

import (
	"net/http"
	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

// AppHandle handler 层入口结构体，所有依赖通过 InitApp 注入。
// 业务 handler 函数通过 UseApp() 获取实例调用：
//
//	UseApp().search.FindById(id)
//	UseApp().files.AddTag(id, tag)
//	UseApp().config.Get().Dirs
type AppHandle struct {
	search service.IndexEngine // 搜索引擎：分页搜索、按ID查找、获取菜单数据
	files  service.FileService // 文件操作：扫描、标签、重命名、移动、删除
	config service.Settings    // 配置管理：读写 setting.json
}

var appHandle *AppHandle

// InitApp 初始化全局 AppHandle，由 main.go 在创建所有依赖后显式调用。
// 参数是接口类型，调用方传入具体实现（searchEngineCore / searchService）。
func InitApp(search service.IndexEngine, files service.FileService, config service.Settings) {
	appHandle = &AppHandle{
		search: search,
		files:  files,
		config: config,
	}
}

// UseApp 返回全局 AppHandle。
// 所有 handler 函数通过此函数获取注入的依赖，不直接引用 service 包中的全局变量。
func UseApp() *AppHandle {
	return appHandle
}

// validatePathOrRespond 验证路径在允许目录范围内，失败时自动响应 403 并返回 false
func validatePathOrRespond(c *gin.Context, path, errMsg string) (string, bool) {
	validated, err := utils.ValidatePath(path, UseApp().config.Get().Dirs)
	if err != nil {
		c.JSON(http.StatusForbidden, utils.NewFailByMsg(errMsg))
		return "", false
	}
	return validated, true
}
