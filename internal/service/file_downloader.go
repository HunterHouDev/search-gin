package service

import (
	"math/rand"
	"net/http"
	"os"
	"search-gin/pkg/utils"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// ── 共享 HTTP 客户端（复用连接池） ─────────────────────────────────
// 不要在热路径上每次创建新 *http.Client，连接复用可显著降低延迟

// 远程搜索/操作用客户端
// NOTE: 两个客户端均有显式超时（peerClient 5s, remoteClient 2s），不存在"无超时永久阻塞"问题。
//       peerClient 超时已足够快：远程节点 5s 无响应即断开，避免 hang 死。
var (
	remoteClient = &http.Client{Timeout: remoteSearchTimeout}
	peerClient   = &http.Client{Timeout: 5 * time.Second}
)

// ── HTTP 客户端（resty） ───────────────────────────────────────────

var httpClient = resty.New().
	SetTimeout(10 * time.Second).
	SetRetryCount(3).
	SetRetryWaitTime(1 * time.Second).
	SetRetryMaxWaitTime(5 * time.Second).
	SetHeaders(map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language":           "zh-CN,zh;q=0.9,en;q=0.8",
		"Accept-Encoding":           "gzip, deflate, br",
		"Cache-Control":             "no-cache",
		"Pragma":                    "no-cache",
		"sec-ch-ua":                 `"Chromium";v="111", "Not_A Brand";v="8"`,
		"sec-ch-ua-mobile":          "?0",
		"sec-ch-ua-platform":        `"Windows"`,
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
	}).
	OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
		ua := browsers[rand.Intn(len(browsers))]
		r.SetHeader("User-Agent", ua)
		r.SetHeader("Cookie", "random="+strconv.Itoa(rand.Intn(999999)))
		return nil
	}).
	OnError(func(req *resty.Request, err error) {
		utils.InfoNormal("http请求失败:", err)
	})

// 常见浏览器UA列表，随机切换
var browsers = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/111.0",
}

// httpGet 发起 HTTP GET 请求
func httpGet(url string) (*resty.Response, error) {
	return httpClient.R().EnableTrace().Get(url)
}

// ── 图片下载 ──────────────────────────────────────────────────────

// DownJpgMakePng 下载 JPG，可选生成 PNG
func DownJpgMakePng(finalPath string, url string, makePng bool) utils.Result {
	result := utils.Result{}
	jpgPath := utils.ConcatSuffix(finalPath, "jpg")
	jpgOut, createErr := os.Create(jpgPath)
	if createErr != nil {
		result.Fail()
		result.Message = "文件创建失败：" + jpgPath
		return result
	}
	defer jpgOut.Close()

	if !strings.Contains(url, "https") {
		url = GetOSSetting().BaseUrl + url
	}
	start := time.Now()
	resp, downErr := httpGet(url)
	LogMem.Add("DownJpg  time:%d  %s %d", time.Since(start).Milliseconds(), url, downErr)
	if downErr != nil {
		result.Fail()
		result.Message = "文件下载失败：" + url
		return result
	}
	if _, err := jpgOut.Write(resp.Body()); err != nil {
		utils.InfoFormat("写入jpg失败: %v", err)
	}
	if makePng {
		if pngErr := utils.ImageToPng(jpgPath); pngErr != nil {
			utils.InfoFormat("pngErr:%v", pngErr)
		}
	}
	result.Success()
	return result
}

// DownJpgAsPng 下载并保存为 PNG
func DownJpgAsPng(finalPath string, url string) utils.Result {
	result := utils.Result{}
	pngPath := utils.ConcatSuffix(finalPath, "png")
	pngOut, createErr := os.Create(pngPath)
	if createErr != nil {
		result.Fail()
		return result
	}
	defer pngOut.Close()

	if !strings.Contains(url, "https") {
		url = GetOSSetting().BaseUrl + url
	}
	start := time.Now()
	resp, downErr := httpGet(url)
	LogMem.Add("DownPng  time:%d  %s %d", time.Since(start).Milliseconds(), url, downErr)
	if downErr != nil {
		result.Fail()
		result.Message = "文件下载失败：" + url
		return result
	}
	if _, err := pngOut.Write(resp.Body()); err != nil {
		utils.InfoFormat("写入png失败: %v", err)
	}
	result.Success()
	return result
}
