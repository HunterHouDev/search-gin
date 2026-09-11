package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

	n, err := runHlsDownload(context.Background(), "test-order", pl, out, srv.URL+"/")
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

	_, err = runHlsDownload(context.Background(), "test-retry", pl, out, "")
	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)

	got, err := os.ReadFile(out.Name())
	assert.NoError(t, err)
	assert.Equal(t, "/seg1", string(got))
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

	_, err = runHlsDownload(ctx, "test-cancel", pl, out, "")
	assert.ErrorIs(t, err, errHlsCanceled)
}
