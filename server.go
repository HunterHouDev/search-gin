package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"search-gin/internal/service"
	"search-gin/pkg/utils"
)

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
