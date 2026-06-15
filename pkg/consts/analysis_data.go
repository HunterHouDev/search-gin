package consts

import (
	"fmt"
	"sync"
	"time"
)

//环境引用
// true 静态文件
// false 打包二进制文佳 (要求打包html目录)
//初始化 扫描路径

var TypeMenu sync.Map
var SeriesCount sync.Map
var TagMenu sync.Map
var FolderTime sync.Map

var LogMemory = []Log{}
var logMemoryMutex sync.Mutex

func AddLogMemory(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	log := Log{Time: time.Now().Local().String(), Msg: msg}
	logMemoryMutex.Lock()
	LogMemory = append(LogMemory, log)
	// 硬上限 500 条，超过则保留最新的 400 条
	if n := len(LogMemory); n > 500 {
	 LogMemory = LogMemory[n-400:]
	}
	logMemoryMutex.Unlock()
}

type Log struct {
	Time string `json:"time"`
	Msg  string `json:"msg"`
}
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

func InitFolderTime() {
	FolderTime = sync.Map{}
}
func AddFolderTime(folder MenuSize) {
	FolderTime.LoadOrStore(folder.Name, folder)
}

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

// GetSyncMapCount 获取 sync.Map 的元素数量（原子安全）
func GetSyncMapCount(m *sync.Map) int {
	count := 0
	m.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

func TypeSizePlus(targetType string, targetSize int64) {
	if targetType == "" {
		targetType = "无"
	}
	TypeMenu.LoadOrStore("全部", MenuSize{
		Name: "全部",
		Cnt:  0,
		Size: 0,
	})
	target, ok := TypeMenu.LoadOrStore(targetType, MenuSize{
		Name: targetType,
		Cnt:  1,
		Size: targetSize,
	})
	if ok {
		TypeMenu.Store(targetType, target.(MenuSize).Plus(targetSize))
	}
	all, okAll := TypeMenu.Load("全部")
	if okAll {
		TypeMenu.Store("全部", all.(MenuSize).Plus(targetSize))
	}
}

func TagSizePlus(targetType string, targetSize int64) {

	target, ok := TagMenu.LoadOrStore(targetType, MenuSize{
		Name:  targetType,
		Cnt:   1,
		IsDir: true,
		Size:  targetSize,
	})
	if ok {
		TagMenu.Store(targetType, target.(MenuSize).Plus(targetSize))
	}
}

func SeriesPlus(targetType string, targetSize int64) {
	if len(targetType) == 0 {
		return
	}
	target, ok := SeriesCount.LoadOrStore(targetType, MenuSize{
		Name:  targetType,
		Cnt:   1,
		IsDir: true,
		Size:  targetSize,
	})
	if ok {
		SeriesCount.Store(targetType, target.(MenuSize).Plus(targetSize))
	}
}
