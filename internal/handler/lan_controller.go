package handler

import (
	"net/http"
	"search-gin/internal/service"
	"strconv"
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

// RemoveLanPeer 删除手动添加的节点
func RemoveLanPeer(c *gin.Context) {
	var req struct {
		ID string `json:"id"` // "ip:port"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "参数错误"})
		return
	}
	if service.RemovePeer(req.ID) {
		c.JSON(http.StatusOK, gin.H{"success": true, "msg": "已删除"})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "节点不存在"})
	}
}

// TogglePeer 启用/禁用节点
func TogglePeer(c *gin.Context) {
	var req struct {
		ID       string `json:"id"`
		Disabled bool   `json:"disabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "参数错误"})
		return
	}
	if service.TogglePeerDisabled(req.ID, req.Disabled) {
		status := "已禁用"
		if !req.Disabled {
			status = "已启用"
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "msg": status})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "节点不存在"})
	}
}

// CleanLanPeers 手动清理超时节点
func CleanLanPeers(c *gin.Context) {
	count := service.CleanExpiredPeers()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   count,
		"msg":     "已清理 " + strconv.Itoa(count) + " 个超时节点",
	})
}

// ToggleLanDiscovery 动态启动/停止局域网发现
func ToggleLanDiscovery(c *gin.Context) {
	var req struct {
		Enable bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "参数错误"})
		return
	}

	if req.Enable {
		service.RestartLanDiscovery()
		c.JSON(http.StatusOK, gin.H{"success": true, "msg": "集群模式已启动"})
	} else {
		service.StopLanDiscovery()
		c.JSON(http.StatusOK, gin.H{"success": true, "msg": "集群模式已关闭"})
	}
}
