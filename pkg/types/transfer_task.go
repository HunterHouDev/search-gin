package types

import (
	"fmt"
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
	Srt        string
	Type       string
	Start      string
	End        string
	From       string
	To         string
	CreateTime time.Time
	FinishTime time.Time
	Status     string
	Log        string
	VCode      string
	Command    string
	ConcatFile string

	Files        []string
	Dest         string
	DeleteSource bool
}

func NewMergeTask(files []string, dest string, concat string, DeleteSource bool) TransferTaskModel {
	now := time.Now()
	return TransferTaskModel{
		ID:           now.Format(time.RFC3339Nano),
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
		ID:         now.Format(time.RFC3339Nano),
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
		ID:         now.Format(time.RFC3339Nano),
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

func (p *TransferTaskModel) Key() string {
	return fmt.Sprintf("%s:%s:%d", p.Path, p.Type, p.CreateTime.UnixNano())
}

func (p *TransferTaskModel) GetLast() int64 {
	return (p.FinishTime.UnixMilli() - p.CreateTime.UnixMilli()) / 1000
}

func (p *TransferTaskModel) SetLog(log string) {
	p.Log = log
}
