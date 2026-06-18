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
	Buttons        []string
	MovieTypes     []string
	Pages          []string
	ControllerHost string
	FileHost       string

	NodeName           string   `json:"nodeName"`
	EnableLanDiscovery *bool    `json:"enableLanDiscovery"`
	DiscoveryPeers     []string `json:"discoveryPeers"`

	Users []User `json:"users"`
}
