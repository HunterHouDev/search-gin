package handler

import (
	"net/http"
	"search-gin/internal/service"
	"search-gin/pkg/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetOpenFolder 本地打开文件夹
func GetOpenFolder(c *gin.Context) {
	id := c.Param("id")
	file := UseApp().search.FindById(id)

	if file.IsNull() {
		res := utils.NewFailByMsg("文件不存在")
		c.JSON(http.StatusNotFound, res)
		return
	}

	if service.HandleRemote(c, file, "openFolder") {
		return
	}

	validatedPath, ok := validatePathOrRespond(c, file.DirPath, "路径不在允许范围内")
	if !ok {
		return
	}

	utils.InfoFormat("open folder:[%v]", validatedPath)
	utils.ExecCmdStart(validatedPath)
	res := utils.NewSuccessByMsg("打开成功")
	c.JSON(http.StatusOK, res)
}

// PostOpenFolderByPath 通过路径打开文件夹
func PostOpenFolderByPath(c *gin.Context) {

	forms := make(map[string]string)
	err := c.ShouldBindJSON(&forms)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}
	dirpath := forms["dirpath"]
	dirpath = strings.ReplaceAll(dirpath, utils.PathSeparator+utils.PathSeparator, utils.PathSeparator)
	validatedPath, ok := validatePathOrRespond(c, dirpath, "路径不在允许范围内")
	if !ok {
		return
	}
	utils.ExecCmdStart(validatedPath)
	res := utils.NewSuccessByMsg("打开成功")
	c.JSON(http.StatusOK, res)
}

// PostDeleteFolderByPath 通过路径删除文件夹
func PostDeleteFolderByPath(c *gin.Context) {

	forms := make(map[string]string)
	err := c.ShouldBindJSON(&forms)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}
	dirpath := forms["dirpath"]
	dirpath = strings.ReplaceAll(dirpath, utils.PathSeparator+utils.PathSeparator, utils.PathSeparator)
	validatedPath, ok := validatePathOrRespond(c, dirpath, "路径不在允许范围内")
	if !ok {
		return
	}
	UseApp().files.DownDeleteDir(validatedPath)
	res := utils.NewSuccessByMsg("删除成功")
	c.JSON(http.StatusOK, res)
}
