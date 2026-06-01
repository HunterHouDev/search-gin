package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os/exec"
	"search-gin/pkg/consts"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login 用户登录
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的请求"))
		return
	}
	
	// 验证用户凭据
	validUser := false
	userRole := ""
	userExpireDate := ""
	for _, user := range consts.OSSetting.Users {
		if user.Username == req.Username && user.Password == req.Password {
			validUser = true
			userRole = user.Role
			userExpireDate = user.ExpireDate
			break
		}
	}
	
	if !validUser {
		c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("用户名或密码错误"))
		return
	}
	
	// 检查用户是否过期
	if userExpireDate != "" {
		expireTime, err := time.Parse("2006-01-02", userExpireDate)
		if err == nil && time.Now().After(expireTime) {
			c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("用户已过期，请联系管理员"))
			return
		}
	}
	
	// 生成简单token（基于时间戳和随机数）
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewFailByMsg("生成token失败，系统错误"))
		return
	}
	token := hex.EncodeToString(tokenBytes)
	
	// 存储token到内存（简单实现，生产环境应使用Redis等）
	consts.SetToken(token, time.Now().Add(2*time.Hour), req.Username, userRole)
	
	// 返回成功响应，包含token和角色
	res := utils.NewSuccess()
	res.Data = gin.H{
		"token"    : token,
		"expireIn" : 2 * 3600, // 2小时过期
		"role"     : userRole,
		"username" : req.Username,
	}
	c.JSON(http.StatusOK, res)
}

// 在consts包中添加token存储
// 这里简单存储在内存中

func GetSettingInfo(c *gin.Context) {
	setting := consts.GetOSSetting()
	safeSetting := setting
	safeSetting.Users = nil
	// 如果启用硬件加速且已检测到模式，同步给前端展示
	if safeSetting.HardwareAcceleration && safeSetting.HardwareAccelMode == "" {
		safeSetting.HardwareAccelMode = service.GetHwAccelModeName()
	}
	c.JSON(http.StatusOK, safeSetting)
}
func PostSetting(c *gin.Context) {
	setInfo := model.Setting{}
	err := c.ShouldBindJSON(&setInfo)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}
	setInfo.SelfPath = consts.GetOSSetting().SelfPath
	consts.SetOSSetting(setInfo)
	service.FlushDictionary(consts.GetOSSetting().SelfPath)
	// 如果硬件加速开关发生变化，强制重新检测
	if service.HwAccelSettingChanged() {
		service.ForceHwAccelDetect()
	}
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

// AddUser 添加用户（仅超管可操作）
func AddUser(c *gin.Context) {
	type AddUserRequest struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		Role       string `json:"role"`
		ExpireDate string `json:"expireDate"` // 有效期，格式：2006-01-02，空字符串表示永不过期
	}
	
	var req AddUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的请求"))
		return
	}
	
	// 验证角色
	if req.Role != "super_admin" && req.Role != "user" {
		req.Role = "user" // 默认普通用户
	}
	
	// 验证有效期格式（如果提供）
	if req.ExpireDate != "" {
		_, err := time.Parse("2006-01-02", req.ExpireDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("有效期格式错误，应为YYYY-MM-DD"))
			return
		}
	}
	
	// 检查用户名是否已存在
	for _, user := range consts.OSSetting.Users {
		if user.Username == req.Username {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("用户名已存在"))
			return
		}
	}
	
	// 添加用户
	newUser := model.User{
		Username:   req.Username,
		Password:   req.Password,
		Role:       req.Role,
		ExpireDate: req.ExpireDate,
	}
	
	// 更新配置
	setting := consts.GetOSSetting()
	setting.Users = append(setting.Users, newUser)
	consts.SetOSSetting(setting)
	
	// 持久化到文件
	service.FlushDictionary(setting.SelfPath)
	
	c.JSON(http.StatusOK, utils.NewSuccess())
}

// DeleteUser 删除用户（仅超管可操作）
func DeleteUser(c *gin.Context) {
	type DeleteUserRequest struct {
		Username string `json:"username"`
	}
	
	var req DeleteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的请求"))
		return
	}
	
	// 不允许删除最后一个超管
	superAdminCount := 0
	deleteIsSuperAdmin := false
	for _, user := range consts.OSSetting.Users {
		if user.Role == "super_admin" {
			superAdminCount++
		}
		if user.Username == req.Username && user.Role == "super_admin" {
			deleteIsSuperAdmin = true
		}
	}
	
	if deleteIsSuperAdmin && superAdminCount <= 1 {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("不能删除最后一个超管账户"))
		return
	}
	
	// 删除用户
	setting := consts.GetOSSetting()
	newUsers := []model.User{}
	for _, user := range setting.Users {
		if user.Username != req.Username {
			newUsers = append(newUsers, user)
		}
	}
	setting.Users = newUsers
	consts.SetOSSetting(setting)
	
	// 持久化到文件
	service.FlushDictionary(setting.SelfPath)
	
	c.JSON(http.StatusOK, utils.NewSuccess())
}

// ChangePassword 修改密码
func ChangePassword(c *gin.Context) {
	type ChangePasswordRequest struct {
		Username    string `json:"username"`
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的请求"))
		return
	}
	
	// 验证旧密码
	validUser := false
	userIndex := -1
	for i, user := range consts.OSSetting.Users {
		if user.Username == req.Username && user.Password == req.OldPassword {
			validUser = true
			userIndex = i
			break
		}
	}
	
	if !validUser {
		c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("用户名或旧密码错误"))
		return
	}
	
	// 修改密码
	setting := consts.GetOSSetting()
	setting.Users[userIndex].Password = req.NewPassword
	consts.SetOSSetting(setting)
	
	// 持久化到文件
	service.FlushDictionary(setting.SelfPath)
	
	c.JSON(http.StatusOK, utils.NewSuccess())
}

// GetUsers 获取用户列表（仅返回用户名、角色和有效期，不返回密码）
func GetUsers(c *gin.Context) {
	type UserInfo struct {
		Username   string `json:"username"`
		Role       string `json:"role"`
		ExpireDate string `json:"expireDate"`
	}
	
	users := []UserInfo{}
	for _, user := range consts.OSSetting.Users {
		users = append(users, UserInfo{
			Username:   user.Username,
			Role:       user.Role,
			ExpireDate: user.ExpireDate,
		})
	}
	
	res := utils.NewSuccess()
	res.Data = users
	c.JSON(http.StatusOK, res)
}
