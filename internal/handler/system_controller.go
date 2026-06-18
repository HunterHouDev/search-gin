package handler

import (
	"encoding/json"
	"maps"
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

	// 先用 map 接收，只覆盖请求中存在的字段（不丢失现有配置）
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}

	// 对搜索相关字段做必填检查
	if dirs, ok := body["Dirs"]; ok {
		if arr, ok2 := dirs.([]any); ok2 && len(arr) == 0 {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("扫描目录不能为空"))
			return
		}
	}
	if types, ok := body["Types"]; ok {
		if arr, ok2 := types.([]any); ok2 && len(arr) == 0 {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("文件类型不能为空"))
			return
		}
	}

	// 将现有配置序列化为 map
	existing := service.GetOSSetting()
	existingJSON, _ := json.Marshal(existing)
	var merged map[string]any
	if err := json.Unmarshal(existingJSON, &merged); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewFailByMsg("合并配置失败"))
		return
	}

	// 用请求中的字段覆盖（只覆盖在 body 中存在的字段）
	maps.Copy(merged, body)

	// 将合并后的 map 反序列化回 Setting
	mergedJSON, _ := json.Marshal(merged)
	var updated model.Setting
	if err := json.Unmarshal(mergedJSON, &updated); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewFailByMsg("反序列化配置失败"))
		return
	}
	updated.SelfPath = existing.SelfPath

	service.SetOSSetting(updated)
	service.FlushDictionary(updated.SelfPath)
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
