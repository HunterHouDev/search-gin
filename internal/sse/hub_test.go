package sse

import (
	"testing"
	"time"
)

func TestNewHub(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("NewHub returned nil")
	}
	if len(h.clients) != 0 {
		t.Errorf("new hub should have 0 clients, got %d", len(h.clients))
	}
	if h.broadcast == nil {
		t.Error("broadcast channel is nil")
	}
	if h.register == nil {
		t.Error("register channel is nil")
	}
	if h.unregister == nil {
		t.Error("unregister channel is nil")
	}
}

func TestHub_RegisterClient(t *testing.T) {
	h := NewHub()
	go h.Run()
	defer func() {
		// stop the Run goroutine by sending a nil register (will panic if not handled)
		// instead, we just let it run with the test
	}()

	client := &Client{ID: 1, Events: make(chan Event, 10)}
	client.lastActiveUnix.Store(time.Now().UnixNano())
	h.register <- client

	// wait for hub to process
	time.Sleep(10 * time.Millisecond)

	if h.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", h.ClientCount())
	}
}

func TestHub_RegisterAndUnregister(t *testing.T) {
	h := NewHub()
	go h.Run()

	client := &Client{ID: 1, Events: make(chan Event, 10)}
	client.lastActiveUnix.Store(time.Now().UnixNano())
	h.register <- client
	time.Sleep(10 * time.Millisecond)

	if h.ClientCount() != 1 {
		t.Fatalf("expected 1 client after register, got %d", h.ClientCount())
	}

	h.unregister <- client
	time.Sleep(10 * time.Millisecond)

	if h.ClientCount() != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", h.ClientCount())
	}

	// unregistered client's Events channel should be closed
	_, ok := <-client.Events
	if ok {
		t.Error("expected client.Events to be closed after unregister")
	}
}

func TestHub_BroadcastToSingleClient(t *testing.T) {
	h := NewHub()
	go h.Run()

	client := &Client{ID: 1, Events: make(chan Event, 10)}
	client.lastActiveUnix.Store(time.Now().UnixNano())
	h.register <- client
	time.Sleep(10 * time.Millisecond)

	event := Event{Type: "test", Data: "hello"}
	h.Broadcast(event)

	select {
	case received := <-client.Events:
		if received.Type != "test" {
			t.Errorf("expected event type 'test', got %q", received.Type)
		}
		s, ok := received.Data.(string)
		if !ok || s != "hello" {
			t.Errorf("expected data 'hello', got %v", received.Data)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for broadcast event")
	}
}

func TestHub_BroadcastToMultipleClients(t *testing.T) {
	h := NewHub()
	go h.Run()

	client1 := &Client{ID: 1, Events: make(chan Event, 10)}
	client1.lastActiveUnix.Store(time.Now().UnixNano())
	client2 := &Client{ID: 2, Events: make(chan Event, 10)}
	client2.lastActiveUnix.Store(time.Now().UnixNano())
	h.register <- client1
	h.register <- client2
	time.Sleep(10 * time.Millisecond)

	event := Event{Type: "broadcast", Data: "to-all"}
	h.Broadcast(event)

	for i, client := range []*Client{client1, client2} {
		select {
		case received := <-client.Events:
			if received.Type != "broadcast" {
				t.Errorf("client %d: expected type 'broadcast', got %q", i+1, received.Type)
			}
		case <-time.After(time.Second):
			t.Errorf("client %d: timeout waiting for broadcast", i+1)
		}
	}
}

func TestHub_CleanupStaleClients(t *testing.T) {
	h := NewHub()
	go h.Run()

	// active client — will not be cleaned
	activeClient := &Client{ID: 1, Events: make(chan Event, 10)}
	activeClient.lastActiveUnix.Store(time.Now().UnixNano())
	// stale client — lastActiveUnix in the past
	staleClient := &Client{ID: 2, Events: make(chan Event, 10)}
	staleClient.lastActiveUnix.Store(time.Now().Add(-10 * time.Minute).UnixNano())

	h.register <- activeClient
	h.register <- staleClient
	time.Sleep(10 * time.Millisecond)

	if h.ClientCount() != 2 {
		t.Fatalf("expected 2 clients before cleanup, got %d", h.ClientCount())
	}

	// trigger cleanup by reducing timeout for this test
	// we call cleanupStaleClients directly
	h.cleanupStaleClients()

	if h.ClientCount() != 1 {
		t.Errorf("expected 1 client after cleanup, got %d", h.ClientCount())
	}

	// stale client should be removed from map, channel is not closed (close only in unregister)
	// active client should still be open
	select {
	case _, ok := <-activeClient.Events:
		if !ok {
			t.Error("expected active client Events channel to be open")
		}
	default:
		// channel is open but empty — this is fine
	}
}
