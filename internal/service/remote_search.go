package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	maxConcurrentPeers  = 5
	remoteSearchTimeout = 2 * time.Second
)

// PeerSearchResult 远程节点搜索结果
type PeerSearchResult struct {
	Movies    []model.FileItem
	TotalCnt  int
	TotalSize int64
}

// SearchPeers 并发搜索所有在线远程节点
func SearchPeers(searchParam model.SearchParam) ([]model.FileItem, int, int64) {
	peers := GetOnlinePeers()
	if len(peers) == 0 {
		return nil, 0, 0
	}

	// 信号量限制并发
	semaphore := make(chan struct{}, maxConcurrentPeers)
	var mu sync.Mutex
	var allMovies []model.FileItem
	var remoteTotalCnt int
	var remoteTotalSize int64
	var wg sync.WaitGroup

	for _, peer := range peers {
		if peer.Disabled {
			continue
		}
		wg.Add(1)
		go func(p *Peer) {
			defer wg.Done()
			defer utils.RecoverPanic()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result, err := p.searchPeer(searchParam)
			if err != nil {
				utils.ErrorFormat("远程搜索失败 [%s:%s]: %v", p.ID, p.IP, err)
				return
			}
			mu.Lock()
			allMovies = append(allMovies, result.Movies...)
			remoteTotalCnt += result.TotalCnt
			remoteTotalSize += result.TotalSize
			mu.Unlock()
		}(peer)
	}
	wg.Wait()

	return allMovies, remoteTotalCnt, remoteTotalSize
}

// searchPeer 向单个远程节点发送搜索请求
func (p *Peer) searchPeer(searchParam model.SearchParam) (*PeerSearchResult, error) {
	reqBody, err := json.Marshal(searchParam)
	if err != nil {
		return nil, fmt.Errorf("序列化请求参数失败: %w", err)
	}

	url := fmt.Sprintf("http://%s:%s/api/movieList", p.IP, p.Port)
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Search-Gin-Remote", "true")

	resp, err := remoteClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("远程节点返回错误: %d", resp.StatusCode)
	}

	var result struct {
		Data       json.RawMessage `json:"Data"`
		TotalCnt   int             `json:"TotalCnt"`
		ResultSize string          `json:"ResultSize"`
		TotalSize  string          `json:"TotalSize"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	var movies []model.FileItem
	if err := json.Unmarshal(result.Data, &movies); err != nil {
		return nil, fmt.Errorf("解析响应数据失败: %w", err)
	}

	return &PeerSearchResult{Movies: movies, TotalCnt: result.TotalCnt, TotalSize: ParseTotalSize(result.TotalSize)}, nil
}

// SearchRemotePeer 搜索指定远程节点，返回完整 Page 结果
func SearchRemotePeer(peer *Peer, searchParam model.SearchParam) (utils.Page, error) {
	reqBody, err := json.Marshal(searchParam)
	if err != nil {
		return utils.Page{}, fmt.Errorf("序列化请求参数失败: %w", err)
	}

	url := fmt.Sprintf("http://%s:%s/api/movieList", peer.IP, peer.Port)
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return utils.Page{}, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Search-Gin-Remote", "true")

	resp, err := peerClient.Do(req)
	if err != nil {
		return utils.Page{}, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return utils.Page{}, fmt.Errorf("远程节点返回错误: %d", resp.StatusCode)
	}

	var rawResult struct {
		Data       json.RawMessage `json:"Data"`
		TotalCnt   int             `json:"TotalCnt"`
		ResultSize string          `json:"ResultSize"`
		TotalSize  string          `json:"TotalSize"`
		ResultCnt  int             `json:"ResultCnt"`
		CurCnt     int             `json:"CurCnt"`
		CurSize    string          `json:"CurSize"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rawResult); err != nil {
		return utils.Page{}, fmt.Errorf("解析响应失败: %w", err)
	}

	var fileItems []model.FileItem
	if err := json.Unmarshal(rawResult.Data, &fileItems); err != nil {
		return utils.Page{}, fmt.Errorf("解析响应数据失败: %w", err)
	}

	// 填充流媒体 URL
	result := utils.Page{
		Data:       fileItems,
		TotalCnt:   rawResult.TotalCnt,
		ResultSize: rawResult.ResultSize,
		TotalSize:  rawResult.TotalSize,
		ResultCnt:  rawResult.ResultCnt,
		CurCnt:     rawResult.CurCnt,
		CurSize:    rawResult.CurSize,
	}

	return result, nil
}

// ParseTotalSize 将 "23.53 G" 格式的字符串解析为 int64 字节数
// 使用整数运算避免浮点精度丢失和溢出
func ParseTotalSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return 0
	}
	val, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(parts[1]) {
	case "B":
		return int64(val)
	case "K":
		return int64(val * 1024)
	case "M":
		return int64(val * 1024 * 1024)
	case "G":
		return int64(val * 1024 * 1024 * 1024)
	case "T":
		tb := int64(val)
		result := tb * 1024 * 1024 * 1024 * 1024
		if result/1024/1024/1024/1024 != tb {
			return 0
		}
		return result
	default:
		return 0
	}
}

// MergeResults 合并本地与远程结果，保留所有文件（不主动去重）
func MergeResults(local, remote []model.FileItem) []model.FileItem {
	merged := make([]model.FileItem, 0, len(local)+len(remote))
	merged = append(merged, local...)
	merged = append(merged, remote...)
	return merged
}

// dedupKey 生成去重 key：Code+Size（优先）或 Name+Size（兜底）
func dedupKey(m model.FileItem) string {
	if m.Code != "" {
		return fmt.Sprintf("code:%s:%d", m.Code, m.Size)
	}
	return fmt.Sprintf("name:%s:%d", m.Name, m.Size)
}

// FillURLs 为搜索结果填充流媒体 URL
// 本机文件使用请求进来的网卡 IP；远程文件指向源节点。
// 在 URL 中附加当前认证 token（而非 HMAC 签名），:10082 侧通过 StreamTokenAuth 校验。
func FillURLs(c *gin.Context, movies []model.FileItem) {
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP, _, _ = net.SplitHostPort(c.Request.RemoteAddr)
	}

	localIP := pickLocalIP(clientIP)
	localNode := LocalNodeHost
	filePort := strings.TrimPrefix(FilePortNo, ":")

	// 从当前请求提取 token（优先 Authorization header，兜底 query）
	token := ""
	if auth := c.GetHeader("Authorization"); auth != "" && strings.HasPrefix(auth, "Bearer ") {
		token = strings.TrimPrefix(auth, "Bearer ")
	}
	if token == "" {
		token = c.Query("token")
	}

	// 预构建本机 base URL，避免每文件重复拼接
	localBase := "http://" + localIP + ":" + filePort
	streamPath := "/api/stream/GetFileByPathUseEncode/"
	pngPath := "/api/stream/png/"
	jpgPath := "/api/stream/jpg/"

	tokenParam := "?token=" + url.QueryEscape(token)

	for i := range movies {
		m := &movies[i]
		if m.NodeHost == localNode || m.NodeHost == "" {
			m.StreamUrl = localBase + streamPath + url.QueryEscape(m.Path) + tokenParam
			m.PngUrl = localBase + pngPath + m.Id + tokenParam
			m.JpgUrl = localBase + jpgPath + m.Id + tokenParam
			m.NodeHost = localNode
			m.NodeName = LocalNodeName
		} else {
			peerFilePort := filePort
			if p := GetPeer(m.NodeHost); p != nil && p.FilePort != "" {
				peerFilePort = p.FilePort
			}
			if peerIP := ResolvePeerIP(m.NodeHost); peerIP != "" {
				peerBase := "http://" + peerIP + ":" + peerFilePort
				m.StreamUrl = peerBase + streamPath + url.QueryEscape(m.Path) + tokenParam
				m.PngUrl = peerBase + pngPath + m.Id + tokenParam
				m.JpgUrl = peerBase + jpgPath + m.Id + tokenParam
			}
		}
	}
}

// pickLocalIP 从客户端 IP 找到本机同网段的出口 IP
//
// 设计说明：多网卡场景下，按请求方网段选择本机 IP（ipNet.Contains 匹配），
// 保证返回的 IP 是客户端可达的。这是正确行为，不是 Bug ——
// 例如 eth0(192.168.1.10) 的客户端拿到 192.168.1.x，eth1(10.0.0.10) 的客户端拿到 10.0.0.x。
// LocalNodeHost 是逻辑标识符 "hostname:port"，不参与 URL 构造。
func pickLocalIP(clientIP string) string {
	parsedIP := net.ParseIP(clientIP)
	if parsedIP == nil || parsedIP.IsLoopback() {
		return fallbackLocalIP()
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return fallbackLocalIP()
	}

	// 优先匹配 IPv4 同网段
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if ipNet.IP.To4() != nil && ipNet.Contains(parsedIP) {
				return ipNet.IP.String()
			}
		}
	}
	// 兜底：匹配 IPv6 同网段
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if ipNet.Contains(parsedIP) {
				return ipNet.IP.String()
			}
		}
	}
	return fallbackLocalIP()
}

func fallbackLocalIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

// PaginateMovies 对合并后的结果进行分页
func PaginateMovies(movies []model.FileItem, pageNo, pageSize int) ([]model.FileItem, int) {
	total := len(movies)
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	start := (pageNo - 1) * pageSize
	if start >= total {
		return nil, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return movies[start:end], total
}
