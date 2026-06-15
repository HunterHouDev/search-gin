package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"search-gin/internal/model"
	"search-gin/pkg/consts"
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
	maxPageSize         = 99999 // 远程搜索获取全部结果
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

			result, err := searchPeer(p, searchParam)
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
func searchPeer(peer *Peer, searchParam model.SearchParam) (*PeerSearchResult, error) {
	// 远程也用大 pageSize 获取全部结果，由请求端做最终分页
	remoteParam := searchParam
	remoteParam.Page = 1
	remoteParam.PageSize = maxPageSize

	reqBody, err := json.Marshal(remoteParam)
	if err != nil {
		return nil, fmt.Errorf("序列化请求参数失败: %w", err)
	}

	url := fmt.Sprintf("http://%s:%s/api/movieList", peer.IP, peer.Port)
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Search-Gin-Remote", "true")

	client := &http.Client{Timeout: remoteSearchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("远程节点返回错误: %d", resp.StatusCode)
	}

	var result utils.Page
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	movies, ok := result.Data.([]interface{})
	if !ok {
		// Data 可能是 []model.FileItem，尝试类型断言
		moviesTyped, ok2 := result.Data.([]model.FileItem)
		if !ok2 {
			return nil, fmt.Errorf("远程节点返回的数据类型非预期: %T", result.Data)
		}
		return &PeerSearchResult{Movies: moviesTyped, TotalCnt: result.TotalCnt, TotalSize: ParseTotalSize(result.TotalSize)}, nil
	}

	// 从 []interface{} 转为 []model.FileItem
	var out []model.FileItem
	raw, _ := json.Marshal(movies)
	json.Unmarshal(raw, &out)
	return &PeerSearchResult{Movies: out, TotalCnt: result.TotalCnt, TotalSize: ParseTotalSize(result.TotalSize)}, nil
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

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return utils.Page{}, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return utils.Page{}, fmt.Errorf("远程节点返回错误: %d", resp.StatusCode)
	}

	var result utils.Page
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return utils.Page{}, fmt.Errorf("解析响应失败: %w", err)
	}

	// 填充流媒体 URL
	if movies, ok := result.Data.([]interface{}); ok {
		var fileItems []model.FileItem
		raw, _ := json.Marshal(movies)
		json.Unmarshal(raw, &fileItems)
		// URL 由前端用本地 IP 填充更准确，远程节点返回的数据已包含 URL
		result.Data = fileItems
	} else if movies, ok := result.Data.([]model.FileItem); ok {
		result.Data = movies
	}

	return result, nil
}

// ParseTotalSize 将 "23.53 G" 格式的字符串解析为 int64 字节数
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
		return int64(val * 1024 * 1024 * 1024 * 1024)
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
// 本机文件使用请求进来的网卡 IP；远程文件指向源节点
func FillURLs(c *gin.Context, movies []model.FileItem) {
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP, _, _ = net.SplitHostPort(c.Request.RemoteAddr)
	}

	localIP := pickLocalIP(clientIP)
	localNode := LocalNodeHost
	filePort := strings.TrimPrefix(consts.FilePortNo, ":")

	for i := range movies {
		m := &movies[i]
		if m.NodeHost == localNode || m.NodeHost == "" {
			// 本机文件 → 用请求进来的网卡 IP，指向文件流端口 :10082
			m.StreamUrl = fmt.Sprintf("http://%s:%s/api/stream/GetFileByPathUseEncode/%s", localIP, filePort, url.QueryEscape(m.Path))
			m.PngUrl = fmt.Sprintf("http://%s:%s/api/stream/png/%s", localIP, filePort, m.Id)
			m.JpgUrl = fmt.Sprintf("http://%s:%s/api/stream/jpg/%s", localIP, filePort, m.Id)
			m.NodeHost = localNode
			m.NodeName = LocalNodeName
		} else {
			// 远程文件 → 指向源节点的文件流端口（优先使用对端上报的 filePort）
			peerFilePort := filePort
			if p := GetPeer(m.NodeHost); p != nil && p.FilePort != "" {
				peerFilePort = p.FilePort
			}
			if peerIP := ResolvePeerIP(m.NodeHost); peerIP != "" {
				m.StreamUrl = fmt.Sprintf("http://%s:%s/api/stream/GetFileByPathUseEncode/%s", peerIP, peerFilePort, url.QueryEscape(m.Path))
				m.PngUrl = fmt.Sprintf("http://%s:%s/api/stream/png/%s", peerIP, peerFilePort, m.Id)
				m.JpgUrl = fmt.Sprintf("http://%s:%s/api/stream/jpg/%s", peerIP, peerFilePort, m.Id)
			}
		}
	}
}

// pickLocalIP 从客户端 IP 找到本机同网段的出口 IP
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
