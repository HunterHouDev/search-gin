package handler

import (
	"net/http"
	"search-gin/internal/service"
	"strings"

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

// AddLanPeer 手动添加节点
func AddLanPeer(c *gin.Context) {
	var req struct {
		Addr string `json:"addr"` // "ip:port"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "参数错误"})
		return
	}

	parts := strings.Split(req.Addr, ":")
	if len(parts) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "格式错误，应为 ip:port"})
		return
	}
	ip := parts[0]
	port := parts[1]

	if service.AddPeer(ip, port) {
		c.JSON(http.StatusOK, gin.H{"success": true, "msg": "添加成功"})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "无法连接到该节点"})
	}
}
