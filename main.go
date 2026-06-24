package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"search-gin/internal/env"
	"search-gin/internal/handler"
	"search-gin/internal/router"
	"search-gin/internal/server"
	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"golang.org/x/sync/errgroup"
)

//go:generate go env -w GO111MODULE=on
//go:generate go env -w GOPROXY=https://goproxy.cn,direct
//go:generate go mod tidy
//go:generate go mod download

// 打包命令参考:
//
//	go build                                            # 有窗口 CLI
//	go build -ldflags "-H=windowsgui -s -w" -tags=prod  # 无窗口 GUI (Windows)
//	go build linux                                       # Linux 交叉编译

func main() {
	defer utils.RecoverPanic()

	// ── 1. 初始化工作目录 ──
	utils.SetLogLevel(env.IsProd)
	workDir, err := os.Getwd()
	if err != nil {
		utils.InfoFormat("获取当前工作目录失败: %v，使用默认路径", err)
		workDir = "."
	}
	service.SetWorkDir(workDir)

	// ── 2. 创建核心组件（显式依赖图） ──
	engine := service.NewSearchEngine()
	settings := service.DefaultSettings()
	events := service.DefaultEventBus()

	// ── 3. 创建扫描队列并关联 searchService ──
	scanQueue := service.NewScanQueue(engine, settings)
	search := service.NewSearchService(engine, settings, events, scanQueue)
	service.SetScanWalkInner(search.WalkDirWithCfg)

	// ── 4. 注册全局（内部函数仍需通过 getter 访问） ──
	service.InitService(engine, search)

	// ── 5. 加载上次扫描的索引缓存（填补启动空窗期） ──
	engine.LoadCachedIndex()

	// ── 6. 解压嵌入式资源 ──
	assetsExtracted := extractAssets(workDir)

	// ── 7. 启动 pprof（生产环境通过 env.IsProd 自动禁用） ──
	service.StartPprof()

	// ── 8. 信号通道 ──
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// ── 9. 初始化配置和扫描队列 ──
	service.InitSetting()
	service.InitSearchPool()
	service.StartScanQueue()

	// ── 10. 初始化节点管理器（手动添加 + 反向心跳自动发现） ──
	service.InitPeerManager()

	// ── 11. 创建 Handler（注入依赖） ──
	handler.InitApp(engine, search, settings)

	// ── 12. 启动 Torrent 清理 ──
	closeTorrent := service.StartTorrentCleanup(workDir)
	defer closeTorrent()

	// ── 13. 构建路由（API 路由 + 文件流路由） ──
	apiRouter := router.BuildAPIRouter(sigChan)
	fileRouter := router.BuildFileRouter()

	// ── 14. 加载前端静态文件（等待解压完成） ──
	go func() {
		defer utils.RecoverPanic()
		loadStaticFiles(apiRouter, workDir, assetsExtracted)
	}()

	// ── 15. 启动后台任务 ──
	service.StartBackgroundTasks()

	// ── 16. 获取配置端口，启动两个 HTTP 服务 ──
	apiPort := server.ResolvePort(service.PortNo, service.GetOSSetting().ControllerHost)
	apiSrv := server.CreateServer(apiPort, apiRouter)
	filePort := server.ResolvePort(service.FilePortNo, service.GetOSSetting().FileHost)
	fileSrv := server.CreateServer(filePort, fileRouter)

	var g errgroup.Group
	g.Go(func() error {
		defer utils.RecoverPanic()
		utils.InfoFormat("API 服务启动端口 %s", apiSrv.Addr)
		return apiSrv.ListenAndServe()
	})
	g.Go(func() error {
		defer utils.RecoverPanic()
		utils.InfoFormat("文件/图片流服务启动端口 %s", fileSrv.Addr)
		return fileSrv.ListenAndServe()
	})

	// ── 17. 优雅关闭监听（启动 goroutine 等信号，非阻塞） ──
	server.GracefulShutdown(sigChan, []*http.Server{apiSrv, fileSrv})

	// ── 19. 等待所有 HTTP 服务退出 ──
	if err := g.Wait(); err != nil && err != http.ErrServerClosed {
		utils.InfoFormat("服务异常: %v", err)
	} else {
		utils.InfoFormat("服务已停止")
	}

}
