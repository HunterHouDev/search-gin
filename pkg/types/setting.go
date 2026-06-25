package types

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

	DeepSeekApiKey string `json:"deepSeekApiKey"`

	NodeName           string   `json:"nodeName"`
	EnableLanDiscovery *bool    `json:"enableLanDiscovery"`
	DiscoveryPeers     []string `json:"discoveryPeers"`

	Users []User `json:"users"`

	// TaskMaxConcurrent 任务调度最大并发数（转码+剪辑+合并），默认 4，≤0 时不限制
	TaskMaxConcurrent int `json:"taskMaxConcurrent"`
}
