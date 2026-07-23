package handler

import (
	"bytes"
	"io"
	"net/http"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/internal/sse"
	"search-gin/pkg/utils"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

var pendingRenameCount atomic.Int32

// readBodyTwice 读取请求体并重置，用于需要多次读取 body 的场景
func readBodyTwice(c *gin.Context) ([]byte, error) {
	bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, 10<<20))
	if err != nil {
		utils.InfoNormal(err)
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("读取请求体失败"))
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return bodyBytes, nil
}

func SetMovieType(c *gin.Context) {
	if !requirePermission(c, "op:movie:type") {
		return
	}
	id := c.Param("id")
	movieType := c.Param("movieType")
	file := UseApp().search.FindById(id)
	if file.IsNull() {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
		return
	}
	res := UseApp().files.SetMovieType(file, movieType)
	c.JSON(http.StatusOK, res)
}

func PostRename(c *gin.Context) {
	if !requirePermission(c, "op:edit") {
		return
	}
	bodyBytes, err := readBodyTwice(c)
	if err != nil {
		return
	}

	currentFile := model.FileEdit{}
	if err = c.ShouldBindJSON(&currentFile); err != nil {
		utils.InfoNormal(err)
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}

	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	if service.HandleRemote(c, currentFile.Host, "rename") {
		return
	}

	utils.InfoFormat("PostRename :searchCnt[%v]", currentFile)
	count := pendingRenameCount.Add(1)
	sse.BroadcastEvent(model.SSERenameStart, map[string]interface{}{
		"count": count,
	})
	res := UseApp().files.Rename(currentFile)
	count = pendingRenameCount.Add(-1)
	sse.BroadcastEvent(model.SSERenameStart, map[string]interface{}{
		"count": count,
	})
	c.JSON(http.StatusOK, res)
}

func PostMove(c *gin.Context) {
	if !requirePermission(c, "op:edit") {
		return
	}
	bodyBytes, err := readBodyTwice(c)
	if err != nil {
		return
	}

	currentFile := model.FileEdit{}
	if err = c.ShouldBindJSON(&currentFile); err != nil {
		utils.InfoNormal(err)
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}

	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	if service.HandleRemote(c, currentFile.Host, "move") {
		return
	}

	utils.InfoFormat("PostMove :[%v]", currentFile)
	res := UseApp().files.Move(currentFile.Id, currentFile.Path, currentFile.Title)
	c.JSON(http.StatusOK, res)
}

func PostAddTag(c *gin.Context) {
	if !requirePermission(c, "op:tag") {
		return
	}
	req, err := BindJSON[model.FileOpRequest](c)
	if err != nil {
		return
	}

	if service.HandleRemote(c, req.Host, "addTag") {
		return
	}

	utils.InfoFormat("PostAddTag [%v] [%v]", req.Id, req.Tag)
	res := UseApp().files.AddTag(req.Id, req.Tag)
	c.JSON(http.StatusOK, res)
}

func PostClearTag(c *gin.Context) {
	if !requirePermission(c, "op:tag") {
		return
	}
	req, err := BindJSON[model.FileOpRequest](c)
	if err != nil {
		return
	}

	if service.HandleRemote(c, req.Host, "clearTag") {
		return
	}

	res := UseApp().files.ClearTag(req.Id, req.Tag)
	c.JSON(http.StatusOK, res)
}

var Images = []string{"png", "jpg", "gif"}

func GetDirInfo(c *gin.Context) {
	id := c.Param("id")
	sort := c.Param("sort")
	file := UseApp().search.FindById(id)
	files := UseApp().files.Walk(file.DirPath, Images, false)
	model.SortFileItems(files, "MTime", sort)
	service.FillURLs(c, files)
	c.JSON(http.StatusOK, files)
}

func PostDelete(c *gin.Context) {
	if !requirePermission(c, "op:edit") {
		return
	}
	req, err := BindJSON[model.FileOpRequest](c)
	if err != nil {
		return
	}

	if service.HandleRemote(c, req.Host, "delete") {
		return
	}

	result := UseApp().files.Delete(req.Id)
	if !result.IsSuccess() {
		c.JSON(http.StatusNotFound, result)
		return
	}
	c.JSON(http.StatusOK, result)
}
