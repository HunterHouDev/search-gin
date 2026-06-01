package handler

import (
	"fmt"
	"net/http"

	"search-gin/internal/service"
	"search-gin/pkg/consts"

	"github.com/gin-gonic/gin"
)

type IndexHealth struct {
	BucketCount     int32    `json:"bucketCount"`
	IndexNumber     int32    `json:"indexNumber"`
	ExpectedDirs    int      `json:"expectedDirs"`
	TotalCount      int      `json:"totalCount"`
	TotalSize       int64    `json:"totalSize"`
	Status          string   `json:"status"`
	Recommendations []string `json:"recommendations"`
}

func GetIndexHealthCheck(c *gin.Context) {
	health := IndexHealth{}

	health.BucketCount = service.SearchEngin.BucketCount
	health.IndexNumber = consts.IndexNumber
	health.ExpectedDirs = len(consts.GetOSSetting().Dirs)
	health.TotalCount = service.SearchEngin.TotalCount
	health.TotalSize = service.SearchEngin.TotalSize

	recommendations := []string{}

	if health.BucketCount == 0 && health.IndexNumber > 0 {
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
