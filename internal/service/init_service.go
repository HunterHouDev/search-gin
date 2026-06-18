package service

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"search-gin/internal/env"
	"search-gin/pkg/utils"
)

// StartPprof 开发环境下启动 pprof 调试接口
func StartPprof() {
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

// StartBackgroundTasks 启动心跳扫描和转换任务执行
func StartBackgroundTasks() {
	go func() {
		defer utils.RecoverPanic()
		SearchApp.HeartBeat()
	}()
	go func() {
		defer utils.RecoverPanic()
		SearchApp.TaskExecuting()
	}()
}

// StartTorrentCleanup 启动 Torrent 清理协程，返回关闭函数
func StartTorrentCleanup(workDir string) func() {
	torrentDir := filepath.Join(workDir, "torrent_data")
	if err := os.MkdirAll(torrentDir, 0755); err != nil {
		utils.ErrorFormat("创建 torrent 目录失败: %v", err)
		return func() {}
	}

	if err := NewTorrentService(torrentDir); err != nil {
		utils.InfoFormat("Torrent 服务启动失败: %v", err)
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	go TorrentApp.StartCleanup(ctx)

	return func() {
		cancel()
		TorrentApp.Close()
	}
}
