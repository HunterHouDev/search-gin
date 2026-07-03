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

// InitSetting 读取配置文件并初始化全局设置
func InitSetting() {
	// 先用默认值填充，确保基础字段不为空
SetOSSetting(defaultSetting())

curDir, err := filepath.Abs(".")
if err != nil {
	utils.ErrorFormat("获取当前目录失败: %v", err)
	curDir = "."
}
settingPath := filepath.Join(curDir, SettingFileName)

// 如果 setting.json 不存在，尝试从 setting.example.json 复制，也都不存在则自动生成
if _, err := os.Stat(settingPath); os.IsNotExist(err) {
	examplePath := filepath.Join(curDir, "setting.example.json")
	if _, err := os.Stat(examplePath); err == nil {
	 utils.InfoFormat("配置文件 %s 不存在，从 %s 复制", SettingFileName, "setting.example.json")
	 input, err := os.ReadFile(examplePath)
	 if err == nil {
	  _ = os.WriteFile(settingPath, input, 0600)
	 }
	} else {
	 utils.InfoFormat("配置文件 %s 和 %s 均不存在，自动生成默认配置", SettingFileName, "setting.example.json")
	 _ = WriteDictionaryToJson(settingPath, defaultSetting())
	}
}

dict := ReadDictionaryFromJson(settingPath)
if dict.ControllerHost == "" {
		dict.ControllerHost = PortNo
	}
	if dict.FileHost == "" {
		dict.FileHost = FilePortNo
	}

	// 多节点配置默认值
	if dict.EnableLanDiscovery == nil {
		dict.EnableLanDiscovery = newBool(true) // 默认启用
	}

	// 如果启用硬件加速，主动检测并同步模式名称
	if dict.HardwareAcceleration {
		detectHwAccel()
		dict.HardwareAccelMode = GetHwAccelModeName()
	}

	SetOSSetting(dict)

	// 预缓存管理员密码的 bcrypt 哈希，避免每次登录时重复计算
	CacheAdminPasswordHash()

	// StreamSecret：优先使用 setting.json 中持久化的密钥，不存在时生成并保存
	if dict.StreamSecret != "" {
		utils.SetStreamSecret(dict.StreamSecret)
		utils.InfoFormat("已加载持久化 StreamSecret")
	} else {
		secret := utils.GenerateStreamSecret()
		utils.SetStreamSecret(secret)
		dict.StreamSecret = secret
		SetOSSetting(dict)
		// 立即写入磁盘，确保密钥持久化（失败则 panic，防止重启后密钥丢失）
		 if err := FlushDictionary(filepath.Join(curDir, SettingFileName)); err != nil {
			panic("StreamSecret 持久化失败，无法安全启动: " + err.Error())
		}
		utils.InfoFormat("已生成并持久化 StreamSecret")
	}
}

// InitSearchPool 初始化 goroutine 池，根据配置的目录数量动态调整
// 必须在 GetOSSetting() 和 SearchEngine 初始化之后调用
func InitSearchPool() {
	dirCount := len(GetOSSetting().Dirs)
	poolSize := min(max(dirCount, 4), 50)
	// （当前仅 main.go 调用一次，无活跃泄漏）
	GetEngine().searchPool = utils.NewGoroutinePool(poolSize)
	GetEngine().KeywordHistoryCache = utils.NewLRUCache(500)
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
	utils.InfoFormat("StartBackgroundTasks: 正在启动后台任务...")

	setting := GetOSSetting()
	InitTaskSlots(setting.TaskMaxConcurrent)

	search := GetSearch()
	if search == nil {
		utils.ErrorFormat("StartBackgroundTasks: GetSearch() 返回 nil，后台任务无法启动")
		return
	}

	go func() {
		defer utils.RecoverPanic()
		search.HeartBeat()
	}()
	go func() {
		defer utils.RecoverPanic()
		search.TaskScheduler()
	}()
	// token 到期自毁由 SetToken 中的 time.AfterFunc 驱动，无需定时轮询
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
	go func() {
		defer utils.RecoverPanic()
		TorrentApp.StartCleanup(ctx)
	}()

	return func() {
		cancel()
		TorrentApp.Close()
	}
}
