package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"search-gin/internal/router"
	"search-gin/internal/service"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	_ "net/http/pprof"

	"golang.org/x/sync/errgroup"
)

//go:generate go env -w GO111MODULE=on
//go:generate go env -w GOPROXY=https://goproxy.cn,direct
//go:generate go mod tidy
//go:generate go mod download

// 打包命令
// 1 命令行UI 常规打包 go build
// 2 命令行UI 常规打包 go build windows
// 2 无窗口  go build -o viteApp/appVite.exe -ldflags  "-H=windowsgui" -tags=prod
// 3 命令行UI 常规打包 go build linux
// 3 无窗口  go build -o viteApp/appVite.exe -ldflags  "-H=windowsgui" -tags=prod

var g errgroup.Group

// createServer 创建具有标准超时配置的HTTP服务器
func createServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

// extractAssets 解压嵌入式资源（setting.json、前端静态文件、ffmpeg/ffplay）
func extractAssets(tempDir string) chan struct{} {
 assetsExtracted := make(chan struct{})

 // 解压 setting.json
 if _, err := os.Stat(filepath.Join(tempDir, "setting.json")); os.IsNotExist(err) {
  utils.InfoFormat("开始解压 setting.json...")
  if err := ExtractSetting(tempDir); err != nil {
   utils.InfoFormat("解压 setting.json 失败: %v", err)
   os.Exit(1)
  }
  utils.InfoFormat("setting.json 解压完成")
 } else {
  utils.InfoFormat("setting.json 已存在，跳过解压")
 }

 // 异步解压前端资源和二进制工具
 go func() {
  defer utils.RecoverPanic()
  defer close(assetsExtracted)
  extractEmbeddedAssets(tempDir)
 }()

 return assetsExtracted
}

// extractEmbeddedAssets 解压 dist、ffmpeg、ffplay
func extractEmbeddedAssets(tempDir string) {
	assets := []struct {
		path string
		name string
		fn   func(string) error
	}{
		{filepath.Join("dist", "index.html"), "前端静态文件", ExtractDist},
		{"ffmpeg.exe", "ffmpeg.exe", ExtractFfmpeg},
		{"ffplay.exe", "ffplay.exe", ExtractFfplay},
	}
	for _, a := range assets {
		if _, err := os.Stat(filepath.Join(tempDir, a.path)); os.IsNotExist(err) {
			utils.InfoFormat("开始解压 %s...", a.name)
			if err := a.fn(tempDir); err != nil {
				utils.InfoFormat("解压 %s 失败: %v", a.name, err)
				os.Exit(1)
			}
			utils.InfoFormat("%s 解压完成", a.name)
		} else {
			utils.InfoFormat("%s 已存在，跳过解压", a.name)
		}
	}
}

// startPprof 开发环境下启动 pprof 调试接口
func startPprof() {
	if os.Getenv("GIN_MODE") != "release" {
		go func() {
			defer utils.RecoverPanic()
			log.Println("pprof调试接口启动在 localhost:6060")
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	} else {
		log.Println("生产环境已禁用pprof调试接口")
	}
}

// startBackgroundTasks 启动所有后台 goroutine（心跳、任务执行、token清理等）
func startBackgroundTasks(app *gin.Engine, sigChan chan os.Signal) {
	// 心跳扫描
	go func() {
		defer utils.RecoverPanic()
		service.FileApp.HeartBeat()
	}()
	// 转换任务执行
	go func() {
		defer utils.RecoverPanic()
		service.FileApp.TaskExecuting()
	}()
	// 定时清理过期 token
	go func() {
		defer utils.RecoverPanic()
		tokenCleanupLoop()
	}()
}

// tokenCleanupLoop 定期清理过期 token
func tokenCleanupLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			consts.CleanExpiredTokens()
		case <-service.TaskCtx.Done():
			utils.InfoFormat("token清理协程已停止")
			return
		}
	}
}

// gracefulShutdown 监听信号并执行优雅关闭
func gracefulShutdown(sigChan chan os.Signal, servers []*http.Server) {
	go func() {
		defer utils.RecoverPanic()
		sig := <-sigChan
		utils.InfoFormat("收到信号 %v，正在优雅关闭所有服务...", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		for _, srv := range servers {
			srv.Close()
			log.Printf("端口 %s 已关闭", srv.Addr)
		}
		service.TaskCancel()
		<-ctx.Done()
		log.Println("服务已全部关闭")
	}()
}

func main() {
	defer utils.RecoverPanic()

	// 1. 初始化工作目录
	tempDir, err := os.Getwd()
	if err != nil {
		utils.InfoFormat("获取当前工作目录失败: %v，使用默认路径", err)
		tempDir = "."
	}
	service.TempDir = tempDir

	// 2. 解压嵌入式资源
	assetsExtracted := extractAssets(tempDir)

	// 3. 启动 pprof（开发环境）
	startPprof()

	// 4. 信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 5. 初始化配置和扫描队列
	service.InitSetting()
	service.StartScanQueue()

	// 6. 启动 Torrent 服务
	torrentDir := filepath.Join(tempDir, "torrent_data")
	os.MkdirAll(torrentDir, 0755)
	if err := service.NewTorrentService(torrentDir); err != nil {
		utils.InfoFormat("Torrent 服务启动失败: %v", err)
	} else {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go service.TorrentApp.StartCleanup(ctx)
		defer service.TorrentApp.Close()
	}

	// 7. 构建路由
	app := router.BuildRouter(tempDir)

	// 8. 加载前端静态文件（等待解压完成）
	go loadStaticFiles(app, tempDir, assetsExtracted)

	// 9. 启动多端口 HTTP 服务
	portNos := []string{consts.PortNo, consts.PortNo2, consts.PortNo3}
	servers := make([]*http.Server, len(portNos))
	for i, port := range portNos {
		srv := createServer(port, app)
		servers[i] = srv
		g.Go(func() error {
			defer utils.RecoverPanic()
			utils.InfoFormat("启动端口%s", srv.Addr)
			return srv.ListenAndServe()
		})
	}

	// 10. 注册关闭接口
	app.GET("api/close", func(c *gin.Context) {
		c.String(200, "即将关闭所有服务器")
		sigChan <- syscall.SIGTERM
	})

	// 11. 启动后台任务
	startBackgroundTasks(app, sigChan)

	// 12. 优雅关闭监听
	gracefulShutdown(sigChan, servers)

	// 13. 等待所有 HTTP 服务退出
	if err := g.Wait(); err != nil && err != http.ErrServerClosed {
		utils.InfoFormat("服务异常: %v", err)
	} else {
		utils.InfoFormat("服务已停止")
	}
}

// loadStaticFiles 加载前端静态文件（延迟执行以等待解压）
func loadStaticFiles(app *gin.Engine, tempDir string, extracted <-chan struct{}) {
	<-extracted
	indexHtml := filepath.Join(tempDir, "dist", "index.html")
	if !utils.ExistsFiles(indexHtml) {
		utils.InfoFormat("static not exists:%s", indexHtml)
		return
	}
	utils.InfoFormat("static exists:%s", indexHtml)
	app.LoadHTMLFiles(indexHtml)
	staticFs := map[string]string{
		"/css":    filepath.Join(tempDir, "dist", "css"),
		"/js":     filepath.Join(tempDir, "dist", "js"),
		"/assets": filepath.Join(tempDir, "dist", "assets"),
	}
	for k, v := range staticFs {
		app.StaticFS(k, http.Dir(v))
		utils.InfoFormat("static exists:%s", k)
	}
}
