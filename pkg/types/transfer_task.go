package types

import (
	"strings"
	"time"
)

const (
	TaskTypeCut   = "分切"
	TaskTypeMerge = "合并"
	TaskTypeTrans = "转码"
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

func (p *TransferTaskModel) SetStatus(sts string) {
	p.Status = sts
}

// safeTaskID 生成不含 `:` 的任务 ID（Windows 文件名安全）
func safeTaskID(t time.Time) string {
	return strings.ReplaceAll(t.Format(time.RFC3339Nano), ":", "-")
}
