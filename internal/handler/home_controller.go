package handler

import (
 "net/http"
 "search-gin/internal/service"
 "search-gin/pkg/consts"
 "search-gin/pkg/utils"
 "sort"
 "sync"

 "github.com/gin-gonic/gin"
)

func Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{"title": "首页"})
}

func GetTypeSize(c *gin.Context) {
	if service.SearchEngine.IsEmpty() {
		service.FileApp.ScanAll()
	}
	res := mapToSlice(&consts.TypeMenu)
	smallCount := len(consts.SmallDir)
	if smallCount > 0 {
		smallSize := consts.NewMenuSize("小文件数量", int64(smallCount))
		smallSize.SizeStr = utils.GetSizeStr(smallSize.Size)
		res = append(res, smallSize)
		for i := 0; i < len(consts.SmallDir); i++ {
			consts.SmallDir[i].SizeStr = utils.GetSizeStr(consts.SmallDir[i].Size)
			res = append(res, consts.SmallDir[i])
		}
	}

	c.JSON(http.StatusOK, res)
}

func GetTagSize(c *gin.Context) {
	res := mapToSlice(&consts.TagMenu)
	c.JSON(http.StatusOK, res)
}

func GetSeriesSize(c *gin.Context) {
	res := mapToSlice(&consts.SeriesCount)
	c.JSON(http.StatusOK, res)
}

func GetLogMemory(c *gin.Context) {
	c.JSON(http.StatusOK, consts.LogMemory)
}

func GetScanTime(c *gin.Context) {
	var res []consts.MenuSize
	consts.FolderTime.Range(func(_, value interface{}) bool {
		res = append(res, value.(consts.MenuSize))
		return true
	})

	sort.Slice(res, func(i, j int) bool {
		return res[i].Cnt > res[j].Cnt
	})
	c.JSON(http.StatusOK, res)
}
func GetHeartBeat(c *gin.Context) {
	c.JSON(http.StatusOK, consts.IndexNumber)
}

func mapToSlice(m *sync.Map) []consts.MenuSize {
	var res []consts.MenuSize
	m.Range(func(_, value interface{}) bool {
		res = append(res, value.(consts.MenuSize))
		return true
	})
	for i := 0; i < len(res); i++ {
		res[i].SizeStr = utils.GetSizeStr(res[i].Size)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Size > res[j].Size
	})
	return res
}
