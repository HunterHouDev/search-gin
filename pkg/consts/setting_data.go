package consts

import (
	"search-gin/pkg/types"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type TokenInfo struct {
	ExpireTime time.Time
	Username   string
	Role       string
}

var (
	AdminUsername = "admin"
	AdminPassword = "qwer"
	AdminRole     = "super_admin"

	adminPasswordHash string

	OSSetting    = types.Setting{}
	settingMutex sync.RWMutex

	TokenStore = make(map[string]TokenInfo)
	tokenMutex sync.RWMutex
)

func init() {
	hash, err := bcrypt.GenerateFromPassword([]byte(AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		panic("生成管理员密码哈希失败: " + err.Error())
	}
	adminPasswordHash = string(hash)
}

// AdminPasswordHash 返回超管密码的 bcrypt 哈希
func AdminPasswordHash() string {
	return adminPasswordHash
}

// HashPassword 使用 bcrypt 对密码进行哈希（自带盐值）
func HashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hash)
}

// VerifyPassword 验证明文密码是否匹配 bcrypt 哈希值
func VerifyPassword(plainPassword, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

// SetToken 设置token
func SetToken(token string, expireTime time.Time, username string, role string) {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	TokenStore[token] = TokenInfo{
		ExpireTime: expireTime,
		Username:   username,
		Role:       role,
	}
}

// ValidateToken 验证token是否有效，同时检查用户是否过期
func ValidateToken(token string) bool {
	_, valid := ValidateTokenWithInfo(token)
	return valid
}

// ValidateTokenWithInfo 验证token并返回TokenInfo
func ValidateTokenWithInfo(token string) (TokenInfo, bool) {
	tokenMutex.RLock()
	tokenInfo, exists := TokenStore[token]
	tokenMutex.RUnlock()

	if !exists {
		return TokenInfo{}, false
	}

	// 检查token是否过期
	if time.Now().After(tokenInfo.ExpireTime) {
		tokenMutex.Lock()
		delete(TokenStore, token)
		tokenMutex.Unlock()
		return TokenInfo{}, false
	}

	// 普通用户检查有效期（超管不在此列）
	if tokenInfo.Username != AdminUsername && tokenInfo.Username != "" {
		for _, user := range GetOSSettingUsers() {
			if user.Username == tokenInfo.Username {
				if user.ExpireDate != "" {
					expireTime, err := time.Parse("2006-01-02", user.ExpireDate)
					if err == nil && time.Now().After(expireTime) {
						tokenMutex.Lock()
						delete(TokenStore, token)
						tokenMutex.Unlock()
						return TokenInfo{}, false
					}
				}
				break
			}
		}
	}

	return tokenInfo, true
}

// CleanExpiredTokens 清理过期的token
func CleanExpiredTokens() {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	now := time.Now()
	for token, tokenInfo := range TokenStore {
		if now.After(tokenInfo.ExpireTime) {
			delete(TokenStore, token)
		}
	}
}

func init() {
	// 定期清理过期 token
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			CleanExpiredTokens()
		}
	}()

	OSSetting = types.Setting{
		IsDb:                 true,
		IsJavBus:             false,
		EnableTimeScan:       true,
		SystemPlayerVolumn:   "30",
		SystemPlayerWidth:    "1280",
		HardwareAcceleration: false,
		HardwareAccelMode:    "",
		SelfPath:             "setting.json",
		ControllerHost:       ":10081",
		FileHost:             ":10082",
		BaseUrl:              "https://www.busjav.blog/",
		ImageUrl:             "",
		Remark:               "",
		Dirs: []string{
			"e://emby",
			"e://code",
		},
		Tags: []string{
			"東京",
		},
		ImageTypes: []string{GIF, PNG, JPG},
		DocsTypes:  []string{TXT, XLSX},
		VideoTypes: []string{AVI, MKV, WMV, MP4},
		Types:      []string{AVI, MKV, WMV, MP4, GIF, PNG, JPG, TXT, XLSX},
		Buttons:    []string{"刮图", "删除", "移动", "扫码"},
		MovieTypes: []string{"骑兵", "步兵", "国产", "漫动"},
		Pages:      []string{"10", "12", "15", "27", "50", "100"},
	}
}

// GetOSSetting 获取系统配置（线程安全）
func GetOSSetting() types.Setting {
	settingMutex.RLock()
	defer settingMutex.RUnlock()
	return OSSetting
}

// SetOSSetting 设置系统配置（线程安全）
func SetOSSetting(setting types.Setting) {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	OSSetting = setting
}

// UpdateOSSetting 原子地读取-修改-写入系统配置
func UpdateOSSetting(fn func(s types.Setting) types.Setting) {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	OSSetting = fn(OSSetting)
}

// GetOSSettingUsers 获取用户列表（线程安全），避免拷贝整个 Setting 结构体
func GetOSSettingUsers() []types.User {
	settingMutex.RLock()
	defer settingMutex.RUnlock()
	users := make([]types.User, len(OSSetting.Users))
	copy(users, OSSetting.Users)
	return users
}
