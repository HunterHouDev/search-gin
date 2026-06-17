package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type Event struct {
	Type string      `json:"Type"`
	Data interface{} `json:"Data"`
}

type Client struct {
	ID     int
	Events chan Event
}

type Hub struct {
	clients    map[int]*Client
	mu         sync.RWMutex
	nextID     int
	broadcast  chan Event
	register   chan *Client
	unregister chan *Client
}

var DefaultHub *Hub

func init() {
	DefaultHub = NewHub()
	go DefaultHub.Run()
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
				default:
					h.mu.Lock()
					if _, ok := h.clients[client.ID]; ok {
						close(client.Events)
						delete(h.clients, client.ID)
					}
					h.mu.Unlock()
				}
			}
		}
	}
}

func (h *Hub) Broadcast(event Event) {
	select {
	case h.broadcast <- event:
	default:
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
		ID:     id,
		Events: make(chan Event, 10),
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
