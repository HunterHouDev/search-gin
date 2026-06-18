package handler

import (
	"fmt"
	"net/http"

	"search-gin/internal/service"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

type IndexHealth struct {
	BucketCount      int32               `json:"bucketCount"`
	IndexNumber      int32               `json:"indexNumber"`
	FullScanInProgress bool               `json:"fullScanInProgress"`
	ExpectedDirs    int                 `json:"expectedDirs"`
	TotalCount      int                 `json:"totalCount"`
	TotalSize       int64               `json:"totalSize"`
	TotalSizeStr    string              `json:"totalSizeStr"`
	LastScanTime    string              `json:"lastScanTime"`
	AuthorCount      int                 `json:"actorCount"`
	TagCount        int                 `json:"tagCount"`
	TypeCount       int                 `json:"typeCount"`
	SeriesCount     int                 `json:"seriesCount"`
	Status          string              `json:"status"`
	Recommendations []string            `json:"recommendations"`
	ScanProgress    service.ScanProgress `json:"scanProgress"`
}

func GetIndexHealthCheck(c *gin.Context) {
	health := IndexHealth{}

	health.BucketCount = service.SearchEngine.BucketCount()
	health.IndexNumber = consts.IndexNumber.Load()
	health.FullScanInProgress = service.FullScanInProgress.Load()
	health.ExpectedDirs = len(service.GetOSSetting().Dirs)
	health.TotalCount = service.SearchEngine.GetTotalCount()
	health.TotalSize = service.SearchEngine.GetTotalSize()
	health.TotalSizeStr = utils.GetSizeStr(health.TotalSize)
	health.LastScanTime = consts.GetLastScanTime().Format("2006-01-02 15:04:05")
	health.AuthorCount = service.SearchEngine.GetAuthorCount()
	health.TagCount = service.GetSyncMapCount(&service.TagMenu)
	health.TypeCount = service.GetSyncMapCount(&service.TypeMenu) - 1 // 排除"全部"
	if health.TypeCount < 0 {
		health.TypeCount = 0
	}
	health.SeriesCount = service.GetSyncMapCount(&service.SeriesCount)

	// 读取扫描进度
	health.ScanProgress = service.Sp.Get()

	recommendations := []string{}

	// 根据 FullScanInProgress 原子锁状态补充判断（比 ScanProgress.Phase 更实时）
	if health.FullScanInProgress {
		health.Status = "scanning"
		if health.ScanProgress.Phase == "scanning" {
			recommendations = append(recommendations,
				fmt.Sprintf("正在扫描目录 %d/%d...", health.ScanProgress.CompletedDirs, health.ScanProgress.TotalDirs))
		} else {
			recommendations = append(recommendations, "全量扫描正在进行中...")
		}
	} else if health.ScanProgress.Phase == "scanning" {
	 health.Status = "scanning"
	 recommendations = append(recommendations,
	  fmt.Sprintf("正在扫描目录 %d/%d...", health.ScanProgress.CompletedDirs, health.ScanProgress.TotalDirs))
	} else if health.ScanProgress.Phase == "building" {
	 health.Status = "building"
	 recommendations = append(recommendations,
	  fmt.Sprintf("正在构建索引 %d/%d...", health.ScanProgress.ProcessedBuckets, health.ScanProgress.TotalBuckets))
	} else if health.BucketCount == 0 && health.IndexNumber > 0 {
	 health.Status = "warning"
	 recommendations = append(recommendations, "BucketCount 为 0，但扫描尚未完成，可能存在竞态条件")
	} else if health.BucketCount != int32(health.ExpectedDirs) {
	 if health.IndexNumber > 0 {
	  health.Status = "warning"
	  recommendations = append(recommendations,
	   fmt.Sprintf("扫描正在进行中，BucketCount(%d) != Expected(%d)", health.BucketCount, health.ExpectedDirs))
	 } else {
	  health.Status = "error"
	  recommendations = append(recommendations,
	   fmt.Sprintf("扫描已完成，但 BucketCount(%d) != Expected(%d)，存在严重并发问题", health.BucketCount, health.ExpectedDirs))
	 }
	} else if health.TotalCount == 0 && health.IndexNumber == 0 {
	 health.Status = "empty"
	 recommendations = append(recommendations, "索引为空，请执行扫描")
	} else {
	 health.Status = "healthy"
	}

	health.Recommendations = recommendations
	c.JSON(http.StatusOK, health)
}
