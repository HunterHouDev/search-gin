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

	// 聚合数据：全量匹配结果中的作者/标签/系列统计（分页前计算）
	AuthorAgg map[string]AggItem `json:"authorAgg,omitempty"`
	TagAgg    map[string]AggItem `json:"tagAgg,omitempty"`
	SeriesAgg map[string]AggItem `json:"seriesAgg,omitempty"`

	// 文件大小范围（用于前端动态生成快捷选项）
	ResultMinSize int64 `json:"resultMinSize,omitempty"`
	ResultMaxSize int64 `json:"resultMaxSize,omitempty"`

	// 文件日期范围（用于前端动态日期快捷）
	ResultMinDate int64 `json:"resultMinDate,omitempty"`
	ResultMaxDate int64 `json:"resultMaxDate,omitempty"`

	// 文件扩展名聚合（用于前端动态扩展名快捷）
	ExtAgg map[string]AggItem `json:"extAgg,omitempty"`
}

func NewPageWrapper() PageResultWrapper {
	return PageResultWrapper{}
}

// NewPageWrapperWithCap 创建指定初始容量的搜索结果包装器
func NewPageWrapperWithCap(capacity int) PageResultWrapper {
	return PageResultWrapper{
		FileList: make([]FileItem, 0, capacity),
	}
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
