package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
	// hlsDownloadConcurrency 同时在途的分片请求数。
	// 分片下载是纯网络 I/O，并发数可以高于任务槽位数（默认 4）。
	hlsDownloadConcurrency = 8
	// hlsDownloadWindow 已派发但尚未落盘的分片上限（滑动窗口）。
	// 顺序写盘时若窗口无限，最前面的慢分片会让后续分片全部堆积在内存里。
	hlsDownloadWindow = hlsDownloadConcurrency * 3
	// hlsSegmentRetry 单个分片的最大尝试次数（含首次）
	hlsSegmentRetry = 3
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
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	},
}

// ── 运行时状态（播放列表文本 + 取消句柄） ──────────────────────

type hlsRuntime struct {
	playlist string
	cancel   context.CancelFunc
}

var (
	hlsRuntimeMap   = map[string]*hlsRuntime{}
	hlsRuntimeMutex sync.Mutex
)

func putHlsRuntime(id string, playlist string) {
	hlsRuntimeMutex.Lock()
	hlsRuntimeMap[id] = &hlsRuntime{playlist: playlist}
	hlsRuntimeMutex.Unlock()
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

// parseHlsPlaylistText 解析 m3u8 文本；base 用于补全相对地址
func parseHlsPlaylistText(text, base string) (*hlsPlaylist, error) {
	pl := &hlsPlaylist{}
	var currentKey *hlsKey
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
				pl.initMap = resolveHlsURL(base, attrs["URI"])
				if br, ok := attrs["BYTERANGE"]; ok {
					pl.initRange = parseHlsByteRange(br, 0)
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

func fetchHlsBytes(ctx context.Context, rawURL string, br *hlsByteRange, referer string) ([]byte, error) {
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
	resp, err := hlsHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// fetchHlsSegment 带重试的分片拉取：CDN 偶发 5xx / 连接重置时，
// 由该分片自己重试，不再让整个任务失败（旧实现单点失败即整任务终止）。
func fetchHlsSegment(ctx context.Context, rawURL string, br *hlsByteRange, referer string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < hlsSegmentRetry; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, errHlsCanceled
			case <-time.After(time.Duration(attempt) * 300 * time.Millisecond):
			}
		}
		data, err := fetchHlsBytes(ctx, rawURL, br, referer)
		if err == nil {
			return data, nil
		}
		if ctx.Err() != nil {
			return nil, errHlsCanceled
		}
		lastErr = err
	}
	return nil, lastErr
}

// downloadHlsSegment 下载并按需解密一个分片
func downloadHlsSegment(ctx context.Context, pl *hlsPlaylist, seg hlsSegmentItem, referer string, keyCache *sync.Map) ([]byte, error) {
	data, err := fetchHlsSegment(ctx, seg.url, seg.byteRange, referer)
	if err != nil {
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
	data, err := fetchHlsBytes(ctx, keyURL, nil, referer)
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
func runHlsDownload(ctx context.Context, taskID string, pl *hlsPlaylist, out *os.File, referer string) (int64, error) {
	total := len(pl.segments)
	var writtenBytes int64

	// 初始化分片（#EXT-X-MAP，fMP4 流）必须最先写入
	if pl.initMap != "" {
		data, err := fetchHlsSegment(ctx, pl.initMap, pl.initRange, referer)
		if err != nil {
			return 0, fmt.Errorf("下载初始化分片失败: %w", err)
		}
		if _, err := out.Write(data); err != nil {
			return 0, err
		}
		writtenBytes += int64(len(data))
	}
	if total == 0 {
		return writtenBytes, nil
	}

	keyCache := &sync.Map{}
	results := make([]hlsSegmentResult, total)
	// inflight 限制已派发但未落盘的分片数量（滑动窗口）
	inflight := make(chan struct{}, hlsDownloadWindow)

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
	for w := 0; w < hlsDownloadConcurrency; w++ {
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

	size, runErr := runHlsDownload(ctx, task.ID, pl, out, hlsOrigin(task.URL))
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

	return utils.NewSuccessByMsg("下载完成")
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
	// Playlist 重建后的 m3u8 文本（可选分片删除结果），地址应为绝对地址
	Playlist string `json:"playlist"`
	// SourceURL 原始 m3u8 地址，用于补全相对地址与展示
	SourceURL string `json:"sourceUrl"`
	// FileName 期望保存的文件名（可含扩展名）
	FileName string `json:"fileName"`
	// Dir 服务端保存目录，留空则使用「工作目录/downloads」
	Dir string `json:"dir"`
}

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

	TransferTaskMutex.Lock()
	if len(TransferTask) >= MaxTransferTaskCount {
		TransferTaskMutex.Unlock()
		return utils.NewFailByMsg("任务队列已满（最多1000个），请清理已完成任务后再试")
	}
	TransferTask[task.ID] = task
	PendingTaskCount.Add(1)
	TransferTaskMutex.Unlock()

	putHlsRuntime(task.ID, playlist)
	wakeTaskScheduler()

	utils.InfoFormat("CreateHlsDownloadTask: 创建成功 dest=%s, 分片数=%d", dest, len(pl.segments))
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
