package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"search-gin/internal/router"
	"search-gin/internal/service"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

//go:generate go env -w GO111MODULE=on
//go:generate go env -w GOPROXY=https://goproxy.cn,direct
//go:generate go mod tidy
//go:generate go mod download

// 打包命令参考:
//   go build                                            # 有窗口 CLI
//   go build -ldflags "-H=windowsgui -s -w" -tags=prod  # 无窗口 GUI (Windows)
//   go build linux                                       # Linux 交叉编译

func main() {
	defer utils.RecoverPanic()

	// ── 1. 初始化工作目录 ──
	tempDir, err := os.Getwd()
	if err != nil {
		utils.InfoFormat("获取当前工作目录失败: %v，使用默认路径", err)
		tempDir = "."
	}
	service.TempDir = tempDir

	// ── 2. 解压嵌入式资源 ──
	assetsExtracted := extractAssets(tempDir)

	// ── 3. 启动 pprof（仅开发环境） ──
	service.StartPprof()

	// ── 4. 信号通道 ──
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// ── 5. 初始化配置和扫描队列 ──
	service.InitSetting()
	service.StartScanQueue()

	// ── 6. 启动 Torrent 清理 ──
	closeTorrent := service.StartTorrentCleanup(tempDir)
	defer closeTorrent()

	// ── 7. 构建路由 ──
	app := router.BuildRouter(tempDir)

	// ── 8. 加载前端静态文件（等待解压完成） ──
	go loadStaticFiles(app, tempDir, assetsExtracted)

	// ── 9. 启动后台任务 ──
	service.StartBackgroundTasks()

	// ── 10. 启动多端口 HTTP 服务 ──
	var g errgroup.Group
	portNos := []string{consts.PortNo, consts.PortNo2, consts.PortNo3}
	servers := make([]*http.Server, len(portNos))

	for i, port := range portNos {
		srv := createServer(port, app)
		servers[i] = srv
		g.Go(func() error {
			defer utils.RecoverPanic()
			utils.InfoFormat("启动端口 %s", srv.Addr)
			return srv.ListenAndServe()
		})
	}

	// ── 11. 注册 /api/close 关闭接口 ──
	app.GET("api/close", func(c *gin.Context) {
		c.String(200, "即将关闭所有服务器")
		sigChan <- syscall.SIGTERM
	})

	// ── 12. 优雅关闭监听 ──
	gracefulShutdown(sigChan, servers)

	// ── 13. 等待所有 HTTP 服务退出 ──
	if err := g.Wait(); err != nil && err != http.ErrServerClosed {
		utils.InfoFormat("服务异常: %v", err)
	} else {
		utils.InfoFormat("服务已停止")
	}
}
