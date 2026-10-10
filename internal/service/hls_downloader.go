package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"search-gin/internal/model"
	"search-gin/internal/sse"
	"search-gin/pkg/utils"
)

// ──────────────────────────────────────────────────────────────
// HLS 分片下载：由服务端拉取 m3u8 分片并合并成本地文件
//
// 任务落在服务端（TransferTask + 调度器 + 任务日志），
// 因此关闭弹窗、刷新页面、甚至关闭浏览器都不会中断下载。
// ──────────────────────────────────────────────────────────────

const (
	// hlsDownloadConcurrency 同时在途的分片请求数（默认值）。
	// 分片下载是纯网络 I/O，并发数可以高于任务槽位数（默认 4）。
	// 注意：该值同时决定滑动窗口与连接池上限，任务可通过参数覆盖
	// （见 HlsDownloadParam.Concurrency）。
	// 若源站持续返回 connection reset（按连接数限流），可下调此值（8 → 4）
	// 换取稳定性，代价是下载变慢。
	hlsDownloadConcurrency = 4
	// hlsMinConcurrency / hlsMaxConcurrency 任务级并发的允许区间：
	// 小于 1 视为未指定（取默认值），大于上限则截断，避免打挂源站或撑爆内存
	hlsMinConcurrency = 1
	hlsMaxConcurrency = 16
	// hlsSegmentRetry 单个分片的最大尝试次数（含首次）
	hlsSegmentRetry = 3
	// hlsThrottleRetry 遇到 429/5xx（限流/暂时不可用）时的最大尝试次数（含首次）：
	// 等待按 1s → 2s → 4s 指数退避，比普通错误多一次机会
	hlsThrottleRetry = 4
	// hlsThrottleStep 限流退避基数：第 1 次重试等 1s，之后逐次翻倍
	hlsThrottleStep = time.Second
	// hlsNetworkRetry 连接被重置 / 读取中断等网络类错误的最大尝试次数（含首次）。
	// 这类错误多半是复用连接被源站掐断，换新连接重试即可，但也不能像普通错误
	// 那样急，故比 hlsSegmentRetry 多两次机会（退避 1s → 2s → 4s → 8s → 8s）
	hlsNetworkRetry = 6
	// hlsNetworkStep 网络类错误退避基数：1s → 2s → 4s → 8s
	hlsNetworkStep = time.Second
	// hlsNetworkMaxBackoff 网络类错误退避封顶。
	//
	// 实测：部分源站对持续下载的连接发 RST（wsarecv: forcibly closed），限流窗口
	// 在 30s 量级，退避不够长时「换连接重试」仍落在窗口内，重试形同虚设
	// （旧值 500ms 起步，5 次总退避仅 7.5s，全部打在同一窗口里）。
	// 加长到 1s 起步并封顶 8s，6 次总退避约 31s，才能跨过窗口。
	hlsNetworkMaxBackoff = 8 * time.Second
	// hlsGateCooldown 并发减半后的冷却期：期间不再触发限流才逐级恢复并发
	hlsGateCooldown = 30 * time.Second
	// hlsProgressInterval 进度回写最小间隔，避免分片很小时高频加锁 + SSE 广播
	hlsProgressInterval = 300 * time.Millisecond
	// hlsBrowserUA 使用浏览器 UA：部分 CDN 会对陌生 UA 限速或降级
	hlsBrowserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

// hlsHTTPClient 分片下载专用客户端。
//
// 注意：此处必须显式提供 Transport。裸的 &http.Client{Timeout:...} 会复用
// http.DefaultTransport，其 MaxIdleConnsPerHost 仅为 2——并发下载分片时
// 大部分请求只能新建连接（重复 TCP + TLS 握手），这是服务端下载明显慢于
// 浏览器下载的主要原因之一。
var hlsHTTPClient = &http.Client{
	Timeout: 120 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          hlsDownloadConcurrency * 4,
		MaxIdleConnsPerHost:   hlsDownloadConcurrency * 2,
		// 空闲连接保持时间刻意短：源站 / 中间设备常单方面关闭空闲连接，
		// 客户端复用这种「已死」连接时就会报连接被重置（wsarecv / ECONNRESET）
		IdleConnTimeout: 20 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	},
}

// hlsRetryHTTPClient 重试专用客户端：每次请求新建一条连接，且不走 HTTP/2。
//
// 必要性：连接被源站/中间设备单方面掐断时（Windows 表现为
// "wsarecv: An existing connection was forcibly closed"），连接池里可能残留
// 已经失效的连接，重试若仍从池中取连接就会连续打在同一条坏连接上，重试形同虚设。
// 这个客户端 DisableKeepAlives，保证每一次重试都是全新的 TCP + TLS。
//
// 同时显式关闭 HTTP/2：部分 CDN 的 h2 实现在长连接多流传输时会单方面 RST 流，
// 退化为 HTTP/1.1 反而稳定。TLSNextProto 置空才是彻底禁用（仅 ForceAttemptHTTP2=false
// 时 Transport 仍可能通过 ALPN 协商升级）。
var hlsRetryHTTPClient = &http.Client{
	Timeout: 120 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// 注意：DisableKeepAlives 下连接不复用，再设 MaxIdleConns 没有意义
		DisableKeepAlives:     true,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     false,
		TLSNextProto:          map[string]func(string, *tls.Conn) http.RoundTripper{},
	},
}

// ── 运行时状态（播放列表文本 + 取消句柄） ──────────────────────

type hlsRuntime struct {
	playlist string
	cancel   context.CancelFunc
	// concurrency 该任务的下载并发数（分片同时在途数）
	concurrency int
}

var (
	hlsRuntimeMap   = map[string]*hlsRuntime{}
	hlsRuntimeMutex sync.Mutex
)

func putHlsRuntime(id string, playlist string) {
	hlsRuntimeMutex.Lock()
	// 重启任务时沿用该任务原本的并发设置（runtime 可能已被丢弃，此时回到默认值）
	concurrency := hlsDownloadConcurrency
	if rt, ok := hlsRuntimeMap[id]; ok && rt.concurrency > 0 {
		concurrency = rt.concurrency
	}
	hlsRuntimeMap[id] = &hlsRuntime{playlist: playlist, concurrency: concurrency}
	hlsRuntimeMutex.Unlock()
}

// putHlsRuntimeConcurrency 创建任务时指定该任务的下载并发数
func putHlsRuntimeConcurrency(id string, playlist string, concurrency int) {
	hlsRuntimeMutex.Lock()
	hlsRuntimeMap[id] = &hlsRuntime{playlist: playlist, concurrency: concurrency}
	hlsRuntimeMutex.Unlock()
}

// hlsConcurrencyOf 取任务的下载并发数；未记录或非法时回落到默认值
func hlsConcurrencyOf(id string) int {
	hlsRuntimeMutex.Lock()
	defer hlsRuntimeMutex.Unlock()
	if rt, ok := hlsRuntimeMap[id]; ok && rt.concurrency >= hlsMinConcurrency {
		if rt.concurrency > hlsMaxConcurrency {
			return hlsMaxConcurrency
		}
		return rt.concurrency
	}
	return hlsDownloadConcurrency
}

// hlsWindowOf 已派发但尚未落盘的分片上限（滑动窗口），随并发数放大。
// 顺序写盘时若窗口无限，最前面的慢分片会让后续分片全部堆积在内存里。
func hlsWindowOf(concurrency int) int {
	if concurrency < hlsMinConcurrency {
		concurrency = hlsDownloadConcurrency
	}
	return concurrency * 3
}

// normalizeConcurrency 归一化任务级并发数：未指定（<1）取默认值，超出上限截断
func normalizeConcurrency(n int) int {
	if n < hlsMinConcurrency {
		return hlsDownloadConcurrency
	}
	if n > hlsMaxConcurrency {
		return hlsMaxConcurrency
	}
	return n
}

func setHlsCancel(id string, cancel context.CancelFunc) {
	hlsRuntimeMutex.Lock()
	if rt, ok := hlsRuntimeMap[id]; ok {
		rt.cancel = cancel
	}
	hlsRuntimeMutex.Unlock()
}

func hlsPlaylistOf(id string) string {
	hlsRuntimeMutex.Lock()
	defer hlsRuntimeMutex.Unlock()
	if rt, ok := hlsRuntimeMap[id]; ok {
		return rt.playlist
	}
	return ""
}

// dropHlsRuntime 释放任务运行时状态；若仍在下载则取消。
// 由 DeleteTaskLog 统一在任务被删除时调用。
func dropHlsRuntime(id string) {
	hlsRuntimeMutex.Lock()
	rt, ok := hlsRuntimeMap[id]
	if ok {
		delete(hlsRuntimeMap, id)
	}
	hlsRuntimeMutex.Unlock()
	if ok && rt.cancel != nil {
		rt.cancel()
	}
}

// forgetHlsRuntime 任务结束后丢弃播放列表，避免常驻内存（不触发取消）
func forgetHlsRuntime(id string) {
	hlsRuntimeMutex.Lock()
	delete(hlsRuntimeMap, id)
	hlsRuntimeMutex.Unlock()
}

// ── m3u8 解析 ────────────────────────────────────────────────

type hlsByteRange struct {
	length int64
	offset int64
}

type hlsKey struct {
	method string // AES-128 / SAMPLE-AES
	url    string
	iv     []byte
}

type hlsSegmentItem struct {
	index     int
	url       string
	byteRange *hlsByteRange
	key       *hlsKey
	// initMap 该分片所属的初始化段（#EXT-X-MAP，fMP4 流）。
	// 合并多个播放列表时中途可能切换初始化段，因此记录在每个分片上，
	// 而不是整个播放列表只保留一个。
	initMap   string
	initRange *hlsByteRange
}

type hlsPlaylist struct {
	segments     []hlsSegmentItem
	initMap      string
	initRange    *hlsByteRange
	mediaSeq     int
	totalSeconds float64
}

var hlsAttrRe = regexp.MustCompile(`([A-Za-z0-9-]+)=("[^"]*"|[^,]*)`)

func parseHlsAttributes(input string) map[string]string {
	attrs := map[string]string{}
	for _, m := range hlsAttrRe.FindAllStringSubmatch(input, -1) {
		attrs[strings.ToUpper(m[1])] = strings.Trim(m[2], `"`)
	}
	return attrs
}

func hlsValueAfterColon(line string) string {
	if i := strings.Index(line, ":"); i >= 0 {
		return strings.TrimSpace(line[i+1:])
	}
	return ""
}

func parseHlsByteRange(value string, prevEnd int64) *hlsByteRange {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.SplitN(value, "@", 2)
	length, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || length <= 0 {
		return nil
	}
	offset := prevEnd
	if len(parts) == 2 {
		o, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil || o < 0 {
			return nil
		}
		offset = o
	}
	return &hlsByteRange{length: length, offset: offset}
}

func hexToBytes(s string) []byte {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	if s == "" || len(s)%2 != 0 {
		return nil
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	return b
}

func parseHlsKey(line string) *hlsKey {
	attrs := parseHlsAttributes(hlsValueAfterColon(line))
	method := strings.ToUpper(attrs["METHOD"])
	if method == "" || method == "NONE" {
		return nil
	}
	return &hlsKey{method: method, url: attrs["URI"], iv: hexToBytes(attrs["IV"])}
}

func resolveHlsURL(base, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || base == "" {
		return ref
	}
	u, err := url.Parse(ref)
	if err != nil || u.IsAbs() {
		return ref
	}
	bu, err := url.Parse(base)
	if err != nil {
		return ref
	}
	return bu.ResolveReference(u).String()
}

// parseHlsPlaylistText 解析 m3u8 文本；base 用于补全相对地址。
// 支持一段文本里出现多个 #EXT-X-MAP（多源合并后的播放列表）。
func parseHlsPlaylistText(text, base string) (*hlsPlaylist, error) {
	pl := &hlsPlaylist{}
	var currentKey *hlsKey
	var currentInitMap string
	var currentInitRange *hlsByteRange
	var pendingRangeRaw string
	var lastRangeEnd int64
	var lastRangeURL string

	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			switch {
			case strings.HasPrefix(line, "#EXTINF"):
				val := strings.SplitN(hlsValueAfterColon(line), ",", 2)[0]
				if d, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
					pl.totalSeconds += d
				}
			case strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE"):
				if n, err := strconv.Atoi(hlsValueAfterColon(line)); err == nil {
					pl.mediaSeq = n
				}
			case strings.HasPrefix(line, "#EXT-X-KEY"):
				currentKey = parseHlsKey(line)
				if currentKey != nil {
					currentKey.url = resolveHlsURL(base, currentKey.url)
				}
			case strings.HasPrefix(line, "#EXT-X-BYTERANGE"):
				pendingRangeRaw = hlsValueAfterColon(line)
			case strings.HasPrefix(line, "#EXT-X-MAP"):
				attrs := parseHlsAttributes(hlsValueAfterColon(line))
				currentInitMap = resolveHlsURL(base, attrs["URI"])
				currentInitRange = nil
				if br, ok := attrs["BYTERANGE"]; ok {
					currentInitRange = parseHlsByteRange(br, 0)
				}
				// 记录首个初始化段：仅用于判定输出容器（mp4 / ts）
				if pl.initMap == "" {
					pl.initMap = currentInitMap
					pl.initRange = currentInitRange
				}
			}
			continue
		}

		// 分片地址行
		segURL := resolveHlsURL(base, line)
		prevEnd := int64(0)
		if lastRangeURL == segURL {
			prevEnd = lastRangeEnd
		}
		var br *hlsByteRange
		if pendingRangeRaw != "" {
			br = parseHlsByteRange(pendingRangeRaw, prevEnd)
		}
		if br != nil {
			lastRangeEnd = br.offset + br.length
			lastRangeURL = segURL
		}
		pl.segments = append(pl.segments, hlsSegmentItem{
			index:     len(pl.segments) + 1,
			url:       segURL,
			byteRange: br,
			key:       currentKey,
			initMap:   currentInitMap,
			initRange: currentInitRange,
		})
		pendingRangeRaw = ""
	}

	if len(pl.segments) == 0 {
		return nil, fmt.Errorf("播放列表中没有可下载的分片")
	}
	return pl, nil
}

// ── 网络 / 解密 ──────────────────────────────────────────────

// hlsOrigin 取地址的源站（scheme://host/），用作分片请求的 Referer
func hlsOrigin(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/"
}

// fetchHlsBytes 拉取一个 URL 的字节内容。
//
// freshConn=true 时走 hlsRetryHTTPClient（新建连接、仅 HTTP/1.1），
// 用于「上一次请求因连接被掐断而失败」的重试场景。
func fetchHlsBytes(ctx context.Context, rawURL string, br *hlsByteRange, referer string, freshConn bool) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", hlsBrowserUA)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	if br != nil {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", br.offset, br.offset+br.length-1))
	}
	client := hlsHTTPClient
	if freshConn {
		client = hlsRetryHTTPClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, hlsHTTPStatusError{code: resp.StatusCode}
	}
	return io.ReadAll(resp.Body)
}

// hlsHTTPStatusError 带 HTTP 状态码的错误，用于识别可退避重试的类别
type hlsHTTPStatusError struct{ code int }

func (e hlsHTTPStatusError) Error() string { return fmt.Sprintf("HTTP %d", e.code) }

// hlsThrottled 源站限流（429）或暂时不可用（5xx）：值得指数退避 + 降低并发
func hlsThrottled(err error) bool {
	var se hlsHTTPStatusError
	if errors.As(err, &se) {
		return se.code == http.StatusTooManyRequests || se.code >= 500
	}
	return false
}

// hlsNetworkError 传输层错误：连接被重置（RST）、读取中途断开、握手/读写超时等。
//
// 典型成因是 Keep-Alive 连接被源站或中间设备单方面关闭后又被复用
// （Windows 上表现为 wsarecv: An existing connection was forcibly closed），
// 也可能是对端主动断流。这类错误换新连接重试往往成功，退避比普通错误长。
//
// 是否代表限流要分情况：偶发一次抖动不是；但同一源站对后续请求持续 RST、
// 换新连接也无效时，就是隐式限流（不回 429 而是直接断连接），此时放慢才有效
// （见 fetchHlsSegment 里对 gate.throttle 的调用条件）。
func hlsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// hlsJitter 给退避时长叠加最多 25% 的随机抖动。
//
// 8 个 worker 同时失败时若按完全相同的间隔重试，会形成同步重试风暴——
// 整齐划一的突发重试正是源站/ WAF 判定异常流量的典型特征，反而招致更多 RST。
func hlsJitter(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}
	return d + time.Duration(rand.Int63n(int64(d)/4+1))
}

// ── 并发自愈闸门：429/5xx 时临时减半全局分片并发 ────────────────

// hlsGateCtxKey 把并发闸门挂到任务 ctx 上传递，避免层层改函数签名
type hlsGateCtxKey struct{}

// hlsGateFrom 取出 ctx 里的并发闸门；未挂载时返回 nil（退化为不限流）
func hlsGateFrom(ctx context.Context) *hlsGate {
	g, _ := ctx.Value(hlsGateCtxKey{}).(*hlsGate)
	return g
}

// hlsGate 分片请求的并发闸门。
//
// 源站限流（429/5xx）时把在途分片请求上限减半，给源站喘息窗口；
// 冷却期内不再触发限流则每次翻倍、逐级恢复到满并发。
type hlsGate struct {
	mu        sync.Mutex
	cond      *sync.Cond
	active    int       // 当前在途分片请求数
	limit     int       // 当前生效的并发上限
	base      int       // 满并发上限
	recoverAt time.Time // 并发恢复（翻倍）的最早时间
}

func newHlsGate(base int) *hlsGate {
	g := &hlsGate{limit: base, base: base}
	g.cond = sync.NewCond(&g.mu)
	return g
}

// acquire 领取一个并发槽位；ctx 取消时返回 false
func (g *hlsGate) acquire(ctx context.Context) bool {
	// ctx 取消时唤醒可能在 cond.Wait 里等待的协程（含本协程）
	stop := context.AfterFunc(ctx, g.cond.Broadcast)
	defer stop()
	g.mu.Lock()
	defer g.mu.Unlock()
	for {
		now := time.Now()
		if g.limit < g.base && !now.Before(g.recoverAt) {
			// 冷却期已过且未再触发限流：并发恢复一级
			g.limit *= 2
			if g.limit > g.base {
				g.limit = g.base
			}
			g.recoverAt = now.Add(hlsGateCooldown)
		}
		if g.active < g.limit {
			g.active++
			return true
		}
		if ctx.Err() != nil {
			return false
		}
		g.cond.Wait()
	}
}

// release 归还一个并发槽位
func (g *hlsGate) release() {
	g.mu.Lock()
	g.active--
	g.cond.Signal()
	g.mu.Unlock()
}

// throttle 源站返回 429/5xx 后调用：并发上限减半并重置恢复冷却计时
func (g *hlsGate) throttle() {
	g.mu.Lock()
	if g.limit > 1 {
		g.limit /= 2
	}
	g.recoverAt = time.Now().Add(hlsGateCooldown)
	g.cond.Broadcast()
	g.mu.Unlock()
}

// fetchHlsSegment 带重试的分片拉取：CDN 偶发 5xx / 连接重置时，
// 由该分片自己重试，不再让整个任务失败（旧实现单点失败即整任务终止）。
//
// 429/5xx 视为源站限流：等待按 1s → 2s → 4s 指数退避（比普通错误多一次机会），
// 同时触发并发闸门减半；连接被掐断等传输层错误按 1s → 2s → 4s → 8s（封顶）退避，
// 连续出现时也触发降并发；其他错误沿用 300ms/600ms 的短间隔重试。
func fetchHlsSegment(ctx context.Context, rawURL string, br *hlsByteRange, referer string) ([]byte, error) {
	gate := hlsGateFrom(ctx)
	maxAttempts := hlsSegmentRetry
	var lastErr error
	for attempt := 0; ; attempt++ {
		if attempt >= maxAttempts {
			break
		}
		if attempt > 0 {
			var wait time.Duration
			switch {
			case hlsThrottled(lastErr):
				// 限流类错误：指数退避，并升级为限流专用尝试次数（仅提升一次）
				if maxAttempts < hlsThrottleRetry {
					maxAttempts = hlsThrottleRetry
				}
				wait = hlsThrottleStep << (attempt - 1)
			case hlsNetworkError(lastErr):
				// 传输层错误（连接重置 / 读中断 / 超时）：中等指数退避，多给两次机会
				if maxAttempts < hlsNetworkRetry {
					maxAttempts = hlsNetworkRetry
				}
				wait = hlsNetworkStep << (attempt - 1)
				// 封顶：指数退避不设上限时最后一次等待会远超源站的限流窗口，
				// 白白拉长任务时长；到顶后按固定间隔重试即可
				if wait > hlsNetworkMaxBackoff {
					wait = hlsNetworkMaxBackoff
				}
			default:
				wait = time.Duration(attempt) * 300 * time.Millisecond
			}
			wait = hlsJitter(wait)
			select {
			case <-ctx.Done():
				return nil, errHlsCanceled
			case <-time.After(wait):
			}
		}
		// 上一跳是连接被掐断，或已重试到第 3 次：改用「新建连接 + 仅 HTTP/1.1」
		// 的客户端，避免重试仍旧打在同一条已被源站关闭的复用连接上
		freshConn := attempt >= 2 || (attempt > 0 && hlsNetworkError(lastErr))
		if gate != nil && !gate.acquire(ctx) {
			return nil, errHlsCanceled
		}
		data, err := fetchHlsBytes(ctx, rawURL, br, referer, freshConn)
		if gate != nil {
			gate.release()
		}
		if err == nil {
			return data, nil
		}
		if ctx.Err() != nil {
			return nil, errHlsCanceled
		}
		// 源站吃不消的两种信号都触发降并发：
		// 1) 显式限流 429/5xx；
		// 2) 隐式限流——不返回状态码而是直接掐断连接（RST）。实测同一源站在
		//    开跑约 40 秒后开始持续 RST，换新连接也无效，只有放慢才过得去。
		//    这类错误要从第 2 次失败（attempt>=1）起才算数，避免偶发抖动就砍半并发。
		if gate != nil && (hlsThrottled(err) || (attempt >= 1 && hlsNetworkError(err))) {
			gate.throttle()
		}
		lastErr = err
	}
	// 带上重试次数，便于从任务日志判断是偶发抖动还是源站持续拒绝
	return nil, fmt.Errorf("%w（已重试 %d 次）", lastErr, maxAttempts-1)
}

// hlsUriRe 匹配播放列表标签里的 URI="..."（#EXT-X-KEY / #EXT-X-MAP ...）
var hlsUriRe = regexp.MustCompile(`URI="([^"]*)"`)

// refreshPlaylistAuth 重新拉取一次 m3u8，按「去掉 query 的路径」把新的鉴权参数
// 替换到已保存的播放列表文本上。
//
// hlsSignedQueryRe 疑似时效签名的 query 参数名。各 CDN 叫法不一，宽松匹配：
// 宁可多刷一次，也不要漏掉真正会过期的签名。
var hlsSignedQueryRe = regexp.MustCompile(`(?i)auth_key|signature|x-amz-|x-oss-|token|sign=|expires|expire|play_session|hdntl`)

// hlsPlaylistHasSignedQuery 播放列表里是否存在带时效签名的地址。
// 没有签名就说明地址不会过期，不值得为了刷新多拉一次源站。
func hlsPlaylistHasSignedQuery(playlistText string) bool {
	for _, line := range strings.Split(playlistText, "\n") {
		// 只看 query 部分：路径里恰好出现同名片段不算
		for _, chunk := range strings.Split(line, "?")[1:] {
			query := chunk
			if i := strings.IndexAny(query, "\"#"); i >= 0 {
				query = query[:i]
			}
			if hlsSignedQueryRe.MatchString(query) {
				return true
			}
		}
	}
	return false
}

// 分片地址常带时效签名（auth_key / token 之类），任务创建时的文本搁置一段时间
// 再跑就过期了；源站对过期鉴权往往直接断开连接（wsarecv / connection reset）
// 而不是返回 403，表现得像网络故障。这里只替换地址里的 query，
// 分片列表本身（已删除的分片、多源顺序、字节区间）一律不动。
// 拉取失败（源站不可达 / 内容不是 m3u8）时原样返回，不阻塞任务。
func refreshPlaylistAuth(ctx context.Context, playlistText string, sourceURL string) string {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return playlistText
	}
	// 地址里没有时效签名就不会过期，不必为刷新多拉一次播放列表
	if !hlsPlaylistHasSignedQuery(playlistText) {
		return playlistText
	}
	// 刷新只是尽力而为：源站迟迟不响应时不该把任务卡在这里
	fetchCtx, cancelFetch := context.WithTimeout(ctx, 15*time.Second)
	defer cancelFetch()
	freshBytes, err := fetchHlsBytes(fetchCtx, sourceURL, nil, hlsOrigin(sourceURL), false)
	if err != nil || !strings.Contains(string(freshBytes), "#EXTM3U") {
		return playlistText
	}
	fresh, err := parseHlsPlaylistText(string(freshBytes), sourceURL)
	if err != nil {
		return playlistText
	}

	// path → 完整地址（含新的 query）
	byPath := make(map[string]string, len(fresh.segments)+8)
	pathOf := func(raw string) string {
		if i := strings.IndexAny(raw, "?#"); i >= 0 {
			return raw[:i]
		}
		return raw
	}
	add := func(raw string) {
		if raw != "" {
			byPath[pathOf(raw)] = raw
		}
	}
	for _, seg := range fresh.segments {
		add(seg.url)
		// 密钥与初始化段地址同样带鉴权参数
		if seg.key != nil {
			add(seg.key.url)
		}
		add(seg.initMap)
	}
	add(fresh.initMap)
	if len(byPath) == 0 {
		return playlistText
	}

	base, baseErr := url.Parse(sourceURL)
	lines := strings.Split(playlistText, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			// 标签行：只替换其中的 URI="..."（密钥 / 初始化段地址同样带鉴权）
			lines[i] = hlsUriRe.ReplaceAllStringFunc(line, func(m string) string {
				sub := hlsUriRe.FindStringSubmatch(m)
				if sub == nil {
					return m
				}
				next, ok := byPath[pathOf(sub[1])]
				if !ok {
					return m
				}
				return `URI="` + next + `"`
			})
			continue
		}
		// 分片行：整行替换为新地址（文本里的地址是绝对的，相对地址按源地址补全后再匹配）
		abs := trimmed
		if u, err := url.Parse(trimmed); err == nil && !u.IsAbs() && baseErr == nil {
			abs = base.ResolveReference(u).String()
		}
		if next, ok := byPath[pathOf(abs)]; ok {
			lines[i] = next
		}
	}
	return strings.Join(lines, "\n")
}

// hlsInitKey 初始化段标识（地址 + 字节区间），空串表示该分片不需要初始化段
func hlsInitKey(seg hlsSegmentItem) string {
	if seg.initMap == "" {
		return ""
	}
	if seg.initRange == nil {
		return seg.initMap
	}
	return fmt.Sprintf("%s#%d-%d", seg.initMap, seg.initRange.offset, seg.initRange.length)
}

// hlsInitBytes 拉取初始化段并缓存：同一初始化段被大量分片共用时只拉一次
func hlsInitBytes(ctx context.Context, seg hlsSegmentItem, referer string, cache *sync.Map) ([]byte, error) {
	key := hlsInitKey(seg)
	if v, ok := cache.Load(key); ok {
		return v.([]byte), nil
	}
	data, err := fetchHlsSegment(ctx, seg.initMap, seg.initRange, referer)
	if err != nil {
		return nil, fmt.Errorf("下载初始化分片失败: %w", err)
	}
	cache.Store(key, data)
	return data, nil
}

// hlsInitMapCount 统计不同的初始化段数量；大于 1 说明是多段 fMP4 合并
func hlsInitMapCount(pl *hlsPlaylist) int {
	seen := map[string]struct{}{}
	for _, seg := range pl.segments {
		if key := hlsInitKey(seg); key != "" {
			seen[key] = struct{}{}
		}
	}
	return len(seen)
}

// downloadHlsSegment 下载并按需解密一个分片
func downloadHlsSegment(ctx context.Context, pl *hlsPlaylist, seg hlsSegmentItem, referer string, keyCache *sync.Map) ([]byte, error) {
	data, err := fetchHlsSegment(ctx, seg.url, seg.byteRange, referer)
	if err != nil {
		// 单个分片耗尽重试即终止整个任务，这里留一条服务端日志便于事后定位；
		// 取消属于正常终止，不写错误日志
		if ctx.Err() == nil {
			utils.ErrorFormat("HLS 分片 %d 下载失败: %v", seg.index, err)
		}
		return nil, fmt.Errorf("分片 %d 下载失败: %w", seg.index, err)
	}
	if seg.key == nil || seg.key.method != "AES-128" || seg.key.url == "" {
		return data, nil
	}
	kb, err := hlsKeyBytes(ctx, seg.key.url, referer, keyCache)
	if err != nil {
		return nil, err
	}
	iv := seg.key.iv
	if iv == nil {
		iv = hlsSequenceIV(pl.mediaSeq + seg.index - 1)
	}
	plain, err := decryptHlsSegment(data, kb, iv)
	if err != nil {
		return nil, fmt.Errorf("分片 %d 解密失败: %w", seg.index, err)
	}
	return plain, nil
}

func hlsKeyBytes(ctx context.Context, keyURL string, referer string, cache *sync.Map) ([]byte, error) {
	if v, ok := cache.Load(keyURL); ok {
		return v.([]byte), nil
	}
	// 走 fetchHlsSegment 而非 fetchHlsBytes：密钥只拉一次却没有重试，
	// 一次连接重置就会让整个任务失败，代价太大
	data, err := fetchHlsSegment(ctx, keyURL, nil, referer)
	if err != nil {
		return nil, fmt.Errorf("获取解密密钥失败: %w", err)
	}
	if len(data) != 16 {
		return nil, fmt.Errorf("解密密钥长度非法(%d)", len(data))
	}
	cache.Store(keyURL, data)
	return data, nil
}

func pkcs7Unpad(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return b, nil
	}
	pad := int(b[len(b)-1])
	if pad == 0 || pad > aes.BlockSize || pad > len(b) {
		return nil, fmt.Errorf("解密数据 padding 非法")
	}
	for _, v := range b[len(b)-pad:] {
		if int(v) != pad {
			return nil, fmt.Errorf("解密数据 padding 非法")
		}
	}
	return b[:len(b)-pad], nil
}

func decryptHlsSegment(data, keyBytes, iv []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	if len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("密文长度不是 16 的倍数")
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}
	if len(iv) != aes.BlockSize {
		return nil, fmt.Errorf("解密 IV 长度非法")
	}
	out := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, data)
	return pkcs7Unpad(out)
}

// hlsSequenceIV 按分片序号推导 IV（未显式指定 IV 时的规范做法）
func hlsSequenceIV(sequence int) []byte {
	iv := make([]byte, 16)
	v := uint64(sequence)
	for i := 15; i >= 0 && v > 0; i-- {
		iv[i] = byte(v & 0xff)
		v >>= 8
	}
	return iv
}

// ── 任务生命周期 ─────────────────────────────────────────────

var errHlsCanceled = fmt.Errorf("任务已取消")

func appendHlsLog(id, line string) {
	if err := ensureTaskLogDir(); err != nil {
		return
	}
	f, err := os.OpenFile(TaskLogPath(id), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("15:04:05"), line)
}

// HlsPlaylistPath 持久化播放列表文件路径。
// 任务失败/取消后运行时内存会被丢弃，重启任务时从这里恢复播放列表。
func HlsPlaylistPath(taskKey string) string {
	return filepath.Join(taskLogDir(), taskKey+".m3u8")
}

// ReadHlsPlaylist 读取任务落盘的播放列表文本（前端浏览器直下备用通道使用）。
func ReadHlsPlaylist(taskID string) utils.Result {
	data, err := os.ReadFile(HlsPlaylistPath(taskID))
	if err != nil || len(data) == 0 {
		return utils.NewFailByMsg("播放列表不存在或已丢失")
	}
	res := utils.NewSuccess()
	res.Data = string(data)
	return res
}

// saveHlsPlaylist 把播放列表文本落盘，供重启失败任务时恢复
func saveHlsPlaylist(taskID, playlist string) {
	if err := ensureTaskLogDir(); err != nil {
		return
	}
	if err := os.WriteFile(HlsPlaylistPath(taskID), []byte(playlist), 0644); err != nil {
		utils.ErrorFormat("保存播放列表失败: %s, 错误: %v", taskID, err)
	}
}

// updateHlsProgress 更新任务进度（内存 + SSE 通知）
func updateHlsProgress(id string, done, total int, size int64) {
	progress := 0
	if total > 0 {
		progress = done * 100 / total
	}
	TransferTaskMutex.Lock()
	if t, ok := TransferTask[id]; ok {
		t.Segments = done
		t.TotalSegments = total
		t.Size = size
		t.Progress = progress
		TransferTask[id] = t
	}
	TransferTaskMutex.Unlock()

	sse.BroadcastEvent(model.SSETaskLog, map[string]interface{}{
		"taskKey":  id,
		"progress": progress,
		"done":     done,
		"total":    total,
	})
}

func finishHlsTask(id string, status string, message string) {
	TransferTaskMutex.Lock()
	t, ok := TransferTask[id]
	if !ok {
		// 任务已被删除（例如取消并清除），无需回写
		TransferTaskMutex.Unlock()
		forgetHlsRuntime(id)
		return
	}
	if status == model.StatusCompleted {
		t.Progress = 100
		t.Segments = t.TotalSegments
	}
	if message != "" {
		t.Log = message
	}
	TransferTask[id] = t
	TransferTaskMutex.Unlock()

	updateTaskStatus(id, status)
	if message != "" {
		appendHlsLog(id, message)
	}
	forgetHlsRuntime(id)
}

// hlsSegmentResult 单个分片的下载结果；按索引放在固定槽位以恢复写入顺序
type hlsSegmentResult struct {
	data []byte
	err  error
	done bool
}

// runHlsDownload 并发下载全部分片并按序写入 out，返回已写入字节数。
//
// 相比「按批并发 + 批间等待」，这里是持续的流水线：任一 worker 空闲即领取下
// 一个分片，不会被整批中最慢的分片拖住。分片按序号严格顺序落盘，写完立即释放
// 引用；在途分片数由 hlsDownloadWindow 限制，避免内存被整个文件占满。
//
// 初始化段（#EXT-X-MAP，fMP4 流）在对应分片之前写入：单段播放列表等价于
// 「文件头写一次」，多段合并（多源拼成一条播放列表）时则按序切换。
func runHlsDownload(ctx context.Context, taskID string, pl *hlsPlaylist, out *os.File, referer string, concurrency int) (int64, error) {
	total := len(pl.segments)
	var writtenBytes int64

	if total == 0 {
		return writtenBytes, nil
	}
	// 并发数由任务指定（未指定时取默认值），并夹在允许区间内
	concurrency = normalizeConcurrency(concurrency)

	// 并发自愈闸门：429/5xx 时临时减半分片并发给源站喘息，冷却后逐级恢复。
	// 挂到 ctx 上随所有 fetch 传递（分片 / 初始化段），无需层层改函数签名。
	gate := newHlsGate(concurrency)
	ctx = context.WithValue(ctx, hlsGateCtxKey{}, gate)

	keyCache := &sync.Map{}
	results := make([]hlsSegmentResult, total)
	// inflight 限制已派发但未落盘的分片数量（滑动窗口）
	inflight := make(chan struct{}, hlsWindowOf(concurrency))

	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	canceled := false

	// ctx 取消时唤醒写盘协程，避免它永久等待尚未就绪的分片
	stopWatch := context.AfterFunc(ctx, func() {
		mu.Lock()
		canceled = true
		cond.Broadcast()
		mu.Unlock()
	})
	defer stopWatch()

	// 写盘失败 / 取消后立即停止继续派发分片
	stopFeed := make(chan struct{})
	var stopOnce sync.Once
	stopDispatch := func() { stopOnce.Do(func() { close(stopFeed) }) }

	writeErr := make(chan error, 1)
	go func() {
		// 注意：此处不能直接用 utils.RecoverPanic——panic 后必须把错误回传给主
		// 协程，否则主协程会永久阻塞在写盘结果上。
		defer func() {
			if r := recover(); r != nil {
				utils.ErrorFormat("HLS 分片写盘协程异常: %v", r)
				writeErr <- fmt.Errorf("写盘协程异常: %v", r)
			}
		}()

		logStep := total / 20
		if logStep < 1 {
			logStep = 1
		}
		written := 0
		var lastNotify time.Time
		// 初始化段缓存 + 已写入标识：多源合并时按序切换，同址同区间只拉取一次
		initCache := &sync.Map{}
		writtenInitKey := ""
		for i := 0; i < total; i++ {
			mu.Lock()
			for !results[i].done && !canceled {
				cond.Wait()
			}
			if !results[i].done {
				mu.Unlock()
				writeErr <- errHlsCanceled
				return
			}
			data, segErr := results[i].data, results[i].err
			results[i] = hlsSegmentResult{} // 及时释放，避免整个文件常驻内存
			mu.Unlock()
			<-inflight

			if segErr != nil {
				stopDispatch()
				writeErr <- segErr
				return
			}
			// 初始化段必须先于它的分片落盘
			if initKey := hlsInitKey(pl.segments[i]); initKey != "" && initKey != writtenInitKey {
				initData, initErr := hlsInitBytes(ctx, pl.segments[i], referer, initCache)
				if initErr != nil {
					stopDispatch()
					writeErr <- initErr
					return
				}
				if _, err := out.Write(initData); err != nil {
					stopDispatch()
					writeErr <- err
					return
				}
				writtenBytes += int64(len(initData))
				writtenInitKey = initKey
			}
			if len(data) > 0 {
				if _, err := out.Write(data); err != nil {
					stopDispatch()
					writeErr <- err
					return
				}
				writtenBytes += int64(len(data))
			}
			written++
			// 进度回写节流：分片很小时避免高频加锁 + SSE 广播
			if written == total || time.Since(lastNotify) >= hlsProgressInterval {
				lastNotify = time.Now()
				updateHlsProgress(taskID, written, total, writtenBytes)
			}
			if written == total || written%logStep == 0 {
				appendHlsLog(taskID, fmt.Sprintf("已下载 %d/%d 分片", written, total))
			}
		}
		writeErr <- nil
	}()

	// 下载协程池：并发拉取 + 解密，结果落到对应槽位
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				seg := pl.segments[idx]
				// 即便单个分片处理时 panic，也必须把结果写回槽位并置 done，
				// 否则写盘协程会永久等待该分片
				data, err := func() (data []byte, err error) {
					defer func() {
						if r := recover(); r != nil {
							utils.ErrorFormat("HLS 分片下载协程异常: %v", r)
							err = fmt.Errorf("分片 %d 处理异常", seg.index)
						}
					}()
					return downloadHlsSegment(ctx, pl, seg, referer, keyCache)
				}()
				mu.Lock()
				results[idx] = hlsSegmentResult{data: data, err: err, done: true}
				cond.Broadcast()
				mu.Unlock()
			}
		}()
	}

	// 派发分片：取消或写盘失败时立刻停止
	var feedErr error
dispatch:
	for idx := range pl.segments {
		if ctx.Err() != nil {
			feedErr = errHlsCanceled
			break
		}
		select {
		case inflight <- struct{}{}:
		case <-ctx.Done():
			feedErr = errHlsCanceled
			break dispatch
		case <-stopFeed:
			break dispatch
		}
		jobs <- idx
	}
	close(jobs)
	wg.Wait()

	if err := <-writeErr; err != nil {
		return writtenBytes, err
	}
	return writtenBytes, feedErr
}

// HlsDownloader 任务执行入口（由调度器调用）
func HlsDownloader(task model.TransferTaskModel) utils.Result {
	ctx, cancel := context.WithCancel(context.Background())
	setHlsCancel(task.ID, cancel)
	defer cancel()

	playlistText := hlsPlaylistOf(task.ID)
	if strings.TrimSpace(playlistText) == "" {
		finishHlsTask(task.ID, model.StatusFailed, "播放列表内容丢失，无法继续下载")
		return utils.NewFailByMsg("播放列表内容丢失")
	}

	// 任务（尤其是重启的失败任务）用的是创建时保存的播放列表文本，
	// 分片地址上的时效签名可能已过期：先按路径把新的鉴权参数换上再解析
	authRefreshed := false
	if refreshed := refreshPlaylistAuth(ctx, playlistText, task.URL); refreshed != playlistText {
		playlistText = refreshed
		authRefreshed = true
	}

	pl, err := parseHlsPlaylistText(playlistText, task.URL)
	if err != nil {
		finishHlsTask(task.ID, model.StatusFailed, "解析播放列表失败: "+err.Error())
		return utils.NewFailByMsg("解析播放列表失败")
	}
	for _, seg := range pl.segments {
		if seg.key != nil && seg.key.method == "SAMPLE-AES" {
			finishHlsTask(task.ID, model.StatusFailed, "该视频使用 SAMPLE-AES 加密，暂不支持服务端下载")
			return utils.NewFailByMsg("不支持的加密方式")
		}
	}

	dest := task.Path
	if dest == "" {
		dest = task.Dest
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		finishHlsTask(task.ID, model.StatusFailed, "创建下载目录失败: "+err.Error())
		return utils.NewFailByMsg("创建下载目录失败")
	}

	tmpPath := dest + ".part"
	out, err := os.Create(tmpPath)
	if err != nil {
		finishHlsTask(task.ID, model.StatusFailed, "创建文件失败: "+err.Error())
		return utils.NewFailByMsg("创建文件失败")
	}

	// 清空旧日志并写入任务头
	_ = os.Remove(TaskLogPath(task.ID))
	appendHlsLog(task.ID, fmt.Sprintf("开始下载 %d 个分片 -> %s", len(pl.segments), dest))
	if authRefreshed {
		appendHlsLog(task.ID, "已刷新分片地址上的鉴权参数")
	}
	if n := hlsInitMapCount(pl); n > 1 {
		appendHlsLog(task.ID, fmt.Sprintf("检测到 %d 个初始化段（多段 fMP4 合并），将按分片顺序写入", n))
	}

	size, runErr := runHlsDownload(ctx, task.ID, pl, out, hlsOrigin(task.URL), hlsConcurrencyOf(task.ID))
	closeErr := out.Close()

	if runErr != nil {
		_ = os.Remove(tmpPath)
		if runErr == errHlsCanceled {
			finishHlsTask(task.ID, model.StatusCancelled, "任务已取消")
			return utils.NewFailByMsg("任务已取消")
		}
		finishHlsTask(task.ID, model.StatusFailed, "下载失败: "+runErr.Error())
		return utils.NewFailByMsg("下载失败")
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		finishHlsTask(task.ID, model.StatusFailed, "写入文件失败: "+closeErr.Error())
		return utils.NewFailByMsg("写入文件失败")
	}

	_ = os.Remove(dest)
	if err := os.Rename(tmpPath, dest); err != nil {
		_ = os.Remove(tmpPath)
		finishHlsTask(task.ID, model.StatusFailed, "保存文件失败: "+err.Error())
		return utils.NewFailByMsg("保存文件失败")
	}

	updateHlsProgress(task.ID, len(pl.segments), len(pl.segments), size)
	finishHlsTask(task.ID, model.StatusCompleted, fmt.Sprintf("下载完成: %s (%s)", dest, formatHlsSize(size)))
	maybeScanDownloaded(dest)
	// 提交时选择了转码方式：下载产物按路径直接建转码任务（不依赖索引是否已扫描）
	maybeTranscodeAfterDownload(task.ID, dest, task.TranscodeAfter)

	return utils.NewSuccessByMsg("下载完成")
}

// maybeTranscodeAfterDownload 下载完成后按路径创建后续转码任务。
// 失败只写任务日志，不改变下载任务本身的完成状态。
func maybeTranscodeAfterDownload(taskID string, dest string, xcode string) {
	if xcode == "" {
		return
	}
	// 转码目标固定是 mp4：下载产物已经是 mp4 时再转一次没有意义，
	// 且转码为避免覆盖源文件会把目标改成 mov（见 transferWithEncoderRetry），
	// 结果反而更差，这里直接跳过
	if strings.EqualFold(utils.GetSuffix(dest), "mp4") {
		appendHlsLog(taskID, fmt.Sprintf("下载产物已是 mp4，跳过转码（%s）", xcode))
		return
	}
	res := CreateTransferTaskByPath(dest, filepath.Base(dest), xcode)
	if res.IsSuccess() {
		appendHlsLog(taskID, fmt.Sprintf("已创建转码任务（%s -> mp4）", xcode))
		return
	}
	appendHlsLog(taskID, fmt.Sprintf("创建转码任务失败: %s", res.Message))
	utils.ErrorFormat("maybeTranscodeAfterDownload: dest=%s, xcode=%s, 错误: %s", dest, xcode, res.Message)
}

// maybeScanDownloaded 若目标文件落在已配置的媒体目录内，触发一次增量扫描
func maybeScanDownloaded(destPath string) {
	search := GetSearch()
	if search == nil {
		return
	}
	cleanDest := filepath.Clean(destPath)
	for _, dir := range GetOSSetting().Dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		base := filepath.Clean(dir)
		if strings.HasPrefix(strings.ToLower(cleanDest), strings.ToLower(base)+strings.ToLower(string(filepath.Separator))) {
			search.ScanTarget(filepath.Dir(cleanDest))
			return
		}
	}
}

// ── 创建 / 取消 ──────────────────────────────────────────────

// HlsDownloadParam 分片下载请求参数
type HlsDownloadParam struct {
	// Playlist 重建后的 m3u8 文本（可选分片删除结果，也可为多个播放列表合并后的
	// 结果——多源之间以 #EXT-X-DISCONTINUITY 分隔、各自带自己的 #EXT-X-MAP），
	// 地址应为绝对地址
	Playlist string `json:"playlist"`
	// SourceURL 原始 m3u8 地址，用于补全相对地址与展示
	SourceURL string `json:"sourceUrl"`
	// FileName 期望保存的文件名（可含扩展名）
	FileName string `json:"fileName"`
	// Dir 服务端保存目录，留空则使用「工作目录/downloads」
	Dir string `json:"dir"`
	// Xcode 下载完成后自动转码的方式：copy（仅换封装为 mp4）/ h264 / h265；
	// 留空表示下载后不转码
	Xcode string `json:"xcode"`
	// Concurrency 该任务内同时下载的分片数（1~16）。
	// 未传 / 传 0 / 超出区间时由服务端取默认值（4）或截断到上限
	Concurrency int `json:"concurrency"`
	// Parallel 服务端同时执行的任务数上限（1~16）。
	// 未传 / 非法时保持服务端现有配置不变；传了则即时生效并写入设置
	Parallel int `json:"parallel"`
}

// 允许在下载完成后自动执行的转码方式
var hlsAllowedXcode = map[string]bool{"copy": true, "h264": true, "h265": true}

var hlsInvalidNameChars = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)
var hlsExtRe = regexp.MustCompile(`\.[A-Za-z0-9]{1,8}$`)

func sanitizeHlsFileName(name string) string {
	name = strings.TrimSpace(name)
	name = hlsInvalidNameChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, " .")
	if r := []rune(name); len(r) > 120 {
		name = string(r[:120])
	}
	return name
}

func hlsFileNameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	base := filepath.Base(u.Path)
	if base == "." || base == "/" || base == "" {
		return ""
	}
	// 去掉 m3u8 后缀，保留主体名
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// nextAvailablePath 目标已存在时自动追加序号，避免覆盖
func nextAvailablePath(p string) string {
	if !utils.ExistsFiles(p) {
		return p
	}
	ext := filepath.Ext(p)
	stem := strings.TrimSuffix(p, ext)
	for i := 1; i < 1000; i++ {
		cand := fmt.Sprintf("%s (%d)%s", stem, i, ext)
		if !utils.ExistsFiles(cand) {
			return cand
		}
	}
	return p
}

// formatHlsSize 将字节数格式化为可读文本
func formatHlsSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(size)
	idx := -1
	for value >= unit && idx < len(units)-1 {
		value /= unit
		idx++
	}
	return fmt.Sprintf("%.2f %s", value, units[idx])
}

func formatHlsDuration(seconds float64) string {
	total := int(seconds)
	if total <= 0 {
		return ""
	}
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

// CreateHlsDownloadTask 创建分片下载任务（服务端异步执行）
func CreateHlsDownloadTask(param HlsDownloadParam) utils.Result {
	playlist := strings.TrimSpace(param.Playlist)
	if playlist == "" || !strings.Contains(playlist, "#EXTM3U") {
		return utils.NewFailByMsg("播放列表内容无效")
	}

	xcode := strings.ToLower(strings.TrimSpace(param.Xcode))
	if xcode != "" && !hlsAllowedXcode[xcode] {
		return utils.NewFailByMsg("转码方式无效（可选 copy / h264 / h265）")
	}

	// 并发下载数量：任务级，只影响本任务
	concurrency := normalizeConcurrency(param.Concurrency)
	// 并行任务数量：全局设置，随本次提交一并调整（不传则不动）
	applyTaskParallel(param.Parallel)

	pl, err := parseHlsPlaylistText(playlist, strings.TrimSpace(param.SourceURL))
	if err != nil {
		return utils.NewFailByMsg(err.Error())
	}

	fileName := sanitizeHlsFileName(param.FileName)
	if fileName == "" {
		fileName = sanitizeHlsFileName(hlsFileNameFromURL(param.SourceURL))
	}
	if fileName == "" {
		fileName = "video"
	}
	if !hlsExtRe.MatchString(fileName) {
		if pl.initMap != "" {
			fileName += ".mp4"
		} else {
			fileName += ".ts"
		}
	}

	// 原视频已是 mp4（fMP4：播放列表带 #EXT-X-MAP 初始化段）时不做转码——
	// 产物本来就是 mp4 封装，再转一次只是白白占用任务槽位
	if xcode != "" && pl.initMap != "" {
		utils.InfoFormat("CreateHlsDownloadTask: 原视频为 mp4，跳过转码（%s）", xcode)
		xcode = ""
	}

	dir := strings.TrimSpace(param.Dir)
	if dir == "" {
		// 默认存到第一个媒体目录：既能被索引扫描到，也能通过 /api/stream 直接回放
		if dirs := GetOSSetting().Dirs; len(dirs) > 0 && strings.TrimSpace(dirs[0]) != "" {
			dir = strings.TrimSpace(dirs[0])
		} else {
			dir = filepath.Join(GetWorkDir(), "downloads")
		}
	}
	dir = filepath.Clean(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return utils.NewFailByMsg("创建下载目录失败: " + err.Error())
	}

	dest := nextAvailablePath(filepath.Join(dir, fileName))

	task := model.NewHlsTask(strings.TrimSpace(param.SourceURL), dest, filepath.Base(dest), len(pl.segments), formatHlsDuration(pl.totalSeconds))
	task.SetStatus(model.StatusPending)
	// 下载完成后自动转码的方式（空=不转码），重启失败任务时同样保留
	task.TranscodeAfter = xcode

	TransferTaskMutex.Lock()
	if len(TransferTask) >= MaxTransferTaskCount {
		TransferTaskMutex.Unlock()
		return utils.NewFailByMsg("任务队列已满（最多1000个），请清理已完成任务后再试")
	}
	TransferTask[task.ID] = task
	PendingTaskCount.Add(1)
	TransferTaskMutex.Unlock()

	putHlsRuntimeConcurrency(task.ID, playlist, concurrency)
	// 播放列表落盘：任务失败后可据此重启（运行时内存会被丢弃）
	saveHlsPlaylist(task.ID, playlist)
	wakeTaskScheduler()

	utils.InfoFormat("CreateHlsDownloadTask: 创建成功 dest=%s, 分片数=%d, 并发=%d", dest, len(pl.segments), concurrency)
	LogTaskEvent("创建", task, fmt.Sprintf("dest=%s, 分片数=%d", dest, len(pl.segments)))
	return utils.NewSuccessByMsg("任务创建成功")
}

// CancelHlsDownload 取消进行中的分片下载任务
func CancelHlsDownload(taskID string) utils.Result {
	hlsRuntimeMutex.Lock()
	rt, ok := hlsRuntimeMap[taskID]
	hlsRuntimeMutex.Unlock()
	if !ok {
		return utils.NewFailByMsg("任务不存在或已结束")
	}
	if rt.cancel != nil {
		rt.cancel()
	}
	return utils.NewSuccessByMsg("已请求取消")
}

// RestartHlsDownload 重启失败/已取消的分片下载任务。
// 播放列表在任务创建时已落盘（HlsPlaylistPath），运行时内存被丢弃也能恢复；
// 复用原目标路径重新入队，由调度器重新执行。
func RestartHlsDownload(taskID string) utils.Result {
	TransferTaskMutex.RLock()
	task, ok := TransferTask[taskID]
	taskType, taskStatus := task.Type, task.Status
	TransferTaskMutex.RUnlock()
	if !ok || taskType != model.TaskTypeHls {
		return utils.NewFailByMsg("任务不存在或不是分片下载任务")
	}
	if taskStatus == model.StatusPending || taskStatus == model.StatusExecuting {
		return utils.NewFailByMsg("任务正在执行中，无需重启")
	}
	if taskStatus == model.StatusCompleted {
		return utils.NewFailByMsg("任务已完成，无需重启")
	}

	data, err := os.ReadFile(HlsPlaylistPath(taskID))
	if err != nil || len(data) == 0 {
		return utils.NewFailByMsg("播放列表已丢失，无法重启（请重新提交下载）")
	}

	putHlsRuntime(taskID, string(data))

	TransferTaskMutex.Lock()
	if t, ok := TransferTask[taskID]; ok {
		t.SetStatus(model.StatusPending)
		t.Segments = 0
		t.Progress = 0
		t.Size = 0
		t.Log = ""
		TransferTask[taskID] = t
	}
	TransferTaskMutex.Unlock()
	PendingTaskCount.Add(1)
	wakeTaskScheduler()

	utils.InfoFormat("RestartHlsDownload: 任务已重启 id=%s, dest=%s", taskID, task.Path)
	LogTaskEvent("重启", task, "dest="+task.Path)
	return utils.NewSuccessByMsg("任务已重新启动")
}
