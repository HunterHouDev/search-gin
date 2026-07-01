package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	assert.Equal(t, "分切", TaskTypeCut)
	assert.Equal(t, "合并", TaskTypeMerge)
	assert.Equal(t, "转码", TaskTypeTrans)
	assert.Equal(t, "等待", StatusPending)
	assert.Equal(t, "执行中", StatusExecuting)
	assert.Equal(t, "完成", StatusCompleted)
	assert.Equal(t, "失败", StatusFailed)
	assert.Equal(t, "取消", StatusCancelled)
	assert.Equal(t, "undefined", UndefinedStr)
}

func TestNewMergeTask(t *testing.T) {
	files := []string{"/a.mp4", "/b.mp4"}
	now := time.Now()
	task := NewMergeTask(files, "/dest.mp4", "/concat.txt", true)

	assert.Equal(t, TaskTypeMerge, task.Type)
	assert.Equal(t, files, task.Files)
	assert.Equal(t, "/dest.mp4", task.Dest)
	assert.Equal(t, "/concat.txt", task.ConcatFile)
	assert.True(t, task.DeleteSource)
	assert.Equal(t, "copy", task.VCode)
	assert.WithinRange(t, task.CreateTime, now.Add(-time.Second), now.Add(time.Second))
}

func TestNewTask(t *testing.T) {
	task := NewTask("/path/to/video.mp4", "test-video", "from", "to")

	assert.Equal(t, TaskTypeTrans, task.Type)
	assert.Equal(t, "/path/to/video.mp4", task.Path)
	assert.Equal(t, "test-video", task.Name)
	assert.Equal(t, "from", task.From)
	assert.Equal(t, "to", task.To)
	assert.Equal(t, "copy", task.VCode)
}

func TestNewCutTask(t *testing.T) {
	task := NewCutTask("/path.mp4", "clip1", "00:00:10", "00:00:30", "/out.mp4")

	assert.Equal(t, TaskTypeCut, task.Type)
	assert.Equal(t, "/path.mp4", task.Path)
	assert.Equal(t, "clip1", task.Name)
	assert.Equal(t, "00:00:10", task.Start)
	assert.Equal(t, "00:00:30", task.End)
	assert.Equal(t, "/out.mp4", task.To)
}

func TestSetStatus(t *testing.T) {
	task := TransferTaskModel{}
	task.SetStatus(StatusCompleted)
	assert.Equal(t, StatusCompleted, task.Status)
}
