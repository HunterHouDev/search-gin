package service

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

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

// GetPng 获取PNG图片
func (fs *fileService) GetPng(c *gin.Context) {
	id := c.Param("path")
	file := SearchApp.FindOne(id)
	if !file.IsNull() {
		if utils.ExistsFiles(file.Png) {
			c.File(file.Png)
			return
		} else if utils.ExistsFiles(file.Jpg) {
			c.File(file.Jpg)
			return
		} else if utils.ExistsFiles(file.Gif) {
			c.File(file.Gif)
			return
		}
	}
	fs.writeNoPic(c)
}

// GetJpg 获取JPG图片
func (fs *fileService) GetJpg(c *gin.Context) {
	id := c.Param("path")
	file := SearchApp.FindOne(id)
	if !file.IsNull() {
		// 按优先级检查图片文件
		jpeg := utils.ConcatSuffix(file.Path, "jpeg")
		if utils.ExistsFiles(file.Jpg) {
			c.File(file.Jpg)
			return
		} else if utils.ExistsFiles(jpeg) {
			c.File(jpeg)
			return
		} else if utils.ExistsFiles(file.Png) {
			c.File(file.Png)
			return
		} else if utils.ExistsFiles(file.Gif) {
			c.File(file.Gif)
			return
		}
	}
	fs.writeNoPic(c)
}

// GetFile 获取文件
func (fs *fileService) GetFile(c *gin.Context) {
	id := c.Param("id")
	file := SearchApp.FindOne(id)
	if utils.ExistsFiles(file.Path) {
		c.File(file.Path)
	} else {
		c.Status(http.StatusNotFound)
	}
}

// writeNoPic 无图时返回默认图片
func (fs *fileService) writeNoPic(c *gin.Context) {
	c.Data(http.StatusOK, contentType, noPic)
}

// generatePlaceholderPNG 生成一个简单的占位PNG图片
func generatePlaceholderPNG(w io.Writer) error {
	width, height := 200, 200
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 填充灰色背景
	bgColor := color.RGBA{R: 204, G: 204, B: 204, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, bgColor)
		}
	}

	// 绘制简单的 "?" 图标（十字形）
	lineColor := color.RGBA{R: 153, G: 153, B: 153, A: 255}
	centerX, centerY := width/2, height/2
	thickness := 6
	size := 30

	// 水平线
	for x := centerX - size; x <= centerX+size; x++ {
		for dy := -thickness / 2; dy <= thickness/2; dy++ {
			img.Set(x, centerY+dy, lineColor)
		}
	}
	// 竖直线（下半部分，形成 ? 的竖）
	for y := centerY; y <= centerY+size; y++ {
		for dx := -thickness / 2; dx <= thickness/2; dx++ {
			img.Set(centerX+dx, y, lineColor)
		}
	}
	// 顶部弧线简化为小方块
	for x := centerX - size + 10; x < centerX+size-10; x++ {
		for y := centerY - size; y < centerY-size+thickness; y++ {
			img.Set(x, y, lineColor)
		}
	}
	// 左侧弧线
	for x := centerX - size; x < centerX-size+thickness; x++ {
		for y := centerY - size; y < centerY; y++ {
			img.Set(x, y, lineColor)
		}
	}
	// 右侧弧线
	for x := centerX + size - thickness; x < centerX+size; x++ {
		for y := centerY - size; y < centerY; y++ {
			img.Set(x, y, lineColor)
		}
	}

	return png.Encode(w, img)
}
