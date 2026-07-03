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
	if service.HandleRemoteByMovieEdit(c, currentFile, "rename") {
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
	if service.HandleRemoteByMovieEdit(c, currentFile, "move") {
		return
	}

	utils.InfoFormat("PostMove :[%v]", currentFile)
	res := UseApp().files.Move(currentFile.Id, currentFile.Path, currentFile.Title)
	c.JSON(http.StatusOK, res)
}

func GetAddTag(c *gin.Context) {
	if !requirePermission(c, "op:tag") {
		return
	}
	id := c.Param("id")
	tag := c.Param("tag")

	file := UseApp().search.FindById(id)
	if file.IsNull() {
		if service.HandleRemote(c, file, "addTag") {
			return
		}
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
		return
	}

	utils.InfoFormat("GetAddTag [%v] [%v]", id, tag)
	res := UseApp().files.AddTag(id, tag)
	c.JSON(http.StatusOK, res)
}

func GetClearTag(c *gin.Context) {
	if !requirePermission(c, "op:tag") {
		return
	}
	id := c.Param("id")
	tag := c.Param("tag")

	file := UseApp().search.FindById(id)
	if file.IsNull() {
		if service.HandleRemote(c, file, "clearTag") {
			return
		}
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
		return
	}

	res := UseApp().files.ClearTag(id, tag)
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

func GetDelete(c *gin.Context) {
	if !requirePermission(c, "op:edit") {
		return
	}
	id := c.Param("id")

	if service.HandleRemoteByID(c, id, "delete") {
		return
	}

	result := UseApp().files.Delete(id)
	if !result.IsSuccess() {
		c.JSON(http.StatusNotFound, result)
		return
	}
	c.JSON(http.StatusOK, result)
}
