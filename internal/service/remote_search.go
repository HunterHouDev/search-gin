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
	// 远程也用大 pageSize 获取全部结果，由请求端做最终分页
	remoteParam := searchParam
	remoteParam.Page = 1
	remoteParam.PageSize = maxPageSize

	reqBody, err := json.Marshal(remoteParam)
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

const signedURLTTL = 4 * time.Hour

// FillURLs 为搜索结果填充带签名的流媒体 URL
// 本机文件使用请求进来的网卡 IP；远程文件指向源节点
func FillURLs(c *gin.Context, movies []model.FileItem) {
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP, _, _ = net.SplitHostPort(c.Request.RemoteAddr)
	}

	localIP := pickLocalIP(clientIP)
	localNode := LocalNodeHost
	filePort := strings.TrimPrefix(FilePortNo, ":")

	// 预构建本机 base URL，避免每文件重复拼接
	localBase := "http://" + localIP + ":" + filePort
	streamPath := "/api/stream/GetFileByPathUseEncode/"
	pngPath := "/api/stream/png/"
	jpgPath := "/api/stream/jpg/"

	for i := range movies {
		m := &movies[i]
		if m.NodeHost == localNode || m.NodeHost == "" {
			m.StreamUrl = utils.SignURL(localBase, streamPath+url.QueryEscape(m.Path), signedURLTTL)
			m.PngUrl = utils.SignURL(localBase, pngPath+m.Id, signedURLTTL)
			m.JpgUrl = utils.SignURL(localBase, jpgPath+m.Id, signedURLTTL)
			m.NodeHost = localNode
			m.NodeName = LocalNodeName
		} else {
			peerFilePort := filePort
			if p := GetPeer(m.NodeHost); p != nil && p.FilePort != "" {
				peerFilePort = p.FilePort
			}
			if peerIP := ResolvePeerIP(m.NodeHost); peerIP != "" {
				peerBase := "http://" + peerIP + ":" + peerFilePort
				m.StreamUrl = utils.SignURL(peerBase, streamPath+url.QueryEscape(m.Path), signedURLTTL)
				m.PngUrl = utils.SignURL(peerBase, pngPath+m.Id, signedURLTTL)
				m.JpgUrl = utils.SignURL(peerBase, jpgPath+m.Id, signedURLTTL)
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
