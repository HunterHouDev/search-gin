package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ── DeleteFilesOnDisk 测试 ──

// 删除主文件及 jpg/png/srt 附属文件
func TestDeleteFilesOnDisk_DeletesMainAndSidecars(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "video.mp4"), []byte("main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "video.jpg"), []byte("jpg"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "video.png"), []byte("png"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "video.srt"), []byte("srt"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	app.DeleteFilesOnDisk(tmpDir, "video")

	assert.NoFileExists(t, filepath.Join(tmpDir, "video.mp4"))
	assert.NoFileExists(t, filepath.Join(tmpDir, "video.jpg"))
	assert.NoFileExists(t, filepath.Join(tmpDir, "video.png"))
	assert.NoFileExists(t, filepath.Join(tmpDir, "video.srt"))
}

// EqualFold 匹配不区分大小写（VIDEO.MP4 → video）
func TestDeleteFilesOnDisk_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "VIDEO.MP4"), []byte("main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "Video.JPG"), []byte("jpg"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	app.DeleteFilesOnDisk(tmpDir, "video")

	assert.NoFileExists(t, filepath.Join(tmpDir, "VIDEO.MP4"))
	assert.NoFileExists(t, filepath.Join(tmpDir, "Video.JPG"))
}

// 精确匹配基名+扩展名，不误伤前缀相同但基名不同的文件
func TestDeleteFilesOnDisk_DoesNotDeleteDifferentBaseName(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "video.mp4"), []byte("main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "video_extra.mp4"), []byte("extra"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "videogame.txt"), []byte("game"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	app.DeleteFilesOnDisk(tmpDir, "video")

	assert.NoFileExists(t, filepath.Join(tmpDir, "video.mp4"))
	assert.FileExists(t, filepath.Join(tmpDir, "video_extra.mp4"), "不应误删 video_extra.mp4")
	assert.FileExists(t, filepath.Join(tmpDir, "videogame.txt"), "不应误删 videogame.txt")
}

// 空 fileName 不执行任何删除
func TestDeleteFilesOnDisk_EmptyFileName_NoOp(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "video.mp4"), []byte("main"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	app.DeleteFilesOnDisk(tmpDir, "")

	assert.FileExists(t, filepath.Join(tmpDir, "video.mp4"))
}

// 目录不存在时不 panic，仅打日志
func TestDeleteFilesOnDisk_NonexistentDir_NoPanic(t *testing.T) {
	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	// 不应 panic
	app.DeleteFilesOnDisk("/nonexistent/path", "video")
}

// 验证所有已知附属扩展名（mp4/jpg/png/gif/srt）均被删除，无关文件保留
func TestDeleteFilesOnDisk_DeletesAllSidecarExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "clip.mp4"), []byte("main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "clip.jpg"), []byte("jpg"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "clip.png"), []byte("png"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "clip.gif"), []byte("gif"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "clip.srt"), []byte("srt"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("other"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	app.DeleteFilesOnDisk(tmpDir, "clip")

	assert.NoFileExists(t, filepath.Join(tmpDir, "clip.mp4"))
	assert.NoFileExists(t, filepath.Join(tmpDir, "clip.jpg"))
	assert.NoFileExists(t, filepath.Join(tmpDir, "clip.png"))
	assert.NoFileExists(t, filepath.Join(tmpDir, "clip.gif"))
	assert.NoFileExists(t, filepath.Join(tmpDir, "clip.srt"))
	assert.FileExists(t, filepath.Join(tmpDir, "other.txt"), "不应误删无关文件")
}
