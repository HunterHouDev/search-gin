package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

func GetSettingInfo(c *gin.Context) {
	setting := UseApp().config.Get()
	safeSetting := setting
	safeSetting.Users = nil
	safeSetting.AdminPassword = "" // 密码不返回前端
	if safeSetting.HardwareAcceleration && safeSetting.HardwareAccelMode == "" {
		safeSetting.HardwareAccelMode = service.GetHwAccelModeName()
	}

	// 序列化为 map 后添加可用硬件加速方案列表（该字段不存入 setting.json）
	b, _ := json.Marshal(safeSetting)
	var result map[string]any
	json.Unmarshal(b, &result)
	result["AvailableHwAccelModes"] = service.GetAvailableHwAccelModes()

	c.JSON(http.StatusOK, result)
}

func PostSetting(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	// 先用 map 接收，只覆盖请求中存在的字段（不丢失现有配置）
	body, err := BindJSON[map[string]any](c)
	if err != nil {
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

	// 白名单过滤：只允许修改可安全暴露的字段，禁止通过 API 修改密钥/密码/用户列表
	// 敏感字段（AdminPassword, Users 等）只能通过 setting.json 直接修改
	allowedFields := map[string]bool{
		"Dirs": true, "Types": true, "VideoTypes": true, "ImageTypes": true,
		"DocsTypes": true, "MovieTypes": true, "Tags": true, "Pages": true,
		"ControllerHost": true, "FileHost": true,
		"NodeName": true, "enableLanDiscovery": true, "discoveryPeers": true,
		"SystemPlayerVolumn": true, "SystemPlayerWidth": true, "SystemPlayer": true,
		"HardwareAcceleration": true, "HardwareAccelMode": true,
		"taskMaxConcurrent": true,
		"TagsLib": true, "DirsLib": true,
		"EnableTimeScan": true, "CutThenDelete": true,
		"BaseUrl": true, "ImageUrl": true, "Remark": true,
	}
	for key := range body {
		if !allowedFields[key] {
			delete(body, key)
		}
	}

	// 将 body map 序列化后反序列化到现有 struct 上
	// 不在 body 中的字段保持不变（Go json.Unmarshal 特性）
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("序列化配置失败"))
		return
	}
	var updated = UseApp().config.Get()
	if err := json.Unmarshal(bodyJSON, &updated); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewFailByMsg("反序列化配置失败"))
		return
	}

	UseApp().config.Set(updated)
	UseApp().config.Flush(service.SettingFileName)
	if service.HwAccelSettingChanged() {
		service.ForceHwAccelDetect()
	}
	// HardwareAccelMode 变更也触发重新检测
	if service.HwAccelModeChanged() {
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
	setting := UseApp().config.Get()
	configured := setting.ControllerHost
	if configured == "" {
		configured = service.PortNo
	}
	cfgPort := configured
	idx := strings.LastIndex(cfgPort, ":")
	if idx >= 0 {
		cfgPort = cfgPort[idx:]
	}
	runningPort := service.PortNo
	changed := cfgPort != runningPort
	res := utils.NewSuccess()
	res.Data = gin.H{
		"runningPort":    runningPort,
		"configuredPort": cfgPort,
		"changed":        changed,
	}
	c.JSON(http.StatusOK, res)
}

func GetShutdown(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	service.ShutdownSystem()
	c.JSON(http.StatusOK, utils.NewSuccess())
}

// PostShutdownSchedule 设置定时关机（秒）
func PostShutdownSchedule(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	body, err := BindJSON[struct {
		Seconds int `json:"seconds"`
	}](c, "参数无效")
	if err != nil {
		return
	}
	if body.Seconds <= 0 {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("秒数必须大于 0"))
		return
	}
	service.ScheduleShutdown(body.Seconds)
	c.JSON(http.StatusOK, utils.NewSuccess())
}

// PostShutdownCancel 取消定时关机
func PostShutdownCancel(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	service.CancelShutdown()
	c.JSON(http.StatusOK, utils.NewSuccess())
}

// GetShutdownStatus 获取当前剩余秒数（供前端初始化时同步）
func GetShutdownStatus(c *gin.Context) {
	remaining := service.GetShutdownRemaining()
	if remaining <= 0 {
		c.JSON(http.StatusOK, gin.H{"remaining": 0})
		return
	}
	c.JSON(http.StatusOK, gin.H{"remaining": remaining})
}
