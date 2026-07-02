package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ─── 管理员账号 ──────────────────────────────────────────────────

const (
	AdminUsername = "admin"
	AdminRole     = "super_admin"
)

// adminPasswordHash 缓存 setting.json 中 AdminPassword 的 bcrypt 哈希，
// 在 InitSetting 中预计算，避免每次登录都重复 bcrypt (~100ms)
var adminPasswordHash string

// CacheAdminPasswordHash 预计算 setting.json 中 AdminPassword 的 bcrypt 哈希并缓存，
// 由 InitSetting 在启动时调用
func CacheAdminPasswordHash() {
	pwd := GetOSSetting().AdminPassword
	if pwd == "" {
		utils.InfoFormat("未配置管理员密码，请在 setting.json 中设置 adminPassword")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		utils.ErrorFormat("缓存管理员密码哈希失败: %v", err)
		return
	}
	adminPasswordHash = string(hash)
	utils.InfoFormat("管理员密码哈希已缓存")
}

// HashPassword 使用 bcrypt 对密码进行哈希
func HashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hash)
}

// VerifyPassword 验证明文密码是否匹配 bcrypt 哈希
func VerifyPassword(plainPassword, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

// ─── Token 存储 ─────────────────────────────────────────────────

type TokenInfo struct {
	ExpireTime  time.Time
	Username    string
	Role        string
	Permissions []string
}

var (
	tokenStore = make(map[string]TokenInfo)
	tokenMu    sync.RWMutex
)

// SetToken 设置 token，到期后自动清理
func SetToken(token string, expireTime time.Time, username string, role string, permissions []string) {
	tokenMu.Lock()
	tokenStore[token] = TokenInfo{
		ExpireTime:  expireTime,
		Username:    username,
		Role:        role,
		Permissions: permissions,
	}
	tokenMu.Unlock()

	// 到期后自动删除，无需定时轮询
	delay := time.Until(expireTime)
	if delay > 0 {
		time.AfterFunc(delay, func() {
			tokenMu.Lock()
			delete(tokenStore, token)
			tokenMu.Unlock()
		})
	}
}

// ValidateTokenWithInfo 验证 token 并返回 TokenInfo
//
// ⚠️ 必须使用 Lock（而非 RLock）：函数内部包含 tokenStore 的写操作
// （过期删除、旧 token 兼容补全、权限同步）。用 RLock 会导致并发 map 写 panic。
// 不要因为"热路径性能"而改回 RLock——token map 规模小（<1000），Lock 争抢开销可忽略。
func ValidateTokenWithInfo(token string) (TokenInfo, bool) {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	tokenInfo, exists := tokenStore[token]
	if !exists {
		return TokenInfo{}, false
	}

	if time.Now().After(tokenInfo.ExpireTime) {
		delete(tokenStore, token)
		return TokenInfo{}, false
	}

	// 兼容旧 token：admin 用户 role 为空时自动补全并持久化
	if tokenInfo.Role == "" && tokenInfo.Username == AdminUsername {
		tokenInfo.Role = AdminRole
		info := tokenStore[token]
		info.Role = AdminRole
		tokenStore[token] = info
	}

	// 兼容旧 token：Permissions 为空时自动填充
	if len(tokenInfo.Permissions) == 0 {
		perms := GetUserPermissions(tokenInfo.Username, tokenInfo.Role)
		tokenInfo.Permissions = perms
		info := tokenStore[token]
		info.Permissions = perms
		tokenStore[token] = info
	}

	// 普通用户检查有效期 + 权限同步
	if tokenInfo.Username != AdminUsername && tokenInfo.Username != "" {
		for _, user := range GetOSSettingUsers() {
			if user.Username == tokenInfo.Username {
				if user.ExpireDate != "" {
					expireTime, err := time.Parse("2006-01-02", user.ExpireDate)
					if err == nil && time.Now().After(expireTime) {
						delete(tokenStore, token)
						return TokenInfo{}, false
					}
				}
				// 权限变更同步：每次验证时从 setting 同步最新权限
				if !stringSliceEqual(tokenInfo.Permissions, user.Permissions) && user.Permissions != nil {
					perms := GetUserPermissions(tokenInfo.Username, tokenInfo.Role)
					tokenInfo.Permissions = perms
					info := tokenStore[token]
					info.Permissions = perms
					tokenStore[token] = info
				}
				break
			}
		}
		return tokenInfo, true
	}

	return tokenInfo, true
}

// ─── 登录服务 ───────────────────────────────────────────────────

// LoginResult 登录结果
type LoginResult struct {
	Success     bool
	Message     string
	Token       string
	ExpireIn    int
	Role        string
	Username    string
	Permissions []string
}

// LoginUser 验证用户名密码并签发 token
// 密码来源：仅 setting.json 的 adminPassword 字段，无编译回退
// 当 username 为空时，仅凭密码匹配 admin 登录
func LoginUser(username, password string) LoginResult {
	// 无用户名或用户名为 admin → 按 admin 密码匹配
	if username == "" || username == AdminUsername {
		if GetOSSetting().AdminPassword == "" {
			return LoginResult{Success: false, Message: "未配置管理员密码，请在 setting.json 中设置 adminPassword"}
		}
		if VerifyPassword(password, adminPasswordHash) {
			return issueToken(AdminUsername, AdminRole)
		}
	}

	// 有用户名时检查普通用户
	for _, user := range GetOSSettingUsers() {
		if user.Username == username && VerifyPassword(password, user.Password) {
			if user.ExpireDate != "" {
				expireTime, err := time.Parse("2006-01-02", user.ExpireDate)
				if err == nil && time.Now().After(expireTime) {
					return LoginResult{Success: false, Message: "用户已过期，请联系管理员"}
				}
			}
			return issueToken(user.Username, user.Role)
		}
	}
	return LoginResult{Success: false, Message: "用户名或密码错误"}
}

// issueToken 生成 token 并存储
func issueToken(username, role string) LoginResult {
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return LoginResult{Success: false, Message: "生成token失败，系统错误"}
	}
	token := hex.EncodeToString(tokenBytes)
	expireIn := 4 * 3600 // 4小时，与服务端 SetToken 的有效期保持一致
	permissions := GetUserPermissions(username, role)
	SetToken(token, time.Now().Add(time.Duration(expireIn)*time.Second), username, role, permissions)
	return LoginResult{
		Success:     true,
		Token:       token,
		ExpireIn:    expireIn,
		Role:        role,
		Username:    username,
		Permissions: permissions,
	}
}

// RequireAdminWithName 检查角色或用户名是否为管理员（兼容旧 token）
func RequireAdminWithName(role, username string) bool {
	return role == AdminRole || username == AdminUsername
}

// ─── OSSetting 管理 ─────────────────────────────────────────────

var (
	OSSetting    = model.Setting{}
	settingMutex sync.RWMutex
)

// defaultSetting 返回默认系统设置
func defaultSetting() model.Setting {
	return model.Setting{
		SelfPath:             "setting.json",
		ControllerHost:       ":10081",
		FileHost:             ":10082",
		BaseUrl:              "https://www.busjav.blog/",
		EnableTimeScan:       true,
		SystemPlayerVolumn:   "30",
		SystemPlayerWidth:    "1280",
		HardwareAcceleration: false,
		Dirs: []string{
			"e://emby",
			"e://code",
		},
		Tags:              []string{"東京"},
		ImageTypes:        []string{"gif", "png", "jpg"},
		DocsTypes:         []string{"txt", "xlsx"},
		VideoTypes:        []string{"avi", "mkv", "wmv", "mp4"},
		Types:             []string{"avi", "mkv", "wmv", "mp4", "gif", "png", "jpg", "txt", "xlsx"},
		MovieTypes:        []string{"骑兵", "步兵", "国产", "漫动"},
		Pages:             []string{"10", "12", "15", "27", "50", "100"},
		TaskMaxConcurrent: 4,
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

// UpdateOSSetting 原子地读取-修改-写入系统配置
func UpdateOSSetting(fn func(s model.Setting) model.Setting) {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	OSSetting = fn(OSSetting)
}

// GetOSSettingUsers 获取用户列表（线程安全）
func GetOSSettingUsers() []model.User {
	settingMutex.RLock()
	defer settingMutex.RUnlock()
	users := make([]model.User, len(OSSetting.Users))
	copy(users, OSSetting.Users)
	return users
}

// FlushDictionary 将当前设置持久化到配置文件
func FlushDictionary(path string) error {
	return WriteDictionaryToJson(path, GetOSSetting())
}

func ReadDictionaryFromJson(path string) model.Setting {
	reader, err := os.ReadFile(path)
	if err != nil {
		utils.InfoFormat("读取配置文件失败: %v", err)
		return model.Setting{}
	}
	dict := model.Setting{}
	err = json.Unmarshal(reader, &dict)
	if err != nil {
		utils.InfoFormat("解析配置文件失败: %v", err)
		return model.Setting{}
	}
	return dict
}

func WriteDictionaryToJson(path string, dict model.Setting) error {
	data, err := json.Marshal(dict)
	if err != nil {
		return fmt.Errorf("序列化配置文件失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}

// stringSliceEqual 比较两个字符串切片是否相等（忽略顺序）
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]int, len(a))
	for _, v := range a {
		m[v]++
	}
	for _, v := range b {
		m[v]--
		if m[v] < 0 {
			return false
		}
	}
	return true
}

// newBool 返回指向给定 bool 值的指针
func newBool(v bool) *bool {
	return &v
}
