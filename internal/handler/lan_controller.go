package handler

import (
	"net/http"
	"search-gin/internal/service"

	"github.com/gin-gonic/gin"
)

// GetLanPeers 获取在线节点列表
func GetLanPeers(c *gin.Context) {
	peers := service.GetOnlinePeers()
	c.JSON(http.StatusOK, gin.H{
		"localNodeHost": service.LocalNodeHost,
		"localNodeName": service.LocalNodeName,
		"peers":         peers,
	})
}
