package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ─── 管理员账号（纯常量） ──────────────────────────────────────────

const (
	AdminUsername = "admin"
	AdminPassword = "qwer"
	AdminRole     = "super_admin"
)

var adminPasswordHash string

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
	ExpireTime time.Time
	Username   string
	Role       string
}

var (
	tokenStore = make(map[string]TokenInfo)
	tokenMu    sync.RWMutex
)

// SetToken 设置 token
func SetToken(token string, expireTime time.Time, username string, role string) {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	tokenStore[token] = TokenInfo{
		ExpireTime: expireTime,
		Username:   username,
		Role:       role,
	}
}

// ValidateTokenWithInfo 验证 token 并返回 TokenInfo
func ValidateTokenWithInfo(token string) (TokenInfo, bool) {
	tokenMu.RLock()
	tokenInfo, exists := tokenStore[token]
	tokenMu.RUnlock()
	if !exists {
		return TokenInfo{}, false
	}

	if time.Now().After(tokenInfo.ExpireTime) {
		tokenMu.Lock()
		delete(tokenStore, token)
		tokenMu.Unlock()
		return TokenInfo{}, false
	}

	// 兼容旧 token：admin 用户 role 为空时自动补全并持久化
	if tokenInfo.Role == "" && tokenInfo.Username == AdminUsername {
		tokenMu.Lock()
		info := tokenStore[token]
		info.Role = AdminRole
		tokenStore[token] = info
		tokenMu.Unlock()
		tokenInfo.Role = AdminRole
	}

	// 普通用户检查有效期
	if tokenInfo.Username != AdminUsername && tokenInfo.Username != "" {
		for _, user := range GetOSSettingUsers() {
			if user.Username == tokenInfo.Username {
				if user.ExpireDate != "" {
					expireTime, err := time.Parse("2006-01-02", user.ExpireDate)
					if err == nil && time.Now().After(expireTime) {
						tokenMu.Lock()
						delete(tokenStore, token)
						tokenMu.Unlock()
						return TokenInfo{}, false
					}
				}
				break
			}
		}
	}

	return tokenInfo, true
}

// CleanExpiredTokens 清理过期 token
func CleanExpiredTokens() {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	now := time.Now()
	for token, tokenInfo := range tokenStore {
		if now.After(tokenInfo.ExpireTime) {
			delete(tokenStore, token)
		}
	}
}

// ─── 登录服务 ───────────────────────────────────────────────────

// LoginResult 登录结果
type LoginResult struct {
	Success  bool
	Message  string
	Token    string
	ExpireIn int
	Role     string
	Username string
}

// LoginUser 验证用户名密码并签发 token
func LoginUser(username, password string) LoginResult {
	if username == AdminUsername && VerifyPassword(password, AdminPasswordHash()) {
		return issueToken(username, AdminRole)
	}

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
	SetToken(token, time.Now().Add(time.Duration(expireIn)*time.Second), username, role)
	return LoginResult{
		Success:  true,
		Token:    token,
		ExpireIn: expireIn,
		Role:     role,
		Username: username,
	}
}

// RequireAdmin 检查角色是否为管理员
func RequireAdmin(role string) bool {
	return role == AdminRole
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
		Tags:       []string{"東京"},
		ImageTypes: []string{"gif", "png", "jpg"},
		DocsTypes:  []string{"txt", "xlsx"},
		VideoTypes: []string{"avi", "mkv", "wmv", "mp4"},
		Types:      []string{"avi", "mkv", "wmv", "mp4", "gif", "png", "jpg", "txt", "xlsx"},
		MovieTypes: []string{"骑兵", "步兵", "国产", "漫动"},
		Pages:      []string{"10", "12", "15", "27", "50", "100"},
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
func FlushDictionary(path string) {
	WriteDictionaryToJson(path, GetOSSetting())
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
	// 迁移明文密码到 bcrypt hash（启动时自动执行）
	dict = migratePlaintextPasswords(dict)
	return dict
}

// migratePlaintextPasswords 将用户列表中的明文密码迁移为 bcrypt hash。
// 检测方式：bcrypt hash 以 $2a$ 或 $2b$ 开头，非此前缀则视为明文。
func migratePlaintextPasswords(s model.Setting) model.Setting {
	migrated := false
	for i, u := range s.Users {
		if u.Password == "" {
			continue
		}
		// bcrypt hash 固定以 $2a$ 或 $2b$ 开头，否则是明文
		if len(u.Password) < 4 || (u.Password[:4] != "$2a$" && u.Password[:4] != "$2b$") {
			hash := HashPassword(u.Password)
			if hash != "" {
				s.Users[i].Password = hash
				migrated = true
				utils.InfoFormat("密码已迁移（bcrypt）: 用户 %s", u.Username)
			}
		}
	}
	if migrated {
		count := migratedCount(s.Users)
		utils.InfoFormat("有 %d 个用户密码已从明文迁移到 bcrypt", count)
		// 立即持久化，避免明文残留在磁盘
		if s.SelfPath != "" {
			WriteDictionaryToJson(s.SelfPath, s)
		}
	}
	return s
}

// migratedCount 统计迁移后的 bcrypt 用户数
func migratedCount(users []model.User) int {
	count := 0
	for _, u := range users {
		if len(u.Password) >= 4 && (u.Password[:4] == "$2a$" || u.Password[:4] == "$2b$") {
			count++
		}
	}
	return count
}

func WriteDictionaryToJson(path string, dict model.Setting) {
	data, err := json.Marshal(dict)
	if err != nil {
		utils.InfoFormat("序列化配置文件失败: %v", err)
		return
	}
	err = os.WriteFile(path, data, 0600)
	if err != nil {
		utils.InfoFormat("写入配置文件失败: %v", err)
		return
	}
}

// newBool 返回指向给定 bool 值的指针
func newBool(v bool) *bool {
	return &v
}
