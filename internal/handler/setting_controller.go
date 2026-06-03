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

	// 生成本次 token 的通用函数
	issueToken := func(username, role string) {
		tokenBytes := make([]byte, 16)
		if _, err := rand.Read(tokenBytes); err != nil {
			c.JSON(http.StatusInternalServerError, utils.NewFailByMsg("生成token失败，系统错误"))
			return
		}
		token := hex.EncodeToString(tokenBytes)
		consts.SetToken(token, time.Now().Add(2*time.Hour), username, role)
		res := utils.NewSuccess()
		res.Data = gin.H{
			"token":    token,
			"expireIn": 2 * 3600,
			"role":     role,
			"username": username,
		}
		c.JSON(http.StatusOK, res)
	}

	// 1. 硬编码超管
	if req.Username == consts.AdminUsername && req.Password == consts.AdminPassword {
		issueToken(consts.AdminUsername, consts.AdminRole)
		return
	}

	// 2. 普通用户（从配置读取）
	for _, user := range consts.GetOSSetting().Users {
		if user.Username == req.Username && user.Password == req.Password {
			if user.ExpireDate != "" {
				expireTime, err := time.Parse("2006-01-02", user.ExpireDate)
				if err == nil && time.Now().After(expireTime) {
					c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("用户已过期，请联系管理员"))
					return
				}
			}
			issueToken(user.Username, user.Role)
			return
		}
	}

	c.JSON(http.StatusUnauthorized, utils.NewFailByMsg("用户名或密码错误"))
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

// AddUser 添加普通用户
func AddUser(c *gin.Context) {
	type AddUserRequest struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		ExpireDate string `json:"expireDate"`
	}

	var req AddUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的请求"))
		return
	}

	// 禁止创建超管用户名
	if req.Username == consts.AdminUsername {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("用户名已存在"))
		return
	}

	if req.ExpireDate != "" {
		if _, err := time.Parse("2006-01-02", req.ExpireDate); err != nil {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("有效期格式错误，应为YYYY-MM-DD"))
			return
		}
	}

	newUser := model.User{
		Username:   req.Username,
		Password:   req.Password,
		Role:       "user",
		ExpireDate: req.ExpireDate,
	}
	added := false
	consts.UpdateOSSetting(func(s model.Setting) model.Setting {
		for _, u := range s.Users {
			if u.Username == req.Username {
				return s
			}
		}
		s.Users = append(s.Users, newUser)
		added = true
		return s
	})
	if !added {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("用户名已存在"))
		return
	}
	service.FlushDictionary(consts.GetOSSetting().SelfPath)
	c.JSON(http.StatusOK, utils.NewSuccess())
}

// DeleteUser 删除普通用户
func DeleteUser(c *gin.Context) {
	type DeleteUserRequest struct {
		Username string `json:"username"`
	}

	var req DeleteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的请求"))
		return
	}

	deleted := false
	consts.UpdateOSSetting(func(s model.Setting) model.Setting {
		idx := -1
		for i, u := range s.Users {
			if u.Username == req.Username {
				idx = i
				break
			}
		}
		if idx >= 0 {
			s.Users = append(s.Users[:idx], s.Users[idx+1:]...)
			deleted = true
		}
		return s
	})
	if !deleted {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("用户不存在"))
		return
	}
	service.FlushDictionary(consts.GetOSSetting().SelfPath)
	c.JSON(http.StatusOK, utils.NewSuccess())
}

// GetUsers 获取普通用户列表（不返回密码）
func GetUsers(c *gin.Context) {
	type UserInfo struct {
		Username   string `json:"username"`
		Role       string `json:"role"`
		ExpireDate string `json:"expireDate"`
	}

	var users []UserInfo
	for _, u := range consts.GetOSSetting().Users {
		users = append(users, UserInfo{
			Username:   u.Username,
			Role:       u.Role,
			ExpireDate: u.ExpireDate,
		})
	}
	res := utils.NewSuccess()
	res.Data = users
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


