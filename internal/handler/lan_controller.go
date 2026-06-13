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
		Addr     string `json:"addr"`     // "ip:port"（兼容旧格式）
		IP       string `json:"ip"`       // IP 地址
		Port     string `json:"port"`     // API 端口
		FilePort string `json:"filePort"` // 文件流端口
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "参数错误"})
		return
	}

	ip, port, filePort := req.IP, req.Port, req.FilePort

	// 兼容旧格式 "ip:port" 或 "ip:port:filePort"
	if ip == "" && req.Addr != "" {
		parts := strings.Split(req.Addr, ":")
		if len(parts) >= 2 {
			ip = parts[0]
			port = parts[1]
		}
		if len(parts) >= 3 {
			filePort = parts[2]
		}
	}

	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "格式错误，需要 ip"})
		return
	}

	if service.AddPeer(ip, port, filePort) {
		c.JSON(http.StatusOK, gin.H{"success": true, "msg": "添加成功"})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "无法连接到该节点"})
	}
}
