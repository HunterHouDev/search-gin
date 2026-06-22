package model

import (
	"testing"
)

func TestNewFileInfo(t *testing.T) {
	fi := NewFileInfo("mp4", 1024)
	if fi.Name != "mp4" {
		t.Errorf("Name = %q, want %q", fi.Name, "mp4")
	}
	if fi.Cnt != 1 {
		t.Errorf("Cnt = %d, want 1", fi.Cnt)
	}
	if fi.Size != 1024 {
		t.Errorf("Size = %d, want 1024", fi.Size)
	}
	if fi.IsDir {
		t.Error("IsDir should be false")
	}
}

func TestNewFileInfo_ZeroSize(t *testing.T) {
	fi := NewFileInfo("empty", 0)
	if fi.Cnt != 0 {
		t.Errorf("Cnt = %d, want 0 for zero size", fi.Cnt)
	}
}

func TestNewFileInfoFold(t *testing.T) {
	fi := NewFileInfoFold("dir", 2048, true)
	if fi.Name != "dir" {
		t.Errorf("Name = %q, want %q", fi.Name, "dir")
	}
	if fi.Cnt != 1 {
		t.Errorf("Cnt = %d, want 1", fi.Cnt)
	}
	if !fi.IsDir {
		t.Error("IsDir should be true")
	}
}

func TestFileInfo_Plus(t *testing.T) {
	fi := FileInfo{Name: "test", Cnt: 1, Size: 100}
	fi2 := fi.Plus(50)

	// 原值不变（值类型）
	if fi.Size != 100 || fi.Cnt != 1 {
		t.Error("Plus should not mutate original")
	}
	// 返回值累加
	if fi2.Size != 150 {
		t.Errorf("Size = %d, want 150", fi2.Size)
	}
	if fi2.Cnt != 2 {
		t.Errorf("Cnt = %d, want 2", fi2.Cnt)
	}
}

func TestFileInfo_Minus(t *testing.T) {
	fi := FileInfo{Name: "test", Cnt: 5, Size: 500}
	fi2 := fi.Minus(100)

	if fi2.Size != 400 {
		t.Errorf("Size = %d, want 400", fi2.Size)
	}
	if fi2.Cnt != 4 {
		t.Errorf("Cnt = %d, want 4", fi2.Cnt)
	}
}

func TestFileInfo_Minus_ClampToZero(t *testing.T) {
	fi := FileInfo{Name: "test", Cnt: 1, Size: 100}
	fi2 := fi.Minus(200)

	if fi2.Size != 0 {
		t.Errorf("Size = %d, want 0 (clamped)", fi2.Size)
	}
	if fi2.Cnt != 0 {
		t.Errorf("Cnt = %d, want 0 (clamped)", fi2.Cnt)
	}
}

func TestFileInfo_Minus_Multiple(t *testing.T) {
	fi := FileInfo{Name: "test", Cnt: 3, Size: 300}
	fi = fi.Minus(100)
	fi = fi.Minus(100)
	if fi.Size != 100 {
		t.Errorf("Size = %d, want 100 after two minus", fi.Size)
	}
	if fi.Cnt != 1 {
		t.Errorf("Cnt = %d, want 1 after two minus", fi.Cnt)
	}
}
