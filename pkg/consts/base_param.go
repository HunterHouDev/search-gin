package consts

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"sync"
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

var MovieFields = utils.InterfaceFields(model.FileItem{})

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

// ScanProgress 索引扫描/构建进度
type ScanProgress struct {
	mu               *sync.RWMutex `json:"-"` // 内嵌写锁指针，JSON 不序列化
	Phase            string       `json:"phase"`            // "idle" | "scanning" | "building" | "done"
	TotalDirs        int          `json:"totalDirs"`        // 待扫描目录总数
	CompletedDirs    int          `json:"completedDirs"`    // 已完成扫描的目录数
	CurrentDir       string      `json:"currentDir"`       // 当前正在扫描的目录
	ScannedFiles     int64        `json:"scannedFiles"`     // 已扫描的文件数
	TotalBuckets     int          `json:"totalBuckets"`     // 待构建索引的 bucket 数
	ProcessedBuckets int          `json:"processedBuckets"` // 已构建完成的 bucket 数（索引构建阶段）
	CurrentPhase     string       `json:"currentPhase"`     // 当前阶段描述，如"正在扫描目录..."、"正在构建索引..."等
}

var Sp = ScanProgress{mu: &sync.RWMutex{}}

// ── ScanProgress 并发安全操作方法 ──────────────────────────

// Get 读锁安全拷贝，供外部读取进度
func (sp *ScanProgress) Get() ScanProgress {
	sp.mu.RLock()
	result := ScanProgress{
		Phase:            sp.Phase,
		TotalDirs:        sp.TotalDirs,
		CompletedDirs:    sp.CompletedDirs,
		CurrentDir:       sp.CurrentDir,
		ScannedFiles:     sp.ScannedFiles,
		TotalBuckets:     sp.TotalBuckets,
		ProcessedBuckets: sp.ProcessedBuckets,
		CurrentPhase:     sp.CurrentPhase,
	}
	sp.mu.RUnlock()
	return result
}

// Init 初始化扫描进度（写锁）
func (sp *ScanProgress) Init(dirCount int) {
	sp.mu.Lock()
	sp.Phase = "scanning"
	sp.TotalDirs = dirCount
	sp.CompletedDirs = 0
	sp.CurrentDir = ""
	sp.ScannedFiles = 0
	sp.TotalBuckets = dirCount
	sp.ProcessedBuckets = 0
	sp.CurrentPhase = "正在扫描目录..."
	sp.mu.Unlock()
}

// setPhase 内部设置阶段和描述（不加锁，由调用方持锁）
func (sp *ScanProgress) setPhase(phase, desc string) {
	sp.Phase = phase
	sp.CurrentPhase = desc
}

// SetPhase 更新阶段和描述（写锁）
func (sp *ScanProgress) SetPhase(phase, desc string) {
	sp.mu.Lock()
	sp.setPhase(phase, desc)
	sp.mu.Unlock()
}

// Complete 标记扫描完成（写锁）
func (sp *ScanProgress) Complete() {
	sp.mu.Lock()
	sp.setPhase("done", "扫描完成")
	sp.CompletedDirs = sp.TotalDirs
	sp.mu.Unlock()
}

// IncrementCompletedDirs 已完成目录数+1（写锁）
func (sp *ScanProgress) IncrementCompletedDirs() {
	sp.mu.Lock()
	sp.CompletedDirs++
	sp.mu.Unlock()
}

// SetCurrentDir 设置当前扫描目录（写锁）
func (sp *ScanProgress) SetCurrentDir(dir string) {
	sp.mu.Lock()
	sp.CurrentDir = dir
	sp.mu.Unlock()
}

// AddScannedFiles 增加已扫描文件数（写锁）
func (sp *ScanProgress) AddScannedFiles(count int64) {
	sp.mu.Lock()
	sp.ScannedFiles += count
	sp.mu.Unlock()
}

var TransferTask = map[time.Time]model.TransferTaskModel{}
var TransferTaskMutex sync.RWMutex // 保护TransferTask的并发访问

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
