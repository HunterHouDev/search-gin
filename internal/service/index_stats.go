package service

import (
	"fmt"
	"search-gin/pkg/utils"
	"sync"
	"time"
)

// ─── MenuSize 菜单条目（用于类型/标签/系列统计） ────────────────

type MenuSize struct {
	Name    string
	Cnt     int64
	Size    int64
	SizeStr string
	IsDir   bool
}

func NewMenuSize(name string, size int64) MenuSize {
	cnt := int64(0)
	if size > 0 {
		cnt = int64(1)
	}
	return MenuSize{
		Name: name,
		Cnt:  cnt,
		Size: size,
	}
}

func NewMenuSizeFold(name string, size int64, isFold bool) MenuSize {
	cnt := int64(0)
	if size > 0 {
		cnt = int64(1)
	}
	return MenuSize{
		Name:  name,
		Cnt:   cnt,
		Size:  size,
		IsDir: isFold,
	}
}

func (m MenuSize) Plus(size int64) MenuSize {
	m.Cnt++
	m.Size += size
	return m
}

func (m MenuSize) Minus(size int64) MenuSize {
	m.Cnt--
	m.Size -= size
	if m.Cnt < 0 {
		m.Cnt = 0
	}
	if m.Size < 0 {
		m.Size = 0
	}
	return m
}

// ─── 全局菜单（线程安全） ─────────────────────────────────────────

var (
	TypeMenu    sync.Map
	SeriesCount sync.Map
	TagMenu     sync.Map
	FolderTime  sync.Map
)

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

var SmallDir []MenuSize
var smallDirMutex sync.Mutex

func AppendSmallDir(item MenuSize) {
	smallDirMutex.Lock()
	SmallDir = append(SmallDir, item)
	smallDirMutex.Unlock()
}

func GetSmallDir() []MenuSize {
	smallDirMutex.Lock()
	result := make([]MenuSize, len(SmallDir))
	copy(result, SmallDir)
	smallDirMutex.Unlock()
	return result
}

func ClearSmallDir() {
	smallDirMutex.Lock()
	SmallDir = []MenuSize{}
	smallDirMutex.Unlock()
}

// ─── 辅助函数 ────────────────────────────────────────────────────

// GetSyncMapCount 获取 sync.Map 的元素数量
func GetSyncMapCount(m *sync.Map) int {
	count := 0
	m.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

func InitFolderTime() {
	FolderTime = sync.Map{}
}

func AddFolderTime(folder MenuSize) {
	FolderTime.LoadOrStore(folder.Name, folder)
}

func TypeSizePlus(targetType string, targetSize int64) {
	if targetType == "" {
		targetType = "无"
	}
	TypeMenu.LoadOrStore("全部", MenuSize{
		Name: "全部", Cnt: 0, Size: 0,
	})
	target, ok := TypeMenu.LoadOrStore(targetType, MenuSize{
		Name: targetType, Cnt: 1, Size: targetSize,
	})
	if ok {
		if t, ok2 := target.(MenuSize); ok2 {
			TypeMenu.Store(targetType, t.Plus(targetSize))
		}
	}
	all, okAll := TypeMenu.Load("全部")
	if okAll {
		if a, ok2 := all.(MenuSize); ok2 {
			TypeMenu.Store("全部", a.Plus(targetSize))
		}
	}
}

func TagSizePlus(targetType string, targetSize int64) {
	target, ok := TagMenu.LoadOrStore(targetType, MenuSize{
		Name: targetType, Cnt: 1, IsDir: true, Size: targetSize,
	})
	if ok {
		if t, ok2 := target.(MenuSize); ok2 {
			TagMenu.Store(targetType, t.Plus(targetSize))
		}
	}
}

func SeriesPlus(targetType string, targetSize int64) {
	if len(targetType) == 0 {
		return
	}
	target, ok := SeriesCount.LoadOrStore(targetType, MenuSize{
		Name: targetType, Cnt: 1, IsDir: true, Size: targetSize,
	})
	if ok {
		if t, ok2 := target.(MenuSize); ok2 {
			SeriesCount.Store(targetType, t.Plus(targetSize))
		}
	}
}
