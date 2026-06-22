package service

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	engine := NewSearchEngine()
	settings := DefaultSettings()
	events := DefaultEventBus()
	scanQueue := NewScanQueue(engine, settings)
	app := NewSearchService(engine, settings, events, scanQueue)
	SetScanWalkInner(app.WalkInner)
	InitService(engine, app)
	os.Exit(m.Run())
}

func TestDeleteFileByPath_NonExistent(t *testing.T) {
	result := DeleteFileByPath("/path/to/nonexistent/file.avi")
	if !result.IsSuccess() {
		t.Log("DeleteFileByPath on non-existent returns fail (expected)")
	}
}

func TestCreateTransferTask_NonExistent(t *testing.T) {
	result := CreateTransferTask("non_existent_id", "h264")
	if result.IsSuccess() {
		t.Error("CreateTransferTask should fail for non-existent file")
	}
}

func TestCreateCutTask_NonExistent(t *testing.T) {
	result := CreateCutTask("non_existent_id", "00:00", "00:10")
	if result.IsSuccess() {
		t.Error("CreateCutTask should fail for non-existent file")
	}
}

func TestCreateMergeTask_EmptyList(t *testing.T) {
	result := CreateMergeTask([]string{}, "", false)
	if result.IsSuccess() {
		t.Error("CreateMergeTask should fail with empty file list")
	}
}

func TestCreateMergeTask_NonExistentFile(t *testing.T) {
	result := CreateMergeTask([]string{"fake_id_1", "fake_id_2"}, "", false)
	if result.IsSuccess() {
		t.Error("CreateMergeTask should fail for non-existent files")
	}
}
