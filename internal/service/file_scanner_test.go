package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ── WalkInner 测试 ──

func TestWalkInner_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	files, size := app.WalkDirWithCfg(tmpDir,[]string{"mp4", "avi"}, true)

	assert.Empty(t, files)
	assert.Equal(t, int64(0), size)
}

func TestWalkInner_WithFiles(t *testing.T) {
	tmpDir := t.TempDir()

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

	files, _ := app.WalkDirWithCfg(tmpDir,[]string{"mp4", "avi"}, true)

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

	f, _ := os.Create(filepath.Join(subDir, "sub_video.mp4"))
	f.Close()

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	files, _ := app.WalkDirWithCfg(tmpDir,[]string{"mp4"}, true)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, "sub_video.mp4", files[0].Name)
}

func TestWalkInner_NoRecursion(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	os.Mkdir(subDir, 0755)

	f, _ := os.Create(filepath.Join(subDir, "sub_video.mp4"))
	f.Close()

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	files, _ := app.WalkDirWithCfg(tmpDir,[]string{"mp4"}, false)
	assert.Empty(t, files)
}

func TestWalkInner_NonexistentDir(t *testing.T) {
	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	files, _ := app.WalkDirWithCfg("/nonexistent/path", []string{"mp4"}, true)
	assert.Empty(t, files)
}

func TestWalkInner_MultipleSubDirs(t *testing.T) {
	tmpDir := t.TempDir()
	sub1 := filepath.Join(tmpDir, "sub1")
	sub2 := filepath.Join(tmpDir, "sub2")
	os.Mkdir(sub1, 0755)
	os.Mkdir(sub2, 0755)

	f1, _ := os.Create(filepath.Join(sub1, "video1.mp4"))
	f1.Close()
	f2, _ := os.Create(filepath.Join(sub2, "video2.mp4"))
	f2.Close()

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	files, _ := app.WalkDirWithCfg(tmpDir,[]string{"mp4"}, true)
	assert.Equal(t, 2, len(files))
}

func TestWalkInner_FileSizeCalculation(t *testing.T) {
	tmpDir := t.TempDir()
	f, _ := os.Create(filepath.Join(tmpDir, "test.mp4"))
	f.Write([]byte("hello world"))
	f.Close()

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	_, size := app.WalkDirWithCfg(tmpDir,[]string{"mp4"}, true)
	assert.Equal(t, int64(11), size)
}

func TestWalkInner_SetsMovieNode(t *testing.T) {
	tmpDir := t.TempDir()
	f, _ := os.Create(filepath.Join(tmpDir, "test.mp4"))
	f.Close()

	// 初始化节点信息
	LocalNodeHost = "test:10081"
	LocalNodeName = "test"
	defer func() {
		LocalNodeHost = ""
		LocalNodeName = ""
	}()

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	files, _ := app.WalkDirWithCfg(tmpDir,[]string{"mp4"}, true)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, "test:10081", files[0].NodeHost)
	assert.Equal(t, "test", files[0].NodeName)
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

func TestScanDirs_MultipleDirs(t *testing.T) {
	tmpDir := t.TempDir()
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")
	os.Mkdir(dir1, 0755)
	os.Mkdir(dir2, 0755)

	os.WriteFile(filepath.Join(dir1, "video1.mp4"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(dir2, "video2.mp4"), []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	buckets := app.ScanDirs([]string{dir1, dir2}, []string{"mp4"})

	assert.Equal(t, 2, len(buckets))
}

// ── ScanTarget 测试 ──

func TestScanTarget_AddsTask(t *testing.T) {
	tmpDir := t.TempDir()

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	app.ScanTarget(tmpDir)

	assert.Equal(t, 1, scanQueue.GetTaskCount())
}

// ── stackItem 测试 ──

func TestStackItem(t *testing.T) {
	item := stackItem{
		path:       "/test/path",
		queryChild: true,
		visited:    false,
	}

	assert.Equal(t, "/test/path", item.path)
	assert.True(t, item.queryChild)
	assert.False(t, item.visited)
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
