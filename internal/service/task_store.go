package service

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"search-gin/internal/model"
	"search-gin/pkg/utils"
)

// ──────────────────────────────────────────────────────────────
// 下载列表持久化
//
// 索引快照走 gob + 去抖异步写入（见 index_engine_cache.go），这里沿用同一套做法。
// 分片下载虽然跑在服务端，任务表却只在内存里：进程一退出，下载列表整片消失，
// 已经下了一半的 .part 也没人管了。存下来之后：
//
//	重启 → 延迟回填任务表 → 未完成的任务重新排队（有 .part 时自动续传）。
//
// 只存分片下载（TaskTypeHls）：转码 / 剪切 / 合并 / 磁力都依赖进程内的运行时
// （ffmpeg 子进程、torrent 客户端），恢复出来只会变成没人管的僵尸任务。
// ──────────────────────────────────────────────────────────────

const (
	// taskStoreFileName 下载列表快照文件（工作目录下）
	taskStoreFileName = "download_tasks.gob"
	// taskStoreVersion 快照格式版本：结构变更时递增，旧快照直接忽略
	taskStoreVersion = 1
	// taskStoreDebounce 去抖窗口：连续的状态变更合并成一次磁盘写入
	taskStoreDebounce = 3 * time.Second
)

// hlsTaskStore 可序列化的下载列表快照
type hlsTaskStore struct {
	Version int
	Tasks   []model.TransferTaskModel
}

var (
	taskStoreMu    sync.Mutex
	taskStoreTimer *time.Timer
)

// SaveHlsTaskStore 记录一次「下载列表变了」：去抖后异步写盘。
// 任务的创建 / 开始执行 / 结束 / 删除都要调用它。
func SaveHlsTaskStore() {
	if GetWorkDir() == "" {
		return
	}
	taskStoreMu.Lock()
	defer taskStoreMu.Unlock()
	if taskStoreTimer != nil {
		taskStoreTimer.Stop()
	}
	taskStoreTimer = time.AfterFunc(taskStoreDebounce, flushHlsTaskStore)
}

// flushHlsTaskStore 去抖到期：异步写一次快照
func flushHlsTaskStore() {
	taskStoreMu.Lock()
	taskStoreTimer = nil
	taskStoreMu.Unlock()

	go func() {
		defer utils.RecoverPanic()
		doFlushHlsTaskStore()
	}()
}

// FlushHlsTaskStore 同步强制写入（退出前调用，避免丢掉去抖窗口内的变更）
func FlushHlsTaskStore() {
	taskStoreMu.Lock()
	if taskStoreTimer != nil {
		taskStoreTimer.Stop()
		taskStoreTimer = nil
	}
	taskStoreMu.Unlock()
	doFlushHlsTaskStore()
}

// snapshotHlsTasks 复制一份当前的分片下载任务（读锁内完成，序列化时不长期持锁）
func snapshotHlsTasks() []model.TransferTaskModel {
	TransferTaskMutex.RLock()
	defer TransferTaskMutex.RUnlock()
	out := make([]model.TransferTaskModel, 0, len(TransferTask))
	for _, t := range TransferTask {
		if strings.EqualFold(t.Type, model.TaskTypeHls) {
			out = append(out, t)
		}
	}
	return out
}

// doFlushHlsTaskStore 序列化 + 落盘（临时文件 + 重命名，避免写一半被读到）
func doFlushHlsTaskStore() {
	if GetWorkDir() == "" {
		return
	}
	tasks := snapshotHlsTasks()
	// 与任务列表的展示顺序一致：最新在上
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreateTime.After(tasks[j].CreateTime) })

	path := filepath.Join(GetWorkDir(), taskStoreFileName)
	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		utils.ErrorFormat("保存下载列表失败(创建临时文件): %v", err)
		return
	}
	if err := gob.NewEncoder(f).Encode(hlsTaskStore{Version: taskStoreVersion, Tasks: tasks}); err != nil {
		f.Close()
		os.Remove(tmpPath)
		utils.ErrorFormat("保存下载列表失败(编码): %v", err)
		return
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		utils.ErrorFormat("保存下载列表失败(关闭文件): %v", err)
		return
	}
	// Windows 上目标已存在时 os.Rename 会失败，先删再改名
	os.Remove(path)
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		utils.ErrorFormat("保存下载列表失败(重命名): %v", err)
	}
}

// LoadHlsTaskStore 从磁盘恢复下载列表，返回恢复的任务数。
//
// 未完成（等待 / 执行中）的任务重置为等待并重新排队——进程退出时它们并没有真的跑完，
// 重新排队后若磁盘上还留着 .part，就会从断点继续（见 hlsResumeOffset）。
func LoadHlsTaskStore() int {
	if GetWorkDir() == "" {
		return 0
	}
	path := filepath.Join(GetWorkDir(), taskStoreFileName)
	if _, err := os.Stat(path); err != nil {
		return 0
	}
	f, err := os.Open(path)
	if err != nil {
		utils.InfoFormat("加载下载列表失败(打开文件): %v", err)
		return 0
	}
	defer f.Close()

	var data hlsTaskStore
	if err := gob.NewDecoder(f).Decode(&data); err != nil {
		utils.InfoFormat("加载下载列表失败(解码): %v，忽略该快照", err)
		return 0
	}
	if data.Version != taskStoreVersion {
		utils.InfoFormat("下载列表快照版本不匹配(%d != %d)，忽略", data.Version, taskStoreVersion)
		return 0
	}

	restored, requeued := 0, 0
	TransferTaskMutex.Lock()
	for _, t := range data.Tasks {
		if t.ID == "" || !strings.EqualFold(t.Type, model.TaskTypeHls) {
			continue
		}
		if _, exists := TransferTask[t.ID]; exists {
			continue
		}
		if len(TransferTask) >= MaxTransferTaskCount {
			break
		}
		switch t.Status {
		case model.StatusExecuting, model.StatusPending:
			// 进程退出后没有真正在跑的下载：回队列继续（有 .part 就续传）
			t.Status = model.StatusPending
			PendingTaskCount.Add(1)
			requeued++
		}
		TransferTask[t.ID] = t
		restored++
	}
	TransferTaskMutex.Unlock()

	if restored == 0 {
		return 0
	}
	if requeued > 0 {
		wakeTaskScheduler()
	}
	utils.InfoFormat("LoadHlsTaskStore: 恢复下载任务 %d 个（其中 %d 个重新排队）", restored, requeued)
	LogMem.Add("下载列表已恢复：%d 个任务（重新排队 %d 个）", restored, requeued)
	return restored
}

// LoadHlsTaskStoreDelayed 延迟加载：等后台调度与 HTTP 服务都起来之后再回填，
// 避免启动阶段读盘和首屏扫描抢 IO，也不拖慢启动。
func LoadHlsTaskStoreDelayed(delay time.Duration) {
	if delay < 0 {
		delay = 0
	}
	go func() {
		defer utils.RecoverPanic()
		time.Sleep(delay)
		LoadHlsTaskStore()
	}()
}
