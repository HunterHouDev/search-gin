package main

import (
	"context"
	"search-gin/internal/router"
	"search-gin/internal/service"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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

var (
	g errgroup.Group
)

// 创建具有标准配置的HTTP服务器
func createServer(addr string, handler http.Handler) *http.Server {
	// 不配置IP地址 仅监听本地端口  （双栈系统 同时有 IPv4 和 IPv6）
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

func main() {
	defer utils.RecoverPanic()
	// 解压静态资源到当前目录
	tempDir := "."
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
	go func() {
		defer utils.RecoverPanic()
		if _, err := os.Stat(filepath.Join(tempDir, "dist", "index.html")); os.IsNotExist(err) {
			utils.InfoFormat("开始解压前端静态文件...")
			if err := ExtractDist(tempDir); err != nil {
				utils.InfoFormat("解压前端静态文件失败: %v", err)
				os.Exit(1)
			}
			utils.InfoFormat("前端静态文件解压完成")
		} else {
			utils.InfoFormat("前端静态文件已存在，跳过解压")
		}
		if _, err := os.Stat(filepath.Join(tempDir, "ffmpeg.exe")); os.IsNotExist(err) {
			utils.InfoFormat("开始解压 ffmpeg.exe...")
			if err := ExtractFfmpeg(tempDir); err != nil {
				utils.InfoFormat("解压 ffmpeg.exe 失败: %v", err)
				os.Exit(1)
			}
			utils.InfoFormat("ffmpeg.exe 解压完成")
		} else {
			utils.InfoFormat("ffmpeg.exe 已存在，跳过解压")
		}
		if _, err := os.Stat(filepath.Join(tempDir, "ffplay.exe")); os.IsNotExist(err) {
			utils.InfoFormat("开始解压 ffplay.exe...")
			if err := ExtractFfplay(tempDir); err != nil {
				utils.InfoFormat("解压 ffplay.exe 失败: %v", err)
				os.Exit(1)
			}
			utils.InfoFormat("ffplay.exe 解压完成")
		} else {
			utils.InfoFormat("ffplay.exe 已存在，跳过解压")
		}
	}()
	// 设置临时目录到 service 包
	service.TempDir = tempDir

	go func() {
		defer utils.RecoverPanic()
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	// 创建信号通道
	sigChan := make(chan os.Signal, 1)
	// 监听SIGINT和SIGTERM信号
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	service.InitSetting()

	app := router.BuildRouter(tempDir)

	port_nos := []string{consts.PortNo, consts.PortNo2, consts.PortNo3}
	// port_nos := []string{":9901"}
	servers := make([]*http.Server, 3)
	for i, port_no := range port_nos {
		// 创建服务器
		servers[i] = createServer(port_no, app)
		// 启动服务
		g.Go(func() error {
			defer utils.RecoverPanic()
			utils.InfoFormat("启动端口%s ", port_no)
			return servers[i].ListenAndServe()
		})
	}
	// 注册关闭路由，使用命名函数避免参数泄漏警告
	app.GET("api/close", func(c *gin.Context) {
		c.String(200, "即将关闭所有服务器")
		// 发出终止信号
		sigChan <- syscall.SIGTERM
	})
	// 启动扫描系统
	go func() {
		defer utils.RecoverPanic()
		service.FileApp.HeartBeat()
	}()
	// 启动转换执行任务
	go func() {
		defer utils.RecoverPanic()
		service.FileApp.TaskExecuting()
	}()
	//默认启动页面
	// go utils.ExecCmdStart("http://127.0.0.1" + consts.PortNo + "/")

	// 等待信号或服务错误
	go func() {
		defer utils.RecoverPanic()
		sig := <-sigChan
		utils.InfoFormat("收到信号 %v，正在优雅关闭所有服务...", sig)
		// 执行优雅关闭逻辑
		// 1. 创建带超时的上下文
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 2. 关闭服务器
		for _, srv := range servers {
			srv.Close()
			log.Printf("端口 %s 已关闭", srv.Addr)
		}
		cancel()
		// 3. 等待关闭完成或超时
		<-ctx.Done()
		log.Println("服务已全部关闭")
	}()

	// 等待所有goroutine完成
	if err := g.Wait(); err != nil && err != http.ErrServerClosed {
		utils.InfoFormat("服务异常: %v", err)
	} else {
		utils.InfoFormat("服务已停止")
	}

}
