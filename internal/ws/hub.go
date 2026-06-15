package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// OnlineSession 在线会话
type OnlineSession struct {
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	IP        string    `json:"ip"`
	LoginTime time.Time `json:"loginTime"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Type      string    `json:"type"` // "online" | "chat" | "system"
	Username  string    `json:"username,omitempty"`
	Role      string    `json:"role,omitempty"`
	Content   string    `json:"content,omitempty"`
	Time      time.Time `json:"time"`
	OnlineUsers []OnlineSession `json:"onlineUsers,omitempty"`
}

// ClientConn 客户端连接
type ClientConn struct {
	Conn     *websocket.Conn
	Username string
	Role     string
	IP       string
	LoginAt  time.Time
	mu       sync.Mutex
}

// SendJSON 线程安全发送 JSON 消息
func (c *ClientConn) SendJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteJSON(v)
}

// SendBatchHistory 线程安全批量发送聊天历史
func (c *ClientConn) SendBatchHistory(history []ChatMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, msg := range history {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		_ = c.Conn.WriteMessage(websocket.TextMessage, data)
	}
}

// Hub WebSocket 连接管理中心
type Hub struct {
	clients   map[*ClientConn]bool
	mu        sync.RWMutex
	register   chan *ClientConn
	unregister chan *ClientConn
	broadcast  chan []byte
	chatHistory []ChatMessage // 最近 N 条聊天记录
	historyMu   sync.RWMutex
	maxHistory  int
}

var DefaultHub *Hub

func init() {
	DefaultHub = NewHub(100)
	go DefaultHub.Run()
}

// NewHub 创建 Hub
func NewHub(maxHistory int) *Hub {
	return &Hub{
		clients:    make(map[*ClientConn]bool),
		register:   make(chan *ClientConn),
		unregister: make(chan *ClientConn),
		broadcast:  make(chan []byte, 256),
		chatHistory: make([]ChatMessage, 0, maxHistory),
		maxHistory:  maxHistory,
	}
}

// Run 启动 Hub 主循环
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.broadcastOnlineUsers()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Conn.Close()
			}
			h.mu.Unlock()
			h.broadcastOnlineUsers()

		case message := <-h.broadcast:
			var chatMsg ChatMessage
			if err := json.Unmarshal(message, &chatMsg); err == nil && chatMsg.Type == "chat" {
				h.historyMu.Lock()
				h.chatHistory = append(h.chatHistory, chatMsg)
				if len(h.chatHistory) > h.maxHistory {
					h.chatHistory = h.chatHistory[len(h.chatHistory)-h.maxHistory:]
				}
				h.historyMu.Unlock()
			}

			h.mu.RLock()
			for client := range h.clients {
			 client := client
			 go func() {
			  client.mu.Lock()
			  err := client.Conn.WriteMessage(websocket.TextMessage, message)
			  client.mu.Unlock()
			  if err != nil {
			   h.unregister <- client
			  }
			 }()
			}
			h.mu.RUnlock()
		}
	}
}

// Register 注册客户端
func (h *Hub) Register(client *ClientConn) {
	h.register <- client
}

// Unregister 注销客户端
func (h *Hub) Unregister(client *ClientConn) {
	h.unregister <- client
}

// Broadcast 广播消息
func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}

// GetOnlineUsers 获取在线用户列表
func (h *Hub) GetOnlineUsers() []OnlineSession {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]OnlineSession, 0, len(h.clients))
	for client := range h.clients {
		users = append(users, OnlineSession{
			Username:  client.Username,
			Role:      client.Role,
			IP:        client.IP,
			LoginTime: client.LoginAt,
		})
	}
	return users
}

// GetChatHistory 获取聊天历史
func (h *Hub) GetChatHistory() []ChatMessage {
	h.historyMu.RLock()
	defer h.historyMu.RUnlock()

	result := make([]ChatMessage, len(h.chatHistory))
	copy(result, h.chatHistory)
	return result
}

// broadcastOnlineUsers 广播在线用户列表
func (h *Hub) broadcastOnlineUsers() {
	onlineUsers := h.GetOnlineUsers()
	msg := ChatMessage{
		Type:        "online",
		Time:        time.Now(),
		OnlineUsers: onlineUsers,
	}
	data, _ := json.Marshal(msg)
	h.Broadcast(data)
}
