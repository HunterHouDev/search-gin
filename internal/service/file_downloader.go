package service

import (
	"encoding/base64"
	"errors"
	"fmt"
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

// dataImageTag 图片 data URI 的前缀与 base64 分隔符
const (
	dataImagePrefix  = "data:image/"
	dataImageBase64  = ";base64,"
	// 解码后的上限：请求体统一按 10MB 截断，base64 有 4/3 膨胀，
	// 故解码上限取 6MB（对应约 8MB 的 data URI），避免策略自相矛盾
	dataImageMaxSize = 6 << 20
)

// IsDataImage 判断是否为内联 base64 图片（data:image/xxx;base64,...）
func IsDataImage(src string) bool {
	return strings.HasPrefix(src, dataImagePrefix) && strings.Contains(src, dataImageBase64)
}

// DecodeDataImage 解析内联 base64 图片，返回原始图片字节
func DecodeDataImage(src string) ([]byte, error) {
	idx := strings.Index(src, dataImageBase64)
	if idx < 0 {
		return nil, errors.New("缺少 base64 分隔符")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(src[idx+len(dataImageBase64):]))
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, errors.New("图片内容为空")
	}
	if len(raw) > dataImageMaxSize {
		return nil, fmt.Errorf("图片超过 %d MB", dataImageMaxSize>>20)
	}
	return raw, nil
}

// hasImageSource 判断封面字段是否为可落盘的图片来源：http(s) 链接或内联 base64 图片
func hasImageSource(src string) bool {
	return strings.HasPrefix(src, "http") || IsDataImage(src)
}

// fetchImageBytes 取回图片字节：内联 base64 直接解码，其余走 HTTP 下载。
// brief 用于日志，避免把整张 base64 图片写进日志。
func fetchImageBytes(url string, brief string) ([]byte, error) {
	if IsDataImage(url) {
		return DecodeDataImage(url)
	}
	if !strings.Contains(url, "https") {
		url = GetOSSetting().BaseUrl + url
	}
	start := time.Now()
	resp, downErr := httpGet(url)
	LogMem.Add("%s  time:%d  %s %d", brief, time.Since(start).Milliseconds(), url, downErr)
	if downErr != nil {
		// 带上地址，便于从失败提示里直接看出是哪个源挂了
		return nil, fmt.Errorf("%w: %s", downErr, url)
	}
	return resp.Body(), nil
}

// imageDownFailMsg 封面取图失败的文案：区分内联 base64 与 HTTP 下载
func imageDownFailMsg(url string, err error) string {
	if IsDataImage(url) {
		return "base64 图片解析失败：" + err.Error()
	}
	return "文件下载失败：" + err.Error()
}

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

	data, downErr := fetchImageBytes(url, "DownJpg")
	if downErr != nil {
		result.Fail()
		result.Message = imageDownFailMsg(url, downErr)
		return result
	}
	if _, err := jpgOut.Write(data); err != nil {
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

	data, downErr := fetchImageBytes(url, "DownPng")
	if downErr != nil {
		result.Fail()
		result.Message = imageDownFailMsg(url, downErr)
		return result
	}
	if _, err := pngOut.Write(data); err != nil {
		utils.InfoFormat("写入png失败: %v", err)
	}
	result.Success()
	return result
}
