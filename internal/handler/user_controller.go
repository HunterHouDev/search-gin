package handler

import (
	"net/http"
	"time"

	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

func GetUsers(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	type UserInfo struct {
		Username    string   `json:"username"`
		Role        string   `json:"role"`
		ExpireDate  string   `json:"expireDate"`
		Permissions []string `json:"permissions"`
	}

	var users []UserInfo
	for _, u := range service.GetOSSettingUsers() {
		users = append(users, UserInfo{
			Username:    u.Username,
			Role:        u.Role,
			ExpireDate:  u.ExpireDate,
			Permissions: u.Permissions,
		})
	}
	res := utils.NewSuccess()
	res.Data = users
	c.JSON(http.StatusOK, res)
}

func AddUser(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	type AddUserRequest struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		Role       string `json:"role"`
		ExpireDate string `json:"expireDate"`
	}

	var req AddUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的请求"))
		return
	}

	if req.Username == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("用户名不能为空"))
		return
	}
	if req.Password == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("密码不能为空"))
		return
	}

	if req.Username == service.AdminUsername {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("用户名已存在"))
		return
	}

	if req.ExpireDate != "" {
		if _, err := time.Parse("2006-01-02", req.ExpireDate); err != nil {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("有效期格式错误，应为YYYY-MM-DD"))
			return
		}
	}

	// 指定的角色必须存在
	if req.Role != "" && req.Role != "user" {
		if _, ok := service.GetRole(req.Role); !ok {
			c.JSON(http.StatusBadRequest, utils.NewFailByMsg("角色不存在: "+req.Role))
			return
		}
	}
	roleName := req.Role
	if roleName == "" {
		roleName = "user"
	}

	newUser := model.User{
		Username:   req.Username,
		Password:   service.HashPassword(req.Password),
		Role:       roleName,
		ExpireDate: req.ExpireDate,
	}
	added := false
	service.UpdateOSSetting(func(s model.Setting) model.Setting {
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
	UseApp().config.Flush(service.SettingFileName)
	 c.JSON(http.StatusOK, utils.NewSuccess())
	}

	func DeleteUser(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	type DeleteUserRequest struct {
		Username string `json:"username"`
	}

	var req DeleteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的请求"))
		return
	}

	deleted := false
	service.UpdateOSSetting(func(s model.Setting) model.Setting {
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
	UseApp().config.Flush(service.SettingFileName)
	 c.JSON(http.StatusOK, utils.NewSuccess())
	}
