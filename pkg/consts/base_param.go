package consts

import (
	"sync/atomic"
	"time"
)

var lastScanTime atomic.Int64

// SetLastScanTime 设置最近扫描时间（并发安全）
func SetLastScanTime(t time.Time) {
	lastScanTime.Store(t.UnixNano())
}

// GetLastScanTime 获取最近扫描时间（并发安全）
func GetLastScanTime() time.Time {
	return time.Unix(0, lastScanTime.Load())
}

var QueryTypes []string

var MovieFields []string

var Types = []string{PNG, JPG, GIF, XLSX, TXT, MP4, WMV, MKV, AVI, JAVA, XML}
var Images = []string{PNG, JPG, GIF}

var IndexHtml = "./dist/index.html"
var StaticFs = map[string]string{
	"/css":    "./dist/css",
	"/js":     "./dist/js",
	"/assets": "./dist/assets",
}

// IndexNumber 索引构建中的目录数量（并发安全）
var IndexNumber atomic.Int32

var PortNo = ":10081"
var FilePortNo = ":10082"

// PNG Base Dictory
const PNG = "png"
const JPG = "jpg"
const GIF = "gif"
const XLSX = "xlsx"
const TXT = "txt"
const MP4 = "mp4"
const WMV = "wmv"
const MKV = "mkv"
const AVI = "avi"
const JAVA = "java"
const XML = "xml"
