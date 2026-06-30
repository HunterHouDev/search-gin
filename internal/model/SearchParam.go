package model

import (
	"strconv"
	"strings"
)

// SearchParam 查询参数
type MergeParam struct {
	Files        []string
	Dest         string
	DeleteSource bool
}

// SearchParam 查询参数
type SearchParam struct {
	Keyword    string
	OnlyRepeat bool
	MovieType  string
	DirPath    string
	Page       int
	PageSize   int
	SortField  string
	SortType   string
	SearchNode string // 目标节点 ""=本机 / "host:port"=指定节点

	// 高级过滤（仅本地搜索有效）
	MinSize      int64    `json:"minSize,omitempty"`      // 文件大小下限（字节）
	MaxSize      int64    `json:"maxSize,omitempty"`      // 文件大小上限（字节）
	DateFrom     string   `json:"dateFrom,omitempty"`     // 修改日期起始 "2006-01-02"
	DateTo       string   `json:"dateTo,omitempty"`       // 修改日期截止 "2006-01-02"
	FileExts     []string `json:"fileExts,omitempty"`     // 文件扩展名白名单，如 ["mp4","avi"]
	FilterAuthor string   `json:"filterAuthor,omitempty"` // 按作者精确过滤
	FilterTag    string   `json:"filterTag,omitempty"`    // 按标签精确过滤
	FilterSeries string   `json:"filterSeries,omitempty"` // 按系列精确过滤
}

func NewSearchParam(keyword string, page int, pageSize int, sortField string, sortType string, moiveType string) SearchParam {
	res := SearchParam{
		Keyword:   strings.TrimSpace(keyword),
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortType:  sortType,
		MovieType: strings.TrimSpace(moiveType),
	}
	return res

}

// UniWords 生成搜索条件的唯一标识，不含 Page/PageSize，确保不同分页命中同一缓存
func (p *SearchParam) UniWords() string {
	p.Keyword = strings.TrimSpace(p.Keyword)
	key := p.Keyword + "::" + p.MovieType + "::" + p.SortField + "::" + p.SortType
	// 高级过滤参数必须参与缓存 key，否则不同过滤条件会命中相同缓存
	if p.MinSize > 0 {
		key += "::min" + strconv.FormatInt(p.MinSize, 10)
	}
	if p.MaxSize > 0 {
		key += "::max" + strconv.FormatInt(p.MaxSize, 10)
	}
	if p.DateFrom != "" {
		key += "::from" + p.DateFrom
	}
	if p.DateTo != "" {
		key += "::to" + p.DateTo
	}
	if len(p.FileExts) > 0 {
		key += "::ext" + strings.Join(p.FileExts, ",")
	}
	if p.FilterAuthor != "" {
		key += "::author" + p.FilterAuthor
	}
	if p.FilterTag != "" {
		key += "::tag" + p.FilterTag
	}
	if p.FilterSeries != "" {
		key += "::series" + p.FilterSeries
	}
	return key
}

func (p *SearchParam) GetKeywords() string {
	p.Keyword = strings.TrimSpace(p.Keyword)
	return p.Keyword
}

func (p *SearchParam) GetMovieType() string {
	p.MovieType = strings.TrimSpace(p.MovieType)
	return p.MovieType
}

func (p *SearchParam) SetOnlyRepeat(b bool) {
	p.OnlyRepeat = b
}

func (p *SearchParam) StartNum() int {
	if p.Page <= 0 {
		return 0
	}
	return (p.Page - 1) * p.PageSize
}

func (p *SearchParam) GetPageSize() int {
	if p.PageSize <= 0 {
		return 0
	}
	return p.PageSize
}

func (p *SearchParam) GetPage() int {
	if p.Page <= 0 {
		return 0
	}
	return p.Page
}

func (p *SearchParam) GetSort() []string {
	if p.SortType == "" {
		p.SortType = "desc"
	}
	return []string{p.GetSortField() + p.SortType}
}

func (p *SearchParam) GetSortField() string {
	if p.SortField == "" {
		p.SortField = "m_time"
	}
	return p.SortField + " "
}
