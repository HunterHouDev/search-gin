package service

import "sync"

// ScanProgress 索引扫描/构建进度
type ScanProgress struct {
	mu               *sync.RWMutex `json:"-"` // 内嵌写锁指针，JSON 不序列化
	Phase            string        `json:"phase"`
	TotalDirs        int           `json:"totalDirs"`
	CompletedDirs    int           `json:"completedDirs"`
	CurrentDir       string        `json:"currentDir"`
	ScannedFiles     int64         `json:"scannedFiles"`
	TotalBuckets     int           `json:"totalBuckets"`
	ProcessedBuckets int           `json:"processedBuckets"`
	CurrentPhase     string        `json:"currentPhase"`
}

var Sp = ScanProgress{mu: &sync.RWMutex{}}

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
