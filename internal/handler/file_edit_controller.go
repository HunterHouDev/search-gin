package handler

import (
	"bytes"
	"io"
	"net/http"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

func SetMovieType(c *gin.Context) {
	if !requireAdmin(c) {
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
	if !requireAdmin(c) {
		return
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, 10<<20)) // 10MB 上限
	if err != nil {
		utils.InfoNormal(err)
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("读取请求体失败"))
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

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
	res := UseApp().files.Rename(currentFile)
	c.JSON(http.StatusOK, res)
}

func PostMove(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, 10<<20)) // 10MB 上限
	if err != nil {
		utils.InfoNormal(err)
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("读取请求体失败"))
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

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
	if !requireAdmin(c) {
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
	if !requireAdmin(c) {
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
	c.JSON(http.StatusOK, files)
}

func GetDelete(c *gin.Context) {
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
