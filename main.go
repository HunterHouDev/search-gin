package main

import (
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"search-gin/internal/env"
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
	utils.SetLogLevel(env.IsProd)
	workDir, err := os.Getwd()
	if err != nil {
		utils.InfoFormat("获取当前工作目录失败: %v，使用默认路径", err)
		workDir = "."
	}
	service.WorkDir = workDir

	// ── 2.1 加载上次扫描的索引缓存（填补启动空窗期） ──
	service.SearchEngine.LoadCachedIndex()

	// ── 2. 解压嵌入式资源 ──
	assetsExtracted := extractAssets(workDir)

	// ── 3. 启动 pprof（生产环境通过 env.IsProd 自动禁用） ──
	service.StartPprof()

	// ── 4. 信号通道 ──
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// ── 5. 初始化配置和扫描队列 ──
	service.InitSetting()
	service.InitSearchPool()
	service.StartScanQueue()

	// ── 5.5 初始化节点管理器（手动添加 + 反向心跳自动发现） ──
	service.InitPeerManager()

	// ── 6. 启动 Torrent 清理 ──
	closeTorrent := service.StartTorrentCleanup(workDir)
	defer closeTorrent()

	// ── 7. 构建路由（API 路由 + 文件流路由） ──
	apiRouter := router.BuildAPIRouter()
	fileRouter := router.BuildFileRouter()

	// ── 8. 加载前端静态文件（等待解压完成） ──
	go loadStaticFiles(apiRouter, workDir, assetsExtracted)

	// ── 9. 启动后台任务 ──
	service.StartBackgroundTasks()

	// ── 10. 获取配置端口，启动两个 HTTP 服务 ──
	apiPort := resolvePort(service.GetOSSetting().ControllerHost)
	apiSrv := createServer(apiPort, apiRouter)

	filePort := consts.FilePortNo
	fileSrv := createServer(filePort, fileRouter)

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

	// ── 11. 注册 /api/close 和 /api/restart 接口 ──
	apiRouter.GET("api/close", func(c *gin.Context) {
		role, _ := c.Get("role")
		if r, ok := role.(string); !ok || r != service.AdminRole {
			c.JSON(403, utils.NewFailByMsg("无权限执行此操作"))
			return
		}
		c.String(200, "即将关闭服务器")
		sigChan <- syscall.SIGTERM
	})
	apiRouter.GET("api/restart", func(c *gin.Context) {
		role, _ := c.Get("role")
		if r, ok := role.(string); !ok || r != service.AdminRole {
			c.JSON(403, utils.NewFailByMsg("无权限执行此操作"))
			return
		}
		c.String(200, "正在重启服务器")
		go func() {
			defer utils.RecoverPanic()
			time.Sleep(200 * time.Millisecond)
			// 先通知旧进程关闭，等待端口释放再启动新进程
			sigChan <- syscall.SIGTERM
			time.Sleep(2 * time.Second)
			exe, err := os.Executable()
			if err != nil {
				utils.ErrorFormat("获取可执行文件路径失败: %v", err)
				return
			}
			cmd := exec.Command(exe, os.Args[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Start()
		}()
	})

	// ── 12. 优雅关闭监听 ──
	gracefulShutdown(sigChan, []*http.Server{apiSrv, fileSrv})

	// 通知后台任务停止
	service.TaskCancel()

	// ── 13. 等待所有 HTTP 服务退出 ──
	if err := g.Wait(); err != nil && err != http.ErrServerClosed {
		utils.InfoFormat("服务异常: %v", err)
	} else {
		utils.InfoFormat("服务已停止")
	}
}
