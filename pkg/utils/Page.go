package utils

type Page struct {
	PageNo    int         `json:"PageNo"`
	PageSize  int         `json:"PageSize"`
	StartNo   int         `json:"StartNo"`
	TotalPage int         `json:"TotalPage"`
	EachPage  []int       `json:"EachPage"`

	Data      interface{} `json:"Data"`
	KeyWord   string      `json:"KeyWord"`
	TotalCnt  int         `json:"TotalCnt"`
	ResultCnt int         `json:"ResultCnt"`
	CurCnt    int         `json:"CurCnt"`

	ResultSize string `json:"ResultSize"`
	TotalSize  string `json:"TotalSize"`
	CurSize    string `json:"CurSize"`

	IndexProgress int32       `json:"IndexProgress"`
	Aggregates    interface{} `json:"Aggregates,omitempty"` // 搜索结果的聚合数据（作者/标签/系列统计）
}

func NewPage() Page {
	return Page{
		PageNo:   0,
		PageSize: 0,
		StartNo:  0,
		TotalCnt: 0,
		Data:     nil,
		KeyWord:  "",
	}
}

func (p *Page) SetProgress(progress int32) {
	p.IndexProgress = progress
}

func (p *Page) SetResultCnt(resultCnt int, pageNo int) {
	p.ResultCnt = resultCnt
	if p.PageSize == 0 {
		p.PageSize = 10
	}
	totalPage := (resultCnt + p.PageSize - 1) / p.PageSize
	p.TotalPage = totalPage
	var pageList []int
	var headNum = 7
	var middNum = 4
	var middPage = pageNo
	if pageNo <= 5 || pageNo >= (totalPage-5) {
		middPage = totalPage / 2
	}
	for i := 0; i < totalPage; i++ {
		if i < headNum || i > totalPage-headNum {
			pageList = append(pageList, i+1)
			continue
		}
		if i < (middPage+middNum) && i > (middPage-middNum) {
			pageList = append(pageList, i+1)
			continue
		}

	}
	p.EachPage = pageList
}
