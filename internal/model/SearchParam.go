package model

import (
	"strconv"
	"strings"
)

// MergeParam 合并请求参数
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
