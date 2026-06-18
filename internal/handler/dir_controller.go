package handler

import (
	"search-gin/internal/service"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetOpenFolder 本地打开文件夹
func GetOpenFolder(c *gin.Context) {
	id := c.Param("id")
	file := service.SearchEngine.FindById(id)

	if file.IsNull() {
		res := utils.NewFailByMsg("文件不存在")
		c.JSON(http.StatusNotFound, res)
		return
	}

	if service.HandleRemote(c, file, "openFolder") {
		return
	}

	validatedPath, err := utils.ValidatePath(file.DirPath, consts.GetOSSetting().Dirs)
	if err != nil {
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("路径不在允许范围内"))
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
	validatedPath, err := utils.ValidatePath(dirpath, consts.GetOSSetting().Dirs)
	if err != nil {
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("路径不在允许范围内"))
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
	validatedPath, err := utils.ValidatePath(dirpath, consts.GetOSSetting().Dirs)
	if err != nil {
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("路径不在允许范围内"))
		return
	}
	service.SearchApp.DownDeleteDir(validatedPath)
	res := utils.NewSuccessByMsg("删除成功")
	c.JSON(http.StatusOK, res)
}
