package model

// PageAuthorResultWrapper 演员搜索结果分页
type PageAuthorResultWrapper struct {
	FileList    []Author
	Size        int64
	LibCount    int
	LibSize     int64
	SearchCount int
	SearchSize  int64
	ResultSize  int64
	ResultCount int
}

// NewAuthorPageWrapper 创建空演员分页结果
func NewAuthorPageWrapper() PageAuthorResultWrapper {
	return PageAuthorResultWrapper{}
}

func (fsw *PageAuthorResultWrapper) IsNotEmpty() bool {
	return len(fsw.FileList) > 0
}

func (fsw *PageAuthorResultWrapper) AddItem(act Author) {
	fsw.FileList = append(fsw.FileList, act)
	fsw.LibCount = fsw.LibCount + 1
	fsw.SearchCount = fsw.SearchCount + 1
	fsw.Size += act.Size
}
