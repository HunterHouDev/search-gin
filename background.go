package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"search-gin/internal/env"
	"search-gin/internal/service"
	"search-gin/pkg/utils"
)

// startPprof 开发环境下启动 pprof 调试接口
func startPprof() {
	if env.IsProd {
		log.Println("生产环境已禁用 pprof 调试接口")
		return
	}
	go func() {
		defer utils.RecoverPanic()
		log.Println("pprof 调试接口启动在 localhost:6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
}

// startBackgroundTasks 启动心跳扫描和转换任务执行
func startBackgroundTasks() {
	go func() {
		defer utils.RecoverPanic()
		service.FileApp.HeartBeat()
	}()
	go func() {
		defer utils.RecoverPanic()
		service.FileApp.TaskExecuting()
	}()
}

// startTorrentCleanup 启动 Torrent 清理协程，返回关闭函数
func startTorrentCleanup(tempDir string) func() {
	torrentDir := filepath.Join(tempDir, "torrent_data")
	os.MkdirAll(torrentDir, 0755)

	if err := service.NewTorrentService(torrentDir); err != nil {
		utils.InfoFormat("Torrent 服务启动失败: %v", err)
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	go service.TorrentApp.StartCleanup(ctx)

	return func() {
		cancel()
		service.TorrentApp.Close()
	}
}
