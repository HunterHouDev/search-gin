package service

import (
	"os"
	"path/filepath"
	"search-gin/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ── cleanPath 测试 ──

func TestCleanPath_fileext(t *testing.T) {
	origPath := "video.mp4"
	ext := filepath.Ext(origPath)
	assert.Equal(t, ".mp4", ext)
}

func TestCleanPath_RemovesMarkers(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"《tag1,tag2》", "tag1,tag2"},
		{"{{type}}", "type"},
		{"《tag》{{type}}", "tagtype"},
		{"normal", "normal"},
		{"  spaces  ", "spaces"},
		{"《》", ""},
		{"{{}}", ""},
		{"《tag1》《tag2》", "tag1tag2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cleanPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ── choose2To1 测试 ──

func TestChoose2To1(t *testing.T) {
	assert.Equal(t, "a", choose2To1(true, "a", "b"))
	assert.Equal(t, "b", choose2To1(false, "a", "b"))
	assert.Equal(t, "", choose2To1(true, "", "b"))
	assert.Equal(t, "b", choose2To1(false, "", "b"))
}

// ── pathReplacer 测试 ──

func TestPathReplacer(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"《tag》", "tag"},
		{"{{type}}", "type"},
		{"《tag》{{type}}", "tagtype"},
		{"normal", "normal"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := pathReplacer.Replace(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ── SetMovieType 测试 ──

func TestSetMovieType_AddTypeToNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:        "test-1",
		Name:      "video.mp4",
		Path:      origPath,
		DirPath:   tmpDir,
		FileType:  "mp4",
		Size:      100,
		BaseDir:   tmpDir,
		MovieType: "",
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	res := app.SetMovieType(movie, "动漫")
	assert.True(t, res.IsSuccess())

	newPath := filepath.Join(tmpDir, "video{{动漫}}.mp4")
	_, err := os.Stat(newPath)
	assert.NoError(t, err, "新文件应存在: %s", newPath)
}

func TestSetMovieType_ChangeExistingType(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video{{动漫}}.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:        "test-2",
		Name:      "video{{动漫}}.mp4",
		Path:      origPath,
		DirPath:   tmpDir,
		FileType:  "mp4",
		Size:      100,
		BaseDir:   tmpDir,
		MovieType: "动漫",
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	res := app.SetMovieType(movie, "国剧")
	assert.True(t, res.IsSuccess())

	newPath := filepath.Join(tmpDir, "video{{国剧}}.mp4")
	_, err := os.Stat(newPath)
	assert.NoError(t, err)
}

func TestSetMovieType_SameTypeNoop(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video{{动漫}}.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:        "test-3",
		Name:      "video{{动漫}}.mp4",
		Path:      origPath,
		DirPath:   tmpDir,
		FileType:  "mp4",
		Size:      100,
		BaseDir:   tmpDir,
		MovieType: "动漫",
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	res := app.SetMovieType(movie, "动漫")
	assert.True(t, res.IsSuccess())
	// 文件不应被修改
	_, err := os.Stat(origPath)
	assert.NoError(t, err)
}

// ── AddTag 测试 ──

func TestAddTag_ToFileWithoutTags(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "tag-1",
		Name:     "video.mp4",
		Path:     origPath,
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
		Tags:     []string{},
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	res := app.AddTag("tag-1", "action")
	assert.True(t, res.IsSuccess())

	newPath := filepath.Join(tmpDir, "video《action》.mp4")
	_, err := os.Stat(newPath)
	assert.NoError(t, err)
}

func TestAddTag_EmptyTagSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "tag-2",
		Name:     "video.mp4",
		Path:     origPath,
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
		Tags:     []string{},
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	res := app.AddTag("tag-2", "")
	assert.True(t, res.IsSuccess())
	// 文件不应被修改
	_, err := os.Stat(origPath)
	assert.NoError(t, err)
}

func TestAddTag_DuplicateTagSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video《action》.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "tag-3",
		Name:     "video《action》.mp4",
		Path:     origPath,
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
		Tags:     []string{"action"},
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	res := app.AddTag("tag-3", "action")
	assert.True(t, res.IsSuccess())
}

func TestAddTag_ToFileWithExistingTags(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video《action》.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "tag-4",
		Name:     "video《action》.mp4",
		Path:     origPath,
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
		Tags:     []string{"action"},
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	res := app.AddTag("tag-4", "drama")
	assert.True(t, res.IsSuccess())

	newPath := filepath.Join(tmpDir, "video《action,drama》.mp4")
	_, err := os.Stat(newPath)
	assert.NoError(t, err)
}

// ── ClearTag 测试 ──

func TestClearTag_RemovesTag(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video《action,drama》.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "clear-1",
		Name:     "video《action,drama》.mp4",
		Path:     origPath,
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
		Tags:     []string{"action", "drama"},
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	res := app.ClearTag("clear-1", "action")
	assert.True(t, res.IsSuccess())

	newPath := filepath.Join(tmpDir, "video《drama》.mp4")
	_, err := os.Stat(newPath)
	assert.NoError(t, err)
}

func TestClearTag_AllTagsRemoved(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video《action》.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "clear-2",
		Name:     "video《action》.mp4",
		Path:     origPath,
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
		Tags:     []string{"action"},
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	res := app.ClearTag("clear-2", "action")
	assert.True(t, res.IsSuccess())

	newPath := filepath.Join(tmpDir, "video.mp4")
	_, err := os.Stat(newPath)
	assert.NoError(t, err)
}

func TestClearTag_NoTagsNoop(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "clear-3",
		Name:     "video.mp4",
		Path:     origPath,
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
		Tags:     []string{},
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	res := app.ClearTag("clear-3", "action")
	assert.True(t, res.IsSuccess())
	// 文件不应被修改
	_, err := os.Stat(origPath)
	assert.NoError(t, err)
}

// ── Delete 测试 ──

func TestDelete_RemovesFileAndIndex(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "del-1",
		Name:     "video.mp4",
		Path:     origPath,
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
		Title:    "video",
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	app.Delete("del-1")

	_, err := os.Stat(origPath)
	assert.True(t, os.IsNotExist(err), "文件应已被删除")
}

func TestDelete_NonExistentIdNoop(t *testing.T) {
	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	// 不应 panic
	app.Delete("nonexistent-id")
}

// ── Move 测试 ──

func TestMove_FileToNewDir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "target")
	origPath := filepath.Join(tmpDir, "video.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "move-1",
		Name:     "video.mp4",
		Path:     origPath,
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	res := app.Move("move-1", newDir, "renamed")
	assert.True(t, res.IsSuccess())

	newPath := filepath.Join(newDir, "renamed.mp4")
	_, err := os.Stat(newPath)
	assert.NoError(t, err, "移动后的文件应存在")
}

func TestMove_NonExistentFile(t *testing.T) {
	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	res := app.Move("nonexistent", "/tmp/newdir", "newname")
	assert.False(t, res.IsSuccess())
}

// ── Rename 测试 ──

func TestRename_ChangeFileName(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "old_name.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "rename-1",
		Name:     "old_name.mp4",
		Path:     origPath,
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	edit := model.FileEdit{
		FileItem: model.FileItem{
			Id:   "rename-1",
			Name: "new_name.mp4",
		},
	}
	res := app.Rename(edit)
	assert.True(t, res.IsSuccess())

	newPath := filepath.Join(tmpDir, "new_name.mp4")
	_, err := os.Stat(newPath)
	assert.NoError(t, err)
}

func TestRename_NonExistentFile(t *testing.T) {
	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	edit := model.FileEdit{
		FileItem: model.FileItem{
			Id: "nonexistent",
		},
	}
	res := app.Rename(edit)
	assert.False(t, res.IsSuccess())
}

func TestRename_FileNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "rename-ne",
		Name:     "nonexistent.mp4",
		Path:     filepath.Join(tmpDir, "nonexistent.mp4"),
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	edit := model.FileEdit{
		FileItem: model.FileItem{
			Id:   "rename-ne",
			Name: "new_name.mp4",
		},
	}
	res := app.Rename(edit)
	assert.False(t, res.IsSuccess())
}

// ── Rename MoveOut 测试 ──

func TestRename_MoveOutWithAuthor(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "rename-mo",
		Name:     "video.mp4",
		Path:     origPath,
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	edit := model.FileEdit{
		FileItem: model.FileItem{
			Id:     "rename-mo",
			Name:   "video.mp4",
			Author: "张三",
			Title:  "电影标题",
		},
		MoveOut: true,
	}
	res := app.Rename(edit)
	assert.True(t, res.IsSuccess())
}

func TestRename_MoveOutWithCode(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "rename-mc",
		Name:     "video.mp4",
		Path:     origPath,
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	edit := model.FileEdit{
		FileItem: model.FileItem{
			Id:    "rename-mc",
			Name:  "video.mp4",
			Title: "电影标题",
			Code:  "ABC-123",
		},
		MoveOut: true,
	}
	res := app.Rename(edit)
	assert.True(t, res.IsSuccess())
}

// ── Delete 带附属文件测试 ──

func TestDelete_WithCompanionFiles(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video.mp4")
	jpgPath := filepath.Join(tmpDir, "video.jpg")
	pngPath := filepath.Join(tmpDir, "video.png")
	os.WriteFile(origPath, []byte("test"), 0644)
	os.WriteFile(jpgPath, []byte("jpg"), 0644)
	os.WriteFile(pngPath, []byte("png"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:       "del-comp",
		Name:     "video.mp4",
		Path:     origPath,
		Jpg:      jpgPath,
		Png:      pngPath,
		DirPath:  tmpDir,
		FileType: "mp4",
		Size:     100,
		BaseDir:  tmpDir,
		Title:    "video",
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	app.Delete("del-comp")

	_, err1 := os.Stat(origPath)
	_, err2 := os.Stat(jpgPath)
	_, err3 := os.Stat(pngPath)
	assert.True(t, os.IsNotExist(err1))
	assert.True(t, os.IsNotExist(err2))
	assert.True(t, os.IsNotExist(err3))
}

// ── SetMovieType "无" 类型测试 ──

func TestSetMovieType_TypeIsWu(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "video.mp4")
	os.WriteFile(origPath, []byte("test"), 0644)

	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)

	movie := model.FileItem{
		Id:        "test-wu",
		Name:      "video.mp4",
		Path:      origPath,
		DirPath:   tmpDir,
		FileType:  "mp4",
		Size:      100,
		BaseDir:   tmpDir,
		MovieType: "无",
	}
	engine.installIndex(buildIndexFromBuckets(map[string]*bucketFile{
		tmpDir: makeBucket(tmpDir, movie),
	}))

	res := app.SetMovieType(movie, "动漫")
	assert.True(t, res.IsSuccess())

	newPath := filepath.Join(tmpDir, "video{{动漫}}.mp4")
	_, err := os.Stat(newPath)
	assert.NoError(t, err)
}

// ── 辅助函数 ──

func splitAndTrim(s string) []string {
	result := []string{}
	for _, part := range splitByComma(s) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitByComma(s string) []string {
	if s == "" {
		return []string{}
	}
	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func joinTags(tags []string) string {
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += ","
		}
		result += tag
	}
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
