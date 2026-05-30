package model

type PageActressResultWrapper struct {
	FileList    []Actress
	Size        int64
	LibCount    int
	LibSize     int64
	SearchCount int
	SearchSize  int64
	ResultSize  int64
	ResultCount int
}

func NewActressPageWrapper() PageActressResultWrapper {
 return PageActressResultWrapper{}
}

func (fsw *PageActressResultWrapper) IsNotEmpty() bool {
	return len(fsw.FileList) > 0
}

func (fsw *PageActressResultWrapper) AddItem(act Actress) {
	fsw.FileList = append(fsw.FileList, act)
	fsw.LibCount = fsw.LibCount + 1
	fsw.SearchCount = fsw.SearchCount + 1
	fsw.Size += act.Size
}
