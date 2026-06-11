package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"search-gin/internal/model"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	maxConcurrentPeers = 5
	remoteSearchTimeout = 2 * time.Second
	maxPageSize        = 99999 // 远程搜索获取全部结果
)

// SearchPeers 并发搜索所有在线远程节点
func SearchPeers(searchParam model.SearchParam) []model.Movie {
	peers := GetOnlinePeers()
	if len(peers) == 0 {
		return nil
	}

	// 信号量限制并发
	semaphore := make(chan struct{}, maxConcurrentPeers)
	var mu sync.Mutex
	var allMovies []model.Movie
	var wg sync.WaitGroup

	for _, peer := range peers {
		wg.Add(1)
		go func(p *Peer) {
			defer wg.Done()
			defer utils.RecoverPanic()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			movies, err := searchPeer(p, searchParam)
			if err != nil {
				utils.ErrorFormat("远程搜索失败 [%s:%s]: %v", p.ID, p.IP, err)
				return
			}
			mu.Lock()
			allMovies = append(allMovies, movies...)
			mu.Unlock()
		}(peer)
	}
	wg.Wait()

	return allMovies
}

// searchPeer 向单个远程节点发送搜索请求
func searchPeer(peer *Peer, searchParam model.SearchParam) ([]model.Movie, error) {
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
		// Data 可能是 []model.Movie，尝试类型断言
		moviesTyped, ok2 := result.Data.([]model.Movie)
		if !ok2 {
			return nil, fmt.Errorf("远程节点返回的数据类型非预期: %T", result.Data)
		}
		return moviesTyped, nil
	}

	// 从 []interface{} 转为 []model.Movie
	var out []model.Movie
	raw, _ := json.Marshal(movies)
	json.Unmarshal(raw, &out)
	return out, nil
}

// MergeResults 合并本地与远程结果，按 Code+Size 或 Name+Size 去重，本机优先
func MergeResults(local, remote []model.Movie) []model.Movie {
	seen := make(map[string]bool)
	merged := make([]model.Movie, 0, len(local)+len(remote))

	// 本机优先
	for _, m := range local {
		key := dedupKey(m)
		seen[key] = true
		merged = append(merged, m)
	}

	// 远程：不重复才加入
	for _, m := range remote {
		key := dedupKey(m)
		if !seen[key] {
			merged = append(merged, m)
			seen[key] = true
		}
	}
	return merged
}

// dedupKey 生成去重 key：Code+Size（优先）或 Name+Size（兜底）
func dedupKey(m model.Movie) string {
	if m.Code != "" {
		return fmt.Sprintf("code:%s:%d", m.Code, m.Size)
	}
	return fmt.Sprintf("name:%s:%d", m.Name, m.Size)
}

// FillURLs 为搜索结果填充流媒体 URL
// 本机文件使用请求进来的网卡 IP；远程文件指向源节点
func FillURLs(c *gin.Context, movies []model.Movie) {
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
			m.StreamUrl = fmt.Sprintf("http://%s:%s/api/stream/file/%s", localIP, filePort, m.Id)
			m.PngUrl = fmt.Sprintf("http://%s:%s/api/stream/png/%s", localIP, filePort, m.Id)
			m.JpgUrl = fmt.Sprintf("http://%s:%s/api/stream/jpg/%s", localIP, filePort, m.Id)
			m.NodeHost = localNode
			m.NodeName = LocalNodeName
		} else {
			// 远程文件 → 指向源节点的文件流端口
			if peerIP := ResolvePeerIP(m.NodeHost); peerIP != "" {
				m.StreamUrl = fmt.Sprintf("http://%s:%s/api/stream/file/%s", peerIP, filePort, m.Id)
				m.PngUrl = fmt.Sprintf("http://%s:%s/api/stream/png/%s", peerIP, filePort, m.Id)
				m.JpgUrl = fmt.Sprintf("http://%s:%s/api/stream/jpg/%s", peerIP, filePort, m.Id)
			}
		}
	}
}

// pickLocalIP 从客户端 IP 找到本机同网段的出口 IP
func pickLocalIP(clientIP string) string {
	parsedIP := net.ParseIP(clientIP)
	if parsedIP == nil {
		return fallbackLocalIP()
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return fallbackLocalIP()
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
func PaginateMovies(movies []model.Movie, pageNo, pageSize int) ([]model.Movie, int) {
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
