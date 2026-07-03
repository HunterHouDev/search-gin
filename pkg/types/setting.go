package types

// Role 自定义角色定义
type Role struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Permissions []string `json:"permissions"`
}

type User struct {
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	Role        string   `json:"role"` // 角色名（引用 roles 中的 name，或 "user"/"super_admin"）
	ExpireDate  string   `json:"expireDate"`
	Permissions []string `json:"permissions"` // 用户级额外权限（叠加在角色之上）
}

type Setting struct {
	EnableTimeScan bool
	CutThenDelete  bool
	BaseUrl        string
	ImageUrl       string
	Remark         string

	SystemPlayerVolumn string
	SystemPlayerWidth  string
	SystemPlayer       string

	HardwareAcceleration bool
	HardwareAccelMode    string

	Tags           []string
	TagsLib        []string
	Dirs           []string
	DirsLib        []string
	ImageTypes     []string
	DocsTypes      []string
	VideoTypes     []string
	Types          []string
	MovieTypes     []string
	Pages          []string
	ControllerHost string
	FileHost       string

	// AdminPassword 管理员密码，从 setting.json 读取；未设置时登录将提示配置
	AdminPassword string `json:"adminPassword"`

	NodeName           string   `json:"nodeName"`
	EnableLanDiscovery *bool    `json:"enableLanDiscovery"`
	DiscoveryPeers     []string `json:"discoveryPeers"`

	Roles              []Role   `json:"roles"`
	Users              []User   `json:"users"`
	DefaultPermissions []string `json:"defaultPermissions"`

	// TaskMaxConcurrent 任务调度最大并发数（转码+剪辑+合并），默认 4，≤0 时不限制
	TaskMaxConcurrent int `json:"taskMaxConcurrent"`

	// StreamSecret AES-256-GCM 密钥（hex，64字符），用于 streamToken 加解密
	// 同一集群的所有节点应使用相同的密钥以实现跨节点流媒体互通
	// 为空时由系统自动生成并持久化
	StreamSecret string `json:"streamSecret"`
}
