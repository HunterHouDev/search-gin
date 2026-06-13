package model

// SearchResultWrapper bucket 级搜索中间结果
type SearchResultWrapper struct {
	FileList []FileItem
	Size     int64
}

func NewSearchWrapper() SearchResultWrapper {
	return SearchResultWrapper{}
}

func (fsw *SearchResultWrapper) IsNotEmpty() bool {
	return len(fsw.FileList) > 0
}

func (fsw *SearchResultWrapper) IsEmpty() bool {
	return !fsw.IsNotEmpty()
}

func (fsw *SearchResultWrapper) AddWrapperItem(item FileItem) {
	fsw.FileList = append(fsw.FileList, item)
	fsw.Size += item.Size
}
