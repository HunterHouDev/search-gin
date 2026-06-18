package handler

import (
	"net/http"
	"os/exec"
	"strings"

	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

func GetSettingInfo(c *gin.Context) {
	setting := service.GetOSSetting()
	safeSetting := setting
	safeSetting.Users = nil
	if safeSetting.HardwareAcceleration && safeSetting.HardwareAccelMode == "" {
		safeSetting.HardwareAccelMode = service.GetHwAccelModeName()
	}
	c.JSON(http.StatusOK, safeSetting)
}

func PostSetting(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	setInfo := model.Setting{}
	if err := c.ShouldBindJSON(&setInfo); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}
	setInfo.SelfPath = service.GetOSSetting().SelfPath
	service.SetOSSetting(setInfo)
	service.FlushDictionary(service.GetOSSetting().SelfPath)
	if service.HwAccelSettingChanged() {
		service.ForceHwAccelDetect()
	}
	c.JSON(http.StatusOK, utils.NewSuccess())
}

func GetIpAddr2(c *gin.Context) {
	res := utils.NewSuccess()
	res.Data = service.GetIpAddr()
	c.JSON(http.StatusOK, res)
}

func GetServerPort(c *gin.Context) {
	setting := service.GetOSSetting()
	configured := setting.ControllerHost
	if configured == "" {
		configured = consts.PortNo
	}
	cfgPort := configured
	idx := strings.LastIndex(cfgPort, ":")
	if idx >= 0 {
		cfgPort = cfgPort[idx:]
	}
	runningPort := consts.PortNo
	changed := cfgPort != runningPort
	c.JSON(http.StatusOK, gin.H{
		"runningPort":    runningPort,
		"configuredPort": cfgPort,
		"changed":        changed,
	})
}

func GetShutdown(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	err := exec.Command("cmd", "/C", "shutdown -s -t 0").Run()
	if err != nil {
		utils.InfoFormat("shutdown:%v", err)
	}
	c.JSON(http.StatusOK, utils.NewSuccess())
}
