package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"search-gin/internal/service"
	"search-gin/pkg/utils"
)

// ResolvePort 从 ControllerHost 中提取端口号
// 支持 ":10081" 和 "127.0.0.1:10081" 两种格式
func ResolvePort(portNo, controllerHost string) string {
	if controllerHost == "" {
		return portNo
	}
	idx := strings.LastIndex(controllerHost, ":")
	if idx < 0 {
		return portNo
	}
	return controllerHost[idx:]
}

// CreateServer 创建 HTTP 服务器
// 注意: 不设置 ReadTimeout 和 WriteTimeout，因为 WebSocket hijack 后超时仍会残留导致连接断开
//
//	使用 ReadHeaderTimeout 防范慢连接攻击即可
func CreateServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

// GracefulShutdown 监听信号，收到后优雅关闭所有 HTTP 服务
func GracefulShutdown(sigChan <-chan os.Signal, servers []*http.Server) {
	go func() {
		defer utils.RecoverPanic()

		sig := <-sigChan
		utils.InfoFormat("收到信号 %v，正在优雅关闭所有服务...", sig)

		shutdownTimeout := 5 * time.Second
		for _, srv := range servers {
			ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			if err := srv.Shutdown(ctx); err != nil {
				log.Printf("端口 %s 关闭失败: %v", srv.Addr, err)
			} else {
				log.Printf("端口 %s 已关闭", srv.Addr)
			}
			cancel()
		}
		service.TaskCancel()
		service.HeartBeatCancel()
		log.Println("服务已全部关闭")
	}()
}
