package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"search-gin/internal/ws"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 开发环境允许所有来源
	},
}

// HandleWebSocket WebSocket 连接入口
func HandleWebSocket(c *gin.Context) {
	// 直接从请求参数获取 token（skipPaths 已跳过全局认证）
	token := c.Query("token")
	if token == "" {
		// WebSocket 用不了 Authorization 头，但保留尝试
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	if token == "" {
		utils.InfoFormat("[WS] no token provided")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	tokenInfo, valid := consts.ValidateTokenWithInfo(token)
	if !valid {
		utils.InfoFormat("[WS] invalid token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	username := tokenInfo.Username
	role := tokenInfo.Role

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		utils.ErrorFormat("WebSocket 升级失败 [%s]: %v", username, err)
		return
	}

	// 设置 Pong 处理器，检测僵尸连接
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	clientIP := c.ClientIP()
	utils.InfoFormat("WebSocket 连接成功: %s (%s)", username, clientIP)

	client := &ws.ClientConn{
		Conn:     conn,
		Username: username,
		Role:     role,
		IP:       clientIP,
		LoginAt:  time.Now(),
	}

	ws.DefaultHub.Register(client)

	// 发送聊天历史给新连接的客户端
	history := ws.DefaultHub.GetChatHistory()
	if len(history) > 0 {
		client.SendBatchHistory(history)
	}

	// 读取消息循环
	defer func() {
		ws.DefaultHub.Unregister(client)
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

	// 解析客户端发来的消息
		var msg struct {
			Type        string          `json:"type"`
			To          string          `json:"to,omitempty"`
			Action      string          `json:"action,omitempty"`
			FromSession string          `json:"fromSession,omitempty"`
			Content     string          `json:"content,omitempty"`
			Data        json.RawMessage `json:"data,omitempty"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "chat":
			if msg.Content == "" {
				continue
			}
			chatMsg := ws.ChatMessage{
				Type:     "chat",
				Username: username,
				Role:     role,
				Content:  msg.Content,
				Time:     time.Now(),
			}
			data, _ := json.Marshal(chatMsg)
			ws.DefaultHub.Broadcast(data)

		case "signal":
			// WebRTC 信令中继：透传给指定用户
			if msg.To == "" {
				continue
			}
			signal := map[string]interface{}{
				"type":   "signal",
				"from":   username,
				"action": msg.Action,
				"data":   msg.Data,
			}
			data, _ := json.Marshal(signal)
			ws.DefaultHub.SendToUser(msg.To, data)

		case "signal-all":
			// 广播给所有人（包括同名其他设备），由前端根据 fromSession 过滤
			signal := map[string]interface{}{
				"type":        "signal",
				"from":        username,
				"fromSession": msg.FromSession,
				"action":      msg.Action,
				"data":        msg.Data,
				"role":        role,
			}
			data, _ := json.Marshal(signal)
			ws.DefaultHub.Broadcast(data)
		}
	}
}
