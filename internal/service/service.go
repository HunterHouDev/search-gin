package service

// WorkDir 工作空间路径
var WorkDir string

var SearchApp = new(searchService)

// SearchEngine 搜索引擎
var SearchEngine = searchEngineCore{}

// searchService 统一服务，嵌入各功能模块
type searchService struct{}

var Downloader = new(downloader)
var VideoEncoder = new(videoEncoder)
