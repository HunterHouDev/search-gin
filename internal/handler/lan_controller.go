package handler

import (
	"net/http"
	"search-gin/internal/service"
	"strconv"
	"strings"
	"sync"

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

// GetPeerStats 获取指定节点文件数与总大小
func GetPeerStats(c *gin.Context) {
	node := c.Query("node") // "host:port"
	if node == "" {
		c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "缺少 node 参数"})
		return
	}
	cnt, size, name := service.GetPeerStats(node)
	c.JSON(http.StatusOK, gin.H{"success": true, "nodeName": name, "totalCnt": cnt, "totalSize": size})
}

// GetLanPeersWithStats 获取在线节点列表（含文件统计，并发请求各节点）
func GetLanPeersWithStats(c *gin.Context) {
	peers := service.GetOnlinePeers()
	type peerInfo struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		IP        string `json:"ip"`
		Port      string `json:"port"`
		FilePort  string `json:"filePort"`
		TotalCnt  int    `json:"totalCnt"`
		TotalSize string `json:"totalSize"`
	}
	result := make([]peerInfo, len(peers))
	var wg sync.WaitGroup
	for i, p := range peers {
		wg.Add(1)
		go func(idx int, peer *service.Peer) {
			defer wg.Done()
			cnt, size, _ := service.GetPeerStats(peer.ID)
			result[idx] = peerInfo{
				ID: peer.ID, Name: peer.Name, IP: peer.IP,
				Port: peer.Port, FilePort: peer.FilePort,
				TotalCnt: cnt, TotalSize: size,
			}
		}(i, p)
	}
	wg.Wait()
	c.JSON(http.StatusOK, gin.H{
		"localNodeHost": service.LocalNodeHost,
		"localNodeName": service.LocalNodeName,
		"peers":         result,
	})
}
