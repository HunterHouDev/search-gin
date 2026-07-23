package service

import (
	"os"
	"path/filepath"
	"search-gin/internal/model"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDelete_RealTitle_RemovesDiskFile 验证：Delete 整体流程调用 FileItem.Delete 后，磁盘文件被真删。
func TestDelete_RealTitle_RemovesDiskFile(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video.mp4")
	assert.NoError(t, os.WriteFile(origPath, []byte("test"), 0644))

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.EasyFile(tmpDir, origPath, "video.mp4", "mp4", 100, time.Now(), tmpDir)
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	res := app.Delete(movie.Id)
	assert.True(t, res.IsSuccess(), "Delete 应返回成功")
	_, err := os.Stat(origPath)
	assert.True(t, os.IsNotExist(err), "磁盘文件应被真正删除, 实际仍存在: %s", origPath)
}

// TestFileItemDelete_DotInBasename 验证修复：基础名含点号（"Movie.2024.1080p.mp4"）时，
// 基于完整路径精确删除，不再因对 Title 二次 TrimSuffix 匹配错位而静默不删。
func TestFileItemDelete_DotInBasename(t *testing.T) {
	tmpDir := t.TempDir()
	name := "Movie.2024.1080p.mp4"
	p := filepath.Join(tmpDir, name)
	assert.NoError(t, os.WriteFile(p, []byte("x"), 0644))
	// 同名 sidecar
	assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Movie.2024.1080p.srt"), []byte("x"), 0644))

	model.FileItem{Path: p}.Delete()

	_, err := os.Stat(p)
	assert.True(t, os.IsNotExist(err), "含点号基础名的主文件应被删除")
	_, err2 := os.Stat(filepath.Join(tmpDir, "Movie.2024.1080p.srt"))
	assert.True(t, os.IsNotExist(err2), "含点号基础名的 sidecar 也应被删除")
}

// TestFileItemDelete_SpecialChars 验证：含日文字符与 {{标记}} 的文件名能正常删除（排除编码问题）。
func TestFileItemDelete_SpecialChars(t *testing.T) {
	tmpDir := t.TempDir()
	name := "[小湊よつ葉] [STARS-910] demo{{骑兵}}.mp4"
	p := filepath.Join(tmpDir, name)
	assert.NoError(t, os.WriteFile(p, []byte("x"), 0644))

	model.FileItem{Path: p}.Delete()

	_, err := os.Stat(p)
	assert.True(t, os.IsNotExist(err), "特殊字符文件名应被正常删除")
}

// TestDelete_EndToEnd_DotBasenameWithSidecar 端到端验证用户的真实痛点场景：
// 含点号基础名("Movie.2024.1080p.mp4") + 同名 sidecar(.srt) + 完整 Delete 链路。
// 修复前：Delete 返回"成功"但磁盘文件因 Title 二次 TrimSuffix 匹配错位而静默不删，
// HeartBeat 每 180s 增量扫描又把磁盘上仍在的文件加回索引 → 表现"提示成功、文件又出来"。
// 修复后：基于完整路径精确删除，磁盘主文件与 sidecar 都消失，索引条目移除，目录清理。
func TestDelete_EndToEnd_DotBasenameWithSidecar(t *testing.T) {
	tmpDir := t.TempDir()
	mainName := "Movie.2024.1080p.mp4"
	mainPath := filepath.Join(tmpDir, mainName)
	srtPath := filepath.Join(tmpDir, "Movie.2024.1080p.srt")
	jpgPath := filepath.Join(tmpDir, "Movie.2024.1080p.jpg")
	assert.NoError(t, os.WriteFile(mainPath, []byte("video"), 0644))
	assert.NoError(t, os.WriteFile(srtPath, []byte("subtitle"), 0644))
	assert.NoError(t, os.WriteFile(jpgPath, []byte("poster"), 0644))

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	// 模拟真实 EasyFile 扫描出的对象（含完整 Path）
	movie := model.EasyFile(tmpDir, mainPath, mainName, "mp4", 100, time.Now(), tmpDir)
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	// 删除前断言：磁盘文件存在且索引中存在
	_, statBefore := os.Stat(mainPath)
	assert.NoError(t, statBefore)
	assert.False(t, engine.FindById(movie.Id).IsNull(), "删除前索引应存在该条目")

	// 执行完整删除链路
	res := app.Delete(movie.Id)
	assert.True(t, res.IsSuccess(), "Delete 应返回成功")

	// 删除后断言：磁盘主文件 + 所有 sidecar 都被精确删除
	_, errMain := os.Stat(mainPath)
	assert.True(t, os.IsNotExist(errMain), "含点号主文件应被真正删除: %s", mainPath)
	_, errSrt := os.Stat(srtPath)
	assert.True(t, os.IsNotExist(errSrt), "sidecar .srt 应被删除")
	_, errJpg := os.Stat(jpgPath)
	assert.True(t, os.IsNotExist(errJpg), "sidecar .jpg 应被删除")

	// 索引条目已移除（不会再被增量扫描加回）
	assert.True(t, engine.FindById(movie.Id).IsNull(), "删除后索引条目应已移除")

	// 目录为空应被清理
	_, errDir := os.Stat(tmpDir)
	assert.True(t, os.IsNotExist(errDir), "目录清空后应被向上清理")
}

// TestFileItemDelete_ClearsEmptyDir 验证：删除后目录变空时向上清理空目录。
func TestFileItemDelete_ClearsEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	sub := filepath.Join(tmpDir, "sub")
	assert.NoError(t, os.MkdirAll(sub, 0755))
	p := filepath.Join(sub, "only.mp4")
	assert.NoError(t, os.WriteFile(p, []byte("x"), 0644))

	model.FileItem{Path: p}.Delete()

	_, err := os.Stat(sub)
	assert.True(t, os.IsNotExist(err), "仅含一个文件的子目录删除后应被清理")
}
