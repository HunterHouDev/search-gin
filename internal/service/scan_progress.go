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
	defer sp.mu.RUnlock()
	return ScanProgress{
		Phase:            sp.Phase,
		TotalDirs:        sp.TotalDirs,
		CompletedDirs:    sp.CompletedDirs,
		CurrentDir:       sp.CurrentDir,
		ScannedFiles:     sp.ScannedFiles,
		TotalBuckets:     sp.TotalBuckets,
		ProcessedBuckets: sp.ProcessedBuckets,
		CurrentPhase:     sp.CurrentPhase,
	}
}

// Init 初始化扫描进度（写锁）
func (sp *ScanProgress) Init(dirCount int) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.Phase = "scanning"
	sp.TotalDirs = dirCount
	sp.CompletedDirs = 0
	sp.CurrentDir = ""
	sp.ScannedFiles = 0
	sp.TotalBuckets = dirCount
	sp.ProcessedBuckets = 0
	sp.CurrentPhase = "正在扫描目录..."
}

// setPhase 内部设置阶段和描述（不加锁，由调用方持锁）
func (sp *ScanProgress) setPhase(phase, desc string) {
	sp.Phase = phase
	sp.CurrentPhase = desc
}

// SetPhase 更新阶段和描述（写锁）
func (sp *ScanProgress) SetPhase(phase, desc string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.setPhase(phase, desc)
}

// Complete 标记扫描完成（写锁）
func (sp *ScanProgress) Complete() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.setPhase("done", "扫描完成")
	sp.CompletedDirs = sp.TotalDirs
}

// IncrementCompletedDirs 已完成目录数+1（写锁）
func (sp *ScanProgress) IncrementCompletedDirs() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.CompletedDirs++
}

// SetCurrentDir 设置当前扫描目录（写锁）
func (sp *ScanProgress) SetCurrentDir(dir string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.CurrentDir = dir
}

// AddScannedFiles 增加已扫描文件数（写锁）
func (sp *ScanProgress) AddScannedFiles(count int64) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.ScannedFiles += count
}

// IncrementProcessedBuckets 已构建 bucket 数+1（写锁）
func (sp *ScanProgress) IncrementProcessedBuckets() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.ProcessedBuckets++
}
