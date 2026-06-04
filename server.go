package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"search-gin/internal/service"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
)

// resolvePort 从 ControllerHost 中提取端口号
// 支持 ":10081" 和 "127.0.0.1:10081" 两种格式
func resolvePort(controllerHost string) string {
	if controllerHost == "" {
		return consts.PortNo
	}
	idx := strings.LastIndex(controllerHost, ":")
	if idx < 0 {
		return consts.PortNo
	}
	return controllerHost[idx:]
}

// createServer 创建具有标准超时配置的 HTTP 服务器
func createServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

// gracefulShutdown 监听信号，收到后优雅关闭所有 HTTP 服务
func gracefulShutdown(sigChan <-chan os.Signal, servers []*http.Server) {
	go func() {
		defer utils.RecoverPanic()

		sig := <-sigChan
		utils.InfoFormat("收到信号 %v，正在优雅关闭所有服务...", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		for _, srv := range servers {
			srv.Shutdown(ctx)
			log.Printf("端口 %s 已关闭", srv.Addr)
		}
		service.TaskCancel()
		<-ctx.Done()
		log.Println("服务已全部关闭")
	}()
}
