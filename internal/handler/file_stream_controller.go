package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

func GetRefreshTargetIndex(c *gin.Context) {
	if !requirePermission(c, "op:scan") {
		return
	}
	dir := c.Param("dir")
	baseDir, _ := url.QueryUnescape(dir)

	validatedDir, err := utils.ValidatePath(baseDir, UseApp().config.Get().Dirs)
	if err != nil {
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("路径不在允许范围内"))
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

func GetFileByPathUseEncode(c *gin.Context) {
	decodedPath, err := url.QueryUnescape(c.Param("path"))
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的文件路径"))
		return
	}

	validatedPath, err := utils.ValidatePath(decodedPath, UseApp().config.Get().Dirs)
	if err != nil {
		utils.ErrorFormat("路径遍历攻击尝试: %s, 错误: %v", decodedPath, err)
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("访问被拒绝：路径不在允许范围内"))
		return
	}

	if utils.ExistsFiles(validatedPath) {
		c.File(validatedPath)
	} else {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
	}
}

func GetDeleteFileByPathUseEncode(c *gin.Context) {
	if !requirePermission(c, "op:edit") {
		return
	}

	decodedPath, err := url.QueryUnescape(c.Param("path"))
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的文件路径"))
		return
	}

	validatedPath, err := utils.ValidatePath(decodedPath, UseApp().config.Get().Dirs)
	if err != nil {
		utils.ErrorFormat("路径遍历攻击尝试: %s, 错误: %v", decodedPath, err)
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("删除被拒绝：路径不在允许范围内"))
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

// placeholderPNGBase64 是预生成的 200x200 占位 PNG 图片 base64 编码
const placeholderPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAMgAAADICAIAAAAiOjnJAAACXUlEQVR4nOzbQUrGMBBAYZHeqPe/QXMntyJSRPKszP99y3aTxWMggTmu63qD3d6fPgAzCYuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi8TAFfvtbNP/golFQlgkhEVCWCSERUJYJIRFYuBzw/Z7+7evFcNeB7YzsUgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi4SwSAiLhLBICIuEsEgIi8TAFfvtbNP/golFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkhEVCWCSERUJYJIRFQlgkPgIAAP//W5kXM9bEb4kAAAAASUVORK5CYII="

func init() {
	var err error
	noPic, err = base64.StdEncoding.DecodeString(placeholderPNGBase64)
	if err != nil {
		panic("解码占位图片失败: " + err.Error())
	}
}
