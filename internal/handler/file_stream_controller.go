package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"search-gin/internal/service"
	"search-gin/pkg/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetRefreshTargetIndex(c *gin.Context) {
	if !requirePermission(c, "op:scan") {
		return
	}
	// rawPathParam 已按路径语义解过码，这里不能再解一次（见其注释）
	baseDir, err := rawPathParam(c, "dir")
	if err != nil {
		baseDir = c.Param("dir")
	}

	validatedDir, ok := validatePathOrRespond(c, baseDir, "路径不在允许范围内")
	if !ok {
		return
	}

	UseApp().files.ScanTarget(validatedDir)
	c.JSON(http.StatusOK, utils.NewSuccessByMsg("扫描任务执行中"))
}

func GetRefreshIndex(c *gin.Context) {
	if !requirePermission(c, "op:scan") {
		return
	}
	cnt := len(UseApp().config.Get().Dirs)
	go UseApp().files.ScanAll()
	c.JSON(http.StatusOK, utils.NewSuccessByMsg("计划扫描："+fmt.Sprint(cnt)))
}

// rawPathParam 取路由参数的原始值，再按「路径」语义解码一次。
//
// 不能直接用 c.Param + url.QueryUnescape：
//  1. net/http 的 URL.Path 已经解过一次码，再解一次遇到 `%`（如 `50%.mp4`）会报错；
//  2. gin 解码参数是 QueryUnescape 语义，`+` 会被当成空格——
//     文件名里的 `+`（前端编码成 %2B）会变成空格，于是找不到文件、返回 404。
//
// 这里从原始 URL（仍未解码）里取最后一段，用 PathUnescape 解码，
// `%2B` → `+`、`%25` → `%` 都能还原成文件名本身；取不到时回落到 c.Param。
func rawPathParam(c *gin.Context, key string) (string, error) {
	raw := c.Request.URL.EscapedPath()
	if idx := strings.LastIndex(raw, "/"); idx >= 0 {
		raw = raw[idx+1:]
	}
	if raw == "" {
		// 没有真实 URL 段（例如单元测试直接调用 handler）：回落到路由参数，
		// 仍按路径语义解码，非法转义照样报错
		return url.PathUnescape(c.Param(key))
	}
	return url.PathUnescape(raw)
}

func GetFileByPathUseEncode(c *gin.Context) {
	decodedPath, err := rawPathParam(c, "path")
	if err != nil || strings.TrimSpace(decodedPath) == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的文件路径"))
		return
	}

	validatedPath, ok := validatePathOrRespond(c, decodedPath, "访问被拒绝：路径不在允许范围内")
	if !ok {
		return
	}

	if utils.ExistsFiles(validatedPath) {
		c.File(validatedPath)
		return
	}
	// 带上解析后的路径：下载回放 404 时能一眼看出是地址被解错了还是文件真的没了
	utils.InfoFormat("GetFileByPathUseEncode: 文件不存在 path=%s", validatedPath)
	c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在: "+validatedPath))
}

// GetFileExists 校验服务端上的文件是否仍然存在。
//
// 下载列表回放前先问一次：下载完成后文件可能已被移走或删除，
// 直接起播只会得到一个打不开的流，先提示使用者更清楚。
// 路径校验与文件流一致（必须在已配置的媒体目录内）。
func GetFileExists(c *gin.Context) {
	rawPath := c.Query("path")
	if strings.TrimSpace(rawPath) == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("缺少文件路径"))
		return
	}

	validatedPath, ok := validatePathOrRespond(c, rawPath, "访问被拒绝：路径不在允许范围内")
	if !ok {
		return
	}

	result := utils.NewSuccess()
	result.Data = map[string]interface{}{
		"exists": utils.ExistsFiles(validatedPath),
		"path":   validatedPath,
	}
	c.JSON(http.StatusOK, result)
}

func GetDeleteFileByPathUseEncode(c *gin.Context) {
	if !requirePermission(c, "op:edit") {
		return
	}

	// 与回放同一套解码规则：文件名里的 % / + 才能还原（见 rawPathParam 注释）
	decodedPath, err := rawPathParam(c, "path")
	if err != nil || strings.TrimSpace(decodedPath) == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的文件路径"))
		return
	}

	validatedPath, ok := validatePathOrRespond(c, decodedPath, "删除被拒绝：路径不在允许范围内")
	if !ok {
		return
	}

	if !utils.ExistsFiles(validatedPath) {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
		return
	}

	c.JSON(http.StatusOK, service.DeleteIndexByPath(validatedPath))
}

func GetFile(c *gin.Context) {
	id := c.Param("id")
	file := UseApp().search.FindById(id)
	if file.Path != "" {
		if validated, err := utils.ValidatePath(file.Path, UseApp().config.Get().Dirs); err == nil {
			c.File(validated)
		} else {
			c.Status(http.StatusForbidden)
		}
	} else {
		c.Status(http.StatusNotFound)
	}
}

func GetPng(c *gin.Context) {
	id := c.Param("path")
	file := UseApp().search.FindById(id)
	if !file.IsNull() {
		for _, candidate := range []string{file.Png, file.Jpg, file.Gif} {
			if candidate != "" {
				if validated, err := utils.ValidatePath(candidate, UseApp().config.Get().Dirs); err == nil && utils.ExistsFiles(validated) {
					c.File(validated)
					return
				}
			}
		}
	}
	c.Data(http.StatusOK, contentType, noPic)
}

func GetJpg(c *gin.Context) {
	id := c.Param("path")
	file := UseApp().search.FindById(id)
	if !file.IsNull() {
		jpeg := utils.ConcatSuffix(file.Path, "jpeg")
		for _, candidate := range []string{file.Jpg, jpeg, file.Png, file.Gif} {
			if candidate != "" {
				if validated, err := utils.ValidatePath(candidate, UseApp().config.Get().Dirs); err == nil && utils.ExistsFiles(validated) {
					c.File(validated)
					return
				}
			}
		}
	}
	c.Data(http.StatusOK, contentType, noPic)
}

// 默认占位图片数据（预生成的 base64 编码 PNG）
var (
	noPic       []byte
	contentType = "image/png"
)

// placeholderPNGBase64 是预生成的 200x200 全透明占位 PNG 图片 base64 编码
const placeholderPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAMgAAADICAIAAAAiOjnJAAAACXBIWXMAAAABAAAAAQBPJcTWAAAAiklEQVR4nO3BAQEAAACCIP+vbkhAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADwYNWXAAG9rB+hAAAAAElFTkSuQmCC"

func init() {
	var err error
	noPic, err = base64.StdEncoding.DecodeString(placeholderPNGBase64)
	if err != nil {
		panic("解码占位图片失败: " + err.Error())
	}
}
