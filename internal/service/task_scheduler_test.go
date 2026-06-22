package service

import (
	"testing"
)

func TestNewScanQueue(t *testing.T) {
	engine := NewSearchEngine()
	q := NewScanQueue(engine, DefaultSettings())
	if q == nil {
		t.Fatal("NewScanQueue returned nil")
	}
	if q.engine == nil {
		t.Error("engine should not be nil")
	}
}

func TestScanQueue_AddTaskAndGetCount(t *testing.T) {
	engine := NewSearchEngine()
	q := NewScanQueue(engine, DefaultSettings())
	if c := q.GetTaskCount(); c != 0 {
		t.Errorf("initial count = %d, want 0", c)
	}

	q.AddTask("/path/to/dir1")
	if c := q.GetTaskCount(); c != 1 {
		t.Errorf("after add count = %d, want 1", c)
	}

	q.AddTask("/path/to/dir2")
	if c := q.GetTaskCount(); c != 2 {
		t.Errorf("after second add count = %d, want 2", c)
	}
}

func TestScanQueue_AddSameDirCancelsPrevious(t *testing.T) {
	engine := NewSearchEngine()
	q := NewScanQueue(engine, DefaultSettings())
	q.AddTask("/path/to/dir")
	c1 := q.GetTaskCount()

	q.AddTask("/path/to/dir")
	c2 := q.GetTaskCount()

	if c2 != c1 {
		t.Errorf("count should stay same after re-add, got %d -> %d", c1, c2)
	}
}

func TestScanQueue_SetWalkInner(t *testing.T) {
	SetScanWalkInner(nil)
	// should not panic when engine is nil
	engine := NewSearchEngine()
	q := NewScanQueue(engine, DefaultSettings())
	SetScanWalkInner(q.walkInner)
}
