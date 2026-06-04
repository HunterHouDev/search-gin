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

	// 普通用户列表（超管已代码写死，不在配置中）
	Users          []User `json:"users"`
}
