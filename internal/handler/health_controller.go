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
	BucketCount     int32               `json:"bucketCount"`
	IndexNumber     int32               `json:"indexNumber"`
	ExpectedDirs    int                 `json:"expectedDirs"`
	TotalCount      int                 `json:"totalCount"`
	TotalSize       int64               `json:"totalSize"`
	TotalSizeStr    string              `json:"totalSizeStr"`
	LastScanTime    string              `json:"lastScanTime"`
	ActorCount      int                 `json:"actorCount"`
	TagCount        int                 `json:"tagCount"`
	TypeCount       int                 `json:"typeCount"`
	SeriesCount     int                 `json:"seriesCount"`
	Status          string              `json:"status"`
	Recommendations []string            `json:"recommendations"`
	ScanProgress    consts.ScanProgress `json:"scanProgress"`
}

func GetIndexHealthCheck(c *gin.Context) {
	health := IndexHealth{}

	health.BucketCount = service.SearchEngin.BucketCount()
	health.IndexNumber = consts.IndexNumber
	health.ExpectedDirs = len(consts.GetOSSetting().Dirs)
	health.TotalCount = service.SearchEngin.GetTotalCount()
	health.TotalSize = service.SearchEngin.GetTotalSize()
	health.TotalSizeStr = utils.GetSizeStr(health.TotalSize)
	health.LastScanTime = consts.LastScanTime.Format("2006-01-02 15:04:05")
	health.ActorCount = service.SearchEngin.GetActorCount()
	health.TagCount = consts.GetSyncMapCount(&consts.TagMenu)
	health.TypeCount = consts.GetSyncMapCount(&consts.TypeMenu) - 1 // 排除"全部"
	if health.TypeCount < 0 {
		health.TypeCount = 0
	}
	health.SeriesCount = consts.GetSyncMapCount(&consts.SeriesCount)

	// 读取扫描进度
	consts.SpMu.RLock()
	health.ScanProgress = consts.Sp
	consts.SpMu.RUnlock()

	recommendations := []string{}

	// 根据扫描阶段自动设置状态
	if health.ScanProgress.Phase == "scanning" {
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
