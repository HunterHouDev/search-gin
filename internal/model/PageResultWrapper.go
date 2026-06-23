package model

// PageResultWrapper 搜索结果（bucket 级中间结果 & 最终分页结果共用）
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

func (fsw PageResultWrapper) IsEmpty() bool {
	return !fsw.IsNotEmpty()
}

func (fsw *PageResultWrapper) AddWrapperItem(item *FileItem) {
	fsw.FileList = append(fsw.FileList, *item)
	fsw.Size += item.Size
}
