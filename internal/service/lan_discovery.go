package service

import (
	"encoding/json"
	"fmt"
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
	IP       string `json:"ip"`       // 可连通的 IP（UDP来源 IP 经 TCP 验证）
	Name     string `json:"name"`     // 节点别名
	LastSeen int64  `json:"lastSeen"` // Unix 时间戳
	FilePort string `json:"filePort"` // 文件流端口，为空时默认 ":10082"
}

// heartbeatMsg UDP 心跳消息
type heartbeatMsg struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	Port     string `json:"port"`
	Name     string `json:"name"`
}

// LanDiscovery 局域网节点发现
type LanDiscovery struct {
	mu       sync.RWMutex
	peers    map[string]*Peer // key: NodeHost
	conn     *net.UDPConn
	stopChan chan struct{}
}

var (
	lanDiscovery = &LanDiscovery{
		peers:    make(map[string]*Peer),
		stopChan: make(chan struct{}),
	}
	stopDiscoveryOnce sync.Once
)

const (
	multicastAddr = "239.255.255.250:10083"
	defaultInterval = 30 * time.Second
	defaultTimeout  = 90 * time.Second
)

// StartLanDiscovery 启动局域网节点发现（由 main.go 调用）
func StartLanDiscovery() {
	initNodeInfo()
	if !IsClusterEnabled() {
		utils.InfoFormat("集群模式未启用")
		return
	}

	go func() {
		defer utils.RecoverPanic()
		if err := lanDiscovery.start(); err != nil {
			utils.ErrorFormat("LAN 节点发现启动失败: %v", err)
		}
	}()
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
		lanDiscovery.mu.Lock()
		lanDiscovery.peers[id] = &Peer{
			ID:       id,
			Hostname: ip,
			Port:     port,
			IP:       ip,
			Name:     ip,
			FilePort: filePort,
			LastSeen: time.Now().Unix(),
		}
		lanDiscovery.mu.Unlock()
		utils.InfoFormat("加载手动节点: %s (%s)", id, ip)
	}
}

// start 启动 UDP 组播监听和心跳发送
func (d *LanDiscovery) start() error {
	addr, err := net.ResolveUDPAddr("udp", multicastAddr)
	if err != nil {
		return fmt.Errorf("解析组播地址失败: %w", err)
	}

	// 监听组播
	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("监听组播失败: %w", err)
	}
	d.conn = conn

	utils.InfoFormat("LAN 节点发现已启动，组播地址: %s", multicastAddr)

	// 加入手动配置节点（无论组播是否成功，都加载）
	loadStaticPeers()

	// 启动时立即发送一次心跳
	d.sendHeartbeat()

	// 心跳发送协程
	go func() {
		defer utils.RecoverPanic()
		ticker := time.NewTicker(defaultInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				d.sendHeartbeat()
			case <-d.stopChan:
				return
			}
		}
	}()

	// 心跳监听协程
	go func() {
		defer utils.RecoverPanic()
		buf := make([]byte, 1024)
		for {
			select {
			case <-d.stopChan:
				return
			default:
			}
			n, src, err := conn.ReadFromUDP(buf)
			if err != nil {
				utils.ErrorFormat("组播接收失败: %v", err)
				continue
			}

			var msg heartbeatMsg
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				continue
			}

			// 忽略自己的消息
			if msg.ID == LocalNodeHost {
				continue
			}

			// TCP 验证可连通性
			peerIP := src.IP.String()
			if d.verifyPeer(peerIP, msg.Port) {
				d.upsertPeer(&Peer{
					ID:       msg.ID,
					Hostname: msg.Hostname,
					Port:     msg.Port,
					IP:       peerIP,
					Name:     msg.Name,
					LastSeen: time.Now().Unix(),
				})
			}
		}
	}()

	// 过期清理协程
	go func() {
		defer utils.RecoverPanic()
		ticker := time.NewTicker(defaultInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				d.cleanExpired(defaultTimeout)
			case <-d.stopChan:
				return
			}
		}
	}()

	return nil
}

// sendHeartbeat 发送心跳消息
func (d *LanDiscovery) sendHeartbeat() {
	if d.conn == nil {
		return
	}

	msg := heartbeatMsg{
		ID:       LocalNodeHost,
		Hostname: getHostname(),
		Port:     strings.TrimPrefix(consts.PortNo, ":"),
		Name:     LocalNodeName,
	}
	data, _ := json.Marshal(msg)

	addr, err := net.ResolveUDPAddr("udp", multicastAddr)
	if err != nil {
		return
	}
	d.conn.WriteTo(data, addr)
}

// verifyPeer HTTP 验证对端搜索服务是否可连通
func (d *LanDiscovery) verifyPeer(ip string, port string) bool {
	port = strings.TrimPrefix(port, ":")
	url := fmt.Sprintf("http://%s:%s/api/heartBeat", ip, port)
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}
	// 增加 header 跳过远程节点的认证检查
	req.Header.Set("X-Search-Gin-Remote", "true")
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// upsertPeer 更新或添加节点
func (d *LanDiscovery) upsertPeer(p *Peer) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.peers[p.ID] = p
}

// cleanExpired 清理超时节点
func (d *LanDiscovery) cleanExpired(timeout time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now()
	for id, p := range d.peers {
		lastSeen := time.Unix(p.LastSeen, 0)
		if now.Sub(lastSeen) > timeout {
			utils.InfoFormat("节点超时离线: %s (%s)", id, p.IP)
			delete(d.peers, id)
		}
	}
}

// GetOnlinePeers 获取在线节点列表
func GetOnlinePeers() []*Peer {
	lanDiscovery.mu.RLock()
	defer lanDiscovery.mu.RUnlock()
	result := make([]*Peer, 0, len(lanDiscovery.peers))
	for _, p := range lanDiscovery.peers {
		result = append(result, p)
	}
	return result
}

// AddPeer 动态添加节点（手动添加）
func AddPeer(ip, port, filePort string) bool {
	if port == "" {
		port = "10081"
	}
	if filePort == "" {
		filePort = "10082"
	}
	// TCP 验证可连通性
	if !lanDiscovery.verifyPeer(ip, port) {
		return false
	}
	id := fmt.Sprintf("%s:%s", ip, port)
	lanDiscovery.upsertPeer(&Peer{
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
	curDir, _ := os.Getwd()
	setting := consts.GetOSSetting()
	FlushDictionary(curDir + utils.PathSeparator + setting.SelfPath)
	utils.InfoFormat("手动添加节点成功: %s (%s)", id, ip)
	return true
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
	lanDiscovery.mu.RLock()
	defer lanDiscovery.mu.RUnlock()
	if p, ok := lanDiscovery.peers[nodeHost]; ok {
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

// StopLanDiscovery 停止节点发现（可安全多次调用）
func StopLanDiscovery() {
	stopDiscoveryOnce.Do(func() {
		close(lanDiscovery.stopChan)
		if lanDiscovery.conn != nil {
			lanDiscovery.conn.Close()
		}
	})
}
