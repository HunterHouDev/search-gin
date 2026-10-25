package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/utils"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// assetExts 静态资源的扩展名（脚本 / 样式 / 图片 / 字体 / 媒体）
var assetExts = map[string]struct{}{
	".js": {}, ".mjs": {}, ".css": {}, ".map": {}, ".json": {},
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".svg": {}, ".ico": {}, ".webp": {},
	".woff": {}, ".woff2": {}, ".ttf": {}, ".eot": {},
	".mp4": {}, ".webm": {}, ".mp3": {},
}

// isAssetPath 判断请求是否指向静态资源文件
func isAssetPath(path string) bool {
	_, ok := assetExts[strings.ToLower(filepath.Ext(path))]
	return ok
}

// Index SPA 入口，同时是 NoRoute 兜底。
//
// 兜底时不能无条件回 index.html：资源文件找不到却返回 HTML，浏览器会拿到一份
// text/html 的脚本，报
//
//	Failed to load module script: Expected a JavaScript module script but the
//	server responded with a MIME type of "text/html"
//
// 真正的错误（构建产物更新后旧页面还在引用已删除的 hash）会被这行报错完全掩盖。
// 所以：接口请求回 JSON 404，资源请求回文本 404，只有页面导航才回 index.html。
func Index(c *gin.Context) {
	path := c.Request.URL.Path

	if strings.HasPrefix(path, "/api/") {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("接口不存在: "+path))
		return
	}
	if isAssetPath(path) {
		utils.WarnFormat("静态资源不存在: %s（多为构建产物已更新，请强制刷新页面 Ctrl+F5）", path)
		c.Header("Cache-Control", "no-store")
		c.String(http.StatusNotFound, "404 not found: "+path)
		return
	}

	// 入口页不缓存：它是资源指纹的唯一入口，缓存住会在重新部署后继续引用已删除的文件
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.HTML(http.StatusOK, "index.html", gin.H{"title": "首页"})
}

const maxMapItems = 200

func GetTypeSize(c *gin.Context) {
	if UseApp().search.IsEmpty() {
		go func() {
			defer utils.RecoverPanic()
			UseApp().files.ScanAll()
		}()
	}
	res := mapToSlice(UseApp().search.GetTypeMenu())
	if len(res) > maxMapItems {
		res = res[:maxMapItems]
	}
	smallDirs := service.GetSmallDir()
	if len(smallDirs) > 0 {
		smallSize := model.NewFileInfo("小文件数量", int64(len(smallDirs)))
		smallSize.SizeStr = utils.GetSizeStr(smallSize.Size)
		res = append(res, smallSize)
		for i := range smallDirs {
			smallDirs[i].SizeStr = utils.GetSizeStr(smallDirs[i].Size)
			res = append(res, smallDirs[i])
		}
	}

	c.JSON(http.StatusOK, res)
}

func GetTagSize(c *gin.Context) {
	search := UseApp().search
	if search.IsEmpty() {
		c.JSON(http.StatusOK, utils.NewFailByMsg("未初始化"))
		return
	}
	res := mapToSlice(search.GetTagMenu())
	if len(res) > maxMapItems {
		res = res[:maxMapItems]
	}
	c.JSON(http.StatusOK, res)
}

func GetSeriesSize(c *gin.Context) {
	search := UseApp().search
	if search.IsEmpty() {
		c.JSON(http.StatusOK, utils.NewFailByMsg("未初始化"))
		return
	}
	res := mapToSlice(UseApp().search.GetSeriesCount())
	if len(res) > maxMapItems {
		res = res[:maxMapItems]
	}
	c.JSON(http.StatusOK, res)
}

func GetScanTime(c *gin.Context) {
	search := UseApp().search
	if search.IsEmpty() {
		c.JSON(http.StatusOK, utils.NewFailByMsg("未初始化"))
		return
	}
	res := make([]model.FileInfo, 0)
	service.GetFolderTime().Range(func(_, value interface{}) bool {
		if ms, ok := value.(model.FileInfo); ok {
			res = append(res, ms)
		}
		return true
	})

	sort.Slice(res, func(i, j int) bool {
		return res[i].Cnt > res[j].Cnt
	})
	c.JSON(http.StatusOK, res)
}

func GetLogMemory(c *gin.Context) {
	c.JSON(http.StatusOK, service.LogMem.GetAll())
}

func ClearMemoryLog(c *gin.Context) {
	service.LogMem.Clear()
	c.JSON(http.StatusOK, utils.NewSuccess())
}

func ClearLocalLog(c *gin.Context) {
	logPath := filepath.Join(service.GetWorkDir(), "gin.log")
	if err := os.Truncate(logPath, 0); err != nil {
		c.JSON(http.StatusOK, utils.NewFailByMsg("清空文件日志失败"))
		return
	}
	c.JSON(http.StatusOK, utils.NewSuccess())
}

type LocalLogLine struct {
	Raw string `json:"raw"`
}

func GetLocalLog(c *gin.Context) {
	logPath := filepath.Join(service.GetWorkDir(), "gin.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		c.JSON(http.StatusOK, []string{})
		return
	}
	lines := splitLines(string(content))
	// 过滤包含 token/Authorization 的敏感行
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, "token=") ||
			strings.Contains(line, "Authorization") ||
			strings.Contains(line, "Bearer ") {
			continue
		}
		filtered = append(filtered, line)
	}
	lines = filtered
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	c.JSON(http.StatusOK, lines)
}

func splitLines(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func GetHeartBeat(c *gin.Context) {
	c.JSON(http.StatusOK, service.IndexNumber.Load())
}

func GetDiskUsage(c *gin.Context) {
	var res []model.DiskStatus
	dirs := UseApp().config.Get().Dirs
	for _, dir := range dirs {
		usage, err := model.GetDiskUsage(dir)
		if err != nil {
			continue
		}
		res = append(res, *usage)
	}
	c.JSON(http.StatusOK, res)
}

func mapToSlice(m map[string]model.FileInfo) []model.FileInfo {
	var res []model.FileInfo
	for _, v := range m {
		res = append(res, v)
	}
	for i := 0; i < len(res); i++ {
		res[i].SizeStr = utils.GetSizeStr(res[i].Size)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Size > res[j].Size
	})
	return res
}
