package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"strconv"
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
	defaultManager   *peerManager
	peerVerifyClient = &http.Client{Timeout: 2 * time.Second}
)

const defaultPeerTimeout = 90 * time.Second

// IsClusterEnabled 集群模式是否启用
// nil（未配置）→ 默认启用；*false → 禁用；*true → 启用
func IsClusterEnabled() bool {
	s := GetOSSetting()
	return s.EnableLanDiscovery == nil ||
		*s.EnableLanDiscovery
}

// initNodeInfo 初始化本机节点信息
func initNodeInfo() {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	port := strings.TrimPrefix(PortNo, ":")
	LocalNodeHost = fmt.Sprintf("%s:%s", hostname, port)

	setting := GetOSSetting()
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
	setting := GetOSSetting()
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
	resp, err := peerVerifyClient.Get(url)
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

// GetOnlinePeers 获取在线节点列表（深拷贝）
func GetOnlinePeers() []*Peer {
	if defaultManager == nil {
		return nil
	}
	defaultManager.mu.RLock()
	defer defaultManager.mu.RUnlock()
	result := make([]*Peer, 0, len(defaultManager.peers))
	for _, p := range defaultManager.peers {
		cp := *p
		result = append(result, &cp)
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
	UpdateOSSetting(func(s model.Setting) model.Setting {
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
	setting := GetOSSetting()
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
	UpdateOSSetting(func(s model.Setting) model.Setting {
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
	setting := GetOSSetting()
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

// DiscoveredPeer 发现到的候选节点
type DiscoveredPeer struct {
	IP       string `json:"ip"`
	Port     string `json:"port"`
	FilePort string `json:"filePort"`
	NodeName string `json:"nodeName"`
}

// GetLocalSubnet 探测本机第一个合适的 /24 子网前缀，返回如 "192.168.1"
func GetLocalSubnet() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.To4() == nil {
				continue
			}
			mask := ipnet.Mask
			if ones, bits := mask.Size(); ones != 24 || bits != 32 {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil {
				continue
			}
			// 跳过私有地址段：10.0.0.0/8, 100.64.0.0/10, 172.16.0.0/12
			if ip4[0] == 10 || ip4[0] == 100 || (ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) {
				continue
			}
			base := ipnet.IP.String()
			idx := strings.LastIndex(base, ".")
			if idx < 0 {
				continue
			}
			return base[:idx]
		}
	}
	return ""
}

// DiscoverLanPeers 扫描局域网发现 search-gin 节点
// subnet: 指定子网前缀如 "192.168.1"，为空时自动探测本机子网
// 返回发现列表和本机子网前缀
func DiscoverLanPeers(subnet string) ([]DiscoveredPeer, string) {
	localPrefix := GetLocalSubnet()
	if subnet == "" {
		subnet = localPrefix
	}

	if subnet == "" {
		utils.InfoFormat("LAN 发现：未指定子网且未找到合适的本地子网")
		return nil, ""
	}

	// 校验格式：三段 IP 前缀如 "192.168.1" 扫 /24，四段完整 IP 单机检测
	parts := strings.Split(subnet, ".")
	if len(parts) == 4 {
		// 单 IP 检测
		for _, p := range parts {
			if n, err := strconv.Atoi(p); err != nil || n < 0 || n > 255 {
				utils.InfoFormat("LAN 发现：IP 格式错误 %q", subnet)
				return nil, localPrefix
			}
		}
		return checkSingleHost(subnet), localPrefix
	}
	if len(parts) != 3 {
		utils.InfoFormat("LAN 发现：子网格式错误 %q，需要三段 IP 前缀如 192.168.1 或完整 IP", subnet)
		return nil, localPrefix
	}
	for _, p := range parts {
		if n, err := strconv.Atoi(p); err != nil || n < 0 || n > 255 {
			utils.InfoFormat("LAN 发现：子网格式错误 %q，包含非法数字", subnet)
			return nil, localPrefix
		}
	}

	base := subnet + "."
	defaultPort := strings.TrimPrefix(PortNo, ":")
	filePort := strings.TrimPrefix(FilePortNo, ":")
	discoverTimeout := 2 * time.Second

	type result struct {
		ip       string
		ok       bool
	}

	// 共享 transport 复用连接
	sharedTransport := &http.Transport{
		MaxIdleConnsPerHost: 100,
	}
	sharedClient := &http.Client{
		Timeout:   discoverTimeout,
		Transport: sharedTransport,
	}
	defer sharedClient.CloseIdleConnections()

	results := make(chan result, 256)
	sem := make(chan struct{}, 20)
	var wg sync.WaitGroup

	for i := 1; i <= 254; i++ {
		targetIP := fmt.Sprintf("%s%d", base, i)

		sem <- struct{}{}
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()

			url := fmt.Sprintf("http://%s:%s/api/heartBeat", ip, defaultPort)
			resp, err := sharedClient.Get(url)
			if err != nil {
				return
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				results <- result{ip: ip, ok: true}
			}
		}(targetIP)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var discovered []DiscoveredPeer
	for r := range results {
		if r.ok {
			nodeName := r.ip
			infoURL := fmt.Sprintf("http://%s:%s/api/lanPeers", r.ip, defaultPort)
			if resp, err := sharedClient.Get(infoURL); err == nil {
				var info struct {
					LocalNodeHost string `json:"localNodeHost"`
					LocalNodeName string `json:"localNodeName"`
				}
				if json.NewDecoder(resp.Body).Decode(&info) == nil && info.LocalNodeName != "" {
					nodeName = info.LocalNodeName
				}
				resp.Body.Close()
			}

			discovered = append(discovered, DiscoveredPeer{
				IP:       r.ip,
				Port:     defaultPort,
				FilePort: filePort,
				NodeName: nodeName,
			})
		}
	}

	return discovered, localPrefix
}

// checkSingleHost 检测单个 IP 是否为 search-gin 节点
func checkSingleHost(ip string) []DiscoveredPeer {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return nil
	}
	defaultPort := strings.TrimPrefix(PortNo, ":")
	filePort := strings.TrimPrefix(FilePortNo, ":")
	client := &http.Client{Timeout: 3 * time.Second}

	url := fmt.Sprintf("http://%s:%s/api/heartBeat", ip, defaultPort)
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	// 获取节点别名
	nodeName := ip
	infoURL := fmt.Sprintf("http://%s:%s/api/lanPeers", ip, defaultPort)
	if r, err := client.Get(infoURL); err == nil {
		var info struct {
			LocalNodeName string `json:"localNodeName"`
		}
		if json.NewDecoder(r.Body).Decode(&info) == nil && info.LocalNodeName != "" {
			nodeName = info.LocalNodeName
		}
		r.Body.Close()
	}

	return []DiscoveredPeer{{
		IP:       ip,
		Port:     defaultPort,
		FilePort: filePort,
		NodeName: nodeName,
	}}
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


