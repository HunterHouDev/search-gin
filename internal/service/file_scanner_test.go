package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ── WalkInner 测试 ──

func TestWalkInner_EmptyDir(t *testing.T) {
	// 创建临时空目录
	tmpDir := t.TempDir()

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	files, size := app.WalkInner(tmpDir, []string{"mp4", "avi"}, true, tmpDir)

	assert.Empty(t, files)
	assert.Equal(t, int64(0), size)
}

func TestWalkInner_WithFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试文件
	testFiles := []string{"test.mp4", "video.avi", "doc.txt"}
	for _, name := range testFiles {
		f, err := os.Create(filepath.Join(tmpDir, name))
		assert.NoError(t, err)
		f.Close()
	}

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	// 只扫描视频文件
	files, _ := app.WalkInner(tmpDir, []string{"mp4", "avi"}, true, tmpDir)

	// 应该找到 2 个视频文件
	assert.Equal(t, 2, len(files))

	names := make(map[string]bool)
	for _, f := range files {
		names[f.Name] = true
	}
	assert.True(t, names["test.mp4"])
	assert.True(t, names["video.avi"])
	assert.False(t, names["doc.txt"])
}

func TestWalkInner_SubDir(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	os.Mkdir(subDir, 0755)

	// 在子目录创建文件
	f, _ := os.Create(filepath.Join(subDir, "sub_video.mp4"))
	f.Close()

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	// queryChild=true 应该递归扫描子目录
	files, _ := app.WalkInner(tmpDir, []string{"mp4"}, true, tmpDir)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, "sub_video.mp4", files[0].Name)
}

func TestWalkInner_NoRecursion(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	os.Mkdir(subDir, 0755)

	// 在子目录创建文件
	f, _ := os.Create(filepath.Join(subDir, "sub_video.mp4"))
	f.Close()

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	// queryChild=false 不应该递归
	files, _ := app.WalkInner(tmpDir, []string{"mp4"}, false, tmpDir)
	assert.Empty(t, files)
}

// ── Walk 测试 ──

func TestWalk_ReturnsFiles(t *testing.T) {
	tmpDir := t.TempDir()

	f, _ := os.Create(filepath.Join(tmpDir, "test.mp4"))
	f.Close()

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	files := app.Walk(tmpDir, []string{"mp4"}, true)
	assert.Equal(t, 1, len(files))
}

// ── ScanDirs 测试 ──

func TestScanDirs_EmptyList(t *testing.T) {
	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	buckets := app.ScanDirs([]string{}, []string{"mp4"})
	assert.Empty(t, buckets)
}

func TestScanDirs_WithFiles(t *testing.T) {
	tmpDir := t.TempDir()

	f, _ := os.Create(filepath.Join(tmpDir, "test.mp4"))
	f.Close()

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	buckets := app.ScanDirs([]string{tmpDir}, []string{"mp4"})

	assert.Equal(t, 1, len(buckets))
	bucket, ok := buckets[tmpDir]
	assert.True(t, ok)
	assert.Equal(t, 1, bucket.TotalCount)
}

// ── stackItem 测试 ──

func TestStackItem(t *testing.T) {
	item := stackItem{
		path:       "/test/path",
		queryChild: true,
		visited:    false,
		fileCount:  10,
	}

	assert.Equal(t, "/test/path", item.path)
	assert.True(t, item.queryChild)
	assert.False(t, item.visited)
	assert.Equal(t, 10, item.fileCount)
}

// ── scanResult 测试 ──

func TestScanResult(t *testing.T) {
	bucket := newInstance("test")
	result := scanResult{
		dir:    "/test/dir",
		bucket: bucket,
	}

	assert.Equal(t, "/test/dir", result.dir)
	assert.NotNil(t, result.bucket)
	assert.Equal(t, "test", result.bucket.InstanceName)
}
