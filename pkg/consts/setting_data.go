package consts

import (
	"search-gin/internal/model"
	"sync"
	"time"
)

type TokenInfo struct {
	ExpireTime time.Time
	Username   string
	Role       string
}

var (
	OSSetting    = model.Setting{}
	settingMutex sync.RWMutex
	
	// Token存储（简单内存实现）
	TokenStore    = make(map[string]TokenInfo)
	tokenMutex   sync.RWMutex
)

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

	// 检查用户是否过期
	if tokenInfo.Username != "" {
		for _, user := range OSSetting.Users {
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
	OSSetting = model.Setting{
		IsDb:               true,
		IsJavBus:           false,
		EnableTimeScan:     true,
		SystemPlayerVolumn:    "30",
		SystemPlayerWidth:     "1280",
		HardwareAcceleration:  false,
		HardwareAccelMode:     "",
		SelfPath:              "setting.json",
		ControllerHost:     "127.0.0.1:17001",
		ImageHost:          "127.0.0.1:17002",
		StreamHost:         "127.0.0.1:17003",
		BaseUrl:            "https://www.busjav.blog/",
		ImageUrl:           "",
		OMUrl:              "https://www.busjav.blog/",
		Remark:             "",
		SystemHtml:         "",
		Dirs: []string{
			"e://emby",
			"e://code",
		},
		Tags: []string{
			"東京熱",
		},
		ImageTypes: []string{GIF, PNG, JPG},
		DocsTypes:  []string{TXT, XLSX},
		VideoTypes: []string{AVI, MKV, WMV, MP4},
		Types:      []string{AVI, MKV, WMV, MP4, GIF, PNG, JPG, TXT, XLSX},
		Buttons:    []string{"刮图", "删除", "移动"},
		MovieTypes: []string{"骑兵", "步兵", "国产", "漫动"},
		Pages:      []string{"10", "12", "15", "27", "50", "100"},
		Users: []model.User{
			{
				Username: "admin",
				Password: "qwer",
				Role:     "super_admin",
			},
		},
	}
}

// GetOSSetting 获取系统配置（线程安全）
func GetOSSetting() model.Setting {
	settingMutex.RLock()
	defer settingMutex.RUnlock()
	return OSSetting
}

// SetOSSetting 设置系统配置（线程安全）
func SetOSSetting(setting model.Setting) {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	OSSetting = setting
}
