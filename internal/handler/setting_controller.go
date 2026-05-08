package handler

import (
	"net/http"
	"os/exec"
	"search-gin/pkg/consts"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

func GetSettingInfo(c *gin.Context) {
	c.JSON(http.StatusOK, consts.OSSetting)
}
func PostSetting(c *gin.Context) {
	setInfo := model.Setting{}
	err := c.ShouldBindJSON(&setInfo)
	if err != nil {
		return
	}
	setInfo.SelfPath = consts.OSSetting.SelfPath
	consts.OSSetting = setInfo
	service.FlushDictionary(consts.OSSetting.SelfPath)
	res := utils.NewSuccess()
	c.JSON(http.StatusOK, res)
}

// GetIpAddr2 获取本地IP地址 利用udp
func GetIpAddr2(c *gin.Context) {
	res := utils.NewSuccess()
	res.Data = service.GetIpAddr()
	c.JSON(http.StatusOK, res)
}

// GetShutdown 系统关机
func GetShutdown(c *gin.Context) {
	res := utils.NewSuccess()
	err := exec.Command("cmd", "/C", "shutdown -s -t 0").Run()
	if err != nil {
		utils.InfoFormat("shutdown:%v", err)
	}
	c.JSON(http.StatusOK, res)
}
