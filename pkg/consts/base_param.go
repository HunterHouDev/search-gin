package consts

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"sync"
	"time"
)

var LastScanTime time.Time

var QueryTypes []string

var MovieFields = utils.InterfaceFields(model.FileItem{})

var Types = []string{PNG, JPG, GIF, XLSX, TXT, MP4, WMV, MKV, AVI, JAVA, XML}
var Images = []string{PNG, JPG, GIF}

var IndexHtml = "./dist/index.html"
var StaticFs = map[string]string{
	"/css":    "./dist/css",
	"/js":     "./dist/js",
	"/assets": "./dist/assets",
}

// IndexNumber 索引构建中得目录数量
var IndexNumber = int32(0)

// ScanProgress 索引扫描/构建进度
type ScanProgress struct {
	Phase            string `json:"phase"`            // "idle" | "scanning" | "building" | "done"
	TotalDirs        int    `json:"totalDirs"`        // 待扫描目录总数
	CompletedDirs    int    `json:"completedDirs"`    // 已完成扫描的目录数
	CurrentDir       string `json:"currentDir"`       // 当前正在扫描的目录
	ScannedFiles     int64  `json:"scannedFiles"`     // 已扫描的文件数
	TotalBuckets     int    `json:"totalBuckets"`     // 待构建索引的 bucket 数
	ProcessedBuckets int    `json:"processedBuckets"` // 已构建完成的 bucket 数（索引构建阶段）
	CurrentPhase     string `json:"currentPhase"`     // 当前阶段描述，如"正在扫描目录..."、"正在构建索引..."等
}

var Sp ScanProgress
var SpMu sync.RWMutex

var TempImage = make(map[string]model.FileItem)
var TempImageMutex sync.RWMutex // 保护TempImage的并发访问

var TransferTask = map[time.Time]model.TransferTaskModel{}
var TransferTaskMutex sync.RWMutex // 保护TransferTask的并发访问

func init() {
 go func() {
  for {
   time.Sleep(10 * time.Minute)
   TempImageMutex.Lock()
   // 保留最近 500 条，清空多余
   if len(TempImage) > 500 {
    TempImage = make(map[string]model.FileItem)
   }
   TempImageMutex.Unlock()
  }
 }()
}

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

var PortNo = ":10081"
var FilePortNo = ":10082"
