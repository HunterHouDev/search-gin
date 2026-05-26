package model

// User 用户结构体
type User struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Role       string `json:"role"`      // super_admin 或 user
	ExpireDate string `json:"expireDate"` // 有效期，格式：2006-01-02，空字符串表示永不过期
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
	ImageHost      string
	StreamHost     string
	
	// 用户列表（替代原来的LoginPassword）
	Users          []User `json:"users"`
}
