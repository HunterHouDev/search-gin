package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRunHlsDownloadPreservesOrder 校验并发下载后严格按分片序号拼接：
// 服务端故意让靠前的分片慢返回，若写盘顺序依赖完成顺序，结果就会错位。
func TestRunHlsDownloadPreservesOrder(t *testing.T) {
	const total = 16

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idx, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/seg"))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		time.Sleep(time.Duration(total-idx) * 5 * time.Millisecond)
		fmt.Fprintf(w, "seg-%02d|", idx)
	}))
	defer srv.Close()

	text := "#EXTM3U\n#EXT-X-MEDIA-SEQUENCE:0\n"
	for i := 1; i <= total; i++ {
		text += fmt.Sprintf("#EXTINF:1.0,\n%s/seg%d\n", srv.URL, i)
	}
	pl, err := parseHlsPlaylistText(text, srv.URL+"/index.m3u8")
	assert.NoError(t, err)

	out, err := os.Create(filepath.Join(t.TempDir(), "video.ts"))
	assert.NoError(t, err)
	defer out.Close()

	// 并发传 0：走服务端默认并发
	n, err := runHlsDownload(context.Background(), "test-order", pl, out, srv.URL+"/", 0)
	assert.NoError(t, err)

	var want strings.Builder
	for i := 1; i <= total; i++ {
		fmt.Fprintf(&want, "seg-%02d|", i)
	}
	got, err := os.ReadFile(out.Name())
	assert.NoError(t, err)
	assert.Equal(t, want.String(), string(got))
	assert.Equal(t, int64(want.Len()), n)
}

// TestRunHlsDownloadRetriesSegment 校验单个分片失败后自动重试，不再让整个任务失败
func TestRunHlsDownloadRetriesSegment(t *testing.T) {
	var attempts int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/seg1" {
			attempts++
			if attempts < 2 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		fmt.Fprint(w, r.URL.Path)
	}))
	defer srv.Close()

	text := fmt.Sprintf("#EXTM3U\n#EXTINF:1.0,\n%s/seg1\n", srv.URL)
	pl, err := parseHlsPlaylistText(text, srv.URL+"/index.m3u8")
	assert.NoError(t, err)

	out, err := os.Create(filepath.Join(t.TempDir(), "video.ts"))
	assert.NoError(t, err)
	defer out.Close()

	// 并发传 0：走服务端默认并发
	_, err = runHlsDownload(context.Background(), "test-retry", pl, out, "", 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)

	got, err := os.ReadFile(out.Name())
	assert.NoError(t, err)
	assert.Equal(t, "/seg1", string(got))
}

// TestRunHlsDownloadWritesInitSegmentPerPart 校验多个播放列表合并后的文本：
// 每个源自己的初始化段（#EXT-X-MAP）必须写在该源分片之前，且同一初始化段只拉取一次。
func TestRunHlsDownloadWritesInitSegmentPerPart(t *testing.T) {
	var mu sync.Mutex
	requests := map[string]int{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests[r.URL.Path]++
		mu.Unlock()

		name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/"), ".mp4")
		fmt.Fprintf(w, "%s|", name)
	}))
	defer srv.Close()

	text := strings.Join([]string{
		"#EXTM3U",
		"#EXT-X-TARGETDURATION:4",
		fmt.Sprintf(`#EXT-X-MAP:URI="%s/initA.mp4"`, srv.URL),
		"#EXTINF:4.0,",
		srv.URL + "/a1",
		"#EXTINF:4.0,",
		srv.URL + "/a2",
		"#EXT-X-DISCONTINUITY",
		fmt.Sprintf(`#EXT-X-MAP:URI="%s/initB.mp4"`, srv.URL),
		"#EXTINF:4.0,",
		srv.URL + "/b1",
		"#EXTINF:4.0,",
		srv.URL + "/b2",
	}, "\n")

	pl, err := parseHlsPlaylistText(text, srv.URL+"/index.m3u8")
	assert.NoError(t, err)
	// 首个初始化段用于判定输出容器，初始化段总数用于提示多段合并
	assert.Equal(t, srv.URL+"/initA.mp4", pl.initMap)
	assert.Equal(t, 2, hlsInitMapCount(pl))

	out, err := os.Create(filepath.Join(t.TempDir(), "video.mp4"))
	assert.NoError(t, err)
	defer out.Close()

	// 并发传 0：走服务端默认并发
	_, err = runHlsDownload(context.Background(), "test-init", pl, out, "", 0)
	assert.NoError(t, err)

	got, err := os.ReadFile(out.Name())
	assert.NoError(t, err)
	assert.Equal(t, "initA|a1|a2|initB|b1|b2|", string(got))

	// 同一初始化段被两个分片共用，只应拉取一次
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, requests["/initA.mp4"])
	assert.Equal(t, 1, requests["/initB.mp4"])
}

// TestRunHlsDownloadCanceled 校验取消后立即返回取消错误，而不是等待全部完成
func TestRunHlsDownloadCanceled(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		fmt.Fprint(w, "x")
	}))
	defer srv.Close()
	defer close(release)

	text := ""
	for i := 1; i <= 8; i++ {
		text += fmt.Sprintf("#EXTINF:1.0,\n%s/seg%d\n", srv.URL, i)
	}
	pl, err := parseHlsPlaylistText("#EXTM3U\n"+text, srv.URL+"/index.m3u8")
	assert.NoError(t, err)

	out, err := os.Create(filepath.Join(t.TempDir(), "video.ts"))
	assert.NoError(t, err)
	defer out.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// 并发传 0：走服务端默认并发
	_, err = runHlsDownload(ctx, "test-cancel", pl, out, "", 0)
	assert.ErrorIs(t, err, errHlsCanceled)
}

// TestHlsNetworkErrorClassifiesReset 校验「连接被源站强行关闭」被识别为可重试的网络错误。
//
// 真实报错形如：read tcp 192.168.3.2:12504->219.155.150.155:443: wsarecv:
// An existing connection was forcibly closed by the remote host.
func TestHlsNetworkErrorClassifiesReset(t *testing.T) {
	resetErr := &net.OpError{
		Op:  "read",
		Net: "tcp",
		Err: os.NewSyscallError("wsarecv", syscall.ECONNRESET),
	}
	assert.True(t, hlsNetworkError(resetErr))
	assert.False(t, hlsThrottled(resetErr))
	// HTTP 状态码类错误不算网络错误，避免重试策略串味
	assert.False(t, hlsNetworkError(hlsHTTPStatusError{code: 503}))
}

// TestHlsJitterRange 校验退避抖动落在 [d, 1.25d] 区间内
func TestHlsJitterRange(t *testing.T) {
	base := time.Second
	for i := 0; i < 50; i++ {
		got := hlsJitter(base)
		assert.GreaterOrEqual(t, got, base)
		assert.LessOrEqual(t, got, base+base/4)
	}
	assert.Equal(t, time.Duration(0), hlsJitter(0))
}

// TestRefreshPlaylistAuthUpdatesQuery 校验过期鉴权参数被按路径替换为新值：
// 分片行、#EXT-X-KEY 与 #EXT-X-MAP 的 URI 都要换，其余内容保持原样。
func TestRefreshPlaylistAuthUpdatesQuery(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.Join([]string{
			"#EXTM3U",
			`#EXT-X-KEY:METHOD=AES-128,URI="`+srv.URL+`/key.bin?auth_key=NEW",IV=0x01`,
			`#EXT-X-MAP:URI="`+srv.URL+`/init.mp4?auth_key=NEW"`,
			"#EXTINF:4.0,",
			srv.URL + "/seg1.m4s?auth_key=NEW",
			"#EXTINF:4.0,",
			srv.URL + "/seg2.m4s?auth_key=NEW",
		}, "\n"))
	}))
	defer srv.Close()

	old := strings.Join([]string{
		"#EXTM3U",
		`#EXT-X-KEY:METHOD=AES-128,URI="` + srv.URL + `/key.bin?auth_key=OLD",IV=0x01`,
		`#EXT-X-MAP:URI="` + srv.URL + `/init.mp4?auth_key=OLD"`,
		"#EXTINF:4.0,",
		srv.URL + "/seg1.m4s?auth_key=OLD",
		"#EXTINF:4.0,",
		srv.URL + "/seg2.m4s?auth_key=OLD",
	}, "\n")

	got := refreshPlaylistAuth(context.Background(), old, srv.URL+"/index.m3u8")
	assert.NotContains(t, got, "auth_key=OLD")
	// 密钥 URI、初始化段 URI、两个分片地址，共 4 处
	assert.Equal(t, 4, strings.Count(got, "auth_key=NEW"))
	// 未被替换的部分（标签本身、时长行）保持原样
	assert.Contains(t, got, "#EXT-X-KEY:METHOD=AES-128")
	assert.Contains(t, got, "#EXTINF:4.0,")
}

// TestRefreshPlaylistAuthKeepsOriginalOnFailure 校验源站不可用时原样返回
func TestRefreshPlaylistAuthKeepsOriginalOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	old := "#EXTM3U\nhttp://example.com/seg1.m4s?auth_key=OLD\n"
	assert.Equal(t, old, refreshPlaylistAuth(context.Background(), old, srv.URL+"/index.m3u8"))
	// 源地址缺失时同样直接返回
	assert.Equal(t, old, refreshPlaylistAuth(context.Background(), old, ""))
}

// TestRefreshPlaylistAuthSkipsUnsigned 校验地址里没有时效签名时直接跳过刷新，
// 不为了刷新白白多打一次源站。
func TestRefreshPlaylistAuthSkipsUnsigned(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		fmt.Fprint(w, "#EXTM3U\nhttp://example.com/seg1.m4s\n")
	}))
	defer srv.Close()

	old := "#EXTM3U\nhttp://example.com/seg1.m4s\n"
	assert.Equal(t, old, refreshPlaylistAuth(context.Background(), old, srv.URL+"/index.m3u8"))
	assert.Equal(t, int32(0), atomic.LoadInt32(&hits))
}

// TestFetchHlsSegmentRetriesOnConnectionReset 校验连接被掐断后会换连接重试成功。
// 服务端前两次直接关闭 TCP 连接（模拟源站单方面 RST），第三次才返回数据。
func TestFetchHlsSegmentRetriesOnConnectionReset(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) <= 2 {
			// 直接断开底层连接，让客户端收到 read: connection reset
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, err := hj.Hijack()
				if err == nil {
					_ = conn.Close()
					return
				}
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "payload")
	}))
	defer srv.Close()

	data, err := fetchHlsSegment(context.Background(), srv.URL+"/seg1", nil, "")
	assert.NoError(t, err)
	assert.Equal(t, "payload", string(data))
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

// TestNormalizeConcurrency 校验任务级并发数的归一化：
// 未指定（≤0）取默认值，超出上限截断；滑动窗口随并发数放大。
func TestNormalizeConcurrency(t *testing.T) {
	assert.Equal(t, hlsDownloadConcurrency, normalizeConcurrency(0))
	assert.Equal(t, hlsDownloadConcurrency, normalizeConcurrency(-3))
	assert.Equal(t, 1, normalizeConcurrency(1))
	assert.Equal(t, 8, normalizeConcurrency(8))
	assert.Equal(t, hlsMaxConcurrency, normalizeConcurrency(hlsMaxConcurrency+1))

	assert.Equal(t, hlsDownloadConcurrency*3, hlsWindowOf(0))
	assert.Equal(t, 12, hlsWindowOf(4))
}
