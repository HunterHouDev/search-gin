package model

// User 普通用户
type User struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Role       string `json:"role"`
	ExpireDate string `json:"expireDate"`
}

type Setting struct {
	IsDb           bool
	IsJavBus       bool
	EnableTimeScan bool
	CutThenDelete  bool
	SelfPath       string
	BaseUrl        string
	OMUrl          string
	ImageUrl       string
	Remark         string
	SystemHtml     string

	SystemPlayerVolumn string
	SystemPlayerWidth  string
	SystemPlayer       string

	HardwareAcceleration bool   // 是否启用硬件加速
	HardwareAccelMode    string // 硬件加速模式名称（如 NVIDIA NVENC）

	Tags           []string
	TagsLib        []string
	Dirs           []string
	DirsLib        []string
	ImageTypes     []string
	DocsTypes      []string
	VideoTypes     []string
	Types          []string
	Buttons        []string
	MovieTypes     []string
	Pages          []string
	ControllerHost string
	FileHost       string

	// 多节点配置
	NodeName             string   `json:"nodeName"`             // 本机节点别名，用于多节点下标识机器
	EnableLanDiscovery   *bool    `json:"enableLanDiscovery"`   // 是否启用 UDP 组播发现，nil 表示未配置（默认启用）
	LanDiscoveryInterval int      `json:"lanDiscoveryInterval"` // 心跳发送间隔（秒），默认 30
	LanDiscoveryTimeout  int      `json:"lanDiscoveryTimeout"`  // 心跳超时时间（秒），默认 90
	DiscoveryPeers       []string `json:"discoveryPeers"`       // 手动指定节点列表 "ip:port"

	// 普通用户列表（超管已代码写死，不在配置中）
	Users          []User `json:"users"`
}
