package model

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
}
