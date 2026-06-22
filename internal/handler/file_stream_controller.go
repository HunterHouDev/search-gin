package handler

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

func GetRefreshTargetIndex(c *gin.Context) {
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
	cnt := UseApp().files.ScanAll()
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

	c.JSON(http.StatusOK, service.DeleteFileByPath(validatedPath))
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

// 默认占位图片数据
var (
	noPic       []byte
	contentType = "image/png"
)

func init() {
	var buf bytes.Buffer
	if err := generatePlaceholderPNG(&buf); err != nil {
		panic("初始化默认图片失败: " + err.Error())
	}
	noPic = buf.Bytes()
}

// generatePlaceholderPNG 生成一个简单的占位PNG图片
func generatePlaceholderPNG(w io.Writer) error {
	width, height := 200, 200
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	bgColor := color.RGBA{R: 204, G: 204, B: 204, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, bgColor)
		}
	}

	lineColor := color.RGBA{R: 153, G: 153, B: 153, A: 255}
	centerX, centerY := width/2, height/2
	thickness := 6
	size := 30

	for x := centerX - size; x <= centerX+size; x++ {
		for dy := -thickness / 2; dy <= thickness/2; dy++ {
			img.Set(x, centerY+dy, lineColor)
		}
	}
	for y := centerY; y <= centerY+size; y++ {
		for dx := -thickness / 2; dx <= thickness/2; dx++ {
			img.Set(centerX+dx, y, lineColor)
		}
	}
	for x := centerX - size + 10; x < centerX+size-10; x++ {
		for y := centerY - size; y < centerY-size+thickness; y++ {
			img.Set(x, y, lineColor)
		}
	}
	for x := centerX - size; x < centerX-size+thickness; x++ {
		for y := centerY - size; y < centerY; y++ {
			img.Set(x, y, lineColor)
		}
	}
	for x := centerX + size - thickness; x < centerX+size; x++ {
		for y := centerY - size; y < centerY; y++ {
			img.Set(x, y, lineColor)
		}
	}

	return png.Encode(w, img)
}
