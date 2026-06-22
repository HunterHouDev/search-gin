package model

// FileInfo 菜单条目（用于类型/标签/系列统计）
type FileInfo struct {
	Name    string
	Cnt     int64
	Size    int64
	SizeStr string
	IsDir   bool
}

func NewFileInfo(name string, size int64) FileInfo {
	cnt := int64(0)
	if size > 0 {
		cnt = int64(1)
	}
	return FileInfo{
		Name: name,
		Cnt:  cnt,
		Size: size,
	}
}

func NewFileInfoFold(name string, size int64, isFold bool) FileInfo {
	cnt := int64(0)
	if size > 0 {
		cnt = int64(1)
	}
	return FileInfo{
		Name:  name,
		Cnt:   cnt,
		Size:  size,
		IsDir: isFold,
	}
}

func (m FileInfo) Plus(size int64) FileInfo {
	m.Cnt++
	m.Size += size
	return m
}

func (m FileInfo) Minus(size int64) FileInfo {
	m.Cnt--
	m.Size -= size
	if m.Cnt < 0 {
		m.Cnt = 0
	}
	if m.Size < 0 {
		m.Size = 0
	}
	return m
}
