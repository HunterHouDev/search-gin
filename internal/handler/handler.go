package handler

import (
	"search-gin/internal/service"
)

type AppHandle struct {
	search service.IndexEngine
	files  service.FileService
	config service.Settings
}

var appHandle *AppHandle

// InitApp 初始化全局 AppHandle（由 main.go 显式调用）
func InitApp(search service.IndexEngine, files service.FileService, config service.Settings) {
	appHandle = &AppHandle{
		search: search,
		files:  files,
		config: config,
	}
}

// UseApp 返回全局 AppHandle
func UseApp() *AppHandle {
	return appHandle
}
