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
// 每个 peer 返回的结果已由远端 Page() 按 PageSize 分页，不会全量返回
// 注意：远端分页确保单节点返回量 = PageSize（默认 ~60 条），不存在 OOM 风险
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

// FillURLs 为搜索结果填充流媒体 URL
// 本机文件使用请求进来的网卡 IP；远程搜索结果由远端节点自行处理 URL。
// 在 URL 中附加当前认证 token（而非 HMAC 签名），:10082 侧通过 StreamTokenAuth 校验。
func FillURLs(c *gin.Context, movies []model.FileItem) {
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP, _, _ = net.SplitHostPort(c.Request.RemoteAddr)
	}

	localIP := pickLocalIP(clientIP)
	localNode := LocalNodeHost
	filePort := strings.TrimPrefix(FilePortNo, ":")
	apiPort := strings.TrimPrefix(PortNo, ":")

	// 预构建本机两个端口的 base URL
	// 对搜索结果交替使用 API 端口（:10081）和文件流端口（:10082），
	// 突破浏览器 HTTP/1.1 每域名 6 连接限制，提升缩略图并行加载性能。
	localBases := [2]string{
		"http://" + localIP + ":" + filePort,
		"http://" + localIP + ":" + apiPort,
	}
	streamPath := "/api/stream/GetFileByPathUseEncode/"
	pngPath := "/api/stream/png/"
	jpgPath := "/api/stream/jpg/"

	// 生成加密的 streamToken（内含过期时间），:10081/:10082 解密后只校验有效期
	// 图片预览 5分钟（防懒加载裂图），视频流 4小时
	imgExpire := time.Now().Add(5 * time.Minute).Unix()
	streamExpire := time.Now().Add(4 * time.Hour).Unix()
	imgToken, err := utils.EncryptStreamToken(imgExpire)
	if err != nil {
		utils.ErrorFormat("生成图片 streamToken 失败: %v", err)
		imgToken = ""
	}
	streamToken, err := utils.EncryptStreamToken(streamExpire)
	if err != nil {
		utils.ErrorFormat("生成视频 streamToken 失败: %v", err)
		streamToken = ""
	}
	imgTokenParam := "?streamToken=" + url.QueryEscape(imgToken)
	streamTokenParam := "?streamToken=" + url.QueryEscape(streamToken)

	for i := range movies {
		m := &movies[i]
		if m.StreamUrl != "" {
			continue
		}
		localBase := localBases[i%2] // 交替使用两个端口，突破浏览器 HTTP/1.1 每域名 6 连接限制
		m.StreamUrl = localBase + streamPath + url.QueryEscape(m.Path) + streamTokenParam
		m.PngUrl = localBase + pngPath + m.Id + imgTokenParam
		m.JpgUrl = localBase + jpgPath + m.Id + imgTokenParam
		m.NodeHost = localNode
		m.NodeName = LocalNodeName
	}
}

// localNetsOnce 一次性枚举本机网卡，进程生命周期内不变
var (
	localNetsOnce sync.Once
	localIPv4Nets []net.IPNet
	localIPv6Nets []net.IPNet
	localFirstIP  string
)

// pickLocalIPCache clientIP → 本机出口 IP，同网段客户端只算一次
var pickLocalIPCache sync.Map

// pickLocalIP 从客户端 IP 找到本机同网段的出口 IP
//
// 多网卡场景下，按请求方网段选择本机 IP（ipNet.Contains 匹配），
// 保证返回的 IP 是客户端可达的。
func pickLocalIP(clientIP string) string {
	if v, ok := pickLocalIPCache.Load(clientIP); ok {
		return v.(string)
	}

	localNetsOnce.Do(func() {
		interfaces, err := net.Interfaces()
		if err != nil {
			return
		}
		for i := range interfaces {
			addrs, err := interfaces[i].Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if !ok {
					continue
				}
				if ipNet.IP.To4() != nil {
					localIPv4Nets = append(localIPv4Nets, *ipNet)
				} else {
					localIPv6Nets = append(localIPv6Nets, *ipNet)
				}
			}
		}
		for i := range localIPv4Nets {
			if !localIPv4Nets[i].IP.IsLoopback() {
				localFirstIP = localIPv4Nets[i].IP.String()
				break
			}
		}
	})

	result := localFirstIP
	if parsedIP := net.ParseIP(clientIP); parsedIP != nil && !parsedIP.IsLoopback() {
		for i := range localIPv4Nets {
			if localIPv4Nets[i].Contains(parsedIP) {
				result = localIPv4Nets[i].IP.String()
				break
			}
		}
		if result == localFirstIP {
			for i := range localIPv6Nets {
				if localIPv6Nets[i].Contains(parsedIP) {
					result = localIPv6Nets[i].IP.String()
					break
				}
			}
		}
	}
	if result == "" {
		result = "127.0.0.1"
	}
	pickLocalIPCache.Store(clientIP, result)
	return result
}

// PaginateMovies 对合并后的结果进行分页
func PaginateMovies(movies []model.FileItem, pageNo, pageSize int) ([]model.FileItem, int) {
	return utils.SlicePage(movies, pageNo, pageSize)
}
