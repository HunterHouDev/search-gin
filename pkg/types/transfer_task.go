package types

import (
	"strings"
	"time"
)

const (
	TaskTypeCut   = "分切"
	TaskTypeMerge = "合并"
	TaskTypeTrans = "转码"
	// TaskTypeHls HLS 分片下载：服务端拉取 m3u8 分片并合并为一个本地文件，
	// 任务落在服务端，关闭弹窗 / 刷新页面都不影响下载
	TaskTypeHls = "分片下载"
)

const (
	StatusPending   = "等待"
	StatusExecuting = "执行中"
	StatusCompleted = "完成"
	StatusFailed    = "失败"
	StatusCancelled = "取消"
)

const UndefinedStr = "undefined"

type TransferTaskModel struct {
	ID         string
	Name       string
	Path       string
	Type       string
	Start      string
	End        string
	From       string
	To         string
	CreateTime time.Time
	FinishTime time.Time
	Status     string
	VCode      string
	Command    string
	ConcatFile string

	Log          string
	Files        []string
	Dest         string
	DeleteSource bool

	// ── HLS 分片下载（TaskTypeHls）专用字段 ──
	// URL 源 m3u8 地址（展示用）
	URL string
	// Segments 已完成分片数（下载中实时更新）
	Segments int
	// TotalSegments 分片总数
	TotalSegments int
	// Progress 进度百分比 0~100
	Progress int
	// Size 已写入字节数
	Size int64
	// Duration 分片总时长文本（如 12:34）
	Duration string
}

func NewMergeTask(files []string, dest string, concat string, DeleteSource bool) TransferTaskModel {
	now := time.Now()
	return TransferTaskModel{
		ID:           safeTaskID(now),
		Files:        files,
		Type:         TaskTypeMerge,
		Dest:         dest,
		VCode:        "copy",
		ConcatFile:   concat,
		DeleteSource: DeleteSource,
		CreateTime:   now,
	}
}

func NewTask(path string, name string, from string, to string) TransferTaskModel {
	now := time.Now()
	return TransferTaskModel{
		ID:         safeTaskID(now),
		Path:       path,
		Type:       TaskTypeTrans,
		VCode:      "copy",
		Name:       name,
		From:       from,
		To:         to,
		CreateTime: now,
	}
}

func NewCutTask(path string, name string, start string, end string, to string) TransferTaskModel {
	now := time.Now()
	return TransferTaskModel{
		ID:         safeTaskID(now),
		Path:       path,
		Type:       TaskTypeCut,
		Name:       name,
		Start:      start,
		End:        end,
		To:         to,
		CreateTime: now,
	}
}

// NewHlsTask 创建 HLS 分片下载任务。
// dest 为最终保存的完整文件路径，name 为文件名，total 为分片总数。
func NewHlsTask(url, dest, name string, total int, duration string) TransferTaskModel {
	now := time.Now()
	return TransferTaskModel{
		ID:            safeTaskID(now),
		Type:          TaskTypeHls,
		URL:           url,
		Path:          dest,
		Dest:          dest,
		Name:          name,
		TotalSegments: total,
		Duration:      duration,
		CreateTime:    now,
	}
}

func (p *TransferTaskModel) SetStatus(sts string) {
	p.Status = sts
}

// safeTaskID 生成不含 `:` 的任务 ID（Windows 文件名安全）
func safeTaskID(t time.Time) string {
	return strings.ReplaceAll(t.Format(time.RFC3339Nano), ":", "-")
}
