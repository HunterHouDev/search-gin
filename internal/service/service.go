package service

import "search-gin/pkg/utils"

type searchService struct {
	engine *searchEngineCore
	dl     *downloader
	ve     *videoEncoder
}

var WorkDir string
var SearchEngine = searchEngineCore{
	KeywordHistoryCache: utils.NewLRUCache(500),
}
var SearchApp = &searchService{
	engine: &SearchEngine,
	dl:     new(downloader),
	ve:     new(videoEncoder),
}
var Downloader = SearchApp.dl
var VideoEncoder = SearchApp.ve
