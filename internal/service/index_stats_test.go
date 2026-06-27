package service

import (
	"testing"

	"search-gin/internal/model"

	"github.com/stretchr/testify/assert"
)

// ── MemoryLog ──

func TestMemoryLog_AddAndGetAll(t *testing.T) {
	LogMem.Add("test %s", "entry")
	entries := LogMem.GetAll()
	assert.NotEmpty(t, entries)
	assert.Contains(t, entries[len(entries)-1].Msg, "test entry")
}

func TestMemoryLog_AddManyTrims(t *testing.T) {
	LogMem.Add("first")
	for i := 0; i < logMemoryMaxLines+50; i++ {
		LogMem.Add("line %d", i)
	}
	entries := LogMem.GetAll()
	assert.True(t, len(entries) <= logMemoryMaxLines)
}

func TestMemoryLog_GetAllCopyIsolation(t *testing.T) {
	LogMem.Add("isolation test")
	entries := LogMem.GetAll()
	entries[0].Msg = "modified"
	entries2 := LogMem.GetAll()
	assert.NotEqual(t, "modified", entries2[0].Msg)
}

// ── SmallDir ──

func TestSmallDir_AppendAndGet(t *testing.T) {
	ClearSmallDir()
	assert.Empty(t, GetSmallDir())

	AppendSmallDir(model.FileInfo{Name: "tiny_dir", Size: 100})
	assert.Len(t, GetSmallDir(), 1)
	assert.Equal(t, "tiny_dir", GetSmallDir()[0].Name)
}

func TestSmallDir_Clear(t *testing.T) {
	AppendSmallDir(model.FileInfo{Name: "temp"})
	ClearSmallDir()
	assert.Empty(t, GetSmallDir())
}

func TestSmallDir_GetReturnsCopy(t *testing.T) {
	ClearSmallDir()
	AppendSmallDir(model.FileInfo{Name: "original"})

	entries := GetSmallDir()
	entries[0].Name = "modified"

	entries2 := GetSmallDir()
	assert.Equal(t, "original", entries2[0].Name)
}


