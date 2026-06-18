package handler

import (
	"net/http"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

func PostMerge(c *gin.Context) {
	searchParam := model.MergeParam{}
	if err := c.Bind(&searchParam); err != nil {
		utils.InfoFormat("PostMerge 参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}
	utils.InfoFormat("PostMerge： [%v]", searchParam)
	c.JSON(http.StatusOK, service.CreateMergeTask(searchParam.Files, searchParam.Dest, searchParam.DeleteSource))
}

func GetTransferToMp4(c *gin.Context) {
	id := c.Param("id")
	if service.HandleRemoteByID(c, id, "transferToMp4") {
		return
	}

	xcode := c.Param("xcode")
	utils.InfoFormat("GetTransferToMp4 newFile [%v][%v]", id, xcode)
	c.JSON(http.StatusOK, service.CreateTransferTask(id, xcode))
}

func GetCutImage(c *gin.Context) {
	idInt := c.Param("id")
	typeImage := c.Param("typeImage")
	start := c.Param("start")

	if service.HandleRemoteByID(c, idInt, "cutImage") {
		return
	}

	movieFile := fileHandler.engine.FindById(idInt)
	if movieFile.IsNull() {
		r := utils.Fail()
		r.Message = "文件不存在"
		c.JSON(http.StatusOK, r)
		return
	}
	c.JSON(http.StatusOK, fileHandler.ve.CutImage(movieFile.Path, typeImage, start))
}

func GetCutMovie(c *gin.Context) {
	id := c.Param("id")
	if service.HandleRemoteByID(c, id, "cutMovie") {
		return
	}

	start := c.Param("start")
	end := c.Param("end")
	utils.InfoFormat("GetCutMovie [%v][%v][%v]", id, start, end)
	c.JSON(http.StatusOK, service.CreateCutTask(id, start, end))
}
