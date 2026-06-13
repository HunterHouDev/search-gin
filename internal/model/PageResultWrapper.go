package model

// PageResultWrapper 搜索引擎返回的分页结果
type PageResultWrapper struct {
	FileList    []FileItem
	Size        int64
	LibCount    int
	LibSize     int64
	SearchCount int
	SearchSize  int64
	ResultSize  int64
	ResultCount int
}

func NewPageWrapper() PageResultWrapper {
	return PageResultWrapper{}
}

func (fsw PageResultWrapper) IsNotEmpty() bool {
	return len(fsw.FileList) > 0
}
