package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"search-gin/internal/ws"

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
	// 从 context 获取认证信息（中间件已注入）
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	usernameStr, ok := username.(string)
	if !ok || usernameStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}
	roleStr, _ := role.(string)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	clientIP := c.ClientIP()

	client := &ws.ClientConn{
		Conn:     conn,
		Username: usernameStr,
		Role:     roleStr,
		IP:       clientIP,
		LoginAt:  time.Now(),
	}

	ws.DefaultHub.Register(client)

	// 发送聊天历史给新连接的客户端
	history := ws.DefaultHub.GetChatHistory()
	if len(history) > 0 {
		for _, msg := range history {
			data, _ := json.Marshal(msg)
			conn.WriteMessage(websocket.TextMessage, data)
		}
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

		// 解析客户端发来的聊天消息
		var chatMsg ws.ChatMessage
		if err := json.Unmarshal(message, &chatMsg); err != nil {
			continue
		}

		if chatMsg.Type == "chat" && chatMsg.Content != "" {
			// 服务端补充用户信息
			chatMsg.Username = usernameStr
			chatMsg.Role = roleStr
			chatMsg.Time = time.Now()

			data, _ := json.Marshal(chatMsg)
			ws.DefaultHub.Broadcast(data)
		}
	}
}
