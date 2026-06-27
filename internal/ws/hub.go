package ws

import (
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// ChatMessage 聊天消息
type ChatMessage struct {
	Type     string       `json:"type"` // "online" | "chat" | "system"
	Username string       `json:"username,omitempty"`
	Role     string       `json:"role,omitempty"`
	Content  string       `json:"content,omitempty"`
	Time     time.Time    `json:"time"`
	IP       string       `json:"ip,omitempty"`
	Client   []ClientConn `json:"onlineUsers,omitempty"`
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
	clients     map[*ClientConn]bool
	mu          sync.RWMutex
	register    chan *ClientConn
	unregister  chan *ClientConn
	broadcast   chan []byte
	chatHistory []ChatMessage // 最近 N 条聊天记录
	historyMu   sync.RWMutex
	maxHistory  int
}

var (
	DefaultHub *Hub
	hubRunning atomic.Bool // 防止 Run() 被递归启动
)

func init() {
	DefaultHub = NewHub(100)
	startHub()
}

// startHub 启动 Hub 主循环，确保最多一个 goroutine 在运行
func startHub() {
	if hubRunning.CompareAndSwap(false, true) {
		go func() {
			defer hubRunning.Store(false)
			defer func() {
				if r := recover(); r != nil {
					// panic 后不再重启，避免无限递归 goroutine → OOM
					fmt.Printf("WS Hub 主循环 panic: %v\n", r)
				}
			}()
			DefaultHub.Run()
		}()
	}
}

// NewHub 创建 Hub
func NewHub(maxHistory int) *Hub {
	return &Hub{
		clients:     make(map[*ClientConn]bool),
		register:    make(chan *ClientConn),
		unregister:  make(chan *ClientConn, 256),
		broadcast:   make(chan []byte, 256),
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
			failedClients := make([]*ClientConn, 0)
			for client := range h.clients {
				client.mu.Lock()
				err := client.Conn.WriteMessage(websocket.TextMessage, message)
				client.mu.Unlock()
				if err != nil {
					failedClients = append(failedClients, client)
				}
			}
			h.mu.RUnlock()

			for _, client := range failedClients {
				h.unregister <- client
			}
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

// SendToUser 向指定用户的所有设备发送消息
// 返回发送成功的设备数
func (h *Hub) SendToUser(username string, msg []byte) int {
	h.mu.RLock()
	var targetClients []*ClientConn
	for client := range h.clients {
		if client.Username == username {
			targetClients = append(targetClients, client)
		}
	}
	h.mu.RUnlock()

	count := 0
	for _, client := range targetClients {
		client.mu.Lock()
		err := client.Conn.WriteMessage(websocket.TextMessage, msg)
		client.mu.Unlock()
		if err == nil {
			count++
		}
	}
	return count
}

// GetOnlineUsers 获取在线用户列表
func (h *Hub) GetOnlineUsers() []*ClientConn {
	h.mu.RLock()
	clients := make([]*ClientConn, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	return clients
}

// contains 辅助判断字符串是否在切片中
func contains(slice []string, s string) bool {
	return slices.Contains(slice, s)
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
	h.mu.RLock()
	clientList := make([]ClientConn, 0, len(h.clients))
	for client := range h.clients {
		clientList = append(clientList, ClientConn{
			Username: client.Username,
			Role:     client.Role,
			IP:       client.IP,
			LoginAt:  client.LoginAt,
		})
	}
	h.mu.RUnlock()

	msg := ChatMessage{
		Type:   "online",
		Time:   time.Now(),
		Client: clientList,
	}
	data, _ := json.Marshal(msg)
	h.Broadcast(data)
}
