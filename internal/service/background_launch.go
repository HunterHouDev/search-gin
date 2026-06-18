package service

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"search-gin/internal/env"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
)

// InitSetting 读取配置文件并初始化全局设置
func InitSetting() {
	curDir, err := filepath.Abs(".")
	if err != nil {
		utils.ErrorFormat("获取当前目录失败: %v", err)
		curDir = "."
	}
	osSetting := consts.GetOSSetting()
	settingPath := filepath.Join(curDir, osSetting.SelfPath)
	dict := ReadDictionaryFromJson(settingPath)
	dict.SelfPath = osSetting.SelfPath
	if dict.ControllerHost == "" {
		dict.ControllerHost = consts.PortNo
	}
	if dict.FileHost == "" {
		dict.FileHost = osSetting.FileHost
	}

	// 多节点配置默认值
	if dict.EnableLanDiscovery == nil {
		dict.EnableLanDiscovery = newBool(true) // 默认启用
	}

	// 如果启用硬件加速，主动检测并同步模式名称
	if dict.HardwareAcceleration {
		VideoEncoder.detectHwAccel()
		dict.HardwareAccelMode = GetHwAccelModeName()
	}

	consts.SetOSSetting(dict)
}

// InitSearchPool 初始化 goroutine 池，根据配置的目录数量动态调整
// 必须在 consts.GetOSSetting() 和 SearchEngine 初始化之后调用
func InitSearchPool() {
	dirCount := len(consts.GetOSSetting().Dirs)
	poolSize := dirCount
	if poolSize < 4 {
		poolSize = 4
	}
	if poolSize > 50 {
		poolSize = 50
	}
	SearchEngine.searchPool = utils.NewGoroutinePool(poolSize)
	SearchEngine.KeywordHistoryCache = utils.NewLRUCache(10)
}

// InitPeerManager 初始化节点管理器，从配置加载静态节点
func InitPeerManager() {
	defaultManager = &peerManager{
		peers: make(map[string]*Peer),
	}
	initNodeInfo()
	loadStaticPeers()
	utils.InfoFormat("节点管理器已初始化，本机: %s (%s)", LocalNodeHost, LocalNodeName)
}

// StartScanQueue 启动扫描任务队列处理器（由 main.go 在初始化完成后显式调用）
func StartScanQueue() {
	go scanQueue.processTasks()
}

// StartPprof 开发环境下启动 pprof 调试接口
func StartPprof() {
	if env.IsProd {
		utils.InfoFormat("生产环境已禁用 pprof 调试接口")
		return
	}
	go func() {
		defer utils.RecoverPanic()
		utils.InfoFormat("pprof 调试接口启动在 localhost:6060")
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
		utils.ErrorFormat("Torrent 服务启动失败: %v", err)
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	go TorrentApp.StartCleanup(ctx)

	return func() {
		cancel()
		TorrentApp.Close()
	}
}
