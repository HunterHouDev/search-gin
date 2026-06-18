package handler

import (
	"bytes"
	"io"
	"net/http"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

func SetMovieType(c *gin.Context) {
	id := c.Param("id")
	movieType := c.Param("movieType")
	file := fileHandler.engine.FindById(id)
	res := fileHandler.fileSvc.SetMovieType(file, movieType)
	c.JSON(http.StatusOK, res)
}

func PostRename(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
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
	res := fileHandler.fileSvc.Rename(currentFile)
	c.JSON(http.StatusOK, res)
}

func PostMove(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
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
	res := fileHandler.fileSvc.Move(currentFile.Id, currentFile.Path, currentFile.Title)
	c.JSON(http.StatusOK, res)
}

func GetAddTag(c *gin.Context) {
	id := c.Param("id")
	tag := c.Param("tag")

	file := fileHandler.engine.FindById(id)
	if file.IsNull() {
		if service.HandleRemote(c, file, "addTag") {
			return
		}
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
		return
	}

	utils.InfoFormat("GetAddTag [%v] [%v]", id, tag)
	res := fileHandler.fileSvc.AddTag(id, tag)
	c.JSON(http.StatusOK, res)
}

func GetClearTag(c *gin.Context) {
	id := c.Param("id")
	tag := c.Param("tag")

	file := fileHandler.engine.FindById(id)
	if file.IsNull() {
		if service.HandleRemote(c, file, "clearTag") {
			return
		}
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
		return
	}

	res := fileHandler.fileSvc.ClearTag(id, tag)
	c.JSON(http.StatusOK, res)
}

func GetDirInfo(c *gin.Context) {
	id := c.Param("id")
	sort := c.Param("sort")
	file := fileHandler.engine.FindById(id)
	files := fileHandler.fileSvc.Walk(file.DirPath, consts.Images, false)
	model.SortFileItems(files, "MTime", sort)
	c.JSON(http.StatusOK, files)
}

func GetDelete(c *gin.Context) {
	id := c.Param("id")

	file := fileHandler.engine.FindById(id)
	if file.IsNull() {
		if service.HandleRemote(c, file, "delete") {
			return
		}
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
		return
	}

	fileHandler.engine.DeleteFile(file)
	fileHandler.fileSvc.DeleteOne(file.DirPath, file.Title)
	c.JSON(http.StatusOK, utils.NewSuccessByMsg("删除成功"))
}
