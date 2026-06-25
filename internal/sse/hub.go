package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// clientTimeout 客户端超时阈值：超过该时间未成功发送事件则视为断连
const clientTimeout = 5 * time.Minute

type Event struct {
	Type string      `json:"Type"`
	Data interface{} `json:"Data"`
}

type Client struct {
	ID         int
	Events     chan Event
	lastActive time.Time // 最后一次成功发送事件的时间
}

type Hub struct {
	clients    map[int]*Client
	mu         sync.RWMutex
	nextID     int
	broadcast  chan Event
	register   chan *Client
	unregister chan *Client
}

var (
	DefaultHub  *Hub
	hubRunning  atomic.Bool // 防止 Run() 被递归启动
)

func init() {
	DefaultHub = NewHub()
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
					fmt.Printf("SSE Hub 主循环 panic: %v\n", r)
				}
			}()
			DefaultHub.Run()
		}()
	}
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int]*Client),
		broadcast:  make(chan Event, 100),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	// 定期清理超时客户端
	cleanupTicker := time.NewTicker(1 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				close(client.Events)
				delete(h.clients, client.ID)
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			h.mu.RLock()
			clients := make([]*Client, 0, len(h.clients))
			for _, client := range h.clients {
				clients = append(clients, client)
			}
			h.mu.RUnlock()

			for _, client := range clients {
				select {
				case client.Events <- event:
					// 发送成功，更新活跃时间
					client.lastActive = time.Now()
				default:
					// 缓冲区满，不更新活跃时间（下次清理时若超时则踢出）
				}
			}

		case <-cleanupTicker.C:
			h.cleanupStaleClients()
		}
	}
}

// cleanupStaleClients 移除超过 clientTimeout 未成功发送事件的客户端
func (h *Hub) cleanupStaleClients() {
	now := time.Now()
	h.mu.Lock()
	for id, client := range h.clients {
		if now.Sub(client.lastActive) > clientTimeout {
			close(client.Events)
			delete(h.clients, id)
		}
	}
	h.mu.Unlock()
}

func (h *Hub) Broadcast(event Event) {
	select {
	case h.broadcast <- event:
	default:
		// 广播 channel 满时丢弃最早的事件，保障最新事件能送达
		select {
		case <-h.broadcast:
		default:
		}
		h.broadcast <- event
	}
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func HandleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	DefaultHub.mu.Lock()
	id := DefaultHub.nextID
	DefaultHub.nextID++
	DefaultHub.mu.Unlock()

	client := &Client{
		ID:         id,
		Events:     make(chan Event, 10),
		lastActive: time.Now(),
	}

	DefaultHub.register <- client

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			DefaultHub.unregister <- client
			return
		case event, ok := <-client.Events:
			if !ok {
				return
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func BroadcastEvent(eventType string, data interface{}) {
	DefaultHub.Broadcast(Event{
		Type: eventType,
		Data: data,
	})
}
