package handler

import (
	"net/http"
	"os"
	"path/filepath"
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

	// 计算目录总大小，超过 20MB 提前退出
	var totalSize int64
	const maxDeleteSize int64 = 20 * 1024 * 1024
	filepath.WalkDir(validatedPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			if info, e := d.Info(); e == nil {
				totalSize += info.Size()
				if totalSize > maxDeleteSize {
					return filepath.SkipAll
				}
			}
		}
		return nil
	})
	if totalSize > maxDeleteSize {
		c.JSON(http.StatusOK, utils.NewFailByMsg("目录超过 20MB，请手动删除"))
		return
	}

	UseApp().files.DownDeleteDir(validatedPath)
	res := utils.NewSuccessByMsg("删除成功")
	c.JSON(http.StatusOK, res)
}
