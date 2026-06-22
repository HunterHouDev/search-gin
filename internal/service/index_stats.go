package service

import (
	"fmt"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"sync"
	"time"
)

// ─── 扫描计时 ─────────────────────────────────────────────────────

var folderTime sync.Map

func InitFolderTime()                { folderTime = sync.Map{} }
func AddFolderTime(f model.FileInfo) { folderTime.LoadOrStore(f.Name, f) }
func GetFolderTime() *sync.Map       { return &folderTime }

// ─── 内存日志 ────────────────────────────────────────────────────

const logMemoryMaxLines = 1000
const logMemoryTrimLines = 800

// MemoryLog 内存日志存储
type MemoryLog struct {
	mu   *sync.Mutex `json:"-"`
	logs []LogEntry
}

// LogEntry 日志条目
type LogEntry struct {
	Time string `json:"time"`
	Msg  string `json:"msg"`
}

var LogMem = MemoryLog{mu: &sync.Mutex{}}

// Add 写入一条日志
func (ml *MemoryLog) Add(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	entry := LogEntry{Time: time.Now().Local().String(), Msg: msg}
	ml.mu.Lock()
	ml.logs = append(ml.logs, entry)
	if n := len(ml.logs); n > logMemoryMaxLines {
		ml.logs = ml.logs[n-logMemoryTrimLines:]
	}
	ml.mu.Unlock()
	utils.InfoFormat(format, v...)
}

// GetAll 返回全部日志
func (ml *MemoryLog) GetAll() []LogEntry {
	ml.mu.Lock()
	result := make([]LogEntry, len(ml.logs))
	copy(result, ml.logs)
	ml.mu.Unlock()
	return result
}

// ─── 小文件目录 ──────────────────────────────────────────────────

var SmallDir []model.FileInfo
var smallDirMutex sync.Mutex

func AppendSmallDir(item model.FileInfo) {
	smallDirMutex.Lock()
	SmallDir = append(SmallDir, item)
	smallDirMutex.Unlock()
}

func GetSmallDir() []model.FileInfo {
	smallDirMutex.Lock()
	result := make([]model.FileInfo, len(SmallDir))
	copy(result, SmallDir)
	smallDirMutex.Unlock()
	return result
}

func ClearSmallDir() {
	smallDirMutex.Lock()
	SmallDir = []model.FileInfo{}
	smallDirMutex.Unlock()
}
