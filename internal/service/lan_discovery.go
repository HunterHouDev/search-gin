package service

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"search-gin/internal/model"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"strings"
	"sync"
	"time"
)

var (
	// LocalNodeHost 本机节点标识 "hostname:port"
	LocalNodeHost string
	// LocalNodeName 本机节点可读别名
	LocalNodeName string
)

// Peer 远程节点信息
type Peer struct {
	ID       string `json:"id"`       // "PC-A:10081"
	Hostname string `json:"hostname"` // "PC-A"
	Port     string `json:"port"`     // "10081" API 端口
	IP       string `json:"ip"`       // 可连通的 IP
	Name     string `json:"name"`     // 节点别名
	LastSeen int64  `json:"lastSeen"` // Unix 时间戳
	FilePort string `json:"filePort"` // 文件流端口，为空时默认 ":10082"
	Disabled bool   `json:"disabled"` // 是否禁用，禁用的节点不会被搜索
}

// peerManager 节点管理器
type peerManager struct {
	mu    sync.RWMutex
	peers map[string]*Peer // key: NodeHost
}

var (
	defaultManager *peerManager
)

const defaultPeerTimeout = 90 * time.Second

// InitPeerManager 初始化节点管理器，从配置加载静态节点
func InitPeerManager() {
	defaultManager = &peerManager{
		peers: make(map[string]*Peer),
	}
	initNodeInfo()
	loadStaticPeers()
	utils.InfoFormat("节点管理器已初始化，本机: %s (%s)", LocalNodeHost, LocalNodeName)
}

// IsClusterEnabled 集群模式是否启用
// nil（未配置）→ 默认启用；*false → 禁用；*true → 启用
func IsClusterEnabled() bool {
	s := consts.GetOSSetting()
	return s.EnableLanDiscovery == nil ||
		*s.EnableLanDiscovery
}

// initNodeInfo 初始化本机节点信息
func initNodeInfo() {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	port := consts.PortNo
	if strings.HasPrefix(port, ":") {
		port = port[1:]
	}
	LocalNodeHost = fmt.Sprintf("%s:%s", hostname, port)

	setting := consts.GetOSSetting()
	if setting.NodeName != "" {
		LocalNodeName = setting.NodeName
	} else {
		LocalNodeName = hostname
	}

	utils.InfoFormat("本机节点信息: NodeHost=%s, NodeName=%s", LocalNodeHost, LocalNodeName)
}

// loadStaticPeers 加载手动配置节点
func loadStaticPeers() {
	if defaultManager == nil {
		return
	}
	setting := consts.GetOSSetting()
	for _, addr := range setting.DiscoveryPeers {
		parts := strings.Split(addr, ":")
		if len(parts) < 2 {
			continue
		}
		ip := parts[0]
		port := parts[1]
		filePort := ""
		if len(parts) >= 3 {
			filePort = parts[2]
		}
		id := fmt.Sprintf("%s:%s", ip, port)
		defaultManager.mu.Lock()
		defaultManager.peers[id] = &Peer{
			ID:       id,
			Hostname: ip,
			Port:     port,
			IP:       ip,
			Name:     ip,
			FilePort: filePort,
			LastSeen: time.Now().Unix(),
		}
		defaultManager.mu.Unlock()
		utils.InfoFormat("加载手动节点: %s (%s)", id, ip)
	}
}

// CleanExpiredPeers 手动清理超时节点
func CleanExpiredPeers() int {
	if defaultManager == nil {
		return 0
	}
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()
	now := time.Now()
	var expired []string
	for id, p := range defaultManager.peers {
		lastSeen := time.Unix(p.LastSeen, 0)
		if now.Sub(lastSeen) > defaultPeerTimeout {
			expired = append(expired, id)
		}
	}
	for _, id := range expired {
		delete(defaultManager.peers, id)
	}
	return len(expired)
}

// verifyPeer HTTP 验证对端搜索服务是否可连通
func (m *peerManager) verifyPeer(ip string, port string) bool {
	url := fmt.Sprintf("http://%s:%s/api/heartBeat", ip, port)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == http.StatusOK
}

// upsertPeer 更新或添加节点
func (m *peerManager) upsertPeer(p *Peer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.peers[p.ID] = p
}

// updateLastSeen 更新已有节点的 LastSeen 和 IP（线程安全）
func (m *peerManager) updateLastSeen(id, ip string, lastSeen int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.peers[id]; ok {
		p.LastSeen = lastSeen
		p.IP = ip
	}
}

// GetOnlinePeers 获取在线节点列表
func GetOnlinePeers() []*Peer {
	if defaultManager == nil {
		return nil
	}
	defaultManager.mu.RLock()
	defer defaultManager.mu.RUnlock()
	result := make([]*Peer, 0, len(defaultManager.peers))
	for _, p := range defaultManager.peers {
		result = append(result, p)
	}
	return result
}

// IsKnownPeerIP 判断指定 IP 是否属于集群内已知节点（含本机回环）
// 用于 AuthMiddleware 校验 X-Search-Gin-Remote 请求的来源是否可信
func IsKnownPeerIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed != nil && parsed.IsLoopback() {
		return true
	}

	if defaultManager == nil {
		return false
	}

	defaultManager.mu.RLock()
	defer defaultManager.mu.RUnlock()

	for _, p := range defaultManager.peers {
		if p.Disabled {
			continue
		}
		if p.IP == ip {
			return true
		}
	}
	return false
}

// TryVerifyAndAddPeer 向指定 IP 发起反向心跳验证，通过则自动加入集群
// 用于 AuthMiddleware 首次遇到未知 IP 时自动发现
func TryVerifyAndAddPeer(ip string) bool {
	if defaultManager == nil {
		return false
	}
	port := strings.TrimPrefix(consts.PortNo, ":")
	filePort := strings.TrimPrefix(consts.FilePortNo, ":")
	return AddPeer(ip, port, filePort)
}

// GetPeerStats 获取指定节点的文件总数和总大小
func GetPeerStats(nodeID string) (totalCnt int, totalSize string, nodeName string) {
	if defaultManager == nil {
		return 0, "", ""
	}
	defaultManager.mu.RLock()
	p, ok := defaultManager.peers[nodeID]
	defaultManager.mu.RUnlock()
	if !ok {
		return 0, "", ""
	}
	param := model.SearchParam{Keyword: "", Page: 1, PageSize: 1}
	result, err := SearchRemotePeer(p, param)
	if err != nil {
		return 0, "", p.Name
	}
	return result.TotalCnt, result.TotalSize, p.Name
}

// AddPeer 动态添加节点（手动添加）
func AddPeer(ip, port, filePort string) bool {
	if defaultManager == nil {
		return false
	}
	if port == "" {
		port = "10081"
	}
	if filePort == "" {
		filePort = "10082"
	}
	// TCP 验证可连通性
	if !defaultManager.verifyPeer(ip, port) {
		return false
	}
	id := fmt.Sprintf("%s:%s", ip, port)
	defaultManager.upsertPeer(&Peer{
		ID:       id,
		Hostname: ip,
		Port:     port,
		IP:       ip,
		Name:     ip,
		FilePort: filePort,
		LastSeen: time.Now().Unix(),
	})
	// 持久化到 setting.json，重启后自动加载
	addr := fmt.Sprintf("%s:%s:%s", ip, port, filePort)
	consts.UpdateOSSetting(func(s model.Setting) model.Setting {
		for _, v := range s.DiscoveryPeers {
			if v == addr {
				return s // 已存在
			}
		}
		s.DiscoveryPeers = append(s.DiscoveryPeers, addr)
		return s
	})
	// 刷新配置文件
	curDir, err := os.Getwd()
	if err != nil {
		utils.ErrorFormat("获取当前目录失败: %v", err)
		return false
	}
	setting := consts.GetOSSetting()
	FlushDictionary(curDir + utils.PathSeparator + setting.SelfPath)
	utils.InfoFormat("手动添加节点成功: %s (%s)", id, ip)
	return true
}

// RemovePeer 删除节点（从内存和配置文件中移除）
func RemovePeer(id string) bool {
	if defaultManager == nil {
		return false
	}
	defaultManager.mu.Lock()
	if _, ok := defaultManager.peers[id]; !ok {
		defaultManager.mu.Unlock()
		return false
	}
	delete(defaultManager.peers, id)
	defaultManager.mu.Unlock()

	// 从 setting.json 的 discoveryPeers 中移除
	consts.UpdateOSSetting(func(s model.Setting) model.Setting {
		var keep []string
		for _, v := range s.DiscoveryPeers {
			if v != id && !strings.HasPrefix(v, id+":") {
				keep = append(keep, v)
			}
		}
		s.DiscoveryPeers = keep
		return s
	})
	curDir, err := os.Getwd()
	if err != nil {
		utils.ErrorFormat("获取当前目录失败: %v", err)
		return false
	}
	setting := consts.GetOSSetting()
	FlushDictionary(curDir + utils.PathSeparator + setting.SelfPath)
	utils.InfoFormat("删除节点: %s", id)
	return true
}

// TogglePeerDisabled 启用/禁用节点
func TogglePeerDisabled(id string, disabled bool) bool {
	if defaultManager == nil {
		return false
	}
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()
	if p, ok := defaultManager.peers[id]; ok {
		p.Disabled = disabled
		utils.InfoFormat("节点 %s 状态: %v", id, map[bool]string{false: "启用", true: "禁用"}[disabled])
		return true
	}
	return false
}

// ResolvePeerIP 从 NodeHost 解析对端 IP
func ResolvePeerIP(nodeHost string) string {
	if p := GetPeer(nodeHost); p != nil {
		return p.IP
	}
	return ""
}

// GetPeer 从 NodeHost 获取完整 Peer 信息
func GetPeer(nodeHost string) *Peer {
	if defaultManager == nil {
		return nil
	}
	defaultManager.mu.RLock()
	defer defaultManager.mu.RUnlock()
	if p, ok := defaultManager.peers[nodeHost]; ok {
		return p
	}
	return nil
}

// SetMovieNode 为 Movie 设置节点信息
func SetMovieNode(m *model.FileItem) {
	m.NodeHost = LocalNodeHost
	m.NodeName = LocalNodeName
}

func getHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
